package chat

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
)

func HandleFileCommand(c *Chat, input string) {
	path := strings.TrimPrefix(input, "/file ")
	color.Yellow("Adding file content: %s", path)

	if err := c.AddFileContext(path); err != nil {
		color.Red("File error: %v", err)
	} else {
		color.Green("Successfully added content from file: %s", path)
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
