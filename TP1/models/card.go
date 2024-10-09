package models

import (
	"database/sql"
	"github.com/google/uuid"
)

type AddCard struct {
	Code   string
	DeckId uuid.UUID
}

type Card struct {
	Code  string `json:"code"`
	Image string `json:"image"`
	Rank  int    `json:"rank"`
	Suit  string `json:"suit"`
}

type CardResponse struct {
	Deck  DeckRequest `json:"deck"`
	Cards []Card      `json:"cards"`
}

type DrawCardRequest struct {
	NbCard  int          `json:"-"`
	Reponse CardResponse `json:"response"`
}

type ShuffleRequest struct {
	DeckId   uuid.UUID `json:"-"`
	Db       *sql.DB   `json:"-"`
	ErrorMsg string    `json:"error,omitempty"`
	Response string    `json:"response,omitempty"`
}
