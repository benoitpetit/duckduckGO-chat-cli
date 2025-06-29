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
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"duckduckgo-chat-cli/internal/chatcontext"
	"duckduckgo-chat-cli/internal/config"
	"duckduckgo-chat-cli/internal/models"
	"duckduckgo-chat-cli/internal/scrape"
	"duckduckgo-chat-cli/internal/ui"

	"github.com/AlecAivazis/survey/v2"
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
	FeSignals  string
	FeVersion  string
	VqdHash1   string
}

type Message struct {
	Content string `json:"content"`
	Role    string `json:"role"`
}

type ToolChoice struct {
	NewsSearch      bool `json:"NewsSearch"`
	VideosSearch    bool `json:"VideosSearch"`
	LocalSearch     bool `json:"LocalSearch"`
	WeatherForecast bool `json:"WeatherForecast"`
}

type Metadata struct {
	ToolChoice ToolChoice `json:"toolChoice"`
}

type ChatPayload struct {
	Model       models.Model `json:"model"`
	Metadata    Metadata     `json:"metadata"`
	Messages    []Message    `json:"messages"`
	CanUseTools bool         `json:"canUseTools"`
}

func InitializeSession(cfg *config.Config) *Chat {
	model := models.GetModel(cfg.DefaultModel)
	chat := NewChat(GetVQD(), model)
	ui.AIln("Chat initialized with model: %s", model)
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

	// Set required cookies avec TOUS les cookies pour une session authentique
	u, _ := url.Parse("https://duckduckgo.com")
	cookies := []*http.Cookie{
		{Name: "5", Value: "1", Domain: ".duckduckgo.com"},
		{Name: "dcm", Value: "3", Domain: ".duckduckgo.com"},
		{Name: "dcs", Value: "1", Domain: ".duckduckgo.com"},
		// Cookies supplémentaires basés sur l'analyse de l'image du navigateur
		{Name: "duckassist-opt-in-count", Value: "1", Domain: ".duckduckgo.com"},
		{Name: "isRecentChatOn", Value: "1", Domain: ".duckduckgo.com"},
		{Name: "preferredDuckAiModel", Value: "3", Domain: ".duckduckgo.com"},
	}
	jar.SetCookies(u, cookies)

	// Try to get dynamic headers
	var feSignals, feVersion, vqdHash1 string

	ui.Warningln("⌛ Attempting to extract dynamic headers from DuckDuckGo...")
	if dynamicHeaders, err := ExtractDynamicHeaders(); err == nil {
		feSignals = dynamicHeaders.FeSignals
		feVersion = dynamicHeaders.FeVersion
		vqdHash1 = dynamicHeaders.VqdHash1
		ui.AIln("✅ Successfully extracted dynamic headers")
	} else {
		ui.Warningln("⚠️ Failed to get dynamic headers, falling back to placeholders. Error: %v", err)
		// Fallback to working static values
		feSignals = "eyJzdGFydCI6MTc0OTgyNTIxMDQ0MiwiZXZlbnRzIjpbeyJuYW1lIjoic3RhcnROZXdDaGF0IiwiZGVsdGEiOjQxfV0sImVuZCI6NDk2OX0="
		feVersion = "serp_20250613_085800_ET-cafd73f97f51c983eb30"
		vqdHash1 = "eyJzZXJ2ZXJfaGFzaGVzIjpbIkJEcjlUQ0o0ZzB4aG9UZ1BmSDI0Lzg3SUg0TFRqeFRYQkRvcU9KY2tKRFU9IiwidjY5enNacDErcUJCNWhzUDhwQ0I0aHUyR0pwQlpIeFhzNlRtQ1RtcXlMRT0iXSwiY2xpZW50X2hhc2hlcyI6WyJpRTNqeXRnSm0xZGJaZlo1bW81M1NmaVAxdXUxeEdzY0F5RnB3V2NVOUtrPSIsImlqZHFtVzl0ZnBXTHZXaW9ka2twYnFLSjdTaUEzb3MxakxrZm9HcWozOFk9Il0sInNpZ25hbHMiOnt9LCJtZXRhIjp7InYiOiIzIiwiY2hhbGxlbmdlX2lkIjoiZGFmMjlmMTdjNzQ2MDQ2ZTU4NjlhYmI4NjgyNGMxNmE1NTQ4MDlhZDFiNjE5ZWI5MTFkZWJmNTc2NzU2NzM3NGg4amJ0IiwidGltZXN0YW1wIjoiMTc0OTgyNTIwOTg1MCIsIm9yaWdpbiI6Imh0dHBzOi8vZHVja2R1Y2tnby5jb20iLCJzdGFjayI6IkVycm9yXG5hdCBiYSAoaHR0cHM6Ly9kdWNrZHVja2dvLmNvbS9kaXN0L3dwbS5jaGF0LmNhZmQ3M2Y5N2Y1MWM5ODNlYjMwLmpzOjE6NzQ4MDMpXG5hdCBhc3luYyBkaXNwYXRjaFNlcnZpY2VJbml0aWFsVlFEIChodHRwczovL2R1Y2tkdWNrZ28uY29tL2Rpc3Qvd3BtLmNoYXQuY2FmZDczZjk3ZjUxYzk4M2ViMzAuanM6MTo5OTUyOSkifX0="
	}

	chat := &Chat{
		OldVqd:     vqd,
		NewVqd:     vqd,
		Model:      model,
		Messages:   []Message{},
		CookieJar:  jar,
		Client:     &http.Client{Timeout: 30 * time.Second, Jar: jar},
		RetryCount: 0,
		FeSignals:  feSignals,
		FeVersion:  feVersion,
		VqdHash1:   vqdHash1,
	}

	return chat
}

func GetVQD() string {
	client := &http.Client{Timeout: 10 * time.Second}

	// Set up cookies avec TOUS les cookies nécessaires
	jar, _ := cookiejar.New(nil)
	u, _ := url.Parse("https://duckduckgo.com")
	cookies := []*http.Cookie{
		{Name: "5", Value: "1", Domain: ".duckduckgo.com"},
		{Name: "dcm", Value: "3", Domain: ".duckduckgo.com"},
		{Name: "dcs", Value: "1", Domain: ".duckduckgo.com"},
		// Cookies supplémentaires pour une session authentique
		{Name: "duckassist-opt-in-count", Value: "1", Domain: ".duckduckgo.com"},
		{Name: "isRecentChatOn", Value: "1", Domain: ".duckduckgo.com"},
		{Name: "preferredDuckAiModel", Value: "3", Domain: ".duckduckgo.com"},
	}
	jar.SetCookies(u, cookies)
	client.Jar = jar

	req, _ := http.NewRequest("GET", models.StatusURL, nil)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.6")
	req.Header.Set("Cache-Control", "no-store")
	req.Header.Set("DNT", "1")
	req.Header.Set("Priority", "u=1, i")
	req.Header.Set("Referer", "https://duckduckgo.com/")
	req.Header.Set("Sec-CH-UA", `"Brave";v="137", "Chromium";v="137", "Not/A)Brand";v="24"`)
	req.Header.Set("Sec-CH-UA-Mobile", "?0")
	req.Header.Set("Sec-CH-UA-Platform", `"Windows"`)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-GPC", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36")
	req.Header.Set("x-vqd-accept", "1")

	resp, err := client.Do(req)
	if err != nil {
		ui.Errorln("Error fetching VQD: %v", err)
		return ""
	}
	defer resp.Body.Close()
	return resp.Header.Get("x-vqd-hash-1")
}

func (c *Chat) Clear(cfg *config.Config) {
	clearTerminal()

	if len(c.Messages) > 0 {
		c.Messages = []Message{}
		c.NewVqd = GetVQD()
		c.OldVqd = c.NewVqd
		// Hash will be refreshed on next request if needed
		c.RetryCount = 0
		ui.AIln("Chat history and context cleared")
	} else {
		ui.Warningln("Chat is already empty")
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

	isFirstMessage := len(c.Messages) == 0
	actualMessage := input
	if isFirstMessage && cfg.GlobalPrompt != "" {
		actualMessage = cfg.GlobalPrompt + "\n\n" + input
	}

	c.Messages = append(c.Messages, Message{
		Role:    "user",
		Content: actualMessage,
	})

	stream, err := c.FetchStream(actualMessage)
	if err != nil {
		ui.Errorln("Error: %v", err)
		return
	}

	// Use the new stable streaming renderer
	modelName := shortenModelName(string(c.Model))
	finalResponse := RenderStream(stream, modelName)

	c.Messages = append(c.Messages, Message{
		Role:    "assistant",
		Content: finalResponse,
	})
}

// renderStreamToString captures the stream output into a single string.
func renderStreamToString(stream <-chan string) string {
	var fullResponse strings.Builder
	var currentLine strings.Builder
	inCodeBlock := false
	var codeBlockLang string

	for chunk := range stream {
		for _, char := range chunk {
			currentLine.WriteRune(char)
			// This is a simplified version of RenderStream.
			// It doesn't handle complex ANSI and formatting, just captures the text.
			if char == '\n' {
				lineStr := currentLine.String()
				if strings.HasPrefix(lineStr, "```") {
					if !inCodeBlock {
						inCodeBlock = true
						codeBlockLang = strings.TrimSpace(strings.TrimPrefix(lineStr, "```"))
						fullResponse.WriteString("```" + codeBlockLang + "\n")
					} else {
						inCodeBlock = false
						fullResponse.WriteString("```\n")
					}
				} else {
					fullResponse.WriteString(lineStr)
				}
				currentLine.Reset()
			}
		}
	}

	if currentLine.Len() > 0 {
		fullResponse.WriteString(currentLine.String())
	}

	return fullResponse.String()
}

func ProcessInputAndReturn(c *Chat, input string, cfg *config.Config) (string, error) {
	if strings.TrimSpace(input) == "" {
		return "", nil
	}

	// Check if this is the first message and if a GlobalPrompt is defined
	isFirstMessage := len(c.Messages) == 0

	// If it's the first message, combine GlobalPrompt and user message
	actualMessage := input
	if isFirstMessage && cfg.GlobalPrompt != "" {
		actualMessage = cfg.GlobalPrompt + "\n\n" + input
	}

	c.Messages = append(c.Messages, Message{
		Role:    "user",
		Content: actualMessage,
	})

	stream, err := c.FetchStream(actualMessage)
	if err != nil {
		return "", fmt.Errorf("error fetching stream: %w", err)
	}

	// Capture the entire response from the stream
	finalResponse := renderStreamToString(stream)

	// Add the assistant's response to the message history
	c.Messages = append(c.Messages, Message{
		Role:    "assistant",
		Content: finalResponse,
	})

	return finalResponse, nil
}

func shortenModelName(model string) string {
	displayNames := map[string]models.ModelAlias{
		"gpt-4o-mini":                               "gpt-4o-mini",
		"claude-3-haiku-20240307":                   "claude-3-haiku",
		"meta-llama/Llama-3.3-70B-Instruct-Turbo":   "llama",
		"mistralai/Mistral-Small-24B-Instruct-2501": "mixtral",
		"o4-mini": "o4mini",
		"o3-mini": "o3mini",
	}

	if shortName, exists := displayNames[model]; exists {
		return string(shortName)
	}

	return "unknown"
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

	// VQD hash is now initialized during chat creation

	payload := ChatPayload{
		Model: c.Model,
		Metadata: Metadata{
			ToolChoice: ToolChoice{
				NewsSearch:      false,
				VideosSearch:    false,
				LocalSearch:     false,
				WeatherForecast: false,
			},
		},
		Messages:    c.Messages,
		CanUseTools: true,
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

	// Set all required headers based on the curl requests
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.6")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("DNT", "1")
	req.Header.Set("Origin", "https://duckduckgo.com")
	req.Header.Set("Priority", "u=1, i")
	req.Header.Set("Referer", "https://duckduckgo.com/")
	req.Header.Set("Sec-CH-UA", `"Brave";v="137", "Chromium";v="137", "Not/A)Brand";v="24"`)
	req.Header.Set("Sec-CH-UA-Mobile", "?0")
	req.Header.Set("Sec-CH-UA-Platform", `"Windows"`)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-GPC", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36")
	req.Header.Set("x-fe-signals", c.FeSignals)
	req.Header.Set("x-fe-version", c.FeVersion)
	req.Header.Set("x-vqd-4", c.NewVqd)

	if c.VqdHash1 != "" {
		req.Header.Set("x-vqd-hash-1", c.VqdHash1)
	}

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
			color.Red("Response Status: %d", resp.StatusCode)
			color.Red("Response Body: %s", string(body))
		}

		// Handle various error conditions including 418 (I'm a teapot)
		if resp.StatusCode == 418 || resp.StatusCode == 429 || strings.Contains(string(body), "ERR_INVALID_VQD") {
			time.Sleep(2 * time.Second)

			// Refresh both VQD and dynamic headers on 418 errors
			c.NewVqd = GetVQD()
			if resp.StatusCode == 418 && c.RetryCount == 0 {
				ui.Warningln("🔄 Error 418 detected, refreshing headers...")
				c.RefreshDynamicHeaders() // Try refreshing headers on first 418 error
			}

			if c.NewVqd != "" && c.RetryCount < 3 {
				c.RetryCount++
				ui.Warningln("Retrying request (attempt %d/3)...", c.RetryCount)
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

	ui.Warningln("⌛ Retrieving webpage content...")
	content, err := scrape.WebContent(url)
	if err != nil {
		return err
	}

	contentLength := len(content.Content)
	if contentLength > 500 {
		ui.AIln("Retrieved %d characters of content", contentLength)
	}

	c.Messages = append(c.Messages, Message{
		Role:    "user",
		Content: fmt.Sprintf("[URL Context]\nURL: %s\n\n%s", url, content.Content),
	})

	return nil
}

func PrintCommands() {
	ui.Warningln("Type /help to show these commands again")
}

// CommandHelp holds information about a CLI command for the help message.
type CommandHelp struct {
	Command     string
	Description string
}

func PrintWelcomeMessage() {
	ui.Systemln("\nDuckDuckGo AI Chat CLI - Help")
	ui.Mutedln("---------------------------------")

	coreCommands := []CommandHelp{
		{"/help", "Show this help message"},
		{"/exit", "Exit the application"},
		{"/clear", "Clear the current chat session and context"},
		{"/history", "Display the conversation history"},
		{"/model", "Change the AI model interactively"},
		{"/config", "Open the interactive configuration menu"},
		{"/copy", "Copy the last AI response to the clipboard"},
		{"/export", "Export the conversation to a file"},
		{"/version", "Show application version information"},
		{"/api", "Start or stop the API server interactively"},
	}

	contextCommands := []CommandHelp{
		{"/search <query>", "Search the web and add results to context"},
		{"/file <path>", "Add the content of a local file to context"},
		{"/url <url>", "Add the content of a webpage to context"},
		{"/library", "Manage and use your local document library"},
		{"/pmp", "Use Pre-Made Prompts for structured generation"},
	}

	apiCommands := []CommandHelp{
		{"GET /", "Shows API documentation"},
		{"POST /chat", "Sends a message to the chat"},
		{"GET /history", "Retrieves the chat session history"},
	}

	ui.AIln("\nCore Commands:")
	printCommandsTable(coreCommands)

	ui.AIln("\nContext Commands:")
	printCommandsTable(contextCommands)

	ui.AIln("\nAPI Documentation:")
	printCommandsTable(apiCommands)

	ui.Warningln("\nNote: You can add '-- <your request>' after /search, /file, or /url to make an immediate request about the context.")
}

// printCommandsTable formats and prints a list of commands.
func printCommandsTable(commands []CommandHelp) {
	// Find the longest command to align descriptions
	maxLength := 0
	for _, cmd := range commands {
		if len(cmd.Command) > maxLength {
			maxLength = len(cmd.Command)
		}
	}

	for _, cmd := range commands {
		// Use UserColor for the command and default white for the description
		ui.UserColor.Printf("  %-*s", maxLength+4, cmd.Command)
		ui.Whiteln("- %s", cmd.Description)
	}
}

func HandleURLCommand(c *Chat, input string, cfg *config.Config, chainCtx *chatcontext.Context) {
	urlStr := strings.TrimSpace(strings.TrimPrefix(input, "/url"))
	if urlStr == "" {
		ui.Errorln("URL cannot be empty.")
		return
	}

	result, err := scrape.WebContent(urlStr)
	if err != nil {
		ui.Errorln("URL error: %v", err)
		return
	}

	if chainCtx != nil {
		chainCtx.AddURL(urlStr, result.Content)
		ui.AIln("Successfully added content from URL to chain context: %s", urlStr)
	} else {
		ui.Warningln("Adding URL content: %s", urlStr)
		c.addURLContext(urlStr, result.Content)
		ui.AIln("Successfully added content from URL: %s", urlStr)
		ui.Warningln("You can now ask questions about the URL content.")
	}
}

func (c *Chat) addURLContext(url string, content string) {
	contentLength := len(content)
	if contentLength > 500 {
		ui.AIln("Adding %d characters from URL", contentLength)
	}

	c.Messages = append(c.Messages, Message{
		Role:    "user",
		Content: fmt.Sprintf("[URL Context]\nURL: %s\n\n%s", url, content),
	})
}

func HandleExportCommand(c *Chat, cfg *config.Config) {
	var choice string
	prompt := &survey.Select{
		Message: "Choose what to export:",
		Options: []string{
			"Full conversation",
			"Last AI response",
			"Largest code block",
			"Search in conversation",
			"Cancel",
		},
		Default: "Full conversation",
	}
	err := survey.AskOne(prompt, &choice, survey.WithStdio(os.Stdin, os.Stdout, os.Stderr))
	if err != nil {
		ui.Warningln("\nExport canceled.")
		return
	}

	var filename, content string

	switch choice {
	case "Full conversation":
		filename, content = c.Export("conversation", "")
	case "Last AI response":
		filename, content = c.Export("last_response", "")
	case "Largest code block":
		filename, content = c.Export("code_block", "")
	case "Search in conversation":
		var searchText string
		searchPrompt := &survey.Input{Message: "Enter text to search for:"}
		survey.AskOne(searchPrompt, &searchText, survey.WithStdio(os.Stdin, os.Stdout, os.Stderr))
		if searchText == "" {
			ui.Warningln("⚠️ Search text cannot be empty")
			return
		}
		filename, content = c.Export("search_conversation", searchText)
	default:
		ui.Warningln("💡 Export canceled.")
		return
	}

	if filename == "" || content == "" {
		ui.Errorln("❌ Nothing to export")
		return
	}

	fullPath := filepath.Join(cfg.ExportDir, filename)
	if err := os.MkdirAll(cfg.ExportDir, 0755); err != nil {
		ui.Errorln("❌ Cannot create export directory: %v", err)
		return
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		ui.Errorln("❌ Error saving file: %v", err)
		return
	}

	ui.AIln("✅ Saved to: %s", fullPath)
}

func (c *Chat) RefreshDynamicHeaders() error {
	ui.Warningln("⌛ Refreshing dynamic headers...")
	if dynamicHeaders, err := ExtractDynamicHeaders(); err == nil {
		c.FeSignals = dynamicHeaders.FeSignals
		c.FeVersion = dynamicHeaders.FeVersion
		c.VqdHash1 = dynamicHeaders.VqdHash1
		ui.AIln("✅ Successfully refreshed dynamic headers")
		return nil
	} else {
		ui.Warningln("⚠️ Failed to refresh dynamic headers: %v", err)
		return err
	}
}

func (c *Chat) ChangeModel(model models.Model) {
	c.Model = model
	setTerminalTitle(fmt.Sprintf("DuckDuckGo Chat - %s", model))
	ui.AIln("Model changed to %s", model)
}

func (c *Chat) AddContextMessage(content string) {
	c.Messages = append(c.Messages, Message{
		Role:    "user",
		Content: content,
	})
}
