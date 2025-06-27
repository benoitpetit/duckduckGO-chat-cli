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

	color.Yellow("⌛ Attempting to extract dynamic headers from DuckDuckGo...")
	if dynamicHeaders, err := ExtractDynamicHeaders(); err == nil {
		feSignals = dynamicHeaders.FeSignals
		feVersion = dynamicHeaders.FeVersion
		vqdHash1 = dynamicHeaders.VqdHash1
		color.Green("✅ Successfully extracted dynamic headers")
	} else {
		color.Yellow("⚠️ Failed to get dynamic headers, falling back to placeholders. Error: %v", err)
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
		color.Red("Error fetching VQD: %v", err)
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
				color.Yellow("🔄 Error 418 detected, refreshing headers...")
				c.RefreshDynamicHeaders() // Try refreshing headers on first 418 error
			}

			if c.NewVqd != "" && c.RetryCount < 3 {
				c.RetryCount++
				color.Yellow("Retrying request (attempt %d/3)...", c.RetryCount)
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

	if content == nil || content.Content == "" {
		return fmt.Errorf("no content found at URL: %s", url)
	}

	contentLength := len(content.Content)
	if contentLength > 500 {
		color.Cyan("Retrieved %d characters of content", contentLength)
	}

	c.Messages = append(c.Messages, Message{
		Role:    "user",
		Content: fmt.Sprintf("[URL Context]\nURL: %s\n\n%s", url, content.Content),
	})

	return nil
}

func PrintCommands() {
	color.Yellow("Type /help to show these commands again")
}

func PrintWelcomeMessage() {
	color.Yellow("Special commands:")
	color.White("/search <query> [-- request] - Search and add context, optionally make a request about results")
	color.White("/file <path> [-- request] - Add file content and optionally make a request about it")
	color.White("/library [list|add <path>|remove <n>|search <pattern>|load <lib>] - Manage library directories")
	color.White("/url <url> [-- request] - Add webpage content and optionally make a request about it")
	color.White("/pmp [path] [options] [-- request] - Generate structured project prompts with PMP")
	color.White("/clear - Clear context")
	color.White("/history - Show history")
	color.White("/export - Export messages")
	color.White("/copy - Copy to clipboard")
	color.White("/config - Configure settings")
	color.White("/model - Change AI model")
	color.White("/version - Show version info")
	color.White("/help - Show this menu")
	color.White("/exit - Quit")
}

func HandleURLCommand(c *Chat, input string, cfg *config.Config) {
	// Parse the command: /url <URL> -- <request>
	commandInput := strings.TrimPrefix(input, "/url ")

	var url, userRequest string

	// Check if there's a -- separator
	if strings.Contains(commandInput, " -- ") {
		parts := strings.SplitN(commandInput, " -- ", 2)
		url = strings.TrimSpace(parts[0])
		if len(parts) > 1 {
			userRequest = strings.TrimSpace(parts[1])
		}
	} else {
		// Fallback: if no --, treat everything as URL for backward compatibility
		url = strings.TrimSpace(commandInput)
	}

	if url == "" {
		color.Red("Usage: /url <URL> [-- request]")
		return
	}

	color.Yellow("Scraping URL: %s (this may take a few seconds...)", url)

	// Add URL context first
	if err := c.AddURLContext(url); err != nil {
		color.Red("URL error: %v", err)
		return
	}

	color.Green("Successfully retrieved webpage content from: %s", url)

	// If user provided a specific request, process it with the URL context
	if userRequest != "" {
		color.Cyan("Processing your request about the webpage...")
		ProcessInput(c, userRequest, cfg)
	} else {
		color.Yellow("Webpage content added to context. You can now ask questions about it.")
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

func (c *Chat) RefreshDynamicHeaders() error {
	color.Yellow("🔄 Refreshing dynamic headers...")

	if dynamicHeaders, err := ExtractDynamicHeaders(); err == nil {
		c.FeSignals = dynamicHeaders.FeSignals
		c.FeVersion = dynamicHeaders.FeVersion
		c.VqdHash1 = dynamicHeaders.VqdHash1
		color.Green("✅ Headers refreshed successfully")
		return nil
	} else {
		color.Yellow("⚠️ Failed to refresh headers: %v", err)
		return err
	}
}

func (c *Chat) ChangeModel(model models.Model) {
	c.Model = model
	displayName := shortenModelName(string(model))
	setTerminalTitle(fmt.Sprintf("DuckDuckGo Chat - %s", displayName))

	// Refresh headers when changing model as they might be model-specific
	if err := c.RefreshDynamicHeaders(); err != nil {
		color.Yellow("⚠️ Continuing with existing headers after refresh failure")
	}

	// Also refresh VQD token
	c.NewVqd = GetVQD()
	c.OldVqd = c.NewVqd
	c.RetryCount = 0

	color.Green("Model changed to: %s", displayName)
}
