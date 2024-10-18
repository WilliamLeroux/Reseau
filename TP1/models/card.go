package models

import (
	"database/sql"
	"github.com/google/uuid"
)

type AddCard struct {
	DeckId  uuid.UUID `json:"deck_id"`
	Db      *sql.DB   `json:"-"`
	NewCard string    `json:"-"`
	Order   int       `json:"-"`
	Error   string    `json:"error,omitempty"`
	Card    []Card    `json:"cards"`
}

type Card struct {
	Code  string `json:"code"`
	Image string `json:"image"`
	Rank  int    `json:"rank"`
	Suit  string `json:"suit"`
	Date  string `json:"date,omitempty"`
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
	DeckId    uuid.UUID `json:"deck_id"`
	Db        *sql.DB   `json:"-"`
	ErrorMsg  string    `json:"error,omitempty"`
	Response  string    `json:"response,omitempty"`
	Remaining int       `json:"remaining"`
}

type ShowDrawRequest struct {
	DeckId   uuid.UUID `json:"-"`
	Bd       *sql.DB   `json:"-"`
	NbCard   int       `json:"-"`
	Error    string    `json:"error,omitempty"`
	Response []Card    `json:"response,omitempty"`
}

type ShowCardRequest struct {
	Code  string  `json:"-"`
	Bd    *sql.DB `json:"-"`
	Image string  `json:"Image,omitempty"`
	Error string  `json:"error,omitempty"`
}
