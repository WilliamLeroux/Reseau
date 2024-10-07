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

func NewDeck(w http.ResponseWriter, r *http.Request) {
	var wg sync.WaitGroup                            // Waitgroup
	var errs = ""                                    // Erreurs
	db, _ := database.DbCreation()                   // base de donnée
	var vars = mux.Vars(r)                           // Arguments
	nbDeck, err := strconv.Atoi(vars["nbDeck"])      // Nombre de paquet
	jokers, err := strconv.ParseBool(vars["jokers"]) // Joker inclu ou non
	var cardAmount = 52 * nbDeck                     // Nombre de carte
	var deckId = uuid.New()                          // id du paquet
	dc := make(chan models.DeckRequest)
	cc := make(chan models.DeckRequest) // DeckChannel

	if jokers {
		cardAmount += 2
	}
	dr := models.DeckRequest{
		DeckId:     deckId,
		CardAmount: cardAmount,
		Joker:      jokers,
		Error:      errs,
	}

	go func() {
		utils.CheckCreateDeckError(&errs, err, &nbDeck)

		dc <- dr
		cc <- dr
	}()

	if err == nil {
		wg.Add(1)
		go database.InsertDeck(dc, db, &wg)
		wg.Wait()

		wg.Add(1)
		go database.InsertCards(cc, db, &wg)
		wg.Wait()
	}

	err = json.NewEncoder(w).Encode(&dr)
	if err != nil {
		return
	}

}
