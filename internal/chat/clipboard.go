package chat

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/fatih/color"
)

func HandleCopyCommand(c *Chat) {
	color.Yellow("Choose what to copy:")
	color.White("1) Last Q&A exchange\n2) Largest code block\n3) Cancel")

	fmt.Print("Enter your choice: ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	var content string
	var err error

	switch input {
	case "1":
		content, err = c.copyLastExchange()
	case "2":
		content, err = c.copyLargestCodeBlock()
	default:
		color.Yellow("Copy canceled.")
		return
	}

	if err != nil {
		color.Red("Error: %v", err)
		return
	}

	if err := clipboard.WriteAll(content); err != nil {
		color.Red("Failed to copy to clipboard: %v", err)
		return
	}
	color.Green("Content copied to clipboard")
}

func (c *Chat) copyLastExchange() (string, error) {
	if len(c.Messages) < 2 {
		return "", fmt.Errorf("no complete exchange found")
	}

	var lastQuestion, lastAnswer string
	for i := len(c.Messages) - 1; i >= 0; i-- {
		if c.Messages[i].Role == "assistant" {
			lastAnswer = c.Messages[i].Content
			if i > 0 && c.Messages[i-1].Role == "user" {
				lastQuestion = c.Messages[i-1].Content
				break
			}
		}
	}

	if lastQuestion == "" || lastAnswer == "" {
		return "", fmt.Errorf("no complete exchange found")
	}

	return fmt.Sprintf("Q: %s\n\nA: %s", lastQuestion, lastAnswer), nil
}

func (c *Chat) copyLargestCodeBlock() (string, error) {
	var lastMessage Message
	for i := len(c.Messages) - 1; i >= 0; i-- {
		if c.Messages[i].Role == "assistant" {
			lastMessage = c.Messages[i]
			break
		}
	}

	if lastMessage.Content == "" {
		return "", fmt.Errorf("no AI response found")
	}

	codeBlockRegex := regexp.MustCompile("```(?:.*?)\n([\\s\\S]*?)```")
	matches := codeBlockRegex.FindAllStringSubmatch(lastMessage.Content, -1)

	if len(matches) == 0 {
		indentRegex := regexp.MustCompile("(?m)^( {4}|\\t).*(?:\n(?:(?:( {4}|\\t).*)|(?:\n))*)")
		matches = indentRegex.FindAllStringSubmatch(lastMessage.Content, -1)
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no code blocks found")
	}

	var largestBlock string
	maxLen := 0
	for _, match := range matches {
		block := match[1]
		if len(block) > maxLen {
			maxLen = len(block)
			largestBlock = block
		}
	}

	return largestBlock, nil
}
