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

	CREATE_CARDS = "INSERT INTO Cards(deck_id, code, image, rank, suit, index_draw) VALUES($deckId, $code, $image, $rank, $suit, $index);"

	ADD_CARD = "INSERT INTO Cards(deck_id, code, image, rank, suit, priority_order) VALUES($deckId, $code, $image, $rank, $suit, $order); UPDATE Decks SET remaining = remaining + 1 WHERE deck_id == $deckId;"

	HAS_DECK             = "SELECT COUNT(*) FROM Decks WHERE deck_Id = $deckId;"
	GET_CARD_PRIORITY    = "SELECT code, image, rank, suit FROM Cards WHERE deck_id = $deckId AND priority_order = ?;"
	GET_PRIORITY         = `SELECT priority_order FROM Cards WHERE deck_id = $deckId AND priority_order > 0 ORDER BY priority_order;`
	GET_HIGHEST_PRIORITY = `SELECT MAX(priority_order) FROM Cards WHERE deck_id = $deckId;`
	HAS_PRIORITY         = `SELECT MIN(priority_order) FROM Cards WHERE deck_id = $deckId;`
	SET_PICKED_DATE      = "UPDATE Cards SET draw_date = date('now') WHERE deck_id = $deckId;"
	GET_DECK             = `SELECT * FROM Decks WHERE deck_id = ?;`
	//DRAW_CARD            = `SELECT code, image, rank, suit FROM Cards WHERE draw_date IS NULL AND priority_order != 0 AND deck_id = ? ORDER BY priority_order , index_draw;`
	DRAW_CARD            = `SELECT cardId, code, image, rank, suit FROM Cards WHERE draw_date IS NULL AND deck_id = ? ORDER BY CASE WHEN priority_order != 0 THEN priority_order ELSE index_draw END,index_draw;`
	UPDATE_DATE          = `UPDATE Cards SET draw_date = date('now'), priority_order = 0 WHERE cardId = $cardId;`
	UPDATE_REMAINING     = `UPDATE Decks SET remaining = remaining - 1 WHERE deck_id = $deckID;`
	GET_REMAINING        = `SELECT remaining FROM Decks WHERE deck_id = $deckID;`
	RESET_PRIORITY_ORDER = `UPDATE Cards SET priority_order = 0 WHERE deck_id = $deckID;`
	UPDATE_INDEX         = `UPDATE Cards SET index_draw = ? WHERE cardId = ?`
	GET_UNDRAWED_CARDS   = `SELECT cardId FROM Cards WHERE deck_id = ? AND draw_date IS NULL;`

	//HAS_REMAINING = "SELECT remaining FROM Cards WHERE deck_Id = $deckId AND code = $code;"
)
