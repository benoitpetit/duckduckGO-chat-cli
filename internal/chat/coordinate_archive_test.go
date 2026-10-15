package chat

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"duckduckgo-chat-cli/internal/analytics"
	"duckduckgo-chat-cli/internal/models"
	"duckduckgo-chat-cli/internal/persistence"

	"github.com/AlecAivazis/survey/v2/terminal"
)

type fixedCoordinateArchivePrompt struct {
	save  bool
	err   error
	calls int
}

func (p *fixedCoordinateArchivePrompt) ChooseSave(*CoordinateRun) (bool, error) {
	p.calls++
	return p.save, p.err
}

func coordinateArchiveTestRun(partial bool) *CoordinateRun {
	started := time.Date(2026, 10, 13, 10, 0, 0, 0, time.UTC)
	run := &CoordinateRun{
		Plan: CoordinatePlan{Strategy: CoordinateIndependent}, Question: "Compare both proposals",
		StartedAt: started, FinalAnswer: "The second proposal is clearer.", Complete: !partial, Partial: partial,
		Stages: []CoordinateStage{
			{ID: "agent-1", Label: "Agent 1", Agent: CoordinateAgent{Model: models.GPT6Luna, Role: "Analyst"}, Status: CoordinateStageCompleted, Attempt: 1, Prompt: "first private prompt", Response: "first private output", StartedAt: started, FinishedAt: started.Add(time.Second)},
			{ID: "agent-2", Label: "Agent 2", Agent: CoordinateAgent{Model: models.GPT5Luna, Role: "Verifier"}, Status: CoordinateStageCompleted, Attempt: 2, Prompt: "second private prompt", Response: "second private output", StartedAt: started.Add(time.Second), FinishedAt: started.Add(2 * time.Second)},
		},
	}
	if partial {
		run.Stages[1].Status = CoordinateStageFailed
		run.Stages[1].Err = "second model unavailable"
	}
	return run
}

func TestCoordinateReportFromRunPreservesStageIdentityAndBoundsText(t *testing.T) {
	run := coordinateArchiveTestRun(false)
	longText := strings.Repeat("é", persistence.CoordinateReportTextLimit)
	run.Question = longText
	run.FinalAnswer = longText
	run.Stages[0].Prompt = longText
	run.Stages[0].Response = longText
	report, err := coordinateReportFromRun(run)
	if err != nil {
		t.Fatalf("coordinateReportFromRun() error = %v", err)
	}
	if !report.Complete || report.Partial || !report.QuestionTruncated || !report.FinalAnswerTruncated {
		t.Fatalf("report status/truncation = %+v", report)
	}
	if len(report.Question) > persistence.CoordinateReportTextLimit || !utf8.ValidString(report.Question) || !strings.HasSuffix(report.Question, "[truncated]") {
		t.Fatalf("truncated question length/encoding/marker invalid: bytes=%d valid=%t", len(report.Question), utf8.ValidString(report.Question))
	}
	if len(report.Stages) != 2 || report.Stages[0].ID != "agent-1" || report.Stages[1].Role != "Verifier" || !report.Stages[0].PromptTruncated || !report.Stages[0].ResponseTruncated {
		t.Fatalf("stage mapping = %+v", report.Stages)
	}
}

func TestCoordinateReportFromRunPreservesPartialStatusesAndErrors(t *testing.T) {
	report, err := coordinateReportFromRun(coordinateArchiveTestRun(true))
	if err != nil {
		t.Fatalf("coordinateReportFromRun() error = %v", err)
	}
	if report.Complete || !report.Partial || report.Stages[1].Status != string(CoordinateStageFailed) || report.Stages[1].Error != "second model unavailable" {
		t.Fatalf("partial report mapping = %+v", report)
	}
}

func TestCoordinateArchiveDeclineDoesNotPersist(t *testing.T) {
	for _, test := range []struct {
		name   string
		prompt *fixedCoordinateArchivePrompt
	}{
		{name: "decline", prompt: &fixedCoordinateArchivePrompt{}},
		{name: "cancel", prompt: &fixedCoordinateArchivePrompt{err: terminal.InterruptErr}},
	} {
		t.Run(test.name, func(t *testing.T) {
			manager := persistence.NewHistoryManager(t.TempDir())
			c := &Chat{SessionID: "session_archive_decline", Model: models.Default(), HistoryManager: manager, Messages: []Message{{Role: "user", Content: "Question"}, {Role: "assistant", Content: "Answer"}}, coordinateArchivePrompt: test.prompt}
			if err := c.saveCoordinateReportIfRequested(coordinateArchiveTestRun(false)); err != nil {
				t.Fatalf("save choice returned error: %v", err)
			}
			if len(c.CoordinateReports) != 0 || test.prompt.calls != 1 {
				t.Fatalf("declined report state=%+v prompt calls=%d", c.CoordinateReports, test.prompt.calls)
			}
			if _, err := manager.LoadSession(c.SessionID); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("decline archive lookup error = %v, want no archive", err)
			}
		})
	}
}

func TestCoordinateArchiveSaveFailureRollsBackInMemoryReport(t *testing.T) {
	c := &Chat{SessionID: "session_archive_failure", Model: models.Default(), Messages: []Message{{Role: "user", Content: "Question"}, {Role: "assistant", Content: "Answer"}}, coordinateArchivePrompt: &fixedCoordinateArchivePrompt{save: true}}
	err := c.saveCoordinateReportIfRequested(coordinateArchiveTestRun(false))
	if err == nil || !strings.Contains(err.Error(), "conversation history manager") {
		t.Fatalf("save error = %v, want unavailable history manager", err)
	}
	if len(c.CoordinateReports) != 0 {
		t.Fatalf("failed save left in-memory report: %+v", c.CoordinateReports)
	}
}

func TestCoordinateArchiveRoundTripKeepsStagesOutOfMessages(t *testing.T) {
	manager := persistence.NewHistoryManager(t.TempDir())
	tracker := analytics.NewChatAnalytics()
	tracker.RecordMessage("user", len("Question"))
	tracker.RecordMessage("assistant", len("Answer"))
	c := &Chat{
		SessionID: "session_archive_roundtrip", Model: models.Default(), HistoryManager: manager,
		Analytics: tracker, Messages: []Message{{Role: "user", Content: "Question"}, {Role: "assistant", Content: "Answer"}},
		coordinateArchivePrompt: &fixedCoordinateArchivePrompt{save: true},
	}
	beforeAnalytics := tracker.Snapshot()
	if err := c.saveCoordinateReportIfRequested(coordinateArchiveTestRun(false)); err != nil {
		t.Fatalf("save coordinate report: %v", err)
	}
	if !reflect.DeepEqual(beforeAnalytics, tracker.Snapshot()) {
		t.Fatal("saving a coordinate report changed chat analytics")
	}
	for _, message := range c.Messages {
		if strings.Contains(message.Content, "private") {
			t.Fatalf("intermediate output entered normal chat history: %+v", c.Messages)
		}
	}
	saved, err := manager.LoadSession(c.SessionID)
	if err != nil {
		t.Fatalf("load archive: %v", err)
	}
	if len(saved.CoordinateReports) != 1 || len(saved.Messages) != 2 {
		t.Fatalf("saved session messages=%d reports=%d", len(saved.Messages), len(saved.CoordinateReports))
	}
	analyticsJSON, err := json.Marshal(saved.Analytics)
	if err != nil || strings.Contains(string(analyticsJSON), "private") {
		t.Fatalf("intermediate outputs leaked into analytics: %s err=%v", analyticsJSON, err)
	}
	restored := &Chat{}
	restored.RestoreContext(saved)
	if len(restored.Messages) != 2 || len(restored.CoordinateReports) != 1 || restored.CoordinateReports[0].Stages[0].Response != "first private output" {
		t.Fatalf("restored transcript/reports = messages:%+v reports:%+v", restored.Messages, restored.CoordinateReports)
	}
	contextJSON, err := json.Marshal(restored.convertMessagesToIntelligence())
	if err != nil || strings.Contains(string(contextJSON), "first private output") {
		t.Fatalf("coordinate stage output entered subsequent model context: %s err=%v", contextJSON, err)
	}
}
