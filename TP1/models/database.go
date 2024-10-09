package models

import (
	"database/sql"
)

type DatabaseRequest struct {
	Db     *sql.DB
	DeckId string
}
type CardDeckDB struct {
	Db *sql.DB
}
