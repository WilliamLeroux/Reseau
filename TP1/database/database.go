package database

import (
	"TP1/models"
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"sync"
)

// DbCreation S'assure que la base de donnée soit créer, sinon la crée
func DbCreation() (*sql.DB, error) {
	db, _ := sql.Open("sqlite3", "CardDeck.db")

	if _, err := db.Exec(DECK_TABLE); err != nil {
		return nil, err
	}

	if _, err := db.Exec(CARD_TABLE); err != nil {
		return nil, err
	}

	//return &models.CardDeckDB{
	//	Db: db,
	//}, nil
	return db, nil
}

func InsertDeck(c chan models.DeckRequest, db *sql.DB) {
	dr := <-c

	tx, _ := db.Begin()

	query, err := db.Prepare(CREATE_DECK)
	if err != nil {
		println(err.Error())
		_ = tx.Rollback()
	}
	defer query.Close()
	_, err = query.Exec(dr.DeckId, dr.Error, dr.CardAmount)
	if err != nil {
		_ = tx.Rollback()
	}

	err = InsertCards(dr, tx)
	if err != nil {
		dr.Error = err.Error()
	}
	_ = tx.Commit()
}

func InsertCards(dr models.DeckRequest, tx *sql.Tx) error {
	var index = 1

	deckAmount := dr.CardAmount / 52
	if dr.Joker {
		deckAmount = dr.CardAmount / 54
		for i := 0; i <= deckAmount; i++ {
			if err := insertCard(tx, dr.DeckId.String(), "0sc", "/static/0sc.svg", "sc", index, 0); err != nil {
				return fmt.Errorf("une carte joker n'a pas plus être ajouté: %w", err)
			}
			index++
			if err := insertCard(tx, dr.DeckId.String(), "0dh", "/static/0dh.svg", "dh", index, 0); err != nil {
				return fmt.Errorf("une carte joker n'a pas plus être ajouté: %w", err)
			}
			index++
		}
	}

	for i := 1; i <= deckAmount; i++ {
		for _, suit := range []string{"d", "s", "h", "c"} {
			for r := 1; r <= 13; r++ {
				code := fmt.Sprintf("%d%s", r, suit)
				image := fmt.Sprintf("/static/%s.svg", code)
				if err := insertCard(tx, dr.DeckId.String(), code, image, suit, index, r); err != nil {
					return fmt.Errorf("cette carte n'a pas plus être ajouté %s: %w", code, err)
				}
				index++
			}
		}
	}
	return fmt.Errorf("")
}

func insertCard(tx *sql.Tx, deckId, code, image, suit string, order int, rank int) error {
	query, err := tx.Prepare(CREATE_CARDS)
	if err != nil {
		return fmt.Errorf("une carte n'a pas plus être créer: %w", err)
	}
	defer query.Close()

	_, err = query.Exec(deckId, code, image, rank, suit, order)
	if err != nil {
		return fmt.Errorf("une carte n'a pas plus être créer: %w", err)
	}
	return nil
}

func AddCards(c chan models.AddCard, db models.CardDeckDB, wg *sync.WaitGroup, order int) {
	defer close(c)
	defer wg.Done()
	ac := <-c
	codes := strings.Split(ac.Code, ",")

	tx, _ := db.Db.Begin()

	for _, card := range codes {

		query, err := db.Db.Prepare(ADD_CARD)
		if err != nil {
			_ = tx.Rollback()
		}

		_, err = query.Exec(ac.DeckId, card, "/static/"+card+".svg", string(card[0]), string(card[1]), order) // TODO: Accepter 10, 11, 12, 13
		if err != nil {
			_ = tx.Rollback()
		}
		_ = tx.Commit()
		order++
	}

}

func CheckDeck(c chan string, db models.CardDeckDB, wg *sync.WaitGroup, isGood *bool) {
	defer close(c)
	defer wg.Done()

	tx, err := db.Db.Begin()
	if err != nil {
		fmt.Printf(err.Error())
	}
	query, err := db.Db.Prepare(HAS_DECK)
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

func DrawCard(c chan models.DrawCardRequest, db *sql.DB) {
	deck := <-c

	tx, err := db.Begin()
	if err != nil {
		deck.Reponse.Deck.Error = err.Error()
		return
	}

	query, err := db.Prepare(GET_DECK)
	if err != nil {
		deck.Reponse.Deck.Error = err.Error()
		return
	}

	result, err := query.Query(deck.Reponse.Deck.DeckId)
	if err != nil {
		deck.Reponse.Deck.Error = err.Error()
		return
	}

	for result.Next() {
		err := result.Scan(&deck.Reponse.Deck.DeckId, &deck.Reponse.Deck.Error, &deck.Reponse.Deck.CardAmount)
		if err != nil {
			deck.Reponse.Deck.Error = err.Error()
			return
		}
	}

	getCard(tx, &deck)

	_ = tx.Commit()
	c <- deck
}

func getCard(tx *sql.Tx, dcr *models.DrawCardRequest) {
	var card = 1
	query, err := tx.Prepare(DRAW_CARD)
	if err != nil {
		dcr.Reponse.Deck.Error = err.Error()
		return
	}
	defer query.Close()
	result, err := query.Query(dcr.Reponse.Deck.DeckId)
	if err != nil {
		dcr.Reponse.Deck.Error = err.Error()
		return
	}

	for result.Next() {
		if card > dcr.NbCard {
			break
		}
		var tempCard models.Card
		var cardId = 0
		err := result.Scan(&cardId, &tempCard.Code, &tempCard.Image, &tempCard.Rank, &tempCard.Suit)
		query, err := tx.Prepare(UPDATE_DATE)
		if err != nil {
			return
		}
		_, err = query.Exec(cardId)

		query, err = tx.Prepare(UPDATE_REMAINING)
		if err != nil {
			return
		}
		_, err = query.Exec(dcr.Reponse.Deck.DeckId)

		if err != nil {
			dcr.Reponse.Deck.Error = err.Error()
			return
		}
		dcr.Reponse.Cards = append(dcr.Reponse.Cards, tempCard)
		dcr.Reponse.Deck.CardAmount--
		card++
	}
}

func ShuffleDeck(c chan models.ShuffleRequest) {
	var mu sync.Mutex
	shuffle := <-c
	var indexList []int
	var remaining = 0
	var goodNumber = false

	/*
		_, err := shuffle.Db.Exec(`PRAGMA journal_mode = WAL`)
		if err != nil {
			log.Fatal("Failed to set WAL mode:", err)
		}

		_, err = shuffle.Db.Exec(`PRAGMA busy_timeout = 30000`)
		if err != nil {
			log.Fatal("Failed to set busy timeout:", err)
		}*/

	mu.Lock()
	tx, err := shuffle.Db.Begin()
	if err != nil {
		shuffle.ErrorMsg = err.Error()
		c <- shuffle
		return
	}

	query, err := tx.Prepare(GET_REMAINING)
	if err != nil {
		shuffle.ErrorMsg = err.Error()
		c <- shuffle
		return
	}

	result, err := query.Query(shuffle.DeckId)
	if err != nil {
		shuffle.ErrorMsg = err.Error()
		c <- shuffle
		return
	}

	for result.Next() {
		err := result.Scan(&remaining)
		if err != nil {
			return
		}
	}
	_ = tx.Commit()
	mu.Unlock()
	mu.Lock()
	tx, err = shuffle.Db.Begin()
	if err != nil {
		shuffle.ErrorMsg = err.Error()
		c <- shuffle
		return
	}
	query, err = shuffle.Db.Prepare(GET_UNDRAWED_CARDS)
	if err != nil {
		shuffle.ErrorMsg = err.Error()
		c <- shuffle
		return
	}

	result, err = query.Query(shuffle.DeckId)
	if err != nil {
		shuffle.ErrorMsg = err.Error()
		c <- shuffle
		return
	}
	_ = tx.Commit()
	mu.Unlock()
	for result.Next() {
		var card = ""
		err := result.Scan(&card)
		var index = 0
		goodNumber = false
		for !goodNumber {
			index = rand.Intn(remaining) + 1

			if len(indexList) == 0 {
				indexList = append(indexList, index)
				goodNumber = true
				break
			}

			for _, i := range indexList {
				if i == index {
					goodNumber = false
				} else {
					indexList = append(indexList, index)
					goodNumber = true
				}
			}
		}

		tx, err = shuffle.Db.Begin()
		if err != nil {
			shuffle.ErrorMsg = err.Error()
			c <- shuffle
			return
		}
		query, err = shuffle.Db.Prepare(UPDATE_INDEX)
		if err != nil {
			shuffle.ErrorMsg = err.Error()
			c <- shuffle
			return
		}
		_, err = query.Exec(index, card)
		if err != nil {
			shuffle.ErrorMsg = err.Error()
			c <- shuffle
			return
		}
		query.Close()
		tx.Commit()

	}
	shuffle.Response = "Le paquet a été mélangé"

	c <- shuffle
}

func GetPriority(c chan string, db models.CardDeckDB, wg *sync.WaitGroup, hasPriority *bool) {
	defer close(c)
	defer wg.Done()
	tx, err := db.Db.Begin()
	if err != nil {
		fmt.Printf(err.Error())
	}
	query, err := db.Db.Prepare(HAS_PRIORITY)
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

func GetHighestPriority(c chan string, db models.CardDeckDB, wg *sync.WaitGroup, order *int) {
	defer close(c)
	defer wg.Done()

	tx, err := db.Db.Begin()
	if err != nil {
		fmt.Printf(err.Error())
	}
	query, err := db.Db.Prepare(GET_HIGHEST_PRIORITY)
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
