package chat

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"duckduckgo-chat-cli/internal/config"
	"duckduckgo-chat-cli/internal/models"
	"duckduckgo-chat-cli/internal/persistence"
	"duckduckgo-chat-cli/internal/ui"

	surveyterminal "github.com/AlecAivazis/survey/v2/terminal"
	"golang.org/x/term"
)

// CoordinateArchivePrompt makes the per-run local persistence choice explicit.
type CoordinateArchivePrompt interface {
	ChooseSave(*CoordinateRun) (bool, error)
}

func coordinateReportFromRun(run *CoordinateRun) (persistence.CoordinateReport, error) {
	if run == nil {
		return persistence.CoordinateReport{}, errors.New("coordinate run is unavailable")
	}
	if run.Complete == run.Partial || strings.TrimSpace(run.FinalAnswer) == "" {
		return persistence.CoordinateReport{}, errors.New("coordinate run has no accepted final answer")
	}
	if !isCoordinateStrategy(run.Plan.Strategy) || strings.TrimSpace(run.Question) == "" || len(run.Stages) == 0 {
		return persistence.CoordinateReport{}, errors.New("coordinate run is missing report details")
	}
	now := time.Now()
	started := run.StartedAt
	if started.IsZero() {
		started = now
	}
	question, questionTruncated := truncateCoordinateReportText(run.Question)
	finalAnswer, finalTruncated := truncateCoordinateReportText(run.FinalAnswer)
	report := persistence.CoordinateReport{
		ID:                   fmt.Sprintf("coordinate_%d", now.UnixNano()),
		Strategy:             string(run.Plan.Strategy),
		Question:             question,
		QuestionTruncated:    questionTruncated,
		StartedAt:            started,
		FinishedAt:           now,
		Complete:             run.Complete,
		Partial:              run.Partial,
		FinalAnswer:          finalAnswer,
		FinalAnswerTruncated: finalTruncated,
		Stages:               make([]persistence.CoordinateReportStage, 0, len(run.Stages)),
	}
	for _, stage := range run.Stages {
		prompt, promptTruncated := truncateCoordinateReportText(stage.Prompt)
		response, responseTruncated := truncateCoordinateReportText(stage.Response)
		stageError, _ := truncateCoordinateReportText(stage.Err)
		report.Stages = append(report.Stages, persistence.CoordinateReportStage{
			ID: string(stage.ID), Label: stage.Label, Model: string(stage.Agent.Model), Role: stage.Agent.Role,
			Status: string(stage.Status), Attempt: stage.Attempt, StartedAt: stage.StartedAt, FinishedAt: stage.FinishedAt,
			Prompt: prompt, PromptTruncated: promptTruncated, Response: response, ResponseTruncated: responseTruncated, Error: stageError,
		})
	}
	return report, nil
}

func truncateCoordinateReportText(value string) (string, bool) {
	if len(value) <= persistence.CoordinateReportTextLimit {
		return value, false
	}
	const marker = "\n[truncated]"
	return truncateUTF8(value, persistence.CoordinateReportTextLimit-len(marker)) + marker, true
}

func (c *Chat) saveCoordinateReportIfRequested(run *CoordinateRun) error {
	prompt := c.coordinateArchivePrompt
	if prompt == nil {
		// Headless runs keep the privacy-preserving default because they cannot
		// make an explicit per-run choice.
		if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
			return nil
		}
		prompt = terminalCoordinateArchivePrompt{prompts: &terminalImagePromptSurvey{}}
	}
	save, err := prompt.ChooseSave(run)
	if errors.Is(err, surveyterminal.InterruptErr) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("could not choose coordinate report save option: %w", err)
	}
	if !save {
		return nil
	}
	report, err := coordinateReportFromRun(run)
	if err != nil {
		return err
	}
	previous := cloneCoordinateReports(c.CoordinateReports)
	c.CoordinateReports = append(c.CoordinateReports, report)
	if err := c.SaveCurrentSession(); err != nil {
		c.CoordinateReports = previous
		return fmt.Errorf("could not save coordinate report: %w", err)
	}
	return nil
}

func (c *Chat) SetNextCoordinate(plan CoordinatePlan) error {
	if err := plan.Validate(); err != nil {
		return err
	}
	copyPlan := plan
	copyPlan.Agents = append([]CoordinateAgent(nil), plan.Agents...)
	c.coordinateMu.Lock()
	c.coordinateNext = &copyPlan
	c.coordinateMu.Unlock()
	return nil
}

func (c *Chat) CancelNextCoordinate() {
	c.coordinateMu.Lock()
	c.coordinateNext = nil
	c.coordinateMu.Unlock()
}

func (c *Chat) HasNextCoordinate() bool {
	c.coordinateMu.Lock()
	defer c.coordinateMu.Unlock()
	return c.coordinateNext != nil
}

func (c *Chat) nextCoordinate() (CoordinatePlan, bool) {
	c.coordinateMu.Lock()
	defer c.coordinateMu.Unlock()
	if c.coordinateNext == nil {
		return CoordinatePlan{}, false
	}
	plan := *c.coordinateNext
	plan.Agents = append([]CoordinateAgent(nil), plan.Agents...)
	return plan, true
}

func (c *Chat) takeCoordinate() (CoordinatePlan, bool) {
	c.coordinateMu.Lock()
	defer c.coordinateMu.Unlock()
	if c.coordinateNext == nil {
		return CoordinatePlan{}, false
	}
	plan := *c.coordinateNext
	plan.Agents = append([]CoordinateAgent(nil), plan.Agents...)
	c.coordinateNext = nil
	return plan, true
}

func (c *Chat) validateCoordinateImages(plan CoordinatePlan) error {
	if len(c.pendingImages) == 0 && !hasImageAttachments(c.Messages) {
		return nil
	}
	for _, agent := range plan.Agents {
		if !models.SupportsImageInput(agent.Model) {
			return fmt.Errorf("%s cannot receive images through Duck.ai; choose an image-capable model in /coordinate", models.DisplayName(agent.Model))
		}
	}
	return nil
}

func cloneCoordinateMessages(messages []Message) []Message {
	cloned := make([]Message, len(messages))
	for i, message := range messages {
		cloned[i] = message
		cloned[i].Images = cloneImageAttachments(message.Images)
	}
	return cloned
}

// RunCoordinatedQuestion runs agents against one immutable snapshot of the
// current conversation and returns all stage results without committing them.
func (c *Chat) RunCoordinatedQuestion(ctx context.Context, plan CoordinatePlan, question string, cfg *config.Config, runner CoordinateRunner, progress CoordinateProgress) (*CoordinateRun, error) {
	if cfg == nil {
		return nil, errors.New("coordinate configuration is unavailable")
	}
	if err := plan.Validate(); err != nil {
		return nil, err
	}
	if err := c.validateCoordinateImages(plan); err != nil {
		return nil, err
	}
	if strings.TrimSpace(question) == "" {
		return nil, errors.New("coordinate question is empty")
	}
	if runner == nil {
		runner = c.coordinateSessionRunner(cfg)
	}
	run, err := RunCoordinate(ctx, plan, question, runner, progress)
	if run != nil {
		run.UserContent = question
		if len(c.Messages) == 0 && cfg.GlobalPrompt != "" {
			run.UserContent = cfg.GlobalPrompt + "\n\n" + question
		}
	}
	return run, err
}

func (c *Chat) coordinateSessionRunner(cfg *config.Config) CoordinateRunner {
	if cfg == nil {
		return nil
	}
	history := cloneCoordinateMessages(c.Messages)
	images := cloneImageAttachments(c.pendingImages)
	capturedConfig := *cfg
	capturedConfig.Tools.ImageGeneration = false
	return func(ctx context.Context, agent CoordinateAgent, prompt string) (string, error) {
		isolatedConfig := capturedConfig
		isolated := NewChat("", "", "", "", agent.Model, &isolatedConfig)
		isolated.Messages = cloneCoordinateMessages(history)
		isolated.QueueImageAttachments(images)
		isolated.suppressSensitiveDebugLogs = true
		isolated.suppressDebugPayload = true
		return ProcessInputContext(ctx, isolated, prompt, &isolatedConfig)
	}
}

// CommitCoordinatedQuestion adds only an accepted complete or partial answer to
// the main conversation. Intermediate stage output stays run-local.
func (c *Chat) CommitCoordinatedQuestion(question string, run *CoordinateRun) error {
	if c == nil || run == nil {
		return errors.New("coordinate run is unavailable")
	}
	if run.committed {
		return errors.New("coordinate run was already committed")
	}
	if (!run.Complete && !run.Partial) || strings.TrimSpace(run.FinalAnswer) == "" {
		return errors.New("coordinate run has no accepted final answer")
	}
	userContent := run.UserContent
	if userContent == "" {
		userContent = question
	}
	if strings.TrimSpace(userContent) == "" {
		return errors.New("coordinate question is empty")
	}
	images := cloneImageAttachments(c.pendingImages)
	c.Messages = append(c.Messages,
		Message{Role: "user", Content: userContent, Images: images, Sources: cloneSourceRefs(c.PendingSources), Timestamp: time.Now()},
		Message{Role: "assistant", Content: run.FinalAnswer, Timestamp: time.Now()},
	)
	run.committed = true
	c.pendingImages = nil
	c.PendingSources = nil
	c.resetDurableConversation()
	if c.Analytics != nil {
		started := run.StartedAt
		if started.IsZero() {
			started = time.Now()
		}
		c.Analytics.RecordChatInteraction(time.Since(started), true, "")
		c.Analytics.RecordMessage("user", len(question))
		c.Analytics.RecordMessage("assistant", len(run.FinalAnswer))
	}
	return nil
}

func (c *Chat) processPendingCoordinate(input string, cfg *config.Config) bool {
	plan, ok := c.nextCoordinate()
	if !ok || strings.TrimSpace(input) == "" {
		return false
	}
	if err := c.validateCoordinateImages(plan); err != nil {
		ui.Errorln("%v", err)
		return true
	}
	plan, _ = c.takeCoordinate()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.setRequestCancel(cancel)
	defer c.clearRequestCancel()
	if c.Display != nil {
		c.Display.SetHeader("DuckDuckGo AI Chat CLI · Coordinate")
		c.Display.SetFramed(cfg.Appearance.FrameResponses)
	}
	runner := c.coordinateRunner
	if runner == nil {
		runner = c.coordinateSessionRunner(cfg)
	}
	var spinner *ui.Spinner
	startRun := func() *ui.Spinner {
		spinner = ui.StartSpinnerWithModel(ui.AccentColor.Sprint("Coordinate"))
		return spinner
	}
	spinner = startRun()
	run, err := c.runCoordinateWithProgress(ctx, buildCoordinateStages(plan), spinner, func(progress CoordinateProgress) (*CoordinateRun, error) {
		return c.RunCoordinatedQuestion(ctx, plan, input, cfg, runner, progress)
	})
	spinner.Stop()
	for err != nil && run != nil && ctx.Err() == nil {
		recovery := c.coordinateRecoveryPrompt
		if recovery == nil {
			recovery = terminalCoordinateRecoveryPrompt{prompts: &terminalImagePromptSurvey{}}
		}
		action, recoveryErr := recovery.ChooseRecovery(run)
		if recoveryErr != nil {
			if !errors.Is(recoveryErr, surveyterminal.InterruptErr) {
				err = recoveryErr
			}
			break
		}
		if action == CoordinateRecoveryStop {
			break
		}
		spinner = startRun()
		if action == CoordinateRecoveryResume {
			run, err = c.runCoordinateWithProgress(ctx, run.Stages, spinner, func(progress CoordinateProgress) (*CoordinateRun, error) {
				return ResumeCoordinate(ctx, run, runner, progress)
			})
		} else if action == CoordinateRecoverySynthesizePartial {
			run, err = c.runCoordinateWithProgress(ctx, run.Stages, spinner, func(progress CoordinateProgress) (*CoordinateRun, error) {
				return SynthesizeCoordinatePartial(ctx, run, runner, progress)
			})
		} else {
			err = fmt.Errorf("unknown coordinate recovery action %q", action)
		}
		spinner.Stop()
	}
	if err == nil && run != nil {
		err = c.CommitCoordinatedQuestion(input, run)
	}
	if err == nil && run != nil {
		if saveErr := c.saveCoordinateReportIfRequested(run); saveErr != nil {
			ui.Warningln("Could not save coordinate report; the answer remains in chat history: %v", saveErr)
		}
	}
	if err != nil {
		c.reportChatFailure(ctx, err)
		announceCoordinateEnded(c)
		return true
	}
	stream := make(chan string, 1)
	stream <- run.FinalAnswer
	close(stream)
	responseLabel := "Coordinate"
	if run.Partial {
		responseLabel = "Coordinate · partial"
	}
	// Keep coordinate results framed as a distinct mode, independent of the
	// global preference for ordinary chat responses.
	RenderStreamWithView(stream, responseLabel, spinner, true, c.Display)
	provenance := coordinateProvenancePanelLines(run)
	ui.PrintStyledPanel("Coordinate results", provenance)
	if c.Display != nil {
		c.Display.RecordAssistantWithFrame(responseLabel, run.FinalAnswer, true)
		c.Display.RecordPanel("Coordinate results", provenance)
	}
	announceCoordinateEnded(c)
	return true
}

func (c *Chat) runCoordinateWithProgress(ctx context.Context, stages []CoordinateStage, spinner *ui.Spinner, execute func(CoordinateProgress) (*CoordinateRun, error)) (*CoordinateRun, error) {
	snapshot := NewCoordinateProgressSnapshot(stages)
	wasCancelled := false
	updateView := func() {
		stages := snapshot.Snapshot()
		if !wasCancelled && spinner != nil {
			spinner.SetProgress(ui.ProgressPreparing, coordinateProgressLabel(stages))
		}
		if c.Display != nil {
			c.Display.UpdatePanel("Coordinate progress", coordinateProgressPanelLines(stages))
		}
	}
	updateView()
	type result struct {
		run *CoordinateRun
		err error
	}
	done := make(chan result, 1)
	go func() {
		run, err := execute(func(event CoordinateStageEvent) {
			snapshot.Update(event)
		})
		done <- result{run: run, err: err}
	}()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	cancelled := ctx.Done()
	for {
		select {
		case result := <-done:
			updateView()
			return result.run, result.err
		case <-ticker.C:
			updateView()
		case <-cancelled:
			cancelled = nil
			wasCancelled = true
			if spinner != nil {
				spinner.SetProgress(ui.ProgressCancelled, "Coordinate cancelled")
			}
		}
	}
}

func coordinateProgressLabel(stages []CoordinateStage) string {
	active := make([]string, 0, 2)
	for _, stage := range stages {
		if stage.Status == CoordinateStageRunning || stage.Status == CoordinateStageRetrying {
			label := stage.Label
			if name := models.DisplayName(stage.Agent.Model); name != "" {
				label += " · " + name
			}
			if stage.Status == CoordinateStageRetrying {
				label += " · retrying"
			}
			active = append(active, label)
		}
	}
	if len(active) == 0 {
		return "Preparing next coordinate stage"
	}
	return strings.Join(active, " + ")
}

func coordinateProgressPanelLines(stages []CoordinateStage) []ui.PanelLine {
	lines := make([]ui.PanelLine, 0, len(stages))
	for _, stage := range stages {
		statusStyle := ui.PanelMuted
		switch stage.Status {
		case CoordinateStageRunning:
			statusStyle = ui.PanelAccent
		case CoordinateStageRetrying:
			statusStyle = ui.PanelWarning
		case CoordinateStageCompleted:
			statusStyle = ui.PanelSuccess
		case CoordinateStageFailed:
			statusStyle = ui.PanelError
		case CoordinateStageCancelled:
			statusStyle = ui.PanelWarning
		}
		lines = append(lines, ui.PanelLine{
			{Text: stage.Label, Style: ui.PanelForeground},
			{Text: " · " + models.DisplayName(stage.Agent.Model), Style: ui.PanelMuted},
			{Text: " · " + string(stage.Status), Style: statusStyle},
		})
	}
	return lines
}

func coordinateProvenancePanelLines(run *CoordinateRun) []ui.PanelLine {
	if run == nil {
		return nil
	}
	outcome, outcomeStyle := "Complete answer", ui.PanelSuccess
	if run.Partial {
		outcome, outcomeStyle = "Partial result", ui.PanelWarning
	}
	lines := []ui.PanelLine{
		coordinatePanelField("Strategy", coordinateStrategyName(run.Plan.Strategy), ui.PanelForeground),
		coordinatePanelField("Outcome", outcome, outcomeStyle),
	}
	missing := make([]string, 0)
	completedCount, plannedCount := 0, 0
	for _, stage := range run.Stages {
		if stage.ID == "synthesis" {
			continue
		}
		plannedCount++
		if stage.Status == CoordinateStageCompleted {
			completedCount++
		} else {
			missing = append(missing, stage.Label)
		}
	}
	if run.Partial {
		missingSummary := strings.Join(missing, ", ")
		if missingSummary == "" {
			missingSummary = "None"
		}
		lines = append(lines,
			coordinatePanelField("Synthesis", "Used available outputs only", ui.PanelWarning),
			coordinatePanelField("Missing outputs", missingSummary, ui.PanelWarning),
		)
	} else {
		lines = append(lines, coordinatePanelField("Outputs", "All planned outputs used", ui.PanelSuccess))
	}
	lines = append(lines, coordinatePanelField("Disagreements", "Synthesis was asked to flag unresolved disagreements; claims are not independently verified.", ui.PanelMuted))
	finalAgentIndex := 0
	if len(run.Plan.Agents) == 3 {
		finalAgentIndex = 2
	}
	for agentIndex, agent := range run.Plan.Agents {
		lines = append(lines, coordinatePanelField(fmt.Sprintf("Agent %d", agentIndex+1), models.DisplayName(agent.Model)+" · "+agent.Role, ui.PanelForeground))
		completed, incomplete := make([]string, 0), make([]string, 0)
		for _, stage := range run.Stages {
			if !coordinateStageBelongsToAgent(stage.ID, agentIndex, finalAgentIndex) {
				continue
			}
			if stage.Status == CoordinateStageCompleted {
				completed = append(completed, stage.Label)
			} else {
				incomplete = append(incomplete, stage.Label)
			}
		}
		completedLabel := "none"
		if len(completed) > 0 {
			completedLabel = strings.Join(completed, ", ")
		}
		lines = append(lines, coordinatePanelField("Completed stages", completedLabel, ui.PanelForeground))
		if len(incomplete) > 0 {
			lines = append(lines, coordinatePanelField("Incomplete stages", strings.Join(incomplete, ", "), ui.PanelWarning))
		}
	}
	if plannedCount == 0 {
		lines = append(lines, ui.PanelLine{{Text: "No agent stages were recorded.", Style: ui.PanelMuted}})
	} else if !run.Partial && completedCount != plannedCount {
		lines = append(lines, ui.PanelLine{{Text: "Stage report is incomplete despite a complete answer.", Style: ui.PanelWarning}})
	}
	return lines
}

func coordinatePanelField(label, value string, style ui.PanelStyle) ui.PanelLine {
	return ui.PanelLine{
		{Text: label, Style: ui.PanelAccent},
		{Text: ": " + value, Style: style},
	}
}

func coordinateStageBelongsToAgent(stageID CoordinateStageID, agentIndex, finalAgentIndex int) bool {
	if stageID == "synthesis" {
		return agentIndex == finalAgentIndex
	}
	suffix := fmt.Sprintf("-agent-%d", agentIndex+1)
	return stageID == CoordinateStageID(fmt.Sprintf("agent-%d", agentIndex+1)) || strings.HasSuffix(string(stageID), suffix)
}

func announceCoordinateEnded(c *Chat) {
	const message = "Coordinate mode ended. The next question will use normal chat."
	if c.Display != nil {
		c.Display.RecordPanel("Coordinate complete", []ui.PanelLine{{{Text: message, Style: ui.PanelMuted}}})
	}
	ui.Mutedln("%s", message)
}
