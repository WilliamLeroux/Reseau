package models

import "github.com/google/uuid"

type DeckRequest struct {
	DeckId     uuid.UUID `json:"deckId"`
	Error      string    `json:"error,omitempty"`
	CardAmount int       `json:"cardAmount"`
	Joker      bool      `json:"-"`
}
