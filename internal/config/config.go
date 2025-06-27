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

	"duckduckgo-chat-cli/internal/interfaces"
	"duckduckgo-chat-cli/internal/models"

	"github.com/fatih/color"
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

type Config struct {
	TOSAccepted    bool          `json:"tos_accepted"`
	DefaultModel   string        `json:"default_model"`
	ExportDir      string        `json:"export_dir"`
	LastUpdateTime time.Time     `json:"last_update_time"`
	Search         SearchConfig  `json:"search"`
	Library        LibraryConfig `json:"library"`
	ShowMenu       bool          `json:"show_menu"`
	GlobalPrompt   string        `json:"global_prompt"`
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

	if err := ensureExportDir(cfg); err != nil {
		color.Yellow("Warning: Failed to create export directory: %v", err)
	}
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
		if err := json.Unmarshal(data, cfg); err != nil {
			color.Yellow("Warning: Failed to parse config file: %v", err)
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

	color.Yellow("Please accept the terms of service")
	color.Blue("Do you accept? (yes/no): ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		color.Red("Error reading input: %v", err)
		return false
	}
	accepted := strings.ToLower(strings.TrimSpace(input)) == "yes"

	if accepted {
		cfg.TOSAccepted = true
		if err := saveConfig(cfg); err != nil {
			color.Yellow("Warning: Failed to save config: %v", err)
		}
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
		color.White("5. Global Prompt")
		color.White("6. Library Settings")
		color.White("7. Back to chat")

		// read user input
		reader := bufio.NewReader(os.Stdin)
		color.Blue("\nEnter your choice (1-7): ")
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
		case "5":
			handleGlobalPromptChange(cfg)
		case "6":
			handleLibrarySettings(cfg)
		case "7", "":
			return
		default:
			color.Red("Invalid choice. Please try again.")
		}
	}
}

func readInput(reader *bufio.Reader) string {
	input, err := reader.ReadString('\n')
	if err != nil {
		color.Red("Error reading input: %v", err)
		return ""
	}
	return strings.TrimSpace(input)
}

func handleModelChange(cfg *Config, chatSession interfaces.ChatSession) {
	color.Yellow("\nChoose Default Model:")
	color.White("1. GPT-4o mini")
	color.White("2. Claude 3 Haiku")
	color.White("3. Llama 3.3 70B")
	color.White("4. Mistral Small 3")
	color.White("5. o4-mini")

	reader := bufio.NewReader(os.Stdin)
	color.Blue("\nEnter your choice (1-5): ")
	choice := readInput(reader)

	modelMap := map[string]string{
		"1": "gpt-4o-mini",
		"2": "claude-3-haiku",
		"3": "llama",
		"4": "mixtral",
		"5": "o4mini",
	}

	if model, ok := modelMap[choice]; ok {
		cfg.DefaultModel = model
		if err := saveConfig(cfg); err != nil {
			color.Red("Error saving config: %v", err)
			return
		}
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
			if err := saveConfig(cfg); err != nil {
				color.Red("Error saving config: %v", err)
				return
			}
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
			if err := saveConfig(cfg); err != nil {
				color.Red("Error saving config: %v", err)
				return
			}
			color.Green("✅ Max results updated to: %d", max)
		} else {
			color.Red("❌ Invalid input. Must be between 1 and 20")
		}

	case "2":
		cfg.Search.IncludeSnippet = !cfg.Search.IncludeSnippet
		if err := saveConfig(cfg); err != nil {
			color.Red("Error saving config: %v", err)
			return
		}
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
			if err := saveConfig(cfg); err != nil {
				color.Red("Error saving config: %v", err)
				return
			}
			color.Green("✅ Max retries updated to: %d", retries)
		} else {
			color.Red("❌ Invalid input. Must be between 1 and 5")
		}

	case "4":
		color.Blue("Enter retry delay in seconds (1-10): ")
		if delay, err := strconv.Atoi(readInput(reader)); err == nil && delay > 0 && delay <= 10 {
			cfg.Search.RetryDelay = delay
			if err := saveConfig(cfg); err != nil {
				color.Red("Error saving config: %v", err)
				return
			}
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
	if err := saveConfig(cfg); err != nil {
		color.Red("Error saving config: %v", err)
		return
	}
	color.Green("Show commands menu updated to: %v", cfg.ShowMenu)
}

func handleGlobalPromptChange(cfg *Config) {
	color.Yellow("\nGlobal Prompt:")
	if cfg.GlobalPrompt != "" {
		color.White("Current prompt: %s", cfg.GlobalPrompt)
	} else {
		color.White("No global prompt set")
	}

	reader := bufio.NewReader(os.Stdin)
	color.Blue("\nEnter new global prompt (or leave empty to disable):\n")
	prompt := readInput(reader)

	if prompt == "" {
		color.Yellow("Global prompt disabled")
	} else {
		color.Green("Global prompt set to: %s", prompt)
	}

	cfg.GlobalPrompt = prompt
	if err := saveConfig(cfg); err != nil {
		color.Red("Error saving config: %v", err)
	}
}

func handleLibrarySettings(cfg *Config) {
	for {
		color.Yellow("\nLibrary Settings:")
		color.White("Current settings:")
		color.White("Enabled: %v", cfg.Library.Enabled)
		color.White("Directories (%d):", len(cfg.Library.Directories))
		for i, dir := range cfg.Library.Directories {
			color.White("  %d. %s", i+1, dir)
		}
		color.White("\nOptions:")
		color.White("1. Toggle library (enabled: %v)", cfg.Library.Enabled)
		color.White("2. Add directory")
		color.White("3. Remove directory")
		color.White("4. Clear all directories")
		color.White("5. Back")

		reader := bufio.NewReader(os.Stdin)
		color.Blue("\nEnter your choice (1-5): ")
		choice := readInput(reader)

		switch choice {
		case "1":
			cfg.Library.Enabled = !cfg.Library.Enabled
			if err := saveConfig(cfg); err != nil {
				color.Red("Error saving config: %v", err)
				return
			}
			if cfg.Library.Enabled {
				color.Green("✅ Library enabled")
			} else {
				color.Yellow("⚠️ Library disabled")
			}

		case "2":
			color.Blue("Enter directory path: ")
			path := readInput(reader)
			if path != "" {
				// Check if directory exists
				if info, err := os.Stat(path); err == nil && info.IsDir() {
					// Check if already exists
					exists := false
					for _, existing := range cfg.Library.Directories {
						if existing == path {
							exists = true
							break
						}
					}
					if !exists {
						cfg.Library.Directories = append(cfg.Library.Directories, path)
						if err := saveConfig(cfg); err != nil {
							color.Red("Error saving config: %v", err)
							return
						}
						color.Green("✅ Directory added: %s", path)
					} else {
						color.Yellow("⚠️ Directory already exists in library")
					}
				} else {
					color.Red("❌ Directory does not exist or is not accessible: %s", path)
				}
			}

		case "3":
			if len(cfg.Library.Directories) == 0 {
				color.Yellow("No directories to remove")
				continue
			}
			color.Yellow("Select directory to remove:")
			for i, dir := range cfg.Library.Directories {
				color.White("%d. %s", i+1, dir)
			}
			color.Blue("Enter directory number: ")
			if num, err := strconv.Atoi(readInput(reader)); err == nil && num > 0 && num <= len(cfg.Library.Directories) {
				removed := cfg.Library.Directories[num-1]
				cfg.Library.Directories = append(cfg.Library.Directories[:num-1], cfg.Library.Directories[num:]...)
				if err := saveConfig(cfg); err != nil {
					color.Red("Error saving config: %v", err)
					return
				}
				color.Green("✅ Directory removed: %s", removed)
			} else {
				color.Red("❌ Invalid directory number")
			}

		case "4":
			if len(cfg.Library.Directories) > 0 {
				cfg.Library.Directories = []string{}
				if err := saveConfig(cfg); err != nil {
					color.Red("Error saving config: %v", err)
					return
				}
				color.Green("✅ All directories cleared")
			} else {
				color.Yellow("No directories to clear")
			}

		case "5":
			return

		default:
			color.Red("❌ Invalid choice")
		}
	}
}
