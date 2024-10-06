package database

import (
	"database/sql"
)

const (
	DECK_TABLE = "CREATE TABLE IF NOT EXISTS Decks ( " +
		"deck_id BLOB PRIMARY KEY," +
		"error VARCHAR(255) NULL," +
		"remaining INTEGER DEFAULT 0)"

	CARD_TABLE = "CREATE TABLE IF NOT EXISTS Cards ( " +
		"cardId INTEGER PRIMARY KEY AUTOINCREMENT," +
		"deck_id BLOB NOT NULL," +
		"code CHARACTER(20) NOT NULL," +
		"image VARCHAR(255) NOT NULL," +
		"rank INT NOT NULL," +
		"suit CHARACTER(20) NOT NULL," +
		"remaining INTEGER DEFAULT 0," +
		"indexDraw INTEGER NULL);"
)

type CardDeckDB struct {
	db *sql.DB
}

// S'assure que la base de donnée soit créer, sinon la crée
func dbCreation() (*CardDeckDB, error) {
	db, _ := sql.Open("sqlite3", "CardDeck.db")

	if _, err := db.Exec(DECK_TABLE); err != nil {
		return nil, err
	}

	if _, err := db.Exec(CARD_TABLE); err != nil {
		return nil, err
	}

	return &CardDeckDB{
		db: db,
	}, nil
}
