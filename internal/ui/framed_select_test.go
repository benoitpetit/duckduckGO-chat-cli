package ui

import (
	"strings"
	"testing"
)

func TestFramedSelectSummaryKeepsMenuDetailsInsideFrame(t *testing.T) {
	lines := []PanelLine{
		{{Text: "Style        ", Style: PanelAccent}, {Text: "Photorealistic", Style: PanelForeground}},
		{{Text: "Reference    ", Style: PanelAccent}, {Text: "None", Style: PanelMuted}},
	}
	framed := framedSelectSummary(lines, 48)
	for _, want := range []string{"│", "Style", "Photorealistic", "Reference", "None"} {
		if !strings.Contains(framed, want) {
			t.Errorf("framed summary %q does not contain %q", framed, want)
		}
	}
	if !strings.Contains(framed, "Style        Photorealistic") && !strings.Contains(framed, "Style Photorealistic") {
		t.Errorf("summary does not separate label from its value: %q", framed)
	}
	for _, row := range strings.Split(strings.TrimSpace(framed), "\n") {
		if !strings.HasPrefix(row, "│ ") || !strings.HasSuffix(row, " │") {
			t.Errorf("summary row is outside the frame: %q", row)
		}
	}
}
