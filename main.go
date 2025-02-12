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
	"context"
	"time"
	"github.com/chromedp/chromedp"

	"github.com/fatih/color"
	"github.com/sap-nocops/duckduckgogo/client"
	"runtime"
	"os/exec"
)

// Constants for API endpoints and headers
const (
	statusURL         = "https://duckduckgo.com/duckchat/v1/status"
	chatURL           = "https://duckduckgo.com/duckchat/v1/chat"
	statusHeaders     = "1"
	termsOfServiceURL = "https://duckduckgo.com/aichat/privacy-terms"
	maxSearchResults  = 10
)

// Model represents the AI model used for chat
type Model string

// ModelAlias represents a user-friendly alias for the AI model
type ModelAlias string

// Define available models and their aliases
const (
	GPT4Mini Model = "gpt-4o-mini"
	Claude3  Model = "claude-3-haiku-20240307"
	Llama    Model = "meta-llama/Llama-3.3-70B-Instruct-Turbo"
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

type SearchResult struct {
	Title        string
	FormattedUrl string
	Snippet      string
}

// Add new methods to Chat struct
func (c *Chat) AddSearchContext(query string) error {
	results, err := performSearch(query)
    if err != nil || len(results) == 0 {
        return fmt.Errorf("Aucun résultat pour '%s' (erreur: %v)", query, err)
    }

    contextMsg := formatSearchResults(results)
    c.Messages = append(c.Messages, Message{
        Role:    "user",
        Content: fmt.Sprintf("[Search Context]\n%s", contextMsg),
    })

    return nil
}

func performSearch(query string) ([]SearchResult, error) {

	ddg := client.NewDuckDuckGoSearchClient()
    results, err := ddg.SearchLimited(query, maxSearchResults)
    if err != nil {
        return nil, fmt.Errorf("erreur de recherche: %v", err)
    }

    searchResults := make([]SearchResult, 0, len(results))
    for _, r := range results {
        searchResults = append(searchResults, SearchResult{
            Title:        r.Title,
            FormattedUrl: r.FormattedUrl,
            Snippet:      r.Snippet,
        })
    }

    return searchResults, nil
}

func formatSearchResults(results []SearchResult) string {
	var sb strings.Builder
	for i, res := range results {
		sb.WriteString(fmt.Sprintf("▸ %s\n  %s\n  %s\n\n", res.Title, res.Snippet, res.FormattedUrl))
		if i >= maxSearchResults-1 {
			break
		}
	}
	return strings.TrimSpace(sb.String())
}

// Nouvelle fonction pour nettoyer le terminal
func clearTerminal() {
	switch runtime.GOOS {
	case "linux", "darwin":
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	case "windows":
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}

// Clear method to reset VQD and messages
func (c *Chat) Clear() {
	// Clear terminal first
	clearTerminal()
	
	// Reset messages
	c.Messages = []Message{}
	
	// Reset VQD
	req, err := http.NewRequest("GET", statusURL, nil)
	if err != nil {
		color.Red("Error resetting VQD: %v", err)
		return
	}

	req.Header.Set("x-vqd-accept", statusHeaders)
	resp, err := c.Client.Do(req)
	if err != nil {
		color.Red("Error resetting VQD: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		color.Red("Error resetting VQD: %s", resp.Status)
		return
	}

	newVqd := resp.Header.Get("x-vqd-4")
	if newVqd == "" {
		color.Red("Error: VQD not found in response")
		return
	}

	c.OldVqd = newVqd
	c.NewVqd = newVqd
	
	color.Yellow("Conversation context cleared and VQD reset successfully.")
	
	// Afficher les commandes disponibles
	color.Yellow("Special commands:\n/search <query> - Add search context\n/file <path> - Add file content to context\n/url <url> - Add webpage content to context\n/clear - Clear conversation context\n/history - Show conversation history\n/markdown - Export conversation to markdown\n/extract - Extract last AI message\n/model - Change AI model\n/exit - Quit")
}

// Add new method to Chat struct
func (c *Chat) AddFileContext(filepath string) error {
    content, err := os.ReadFile(filepath)
    if err != nil {
		return fmt.Errorf("error reading file: %v", err)
    }

    // Add file content to messages
    c.Messages = append(c.Messages, Message{
        Role:    "user",
        Content: fmt.Sprintf("[File Context]\nFile: %s\n\n%s", filepath, string(content)),
    })

    return nil
}

// Add new method to Chat struct
func (c *Chat) AddURLContext(url string) error {
	color.Yellow("Retrieving webpage content (this may take a few seconds)...")
    content, err := scrapeURL(url)
    if err != nil {
        return fmt.Errorf("erreur lors de la lecture de l'URL: %v", err)
    }

    if content == "" {
        return fmt.Errorf("aucun contenu trouvé sur la page")
    }

    c.Messages = append(c.Messages, Message{
        Role:    "user",
        Content: fmt.Sprintf("[URL Context]\nURL: %s\n\n%s", url, content),
    })

    return nil
}

// scrapeURL performs text extraction from a webpage with JavaScript support
func scrapeURL(url string) (string, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    opts := append(chromedp.DefaultExecAllocatorOptions[:],
        chromedp.DisableGPU,
        chromedp.NoSandbox,
        chromedp.Headless,
    )

    allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
    defer cancel()

    taskCtx, cancel := chromedp.NewContext(allocCtx)
    defer cancel()

    var textContent string
    err := chromedp.Run(taskCtx,
        chromedp.Navigate(url),
        chromedp.Sleep(2*time.Second),
        chromedp.EvaluateAsDevTools(`
            (function() {
                // Supprimer les éléments non pertinents
                const elementsToRemove = document.querySelectorAll('script, style, noscript, iframe, img');
                elementsToRemove.forEach(el => el.remove());

                // Récupérer le texte de manière plus fiable
                const textNodes = [];
                const walk = document.createTreeWalker(
                    document.body,
                    NodeFilter.SHOW_TEXT,
                    null,
                    false
                );

                let node;
                while (node = walk.nextNode()) {
                    const text = node.textContent.trim();
                    if (text && text.length > 0) {
                        textNodes.push(text);
                    }
                }

                return textNodes
                    .filter(text => text && text.length > 1) // Filtrer les textes vides ou trop courts
                    .join('\n');
            })()
        `, &textContent),
    )

    if err != nil {
        return "", fmt.Errorf("erreur lors du scraping: %v", err)
    }

    if textContent == "" {
        // Essayer une méthode alternative si aucun contenu n'est trouvé
        err = chromedp.Run(taskCtx,
            chromedp.EvaluateAsDevTools(`
                document.body.innerText || document.documentElement.innerText || "Pas de contenu trouvé"
            `, &textContent),
        )
        if err != nil {
            return "", fmt.Errorf("erreur lors de la récupération alternative: %v", err)
        }
    }

    // Nettoyer le texte
    lines := strings.Split(textContent, "\n")
    var cleanedLines []string
    seen := make(map[string]bool)

    for _, line := range lines {
        line = strings.TrimSpace(line)
        if line != "" && !seen[line] && len(line) > 1 {
            cleanedLines = append(cleanedLines, line)
            seen[line] = true
        }
    }

    result := strings.Join(cleanedLines, "\n")
    if result == "" {
        return "", fmt.Errorf("aucun contenu textuel trouvé sur la page")
    }

    return result, nil
}

// Nouvelle fonction pour exporter en markdown
func (c *Chat) exportToMarkdown() string {
    var md strings.Builder
    
    // Écrire l'en-tête
    md.WriteString("# DuckDuckGo AI Chat Conversation\n\n")
    md.WriteString("## Conversation History\n\n")
    
    for _, msg := range c.Messages {
        switch {
        case strings.Contains(msg.Content, "[Search Context]"):
            content := strings.TrimPrefix(msg.Content, "[Search Context]\n")
            md.WriteString("### Search Context\n\n")
            md.WriteString("```\n")
            md.WriteString(content)
            md.WriteString("\n```\n\n")
            
        case strings.Contains(msg.Content, "[File Context]"):
            content := strings.TrimPrefix(msg.Content, "[File Context]\n")
            md.WriteString("### File Context\n\n")
            md.WriteString("```\n")
            md.WriteString(content)
            md.WriteString("\n```\n\n")
            
        case strings.Contains(msg.Content, "[URL Context]"):
            content := strings.TrimPrefix(msg.Content, "[URL Context]\n")
            md.WriteString("### URL Context\n\n")
            md.WriteString("```\n")
            md.WriteString(content)
            md.WriteString("\n```\n\n")
            
        default:
            if msg.Role == "user" {
                md.WriteString("### User\n\n")
                md.WriteString(msg.Content)
                md.WriteString("\n\n")
            } else {
                md.WriteString("### Assistant\n\n")
                md.WriteString(msg.Content)
                md.WriteString("\n\n")
            }
        }
    }
    
    return md.String()
}

// Nouvelle méthode pour extraire le dernier message
func (c *Chat) extractLastMessage() (string, string) {
    // Trouver le dernier message de l'assistant
    var lastMessage Message
    for i := len(c.Messages) - 1; i >= 0; i-- {
        if c.Messages[i].Role == "assistant" {
            lastMessage = c.Messages[i]
            break
        }
    }

    if lastMessage.Content == "" {
        return "", ""
    }

    // Générer un nom de fichier basé sur le contenu
    words := strings.Fields(lastMessage.Content)
    var title string
    if len(words) > 5 {
        title = strings.Join(words[:5], "_")
    } else {
        title = strings.Join(words, "_")
    }
    
    // Nettoyer le titre pour le nom de fichier
    title = strings.Map(func(r rune) rune {
        if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
            return r
        }
        return '_'
    }, title)
    
    // Limiter la longueur du titre
    if len(title) > 50 {
        title = title[:50]
    }
    
    filename := fmt.Sprintf("%s_%s.md", title, time.Now().Format("20060102"))
    
    // Créer le contenu markdown
    var md strings.Builder
    md.WriteString("# Message extrait de DuckDuckGo AI Chat\n\n")
    md.WriteString(lastMessage.Content)
    md.WriteString("\n")
    
    return filename, md.String()
}

func (c *Chat) ChangeModel(model Model) {
	c.Model = model
	color.Green("Model changed to: %s", model)
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
	color.Yellow("Special commands:\n/search <query> - Add search context\n/file <path> - Add file content to context\n/url <url> - Add webpage content to context\n/clear - Clear conversation context\n/history - Show conversation history\n/markdown - Export conversation to markdown\n/extract - Extract last AI message\n/model - Change AI model\n/exit - Quit")


	reader := bufio.NewReader(os.Stdin)
	for {
		color.Blue("You: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch {
		case input == "/exit":
			color.Yellow("Exiting chat. Goodbye!")
			return

		case input == "/clear":
			chat.Clear()
			continue

		case input == "/history":
			color.Magenta("\n=== Conversation History ===")
			for _, msg := range chat.Messages {
				prefix := "You: "
				if msg.Role == "assistant" {
					prefix = "AI:  "
					color.Cyan("%s%s", prefix, msg.Content)
				} else {
					switch {
					case strings.Contains(msg.Content, "[Search Context]"):
						color.Yellow("%s%s", "CTX: ", strings.ReplaceAll(msg.Content, "[Search Context]\n", ""))
					case strings.Contains(msg.Content, "[File Context]"):
						color.Yellow("%s%s", "FILE: ", strings.ReplaceAll(msg.Content, "[File Context]\n", ""))
					case strings.Contains(msg.Content, "[URL Context]"):
						color.Yellow("%s%s", "URL: ", strings.ReplaceAll(msg.Content, "[URL Context]\n", ""))
					default:
						color.Blue("%s%s", prefix, msg.Content)
					}
				}
			}
			fmt.Println()
			continue

		case strings.HasPrefix(input, "/search "):
			query := strings.TrimPrefix(input, "/search ")
			if err := chat.AddSearchContext(query); err != nil {
				color.Red("Search error: %v", err)
			} else {
				color.Cyan("Added search context to conversation")
			}
			continue

		case strings.HasPrefix(input, "/file "):
            filepath := strings.TrimPrefix(input, "/file ")
            if err := chat.AddFileContext(filepath); err != nil {
                color.Red("File error: %v", err)
            } else {
                color.Cyan("Added file context to conversation")
            }
            continue

		case strings.HasPrefix(input, "/url "):
            url := strings.TrimPrefix(input, "/url ")
            if err := chat.AddURLContext(url); err != nil {
                color.Red("URL error: %v", err)
            } else {
                color.Cyan("Added URL content to conversation")
            }
            continue

		case input == "/markdown":
            markdown := chat.exportToMarkdown()
            filename := fmt.Sprintf("chat_export_%s.md", time.Now().Format("20060102_150405"))
            err := os.WriteFile(filename, []byte(markdown), 0644)
            if err != nil {
                color.Red("Error exporting to markdown: %v", err)
            } else {
                color.Green("Conversation exported to %s", filename)
            }
            continue

		case input == "/extract":
            filename, content := chat.extractLastMessage()
            if filename == "" || content == "" {
                color.Red("No AI message found to extract")
                continue
            }
            
            err := os.WriteFile(filename, []byte(content), 0644)
            if err != nil {
                color.Red("Error extracting message: %v", err)
            } else {
                color.Green("Last message extracted to %s", filename)
            }
            continue

		case input == "/model":
			newModelAlias := chooseModel()
			chat.ChangeModel(modelMap[newModelAlias])
			continue

		default:
			stream, err := chat.FetchStream(input)
			if err != nil {
				color.Red("Error: %v", err)
				continue
			}

			color.Green("AI: ")
			printResponse(stream)
		}
	}
}

// chooseModel prompts the user to select an AI model
func chooseModel() ModelAlias {
	color.Yellow("Please choose an AI model:")
	color.White("1. GPT-4o mini")
	color.White("2. Claude 3 Haiku")
	color.White("3. Llama 3.3 70B")
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
