package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/fatih/color"
)

// Constants for API endpoints and headers
const (
	statusURL         = "https://duckduckgo.com/duckchat/v1/status"
	chatURL           = "https://duckduckgo.com/duckchat/v1/chat"
	statusHeaders     = "1"
	termsOfServiceURL = "https://duckduckgo.com/aichat/privacy-terms"
)

// Model represents the AI model used for chat
type Model string

// ModelAlias represents a user-friendly alias for the AI model
type ModelAlias string

// Define available models and their aliases
const (
	GPT4Mini Model = "gpt-4o-mini"
	Claude3  Model = "claude-3-haiku-20240307"
	Llama    Model = "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo"
	Mixtral  Model = "mistralai/Mixtral-8x7B-Instruct-v0.1"

	GPT4MiniAlias ModelAlias = "gpt-4o-mini"
	Claude3Alias  ModelAlias = "claude-3-haiku"
	LlamaAlias    ModelAlias = "llama"
	MixtralAlias  ModelAlias = "mixtral"
)

// Map model aliases to their corresponding Model values
var modelMap = map[ModelAlias]Model{
	GPT4MiniAlias: GPT4Mini,
	Claude3Alias:  Claude3,
	LlamaAlias:    Llama,
	MixtralAlias:  Mixtral,
}

// Message represents a chat message
type Message struct {
	Content string `json:"content"`
	Role    string `json:"role"`
}

// ChatPayload represents the payload sent to the chat API
type ChatPayload struct {
	Model    Model     `json:"model"`
	Messages []Message `json:"messages"`
}

// Chat represents a chat session
type Chat struct {
	OldVqd   string
	NewVqd   string
	Model    Model
	Messages []Message
	Client   *http.Client
}

// NewChat creates a new Chat instance
func NewChat(vqd string, model Model) *Chat {
	return &Chat{
		OldVqd:   vqd,
		NewVqd:   vqd,
		Model:    model,
		Messages: []Message{},
		Client:   &http.Client{},
	}
}

// Fetch sends a chat message and returns the response
func (c *Chat) Fetch(content string) (*http.Response, error) {
	c.Messages = append(c.Messages, Message{Content: content, Role: "user"})
	payload := ChatPayload{
		Model:    c.Model,
		Messages: c.Messages,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling payload: %v", err)
	}

	req, err := http.NewRequest("POST", chatURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("x-vqd-4", c.NewVqd)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("%d: Failed to send message. %s. Body: %s", resp.StatusCode, resp.Status, string(body))
	}

	return resp, nil
}

// FetchStream sends a chat message and returns a channel for streaming the response
func (c *Chat) FetchStream(content string) (<-chan string, error) {
	resp, err := c.Fetch(content)
	if err != nil {
		return nil, err
	}

	stream := make(chan string)
	go func() {
		defer resp.Body.Close()
		defer close(stream)

		var text strings.Builder
		scanner := bufio.NewScanner(resp.Body)

		for scanner.Scan() {
			line := scanner.Text()

			if line == "data: [DONE]" {
				break
			}

			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				var messageData struct {
					Message string `json:"message"`
				}
				if err := json.Unmarshal([]byte(data), &messageData); err != nil {
					log.Printf("Error unmarshaling data: %v\n", err)
					continue
				}

				if messageData.Message != "" {
					text.WriteString(messageData.Message)
					stream <- messageData.Message
				}
			}
		}

		if err := scanner.Err(); err != nil {
			log.Printf("Error reading response body: %v\n", err)
		}

		c.OldVqd = c.NewVqd
		c.NewVqd = resp.Header.Get("x-vqd-4")
		c.Messages = append(c.Messages, Message{Content: text.String(), Role: "assistant"})
	}()

	return stream, nil
}

// Redo resets the chat to the previous state
func (c *Chat) Redo() {
	c.NewVqd = c.OldVqd
	if len(c.Messages) >= 2 {
		c.Messages = c.Messages[:len(c.Messages)-2]
	}
}

// InitChat initializes a new chat session
func InitChat(model ModelAlias) (*Chat, error) {
	req, err := http.NewRequest("GET", statusURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("x-vqd-accept", statusHeaders)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%d: Failed to initialize chat. %s", resp.StatusCode, resp.Status)
	}

	vqd := resp.Header.Get("x-vqd-4")
	if vqd == "" {
		return nil, fmt.Errorf("failed to get VQD from response headers")
	}

	return NewChat(vqd, modelMap[model]), nil
}

func acceptTermsOfService() bool {
	color.Yellow("Before using this application, you must accept the terms of service.")
	color.White("Please read the terms of service at: %s", termsOfServiceURL)
	color.Blue("Do you accept the terms of service? (yes/no): ")

	reader := bufio.NewReader(os.Stdin)
	for {
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "yes", "y":
			return true
		case "no", "n":
			return false
		default:
			color.Red("Invalid input. Please enter 'yes' or 'no'.")
		}
	}
}

func main() {
	color.Cyan("Welcome to DuckDuckGo AI Chat CLI!")

	if !acceptTermsOfService() {
		color.Yellow("You must accept the terms of service to use this application. Exiting.")
		return
	}

	model := chooseModel()

	chat, err := InitChat(model)
	if err != nil {
		log.Fatalf("Failed to initialize chat: %v", err)
	}

	color.Green("Chat initialized successfully. You can start chatting now.")
	color.Yellow("Type 'exit' to end the conversation.")

	reader := bufio.NewReader(os.Stdin)
	for {
		color.Blue("You: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "exit" {
			color.Yellow("Exiting chat. Goodbye!")
			break
		}

		stream, err := chat.FetchStream(input)
		if err != nil {
			color.Red("Error: %v", err)
			continue
		}

		color.Green("AI: ")
		printResponse(stream)
	}
}

// chooseModel prompts the user to select an AI model
func chooseModel() ModelAlias {
	color.Yellow("Please choose an AI model:")
	color.White("1. GPT-4o mini")
	color.White("2. Claude 3 Haiku")
	color.White("3. Llama 3.1 70B")
	color.White("4. Mixtral 8x7B")

	reader := bufio.NewReader(os.Stdin)
	for {
		color.Blue("Enter your choice (1-4): ")
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			return GPT4MiniAlias
		case "2":
			return Claude3Alias
		case "3":
			return LlamaAlias
		case "4":
			return MixtralAlias
		default:
			color.Red("Invalid choice. Please try again.")
		}
	}
}

// printResponse prints the AI's response in real-time
func printResponse(stream <-chan string) {
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for chunk := range stream {
			fmt.Print(chunk)
		}
	}()

	wg.Wait()
	fmt.Println()
}
