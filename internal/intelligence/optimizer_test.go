package intelligence

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"duckduckgo-chat-cli/internal/chatcontext"
	"duckduckgo-chat-cli/internal/media"
)

func TestOptimizerKeepsDistinctImageMessagesAndAttachmentsWhenTruncating(t *testing.T) {
	optimizer := NewContextOptimizer()
	optimizer.MaxContextSize = 1
	messages := []Message{
		{Role: "user", Content: "same", Images: []media.ImageAttachment{{Name: "first.png", MIMEType: "image/png", Data: []byte{1}}}},
		{Role: "user", Content: "same", Images: []media.ImageAttachment{{Name: "second.png", MIMEType: "image/png", Data: []byte{2}}}},
		{Role: "assistant", Content: strings.Repeat("uncompressible", 100)},
	}

	optimized, _ := optimizer.OptimizeContext(messages)
	if len(optimized) != 2 {
		t.Fatalf("optimized message count = %d, want both image messages and no oversized text message: %+v", len(optimized), optimized)
	}
	seen := map[byte]bool{}
	for _, message := range optimized {
		if len(message.Images) != 1 || len(message.Images[0].Data) != 1 {
			t.Fatalf("optimized message lost image bytes: %+v", message)
		}
		seen[message.Images[0].Data[0]] = true
	}
	if !seen[1] || !seen[2] {
		t.Fatalf("optimizer did not retain both distinct images: %+v", optimized)
	}
}

func TestCompressContentCompactsProseWithoutDestroyingCodeBlocks(t *testing.T) {
	optimizer := NewContextOptimizer()
	prose := "repeat   this\nrepeat this\nkeep   this"
	if got, want := optimizer.compressContent(prose), "repeat this\nkeep this"; got != want {
		t.Fatalf("compressContent(prose) = %q, want %q", got, want)
	}
	code := "```go\nif  x ==  1 {\n\treturn\n}\n```"
	if got := optimizer.compressContent(code); got != code {
		t.Fatalf("compressContent(code) = %q, want unchanged code block %q", got, code)
	}
	contextCode := "[File Context]\nFile: sample.go\n\n```go\nif  x ==  1 {\n\treturn\n}\n```"
	if got := optimizer.compressContent(contextCode); got != contextCode {
		t.Fatalf("compressContent(context code) = %q, want unchanged code context %q", got, contextCode)
	}
}

func TestCompressContentHonorsCompressionRatio(t *testing.T) {
	optimizer := NewContextOptimizer()
	optimizer.CompressionRatio = 0.5
	content := strings.Repeat("🦆 unique words ", 100)

	compressed := optimizer.compressContent(content)
	if len(compressed) > len(content)/2 {
		t.Fatalf("compressed size = %d bytes, want at most half of %d", len(compressed), len(content))
	}
	if !utf8.ValidString(compressed) {
		t.Fatalf("compressed content is not valid UTF-8: %q", compressed)
	}
	if !strings.Contains(compressed, "[...]") {
		t.Fatalf("compressed content lacks the omission marker: %q", compressed)
	}
}

func TestCompressContentUsesDefaultRatioForInvalidValues(t *testing.T) {
	content := strings.Repeat("unique words ", 100)
	for _, ratio := range []float64{0, -0.1, 1.1, math.NaN(), math.Inf(1)} {
		optimizer := NewContextOptimizer()
		optimizer.CompressionRatio = ratio
		compressed := optimizer.compressContent(content)
		if len(compressed) > int(float64(len(content))*0.7) {
			t.Errorf("ratio %v retained %d bytes, want default 70%% ceiling of %d", ratio, len(compressed), len(content))
		}
	}
}

func TestCompressionRatioOneDoesNotTruncate(t *testing.T) {
	optimizer := NewContextOptimizer()
	optimizer.CompressionRatio = 1
	content := strings.Repeat("unique words ", 100)
	if got, want := optimizer.compressContent(content), strings.TrimSpace(content); got != want {
		t.Fatalf("ratio 1 compressed to %q, want full prose %q", got, want)
	}
}

func TestOptimizeContextAppliesRatioToOlderConversationMessages(t *testing.T) {
	optimizer := NewContextOptimizer()
	optimizer.CompressionRatio = 0.5
	optimizer.MaxContextSize = 100_000
	messages := make([]Message, 31)
	started := time.Now()
	messages[0] = Message{Role: "user", Content: strings.Repeat("older context words ", 60), Timestamp: started}
	for i := 1; i < len(messages)-1; i++ {
		messages[i] = Message{Role: "assistant", Content: "short reply " + strings.Repeat("x", i), Timestamp: started.Add(time.Duration(i) * time.Second)}
	}
	messages[len(messages)-1] = Message{Role: "user", Content: "latest question", Timestamp: started.Add(time.Duration(len(messages)) * time.Second)}

	optimized, _ := optimizer.OptimizeContext(messages)
	if len(optimized) != len(messages) {
		t.Fatalf("OptimizeContext() retained %d messages, want %d", len(optimized), len(messages))
	}
	if !optimized[0].Compressed {
		t.Fatal("old low-importance message was not compressed")
	}
	if len(optimized[0].Content) > len(messages[0].Content)/2 {
		t.Fatalf("compressed old message retained %d bytes, want at most %d", len(optimized[0].Content), len(messages[0].Content)/2)
	}
	if optimized[len(optimized)-1].Content != "latest question" {
		t.Fatalf("latest message changed to %q", optimized[len(optimized)-1].Content)
	}
}

func TestOptimizationReportReportsCompressedTruncatedAndOmittedSources(t *testing.T) {
	newSource := func(id string) chatcontext.SourceRef {
		return chatcontext.SourceRef{ID: id, Kind: chatcontext.SourceKindFile, Name: id + ".txt", SizeBytes: 4000, State: chatcontext.SourceStateActive}
	}

	t.Run("compressed", func(t *testing.T) {
		optimizer := NewContextOptimizer()
		optimizer.ImportanceThreshold = 1.1
		optimizer.MaxContextSize = 100_000
		original := []Message{
			{Role: "system", Content: strings.Repeat("same line\n", 200), Sources: []chatcontext.SourceRef{newSource("source-1")}},
			{Role: "user", Content: "question"},
		}
		_, _, changes := optimizer.OptimizeContextWithReport(original)
		change := optimizationChangeByID(t, changes, "source-1")
		if change.State != chatcontext.SourceStateCompressed || change.BytesAfter >= change.BytesBefore {
			t.Fatalf("compressed change = %+v, want compressed and shorter", change)
		}
	})

	t.Run("truncated", func(t *testing.T) {
		optimizer := NewContextOptimizer()
		optimizer.ImportanceThreshold = 1.1
		optimizer.MaxContextSize = 100_000
		lines := make([]string, 0, 120)
		for i := 0; i < 120; i++ {
			lines = append(lines, fmt.Sprintf("unique line %03d with enough contextual detail to remain meaningful", i))
		}
		original := []Message{
			{Role: "system", Content: strings.Join(lines, "\n"), Sources: []chatcontext.SourceRef{newSource("source-2")}},
			{Role: "user", Content: "question"},
		}
		optimized, _, changes := optimizer.OptimizeContextWithReport(original)
		change := optimizationChangeByID(t, changes, "source-2")
		if change.State != chatcontext.SourceStateTruncated || change.BytesAfter >= change.BytesBefore {
			t.Fatalf("truncated change = %+v, want truncated and shorter", change)
		}
		if !strings.Contains(optimized[0].Content, "[...]") {
			t.Fatalf("optimized content = %q, want truncation marker", optimized[0].Content)
		}
	})

	t.Run("omitted", func(t *testing.T) {
		optimizer := NewContextOptimizer()
		optimizer.ImportanceThreshold = 1.1
		optimizer.MaxContextSize = 1
		original := []Message{
			{Role: "system", Content: "```text\n" + strings.Repeat("private source body ", 100) + "\n```", Sources: []chatcontext.SourceRef{newSource("source-3")}},
			{Role: "user", Content: "q"},
		}
		_, _, changes := optimizer.OptimizeContextWithReport(original)
		change := optimizationChangeByID(t, changes, "source-3")
		if change.State != chatcontext.SourceStateOmitted || change.BytesAfter != 0 || change.BytesBefore == 0 {
			t.Fatalf("omitted change = %+v, want omitted with zero retained bytes", change)
		}
	})
}

func optimizationChangeByID(t *testing.T, changes []OptimizationChange, id string) OptimizationChange {
	t.Helper()
	for _, change := range changes {
		if change.SourceID == id {
			return change
		}
	}
	t.Fatalf("optimization report has no change for %s: %+v", id, changes)
	return OptimizationChange{}
}
