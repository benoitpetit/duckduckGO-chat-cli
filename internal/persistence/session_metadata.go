package persistence

import (
	"fmt"
	"strings"
	"unicode"

	"duckduckgo-chat-cli/internal/intelligence"
)

const (
	maxSessionTitleRunes = 100
	maxSessionTagRunes   = 24
	maxSessionTags       = 10
)

// SessionMetadata contains the user-editable fields attached to an archive.
type SessionMetadata struct {
	Title    string   `json:"title"`
	Tags     []string `json:"tags"`
	Favorite bool     `json:"favorite"`
}

func validateSessionMetadata(metadata SessionMetadata) (SessionMetadata, error) {
	metadata.Title = normalizeSessionText(metadata.Title)
	if len([]rune(metadata.Title)) > maxSessionTitleRunes {
		return SessionMetadata{}, fmt.Errorf("session title must be at most %d characters", maxSessionTitleRunes)
	}

	tags := make([]string, 0, len(metadata.Tags))
	seen := make(map[string]struct{}, len(metadata.Tags))
	for _, tag := range metadata.Tags {
		tag = normalizeSessionText(tag)
		if tag == "" {
			continue
		}
		if len([]rune(tag)) > maxSessionTagRunes {
			return SessionMetadata{}, fmt.Errorf("session tags must be at most %d characters", maxSessionTagRunes)
		}
		key := strings.ToLower(tag)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		tags = append(tags, tag)
		if len(tags) > maxSessionTags {
			return SessionMetadata{}, fmt.Errorf("a session can have at most %d tags", maxSessionTags)
		}
	}
	metadata.Tags = tags
	return metadata, nil
}

func normalizeLoadedSessionMetadata(session *ConversationSession) {
	if session == nil {
		return
	}
	session.Title = truncateSessionTitle(normalizeSessionText(session.Title))
	if session.Title == "" {
		session.Title = generatedSessionTitle(session.Messages)
	}

	metadata, _ := validateSessionMetadata(SessionMetadata{Tags: session.Tags})
	session.Tags = metadata.Tags
}

func generatedSessionTitle(messages []intelligence.Message) string {
	for _, message := range messages {
		if message.Role != "user" {
			continue
		}
		if title := truncateSessionTitle(normalizeSessionText(message.Content)); title != "" {
			return title
		}
	}
	return "Untitled conversation"
}

func normalizeSessionText(value string) string {
	var normalized strings.Builder
	spacePending := false
	for _, r := range value {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			spacePending = normalized.Len() > 0
			continue
		}
		if spacePending {
			normalized.WriteByte(' ')
			spacePending = false
		}
		normalized.WriteRune(r)
	}
	return normalized.String()
}

func truncateSessionTitle(title string) string {
	runes := []rune(title)
	if len(runes) <= maxSessionTitleRunes {
		return title
	}
	return string(runes[:maxSessionTitleRunes-1]) + "…"
}
