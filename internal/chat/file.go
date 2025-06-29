package chat

import (
	"duckduckgo-chat-cli/internal/config"
	"duckduckgo-chat-cli/internal/ui"
	"fmt"
	"os"
	"strings"
)

func HandleFileCommand(c *Chat, input string, cfg *config.Config) {
	var path, userRequest string
	var err error

	// Handle the case where the command is just "/file" to open the browser
	if strings.TrimSpace(input) == "/file" {
		path, err = ui.SelectFile()
		if err != nil {
			ui.Errorln("Error selecting file: %v", err)
			return
		}
	} else {
		// Handle the case with arguments: /file <path> [-- <request>]
		commandInput := strings.TrimPrefix(input, "/file ")

		if strings.Contains(commandInput, " -- ") {
			parts := strings.SplitN(commandInput, " -- ", 2)
			path = strings.TrimSpace(parts[0])
			if len(parts) > 1 {
				userRequest = strings.TrimSpace(parts[1])
			}
		} else {
			path = strings.TrimSpace(commandInput)
		}
	}

	// If no path was selected or provided, exit the command.
	if path == "" {
		ui.Warningln("No file selected or specified.")
		return
	}

	ui.Warningln("Adding file content: %s", path)

	// Add file context first
	if err := c.AddFileContext(path); err != nil {
		ui.Errorln("File error: %v", err)
		return
	}

	ui.AIln("Successfully added content from file: %s", path)

	// If user provided a specific request, process it with the file context
	if userRequest != "" {
		ui.Systemln("Processing your request about the file...")
		ProcessInput(c, userRequest, cfg)
	} else {
		ui.Warningln("File content added to context. You can now ask questions about it.")
	}
}

func (c *Chat) AddFileContext(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	contentLength := len(content)
	if contentLength > 500 {
		ui.AIln("Adding %d characters from file", contentLength)
	}

	c.Messages = append(c.Messages, Message{
		Role:    "user",
		Content: fmt.Sprintf("[File Context]\nFile: %s\n\n%s", path, string(content)),
	})
	return nil
}
