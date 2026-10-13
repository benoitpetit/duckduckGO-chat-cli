package persistence

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// CoordinateReportTextLimit matches the coordinator's existing per-stage
// response bound and keeps each saved prompt, response, and error bounded.
const CoordinateReportTextLimit = 12000

var (
	coordinateReportIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)
	coordinateStageIDPattern  = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)
)

// CoordinateReport is an optional, local record of one explicitly saved
// multi-model run. It is stored with its parent conversation archive.
type CoordinateReport struct {
	ID                   string                  `json:"id"`
	Strategy             string                  `json:"strategy"`
	Question             string                  `json:"question"`
	QuestionTruncated    bool                    `json:"question_truncated,omitempty"`
	StartedAt            time.Time               `json:"started_at"`
	FinishedAt           time.Time               `json:"finished_at"`
	Complete             bool                    `json:"complete"`
	Partial              bool                    `json:"partial"`
	FinalAnswer          string                  `json:"final_answer"`
	FinalAnswerTruncated bool                    `json:"final_answer_truncated,omitempty"`
	Stages               []CoordinateReportStage `json:"stages"`
}

// CoordinateReportStage preserves the ordered identity and outcome of one
// model request inside a coordinate run.
type CoordinateReportStage struct {
	ID                string    `json:"id"`
	Label             string    `json:"label"`
	Model             string    `json:"model"`
	Role              string    `json:"role"`
	Status            string    `json:"status"`
	Attempt           int       `json:"attempt"`
	StartedAt         time.Time `json:"started_at"`
	FinishedAt        time.Time `json:"finished_at"`
	Prompt            string    `json:"prompt"`
	PromptTruncated   bool      `json:"prompt_truncated,omitempty"`
	Response          string    `json:"response"`
	ResponseTruncated bool      `json:"response_truncated,omitempty"`
	Error             string    `json:"error,omitempty"`
}

func validateCoordinateReports(reports []CoordinateReport) error {
	seenReports := make(map[string]struct{}, len(reports))
	for reportIndex, report := range reports {
		if !coordinateReportIDPattern.MatchString(report.ID) {
			return fmt.Errorf("coordinate report %d has an invalid ID", reportIndex+1)
		}
		if _, exists := seenReports[report.ID]; exists {
			return fmt.Errorf("duplicate coordinate report ID %q", report.ID)
		}
		seenReports[report.ID] = struct{}{}
		if report.Strategy == "" || len(report.Strategy) > 32 {
			return fmt.Errorf("coordinate report %q has an invalid strategy", report.ID)
		}
		if report.Complete == report.Partial {
			return fmt.Errorf("coordinate report %q must be complete or partial", report.ID)
		}
		if strings.TrimSpace(report.Question) == "" || strings.TrimSpace(report.FinalAnswer) == "" || len(report.Stages) == 0 {
			return fmt.Errorf("coordinate report %q is missing question, answer, or stages", report.ID)
		}
		if err := validateCoordinateReportText(report.ID, "question", report.Question); err != nil {
			return err
		}
		if err := validateCoordinateReportText(report.ID, "final answer", report.FinalAnswer); err != nil {
			return err
		}
		seenStages := make(map[string]struct{}, len(report.Stages))
		for stageIndex, stage := range report.Stages {
			if !coordinateStageIDPattern.MatchString(stage.ID) {
				return fmt.Errorf("coordinate report %q stage %d has an invalid ID", report.ID, stageIndex+1)
			}
			if _, exists := seenStages[stage.ID]; exists {
				return fmt.Errorf("coordinate report %q has duplicate stage ID %q", report.ID, stage.ID)
			}
			seenStages[stage.ID] = struct{}{}
			if stage.Status == "" || len(stage.Status) > 32 {
				return fmt.Errorf("coordinate report %q stage %q has an invalid status", report.ID, stage.ID)
			}
			for _, field := range []struct{ name, value string }{
				{"prompt", stage.Prompt}, {"response", stage.Response}, {"error", stage.Error},
			} {
				if err := validateCoordinateReportText(report.ID, "stage "+stage.ID+" "+field.name, field.value); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func validateCoordinateReportText(reportID, field, value string) error {
	if len(value) > CoordinateReportTextLimit {
		return fmt.Errorf("coordinate report %q %s exceeds the %d-byte limit", reportID, field, CoordinateReportTextLimit)
	}
	return nil
}
