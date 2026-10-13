package chat

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"duckduckgo-chat-cli/internal/chatcontext"
	"duckduckgo-chat-cli/internal/config"
)

func TestProcessInputContextStreamEmitsOrderedDeltasAndCommitsFinalAnswer(t *testing.T) {
	c := &Chat{}
	fetch := func(context.Context, string) (<-chan string, <-chan error, error) {
		chunks := make(chan string, 3)
		chunks <- "first "
		chunks <- "second "
		chunks <- "third"
		close(chunks)
		return chunks, nil, nil
	}
	var deltas []string
	response, err := processInputContextStreamWithFetcher(context.Background(), c, "hello", &config.Config{}, func(delta string) error {
		deltas = append(deltas, delta)
		return nil
	}, fetch)
	if err != nil {
		t.Fatalf("stream processing: %v", err)
	}
	if response != "first second third" || !reflect.DeepEqual(deltas, []string{"first ", "second ", "third"}) {
		t.Fatalf("response=%q deltas=%q", response, deltas)
	}
	if len(c.Messages) != 2 || c.Messages[1].Role != "assistant" || c.Messages[1].Content != response {
		t.Fatalf("committed messages = %+v", c.Messages)
	}
}

func TestProcessInputContextStreamCancelsAndRollsBackOnDeltaFailure(t *testing.T) {
	clientErr := errors.New("client disconnected")
	source := chatcontext.SourceRef{ID: "source-1", Kind: chatcontext.SourceKindFile, Name: "notes.txt"}
	c := &Chat{PendingSources: []chatcontext.SourceRef{source}}
	fetch := func(ctx context.Context, _ string) (<-chan string, <-chan error, error) {
		chunks := make(chan string)
		go func() {
			defer close(chunks)
			chunks <- "partial answer"
			<-ctx.Done()
		}()
		return chunks, nil, nil
	}
	response, err := processInputContextStreamWithFetcher(context.Background(), c, "hello", &config.Config{}, func(string) error {
		return clientErr
	}, fetch)
	if response != "" || !errors.Is(err, clientErr) {
		t.Fatalf("stream result = %q, %v; want callback error", response, err)
	}
	if len(c.Messages) != 0 || len(c.PendingSources) != 1 || c.PendingSources[0].ID != source.ID {
		t.Fatalf("disconnect left partial chat state: messages=%+v sources=%+v", c.Messages, c.PendingSources)
	}
}

func TestProcessInputContextStreamPreservesRateLimitErrors(t *testing.T) {
	c := &Chat{}
	fetch := func(context.Context, string) (<-chan string, <-chan error, error) {
		return nil, nil, ErrRateLimited
	}
	_, err := processInputContextStreamWithFetcher(context.Background(), c, "hello", &config.Config{}, func(string) error { return nil }, fetch)
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("rate limit error = %v, want %v", err, ErrRateLimited)
	}
	if len(c.Messages) != 0 {
		t.Fatalf("rate-limited request changed chat history: %+v", c.Messages)
	}
}
