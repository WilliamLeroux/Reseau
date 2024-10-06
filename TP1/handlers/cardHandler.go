package handlers

import (
	"TP1/database"
	"TP1/models"
	"TP1/utils"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// /deck/{deckid}/add/?cards={cards}
func AddMoreCards(w http.ResponseWriter, r *http.Request) {
	var wg sync.WaitGroup
	var errs = ""
	db, _ := database.DbCreation()
	var vars = mux.Vars(r)
	var deckId, _ = uuid.Parse(vars["deckid"])
	url := r.URL.Query().Get("cards")
	url = strings.Trim(url, "")
	cards := strings.Split(url, ",")
	dc := make(chan string)
	cc := make(chan models.AddCard)
	var ac models.AddCard

	for card := range cards {
		if !utils.CheckCard(cards[card]) {
			errs += "\nUne carte ne respecte pas la syntaxe (6h, si joker: 0sc)\n"
		}
	}
	if errs == "" {
		ac = models.AddCard{
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
		go database.CheckDeck(dc, db, &wg, &isGood)
		if isGood {
			errs += "Aucun paquet n'est lié a se deckId"
		}
		wg.Wait()
	}

	if errs == "" {
		wg.Add(1)
		go database.AddCards(cc, db, &wg)
		wg.Wait()
		_, _ = w.Write([]byte("Cartes ajoutées"))
	} else {
		_, _ = w.Write([]byte(errs))
	}

}

// /deck/{deckid}/draw/{nbrCarte:1}
func Draw(w http.ResponseWriter, r *http.Request) {
	var wg sync.WaitGroup
	var deckId = mux.Vars(r)["deckid"]
	var isGood = false
	var cardDrew = false
	nbCard, ok := strconv.Atoi(mux.Vars(r)["nbCard"])
	db, _ := database.DbCreation()
	request := new(models.AddCard)
	dc := make(chan string)
	rc := make(chan models.AddCard)
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
	go database.CheckDeck(dc, db, &wg, &isGood)
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
				go database.HasRemaining(rc, db, &wg, &cardRemaining)
				wg.Wait()
				//if cardRemaining {
				//	getCard()
				//}
			}
		}()
		wg.Wait()
	}

}
