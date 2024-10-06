package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	_ "github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

const DECK_TABLE = "CREATE TABLE IF NOT EXISTS Decks ( " +
	"deck_id BLOB PRIMARY KEY," +
	"error VARCHAR(255) NULL," +
	"remaining INTEGER DEFAULT 0)"

const CARD_TABLE = "CREATE TABLE IF NOT EXISTS Cards ( " +
	"cardId INTEGER PRIMARY KEY AUTOINCREMENT," +
	"deck_id BLOB NOT NULL," +
	"code CHARACTER(20) NOT NULL," +
	"image VARCHAR(255) NOT NULL," +
	"rank INT NOT NULL," +
	"suit CHARACTER(20) NOT NULL," +
	"remaining INTEGER DEFAULT 0," +
	"indexDraw INTEGER NULL);"

const PICKED_CARD_TABLE = "CREATE TABLE IF NOT EXISTS PickedCards ( " +
	"pickedCardId INTEGER PRIMARY KEY AUTOINCREMENT,"

const CREATE_DECK = "INSERT INTO Decks(deck_id, error, remaining) VALUES($deckId, $err, $cardAmount);"

const CREATE_CARDS = "INSERT INTO Cards(deck_id, code, image, rank, suit, remaining) VALUES($deckId, $code, $image, $rank, $suit, $remaining);"

const UPDATE_CARDS = "UPDATE Cards SET remaining = remaining + 1 WHERE deck_id == $deckId AND code == $code; UPDATE Decks SET remaining = remaining + 1 WHERE deck_id == $deckId;"

const GET_DECK = "SELECT COUNT(*) FROM Decks WHERE deckId = $deckId;"

const GET_CARD = "SELECT * FROM Cards WHERE code = $code AND deckId = $deckId; UPDATE Decks SET remaining = remaining - 1 WHERE deckId = $deckId;"

const HAS_REMAINING = "SELECT remaining FROM Cards WHERE deckId = $deckId AND code = $code;"

type CardDeckDB struct {
	db *sql.DB
}

type DeckRequest struct {
	DeckId     uuid.UUID `json:"deckId"`
	Error      string    `json:"error"`
	CardAmount int       `json:"cardAmount"`
	Joker      bool
}

type Response struct {
	DeckId    uuid.UUID `json:"deckId"`
	Error     string    `json:"error"`
	Card      []Card    `json:"card"`
	Remaining int       `json:"remaining"`
}

type Card struct {
	Code  string `json:"code"`
	Image string `json:"image"`
	Rank  int    `json:"rank"`
	Suit  string `json:"suit"`
}

type AddCard struct {
	Code   string
	DeckId uuid.UUID
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

// Crée un nouveau deck dans la bd
func insertDeck(c chan DeckRequest, db *CardDeckDB, wg *sync.WaitGroup) {
	defer close(c)
	defer wg.Done()
	dr := <-c

	tx, _ := db.db.Begin()

	query, err := db.db.Prepare(CREATE_DECK)
	if err != nil {
		_ = tx.Rollback()
	}
	_, err = query.Exec(dr.DeckId, dr.Error, dr.CardAmount)
	if err != nil {
		_ = tx.Rollback()
	}
	_ = tx.Commit()
}

// Ajoute des cartes dans la bd
func insertCards(c chan DeckRequest, db *CardDeckDB, wg *sync.WaitGroup) {
	defer close(c)
	defer wg.Done()
	dr := <-c

	tx, _ := db.db.Begin()

	deckAmount := dr.CardAmount / 52
	if dr.Joker {
		deckAmount = dr.CardAmount / 54
		query, err := db.db.Prepare(CREATE_CARDS)
		if err != nil {
			_ = tx.Rollback()
		}
		_, err = query.Exec(dr.DeckId, "0sc", "/static/0sc.svg", 0, "sc", deckAmount)
		if err != nil {
			_ = tx.Rollback()
		}
		_ = tx.Commit()
		_, err = query.Exec(dr.DeckId, "0dh", "/static/0dh.svg", 0, "dh", deckAmount)
		if err != nil {
			_ = tx.Rollback()
		}
		_ = tx.Commit()
	}

	for _, suit := range []string{"d", "s", "h", "c"} {
		for r := 1; r <= 13; r++ {
			code := fmt.Sprintf("%d%s", r, suit)
			image := fmt.Sprintf("/static/%s.svg", code)
			query, err := db.db.Prepare(CREATE_CARDS)
			if err != nil {
				_ = tx.Rollback()
			}
			_, err = query.Exec(dr.DeckId, code, image, r, suit, deckAmount)
			_ = tx.Commit()
		}
	}
}

// Vérifie s'il n'y a pas d'erreur pour la création d'un deck
// error: Pointeur de string qui contient les erreurs
// err: erreur lors de la prise des arguments
// deck: Pointeur de int qui contient le nombre de paquet demandeé
func checkCreateDeckError(error *string, err error, deck *int) {
	if *deck <= 0 {
		*error = "Le nombre de deck demandé est trop bas, 1 minimum"
	} else if *deck > 10 {
		*error = "Le nombre de deck demandé est trop haut, 10 maximum"
	}
	if err != nil {
		*error += err.Error()
	}
}

// Est appelé lors de cette requête: /deck/new/{nbDeck:1}/{jokers:false}
// Crée un mpaquets contenant nbDeck de paquet
// Inclue les jokers si jokers est true

func newDeck(w http.ResponseWriter, r *http.Request) {
	var wg sync.WaitGroup                            // Waitgroup
	var errs = ""                                    // Erreurs
	db, _ := dbCreation()                            // base de donnée
	var vars = mux.Vars(r)                           // Arguments
	nbDeck, err := strconv.Atoi(vars["nbDeck"])      // Nombre de paquet
	jokers, err := strconv.ParseBool(vars["jokers"]) // Joker inclu ou non
	var cardAmount = 52                              // Nombre de carte
	var deckId = uuid.New()                          // id du paquet
	dc := make(chan DeckRequest)                     // DeckChannel
	cc := make(chan DeckRequest)                     //CardChannel
	dr := new(DeckRequest)                           // DeckRequest

	if jokers {
		cardAmount += 2
	}

	go func() {
		dr.DeckId = deckId
		checkCreateDeckError(&errs, err, &nbDeck)
		dr.Error = errs
		dr.CardAmount = cardAmount * nbDeck
		dr.Joker = jokers
		dc <- *dr
		cc <- *dr
	}()

	if err == nil {
		wg.Add(1)
		go insertDeck(dc, db, &wg)
		wg.Wait()

		wg.Add(1)
		go insertCards(cc, db, &wg)
		wg.Wait()
	}

	err = json.NewEncoder(w).Encode(&dr)
	if err != nil {
		return
	}

}

func checkCard(card string) bool {
	if len(card) > 3 || len(card) < 1 {
		return false
	}

	rank := card[0]
	num, err := strconv.Atoi(string(rank))

	if err != nil || num < 0 || num > 13 {
		return false
	}

	if len(card) < 3 {
		suit := card[len(card)-1]
		if suit != 'd' && suit != 's' && suit != 'h' && suit != 'c' {
			return false
		}
	} else if len(card) == 3 {
		suit := card[1:]
		if suit != "sc" && suit != "dh" {
			return false
		}
	}
	return true
}

func checkDeck(c chan string, db *CardDeckDB, wg *sync.WaitGroup, isGood *bool) {
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
		}
	}
	*isGood = false
}

func addCards(c chan AddCard, db *CardDeckDB, wg *sync.WaitGroup) {
	defer close(c)
	defer wg.Done()
	ac := <-c
	codes := strings.Split(ac.Code, ",")
	for _, card := range codes {
		tx, _ := db.db.Begin()
		query, err := db.db.Prepare(UPDATE_CARDS)
		if err != nil {
			_ = tx.Rollback()
		}
		_, err = query.Exec(ac.DeckId, card)
		if err != nil {
			_ = tx.Rollback()
		}
		_ = tx.Commit()
	}

}

// /deck/{deckid}/add/?cards={cards}
func addMoreCards(w http.ResponseWriter, r *http.Request) {
	var wg sync.WaitGroup
	var errs = ""
	db, _ := dbCreation()
	var vars = mux.Vars(r)
	var deckId, _ = uuid.Parse(vars["deckid"])
	url := r.URL.Query().Get("cards")
	url = strings.Trim(url, "")
	cards := strings.Split(url, ",")
	dc := make(chan string)
	cc := make(chan AddCard)
	var ac AddCard

	for card := range cards {
		if !checkCard(cards[card]) {
			errs += "\nUne carte ne respecte pas la syntaxe (6h, si joker: 0sc)\n"
		}
	}
	if errs == "" {
		ac = AddCard{
			DeckId: deckId,
			Code:   url,
		}
	}

	go func() {
		dc <- deckId.String()
	}()

	go func() {
		cc <- ac
	}()

	if len(errs) < 0 {
		wg.Add(1)
		var isGood = false
		go checkDeck(dc, db, &wg, &isGood)
		if isGood {
			errs += "Aucun paquet n'est lié a se deckId"
		}
		wg.Wait()
	}

	if errs == "" {
		wg.Add(1)
		go addCards(cc, db, &wg)
		wg.Wait()
		_, _ = w.Write([]byte("Cartes ajoutées"))
	} else {
		_, _ = w.Write([]byte(errs))
	}

}

func getCard(c chan string, db *CardDeckDB, wg *sync.WaitGroup, cardDrew *bool) {
	defer close(c)
	defer wg.Done()

	//code := <-c

}

func hasRemaining(c chan AddCard, db *CardDeckDB, wg *sync.WaitGroup, cardRemaining *bool) bool {
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
}

// /deck/{deckid}/draw/{nbrCarte:1}
func draw(w http.ResponseWriter, r *http.Request) {
	var wg sync.WaitGroup
	var deckId = mux.Vars(r)["deckid"]
	var isGood = false
	var cardDrew = false
	nbCard, ok := strconv.Atoi(mux.Vars(r)["nbCard"])
	db, _ := dbCreation()
	request := new(AddCard)
	dc := make(chan string)
	rc := make(chan AddCard)
	var cardSuits = [14]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13"}
	var cardRanks = [4]string{"d", "s", "h", "c"}

	if ok != nil {
		nbCard = 1
		println(nbCard)
	}
	go func() {
		dc <- deckId
	}()

	wg.Add(1)
	go checkDeck(dc, db, &wg, &isGood)
	wg.Wait()

	if isGood {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for !cardDrew {
				request.Code = cardSuits[rand.Intn(len(cardSuits))] + cardRanks[rand.Intn(len(cardRanks))]
				rc <- *request
				var cardRemaining = false
				wg.Add(1)
				go hasRemaining(rc, db, &wg, &cardRemaining)
				wg.Wait()
				//if cardRemaining {
				//	getCard()
				//}
			}
		}()
		wg.Wait()
	}

}

//TODO: /deck/{deckid}/shuffle

//TODO: /deck/{deckid}/show/0/{nbrCarte:1}

//TODO: /deck/{deckid}/show/1/{nbrCarte:1}

//TODO: /static/{code}.svg

//TODO: /static/back.svg

func requestHandler() {
	r := mux.NewRouter()
	r.HandleFunc("/deck/new/{nbDeck:1}/{jokers:false}", newDeck).Methods("GET")
	r.HandleFunc("/deck/{deckid}/add", addMoreCards).Methods("GET")
	r.HandleFunc("/deck/{deckid}/draw/{nbrCarte}", draw).Methods("GET")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func main() {
	_, err := dbCreation()
	if err == nil {
		fmt.Printf("DB Created\n")
	}

	requestHandler()
}
