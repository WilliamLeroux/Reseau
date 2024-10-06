package database

const (
	DECK_TABLE = `CREATE TABLE IF NOT EXISTS Decks (
		deck_id BLOB PRIMARY KEY,
		error VARCHAR(255) NULL,
		remaining INTEGER DEFAULT 0
	 	);`

	CARD_TABLE = `CREATE TABLE IF NOT EXISTS Cards (
		cardId INTEGER PRIMARY KEY AUTOINCREMENT,
		deck_id BLOB NOT NULL,
		code CHARACTER(20) NOT NULL,
		image VARCHAR(255) NOT NULL,
		rank INT NOT NULL,
		suit CHARACTER(20) NOT NULL,
		index_Draw INTEGER DEFAULT 0,
		draw_date DATETIME DEFAULT NULL,
		priority_order INTEGER DEFAULT 0,
		FOREIGN KEY (deck_id) REFERENCES decks(deck_id)
		);`

	CREATE_DECK = "INSERT INTO Decks(deck_id, error, remaining) VALUES($deckId, $err, $cardAmount);"

	CREATE_CARDS = "INSERT INTO Cards(deck_id, code, image, rank, suit, remaining) VALUES($deckId, $code, $image, $rank, $suit, $remaining);"

	UPDATE_CARDS = "UPDATE Cards SET remaining = remaining + 1 WHERE deck_id == $deckId AND code == $code; UPDATE Decks SET remaining = remaining + 1 WHERE deck_id == $deckId;"

	GET_DECK = "SELECT COUNT(*) FROM Decks WHERE deck_Id = $deckId;"

	GET_CARD = "SELECT * FROM Cards WHERE code = $code AND deck_Id = $deckId; UPDATE Decks SET remaining = remaining - 1 WHERE deck_Id = $deckId;"

	HAS_REMAINING = "SELECT remaining FROM Cards WHERE deck_Id = $deckId AND code = $code;"
)
