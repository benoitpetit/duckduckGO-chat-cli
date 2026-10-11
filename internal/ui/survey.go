package ui

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
)

// SurveyAccentFormat converts the active theme accent into a survey-compatible
// 256-color foreground format so prompts follow the CLI theme.
func SurveyAccentFormat() string {
	r, g, b := parseHexColor(CurrentTheme().Colors.Accent)
	bestCode, bestDistance := 0, int(^uint(0)>>1)
	for code := 0; code < 256; code++ {
		cr, cg, cb := xtermColor(code)
		dr, dg, db := r-cr, g-cg, b-cb
		distance := dr*dr + dg*dg + db*db
		if distance < bestDistance {
			bestCode, bestDistance = code, distance
		}
	}
	return fmt.Sprintf("%d+b", bestCode)
}

// ThemedSurveyIcons applies the current CLI accent color to survey prompts.
func ThemedSurveyIcons() survey.AskOpt {
	accent := SurveyAccentFormat()
	return survey.WithIcons(func(icons *survey.IconSet) {
		icons.Question.Format = accent
		icons.SelectFocus.Format = accent
		icons.MarkedOption.Format = accent
	})
}

func xtermColor(code int) (int, int, int) {
	if code < 16 {
		basic := [16][3]int{{0, 0, 0}, {128, 0, 0}, {0, 128, 0}, {128, 128, 0}, {0, 0, 128}, {128, 0, 128}, {0, 128, 128}, {192, 192, 192}, {128, 128, 128}, {255, 0, 0}, {0, 255, 0}, {255, 255, 0}, {0, 0, 255}, {255, 0, 255}, {0, 255, 255}, {255, 255, 255}}
		return basic[code][0], basic[code][1], basic[code][2]
	}
	if code < 232 {
		levels := [...]int{0, 95, 135, 175, 215, 255}
		index := code - 16
		return levels[index/36], levels[(index/6)%6], levels[index%6]
	}
	gray := 8 + (code-232)*10
	return gray, gray, gray
}
