package database

import (
	"TP1/models"
	"TP1/utils"
	"database/sql"
	"fmt"
	"math/rand"
	"strconv"
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

	return db, nil
}

// InsertDeck Ajoute un deck
func InsertDeck(c chan models.DeckRequest) {
	dr := <-c

	tx, _ := dr.Db.Begin()

	query, err := dr.Db.Prepare(CREATE_DECK)
	if err != nil {
		_ = tx.Rollback()
		dr.Error = err.Error()
		c <- dr
		return
	}
	defer query.Close()
	_, err = query.Exec(dr.DeckId, dr.Error, dr.CardAmount)
	if err != nil {
		_ = tx.Rollback()
		dr.Error = err.Error()
		c <- dr
		return
	}

	err = InsertCards(dr, tx)
	if err != nil {
		dr.Error = err.Error()
		dr.Error = err.Error()
		c <- dr
		return
	}
	_ = tx.Commit()
	c <- dr
}

// InsertCards Ajoute les cartes au deck
func InsertCards(dr models.DeckRequest, tx *sql.Tx) error {
	var index = 1

	deckAmount := dr.CardAmount / 52
	if dr.Joker {
		deckAmount = dr.CardAmount / 54
		for i := 1; i <= deckAmount; i++ {
			if err := insertCard(tx, dr.DeckId.String(), "0jr", "/static/0jr.svg", "sc", index, 0); err != nil {
				return fmt.Errorf("une carte joker n'a pas plus être ajouté: %w", err)
			}
			index++
			if err := insertCard(tx, dr.DeckId.String(), "0jn", "/static/0jn.svg", "dh", index, 0); err != nil {
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

// ajoute les cartes
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

// AddCards Ajoute des cartes que l'utilisateurs souhaitent ajouter
func AddCards(c chan models.AddCard) {
	cards := <-c
	var rank = 0
	var suit = ""

	CheckDeck(&cards)

	if cards.Error != "" {
		c <- cards
		return
	}

	GetHighestPriority(&cards)

	codes := strings.Split(cards.NewCard, ",")

	tx, err := cards.Db.Begin()
	if err != nil {
		cards.Error = err.Error()
		c <- cards
	}
	for _, card := range codes {

		query, err := cards.Db.Prepare(ADD_CARD)
		if err != nil {
			_ = tx.Rollback()
			cards.Error = err.Error()
			c <- cards
			return
		}

		if len(card) == 3 {
			if _, err = strconv.Atoi(card[0:2]); err == nil {
				rank, _ = strconv.Atoi(card[0:2])
				suit = card[2:]
			} else {
				rank, _ = strconv.Atoi(card[0:1])
				suit = card[1:]
			}
		} else {
			rank, _ = strconv.Atoi(card[0:1])
			suit = card[1:]
		}

		_, err = query.Exec(cards.DeckId, card, "/static/"+card+".svg", rank, suit, cards.Order)
		if err != nil {
			_ = tx.Rollback()
			cards.Error = err.Error()
			c <- cards
			return
		}
		_ = tx.Commit()

		cards.Card = append(cards.Card, models.Card{
			Code:  card,
			Image: "/static/" + card + ".svg",
			Rank:  rank,
			Suit:  suit,
		})
		cards.Order++
	}
	c <- cards
}

// GetHighestPriority Trouve la plus grande priorité
func GetHighestPriority(cards *models.AddCard) {

	tx, err := cards.Db.Begin()
	if err != nil {
		cards.Error = err.Error()
		return
	}
	query, err := cards.Db.Prepare(GET_HIGHEST_PRIORITY)
	if err != nil {
		_ = tx.Rollback()
		cards.Error = err.Error()
		return
	}
	result, err := query.Query(cards.DeckId)
	if err != nil {
		_ = tx.Rollback()
		cards.Error = err.Error()
		return
	}

	for result.Next() {
		err = result.Scan(&cards.Order)
		if err != nil {
			cards.Error = err.Error()
			break
		}
	}
	cards.Order++
	query.Close()
	result.Close()
}

// CheckDeck Vérifie que le deck existe
func CheckDeck(card *models.AddCard) {

	tx, err := card.Db.Begin()
	if err != nil {
		card.Error = err.Error()
		return
	}
	query, err := card.Db.Prepare(HAS_DECK)
	if err != nil {
		_ = tx.Rollback()
		card.Error = err.Error()
		return
	}

	result, err := query.Query(card.DeckId)
	if err != nil {
		_ = tx.Rollback()
		card.Error = err.Error()
		return
	}
	if result != nil {
		var res = 0
		for result.Next() {
			result.Scan(&res)
		}
		if res > 0 {
			return
		} else {
			card.Error = "le deck n'existe pas"
		}
	}
}

// DrawCard Pige une carte
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

// Trouve la carte à piger
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

// ShuffleDeck Mélange le deck
func ShuffleDeck(c chan models.ShuffleRequest) {
	var mu sync.Mutex
	shuffle := <-c

	var cardList []string

	mu.Lock()
	tx, err := shuffle.Db.Begin()
	if err != nil {
		shuffle.ErrorMsg = err.Error()
		c <- shuffle
		return
	}
	query, err := shuffle.Db.Prepare(GET_UNDRAWED_CARDS)
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
	_ = tx.Commit()
	mu.Unlock()
	for result.Next() {
		var card = ""
		err := result.Scan(&card)
		cardList = append(cardList, card)
		if err != nil {
			shuffle.ErrorMsg = err.Error()
			c <- shuffle
			return
		}
	}
	var indexList = utils.MakeRange(1, len(cardList))
	var index = 0
	shuffle.Remaining = len(cardList)
	for _, card := range cardList {

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
		println(len(indexList))
		if len(indexList) == 0 {
			c <- shuffle
			return
		} else {
			index = rand.Intn(len(indexList))
		}

		_, err = query.Exec(indexList[index], card)
		if err != nil {
			shuffle.ErrorMsg = err.Error()
			c <- shuffle
			return
		}
		indexList = RemoveIndex(indexList, index)
		_ = tx.Commit()
	}
	shuffle.Response = "Le paquet a été mélangé"

	c <- shuffle
}

// RemoveIndex Retire un index
func RemoveIndex(s []int, index int) []int {
	return append(s[:index], s[index+1:]...)
}

// ShowDrawCard Renvoie les informations des cartes pigé
func ShowDrawCard(c chan models.ShowDrawRequest) {
	cards := <-c

	tx, err := cards.Bd.Begin()
	if err != nil {
		cards.Error = err.Error()
		c <- cards
		return
	}

	query, err := tx.Prepare(GET_DRAW_CARD)
	if err != nil {
		cards.Error = err.Error()
		c <- cards
		return
	}

	result, err := query.Query(cards.DeckId)
	if err != nil {
		cards.Error = err.Error()
		c <- cards
		return
	}
	var index = 1
	for result.Next() {
		if index > cards.NbCard {
			break
		}
		var tempCard models.Card
		err := result.Scan(&tempCard.Code, &tempCard.Image, &tempCard.Rank, &tempCard.Suit, &tempCard.Date)
		if err != nil {
			cards.Error = err.Error()
			c <- cards
			return
		}
		cards.Response = append(cards.Response, tempCard)
		index++
	}
	_ = tx.Commit()
	c <- cards
}

// ShowUndrawCard Renvoie les informations des cartes pas encore pigé
func ShowUndrawCard(c chan models.ShowDrawRequest) {
	cards := <-c
	tx, err := cards.Bd.Begin()
	if err != nil {
		cards.Error = err.Error()
		c <- cards
		return
	}

	query, err := tx.Prepare(DRAW_CARD)
	if err != nil {
		cards.Error = err.Error()
		c <- cards
		return
	}

	result, err := query.Query(cards.DeckId)
	if err != nil {
		cards.Error = err.Error()
		c <- cards
		return
	}
	var index = 1
	for result.Next() {
		if index > cards.NbCard {
			break
		}
		var tempCard models.Card
		var id = 0
		err := result.Scan(&id, &tempCard.Code, &tempCard.Image, &tempCard.Rank, &tempCard.Suit)
		if err != nil {
			cards.Error = err.Error()
			c <- cards
			return
		}
		cards.Response = append(cards.Response, tempCard)
		index++
	}
	_ = tx.Commit()
	c <- cards
}

// GetImage Renvoie le chemin ou l'image est
func GetImage(c chan models.ShowCardRequest) {
	card := <-c
	tx, err := card.Bd.Begin()
	if err != nil {
		card.Error = err.Error()
		c <- card
		return
	}

	query, err := tx.Prepare(GET_IMAGE_PATH)
	if err != nil {
		card.Error = err.Error()
		c <- card
		return
	}

	result, err := query.Query(card.Code)
	if err != nil {
		card.Error = err.Error()
		c <- card
		return
	}

	for result.Next() {
		err := result.Scan(&card.Image)
		if err != nil {
			card.Error = err.Error()
			c <- card
			return
		}
		break
	}
	c <- card
}
