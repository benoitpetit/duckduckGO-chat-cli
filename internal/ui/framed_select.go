package ui

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/charmbracelet/x/ansi"
	"golang.org/x/term"
)

// AskFramedSelect renders a survey selector and its summary inside the same
// theme-aware border used by the configuration menu.
func AskFramedSelect(prompt *survey.Select, choice *string, title string, summary []PanelLine, options ...survey.AskOpt) error {
	width, cursorWidth := framedSelectWidths()
	labelWidth := max(10, width-6)
	if ansi.StringWidth(title) > width-6 {
		title = ansi.Truncate(title, max(1, width-6), "…")
	}

	muted := MutedColor.Sprint("╭─ ")
	top := muted + AccentColor.Sprint(title) + MutedColor.Sprint(" "+strings.Repeat("─", max(0, width-utf8.RuneCountInString("─ "+title+" ")-2))+"╮")
	bottom := MutedColor.Sprint("╰" + strings.Repeat("─", width-2) + "╯")
	vertical := MutedColor.Sprint("│")
	focus := AccentColor.Sprint("›")

	priorSelect, priorMulti := survey.SelectQuestionTemplate, survey.MultiSelectQuestionTemplate
	priorHelp := prompt.Help
	prompt.Help = framedSelectSummary(summary, width)
	survey.SelectQuestionTemplate = `
{{- define "option"}}
{{- printf "%s " "` + vertical + `"}}
{{- if eq .SelectedIndex .CurrentIndex}}{{printf "%s " "` + focus + `"}}{{else}}  {{end}}
{{- printf "%-*s" ` + strconv.Itoa(labelWidth) + ` .CurrentOpt.Value}}
{{- printf " %s\n" "` + vertical + `"}}
{{- end}}
{{- printf "%s\n" "` + top + `"}}
{{- printf "%s" .Help}}
{{- if .ShowAnswer}}
{{- printf "%s %-*s %s\n" "` + vertical + `" ` + strconv.Itoa(width-4) + ` (printf "%s: %s" .Message .Answer) "` + vertical + `"}}
{{- printf "%s\n" "` + bottom + `"}}
{{- else}}
{{- printf "%s %-*s %s\n" "` + vertical + `" ` + strconv.Itoa(width-4) + ` (printf "%s%s" .Message .FilterMessage) "` + vertical + `"}}
{{- printf "%s %-*s %s\n" "` + vertical + `" ` + strconv.Itoa(width-4) + ` "↑/↓ move · Enter select · type to filter · ? for help" "` + vertical + `"}}
{{- if .ShowHelp}}{{- printf "%s %-*s %s\n" "` + vertical + `" ` + strconv.Itoa(width-4) + ` "Use ↑/↓ to move, Enter to select, or type to filter." "` + vertical + `"}}{{end}}
{{- range $ix, $option := .PageEntries}}{{template "option" $.IterateOption $ix $option}}{{end}}
{{- printf "%s\n" "` + bottom + `"}}
{{- end}}`
	// survey uses the multi-select template to clear the previous highlighted
	// row during navigation, even for a single-select prompt.
	survey.MultiSelectQuestionTemplate = `{{- define "option"}}{{if eq .SelectedIndex .CurrentIndex}}{{printf "%s" "` + strings.Repeat(" ", cursorWidth+1) + `"}}{{else}}│  {{printf "%-*s" ` + strconv.Itoa(labelWidth) + ` .CurrentOpt.Value}} │{{end}}{{end}}`
	defer func() {
		survey.SelectQuestionTemplate, survey.MultiSelectQuestionTemplate = priorSelect, priorMulti
		prompt.Help = priorHelp
	}()
	if err := survey.AskOne(prompt, choice, options...); err != nil {
		return err
	}
	clearFramedSelectAnswer(prompt, choice, width)
	return nil
}

func framedSelectWidths() (width, cursorWidth int) {
	width, cursorWidth = 80, 10000
	if term.IsTerminal(int(os.Stdout.Fd())) {
		if columns, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && columns > 0 {
			width, cursorWidth = columns, columns
		}
	}
	width = min(112, max(24, width))
	return width, cursorWidth
}

func framedSelectSummary(lines []PanelLine, width int) string {
	var framed strings.Builder
	contentWidth := max(1, width-4)
	for _, line := range lines {
		var styled strings.Builder
		for _, span := range line {
			value := sanitizePanelLine(span.Text)
			if value != "" {
				if styled.Len() > 0 {
					styled.WriteByte(' ')
				}
				styled.WriteString(panelColor(span.Style).Sprint(value))
			}
		}
		parts := []string{styled.String()}
		if ansi.StringWidth(parts[0]) > contentWidth {
			parts = strings.Split(ansi.Hardwrap(parts[0], contentWidth, true), "\n")
		}
		for _, part := range parts {
			padding := max(0, contentWidth-ansi.StringWidth(part))
			fmt.Fprintf(&framed, "%s %s%s %s\n", MutedColor.Sprint("│"), part, strings.Repeat(" ", padding), MutedColor.Sprint("│"))
		}
	}
	return framed.String()
}

func clearFramedSelectAnswer(prompt *survey.Select, choice *string, width int) {
	if !term.IsTerminal(int(os.Stdout.Fd())) || choice == nil {
		return
	}
	answer := fmt.Sprintf("%s: %s", prompt.Message, *choice)
	answerRows := max(1, (ansi.StringWidth(answer)+width-5)/(width-4))
	helpRows := 0
	if prompt.Help != "" {
		helpRows = len(strings.Split(strings.TrimSuffix(prompt.Help, "\n"), "\n"))
	}
	rows := helpRows + answerRows + 2
	cursor := &terminal.Cursor{Out: os.Stdout}
	if err := cursor.Up(1); err != nil {
		return
	}
	for row := 0; row < rows; row++ {
		if err := cursor.HorizontalAbsolute(0); err != nil {
			return
		}
		if err := terminal.EraseLine(os.Stdout, terminal.ERASE_LINE_ALL); err != nil {
			return
		}
		if row+1 < rows {
			if err := cursor.Up(1); err != nil {
				return
			}
		}
	}
}
