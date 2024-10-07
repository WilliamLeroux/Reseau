package database

import (
	"TP1/models"
	"database/sql"
	"fmt"
	"strings"
	"sync"
)

type CardDeckDB struct {
	db *sql.DB
}

// DbCreation S'assure que la base de donnée soit créer, sinon la crée
func DbCreation() (*CardDeckDB, error) {
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

func InsertDeck(c chan models.DeckRequest, db *CardDeckDB, wg *sync.WaitGroup) {
	defer close(c)
	defer wg.Done()
	dr := <-c

	tx, _ := db.db.Begin()

	query, err := db.db.Prepare(CREATE_DECK)
	if err != nil {
		println(err.Error())
		_ = tx.Rollback()
	}
	_, err = query.Exec(dr.DeckId, dr.Error, dr.CardAmount)
	if err != nil {
		_ = tx.Rollback()
	}
	_ = tx.Commit()
}

func InsertCards(c chan models.DeckRequest, db *CardDeckDB, wg *sync.WaitGroup) {
	defer close(c)
	defer wg.Done()
	dr := <-c
	var index = 1
	tx, _ := db.db.Begin()

	deckAmount := dr.CardAmount / 52
	if dr.Joker {
		deckAmount = dr.CardAmount / 54
		for i := 0; i <= deckAmount; i++ {
			query, err := db.db.Prepare(CREATE_CARDS)
			if err != nil {
				_ = tx.Rollback()
			}
			_, err = query.Exec(dr.DeckId, "0sc", "/static/0sc.svg", 0, "sc", index)
			if err != nil {
				println(err.Error())
				_ = tx.Rollback()
			}
			_ = tx.Commit()
			index++
			query, err = db.db.Prepare(CREATE_CARDS)
			if err != nil {
				_ = tx.Rollback()
			}
			_, err = query.Exec(dr.DeckId, "0dh", "/static/0dh.svg", 0, "dh", index)
			if err != nil {
				_ = tx.Rollback()
			}
			_ = tx.Commit()
			index++
		}

	}

	for i := 1; i <= deckAmount; i++ {
		for _, suit := range []string{"d", "s", "h", "c"} {
			for r := 1; r <= 13; r++ {
				code := fmt.Sprintf("%d%s", r, suit)
				image := fmt.Sprintf("/static/%s.svg", code)
				query, err := db.db.Prepare(CREATE_CARDS)
				if err != nil {
					println(err.Error())
					_ = tx.Rollback()
				}
				_, err = query.Exec(dr.DeckId, code, image, r, suit, index)
				_ = tx.Commit()
				index++
			}
		}
	}
}

func AddCards(c chan models.AddCard, db *CardDeckDB, wg *sync.WaitGroup, order int) {
	defer close(c)
	defer wg.Done()
	ac := <-c
	codes := strings.Split(ac.Code, ",")

	tx, _ := db.db.Begin()

	for _, card := range codes {

		query, err := db.db.Prepare(ADD_CARD)
		if err != nil {
			_ = tx.Rollback()
		}

		_, err = query.Exec(ac.DeckId, card, "/static/"+card+".svg", string(card[0]), string(card[1]), order)
		if err != nil {
			_ = tx.Rollback()
		}
		_ = tx.Commit()
		order++
	}

}

func CheckDeck(c chan string, db *CardDeckDB, wg *sync.WaitGroup, isGood *bool) {
	defer close(c)
	defer wg.Done()

	tx, err := db.db.Begin()
	if err != nil {
		fmt.Printf(err.Error())
	}
	query, err := db.db.Prepare(GET_DECK)
	if err != nil {
		_ = tx.Rollback()
	}

	result, err := query.Query(<-c)
	if err != nil {
		_ = tx.Rollback()
	}
	if result != nil {
		row, _ := result.Columns()
		if len(row) > 0 {
			*isGood = true
			return
		}
	}
	*isGood = false
}

func getCard(c chan models.CardResponse, db *CardDeckDB, wg *sync.WaitGroup) {
	defer close(c)
	defer wg.Done()
	//code := <-c

}

func GetPriority(c chan string, db *CardDeckDB, wg *sync.WaitGroup, hasPriority *bool) {
	defer close(c)
	defer wg.Done()
	tx, err := db.db.Begin()
	if err != nil {
		fmt.Printf(err.Error())
	}
	query, err := db.db.Prepare(HAS_PRIORITY)
	if err != nil {
		_ = tx.Rollback()
	}
	result, err := query.Query(<-c)
	if err != nil {
		_ = tx.Rollback()
		return
	}
	for result.Next() {
		var priority = 0
		err := result.Scan(priority)
		if err != nil {
			return
		}
		if priority != 0 {
			*hasPriority = true
		}
	}
}

func GetHighestPriority(c chan string, db *CardDeckDB, wg *sync.WaitGroup, order *int) {
	defer close(c)
	defer wg.Done()

	tx, err := db.db.Begin()
	if err != nil {
		fmt.Printf(err.Error())
	}
	query, err := db.db.Prepare(GET_HIGHEST_PRIORITY)
	if err != nil {
		_ = tx.Rollback()
	}
	result, err := query.Query(<-c)
	if err != nil {
		_ = tx.Rollback()
		return
	}

	for result.Next() {
		err = result.Scan(order)
		if err != nil {
			return
		}
	}
}

/*
func HasRemaining(c chan models.AddCard, db *CardDeckDB, wg *sync.WaitGroup, cardRemaining *bool) bool {
	defer wg.Done()

	r := <-c

	tx, _ := db.db.Begin()
	query, err := db.db.Prepare(HAS_REMAINING)
	if err != nil {
		_ = tx.Rollback()
	}
	_, err = query.Exec(r.DeckId, r.Code)
	if err != nil {
		_ = tx.Rollback()
	}
	_ = tx.Commit()
	return false
}*/
