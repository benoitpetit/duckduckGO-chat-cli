package api

import (
	"context"
	"sync"

	"duckduckgo-chat-cli/internal/chat"
	"duckduckgo-chat-cli/internal/config"
)

// Session serializes access to the conversation owned by the HTTP API server.
type Session struct {
	mu              sync.RWMutex
	chat            *chat.Chat
	cfg             *config.Config
	streamProcessor streamProcessor
}

type streamProcessor func(context.Context, *chat.Chat, string, *config.Config, func(string) error) (string, error)

func NewSession(chatSession *chat.Chat, cfg *config.Config) *Session {
	return &Session{chat: chatSession, cfg: cfg}
}
