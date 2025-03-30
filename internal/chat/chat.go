package chat

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"duckduckgo-chat-cli/internal/config"
	"duckduckgo-chat-cli/internal/models"
	"duckduckgo-chat-cli/internal/scrape"

	"github.com/fatih/color"
)

type Chat struct {
	OldVqd     string
	NewVqd     string
	Model      models.Model
	Messages   []Message
	Client     *http.Client
	CookieJar  *cookiejar.Jar
	LastHash   string
	RetryCount int
}

type Message struct {
	Content string `json:"content"`
	Role    string `json:"role"`
}

type ChatPayload struct {
	Model    models.Model `json:"model"`
	Messages []Message    `json:"messages"`
}

func InitializeSession(cfg *config.Config) *Chat {
	model := models.GetModel(cfg.DefaultModel)
	chat := NewChat(GetVQD(), model)
	color.Green("Chat initialized with model: %s", model)
	setTerminalTitle(fmt.Sprintf("DuckDuckGo Chat - %s", model))
	return chat
}

func setTerminalTitle(title string) {
	switch runtime.GOOS {
	case "windows":
		exec.Command("cmd", "/c", fmt.Sprintf("title %s", title)).Run()
	default:
		fmt.Printf("\033]0;%s\007", title)
	}
}

func NewChat(vqd string, model models.Model) *Chat {
	jar, _ := cookiejar.New(nil)
	return &Chat{
		OldVqd:     vqd,
		NewVqd:     vqd,
		Model:      model,
		Messages:   []Message{},
		CookieJar:  jar,
		Client:     &http.Client{Timeout: 30 * time.Second, Jar: jar},
		RetryCount: 0,
	}
}

func GetVQD() string {
	req, _ := http.NewRequest("GET", models.StatusURL, nil)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36")
	req.Header.Set("x-vqd-accept", "1")
	req.Header.Set("Referer", "https://duckduckgo.com/")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		color.Red("Error fetching VQD: %v", err)
		return ""
	}
	defer resp.Body.Close()
	return resp.Header.Get("x-vqd-4")
}

func (c *Chat) generateClientHash() string {
	return ""
}

func (c *Chat) Clear(cfg *config.Config) {
	clearTerminal()

	if len(c.Messages) > 0 {
		c.Messages = []Message{}
		c.NewVqd = GetVQD()
		c.OldVqd = c.NewVqd
		color.Green("Chat history and context cleared")
	} else {
		color.Yellow("Chat is already empty")
	}

	if cfg.ShowMenu {
		PrintWelcomeMessage()
	} else {
		PrintCommands()
	}
}

func clearTerminal() {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "cls")
	default:
		cmd = exec.Command("clear")
	}
	cmd.Stdout = color.Output
	cmd.Run()
}

func ProcessInput(c *Chat, input string, cfg *config.Config) {
	if strings.TrimSpace(input) == "" {
		return
	}

	// Check if this is the first message and if a GlobalPrompt is defined
	isFirstMessage := len(c.Messages) == 0

	// If it's the first message, combine GlobalPrompt and user message
	actualMessage := input
	if isFirstMessage && cfg.GlobalPrompt != "" {
		// For the API, send GlobalPrompt and user message together
		actualMessage = cfg.GlobalPrompt + "\n\n" + input
	}

	// Add the combined message (or just user message) to send to the API
	c.Messages = append(c.Messages, Message{
		Role:    "user",
		Content: actualMessage,
	})

	// Send the message and display the response
	stream, err := c.FetchStream(actualMessage)
	if err != nil {
		color.Red("Error: %v", err)
		return
	}

	modelName := shortenModelName(string(c.Model))
	fmt.Print("\033[32m" + modelName + ":\033[0m ")

	var responseBuffer strings.Builder
	for chunk := range stream {
		fmt.Print(chunk)
		responseBuffer.WriteString(chunk)
	}
	fmt.Print("\n")

	c.Messages = append(c.Messages, Message{
		Role:    "assistant",
		Content: responseBuffer.String(),
	})
}

func shortenModelName(model string) string {
	displayNames := map[string]models.ModelAlias{
		"gpt-4o-mini":                               "gpt-4o-mini",
		"claude-3-haiku-20240307":                   "claude-3-haiku",
		"meta-llama/Llama-3.3-70B-Instruct-Turbo":   "llama",
		"mistralai/Mistral-Small-24B-Instruct-2501": "mixtral",
		"o3-mini": "o3mini",
	}

	if shortName, exists := displayNames[model]; exists {
		return string(shortName)
	}
	return model
}

func (c *Chat) FetchStream(content string) (<-chan string, error) {
	resp, err := c.Fetch(content)
	if err != nil {
		return nil, err
	}

	stream := make(chan string)
	go func() {
		defer resp.Body.Close()
		defer close(stream)

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
					stream <- messageData.Message
				}
			}
		}

		if err := scanner.Err(); err != nil {
			log.Printf("Error reading response body: %v\n", err)
		}

		if newVqd := resp.Header.Get("x-vqd-4"); newVqd != "" {
			c.OldVqd = c.NewVqd
			c.NewVqd = newVqd
		}

		c.RetryCount = 0
	}()

	return stream, nil
}

func (c *Chat) Fetch(content string) (*http.Response, error) {
	if c.NewVqd == "" {
		c.NewVqd = GetVQD()
		if c.NewVqd == "" {
			return nil, fmt.Errorf("failed to get VQD")
		}
	}

	payload := ChatPayload{
		Model:    c.Model,
		Messages: c.Messages,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling payload: %v", err)
	}

	if os.Getenv("DEBUG") == "true" {
		color.Cyan("Payload: %s", string(jsonPayload))
	}

	req, err := http.NewRequest("POST", models.ChatURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://duckduckgo.com/")
	req.Header.Set("Origin", "https://duckduckgo.com")
	req.Header.Set("x-vqd-4", c.NewVqd)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if os.Getenv("DEBUG") == "true" {
			color.Red("Request Headers: %+v", req.Header)
			color.Red("Response Headers: %+v", resp.Header)
		}

		if resp.StatusCode == 429 || strings.Contains(string(body), "ERR_INVALID_VQD") {
			time.Sleep(1 * time.Second)
			c.NewVqd = GetVQD()
			if c.NewVqd != "" && c.RetryCount < 3 {
				c.RetryCount++
				return c.Fetch(content)
			}
		}
		return nil, fmt.Errorf("%d: Failed to send message. %s. Body: %s", resp.StatusCode, resp.Status, string(body))
	}

	newVqd := resp.Header.Get("x-vqd-4")
	if newVqd != "" {
		c.OldVqd = c.NewVqd
		c.NewVqd = newVqd
	}

	return resp, nil
}

func (c *Chat) AddURLContext(url string) error {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	color.Yellow("⌛ Retrieving webpage content...")
	content, err := scrape.WebContent(url)
	if err != nil {
		return fmt.Errorf("failed to scrape URL: %v", err)
	}

	if content == "" {
		return fmt.Errorf("no content found at URL: %s", url)
	}

	contentLength := len(content)
	if contentLength > 500 {
		color.Cyan("Retrieved %d characters of content", contentLength)
	}

	c.Messages = append(c.Messages, Message{
		Role:    "user",
		Content: fmt.Sprintf("[URL Context]\nURL: %s\n\n%s", url, content),
	})

	return nil
}

func PrintCommands() {
	color.Yellow("Type /help to show these commands again")
}

func PrintWelcomeMessage() {
	color.Yellow("Special commands:")
	color.White("/search <query> - Search and add context")
	color.White("/file <path> - Add file content")
	color.White("/url <url> - Add webpage content")
	color.White("/clear - Clear context")
	color.White("/history - Show history")
	color.White("/export - Export messages")
	color.White("/copy - Copy to clipboard")
	color.White("/config - Configure settings")
	color.White("/model - Change AI model")
	color.White("/help - Show this menu")
	color.White("/exit - Quit")
}

func HandleURLCommand(c *Chat, input string) {
	url := strings.TrimPrefix(input, "/url ")
	color.Yellow("Scraping URL: %s (this may take a few seconds...)", url)

	if err := c.AddURLContext(url); err != nil {
		color.Red("URL error: %v", err)
	} else {
		color.Green("Successfully added webpage content from: %s", url)
	}
}

func HandleExportCommand(c *Chat, cfg *config.Config) {
	color.Yellow("\nExport options:")
	color.White("1. Full conversation")
	color.White("2. Last AI response")
	color.White("3. Largest code block")
	color.White("4. Search in conversation")
	color.White("5. Cancel")

	color.Blue("\nEnter your choice (1-5): ")
	reader := bufio.NewReader(os.Stdin)
	choice := strings.TrimSpace(readLine(reader))

	var filename, content string
	switch choice {
	case "1":
		filename, content = c.Export("conversation", "")
	case "2":
		filename, content = c.Export("last_response", "")
	case "3":
		filename, content = c.Export("code_block", "")
	case "4":
		searchText := readSearchInput()
		if searchText == "" {
			color.Yellow("⚠️ Search text cannot be empty")
			return
		}
		filename, content = c.Export("search_conversation", searchText)
	default:
		color.Yellow("💡 Export canceled.")
		return
	}

	if filename == "" || content == "" {
		color.Red("❌ Nothing to export")
		return
	}

	fullPath := filepath.Join(cfg.ExportDir, filename)
	if err := os.MkdirAll(cfg.ExportDir, 0755); err != nil {
		color.Red("❌ Cannot create export directory: %v", err)
		return
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		color.Red("❌ Error saving file: %v", err)
		return
	}

	color.Green("✅ Saved to: %s", fullPath)
}

func readLine(reader *bufio.Reader) string {
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func (c *Chat) ChangeModel(model models.Model) {
	c.Model = model
	displayName := shortenModelName(string(model))
	setTerminalTitle(fmt.Sprintf("DuckDuckGo Chat - %s", displayName))
	color.Green("Model changed to: %s", displayName)
}
