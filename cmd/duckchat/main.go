package main

import (
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"duckduckgo-chat-cli/internal/api"
	"duckduckgo-chat-cli/internal/chat"
	"duckduckgo-chat-cli/internal/config"
	"duckduckgo-chat-cli/internal/models"
	"duckduckgo-chat-cli/internal/ui"

	"github.com/AlecAivazis/survey/v2"
	"github.com/c-bata/go-prompt"
)

// Version will be set at build time via ldflags
var Version = "dev"

var chatSession *chat.Chat
var cfg *config.Config

var commands = []prompt.Suggest{
	{Text: "/help", Description: "Show the welcome message and command list"},
	{Text: "/exit", Description: "Exit the chat"},
	{Text: "/clear", Description: "Clear the chat history"},
	{Text: "/history", Description: "Show the chat history"},
	{Text: "/search", Description: "Search with a query"},
	{Text: "/file", Description: "Chat with a file"},
	{Text: "/library", Description: "Chat with your library"},
	{Text: "/url", Description: "Chat with a URL"},
	{Text: "/pmp", Description: "Use a predefined prompt"},
	{Text: "/export", Description: "Export the chat history"},
	{Text: "/copy", Description: "Copy the last response to the clipboard"},
	{Text: "/config", Description: "Open the configuration menu"},
	{Text: "/model", Description: "Change the chat model"},
	{Text: "/version", Description: "Show version information"},
	{Text: "/api", Description: "Start or stop the API server interactively"},
}

func completer(d prompt.Document) []prompt.Suggest {
	// Only show suggestions if the text starts with a slash.
	if !strings.HasPrefix(d.TextBeforeCursor(), "/") {
		return nil
	}

	// If there's a space, we assume the user is typing an argument, not a command.
	if strings.Contains(d.TextBeforeCursor(), " ") {
		return nil // No suggestions
	}

	// Otherwise, suggest commands.
	return prompt.FilterHasPrefix(commands, d.GetWordBeforeCursor(), true)
}

func main() {
	// create a channel to listen for interrupts
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		ui.Warningln("\nReceived interrupt. Exiting gracefully.")
		os.Exit(0)
	}()

	ui.Systemln("Welcome to DuckDuckGo AI Chat CLI!")

	cfg = config.Initialize()
	models.CheckChromeVersion()

	if !config.AcceptTermsOfService(cfg) {
		ui.Warningln("You must accept the terms to use this app. Exiting.")
		return
	}

	chatSession = chat.InitializeSession(cfg)

	if cfg.API.Enabled && cfg.API.Autostart {
		api.StartServer(chatSession, cfg, cfg.API.Port)
	}

	if cfg.ShowMenu {
		chat.PrintWelcomeMessage()
	} else {
		chat.PrintCommands()
	}

	p := prompt.New(
		executor,
		completer,
		prompt.OptionTitle("duckduckgo-chat-cli"),
		prompt.OptionPrefix("You: "),
		prompt.OptionPrefixTextColor(prompt.Blue),
	)
	p.Run()

}

func executor(input string) {
	if input == "" {
		return
	}
	if input == "/exit" {
		ui.Warningln("\nExiting chat. Goodbye!")
		os.Exit(0)
	}
	handleCommand(chatSession, cfg, input)
}

func handleCommand(chatSession *chat.Chat, cfg *config.Config, input string) {
	// if the input is empty, return
	if input == "" {
		return
	}

	switch {
	case input == "/clear":
		chatSession.Clear(cfg)
	case input == "/history":
		chat.PrintHistory(chatSession)
	case strings.HasPrefix(input, "/search "):
		chat.HandleSearchCommand(chatSession, input, cfg)
	case input == "/file" || strings.HasPrefix(input, "/file "):
		chat.HandleFileCommand(chatSession, input, cfg)
	case input == "/library" || strings.HasPrefix(input, "/library "):
		chat.HandleLibraryCommand(chatSession, input, cfg)
	case strings.HasPrefix(input, "/url "):
		chat.HandleURLCommand(chatSession, input, cfg)
	case strings.HasPrefix(input, "/pmp"):
		chat.HandlePMPCommand(chatSession, input, cfg)
	case input == "/export":
		chat.HandleExportCommand(chatSession, cfg)
	case input == "/copy":
		chat.HandleCopyCommand(chatSession)
	case input == "/config":
		config.HandleConfiguration(cfg, chatSession)
	case strings.HasPrefix(input, "/model"):
		modelArg := strings.TrimSpace(strings.TrimPrefix(input, "/model"))
		newModel := models.HandleModelChange(chatSession, modelArg)
		if newModel != "" {
			chatSession.ChangeModel(models.GetModel(string(newModel)))
			cfg.DefaultModel = string(newModel)
			if err := config.SaveConfig(cfg); err != nil {
				ui.Errorln("Failed to save config: %v", err)
			}
		}
	case input == "/help":
		chat.PrintWelcomeMessage()
	case strings.HasPrefix(input, "/api"):
		if api.IsRunning() {
			confirm := false
			prompt := &survey.Confirm{
				Message: "The API server is currently running. Do you want to stop it?",
				Default: true,
			}
			survey.AskOne(prompt, &confirm)
			if confirm {
				api.StopServer()
			}
		} else {
			if !cfg.API.Enabled {
				ui.Warningln("API is disabled in the configuration. Use /config to enable it.")
				return
			}
			portStr := strings.TrimSpace(strings.TrimPrefix(input, "/api"))
			port := cfg.API.Port
			if portStr != "" {
				if p, err := strconv.Atoi(portStr); err == nil {
					port = p
				} else {
					ui.Errorln("Invalid port number.")
					return
				}
			}
			api.StartServer(chatSession, cfg, port)
		}
	case input == "/version":
		ui.AIln("DuckDuckGo AI Chat CLI version %s", Version)
		ui.Mutedln("Go version: %s", runtime.Version())
		ui.Mutedln("OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	default:
		chat.ProcessInput(chatSession, input, cfg)
	}
}
