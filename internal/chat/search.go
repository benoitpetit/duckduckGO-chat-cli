package chat

import (
	"fmt"
	"strings"
	"time"

	"duckduckgo-chat-cli/internal/config"

	"github.com/fatih/color"
	"github.com/sap-nocops/duckduckgogo/client"
)

type SearchResult struct {
	Title        string
	FormattedUrl string
	Snippet      string
}

func HandleSearchCommand(c *Chat, input string, cfg *config.Config) {
	query := strings.TrimPrefix(input, "/search ")
	if query == "" {
		color.Red("Search query cannot be empty")
		return
	}

	color.Yellow("🔍 Searching for: %s (this may take a few seconds...)", query)
	results, err := performSearch(query, cfg.Search.MaxResults)
	if err != nil {
		color.Red("Search error: %v", err)
		return
	}

	if len(results) == 0 {
		color.Red("No results found for: %s", query)
		return
	}

	contextMsg := formatSearchResults(results, cfg.Search.IncludeSnippet)
	c.Messages = append(c.Messages, Message{
		Role:    "user",
		Content: fmt.Sprintf("[Search Context]\n%s", contextMsg),
	})

	color.Green("Added %d search results to the context", len(results))
}

func performSearch(query string, maxResults int) ([]SearchResult, error) {
	if maxResults <= 0 {
		maxResults = 5
	}

	ddg := client.NewDuckDuckGoSearchClient()

	// Implement retry logic
	maxRetries := 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			sleepDuration := time.Duration(1<<attempt) * time.Second
			color.Yellow("Retrying search in %v... (attempt %d/%d)", sleepDuration, attempt+1, maxRetries)
			time.Sleep(sleepDuration)
		}

		results, err := ddg.SearchLimited(query, maxResults)
		if err != nil {
			lastErr = err
			// Check if it's a 202 status or other temporary error
			if strings.Contains(err.Error(), "202") {
				continue // Retry on 202
			}
			return nil, fmt.Errorf("search failed: %v", err)
		}

		// Process valid results
		searchResults := make([]SearchResult, 0, len(results))
		for _, r := range results {
			if r.Title != "" && r.FormattedUrl != "" {
				searchResults = append(searchResults, SearchResult{
					Title:        r.Title,
					FormattedUrl: r.FormattedUrl,
					Snippet:      r.Snippet,
				})
			}
		}

		if len(searchResults) > 0 {
			return searchResults, nil
		}

		// If we got no results but no error, wait and retry
		lastErr = fmt.Errorf("no results found")
		continue
	}

	return nil, fmt.Errorf("search failed after %d attempts: %v", maxRetries, lastErr)
}

func formatSearchResults(results []SearchResult, includeSnippet bool) string {
	var sb strings.Builder
	for _, res := range results {
		sb.WriteString(fmt.Sprintf("▸ %s\n  %s", res.Title, res.FormattedUrl))
		if includeSnippet && res.Snippet != "" {
			sb.WriteString(fmt.Sprintf("\n  %s", res.Snippet))
		}
		sb.WriteString("\n\n")
	}
	return strings.TrimSpace(sb.String())
}
