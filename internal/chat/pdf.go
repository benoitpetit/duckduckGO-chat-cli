package chat

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/giraffesyo/pdf"
)

var errPDFExtractionLimit = errors.New("PDF extracted text exceeds the configured limit")

// PDFPage holds extracted text with its one-based page number.
type PDFPage struct {
	Number int
	Text   string
}

// extractPDFPages reads page text in order and refuses to retain more than the
// caller's UTF-8 byte budget.
func extractPDFPages(ctx context.Context, reader io.ReaderAt, size int64, maxExtractedBytes int64) ([]PDFPage, error) {
	if ctx == nil {
		return nil, errors.New("PDF extraction context is nil")
	}
	if maxExtractedBytes < 0 {
		return nil, errPDFExtractionLimit
	}
	pages := make([]PDFPage, 0)
	var extractedBytes int64
	_, err := pdf.ExtractPages(ctx, reader, size, pdf.Options{}, func(page pdf.Page) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		text := page.Text()
		pageBytes := int64(len(text))
		if pageBytes > maxExtractedBytes-extractedBytes {
			return errPDFExtractionLimit
		}
		extractedBytes += pageBytes
		pages = append(pages, PDFPage{Number: page.Number, Text: text})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("extract PDF pages: %w", err)
	}
	return pages, nil
}
