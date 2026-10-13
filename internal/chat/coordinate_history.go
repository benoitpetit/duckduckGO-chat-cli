package chat

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"duckduckgo-chat-cli/internal/persistence"
	"duckduckgo-chat-cli/internal/ui"
)

func coordinateHistoryCommand(c *Chat, args []string) (string, []ui.PanelLine, error) {
	if c == nil || len(args) == 0 {
		return "", nil, errors.New("usage: /coordinate runs | /coordinate show <run_id>")
	}
	switch args[0] {
	case "runs":
		if len(args) != 1 {
			return "", nil, errors.New("usage: /coordinate runs")
		}
		return "Saved coordinate runs", coordinateReportListLines(c.CoordinateReports), nil
	case "show":
		if len(args) != 2 {
			return "", nil, errors.New("usage: /coordinate show <run_id>")
		}
		for _, report := range c.CoordinateReports {
			if report.ID == args[1] {
				return "Coordinate report · " + sanitizeCoordinateReportDisplay(report.ID), coordinateReportDetailLines(report), nil
			}
		}
		return "", nil, fmt.Errorf("coordinate report %q was not found in the active conversation", args[1])
	default:
		return "", nil, errors.New("usage: /coordinate [cancel|runs|show <run_id>]")
	}
}

func coordinateReportListLines(reports []persistence.CoordinateReport) []ui.PanelLine {
	if len(reports) == 0 {
		return []ui.PanelLine{coordinatePanelField("Reports", "No saved coordinate reports in this conversation.", ui.PanelMuted)}
	}
	indexes := make([]int, len(reports))
	for index := range indexes {
		indexes[index] = index
	}
	sort.SliceStable(indexes, func(i, j int) bool {
		return reports[indexes[i]].FinishedAt.After(reports[indexes[j]].FinishedAt)
	})
	lines := []ui.PanelLine{coordinatePanelField("Saved reports", "Newest first · use /coordinate show <run_id> to inspect a report.", ui.PanelMuted)}
	for _, index := range indexes {
		report := reports[index]
		finished := "date unavailable"
		if !report.FinishedAt.IsZero() {
			finished = report.FinishedAt.Local().Format("2006-01-02 15:04")
		}
		lines = append(lines, coordinatePanelField(sanitizeCoordinateReportDisplay(report.ID), fmt.Sprintf("%s · %s · %d stages · %s", sanitizeCoordinateReportDisplay(report.Strategy), coordinateReportStatus(report), len(report.Stages), finished), ui.PanelForeground))
	}
	return lines
}

func coordinateReportDetailLines(report persistence.CoordinateReport) []ui.PanelLine {
	lines := []ui.PanelLine{
		coordinatePanelField("Status", coordinateReportStatus(report), ui.PanelInfo),
		coordinatePanelField("Strategy", sanitizeCoordinateReportDisplay(report.Strategy), ui.PanelForeground),
	}
	appendCoordinateReportText(&lines, "Question", report.Question, ui.PanelUser)
	if report.QuestionTruncated {
		lines = append(lines, coordinatePanelField("Question", "Stored question was truncated at the coordinator limit.", ui.PanelWarning))
	}
	if !report.StartedAt.IsZero() {
		lines = append(lines, coordinatePanelField("Started", report.StartedAt.Local().Format(time.RFC3339), ui.PanelMuted))
	}
	if !report.FinishedAt.IsZero() {
		lines = append(lines, coordinatePanelField("Finished", report.FinishedAt.Local().Format(time.RFC3339), ui.PanelMuted))
	}
	for index, stage := range report.Stages {
		label := stage.Label
		if label == "" {
			label = stage.ID
		}
		header := fmt.Sprintf("%d. %s · %s · %s · attempt %d", index+1, label, stage.Role, stage.Model, stage.Attempt)
		lines = append(lines, coordinatePanelField("Stage", sanitizeCoordinateReportDisplay(header), ui.PanelAccent))
		lines = append(lines, coordinatePanelField("Outcome", sanitizeCoordinateReportDisplay(stage.Status), ui.PanelInfo))
		appendCoordinateReportText(&lines, "Prompt", stage.Prompt, ui.PanelMuted)
		if stage.PromptTruncated {
			lines = append(lines, coordinatePanelField("Prompt", "Stored prompt was truncated at the coordinator limit.", ui.PanelWarning))
		}
		appendCoordinateReportText(&lines, "Output", stage.Response, ui.PanelForeground)
		if stage.ResponseTruncated {
			lines = append(lines, coordinatePanelField("Output", "Stored output was truncated at the coordinator limit.", ui.PanelWarning))
		}
		if stage.Error != "" {
			appendCoordinateReportText(&lines, "Error", stage.Error, ui.PanelError)
		}
	}
	appendCoordinateReportText(&lines, "Final answer", report.FinalAnswer, ui.PanelAssistant)
	if report.FinalAnswerTruncated {
		lines = append(lines, coordinatePanelField("Final answer", "Stored answer was truncated at the coordinator limit.", ui.PanelWarning))
	}
	return lines
}

func appendCoordinateReportText(lines *[]ui.PanelLine, label, value string, style ui.PanelStyle) {
	*lines = append(*lines, ui.PanelLine{{Text: label + ":", Style: ui.PanelAccent}})
	value = sanitizeCoordinateReportDisplay(value)
	if value == "" {
		*lines = append(*lines, ui.PanelLine{{Text: "  (empty)", Style: ui.PanelMuted}})
		return
	}
	for _, line := range strings.Split(value, "\n") {
		*lines = append(*lines, ui.PanelLine{{Text: "  " + line, Style: style}})
	}
}

func sanitizeCoordinateReportDisplay(value string) string {
	value = strings.ToValidUTF8(value, "�")
	var safe strings.Builder
	safe.Grow(len(value))
	for _, char := range value {
		switch {
		case char == '\n':
			safe.WriteRune(char)
		case char == '\t':
			safe.WriteString("    ")
		case unicode.IsControl(char):
			// Remove terminal escapes and other control bytes from model output.
		default:
			safe.WriteRune(char)
		}
	}
	return safe.String()
}

func coordinateReportStatus(report persistence.CoordinateReport) string {
	if report.Partial {
		return "partial"
	}
	return "complete"
}
