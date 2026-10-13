package chat

import (
	"duckduckgo-chat-cli/internal/chatcontext"
	"strings"
	"testing"
)

func TestFormatSearchResultsHonorsSnippetSetting(t *testing.T) {
	results := []SearchResult{{
		Title:        "Example",
		FormattedUrl: "https://example.com",
		Snippet:      "secret snippet",
		HtmlSnippet:  "<b>secret snippet</b>",
	}}

	withoutSnippet := formatSearchResults(results, false)
	if withoutSnippet == "" || strings.Contains(withoutSnippet, "secret snippet") {
		t.Fatalf("snippet leaked while disabled: %q", withoutSnippet)
	}
	if withSnippet := formatSearchResults(results, true); withSnippet == "" || !strings.Contains(withSnippet, "secret snippet") {
		t.Fatalf("snippet missing when enabled: %q", withSnippet)
	}
}

func TestSearchSourceReferencesKeepResultOrderAndURLs(t *testing.T) {
	results := []SearchResult{
		{Title: "First result", FormattedUrl: "https://example.test/first", Snippet: "first snippet"},
		{Title: "Second result", HtmlFormattedUrl: "https://example.test/second", HtmlTitle: "Second title"},
	}
	got := searchSourceRefs("duck context", results)
	if len(got) != 2 {
		t.Fatalf("searchSourceRefs() = %+v, want two results", got)
	}
	if got[0].Kind != chatcontext.SourceKindSearch || got[0].Name != "First result" || got[0].URI != "https://example.test/first" || got[0].SizeBytes == 0 {
		t.Fatalf("first source = %+v", got[0])
	}
	if got[1].Kind != chatcontext.SourceKindSearch || got[1].Name != "Second result" || got[1].URI != "https://example.test/second" {
		t.Fatalf("second source = %+v", got[1])
	}
}
