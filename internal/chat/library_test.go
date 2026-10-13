package chat

import (
	"path/filepath"
	"strings"
	"testing"

	"duckduckgo-chat-cli/internal/chatcontext"
)

func TestLibrarySourceReferenceUsesShortNameAndPrivateURI(t *testing.T) {
	path := filepath.Join(t.TempDir(), "library", "nested", "notes.md")
	content := []byte("library notes")
	got := librarySourceRef(path, content)
	if got.Kind != chatcontext.SourceKindLibrary || got.Name != "notes.md" || got.URI != path || got.SizeBytes != len(content) || got.State != chatcontext.SourceStateActive {
		t.Fatalf("librarySourceRef() = %+v, want short name and private URI", got)
	}
}

func TestLibraryContextMessageUsesShortDisplayName(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private", "library", "report.md")
	message := libraryContextMessage(path, []byte("report text"))
	if !strings.Contains(message, "File: report.md") {
		t.Fatalf("library context message = %q, want short file name", message)
	}
	if strings.Contains(message, path) {
		t.Fatalf("library context message exposed the full local path: %q", message)
	}
}
