package chat

import (
	"context"
	"strings"
	"testing"

	"duckduckgo-chat-cli/internal/chatcontext"
	"duckduckgo-chat-cli/internal/config"
	"duckduckgo-chat-cli/internal/intelligence"
	"duckduckgo-chat-cli/internal/ui"
)

func TestContextEstimateTokensUsesApproximateCeiling(t *testing.T) {
	for _, test := range []struct{ bytes, want int }{{0, 0}, {1, 1}, {4, 1}, {5, 2}, {9, 3}, {-2, 0}} {
		if got := contextEstimateTokens(test.bytes); got != test.want {
			t.Errorf("contextEstimateTokens(%d) = %d, want %d", test.bytes, got, test.want)
		}
	}
}

func TestContextCommandShowsSourceCountsAndApproximateSizesWithoutPrivatePaths(t *testing.T) {
	c := &Chat{Messages: []Message{
		{Role: "user", Content: "secret body", Sources: []chatcontext.SourceRef{{ID: "source-1", Kind: chatcontext.SourceKindFile, Name: "notes.txt", URI: "/home/alice/private/notes.txt", SizeBytes: 101, State: chatcontext.SourceStateCompressed}}},
		{Role: "user", Content: "page body", Sources: []chatcontext.SourceRef{{ID: "source-2", Kind: chatcontext.SourceKindURL, Name: "https://example.test/guide", URI: "https://example.test/guide", SizeBytes: 40, State: chatcontext.SourceStateTruncated}}},
	}}
	title, lines := captureSourcePanel(t, func() { HandleContextCommand(c) })
	joined := strings.Join(lines, "\n")
	for _, want := range []string{"Context", "2 sources", "file", "URL", "compressed", "truncated", "approx", "notes.txt"} {
		if !strings.Contains(strings.ToLower(title+"\n"+joined), strings.ToLower(want)) {
			t.Errorf("/context panel lacks %q: %s\n%s", want, title, joined)
		}
	}
	if strings.Contains(joined, "/home/alice/private") || strings.Contains(joined, "secret body") || strings.Contains(joined, "page body") {
		t.Fatalf("/context exposed private path or source body: %s", joined)
	}
}

func TestSourcesCommandShowsRetainedMultiTurnContextAndLatestOptimizationChanges(t *testing.T) {
	c := &Chat{
		Messages: []Message{
			{Role: "user", Content: "[File Context]\nold body", Sources: []chatcontext.SourceRef{{ID: "source-1", Kind: chatcontext.SourceKindFile, Name: "older.txt", URI: "/private/older.txt", SizeBytes: 800, State: chatcontext.SourceStateCompressed}}},
			{Role: "user", Content: "first question"},
			{Role: "assistant", Content: "first answer"},
			{Role: "user", Content: "[Search Context]\nnew body", Sources: []chatcontext.SourceRef{{ID: "source-2", Kind: chatcontext.SourceKindSearch, Name: "current result", URI: "https://example.test", SizeBytes: 120, State: chatcontext.SourceStateActive}}},
			{Role: "user", Content: "latest question", Sources: []chatcontext.SourceRef{{ID: "source-2", Kind: chatcontext.SourceKindSearch, Name: "current result", URI: "https://example.test", SizeBytes: 120, State: chatcontext.SourceStateActive}}},
			{Role: "assistant", Content: "latest answer"},
		},
		LastOptimizationChanges: []intelligence.OptimizationChange{{SourceID: "source-3", State: chatcontext.SourceStateOmitted, BytesBefore: 250, BytesAfter: 0}},
	}
	title, lines := captureSourcePanel(t, func() { HandleSourcesCommand(c) })
	joined := title + "\n" + strings.Join(lines, "\n")
	for _, want := range []string{"Sources", "older.txt", "current result", "source-3", "omitted", "compressed", "active"} {
		if !strings.Contains(strings.ToLower(joined), strings.ToLower(want)) {
			t.Errorf("/sources panel lacks %q: %s", want, joined)
		}
	}
	if strings.Contains(joined, "/private/older.txt") || strings.Contains(joined, "old body") || strings.Contains(joined, "new body") {
		t.Fatalf("/sources exposed a private path or context body: %s", joined)
	}
}

func captureSourcePanel(t *testing.T, render func()) (string, []string) {
	t.Helper()
	var title string
	var lines []string
	ui.SetPanelRecorder(func(gotTitle string, panelLines []ui.PanelLine) {
		title = gotTitle
		for _, line := range panelLines {
			var text strings.Builder
			for _, span := range line {
				text.WriteString(span.Text)
			}
			lines = append(lines, text.String())
		}
	})
	t.Cleanup(func() { ui.SetPanelRecorder(nil) })
	render()
	return title, lines
}

func TestRequestShowsDurableOptimizationNoticeOnlyWhenContentChanges(t *testing.T) {
	optimizer := intelligence.NewContextOptimizer()
	optimizer.ImportanceThreshold = 1.1
	optimizer.MaxContextSize = 1
	view := NewConversationView("", "", false)
	c := &Chat{
		Display:          view,
		ContextOptimizer: optimizer,
		Messages: []Message{{
			Role: "user", Content: "```text\n" + strings.Repeat("source body ", 100) + "\n```",
			Sources: []chatcontext.SourceRef{{ID: "source-1", Kind: chatcontext.SourceKindFile, Name: "notes.txt", URI: "/private/notes.txt", SizeBytes: 1200, State: chatcontext.SourceStateActive}},
		}},
	}
	fetch := func(context.Context, string) (<-chan string, error) {
		stream := make(chan string, 1)
		stream <- "answer"
		close(stream)
		return stream, nil
	}
	if _, err := processInputWithFetcher(context.Background(), c, "q", &config.Config{}, renderStreamToString, fetch); err != nil {
		t.Fatalf("optimized request failed: %v", err)
	}
	foundNotice := false
	for _, entry := range view.entries {
		if entry.panelTitle == "Context optimization" {
			foundNotice = true
		}
	}
	if !foundNotice {
		t.Fatalf("durable notice entries = %+v, want a context optimization panel", view.entries)
	}
	if len(c.LastOptimizationChanges) != 1 || c.LastOptimizationChanges[0].State != chatcontext.SourceStateOmitted {
		t.Fatalf("last optimization changes = %+v, want the omitted source", c.LastOptimizationChanges)
	}
}

func TestRequestWithoutOptimizationClearsPreviousChangesAndShowsNoNotice(t *testing.T) {
	view := NewConversationView("", "", false)
	c := &Chat{
		Display:                 view,
		ContextOptimizer:        intelligence.NewContextOptimizer(),
		LastOptimizationChanges: []intelligence.OptimizationChange{{SourceID: "source-old", State: chatcontext.SourceStateCompressed}},
	}
	fetch := func(context.Context, string) (<-chan string, error) {
		stream := make(chan string, 1)
		stream <- "answer"
		close(stream)
		return stream, nil
	}
	if _, err := processInputWithFetcher(context.Background(), c, "small question", &config.Config{}, renderStreamToString, fetch); err != nil {
		t.Fatalf("request without optimization failed: %v", err)
	}
	if len(c.LastOptimizationChanges) != 0 {
		t.Fatalf("stale optimization changes remain: %+v", c.LastOptimizationChanges)
	}
	if len(view.entries) != 0 {
		t.Fatalf("unexpected durable panel entries for request without an optimization change: %+v", view.entries)
	}
}

func TestSourceLabelsAreAddedToSubmittedPromptAndLatestTurnOnly(t *testing.T) {
	first := chatcontext.SourceRef{ID: "source-1", Kind: chatcontext.SourceKindFile, Name: "older.md", URI: "/private/older.md", SizeBytes: 80, State: chatcontext.SourceStateActive}
	second := chatcontext.SourceRef{ID: "source-2", Kind: chatcontext.SourceKindURL, Name: "https://example.test/guide", URI: "https://example.test/guide", SizeBytes: 120, State: chatcontext.SourceStateActive}
	c := &Chat{
		Messages: []Message{
			{Role: "user", Content: "[File Context]\nolder context", Sources: []chatcontext.SourceRef{first}},
			{Role: "assistant", Content: "earlier answer"},
		},
		PendingSources: []chatcontext.SourceRef{second},
	}
	var submitted string
	fetch := func(_ context.Context, content string) (<-chan string, error) {
		submitted = content
		stream := make(chan string, 1)
		stream <- "answer"
		close(stream)
		return stream, nil
	}
	if _, err := processInputWithFetcher(context.Background(), c, "Compare them.", &config.Config{}, renderStreamToString, fetch); err != nil {
		t.Fatalf("request failed: %v", err)
	}
	for _, want := range []string{"[Source references]", "[1] source-1 · file · older.md", "[2] source-2 · url · https://example.test/guide"} {
		if !strings.Contains(submitted, want) {
			t.Errorf("submitted prompt lacks %q: %s", want, submitted)
		}
	}
	if strings.Contains(submitted, "/private/older.md") {
		t.Fatalf("source labels exposed a local path: %s", submitted)
	}
	var latestUser *Message
	for i := range c.Messages {
		if c.Messages[i].Role == "user" {
			latestUser = &c.Messages[i]
		}
	}
	if latestUser == nil || len(latestUser.Sources) != 1 || latestUser.Sources[0].ID != "source-2" {
		t.Fatalf("latest user source association = %+v, want only the source supplied for this turn", latestUser)
	}
	if !strings.Contains(latestUser.Content, "[Source references]") {
		t.Fatalf("latest user message did not retain its source labels: %q", latestUser.Content)
	}
}

func TestRequestWithoutSourceMetadataKeepsPromptUnchanged(t *testing.T) {
	c := &Chat{}
	var submitted string
	fetch := func(_ context.Context, content string) (<-chan string, error) {
		submitted = content
		stream := make(chan string, 1)
		stream <- "answer"
		close(stream)
		return stream, nil
	}
	if _, err := processInputWithFetcher(context.Background(), c, "plain question", &config.Config{}, renderStreamToString, fetch); err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if submitted != "plain question" {
		t.Fatalf("submitted prompt = %q, want unchanged prompt without source metadata", submitted)
	}
	if strings.Contains(c.Messages[0].Content, "Source references") {
		t.Fatalf("user history gained source labels without metadata: %q", c.Messages[0].Content)
	}
}

func TestSourceReferencePanelIsRetainedInConversationView(t *testing.T) {
	view := NewConversationView("", "", false)
	view.RecordSourceReferences([]chatcontext.SourceRef{{ID: "source-1", Kind: chatcontext.SourceKindFile, Name: "notes.txt", URI: "/home/alice/private/notes.txt", SizeBytes: 16, State: chatcontext.SourceStateActive}})
	if len(view.entries) != 1 || view.entries[0].panelTitle != "Sources for this exchange" {
		t.Fatalf("source reference panel entries = %+v", view.entries)
	}
	var lines strings.Builder
	for _, line := range view.entries[0].panelLines {
		for _, span := range line {
			lines.WriteString(span.Text)
		}
	}
	if !strings.Contains(lines.String(), "notes.txt") || !strings.Contains(lines.String(), "source-1") {
		t.Fatalf("source reference panel lacks compact reference: %s", lines.String())
	}
	if strings.Contains(lines.String(), "/home/alice/private") {
		t.Fatalf("source reference panel exposed local path: %s", lines.String())
	}
}
