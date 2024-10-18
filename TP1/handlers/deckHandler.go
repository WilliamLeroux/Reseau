package handlers

import (
	"TP1/database"
	"TP1/models"
	"TP1/utils"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
	"sync"
)

// NewDeck Crée un nouveau deck
func NewDeck(w http.ResponseWriter, r *http.Request) {
	var mu sync.Mutex
	var errs = ""                                    // Erreurs
	db, _ := database.DbCreation()                   // base de donnée
	var vars = mux.Vars(r)                           // Arguments
	nbDeck, err := strconv.Atoi(vars["nbDeck"])      // Nombre de paquet
	jokers, err := strconv.ParseBool(vars["jokers"]) // Joker inclu ou non
	var cardAmount = 52 * nbDeck                     // Nombre de carte
	var deckId = uuid.New()
	var c = make(chan models.DeckRequest)

	if jokers {
		cardAmount += 2
	}

	utils.CheckCreateDeckError(&errs, err, &nbDeck)

	dr := models.DeckRequest{
		DeckId:     deckId,
		CardAmount: cardAmount,
		Joker:      jokers,
		Error:      errs,
		Db:         db,
	}

	if err == nil {
		mu.Lock()

		go database.InsertDeck(c)
		mu.Unlock()
		c <- dr
		dr = <-c
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(dr)
	if err != nil {
		return
	}
}
