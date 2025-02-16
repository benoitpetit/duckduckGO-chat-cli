package interfaces

import "duckduckgo-chat-cli/internal/models"

type ChatSession interface {
	ChangeModel(model models.Model)
}
