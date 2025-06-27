package chat

import (
	"duckduckgo-chat-cli/internal/config"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
)

func HandleFileCommand(c *Chat, input string, cfg *config.Config) {
	// Parse the command: /file <path> -- <request>
	commandInput := strings.TrimPrefix(input, "/file ")

	var path, userRequest string

	// Check if there's a -- separator
	if strings.Contains(commandInput, " -- ") {
		parts := strings.SplitN(commandInput, " -- ", 2)
		path = strings.TrimSpace(parts[0])
		if len(parts) > 1 {
			userRequest = strings.TrimSpace(parts[1])
		}
	} else {
		// Fallback: if no --, treat everything as path for backward compatibility
		path = strings.TrimSpace(commandInput)
	}

	if path == "" {
		color.Red("Usage: /file <path> [-- request]")
		return
	}

	color.Yellow("Adding file content: %s", path)

	// Add file context first
	if err := c.AddFileContext(path); err != nil {
		color.Red("File error: %v", err)
		return
	}

	color.Green("Successfully added content from file: %s", path)

	// If user provided a specific request, process it with the file context
	if userRequest != "" {
		color.Cyan("Processing your request about the file...")
		ProcessInput(c, userRequest, cfg)
	} else {
		color.Yellow("File content added to context. You can now ask questions about it.")
	}
}

func (c *Chat) AddFileContext(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	contentLength := len(content)
	if contentLength > 500 {
		color.Cyan("Adding %d characters from file", contentLength)
	}

	c.Messages = append(c.Messages, Message{
		Role:    "user",
		Content: fmt.Sprintf("[File Context]\nFile: %s\n\n%s", path, string(content)),
	})
	return nil
}
