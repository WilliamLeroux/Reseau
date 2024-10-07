package models

import "github.com/google/uuid"

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
	NbCard int
	Cards  []Card `json:"cards"`
}
