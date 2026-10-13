package chat

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"duckduckgo-chat-cli/internal/analytics"
	"duckduckgo-chat-cli/internal/chatcontext"
	"duckduckgo-chat-cli/internal/config"
	"duckduckgo-chat-cli/internal/models"
	"duckduckgo-chat-cli/internal/persistence"
	"duckduckgo-chat-cli/internal/scrape"
)

func TestPrepareOneShotContextPreservesFileURLSearchOrder(t *testing.T) {
	firstFile := filepath.Join(t.TempDir(), "first.txt")
	secondFile := filepath.Join(t.TempDir(), "second.txt")
	if err := os.WriteFile(firstFile, []byte("file one"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secondFile, []byte("file two"), 0600); err != nil {
		t.Fatal(err)
	}
	var fetchedURLs, searched []string
	fetchURL := func(_ context.Context, raw string) (*scrape.WebContentResult, error) {
		fetchedURLs = append(fetchedURLs, raw)
		return &scrape.WebContentResult{URL: raw, Content: "page " + raw}, nil
	}
	search := func(_ context.Context, query string, _ *config.Config) ([]SearchResult, error) {
		searched = append(searched, query)
		return []SearchResult{{Title: query, FormattedUrl: "https://search.test/" + query}}, nil
	}
	prepared, err := prepareOneShotContext(context.Background(),
		[]string{firstFile, secondFile},
		[]string{"https://one.test", "https://two.test"},
		[]string{"first query", "second query"},
		&config.Config{Search: config.SearchConfig{IncludeSnippet: true}}, fetchURL, search)
	if err != nil {
		t.Fatalf("prepare one-shot context: %v", err)
	}
	wantParts := []string{
		"[File Context]\nFile: first.txt\n\nfile one",
		"[File Context]\nFile: second.txt\n\nfile two",
		"[URL Context]\nURL: https://one.test\n\npage https://one.test",
		"[URL Context]\nURL: https://two.test\n\npage https://two.test",
		"[Search Context]\nQuery: first query\n\n[Web Search Results: JSON]\n[\n  {\n    \"title\": \"first query\",\n    \"url\": \"https://search.test/first query\"\n  }\n]",
		"[Search Context]\nQuery: second query\n\n[Web Search Results: JSON]\n[\n  {\n    \"title\": \"second query\",\n    \"url\": \"https://search.test/second query\"\n  }\n]",
	}
	last := -1
	for _, want := range wantParts {
		position := strings.Index(prepared.Text, want)
		if position < 0 || position <= last {
			t.Errorf("context block %q is missing or out of order", want)
			continue
		}
		last = position
	}
	if !reflect.DeepEqual(fetchedURLs, []string{"https://one.test", "https://two.test"}) || !reflect.DeepEqual(searched, []string{"first query", "second query"}) {
		t.Fatalf("fetch order: URLs=%v searches=%v", fetchedURLs, searched)
	}
	if len(prepared.Sources) != 6 || prepared.Sources[0].Kind != chatcontext.SourceKindFile || prepared.Sources[1].Name != "second.txt" || prepared.Sources[2].Kind != chatcontext.SourceKindURL || prepared.Sources[4].Kind != chatcontext.SourceKindSearch {
		t.Fatalf("source order = %+v", prepared.Sources)
	}
}

func TestPrepareOneShotContextSupportsImages(t *testing.T) {
	path := filepath.Join(t.TempDir(), "input.png")
	var imageData bytes.Buffer
	if err := png.Encode(&imageData, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, imageData.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	prepared, err := prepareOneShotContext(context.Background(), []string{path}, nil, nil, &config.Config{}, nil, nil)
	if err != nil {
		t.Fatalf("prepare image: %v", err)
	}
	if len(prepared.Images) != 1 || prepared.Images[0].Name != "input.png" || prepared.Images[0].MIMEType != "image/webp" {
		t.Fatalf("images = %+v, want one normalized image", prepared.Images)
	}
	if !strings.Contains(prepared.Text, "[Image Context]\nFile: input.png") || len(prepared.Sources) != 1 || prepared.Sources[0].Kind != chatcontext.SourceKindImage {
		t.Fatalf("image context = %+v", prepared)
	}
}

func TestPrepareOneShotContextRejectsAggregateLimitWithoutPartialResult(t *testing.T) {
	first := filepath.Join(t.TempDir(), "first.txt")
	second := filepath.Join(t.TempDir(), "second.txt")
	if err := os.WriteFile(first, []byte(strings.Repeat("a", maxLibraryContextSize/2)), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte(strings.Repeat("b", maxLibraryContextSize/2)), 0600); err != nil {
		t.Fatal(err)
	}
	prepared, err := prepareOneShotContext(context.Background(), []string{first, second}, nil, nil, &config.Config{}, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "aggregate") {
		t.Fatalf("aggregate error = %v, want context limit error", err)
	}
	if prepared.Text != "" || len(prepared.Images) != 0 || len(prepared.Sources) != 0 {
		t.Fatalf("failed preparation returned partial context: %+v", prepared)
	}
}

func TestPrepareOneShotContextReturnsURLAndSearchErrorsWithoutPartialResult(t *testing.T) {
	tests := []struct {
		name     string
		urls     []string
		searches []string
		fetch    func(context.Context, string) (*scrape.WebContentResult, error)
		search   func(context.Context, string, *config.Config) ([]SearchResult, error)
	}{
		{
			name: "URL failure", urls: []string{"https://unavailable.test"},
			fetch: func(context.Context, string) (*scrape.WebContentResult, error) { return nil, errors.New("offline") },
			search: func(context.Context, string, *config.Config) ([]SearchResult, error) {
				return nil, errors.New("unused")
			},
		},
		{
			name: "search failure", searches: []string{"query"},
			fetch: func(context.Context, string) (*scrape.WebContentResult, error) { return nil, errors.New("unused") },
			search: func(context.Context, string, *config.Config) ([]SearchResult, error) {
				return nil, errors.New("offline")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prepared, err := prepareOneShotContext(context.Background(), nil, test.urls, test.searches, &config.Config{}, test.fetch, test.search)
			if err == nil {
				t.Fatal("preparation succeeded, want source error")
			}
			if prepared.Text != "" || len(prepared.Images) != 0 || len(prepared.Sources) != 0 {
				t.Fatalf("failed preparation returned partial result: %+v", prepared)
			}
		})
	}
}

func TestProcessOneShotContextPreservesSourcesAndSavesHistory(t *testing.T) {
	manager := persistence.NewHistoryManager(t.TempDir())
	tracker := analytics.NewChatAnalytics()
	c := &Chat{SessionID: "one-shot-context", Model: models.Default(), HistoryManager: manager, Analytics: tracker}
	contextText := "[File Context]\nFile: notes.txt\n\nreference facts"
	sources := []chatcontext.SourceRef{{Kind: chatcontext.SourceKindFile, Name: "notes.txt", URI: "/tmp/notes.txt", SizeBytes: 14, State: chatcontext.SourceStateActive}}
	fetch := func(_ context.Context, message string) (<-chan string, <-chan error, error) {
		if !strings.Contains(message, contextText) || !strings.Contains(message, "answer from these facts") {
			t.Errorf("chat request omitted prepared context or prompt: %q", message)
		}
		stream := make(chan string, 1)
		stream <- "answer"
		close(stream)
		return stream, nil, nil
	}
	response, err := processInputContextWithFetcher(context.Background(), c, contextText, "answer from these facts", &config.Config{}, sources, fetch)
	if err != nil || response != "answer" {
		t.Fatalf("process result = %q, %v", response, err)
	}
	if len(c.Messages) != 2 || len(c.Messages[0].Sources) != 1 || c.Messages[0].Sources[0].Kind != chatcontext.SourceKindFile {
		t.Fatalf("messages did not retain prepared source: %+v", c.Messages)
	}
	if snapshot := tracker.Snapshot(); snapshot.UserTokensEstimate == 0 || snapshot.ContextTokensEstimate == 0 {
		t.Fatalf("context and prompt token estimates not separated: %+v", snapshot)
	}
	if err := c.SaveCurrentSession(); err != nil {
		t.Fatalf("save one-shot history: %v", err)
	}
	saved, err := manager.LoadSession(c.SessionID)
	if err != nil {
		t.Fatalf("load saved one-shot history: %v", err)
	}
	if len(saved.Messages) != 2 || len(saved.Messages[0].Sources) != 1 || saved.Messages[0].Sources[0].URI != "/tmp/notes.txt" {
		t.Fatalf("saved messages = %+v, want answer and original source reference", saved.Messages)
	}
}
