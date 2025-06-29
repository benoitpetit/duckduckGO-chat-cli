package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"duckduckgo-chat-cli/internal/interfaces"
	"duckduckgo-chat-cli/internal/models"
	"duckduckgo-chat-cli/internal/ui"

	"github.com/AlecAivazis/survey/v2"
)

type SearchConfig struct {
	MaxResults     int  `json:"max_results"`
	IncludeSnippet bool `json:"include_snippet"`
	MaxRetries     int  `json:"max_retries"`
	RetryDelay     int  `json:"retry_delay"`
}

type LibraryConfig struct {
	Directories []string `json:"directories"`
	Enabled     bool     `json:"enabled"`
}

type APIConfig struct {
	Enabled     bool `json:"enabled"`
	Port        int  `json:"port"`
	Autostart   bool `json:"autostart"`
	LogRequests bool `json:"log_requests"`
}

type Config struct {
	TOSAccepted      bool          `json:"tos_accepted"`
	DefaultModel     string        `json:"default_model"`
	ExportDir        string        `json:"export_dir"`
	LastUpdateTime   time.Time     `json:"last_update_time"`
	Search           SearchConfig  `json:"search"`
	Library          LibraryConfig `json:"library"`
	API              APIConfig     `json:"api"`
	ShowMenu         bool          `json:"show_menu"`
	GlobalPrompt     string        `json:"global_prompt"`
	ConfirmLongInput bool          `json:"confirm_long_input"`
}

func Initialize() *Config {
	cfg := loadConfig()
	if cfg.DefaultModel == "" {
		cfg.DefaultModel = "gpt-4o-mini"
	}
	if cfg.Search.MaxResults == 0 {
		cfg.Search.MaxResults = 10
	}
	if cfg.Search.MaxRetries == 0 {
		cfg.Search.MaxRetries = 3
	}
	if cfg.Search.RetryDelay == 0 {
		cfg.Search.RetryDelay = 1
	}
	cfg.Search.IncludeSnippet = true // default to true

	// Initialize library config with defaults
	if len(cfg.Library.Directories) == 0 {
		cfg.Library.Directories = []string{}
	}
	if !cfg.Library.Enabled {
		cfg.Library.Enabled = true // default to enabled
	}

	// Initialize API config with defaults
	if cfg.API.Port == 0 {
		cfg.API.Port = 8080 // default port
	}
	cfg.API.LogRequests = true // default to true

	if err := ensureExportDir(cfg); err != nil {
		ui.Warningln("Warning: Failed to create export directory: %v", err)
	}
	return cfg
}

func loadConfig() *Config {
	cfg := &Config{
		TOSAccepted:      false,
		DefaultModel:     "gpt-4o-mini",
		ExportDir:        defaultExportPath(),
		LastUpdateTime:   time.Now(),
		ConfirmLongInput: true, // default to enabled for safety
	}

	if data, err := os.ReadFile(configPath()); err == nil {
		if err := json.Unmarshal(data, cfg); err != nil {
			ui.Warningln("Warning: Failed to parse config file: %v", err)
		}
	}
	return cfg
}

func defaultExportPath() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("USERPROFILE"), "Documents", "duckchat")
	}
	return filepath.Join(os.Getenv("HOME"), "Documents", "duckchat")
}

func configPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		// Fallback to home directory if config dir is not available
		if runtime.GOOS == "windows" {
			configDir = filepath.Join(os.Getenv("USERPROFILE"), ".config")
		} else {
			configDir = filepath.Join(os.Getenv("HOME"), ".config")
		}
	}
	return filepath.Join(configDir, "duckduckgo-chat-cli", "config.json")
}

func ensureExportDir(cfg *Config) error {
	return os.MkdirAll(cfg.ExportDir, 0755)
}

// SaveConfig saves the configuration to file (exported version of saveConfig)
func SaveConfig(cfg *Config) error {
	configDir := filepath.Dir(configPath())
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath(), data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %v", err)
	}

	return nil
}

// Private version for internal use
func saveConfig(cfg *Config) error {
	return SaveConfig(cfg)
}

func AcceptTermsOfService(cfg *Config) bool {
	if cfg.TOSAccepted {
		return true
	}

	var accepted bool
	prompt := &survey.Confirm{
		Message: "Please accept the terms of service to continue. Do you accept?",
		Default: true,
	}
	survey.AskOne(prompt, &accepted)

	if accepted {
		cfg.TOSAccepted = true
		if err := saveConfig(cfg); err != nil {
			ui.Warningln("Warning: Failed to save config: %v", err)
		}
	}
	return accepted
}

func HandleConfiguration(cfg *Config, chatSession interfaces.ChatSession) {
	for {
		choice := ""
		prompt := &survey.Select{
			Message: "DuckDuckGo Chat CLI Configuration",
			Help:    "Current settings are shown as defaults. Choose an option to edit.",
			Options: []string{
				"Default Model",
				"Export Directory",
				"Search Settings",
				"Show Commands Menu",
				"Global Prompt",
				"Long Input Protection",
				"Library Settings",
				"API Settings",
				"Back to chat",
			},
			Default: "Back to chat",
		}
		survey.AskOne(prompt, &choice)

		switch choice {
		case "Default Model":
			handleModelChange(cfg, chatSession)
		case "Export Directory":
			handleExportDirChange(cfg)
		case "Search Settings":
			handleSearchSettings(cfg)
		case "Show Commands Menu":
			handleShowMenuChange(cfg)
		case "Global Prompt":
			handleGlobalPromptChange(cfg)
		case "Long Input Protection":
			handleLongInputProtectionChange(cfg)
		case "Library Settings":
			handleLibrarySettings(cfg)
		case "API Settings":
			handleAPISettings(cfg)
		case "Back to chat", "":
			return
		default:
			ui.Errorln("Invalid choice. Please try again.")
		}
	}
}

func handleModelChange(cfg *Config, chatSession interfaces.ChatSession) {
	model := ""
	prompt := &survey.Select{
		Message: "Choose Default Model:",
		Options: []string{
			"gpt-4o-mini",
			"claude-3-haiku",
			"llama",
			"mixtral",
			"o4mini",
		},
		Default: cfg.DefaultModel,
	}
	survey.AskOne(prompt, &model)

	if model != "" {
		cfg.DefaultModel = model
		if err := saveConfig(cfg); err != nil {
			ui.Errorln("Error saving config: %v", err)
			return
		}
		chatSession.ChangeModel(models.GetModel(model))
		ui.AIln("Default model updated and applied: %s", model)
	} else {
		ui.Errorln("Invalid choice. No changes made.")
	}
}

func handleExportDirChange(cfg *Config) {
	path := ""
	prompt := &survey.Input{
		Message: "Enter new export directory path:",
		Default: cfg.ExportDir,
		Help:    "Press Enter to use the default path.",
	}
	survey.AskOne(prompt, &path)

	if path == "" {
		path = defaultExportPath()
	}

	cfg.ExportDir = path
	if err := ensureExportDir(cfg); err != nil {
		ui.Errorln("Error creating directory: %v", err)
	}

	if err := saveConfig(cfg); err != nil {
		ui.Errorln("Error saving config: %v", err)
	} else {
		ui.AIln("Export directory updated to: %s", path)
	}
}

func handleSearchSettings(cfg *Config) {
	qs := []*survey.Question{
		{
			Name:   "max_results",
			Prompt: &survey.Input{Message: "Max search results:", Default: strconv.Itoa(cfg.Search.MaxResults)},
		},
		{
			Name:   "include_snippet",
			Prompt: &survey.Confirm{Message: "Include snippets in search results?", Default: cfg.Search.IncludeSnippet},
		},
	}
	answers := struct {
		MaxResults     string `survey:"max_results"`
		IncludeSnippet bool   `survey:"include_snippet"`
	}{}

	err := survey.Ask(qs, &answers)
	if err != nil {
		ui.Errorln("Error reading input: %v", err)
		return
	}

	maxResults, err := strconv.Atoi(answers.MaxResults)
	if err != nil {
		ui.Errorln("Invalid number for max results: %v", err)
	} else {
		cfg.Search.MaxResults = maxResults
	}
	cfg.Search.IncludeSnippet = answers.IncludeSnippet

	if err := saveConfig(cfg); err != nil {
		ui.Errorln("Error saving config: %v", err)
	} else {
		ui.AIln("Search settings updated.")
	}
}

func handleShowMenuChange(cfg *Config) {
	showMenu := false
	prompt := &survey.Confirm{
		Message: "Show commands menu on startup?",
		Default: cfg.ShowMenu,
	}
	survey.AskOne(prompt, &showMenu)
	cfg.ShowMenu = showMenu
	if err := saveConfig(cfg); err != nil {
		ui.Errorln("Error saving config: %v", err)
	} else {
		ui.AIln("Show menu preference updated.")
	}
}

func handleGlobalPromptChange(cfg *Config) {
	prompt := ""
	p := &survey.Input{
		Message: "Enter global prompt (or leave empty to clear):",
		Default: cfg.GlobalPrompt,
	}
	survey.AskOne(p, &prompt)
	cfg.GlobalPrompt = prompt
	if err := saveConfig(cfg); err != nil {
		ui.Errorln("Error saving config: %v", err)
	} else {
		ui.AIln("Global prompt updated.")
	}
}

func handleLibrarySettings(cfg *Config) {
	choice := ""
	prompt := &survey.Select{
		Message: "Library Settings",
		Options: []string{
			fmt.Sprintf("Enabled (%t)", cfg.Library.Enabled),
			"Manage Directories",
			"Back",
		},
		Default: "Back",
	}
	survey.AskOne(prompt, &choice)

	switch {
	case strings.HasPrefix(choice, "Enabled"):
		cfg.Library.Enabled = !cfg.Library.Enabled
		if err := saveConfig(cfg); err != nil {
			ui.Errorln("Error saving config: %v", err)
		} else {
			ui.AIln("Library system set to: %t", cfg.Library.Enabled)
		}
	case choice == "Manage Directories":
		ui.Warningln("Directory management is handled via /library add and /library remove commands.")
	}
}

func handleAPISettings(cfg *Config) {
	for {
		choice := ""
		prompt := &survey.Select{
			Message: "API Settings",
			Options: []string{
				fmt.Sprintf("Enabled (%t)", cfg.API.Enabled),
				fmt.Sprintf("Port (%d)", cfg.API.Port),
				fmt.Sprintf("Autostart on launch (%t)", cfg.API.Autostart),
				fmt.Sprintf("Log API Requests (%t)", cfg.API.LogRequests),
				"Back",
			},
			Default: "Back",
		}
		survey.AskOne(prompt, &choice)

		switch {
		case strings.HasPrefix(choice, "Enabled"):
			cfg.API.Enabled = !cfg.API.Enabled
			saveAndReport(cfg, fmt.Sprintf("API Enabled status set to: %t", cfg.API.Enabled))
		case strings.HasPrefix(choice, "Port"):
			handleAPIPortChange(cfg)
		case strings.HasPrefix(choice, "Autostart"):
			cfg.API.Autostart = !cfg.API.Autostart
			saveAndReport(cfg, fmt.Sprintf("API Autostart set to: %t", cfg.API.Autostart))
		case strings.HasPrefix(choice, "Log API Requests"):
			cfg.API.LogRequests = !cfg.API.LogRequests
			saveAndReport(cfg, fmt.Sprintf("API Request Logging set to: %t", cfg.API.LogRequests))
		case choice == "Back":
			return
		}
	}
}

func handleAPIPortChange(cfg *Config) {
	portStr := ""
	prompt := &survey.Input{
		Message: "Enter API Port:",
		Default: strconv.Itoa(cfg.API.Port),
	}
	survey.AskOne(prompt, &portStr)

	if port, err := strconv.Atoi(portStr); err == nil {
		cfg.API.Port = port
		saveAndReport(cfg, fmt.Sprintf("API Port updated to: %d", port))
	} else {
		ui.Errorln("Invalid port number. No changes made.")
	}
}

func handleLongInputProtectionChange(cfg *Config) {
	confirmLongInput := false
	prompt := &survey.Confirm{
		Message: "Enable confirmation prompt for long input? (Helps prevent accidental AI requests when pasting URLs or long text)",
		Default: cfg.ConfirmLongInput,
		Help:    "When enabled, you'll be asked to confirm before sending long text (>500 chars), URLs, or multi-line content to the AI.",
	}
	survey.AskOne(prompt, &confirmLongInput)
	cfg.ConfirmLongInput = confirmLongInput
	if err := saveConfig(cfg); err != nil {
		ui.Errorln("Error saving config: %v", err)
	} else {
		ui.AIln("Long input protection set to: %t", confirmLongInput)
	}
}

func saveAndReport(cfg *Config, message string) {
	if err := saveConfig(cfg); err != nil {
		ui.Errorln("Error saving config: %v", err)
	} else {
		ui.AIln(message)
	}
}
