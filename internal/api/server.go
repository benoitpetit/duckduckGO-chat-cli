package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"duckduckgo-chat-cli/internal/chat"
	"duckduckgo-chat-cli/internal/config"
	"duckduckgo-chat-cli/internal/ui"
)

var server *http.Server

// IsRunning checks if the API server is currently active.
func IsRunning() bool {
	return server != nil
}

// StartServer starts the API server in a new goroutine.
func StartServer(chatSession *chat.Chat, cfg *config.Config, port int) {
	if server != nil {
		ui.Warningln("API server is already running.")
		return
	}

	mux := http.NewServeMux()
	registerRoutes(mux, chatSession, cfg)

	server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		ui.Systemln("Starting API server on port %d", port)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			ui.Errorln("API server error: %v", err)
			server = nil
		}
	}()
}

// StopServer gracefully shuts down the API server.
func StopServer() {
	if server == nil {
		ui.Warningln("API server is not running.")
		return
	}

	ui.Systemln("Stopping API server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		ui.Errorln("API server shutdown error: %v", err)
	} else {
		ui.Systemln("API server stopped.")
	}
	server = nil
}

func registerRoutes(mux *http.ServeMux, chatSession *chat.Chat, cfg *config.Config) {
	mux.HandleFunc("/", documentationHandler)
	mux.HandleFunc("/chat", chatHandler(chatSession, cfg))
	mux.HandleFunc("/history", historyHandler(chatSession))
	// Add more routes here as needed
}

func documentationHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	doc := map[string]interface{}{
		"message": "DuckDuckGo Chat CLI API",
		"endpoints": []map[string]string{
			{
				"path":        "/",
				"method":      "GET",
				"description": "Shows this API documentation.",
			},
			{
				"path":        "/chat",
				"method":      "POST",
				"description": "Send a message to the chat and get the AI's response back.",
				"body":        `{"message": "your message here"}`,
			},
			{
				"path":        "/history",
				"method":      "GET",
				"description": "Retrieves the current chat session history.",
			},
		},
	}
	json.NewEncoder(w).Encode(doc)
}

func chatHandler(chatSession *chat.Chat, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
			return
		}

		var requestBody struct {
			Message string `json:"message"`
		}

		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if requestBody.Message == "" {
			http.Error(w, "Message cannot be empty", http.StatusBadRequest)
			return
		}

		// Process the input and get the response back
		response, err := chat.ProcessInputAndReturn(chatSession, requestBody.Message, cfg)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error processing chat: %v", err), http.StatusInternalServerError)
			return
		}

		// If logging is enabled, print to the console
		if cfg.API.LogRequests {
			ui.APILog("Received message: '%s'", requestBody.Message)
			// The full response can be long, so let's log a snippet
			responseSnippet := response
			if len(responseSnippet) > 100 {
				responseSnippet = responseSnippet[:100] + "..."
			}
			ui.APILog("Sending response snippet: '%s'", responseSnippet)
		}

		// Return the response to the API client
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"response": response,
		})
	}
}

func historyHandler(chatSession *chat.Chat) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chatSession.Messages)
	}
}
