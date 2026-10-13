package chatcontext

import (
	"reflect"
	"testing"

	"duckduckgo-chat-cli/internal/media"
)

func TestChainContextStoresAndReturnsImages(t *testing.T) {
	ctx := New()
	image := media.ImageAttachment{Name: "logo.png", MIMEType: "image/png", Data: []byte{1, 2, 3}}
	ctx.AddImage("/tmp/logo.png", image)

	if ctx.IsEmpty() {
		t.Fatal("image-only context is empty")
	}
	if got := ctx.ImageAttachments(); len(got) != 1 || got[0].Name != "logo.png" || string(got[0].Data) != string(image.Data) {
		t.Fatalf("ImageAttachments() = %+v, want image metadata and bytes", got)
	}
	if got := ctx.String(); got != "[Image Context]\nFile: logo.png" {
		t.Fatalf("String() = %q, want image marker", got)
	}
}

func TestContextSourcesAreOrderedStableAndUseShortDisplayNames(t *testing.T) {
	ctx := New()
	ctx.AddFile("/home/alice/project/notes.md", []byte("local notes"))
	ctx.AddImage("/home/alice/project/diagram.png", media.ImageAttachment{
		Name: "diagram.png", MIMEType: "image/png", Data: []byte{1, 2, 3},
	})
	ctx.AddURL("https://example.test/guide", "page body")
	ctx.AddSearch("context provenance", "search summary",
		SourceRef{Kind: SourceKindSearch, Name: "First result", URI: "https://example.test/first", SizeBytes: 10},
		SourceRef{Kind: SourceKindSearch, Name: "Second result", URI: "https://example.test/second", SizeBytes: 20},
	)

	got := ctx.Sources()
	want := []SourceRef{
		{ID: "source-1", Kind: SourceKindFile, Name: "notes.md", URI: "/home/alice/project/notes.md", SizeBytes: len("local notes"), State: SourceStateActive},
		{ID: "source-2", Kind: SourceKindImage, Name: "diagram.png", URI: "/home/alice/project/diagram.png", SizeBytes: 3, State: SourceStateActive},
		{ID: "source-3", Kind: SourceKindURL, Name: "https://example.test/guide", URI: "https://example.test/guide", SizeBytes: len("page body"), State: SourceStateActive},
		{ID: "source-4", Kind: SourceKindSearch, Name: "First result", URI: "https://example.test/first", SizeBytes: 10, State: SourceStateActive},
		{ID: "source-5", Kind: SourceKindSearch, Name: "Second result", URI: "https://example.test/second", SizeBytes: 20, State: SourceStateActive},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Sources() = %+v, want %+v", got, want)
	}

	got[0].Name = "mutated"
	if again := ctx.Sources(); again[0].Name != "notes.md" {
		t.Fatalf("Sources() exposed internal metadata: %+v", again[0])
	}
}
