package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"duckduckgo-chat-cli/internal/chat"
	"duckduckgo-chat-cli/internal/config"
	"duckduckgo-chat-cli/internal/models"

	"github.com/fatih/color"
)

func main() {
	// create a channel to listen for interrupts
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	color.Cyan("Welcome to DuckDuckGo AI Chat CLI!")

	cfg := config.Initialize()
	models.CheckChromeVersion()

	if !config.AcceptTermsOfService(cfg) {
		color.Yellow("You must accept the terms to use this app. Exiting.")
		return
	}

	chatSession := chat.InitializeSession(cfg)

	// move the cleanup function to a defer statement
	cleanup := func() {
		color.Yellow("\nExiting chat. Goodbye!")
	}
	defer cleanup()

	if cfg.ShowMenu {
		chat.PrintWelcomeMessage()
	} else {
		chat.PrintCommands()
	}

	// create a channel to listen for user input
	inputChan := make(chan string)
	stopChan := make(chan struct{})
	go readInput(inputChan, stopChan)

	for {
		select {
		case <-sigChan:
			// stop the input goroutine 
			close(stopChan)
			fmt.Println() // clear the line
			return
		case input := <-inputChan:
			if input == "/exit" {
				close(stopChan)
				return
			}
			handleCommand(chatSession, cfg, input)
			go readInput(inputChan, stopChan)
		}
	}
}

func readInput(inputChan chan string, stopChan chan struct{}) {
	reader := bufio.NewReader(os.Stdin)

	// show the user prompt
	fmt.Print("\033[34mYou: \033[0m") // Blue color without newline

	// read the user input
	input, err := reader.ReadString('\n')
	select {
	case <-stopChan:
		return
	default:
		if err != nil {
			return
		}
		inputChan <- strings.TrimSpace(input)
	}
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
	case strings.HasPrefix(input, "/file "):
		chat.HandleFileCommand(chatSession, input)
	case strings.HasPrefix(input, "/url "):
		chat.HandleURLCommand(chatSession, input)
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
				color.Red("Failed to save config: %v", err)
			}
		}
	case input == "/help":
		chat.PrintWelcomeMessage()
	default:
		chat.ProcessInput(chatSession, input, cfg)
	}
}
