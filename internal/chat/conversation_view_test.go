package chat

import (
	"testing"

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
