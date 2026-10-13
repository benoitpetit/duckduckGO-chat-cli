package chat

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"duckduckgo-chat-cli/internal/persistence"
	"duckduckgo-chat-cli/internal/ui"
)

func TestCoordinateRunsListNewestFirstWithoutIntermediateOutputs(t *testing.T) {
	older, err := coordinateReportFromRun(coordinateArchiveTestRun(false))
	if err != nil {
		t.Fatal(err)
	}
	newer, err := coordinateReportFromRun(coordinateArchiveTestRun(true))
	if err != nil {
		t.Fatal(err)
	}
	older.ID, newer.ID = "coordinate_older", "coordinate_newer"
	newer.Strategy = "chain\x1b[31m"
	older.FinishedAt = time.Now().Add(-time.Hour)
	newer.FinishedAt = time.Now()
	c := &Chat{CoordinateReports: []persistence.CoordinateReport{older, newer}}
	title, lines, err := coordinateHistoryCommand(c, []string{"runs"})
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	joined := coordinatePanelText(lines)
	if title != "Saved coordinate runs" || strings.Index(joined, "coordinate_newer") > strings.Index(joined, "coordinate_older") {
		t.Fatalf("run list title/order = %q %q", title, joined)
	}
	if !strings.Contains(joined, "partial") || strings.Contains(joined, "first private output") || strings.ContainsRune(joined, '\x1b') {
		t.Fatalf("run list omitted status or leaked intermediate content: %q", joined)
	}
}

func TestCoordinateShowFormatsSelectedReportAndStripsTerminalControls(t *testing.T) {
	run := coordinateArchiveTestRun(false)
	run.Question = "Question with color \x1b[31mtext\x1b[0m"
	run.Stages[0].Prompt += "\x07"
	report, err := coordinateReportFromRun(run)
	if err != nil {
		t.Fatal(err)
	}
	report.ID = "coordinate_show"
	c := &Chat{CoordinateReports: []persistence.CoordinateReport{report}, Messages: []Message{{Role: "user", Content: "existing"}}}
	beforeMessages := append([]Message(nil), c.Messages...)
	beforeReports := cloneCoordinateReports(c.CoordinateReports)
	title, lines, err := coordinateHistoryCommand(c, []string{"show", report.ID})
	if err != nil {
		t.Fatalf("show report: %v", err)
	}
	joined := coordinatePanelText(lines)
	for _, want := range []string{"Question with color", "Analyst", "first private prompt", "first private output", "The second proposal is clearer."} {
		if !strings.Contains(joined, want) {
			t.Errorf("coordinate detail is missing %q: %q", want, joined)
		}
	}
	if title != "Coordinate report · "+report.ID || strings.ContainsRune(joined, '\x1b') || strings.ContainsRune(joined, '\x07') {
		t.Fatalf("report title/output has unsafe controls: title=%q output=%q", title, joined)
	}
	if !reflect.DeepEqual(c.Messages, beforeMessages) || !reflect.DeepEqual(c.CoordinateReports, beforeReports) {
		t.Fatal("reading a coordinate report mutated chat state")
	}
}

func TestCoordinateShowRejectsUnknownIDAndInvalidArguments(t *testing.T) {
	c := &Chat{CoordinateReports: []persistence.CoordinateReport{}}
	for _, args := range [][]string{{"runs", "extra"}, {"show"}, {"show", "one", "two"}, {"show", "coordinate_missing"}, {"unknown"}} {
		if _, _, err := coordinateHistoryCommand(c, args); err == nil {
			t.Errorf("coordinateHistoryCommand(%v) succeeded, want validation error", args)
		}
	}
}

func TestCoordinateRunsEmptyArchiveHasHelpfulMessage(t *testing.T) {
	title, lines, err := coordinateHistoryCommand(&Chat{}, []string{"runs"})
	if err != nil {
		t.Fatal(err)
	}
	if title != "Saved coordinate runs" || !strings.Contains(coordinatePanelText(lines), "No saved coordinate reports") {
		t.Fatalf("empty run list = %q %q", title, coordinatePanelText(lines))
	}
}

func coordinatePanelText(lines []ui.PanelLine) string {
	var values []string
	for _, line := range lines {
		var row strings.Builder
		for _, cell := range line {
			row.WriteString(cell.Text)
		}
		values = append(values, row.String())
	}
	return strings.Join(values, "\n")
}
