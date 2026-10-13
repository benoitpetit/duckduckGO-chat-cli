package chat

import (
	"errors"
	"os"
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

func TestLibraryLoadAddsPDFPagesAndReference(t *testing.T) {
	path := copyPDFFixture(t, "two_pages.pdf")
	c := &Chat{}
	loaded, totalBytes, skipped, err := loadLibraryFiles(c, []string{path})
	if err != nil {
		t.Fatalf("loadLibraryFiles() error = %v", err)
	}
	if loaded != 1 || totalBytes == 0 || len(skipped) != 0 {
		t.Fatalf("load result = loaded:%d bytes:%d skipped:%v, want one PDF", loaded, totalBytes, skipped)
	}
	if len(c.Messages) != 1 || !strings.Contains(c.Messages[0].Content, "[Page 1]") || !strings.Contains(c.Messages[0].Content, "[Page 2]") || !strings.Contains(c.Messages[0].Content, "Beta café second page") {
		t.Fatalf("library PDF context = %+v, want ordered pages", c.Messages)
	}
	if len(c.PendingSources) != 1 || c.PendingSources[0].Kind != chatcontext.SourceKindLibrary || c.PendingSources[0].Name != "report.pdf" || c.PendingSources[0].URI != path {
		t.Fatalf("library PDF source = %+v, want private URI and short name", c.PendingSources)
	}
	if !SupportedExtensions[".pdf"] {
		t.Fatal("PDF extension is not available in the library file selector")
	}
}

func TestLibraryPDFLimitFailureDoesNotPartiallyMutateContext(t *testing.T) {
	largeTextPath := filepath.Join(t.TempDir(), "large.txt")
	if err := os.WriteFile(largeTextPath, []byte(strings.Repeat("x", maxLibraryContextSize-10)), 0600); err != nil {
		t.Fatal(err)
	}
	pdfPath := copyPDFFixture(t, "two_pages.pdf")
	c := &Chat{}
	loaded, totalBytes, skipped, err := loadLibraryFiles(c, []string{largeTextPath, pdfPath})
	if !errors.Is(err, errLibraryContextLimit) {
		t.Fatalf("loadLibraryFiles() error = %v, want aggregate limit error", err)
	}
	if loaded != 0 || totalBytes != 0 || len(skipped) != 0 || len(c.Messages) != 0 || len(c.PendingSources) != 0 {
		t.Fatalf("aggregate rejection partially mutated state: loaded=%d bytes=%d skipped=%v messages=%+v sources=%+v", loaded, totalBytes, skipped, c.Messages, c.PendingSources)
	}
}

func TestLibraryPDFExtractionFailureDoesNotPartiallyMutateContext(t *testing.T) {
	firstTextPath := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(firstTextPath, []byte("notes"), 0600); err != nil {
		t.Fatal(err)
	}
	badPDFPath := filepath.Join(t.TempDir(), "broken.pdf")
	if err := os.WriteFile(badPDFPath, []byte("not a PDF"), 0600); err != nil {
		t.Fatal(err)
	}
	c := &Chat{}
	loaded, totalBytes, skipped, err := loadLibraryFiles(c, []string{firstTextPath, badPDFPath})
	if err == nil || !strings.Contains(err.Error(), "invalid or unsupported PDF") {
		t.Fatalf("loadLibraryFiles() error = %v, want malformed PDF error", err)
	}
	if loaded != 0 || totalBytes != 0 || len(skipped) != 0 || len(c.Messages) != 0 || len(c.PendingSources) != 0 {
		t.Fatalf("PDF rejection partially mutated state: loaded=%d bytes=%d skipped=%v messages=%+v sources=%+v", loaded, totalBytes, skipped, c.Messages, c.PendingSources)
	}
}
