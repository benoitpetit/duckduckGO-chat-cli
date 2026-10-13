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
	if len(view.entries) != 1 || view.entries[0].panelTitle != "Context optimization" {
		t.Fatalf("durable notice entries = %+v, want one context optimization panel", view.entries)
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
