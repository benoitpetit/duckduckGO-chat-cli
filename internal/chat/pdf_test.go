package chat

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestExtractPDFPagesTwoPageOrderingAndUTF8(t *testing.T) {
	pages := extractPDFFixture(t, "two_pages.pdf", context.Background(), 1<<20)
	if len(pages) != 2 {
		t.Fatalf("page count = %d, want 2: %+v", len(pages), pages)
	}
	if pages[0].Number != 1 || pages[1].Number != 2 {
		t.Fatalf("page numbers = %d, %d, want one-based order", pages[0].Number, pages[1].Number)
	}
	if !strings.Contains(pages[0].Text, "Alpha first page") || !strings.Contains(pages[1].Text, "Beta café second page") {
		t.Fatalf("page text = %+v, want ordered UTF-8 page text", pages)
	}
	if !strings.Contains(pages[1].Text, "café") {
		t.Fatalf("page text is not decoded as UTF-8: %q", pages[1].Text)
	}
}

func TestExtractPDFPagesKeepsEmptyTextPage(t *testing.T) {
	pages := extractPDFFixture(t, "empty_page.pdf", context.Background(), 1<<20)
	if len(pages) != 1 || pages[0].Number != 1 || strings.TrimSpace(pages[0].Text) != "" {
		t.Fatalf("empty PDF pages = %+v, want one empty page", pages)
	}
}

func TestExtractPDFPagesRejectsMalformedStructure(t *testing.T) {
	_, err := extractPDFFixtureWithError(t, "malformed.pdf", context.Background(), 1<<20)
	if err == nil {
		t.Fatal("malformed PDF extraction succeeded")
	}
}

func TestExtractPDFPagesStopsAtExtractedByteLimit(t *testing.T) {
	_, err := extractPDFFixtureWithError(t, "two_pages.pdf", context.Background(), 5)
	if !errors.Is(err, errPDFExtractionLimit) {
		t.Fatalf("limit error = %v, want %v", err, errPDFExtractionLimit)
	}
}

func TestExtractPDFPagesHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := extractPDFFixtureWithError(t, "two_pages.pdf", ctx, 1<<20)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled extraction error = %v, want context.Canceled", err)
	}
}

func extractPDFFixture(t *testing.T, fixture string, ctx context.Context, maxBytes int64) []PDFPage {
	t.Helper()
	pages, err := extractPDFFixtureWithError(t, fixture, ctx, maxBytes)
	if err != nil {
		t.Fatalf("extractPDFPages(%s): %v", fixture, err)
	}
	return pages
}

func extractPDFFixtureWithError(t *testing.T, fixture string, ctx context.Context, maxBytes int64) ([]PDFPage, error) {
	t.Helper()
	file, err := os.Open("testdata/pdf/" + fixture)
	if err != nil {
		t.Fatalf("open fixture %s: %v", fixture, err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		t.Fatalf("stat fixture %s: %v", fixture, err)
	}
	return extractPDFPages(ctx, file, info.Size(), maxBytes)
}

func TestExtractPDFPagesRejectsCyclicPageTree(t *testing.T) {
	_, err := extractPDFFixtureWithError(t, "cyclic_page_tree.pdf", context.Background(), 1<<20)
	if err == nil {
		t.Fatal("cyclic page tree extraction succeeded")
	}
}
