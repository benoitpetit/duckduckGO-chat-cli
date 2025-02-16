package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"duckduckgo-chat-cli/internal/interfaces"
	"duckduckgo-chat-cli/internal/models"
)

type SearchConfig struct {
	MaxResults     int  `json:"max_results"`
	IncludeSnippet bool `json:"include_snippet"`
	MaxRetries     int  `json:"max_retries"`
	RetryDelay     int  `json:"retry_delay"`
}

type Config struct {
	TOSAccepted    bool         `json:"tos_accepted"`
	DefaultModel   string       `json:"default_model"`
	ExportDir      string       `json:"export_dir"`
	LastUpdateTime time.Time    `json:"last_update_time"`
	Search         SearchConfig `json:"search"`
	ShowMenu       bool         `json:"show_menu"`
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

	ensureExportDir(cfg)
	return cfg
}

func loadConfig() *Config {
	cfg := &Config{
		TOSAccepted:    false,
		DefaultModel:   "gpt-4o-mini",
		ExportDir:      defaultExportPath(),
		LastUpdateTime: time.Now(),
	}

	if data, err := os.ReadFile(configPath()); err == nil {
		json.Unmarshal(data, cfg)
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
	configDir, _ := os.UserConfigDir()
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

	color.Yellow("Please accept the terms of service")
	color.Blue("Do you accept? (yes/no): ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	accepted := strings.ToLower(strings.TrimSpace(input)) == "yes"

	if accepted {
		cfg.TOSAccepted = true
		saveConfig(cfg)
	}
	return accepted
}

func HandleConfiguration(cfg *Config, chatSession interfaces.ChatSession) {
	for {
		// show configuration menu
		color.Yellow("\nDuckDuckGo Chat CLI Configuration")
		color.White("Current settings:")
		color.White("1. Default Model: %s", cfg.DefaultModel)
		color.White("2. Export Directory: %s", cfg.ExportDir)
		color.White("3. Search Settings")
		color.White("4. Show Commands Menu: %v", cfg.ShowMenu)
		color.White("5. Back to chat")

		// read user input
		reader := bufio.NewReader(os.Stdin)
		color.Blue("\nEnter your choice (1-5): ")
		choice := readInput(reader)

		switch choice {
		case "1":
			handleModelChange(cfg, chatSession)
		case "2":
			handleExportDirChange(cfg)
		case "3":
			handleSearchSettings(cfg)
		case "4":
			handleShowMenuChange(cfg)
		case "5", "":
			return
		default:
			color.Red("Invalid choice. Please try again.")
		}
	}
}

func readInput(reader *bufio.Reader) string {
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func handleModelChange(cfg *Config, chatSession interfaces.ChatSession) {
	color.Yellow("\nChoose Default Model:")
	color.White("1. GPT-4o mini")
	color.White("2. Claude 3 Haiku")
	color.White("3. Llama 3.3 70B")
	color.White("4. Mixtral 8x7B")
	color.White("5. o3-mini")

	reader := bufio.NewReader(os.Stdin)
	color.Blue("\nEnter your choice (1-5): ")
	choice := readInput(reader)

	modelMap := map[string]string{
		"1": "gpt-4o-mini",
		"2": "claude-3-haiku",
		"3": "llama",
		"4": "mixtral",
		"5": "o3mini",
	}

	if model, ok := modelMap[choice]; ok {
		cfg.DefaultModel = model
		saveConfig(cfg)
		chatSession.ChangeModel(models.GetModel(model))
		color.Green("Default model updated and applied: %s", model)
	} else {
		color.Red("Invalid choice. No changes made.")
	}
}

func handleExportDirChange(cfg *Config) {
	reader := bufio.NewReader(os.Stdin)
	color.Blue("\nEnter new export directory path (or press Enter for default): ")
	path := readInput(reader)

	if path == "" {
		// use default export path
		userHome, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(userHome, "Documents", "duckchat")
		}
	}

	if path != "" {
		if err := os.MkdirAll(path, 0755); err == nil {
			cfg.ExportDir = path
			saveConfig(cfg)
			color.Green("Export directory updated to: %s", path)
		} else {
			color.Red("Error creating directory: %v", err)
		}
	}
}

func handleSearchSettings(cfg *Config) {
	color.Yellow("\nSearch Settings:")
	color.White("Current settings:")
	color.White("1. Max Results: %d", cfg.Search.MaxResults)
	color.White("2. Include Snippets: %v", cfg.Search.IncludeSnippet)
	color.White("3. Max Retries: %d", cfg.Search.MaxRetries)
	color.White("4. Retry Delay: %d seconds", cfg.Search.RetryDelay)
	color.White("5. Back")

	reader := bufio.NewReader(os.Stdin)
	color.Blue("\nEnter your choice (1-5): ")
	choice := readInput(reader)

	switch choice {
	case "1":
		color.Blue("Enter maximum number of results (1-20): ")
		if max, err := strconv.Atoi(readInput(reader)); err == nil && max > 0 && max <= 20 {
			cfg.Search.MaxResults = max
			saveConfig(cfg)
			color.Green("✅ Max results updated to: %d", max)
		} else {
			color.Red("❌ Invalid input. Must be between 1 and 20")
		}

	case "2":
		cfg.Search.IncludeSnippet = !cfg.Search.IncludeSnippet
		saveConfig(cfg)
		color.Green("✅ Include snippets set to: %v", cfg.Search.IncludeSnippet)
		if cfg.Search.IncludeSnippet {
			color.Yellow("ℹ️ Search results will include descriptions")
		} else {
			color.Yellow("ℹ️ Search results will be compact (titles only)")
		}

	case "3":
		color.Blue("Enter maximum number of retries (1-5): ")
		if retries, err := strconv.Atoi(readInput(reader)); err == nil && retries > 0 && retries <= 5 {
			cfg.Search.MaxRetries = retries
			saveConfig(cfg)
			color.Green("✅ Max retries updated to: %d", retries)
		} else {
			color.Red("❌ Invalid input. Must be between 1 and 5")
		}

	case "4":
		color.Blue("Enter retry delay in seconds (1-10): ")
		if delay, err := strconv.Atoi(readInput(reader)); err == nil && delay > 0 && delay <= 10 {
			cfg.Search.RetryDelay = delay
			saveConfig(cfg)
			color.Green("✅ Retry delay updated to: %d seconds", delay)
		} else {
			color.Red("❌ Invalid input. Must be between 1 and 10")
		}

	case "5":
		return

	default:
		color.Red("❌ Invalid choice")
	}
}

func handleShowMenuChange(cfg *Config) {
	cfg.ShowMenu = !cfg.ShowMenu
	saveConfig(cfg)
	color.Green("Show commands menu updated to: %v", cfg.ShowMenu)
}