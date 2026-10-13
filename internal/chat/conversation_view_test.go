package chat

import (
	"os"
	"strings"
	"testing"

	"github.com/fatih/color"

	"duckduckgo-chat-cli/internal/ptytest"
	"duckduckgo-chat-cli/internal/ui"
)

func TestCoordinateProgressPanelReplacesPriorSnapshot(t *testing.T) {
	view := NewConversationView("", "", false)
	view.RecordPanel("Coordinate progress", []ui.PanelLine{{{Text: "queued"}}})
	lines := []ui.PanelLine{{{Text: "completed"}}}
	view.UpdatePanel("Coordinate progress", lines)
	if len(view.entries) != 1 || view.entries[0].panelLines[0][0].Text != "completed" {
		t.Fatalf("progress panel state = %+v, want one updated panel", view.entries)
	}
}

func TestOptimizationNoticeSurvivesTerminalResize(t *testing.T) {
	master, slave, err := ptytest.Open()
	if err != nil {
		t.Skipf("cannot allocate a pseudo-terminal: %v", err)
	}
	defer master.Close()
	if err := ptytest.SetSize(slave, 72, 24); err != nil {
		t.Fatalf("set pty size: %v", err)
	}

	previousStdout, previousColorOutput, previousNoColor := os.Stdout, color.Output, color.NoColor
	defer func() { os.Stdout, color.Output, color.NoColor = previousStdout, previousColorOutput, previousNoColor }()
	os.Stdout, color.Output, color.NoColor = slave, slave, true

	view := NewConversationView("", "", false)
	view.RecordPanel("Context optimization", []ui.PanelLine{{{Text: "Conversation context was automatically reduced."}}})
	got, err := ptytest.Capture(master, slave, func() error {
		view.Redraw(72)
		return nil
	})
	if err != nil {
		t.Fatalf("capturing resized conversation view failed: %v", err)
	}
	if !strings.Contains(got, "Context optimization") || !strings.Contains(got, "automatically reduced") {
		t.Fatalf("resized view lost optimization notice: %q", got)
	}
}
