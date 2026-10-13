package chat

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"duckduckgo-chat-cli/internal/chatcontext"
	"duckduckgo-chat-cli/internal/config"
	"duckduckgo-chat-cli/internal/media"
	"duckduckgo-chat-cli/internal/scrape"
)

// OneShotContext contains every source prepared for one headless chat turn.
// The fields are staged together so a failed import cannot partially mutate a Chat.
type OneShotContext struct {
	Text    string
	Images  []media.ImageAttachment
	Sources []chatcontext.SourceRef
}

type oneShotURLFetcher func(context.Context, string) (*scrape.WebContentResult, error)
type oneShotSearchFetcher func(context.Context, string, *config.Config) ([]SearchResult, error)

// PrepareOneShotContext loads local files, web pages, then searches in that
// order. It enforces the same aggregate context budget used by library imports.
func PrepareOneShotContext(ctx context.Context, files, urls, searches []string, cfg *config.Config) (OneShotContext, error) {
	return prepareOneShotContext(ctx, files, urls, searches, cfg, scrape.WebContentContext, func(ctx context.Context, query string, cfg *config.Config) ([]SearchResult, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		results, err := performSearch(query, cfg.Search.MaxResults, cfg.Search.MaxRetries, cfg.Search.RetryDelay)
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return results, nil
	})
}

func prepareOneShotContext(ctx context.Context, files, urls, searches []string, cfg *config.Config, fetchURL oneShotURLFetcher, search oneShotSearchFetcher) (OneShotContext, error) {
	if cfg == nil {
		cfg = &config.Config{}
	}
	if fetchURL == nil {
		fetchURL = scrape.WebContentContext
	}
	if search == nil {
		search = func(ctx context.Context, query string, cfg *config.Config) ([]SearchResult, error) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			return performSearch(query, cfg.Search.MaxResults, cfg.Search.MaxRetries, cfg.Search.RetryDelay)
		}
	}

	var prepared OneShotContext
	var blocks []string
	textBytes := 0
	appendSource := func(source chatcontext.SourceRef) {
		prepared.Sources = append(prepared.Sources, source)
	}
	appendBlock := func(block string, source chatcontext.SourceRef) error {
		separatorBytes := 0
		if len(blocks) > 0 {
			separatorBytes = 2
		}
		if len(block)+separatorBytes > maxLibraryContextSize-textBytes {
			return fmt.Errorf("one-shot context exceeds the %d MiB aggregate limit", maxLibraryContextSize/(1<<20))
		}
		blocks = append(blocks, block)
		textBytes += len(block) + separatorBytes
		appendSource(source)
		return nil
	}
	checkContext := func() error { return ctx.Err() }

	for _, path := range files {
		if err := checkContext(); err != nil {
			return OneShotContext{}, err
		}
		if isSupportedImageExtension(path) {
			image, err := media.ReadImage(path)
			if err != nil {
				return OneShotContext{}, fmt.Errorf("prepare image %q: %w", path, err)
			}
			image.Name = filepath.Base(path)
			block := fmt.Sprintf("[Image Context]\nFile: %s", image.Name)
			source := chatcontext.SourceRef{Kind: chatcontext.SourceKindImage, Name: image.Name, URI: path, SizeBytes: len(image.Data), State: chatcontext.SourceStateActive}
			if err := appendBlock(block, source); err != nil {
				return OneShotContext{}, err
			}
			prepared.Images = append(prepared.Images, image)
			continue
		}
		content, err := readContextFile(path)
		if err != nil {
			return OneShotContext{}, fmt.Errorf("prepare file %q: %w", path, err)
		}
		name := filepath.Base(path)
		block := fmt.Sprintf("[File Context]\nFile: %s\n\n%s", name, string(content))
		source := chatcontext.SourceRef{Kind: chatcontext.SourceKindFile, Name: name, URI: path, SizeBytes: len(content), State: chatcontext.SourceStateActive}
		if err := appendBlock(block, source); err != nil {
			return OneShotContext{}, err
		}
	}

	for _, rawURL := range urls {
		if err := checkContext(); err != nil {
			return OneShotContext{}, err
		}
		result, err := fetchURL(ctx, rawURL)
		if err != nil {
			return OneShotContext{}, fmt.Errorf("prepare URL %q: %w", rawURL, err)
		}
		if result == nil || strings.TrimSpace(result.Content) == "" {
			return OneShotContext{}, fmt.Errorf("prepare URL %q: no readable page content", rawURL)
		}
		if err := checkContext(); err != nil {
			return OneShotContext{}, err
		}
		resolvedURL := result.URL
		if resolvedURL == "" {
			resolvedURL = rawURL
		}
		block := fmt.Sprintf("[URL Context]\nURL: %s\n\n%s", resolvedURL, result.Content)
		source := chatcontext.SourceRef{Kind: chatcontext.SourceKindURL, Name: resolvedURL, URI: resolvedURL, SizeBytes: len(result.Content), State: chatcontext.SourceStateActive}
		if err := appendBlock(block, source); err != nil {
			return OneShotContext{}, err
		}
	}

	for _, query := range searches {
		if err := checkContext(); err != nil {
			return OneShotContext{}, err
		}
		results, err := search(ctx, query, cfg)
		if err != nil {
			return OneShotContext{}, fmt.Errorf("prepare search %q: %w", query, err)
		}
		if len(results) == 0 {
			return OneShotContext{}, fmt.Errorf("prepare search %q: no results found", query)
		}
		if err := checkContext(); err != nil {
			return OneShotContext{}, err
		}
		formatted := formatSearchResults(results, cfg.Search.IncludeSnippet)
		block := fmt.Sprintf("[Search Context]\nQuery: %s\n\n%s", query, formatted)
		sources := searchSourceRefs(query, results)
		if err := appendBlock(block, sources[0]); err != nil {
			return OneShotContext{}, err
		}
		for _, source := range sources[1:] {
			appendSource(source)
		}
	}

	prepared.Text = strings.Join(blocks, "\n\n")
	return prepared, nil
}

// ProcessInputContextWithContext submits prepared text and source references
// through the same chat pipeline as other callers. Images should be queued on c
// before this call so they are consumed atomically with the prompt.
func ProcessInputContextWithContext(ctx context.Context, c *Chat, contextText, prompt string, cfg *config.Config, sources []chatcontext.SourceRef) (string, error) {
	if c == nil {
		return "", fmt.Errorf("chat session is required")
	}
	return processInputContextWithFetcher(ctx, c, contextText, prompt, cfg, sources, c.FetchStreamWithErrors)
}

func processInputContextWithFetcher(ctx context.Context, c *Chat, contextText, prompt string, cfg *config.Config, sources []chatcontext.SourceRef, fetch func(context.Context, string) (<-chan string, <-chan error, error)) (string, error) {
	if c == nil {
		return "", fmt.Errorf("chat session is required")
	}
	if strings.TrimSpace(prompt) == "" {
		return "", fmt.Errorf("prompt cannot be empty")
	}
	c.LastOptimizationChanges = nil
	c.AddPendingSources(sources)
	input := prompt
	if contextText != "" {
		input = contextText + "\n\n" + prompt
	}
	return processInputWithStreamFetcherAndTokenRoles(ctx, c, input, cfg, renderStreamToString, fetch, len(prompt), len(contextText))
}
