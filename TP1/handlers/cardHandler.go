package handlers

import (
	"TP1/database"
	"TP1/models"
	"TP1/utils"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"net/http"
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
	var order = 0

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
		go database.GetHighestPriority(dc, db, &wg, &order)
		wg.Wait()
		order++
		wg.Add(1)
		go database.AddCards(cc, db, &wg, order)
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
	var hasPriority = false
	var priorityCards = 0
	//nbCard, ok := strconv.Atoi(mux.Vars(r)["nbCard"])
	db, _ := database.DbCreation()
	//request := new(models.AddCard)
	dc := make(chan string)
	rc := make(chan string)

	//if ok != nil {
	//	nbCard = 1
	//}
	go func() {
		dc <- deckId
		rc <- deckId
	}()

	wg.Add(1)
	go database.CheckDeck(dc, db, &wg, &isGood)
	wg.Wait()

	if isGood {
		wg.Add(1)
		go database.GetPriority(rc, db, &wg, &hasPriority)
		wg.Wait()
	}

	if hasPriority {

	}

}
