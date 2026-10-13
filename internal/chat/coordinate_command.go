package chat

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"duckduckgo-chat-cli/internal/config"
	"duckduckgo-chat-cli/internal/models"
	"duckduckgo-chat-cli/internal/ui"

	surveyterminal "github.com/AlecAivazis/survey/v2/terminal"
	"golang.org/x/term"
)

type coordinateSurvey interface {
	Panel(string, []ui.PanelLine)
	Select(string, []string, string) (string, error)
	Input(string) (string, error)
}

func defaultCoordinateRole(strategy CoordinateStrategy, index int) string {
	roles := []string{"Analyst", "Verifier", "Synthesizer"}
	switch strategy {
	case CoordinateChain:
		roles = []string{"Solver", "Critic", "Synthesizer"}
	case CoordinateDebate:
		roles = []string{"Debater", "Skeptic", "Synthesizer"}
	}
	return roles[index]
}

func initialCoordinatePlan(c *Chat, cfg *config.Config) CoordinatePlan {
	if plan, ok := c.nextCoordinate(); ok {
		return plan
	}
	strategy := CoordinateIndependent
	if cfg != nil && isCoordinateStrategy(CoordinateStrategy(cfg.Coordinate.Strategy)) {
		strategy = CoordinateStrategy(cfg.Coordinate.Strategy)
	}
	if plan, ok := savedCoordinatePlanForStrategy(cfg, strategy); ok {
		return plan
	}
	return defaultCoordinatePlan(c, strategy)
}

func defaultCoordinatePlan(c *Chat, strategy CoordinateStrategy) CoordinatePlan {
	first := models.Default()
	if c != nil {
		first = c.Model
	}
	if resolved, ok := models.ResolveModel(string(first)); !ok || resolved != first {
		first = models.Default()
	}
	second := first
	for _, candidate := range models.Available() {
		if candidate.ID != first {
			second = candidate.ID
			break
		}
	}
	return CoordinatePlan{
		Strategy: strategy, DebateRounds: 1,
		Agents: []CoordinateAgent{
			{Model: first, Role: defaultCoordinateRole(strategy, 0)},
			{Model: second, Role: defaultCoordinateRole(strategy, 1)},
		},
	}
}

func savedCoordinatePlan(cfg *config.Config) (CoordinatePlan, bool) {
	if cfg == nil {
		return CoordinatePlan{}, false
	}
	strategy := CoordinateStrategy(cfg.Coordinate.Strategy)
	if strategy == "" {
		strategy = CoordinateIndependent
	}
	return savedCoordinatePlanForStrategy(cfg, strategy)
}

func savedCoordinatePlanForStrategy(cfg *config.Config, strategy CoordinateStrategy) (CoordinatePlan, bool) {
	if cfg == nil || !isCoordinateStrategy(strategy) {
		return CoordinatePlan{}, false
	}
	if preset, ok := cfg.Coordinate.Profiles[string(strategy)]; ok {
		return coordinatePlanFromConfig(strategy, preset)
	}
	legacyStrategy := CoordinateStrategy(cfg.Coordinate.Strategy)
	if legacyStrategy == "" {
		legacyStrategy = CoordinateIndependent
	}
	if legacyStrategy != strategy || len(cfg.Coordinate.Agents) == 0 {
		return CoordinatePlan{}, false
	}
	return coordinatePlanFromConfig(strategy, config.CoordinatePresetConfig{
		Agents:       cfg.Coordinate.Agents,
		DebateRounds: cfg.Coordinate.DebateRounds,
		Parallel:     cfg.Coordinate.Parallel,
	})
}

func coordinatePlanFromConfig(strategy CoordinateStrategy, saved config.CoordinatePresetConfig) (CoordinatePlan, bool) {
	plan := CoordinatePlan{Strategy: strategy, DebateRounds: saved.DebateRounds, Parallel: saved.Parallel}
	for _, agent := range saved.Agents {
		plan.Agents = append(plan.Agents, CoordinateAgent{Model: models.Model(agent.Model), Role: agent.Role})
	}
	if err := validateCoordinatePresetStructure(plan); err != nil {
		return CoordinatePlan{}, false
	}
	if plan.Parallel && !plan.SupportsParallel() {
		plan.Parallel = false
	}
	return plan, true
}

func isCoordinateStrategy(strategy CoordinateStrategy) bool {
	switch strategy {
	case CoordinateIndependent, CoordinateChain, CoordinateDebate:
		return true
	default:
		return false
	}
}

func validateCoordinatePresetStructure(plan CoordinatePlan) error {
	if len(plan.Agents) < 2 || len(plan.Agents) > 3 {
		return errors.New("select two or three coordinate agents")
	}
	switch plan.Strategy {
	case CoordinateChain, CoordinateIndependent:
	case CoordinateDebate:
		if plan.DebateRounds < 1 || plan.DebateRounds > 2 {
			return errors.New("debate must have one or two rounds")
		}
	default:
		return fmt.Errorf("unknown coordinate strategy %q", plan.Strategy)
	}
	for index, agent := range plan.Agents {
		role := strings.TrimSpace(agent.Role)
		if role == "" || len(role) > 80 || strings.ContainsAny(role, "\r\n") {
			return fmt.Errorf("agent %d needs a one-line role of at most 80 bytes", index+1)
		}
	}
	return nil
}

func coordinateModelAvailability(plan CoordinatePlan) []int {
	unavailable := make([]int, 0)
	for index, agent := range plan.Agents {
		resolved, ok := models.ResolveModel(string(agent.Model))
		if !ok || resolved != agent.Model {
			unavailable = append(unavailable, index)
		}
	}
	return unavailable
}

func coordinateDuplicateAdvisory(plan CoordinatePlan) string {
	for i, agent := range plan.Agents {
		role := strings.ToLower(strings.TrimSpace(agent.Role))
		for _, other := range plan.Agents[i+1:] {
			otherRole := strings.ToLower(strings.TrimSpace(other.Role))
			if agent.Model != other.Model && role == otherRole {
				return "Duplicate roles across different models are allowed; agents with the same role may give similar answers."
			}
		}
	}
	return ""
}

func coordinateUnavailableModelLabel(model models.Model) string {
	if resolved, ok := models.ResolveModel(string(model)); !ok || resolved != model {
		return "Unavailable: " + string(model)
	}
	return models.DisplayName(model)
}

func saveCoordinatePlan(cfg *config.Config, plan CoordinatePlan) error {
	if cfg == nil {
		return errors.New("configuration is unavailable")
	}
	saved := cfg.Coordinate
	saved.Profiles = cloneCoordinateProfiles(cfg.Coordinate.Profiles)
	if saved.Profiles == nil {
		saved.Profiles = make(map[string]config.CoordinatePresetConfig)
	}
	legacyStrategy := CoordinateStrategy(saved.Strategy)
	if legacyStrategy == "" {
		legacyStrategy = CoordinateIndependent
	}
	if len(saved.Agents) > 0 && isCoordinateStrategy(legacyStrategy) {
		if _, exists := saved.Profiles[string(legacyStrategy)]; !exists {
			saved.Profiles[string(legacyStrategy)] = config.CoordinatePresetConfig{
				Agents:       cloneCoordinateAgentConfigs(saved.Agents),
				DebateRounds: saved.DebateRounds,
				Parallel:     saved.Parallel,
			}
		}
	}
	preset := config.CoordinatePresetConfig{
		DebateRounds: plan.DebateRounds,
		Parallel:     plan.Parallel,
		Agents:       make([]config.CoordinateAgentConfig, 0, len(plan.Agents)),
	}
	for _, agent := range plan.Agents {
		preset.Agents = append(preset.Agents, config.CoordinateAgentConfig{Model: string(agent.Model), Role: agent.Role})
	}
	saved.Profiles[string(plan.Strategy)] = preset
	saved.Strategy = string(plan.Strategy)
	saved.DebateRounds = plan.DebateRounds
	saved.Parallel = plan.Parallel
	saved.Agents = cloneCoordinateAgentConfigs(preset.Agents)
	previous := cfg.Coordinate
	cfg.Coordinate = saved
	if err := config.SaveConfig(cfg); err != nil {
		cfg.Coordinate = previous
		return err
	}
	return nil
}

func cloneCoordinateProfiles(profiles map[string]config.CoordinatePresetConfig) map[string]config.CoordinatePresetConfig {
	if len(profiles) == 0 {
		return nil
	}
	cloned := make(map[string]config.CoordinatePresetConfig, len(profiles))
	for strategy, preset := range profiles {
		preset.Agents = cloneCoordinateAgentConfigs(preset.Agents)
		cloned[strategy] = preset
	}
	return cloned
}

func cloneCoordinateAgentConfigs(agents []config.CoordinateAgentConfig) []config.CoordinateAgentConfig {
	if agents == nil {
		return nil
	}
	return append([]config.CoordinateAgentConfig(nil), agents...)
}

func persistCoordinateDraft(cfg *config.Config, plan CoordinatePlan) {
	if validateCoordinatePresetStructure(plan) != nil {
		return
	}
	if err := saveCoordinatePlan(cfg, plan); err != nil {
		ui.Warningln("Could not save coordinate preset: %v", err)
	}
}

func coordinateStrategyName(strategy CoordinateStrategy) string {
	switch strategy {
	case CoordinateChain:
		return "Chain"
	case CoordinateDebate:
		return "Debate"
	default:
		return "Independent"
	}
}

func coordinateStrategyDescription(strategy CoordinateStrategy) string {
	switch strategy {
	case CoordinateChain:
		return "Agent 2 critiques Agent 1's draft before synthesis."
	case CoordinateDebate:
		return "Agents revise against each other for one or two rounds."
	default:
		return "Two separate answers, then a synthesis."
	}
}

func setCoordinateStrategy(plan *CoordinatePlan, strategy CoordinateStrategy) {
	previous := plan.Strategy
	for i := range plan.Agents {
		if plan.Agents[i].Role == defaultCoordinateRole(previous, i) {
			plan.Agents[i].Role = defaultCoordinateRole(strategy, i)
		}
	}
	plan.Strategy = strategy
	if strategy == CoordinateDebate && (plan.DebateRounds < 1 || plan.DebateRounds > 2) {
		plan.DebateRounds = 1
	}
	if !plan.SupportsParallel() {
		plan.Parallel = false
	}
}

func coordinatePlanForStrategy(c *Chat, cfg *config.Config, current CoordinatePlan, strategy CoordinateStrategy) CoordinatePlan {
	if strategy == current.Strategy {
		return current
	}
	if plan, ok := savedCoordinatePlanForStrategy(cfg, strategy); ok {
		return plan
	}
	return defaultCoordinatePlan(c, strategy)
}

func runCoordinateMenu(c *Chat, cfg *config.Config, prompts coordinateSurvey) error {
	plan := initialCoordinatePlan(c, cfg)
	for {
		lines := []ui.PanelLine{
			coordinatePanelField("Algorithm", coordinateStrategyName(plan.Strategy), ui.PanelForeground),
			coordinatePanelField("Method", coordinateStrategyDescription(plan.Strategy), ui.PanelForeground),
			coordinatePanelField("Agents", fmt.Sprintf("%d selected", len(plan.Agents)), ui.PanelForeground),
		}
		for i, agent := range plan.Agents {
			style := ui.PanelForeground
			if len(coordinateModelAvailability(CoordinatePlan{Agents: []CoordinateAgent{agent}})) > 0 {
				style = ui.PanelWarning
			}
			lines = append(lines, coordinatePanelField(fmt.Sprintf("Agent %d", i+1), coordinateUnavailableModelLabel(agent.Model)+" · "+agent.Role, style))
		}
		lines = append(lines,
			coordinatePanelField("Calls", fmt.Sprintf("%d planned Duck.ai calls (retries excluded)", plan.RequestCount()), ui.PanelForeground),
		)
		concurrency := "1 request at a time"
		if plan.Parallel && plan.SupportsParallel() {
			concurrency = fmt.Sprintf("Up to %d simultaneous requests", plan.MaxParallelism())
		}
		lines = append(lines, coordinatePanelField("Concurrency", concurrency, ui.PanelForeground))
		if advisory := coordinateDuplicateAdvisory(plan); advisory != "" {
			lines = append(lines, coordinatePanelField("Advisory", advisory, ui.PanelWarning))
		}
		if plan.Strategy == CoordinateDebate {
			lines = append(lines, coordinatePanelField("Rounds", fmt.Sprintf("%d", plan.DebateRounds), ui.PanelForeground))
		}
		prompts.Panel("Coordinate", lines)
		choices := []string{"Algorithm", "Agents"}
		if plan.SupportsParallel() {
			choices = append(choices, "Execution")
		}
		if plan.Strategy == CoordinateDebate {
			choices = append(choices, "Debate rounds")
		}
		choices = append(choices, "Use for next question", "Cancel")
		action, err := prompts.Select("Coordinate options", choices, "Use for next question")
		if err != nil {
			return err
		}
		switch action {
		case "Algorithm":
			prompts.Panel("Coordinate · algorithm", []ui.PanelLine{
				coordinatePanelField("Independent", coordinateStrategyDescription(CoordinateIndependent), ui.PanelForeground),
				coordinatePanelField("Chain", coordinateStrategyDescription(CoordinateChain), ui.PanelForeground),
				coordinatePanelField("Debate", coordinateStrategyDescription(CoordinateDebate), ui.PanelForeground),
			})
			selected, err := prompts.Select("Algorithm", []string{"Independent", "Chain", "Debate", "Back"}, coordinateStrategyName(plan.Strategy))
			if err != nil {
				return err
			}
			switch selected {
			case "Independent":
				plan = coordinatePlanForStrategy(c, cfg, plan, CoordinateIndependent)
				persistCoordinateDraft(cfg, plan)
			case "Chain":
				plan = coordinatePlanForStrategy(c, cfg, plan, CoordinateChain)
				persistCoordinateDraft(cfg, plan)
			case "Debate":
				plan = coordinatePlanForStrategy(c, cfg, plan, CoordinateDebate)
				persistCoordinateDraft(cfg, plan)
			}
		case "Agents":
			if err := runCoordinateAgentsMenu(&plan, cfg, prompts); err != nil {
				return err
			}
		case "Debate rounds":
			prompts.Panel("Coordinate · debate rounds", []ui.PanelLine{{{Text: "Each extra round adds two Duck.ai calls.", Style: ui.PanelForeground}}})
			selected, err := prompts.Select("Debate rounds", []string{"1 round", "2 rounds", "Back"}, fmt.Sprintf("%d round", plan.DebateRounds))
			if err != nil {
				return err
			}
			if selected == "1 round" {
				plan.DebateRounds = 1
				persistCoordinateDraft(cfg, plan)
			} else if selected == "2 rounds" {
				plan.DebateRounds = 2
				persistCoordinateDraft(cfg, plan)
			}
		case "Execution":
			prompts.Panel("Coordinate · execution", []ui.PanelLine{{{Text: "Parallel runs overlap independent stages. Chain dependencies stay sequential.", Style: ui.PanelForeground}}})
			currentExecution := "Sequential"
			if plan.Parallel {
				currentExecution = "Parallel independent steps"
			}
			selected, err := prompts.Select("Execution", []string{"Sequential", "Parallel independent steps", "Back"}, currentExecution)
			if err != nil {
				return err
			}
			if selected == "Sequential" {
				plan.Parallel = false
				persistCoordinateDraft(cfg, plan)
			} else if selected == "Parallel independent steps" {
				plan.Parallel = true
				persistCoordinateDraft(cfg, plan)
			}
		case "Use for next question":
			if unavailable := coordinateModelAvailability(plan); len(unavailable) > 0 {
				for _, index := range unavailable {
					ui.Warningln("Agent %d uses unavailable model %q. Replace it before using /coordinate.", index+1, plan.Agents[index].Model)
				}
				continue
			}
			if err := c.SetNextCoordinate(plan); err != nil {
				ui.Errorln("Could not activate /coordinate: %v", err)
				continue
			}
			if err := saveCoordinatePlan(cfg, plan); err != nil {
				ui.Warningln("Coordinate is armed for this question, but its settings could not be saved: %v", err)
			}
			if advisory := coordinateDuplicateAdvisory(plan); advisory != "" {
				ui.Warningln("%s", advisory)
			}
			ui.AIln("Coordinate ready for the next question: %s with %d agents.", coordinateStrategyName(plan.Strategy), len(plan.Agents))
			return nil
		case "Cancel":
			return nil
		}
	}
}

func runCoordinateAgentsMenu(plan *CoordinatePlan, cfg *config.Config, prompts coordinateSurvey) error {
	for {
		lines := []ui.PanelLine{{{Text: "Choose an agent to edit its model or role.", Style: ui.PanelForeground}}}
		for i, agent := range plan.Agents {
			lines = append(lines, coordinatePanelField(fmt.Sprintf("Agent %d", i+1), coordinateUnavailableModelLabel(agent.Model)+" · "+agent.Role, ui.PanelForeground))
		}
		prompts.Panel("Coordinate · agents", lines)
		choices := make([]string, 0, 5)
		for i := range plan.Agents {
			choices = append(choices, fmt.Sprintf("Agent %d", i+1))
		}
		if len(plan.Agents) < 3 {
			choices = append(choices, "Add agent")
		}
		choices = append(choices, "Back")
		action, err := prompts.Select("Agents", choices, "Back")
		if err != nil {
			return err
		}
		if action == "Back" {
			return nil
		}
		if action == "Add agent" {
			preferred := models.Default()
			for _, candidate := range models.Available() {
				used := false
				for _, agent := range plan.Agents {
					if agent.Model == candidate.ID {
						used = true
						break
					}
				}
				if !used {
					preferred = candidate.ID
					break
				}
			}
			model, err := chooseCoordinateModel(prompts, preferred)
			if err != nil {
				return err
			}
			if model != "" {
				plan.Agents = append(plan.Agents, CoordinateAgent{Model: model, Role: defaultCoordinateRole(plan.Strategy, len(plan.Agents))})
				persistCoordinateDraft(cfg, *plan)
			}
			continue
		}
		var index int
		if _, err := fmt.Sscanf(action, "Agent %d", &index); err != nil || index < 1 || index > len(plan.Agents) {
			return fmt.Errorf("unknown coordinate agent selection %q", action)
		}
		if err := runCoordinateAgentMenu(plan, cfg, index-1, prompts); err != nil {
			return err
		}
	}
}

func runCoordinateAgentMenu(plan *CoordinatePlan, cfg *config.Config, index int, prompts coordinateSurvey) error {
	for {
		agent := plan.Agents[index]
		prompts.Panel(fmt.Sprintf("Coordinate · agent %d", index+1), []ui.PanelLine{
			coordinatePanelField("Model", coordinateUnavailableModelLabel(agent.Model), ui.PanelForeground),
			coordinatePanelField("Role", agent.Role, ui.PanelForeground),
		})
		choices := []string{"Change model", "Edit role"}
		if len(plan.Agents) > 2 {
			choices = append(choices, "Remove agent")
		}
		choices = append(choices, "Back")
		action, err := prompts.Select("Edit agent", choices, "Back")
		if err != nil {
			return err
		}
		switch action {
		case "Change model":
			model, err := chooseCoordinateModel(prompts, agent.Model)
			if err != nil {
				return err
			}
			if model != "" {
				plan.Agents[index].Model = model
				persistCoordinateDraft(cfg, *plan)
			}
		case "Edit role":
			role, err := prompts.Input("Role for this agent")
			if err != nil {
				return err
			}
			role = strings.TrimSpace(role)
			if role == "" || len(role) > 80 || strings.ContainsAny(role, "\r\n") {
				ui.Errorln("The role must be one line of at most 80 bytes.")
				continue
			}
			plan.Agents[index].Role = role
			persistCoordinateDraft(cfg, *plan)
		case "Remove agent":
			plan.Agents = append(plan.Agents[:index], plan.Agents[index+1:]...)
			persistCoordinateDraft(cfg, *plan)
			return nil
		case "Back":
			return nil
		}
	}
}

func chooseCoordinateModel(prompts coordinateSurvey, current models.Model) (models.Model, error) {
	definitions := models.Available()
	choices := make([]string, 0, len(definitions)+1)
	for _, definition := range definitions {
		choices = append(choices, definition.Name)
	}
	choices = append(choices, "Back")
	prompts.Panel("Coordinate · model", []ui.PanelLine{{{Text: "Choose a Duck.ai model for this agent.", Style: ui.PanelForeground}}})
	defaultModel := models.DisplayName(current)
	if resolved, ok := models.ResolveModel(string(current)); !ok || resolved != current {
		defaultModel = models.DisplayName(models.Available()[0].ID)
	}
	choice, err := prompts.Select("Model", choices, defaultModel)
	if err != nil || choice == "Back" {
		return "", err
	}
	for _, definition := range definitions {
		if choice == definition.Name {
			return definition.ID, nil
		}
	}
	return "", fmt.Errorf("unknown coordinate model %q", choice)
}

type terminalCoordinateRecoveryPrompt struct {
	prompts coordinateSurvey
}

func (p terminalCoordinateRecoveryPrompt) ChooseRecovery(run *CoordinateRun) (CoordinateRecoveryAction, error) {
	if p.prompts == nil {
		return CoordinateRecoveryStop, errors.New("coordinate recovery menu is unavailable")
	}
	completed, incomplete := 0, 0
	for _, stage := range run.Stages {
		if stage.ID == "synthesis" {
			continue
		}
		if stage.Status == CoordinateStageCompleted {
			completed++
		} else {
			incomplete++
		}
	}
	lines := []ui.PanelLine{
		coordinatePanelField("Completed outputs", fmt.Sprintf("%d", completed), ui.PanelForeground),
		coordinatePanelField("Incomplete stages", fmt.Sprintf("%d", incomplete), ui.PanelForeground),
	}
	if err := firstCoordinateStageError(run); err != nil {
		lines = append(lines, coordinatePanelField("Last error", err.Error(), ui.PanelError))
	}
	p.prompts.Panel("Coordinate recovery", lines)
	choices := []string{"Resume incomplete stages"}
	partialAvailable := coordinateCanSynthesizePartial(run)
	if partialAvailable {
		choices = append(choices, "Synthesize partial result")
	}
	choices = append(choices, "Stop")
	selected, err := p.prompts.Select("Coordinate recovery", choices, choices[0])
	if err != nil {
		return CoordinateRecoveryStop, err
	}
	switch selected {
	case "Resume incomplete stages":
		return CoordinateRecoveryResume, nil
	case "Synthesize partial result":
		if partialAvailable {
			return CoordinateRecoverySynthesizePartial, nil
		}
	case "Stop":
		return CoordinateRecoveryStop, nil
	}
	return CoordinateRecoveryStop, fmt.Errorf("unknown coordinate recovery action %q", selected)
}

func firstCoordinateStageError(run *CoordinateRun) error {
	if run == nil {
		return nil
	}
	for _, stage := range run.Stages {
		if stage.Status == CoordinateStageFailed && stage.Err != "" {
			return errors.New(stage.Err)
		}
	}
	for _, stage := range run.Stages {
		if stage.Err != "" {
			return errors.New(stage.Err)
		}
	}
	return nil
}

func HandleCoordinateCommand(c *Chat, cfg *config.Config, args string) {
	if strings.TrimSpace(args) == "cancel" {
		c.CancelNextCoordinate()
		ui.AIln("Coordinate selection cleared.")
		return
	}
	if strings.TrimSpace(args) != "" {
		ui.Errorln("Usage: /coordinate [cancel]")
		return
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		ui.Errorln("The /coordinate menu requires an interactive terminal.")
		return
	}
	if err := runCoordinateMenu(c, cfg, &terminalImagePromptSurvey{}); err != nil && err != surveyterminal.InterruptErr {
		ui.Errorln("Could not configure /coordinate: %v", err)
	}
}
