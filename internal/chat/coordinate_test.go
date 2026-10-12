package chat

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"duckduckgo-chat-cli/internal/models"
	"duckduckgo-chat-cli/internal/ui"
)

func coordinateTestPlan(strategy CoordinateStrategy, rounds int) CoordinatePlan {
	return CoordinatePlan{
		Strategy: strategy,
		Agents: []CoordinateAgent{
			{Model: models.GPT6Luna, Role: "Analyst"},
			{Model: models.ClaudeHaiku, Role: "Skeptic"},
			{Model: models.Gemma431B, Role: "Synthesizer"},
		},
		DebateRounds: rounds,
	}
}

func parallelCoordinateTestPlan(strategy CoordinateStrategy, rounds int) CoordinatePlan {
	plan := coordinateTestPlan(strategy, rounds)
	plan.Parallel = true
	return plan
}

func assertCoordinateRunReport(t *testing.T, value any, wantLabels []string, wantFinal string) {
	t.Helper()
	report := reflect.ValueOf(value)
	if report.Kind() != reflect.Pointer || report.IsNil() {
		t.Fatalf("RunCoordinate result has type %T; want a non-nil coordinate run report", value)
	}
	report = report.Elem()
	stages := report.FieldByName("Stages")
	if !stages.IsValid() || stages.Kind() != reflect.Slice {
		t.Fatalf("coordinate run report %T has no Stages slice", value)
	}
	if stages.Len() != len(wantLabels) {
		t.Fatalf("stages = %d, want %d", stages.Len(), len(wantLabels))
	}
	seenIDs := map[string]bool{}
	for i, want := range wantLabels {
		stage := stages.Index(i)
		label := stage.FieldByName("Label").String()
		id := stage.FieldByName("ID").String()
		status := stage.FieldByName("Status").String()
		if label != want || id == "" || seenIDs[id] {
			t.Fatalf("stage %d = (id=%q, label=%q, status=%q); want unique ID and label %q", i, id, label, status, want)
		}
		if wantFinal != "" && status != "completed" {
			t.Fatalf("stage %d status = %q, want completed", i, status)
		}
		seenIDs[id] = true
	}
	if got := report.FieldByName("FinalAnswer").String(); got != wantFinal {
		t.Fatalf("FinalAnswer = %q, want %q", got, wantFinal)
	}
}

func TestRunCoordinateReportsOrderedStagesForEachStrategy(t *testing.T) {
	tests := []struct {
		name     string
		strategy CoordinateStrategy
		rounds   int
		labels   []string
	}{
		{name: "independent", strategy: CoordinateIndependent, labels: []string{"Agent 1", "Agent 2", "Synthesis"}},
		{name: "chain", strategy: CoordinateChain, labels: []string{"Agent 1", "Agent 2", "Synthesis"}},
		{name: "debate one round", strategy: CoordinateDebate, rounds: 1, labels: []string{"Agent 1", "Agent 2", "Debate 1 · Agent 1", "Debate 1 · Agent 2", "Synthesis"}},
		{name: "debate two rounds", strategy: CoordinateDebate, rounds: 2, labels: []string{"Agent 1", "Agent 2", "Debate 1 · Agent 1", "Debate 1 · Agent 2", "Debate 2 · Agent 1", "Debate 2 · Agent 2", "Synthesis"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			run, err := RunCoordinate(context.Background(), coordinateTestPlan(tc.strategy, tc.rounds), "Question", func(_ context.Context, _ CoordinateAgent, _ string) (string, error) {
				calls++
				if calls == len(tc.labels) {
					return "final", nil
				}
				return "agent output", nil
			}, nil)
			if err != nil {
				t.Fatal(err)
			}
			assertCoordinateRunReport(t, run, tc.labels, "final")
		})
	}
}

func TestRunCoordinateReturnsCompletedStagesWhenAnAgentFails(t *testing.T) {
	plan := coordinateTestPlan(CoordinateIndependent, 0)
	plan.Agents = plan.Agents[:2]
	run, err := RunCoordinate(context.Background(), plan, "Question", func(_ context.Context, agent CoordinateAgent, _ string) (string, error) {
		if agent.Model == plan.Agents[1].Model {
			return "", errors.New("agent unavailable")
		}
		return "draft retained", nil
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "agent unavailable") {
		t.Fatalf("RunCoordinate error = %v, want agent failure", err)
	}
	assertCoordinateRunReport(t, run, []string{"Agent 1", "Agent 2", "Synthesis"}, "")
	report := reflect.ValueOf(run).Elem()
	stages := report.FieldByName("Stages")
	if got := stages.Index(0).FieldByName("Response").String(); got != "draft retained" {
		t.Fatalf("completed response = %q, want retained draft", got)
	}
	if got := stages.Index(0).FieldByName("Status").String(); got != "completed" {
		t.Fatalf("first stage status = %q, want completed", got)
	}
	if got := stages.Index(1).FieldByName("Status").String(); got != "failed" {
		t.Fatalf("second stage status = %q, want failed", got)
	}
	if got := stages.Index(2).FieldByName("Status").String(); got != "queued" {
		t.Fatalf("synthesis status = %q, want queued", got)
	}
}

func TestCoordinateProgressSnapshotIsConcurrentAndStable(t *testing.T) {
	stages := buildCoordinateStages(coordinateTestPlan(CoordinateIndependent, 0))
	snapshot := NewCoordinateProgressSnapshot(stages)
	var wait sync.WaitGroup
	for _, stage := range stages[:2] {
		stage := stage
		wait.Add(1)
		go func() {
			defer wait.Done()
			snapshot.Update(CoordinateStageEvent{ID: stage.ID, Label: stage.Label, Status: CoordinateStageRunning, Attempt: 1, At: time.Now()})
			snapshot.Update(CoordinateStageEvent{ID: stage.ID, Label: stage.Label, Status: CoordinateStageRetrying, Attempt: 1, At: time.Now()})
			snapshot.Update(CoordinateStageEvent{ID: stage.ID, Label: stage.Label, Status: CoordinateStageRunning, Attempt: 2, At: time.Now()})
			snapshot.Update(CoordinateStageEvent{ID: stage.ID, Label: stage.Label, Status: CoordinateStageCompleted, Attempt: 2, At: time.Now()})
		}()
	}
	wait.Wait()
	got := snapshot.Snapshot()
	if len(got) != 3 || got[0].ID != "agent-1" || got[1].ID != "agent-2" || got[2].ID != "synthesis" {
		t.Fatalf("snapshot stage order = %#v", got)
	}
	for i := 0; i < 2; i++ {
		if got[i].Status != CoordinateStageCompleted || got[i].Attempt != 2 {
			t.Errorf("stage %s = status %q attempt %d, want completed attempt 2", got[i].ID, got[i].Status, got[i].Attempt)
		}
	}
	if got[2].Status != CoordinateStageQueued {
		t.Fatalf("synthesis status = %q, want queued", got[2].Status)
	}
}

func TestCoordinateResumeRetriesOnlyIncompleteStages(t *testing.T) {
	plan := coordinateTestPlan(CoordinateIndependent, 0)
	plan.Agents = plan.Agents[:2]
	var mu sync.Mutex
	calls := map[string]int{}
	var events []CoordinateStageEvent
	failedPrompt, retriedPrompt := "", ""
	firstCompleted := make(chan struct{})
	runner := func(_ context.Context, agent CoordinateAgent, prompt string) (string, error) {
		if strings.Contains(prompt, "Write one clear final answer") {
			mu.Lock()
			calls["synthesis"]++
			mu.Unlock()
			return "final", nil
		}
		mu.Lock()
		calls[agent.Role]++
		attempt := calls[agent.Role]
		mu.Unlock()
		if agent.Role == plan.Agents[0].Role {
			return "first draft", nil
		}
		if attempt == 1 {
			mu.Lock()
			failedPrompt = prompt
			mu.Unlock()
			<-firstCompleted
			return "", errors.New("temporary agent failure")
		}
		mu.Lock()
		retriedPrompt = prompt
		mu.Unlock()
		return "second draft", nil
	}
	progress := func(event CoordinateStageEvent) {
		mu.Lock()
		events = append(events, event)
		mu.Unlock()
		if event.ID == "agent-1" && event.Status == CoordinateStageCompleted {
			close(firstCompleted)
		}
	}
	run, err := RunCoordinate(context.Background(), plan, "Question", runner, progress)
	if err == nil || run == nil {
		t.Fatalf("initial run = %#v, %v; want a report and failure", run, err)
	}
	run, err = ResumeCoordinate(context.Background(), run, runner, progress)
	if err != nil || run.FinalAnswer != "final" || !run.Complete {
		t.Fatalf("resumed run = %#v, %v; want complete final answer", run, err)
	}
	mu.Lock()
	defer mu.Unlock()
	if calls[plan.Agents[0].Role] != 1 || calls[plan.Agents[1].Role] != 2 || calls["synthesis"] != 1 {
		t.Fatalf("stage calls = %#v, want first once, second twice, synthesis once", calls)
	}
	if failedPrompt == "" || failedPrompt != retriedPrompt {
		t.Fatalf("retried stage prompt differs from its first attempt: first=%q retry=%q", failedPrompt, retriedPrompt)
	}
	gotRetrying := false
	for _, event := range events {
		if event.ID == "agent-2" && event.Status == CoordinateStageRetrying && event.Attempt == 2 {
			gotRetrying = true
			break
		}
	}
	if !gotRetrying {
		t.Fatalf("progress events = %#v; want agent-2 retrying on attempt 2", events)
	}
}

func TestCoordinateResumeContinuesChainDependencies(t *testing.T) {
	plan := coordinateTestPlan(CoordinateChain, 0)
	var calls [3]int
	runner := func(_ context.Context, agent CoordinateAgent, prompt string) (string, error) {
		for i, candidate := range plan.Agents {
			if candidate.Model == agent.Model {
				calls[i]++
			}
		}
		if agent.Model == plan.Agents[1].Model && calls[1] == 1 {
			return "", errors.New("critique unavailable")
		}
		if strings.Contains(prompt, "Write one clear final answer") {
			return "chain final", nil
		}
		return "stage output", nil
	}
	run, err := RunCoordinate(context.Background(), plan, "Question", runner, nil)
	if err == nil || run == nil {
		t.Fatalf("initial chain = %#v, %v; want failure", run, err)
	}
	run, err = ResumeCoordinate(context.Background(), run, runner, nil)
	if err != nil || !run.Complete || run.FinalAnswer != "chain final" {
		t.Fatalf("resumed chain = %#v, %v; want final answer", run, err)
	}
	if calls != [3]int{1, 2, 1} {
		t.Fatalf("stage calls = %v, want [1 2 1]", calls)
	}
}

func TestCoordinateResumePreservesRateLimitErrorIdentity(t *testing.T) {
	plan := coordinateTestPlan(CoordinateChain, 0)
	calls := 0
	runner := func(_ context.Context, _ CoordinateAgent, _ string) (string, error) {
		calls++
		if calls == 2 || calls == 3 {
			return "", &RateLimitError{Code: "ERR_RATE_LIMIT"}
		}
		return "first output", nil
	}
	run, err := RunCoordinate(context.Background(), plan, "Question", runner, nil)
	if err == nil || run == nil {
		t.Fatalf("initial run = %#v, %v; want 429", run, err)
	}
	_, err = ResumeCoordinate(context.Background(), run, runner, nil)
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("resume error = %v, want ErrRateLimited", err)
	}
}

func TestCoordinateParallelFailurePreservesRateLimitErrorIdentity(t *testing.T) {
	plan := coordinateTestPlan(CoordinateIndependent, 0)
	plan.Agents = plan.Agents[:2]
	plan.Parallel = true
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	runner := func(_ context.Context, agent CoordinateAgent, _ string) (string, error) {
		started <- struct{}{}
		<-release
		if agent.Model == plan.Agents[0].Model {
			return "", errors.New("first agent failed")
		}
		return "", &RateLimitError{Code: "ERR_RATE_LIMIT"}
	}
	result := make(chan error, 1)
	go func() {
		_, err := RunCoordinate(context.Background(), plan, "Question", runner, nil)
		result <- err
	}()
	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			close(release)
			<-result
			t.Fatal("both parallel agents did not enter the runner")
		}
	}
	close(release)
	err := <-result
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("parallel failure = %v, want to preserve the 429 error identity", err)
	}
}

func TestCoordinatePartialSynthesisMarksMissingStages(t *testing.T) {
	plan := coordinateTestPlan(CoordinateIndependent, 0)
	plan.Agents = plan.Agents[:2]
	var completed string
	runner := func(_ context.Context, agent CoordinateAgent, prompt string) (string, error) {
		if strings.Contains(prompt, "Write a partial answer") {
			if !strings.Contains(prompt, "Agent 1") || !strings.Contains(prompt, "first usable draft") || !strings.Contains(prompt, "Agent 2") || !strings.Contains(prompt, "missing") {
				t.Errorf("partial synthesis prompt omitted available or missing stage context: %q", prompt)
			}
			return "incomplete final", nil
		}
		if agent.Model == plan.Agents[0].Model {
			return "first usable draft", nil
		}
		return "", errors.New("second agent unavailable")
	}
	run, err := RunCoordinate(context.Background(), plan, "Question", runner, nil)
	if err == nil || run == nil {
		t.Fatalf("initial run = %#v, %v; want incomplete report", run, err)
	}
	for _, stage := range run.Stages {
		if stage.ID == "agent-1" && stage.Status == CoordinateStageCompleted {
			completed = stage.Response
		}
	}
	if completed != "first usable draft" {
		t.Fatalf("retained completed output = %q", completed)
	}
	run, err = SynthesizeCoordinatePartial(context.Background(), run, runner, nil)
	if err != nil || !run.Partial || run.Complete || !strings.Contains(run.FinalAnswer, "incomplete final") || !strings.Contains(run.FinalAnswer, "Partial coordinate result") {
		t.Fatalf("partial run = %#v, %v; want accepted partial final", run, err)
	}
}

func TestCoordinatePlanExecutionEstimate(t *testing.T) {
	tests := []struct {
		name   string
		plan   CoordinatePlan
		calls  int
		fanout int
	}{
		{name: "independent sequential", plan: coordinateTestPlan(CoordinateIndependent, 0), calls: 3, fanout: 1},
		{name: "independent parallel", plan: parallelCoordinateTestPlan(CoordinateIndependent, 0), calls: 3, fanout: 2},
		{name: "chain", plan: parallelCoordinateTestPlan(CoordinateChain, 0), calls: 3, fanout: 1},
		{name: "debate one round", plan: coordinateTestPlan(CoordinateDebate, 1), calls: 5, fanout: 1},
		{name: "debate two rounds parallel", plan: parallelCoordinateTestPlan(CoordinateDebate, 2), calls: 7, fanout: 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.plan.RequestCount(); got != tc.calls {
				t.Errorf("RequestCount() = %d, want %d", got, tc.calls)
			}
			if got := tc.plan.MaxParallelism(); got != tc.fanout {
				t.Errorf("MaxParallelism() = %d, want %d", got, tc.fanout)
			}
		})
	}
}

func TestCoordinateProvenanceShowsCompletedAndMissingStages(t *testing.T) {
	plan := coordinateTestPlan(CoordinateIndependent, 0)
	complete := &CoordinateRun{Plan: plan, Complete: true, FinalAnswer: "final answer", Stages: buildCoordinateStages(plan)}
	for i := range complete.Stages {
		complete.Stages[i].Status = CoordinateStageCompleted
		complete.Stages[i].Response = "private agent output"
	}
	completeText := flattenCoordinatePanel(coordinateProvenancePanelLines(complete))
	for _, want := range []string{
		models.DisplayName(plan.Agents[0].Model), plan.Agents[0].Role,
		models.DisplayName(plan.Agents[1].Model), plan.Agents[1].Role,
		models.DisplayName(plan.Agents[2].Model), plan.Agents[2].Role,
		"Agent 1", "Agent 2", "Synthesis", "All planned outputs used", "unresolved disagreements",
	} {
		if !strings.Contains(completeText, want) {
			t.Errorf("complete provenance = %q; missing %q", completeText, want)
		}
	}
	if strings.Contains(completeText, "private agent output") {
		t.Fatal("provenance panel exposed a full intermediate response")
	}

	partial := &CoordinateRun{Plan: plan, Partial: true, FinalAnswer: "partial answer", Stages: buildCoordinateStages(plan)}
	partial.Stages[0].Status = CoordinateStageCompleted
	partial.Stages[0].Response = "private usable draft"
	partial.Stages[1].Status = CoordinateStageFailed
	partial.Stages[1].Err = "agent unavailable"
	partial.Stages[2].Status = CoordinateStageCompleted
	partial.Stages[2].Response = "private partial synthesis"
	partialText := flattenCoordinatePanel(coordinateProvenancePanelLines(partial))
	for _, want := range []string{"Partial result", "Agent 2", "Missing outputs", "available outputs", "unresolved disagreements"} {
		if !strings.Contains(partialText, want) {
			t.Errorf("partial provenance = %q; missing %q", partialText, want)
		}
	}
	if strings.Contains(partialText, "private usable draft") || strings.Contains(partialText, "private partial synthesis") {
		t.Fatal("partial provenance panel exposed a full intermediate response")
	}
}

func flattenCoordinatePanel(lines []ui.PanelLine) string {
	var flattened strings.Builder
	for _, line := range lines {
		for _, segment := range line {
			flattened.WriteString(segment.Text)
		}
		flattened.WriteByte('\n')
	}
	return flattened.String()
}

func TestCoordinateIndependentKeepsDraftsSeparateUntilSynthesis(t *testing.T) {
	var prompts []string
	runner := func(_ context.Context, _ CoordinateAgent, prompt string) (string, error) {
		prompts = append(prompts, prompt)
		return []string{"first-draft-secret", "second-draft-secret", "final answer"}[len(prompts)-1], nil
	}
	run, err := RunCoordinate(context.Background(), coordinateTestPlan(CoordinateIndependent, 0), "How does this work?", runner, nil)
	if err != nil || run.FinalAnswer != "final answer" {
		t.Fatalf("RunCoordinate() = %q, %v; want final answer", run.FinalAnswer, err)
	}
	if len(prompts) != 3 {
		t.Fatalf("calls = %d, want 3", len(prompts))
	}
	if strings.Contains(prompts[1], "first-draft-secret") {
		t.Fatal("second independent agent saw the first draft")
	}
	if !strings.Contains(prompts[2], "first-draft-secret") || !strings.Contains(prompts[2], "second-draft-secret") {
		t.Fatal("synthesizer did not receive both drafts")
	}
}

func TestCoordinateIndependentParallelRunsFirstAgentsConcurrently(t *testing.T) {
	plan := coordinateTestPlan(CoordinateIndependent, 0)
	plan.Parallel = true
	started := make(chan models.Model, 2)
	release := make(chan struct{})
	type result struct {
		answer string
		err    error
	}
	done := make(chan result, 1)
	runner := func(ctx context.Context, agent CoordinateAgent, _ string) (string, error) {
		if agent.Model == plan.Agents[2].Model {
			return "final answer", nil
		}
		started <- agent.Model
		select {
		case <-release:
			return string(agent.Model), nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	go func() {
		run, err := RunCoordinate(context.Background(), plan, "Question", runner, nil)
		var answer string
		if run != nil {
			answer = run.FinalAnswer
		}
		done <- result{answer: answer, err: err}
	}()

	for count := 0; count < 2; count++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			close(release)
			<-done
			t.Fatal("the first two independent agents did not overlap")
		}
	}
	close(release)
	got := <-done
	if got.err != nil || got.answer != "final answer" {
		t.Fatalf("parallel coordination = %q, %v", got.answer, got.err)
	}
}

func TestCoordinateParallel429CancelsOtherAgentAndStops(t *testing.T) {
	plan := coordinateTestPlan(CoordinateIndependent, 0)
	plan.Parallel = true
	started := make(chan struct{}, 1)
	runner := func(ctx context.Context, agent CoordinateAgent, _ string) (string, error) {
		switch agent.Model {
		case plan.Agents[0].Model:
			<-started
			return "", &RateLimitError{Code: "ERR_RATE_LIMIT"}
		case plan.Agents[1].Model:
			started <- struct{}{}
			<-ctx.Done()
			return "", ctx.Err()
		default:
			t.Error("synthesis must not run after a persistent 429")
			return "", nil
		}
	}
	_, err := RunCoordinate(context.Background(), plan, "Question", runner, nil)
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("RunCoordinate() error = %v, want persistent rate limit", err)
	}
}

func TestCoordinateChainPassesCritiqueToFinalAgent(t *testing.T) {
	var prompts []string
	runner := func(_ context.Context, _ CoordinateAgent, prompt string) (string, error) {
		prompts = append(prompts, prompt)
		return []string{"draft", "critical correction", "corrected final"}[len(prompts)-1], nil
	}
	run, err := RunCoordinate(context.Background(), coordinateTestPlan(CoordinateChain, 0), "Question", runner, nil)
	if err != nil || run.FinalAnswer != "corrected final" {
		t.Fatalf("RunCoordinate() = %q, %v", run.FinalAnswer, err)
	}
	if len(prompts) != 3 || !strings.Contains(prompts[1], "draft") || !strings.Contains(prompts[2], "critical correction") {
		t.Fatalf("chain did not pass draft and critique in order: %q", prompts)
	}
}

func TestCoordinateDebateStopsAfterConfiguredRounds(t *testing.T) {
	for _, tc := range []struct{ rounds, wantCalls int }{{1, 5}, {2, 7}} {
		calls := 0
		runner := func(_ context.Context, _ CoordinateAgent, _ string) (string, error) {
			calls++
			return "answer", nil
		}
		_, err := RunCoordinate(context.Background(), coordinateTestPlan(CoordinateDebate, tc.rounds), "Question", runner, nil)
		if err != nil || calls != tc.wantCalls {
			t.Errorf("rounds=%d: calls=%d, err=%v; want %d", tc.rounds, calls, err, tc.wantCalls)
		}
	}
}

func TestCoordinateTwoAgentsReuseSelectedModelForSynthesis(t *testing.T) {
	plan := coordinateTestPlan(CoordinateIndependent, 0)
	plan.Agents = plan.Agents[:2]
	var called []models.Model
	runner := func(_ context.Context, agent CoordinateAgent, _ string) (string, error) {
		called = append(called, agent.Model)
		return "answer", nil
	}
	_, err := RunCoordinate(context.Background(), plan, "Question", runner, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(called) != 3 || called[2] != models.GPT6Luna {
		t.Fatalf("models used = %v; want first model to synthesize", called)
	}
}

func TestCoordinatePersistentRateLimitStopsRemainingAgents(t *testing.T) {
	calls := 0
	runner := func(_ context.Context, _ CoordinateAgent, _ string) (string, error) {
		calls++
		return "", &RateLimitError{Code: "ERR_RATE_LIMIT"}
	}
	_, err := RunCoordinate(context.Background(), coordinateTestPlan(CoordinateDebate, 2), "Question", runner, nil)
	if !errors.Is(err, ErrRateLimited) || calls != 1 {
		t.Fatalf("calls=%d, error=%v; want first persistent 429 returned", calls, err)
	}
}

func TestCoordinateMarksTruncatedAgentOutputForSynthesis(t *testing.T) {
	longDraft := strings.Repeat("x", 13000)
	calls := 0
	var synthesisPrompt string
	runner := func(_ context.Context, _ CoordinateAgent, prompt string) (string, error) {
		calls++
		if calls == 3 {
			synthesisPrompt = prompt
			return "final", nil
		}
		return longDraft, nil
	}
	_, err := RunCoordinate(context.Background(), coordinateTestPlan(CoordinateIndependent, 0), "Question", runner, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(synthesisPrompt, "[truncated]") || strings.Contains(synthesisPrompt, strings.Repeat("x", 13000)) {
		t.Fatal("synthesis prompt did not bound and mark long agent output")
	}
}

func TestCoordinateRejectsModelAliasBeforeMakingRequest(t *testing.T) {
	plan := coordinateTestPlan(CoordinateIndependent, 0)
	plan.Agents[0].Model = models.Model("gpt-oss-120b")
	calls := 0
	_, err := RunCoordinate(context.Background(), plan, "Question", func(_ context.Context, _ CoordinateAgent, _ string) (string, error) {
		calls++
		return "answer", nil
	}, nil)
	if err == nil || calls != 0 {
		t.Fatalf("error=%v, calls=%d; want alias rejected before sending invalid model ID", err, calls)
	}
}
