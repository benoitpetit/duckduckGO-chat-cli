package chat

import (
	"strings"
	"testing"

	"duckduckgo-chat-cli/internal/chatcontext"
)

func TestConversationExportIncludesCompactSourceReferences(t *testing.T) {
	chat := &Chat{Messages: []Message{
		{Role: "user", Content: "[File Context]\nprivate body", Sources: []chatcontext.SourceRef{{ID: "source-1", Kind: chatcontext.SourceKindFile, Name: "notes.md", URI: "/home/alice/private/notes.md", SizeBytes: 42, State: chatcontext.SourceStateCompressed}}},
		{Role: "user", Content: "Summarize this", Sources: []chatcontext.SourceRef{{ID: "source-1", Kind: chatcontext.SourceKindFile, Name: "notes.md", URI: "/home/alice/private/notes.md", SizeBytes: 42, State: chatcontext.SourceStateCompressed}}},
		{Role: "assistant", Content: "Summary."},
	}}
	_, exported := chat.Export("conversation", "")
	for _, want := range []string{"source-1", "notes.md", "/home/alice/private/notes.md", "42 bytes", "compressed"} {
		if !strings.Contains(exported, want) {
			t.Errorf("conversation export lacks %q: %s", want, exported)
		}
	}
	if strings.Count(exported, "private body") != 1 {
		t.Fatalf("conversation export duplicated source body: %s", exported)
	}
}

func TestConversationExportWithoutSourceMetadataKeepsExistingSections(t *testing.T) {
	chat := &Chat{Messages: []Message{{Role: "user", Content: "Plain question"}, {Role: "assistant", Content: "Plain answer"}}}
	_, exported := chat.Export("conversation", "")
	if !strings.Contains(exported, "Plain question") || !strings.Contains(exported, "Plain answer") {
		t.Fatalf("legacy conversation export lost messages: %s", exported)
	}
	if strings.Contains(exported, "Source references") || strings.Contains(exported, "source-") {
		t.Fatalf("conversation without metadata gained source references: %s", exported)
	}
}
