package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"duckduckgo-chat-cli/internal/analytics"
	"duckduckgo-chat-cli/internal/chatcontext"
	"duckduckgo-chat-cli/internal/config"
	"duckduckgo-chat-cli/internal/intelligence"
	"duckduckgo-chat-cli/internal/media"
	"duckduckgo-chat-cli/internal/models"
)

var testPNG = func() []byte {
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		panic(err)
	}
	return encoded.Bytes()
}()

func writeTestPNG(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "logo.png")
	if err := os.WriteFile(path, testPNG, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestAddedFileAndChainContextUpdateContextTokenEstimate(t *testing.T) {
	tracker := analytics.NewChatAnalytics()
	c := &Chat{Analytics: tracker}
	c.addFileContext("notes.txt", []byte("file information"))
	c.AddContextMessage("[URL Context]\nhttps://example.test\n\npage information")
	snapshot := tracker.Snapshot()
	if snapshot.ContextMessages != 2 || snapshot.ContextTokensEstimate == 0 || snapshot.UserTokensEstimate != 0 {
		t.Fatalf("context estimates = messages:%d tokens:%d user tokens:%d, want two tracked context messages and no user tokens", snapshot.ContextMessages, snapshot.ContextTokensEstimate, snapshot.UserTokensEstimate)
	}
	if snapshot.TotalTokensEstimate != snapshot.ContextTokensEstimate {
		t.Fatalf("total token estimate = %d, context estimate = %d, want context represented in total", snapshot.TotalTokensEstimate, snapshot.ContextTokensEstimate)
	}
}

func TestPromptWithInsertedContextSplitsRoleTokenEstimates(t *testing.T) {
	tracker := analytics.NewChatAnalytics()
	c := &Chat{Analytics: tracker}
	contextText := strings.Repeat("c", 80)
	prompt := strings.Repeat("p", 40)
	fetch := func(context.Context, string) (<-chan string, error) {
		stream := make(chan string, 1)
		stream <- "answer"
		close(stream)
		return stream, nil
	}
	_, err := processInputWithFetcherAndTokenRoles(context.Background(), c, contextText+"\n\n"+prompt, &config.Config{}, renderStreamToString, fetch, len(prompt), len(contextText))
	if err != nil {
		t.Fatal(err)
	}
	snapshot := tracker.Snapshot()
	if snapshot.UserTokensEstimate == 0 || snapshot.ContextTokensEstimate == 0 {
		t.Fatalf("role token estimates = user:%d context:%d, want both nonzero", snapshot.UserTokensEstimate, snapshot.ContextTokensEstimate)
	}
	if snapshot.TotalTokensEstimate != snapshot.UserTokensEstimate+snapshot.ContextTokensEstimate+snapshot.AssistantTokensEstimate {
		t.Fatalf("total estimate %d does not equal role estimates user:%d context:%d assistant:%d", snapshot.TotalTokensEstimate, snapshot.UserTokensEstimate, snapshot.ContextTokensEstimate, snapshot.AssistantTokensEstimate)
	}
}

func TestImageFileContextQueuesImageForNextPrompt(t *testing.T) {
	path := writeTestPNG(t)
	expected, err := media.ReadImage(path)
	if err != nil {
		t.Fatal(err)
	}
	c := &Chat{}
	handleFileCommand(c, "/file "+path, &config.Config{}, nil, func(*Chat, string, *config.Config) {})

	if len(c.pendingImages) != 1 {
		t.Fatalf("pending image count = %d, want 1", len(c.pendingImages))
	}
	if c.pendingImages[0].Name != expected.Name || c.pendingImages[0].MIMEType != "image/webp" || !bytes.Equal(c.pendingImages[0].Data, expected.Data) {
		t.Fatalf("pending image = %+v, want the normalized WebP attachment", c.pendingImages[0])
	}
	if len(c.Messages) != 0 {
		t.Fatalf("queued image was inserted as a separate message: %+v", c.Messages)
	}
	if len(c.PendingSources) != 1 || c.PendingSources[0].Kind != chatcontext.SourceKindImage || c.PendingSources[0].Name != "logo.png" || c.PendingSources[0].URI != path {
		t.Fatalf("pending image sources = %+v, want local image reference", c.PendingSources)
	}
}

func TestImageFilePromptQueuesAttachmentWithPrompt(t *testing.T) {
	path := writeTestPNG(t)
	expected, err := media.ReadImage(path)
	if err != nil {
		t.Fatal(err)
	}
	c := &Chat{}
	var requested string
	var message Message
	handleFileCommand(c, "/file "+path+" -- describe this duck", &config.Config{}, nil, func(chat *Chat, prompt string, cfg *config.Config) {
		requested = prompt
		_, err := processInputWithFetcher(context.Background(), chat, prompt, cfg, func(stream <-chan string) string {
			for range stream {
			}
			return "answer"
		}, func(_ context.Context, _ string) (<-chan string, error) {
			message = chat.Messages[len(chat.Messages)-1]
			stream := make(chan string, 1)
			stream <- "answer"
			close(stream)
			return stream, nil
		})
		if err != nil {
			t.Errorf("processInputWithFetcher() error = %v", err)
		}
	})

	if requested != "describe this duck" {
		t.Fatalf("requested prompt = %q, want %q", requested, "describe this duck")
	}
	if message.Content != requested || len(message.Images) != 1 || message.Images[0].MIMEType != "image/webp" || !bytes.Equal(message.Images[0].Data, expected.Data) {
		t.Fatalf("submitted user message = %+v, want prompt and image in one message", message)
	}
	if len(message.Sources) != 1 || message.Sources[0].Kind != chatcontext.SourceKindImage || message.Sources[0].Name != "logo.png" {
		t.Fatalf("submitted image sources = %+v, want image reference on the prompt", message.Sources)
	}
	wireMessage, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("serialize image message: %v", err)
	}
	if !bytes.Contains(wireMessage, []byte(`"mimeType":"image/webp"`)) || !bytes.Contains(wireMessage, []byte(`"image":"data:image/webp;base64,`)) {
		t.Fatalf("serialized image message does not match Duck.ai's WebP image block format: %s", wireMessage)
	}
}

func TestPendingImagesRestoreAfterFailedInputAndConsumeAfterSuccess(t *testing.T) {
	image := media.ImageAttachment{Name: "logo.png", MIMEType: "image/png", Data: testPNG}
	c := &Chat{pendingImages: []media.ImageAttachment{image}}
	render := func(stream <-chan string) string {
		for range stream {
		}
		return "answer"
	}
	_, err := processInputWithFetcher(context.Background(), c, "retry me", &config.Config{}, render, func(context.Context, string) (<-chan string, error) {
		return nil, errors.New("temporary failure")
	})
	if err == nil {
		t.Fatal("failed input unexpectedly succeeded")
	}
	if len(c.pendingImages) != 1 || c.Messages != nil {
		t.Fatalf("failed input did not restore state: pending=%d messages=%+v", len(c.pendingImages), c.Messages)
	}

	_, err = processInputWithFetcher(context.Background(), c, "retry me", &config.Config{}, render, func(context.Context, string) (<-chan string, error) {
		stream := make(chan string, 1)
		stream <- "answer"
		close(stream)
		return stream, nil
	})
	if err != nil {
		t.Fatalf("successful retry error = %v", err)
	}
	if len(c.pendingImages) != 0 {
		t.Fatalf("successful input left %d pending images, want 0", len(c.pendingImages))
	}
	if len(c.Messages) != 2 || len(c.Messages[0].Images) != 1 || string(c.Messages[0].Images[0].Data) != string(testPNG) {
		t.Fatalf("successful retry messages = %+v, want attached user message and assistant response", c.Messages)
	}
}

func TestPendingImagesRestoreAfterMidStreamFailure(t *testing.T) {
	image := media.ImageAttachment{Name: "logo.png", MIMEType: "image/png", Data: testPNG}
	c := &Chat{pendingImages: []media.ImageAttachment{image}}
	streamErrs := make(chan error, 1)
	streamErrs <- errors.New("truncated response")
	close(streamErrs)
	_, err := processInputWithStreamFetcher(context.Background(), c, "retry me", &config.Config{}, func(stream <-chan string) string {
		for range stream {
		}
		return "partial answer"
	}, func(context.Context, string) (<-chan string, <-chan error, error) {
		stream := make(chan string)
		close(stream)
		return stream, streamErrs, nil
	})
	if err == nil {
		t.Fatal("mid-stream failure unexpectedly succeeded")
	}
	if len(c.pendingImages) != 1 || len(c.Messages) != 0 {
		t.Fatalf("mid-stream failure did not roll back state: pending=%d messages=%+v", len(c.pendingImages), c.Messages)
	}
}

func TestInvalidImageFileDoesNotInvokeProcessor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.png")
	if err := os.WriteFile(path, []byte("not an image"), 0600); err != nil {
		t.Fatal(err)
	}
	c := &Chat{}
	called := false
	err := handleFileCommand(c, "/file "+path+" -- describe this", &config.Config{}, nil, func(*Chat, string, *config.Config) {
		called = true
	})
	if err == nil {
		t.Fatal("invalid image unexpectedly succeeded")
	}
	if called || len(c.pendingImages) != 0 || len(c.Messages) != 0 {
		t.Fatalf("invalid image had side effects: processor called=%t pending=%d messages=%+v", called, len(c.pendingImages), c.Messages)
	}
}

func TestImageRequestsUseVisionCapableDuckAIModel(t *testing.T) {
	for _, selectedModel := range []models.Model{models.MistralSmall, models.GPTOSS120B} {
		t.Run(string(selectedModel), func(t *testing.T) {
			c := &Chat{Model: selectedModel, pendingImages: []media.ImageAttachment{{Name: "logo.png", MIMEType: "image/png", Data: testPNG}}}
			var requestModel models.Model
			_, err := processInputWithStreamFetcher(context.Background(), c, "What is in this image?", &config.Config{}, func(stream <-chan string) string {
				for range stream {
				}
				return "A duck."
			}, func(context.Context, string) (<-chan string, <-chan error, error) {
				requestModel = c.Model
				stream := make(chan string)
				close(stream)
				return stream, nil, nil
			})
			if err != nil {
				t.Fatalf("processInputWithStreamFetcher() error = %v", err)
			}
			if requestModel != models.GPT54Mini {
				t.Fatalf("request model = %q, want vision-capable fallback %q", requestModel, models.GPT54Mini)
			}
			if c.Model != selectedModel {
				t.Fatalf("selected model changed to %q after image request, want %q", c.Model, selectedModel)
			}
		})
	}
}

func TestAddedFileRecordsPrivateURIAndShortDisplayName(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private", "notes.txt")
	content := []byte("project notes")
	c := &Chat{}
	c.addFileContext(path, content)

	if len(c.PendingSources) != 1 {
		t.Fatalf("pending sources = %+v, want one file source", c.PendingSources)
	}
	source := c.PendingSources[0]
	if source.Kind != chatcontext.SourceKindFile || source.Name != "notes.txt" || source.URI != path || source.SizeBytes != len(content) || source.State != chatcontext.SourceStateActive {
		t.Fatalf("file source = %+v, want short name and full internal URI", source)
	}
}

func TestPendingSourcesRestoreAfterFailedInputAndConsumeOnSuccessfulTurn(t *testing.T) {
	source := chatcontext.SourceRef{ID: "source-1", Kind: chatcontext.SourceKindFile, Name: "notes.txt", URI: "/private/notes.txt", SizeBytes: 12, State: chatcontext.SourceStateActive}
	c := &Chat{PendingSources: []chatcontext.SourceRef{source}}
	render := func(stream <-chan string) string {
		for range stream {
		}
		return "answer"
	}

	_, err := processInputWithFetcher(context.Background(), c, "retry me", &config.Config{}, render, func(context.Context, string) (<-chan string, error) {
		return nil, errors.New("temporary failure")
	})
	if err == nil {
		t.Fatal("failed input unexpectedly succeeded")
	}
	if len(c.PendingSources) != 1 || c.PendingSources[0].ID != source.ID || len(c.Messages) != 0 {
		t.Fatalf("failed input did not restore pending sources: pending=%+v messages=%+v", c.PendingSources, c.Messages)
	}

	_, err = processInputWithFetcher(context.Background(), c, "retry me", &config.Config{}, render, func(context.Context, string) (<-chan string, error) {
		stream := make(chan string, 1)
		stream <- "answer"
		close(stream)
		return stream, nil
	})
	if err != nil {
		t.Fatalf("successful retry error = %v", err)
	}
	if len(c.PendingSources) != 0 {
		t.Fatalf("successful turn left pending sources: %+v", c.PendingSources)
	}
	if len(c.Messages) != 2 || len(c.Messages[0].Sources) != 1 || c.Messages[0].Sources[0] != source || len(c.Messages[1].Sources) != 0 {
		t.Fatalf("committed messages = %+v, want source on user turn only", c.Messages)
	}
}

func TestSourceMetadataSurvivesOptimizerConversionButStaysOffWire(t *testing.T) {
	source := chatcontext.SourceRef{ID: "source-1", Kind: chatcontext.SourceKindURL, Name: "Guide", URI: "https://example.test", SizeBytes: 40, State: chatcontext.SourceStateActive}
	c := &Chat{Messages: []Message{{Role: "user", Content: "Summarize this", Sources: []chatcontext.SourceRef{source}}}}

	converted := c.convertMessagesToIntelligence()
	if len(converted) != 1 || len(converted[0].Sources) != 1 || converted[0].Sources[0] != source {
		t.Fatalf("optimizer conversion lost sources: %+v", converted)
	}
	optimizer := intelligence.NewContextOptimizer()
	optimizer.MaxContextSize = 10000
	optimized, _ := optimizer.OptimizeContext(converted)
	restored := c.convertFromIntelligenceMessages(optimized)
	if len(restored) != 1 || len(restored[0].Sources) != 1 || restored[0].Sources[0] != source {
		t.Fatalf("optimized conversion lost sources: %+v", restored)
	}

	wire, err := json.Marshal(c.Messages[0])
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(wire, []byte("sources")) || bytes.Contains(wire, []byte("example.test")) {
		t.Fatalf("internal source metadata entered the Duck.ai wire message: %s", wire)
	}
}

func TestAddedURLRecordsSourceForNextTurn(t *testing.T) {
	c := &Chat{}
	c.addURLContext("https://example.test/guide", "page text")

	if len(c.PendingSources) != 1 {
		t.Fatalf("pending sources = %+v, want one URL reference", c.PendingSources)
	}
	source := c.PendingSources[0]
	if source.Kind != chatcontext.SourceKindURL || source.Name != "https://example.test/guide" || source.URI != source.Name || source.SizeBytes != len("page text") {
		t.Fatalf("URL source = %+v", source)
	}
}
