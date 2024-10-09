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
	"strings"
	"sync"
)

// /deck/{deckid}/add/?cards={cards}
func AddMoreCards(w http.ResponseWriter, r *http.Request) {
	var wg sync.WaitGroup
	var errs = ""
	Db, _ := database.DbCreation()
	var vars = mux.Vars(r)
	var deckId, _ = uuid.Parse(vars["deckid"])
	url := r.URL.Query().Get("cards")
	url = strings.Trim(url, "")
	cards := strings.Split(url, ",")
	dc := make(chan string)
	cc := make(chan models.AddCard)
	var ac models.AddCard
	var order = 0
	var db = models.CardDeckDB{
		Db: Db,
	}
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
	var mu sync.Mutex
	var deckId, _ = uuid.Parse(mux.Vars(r)["deckid"])
	nbCard, ok := strconv.Atoi(mux.Vars(r)["nbCard"])
	db, _ := database.DbCreation()
	cr := make(chan models.DrawCardRequest)

	if ok != nil {
		_, _ = w.Write([]byte("Erreur dans le nombre de carte demandé"))
		return
	}

	mu.Lock()
	go database.DrawCard(cr, db)
	cr <- models.DrawCardRequest{
		NbCard: nbCard,
		Reponse: models.CardResponse{
			Deck: models.DeckRequest{
				DeckId: deckId,
			},
			Cards: []models.Card{},
		},
	}
	mu.Unlock()
	response := <-cr
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(response.Reponse)
	if err != nil {
		return
	}
}

func Shuffle(w http.ResponseWriter, r *http.Request) {
	var mu sync.Mutex
	var vars = mux.Vars(r)
	var deckId, _ = uuid.Parse(vars["deckid"])
	db, _ := database.DbCreation()
	sc := make(chan models.ShuffleRequest)

	mu.Lock()
	go database.ShuffleDeck(sc)
	sc <- models.ShuffleRequest{
		DeckId: deckId,
		Db:     db,
	}
	mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(<-sc)
	if err != nil {
		return
	}
}
