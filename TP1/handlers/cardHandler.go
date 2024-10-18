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
	var mu sync.Mutex
	var vars = mux.Vars(r)
	var deckId, _ = uuid.Parse(vars["deckid"])
	url := r.URL.Query().Get("cards")
	url = strings.Trim(url, "")
	cards := strings.Split(url, ",")
	var addCard models.AddCard
	var db, _ = database.DbCreation()

	var c = make(chan models.AddCard)

	for card := range cards {
		if !utils.CheckCard(cards[card]) {
			addCard.Error = "Une carte ne respecte pas la syntaxe (6h, si joker: 0sc)"
		}
	}
	if addCard.Error == "" {
		addCard.DeckId = deckId
		addCard.Db = db
		addCard.NewCard = url

		mu.Lock()
		go database.AddCards(c)
		mu.Unlock()

		c <- addCard

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(<-c)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(addCard.Error)
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

// Shuffle Mélange les cartes
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

// ShowDrawCard Affiche les cartes pigé
func ShowDrawCard(w http.ResponseWriter, r *http.Request) {
	var mu sync.Mutex
	var vars = mux.Vars(r)
	var deckId, _ = uuid.Parse(vars["deckid"])
	var nbCard, _ = strconv.Atoi(vars["nbCard"])
	db, _ := database.DbCreation()
	cr := make(chan models.ShowDrawRequest)

	if nbCard > 0 {
		mu.Lock()
		go database.ShowDrawCard(cr)
		cr <- models.ShowDrawRequest{
			DeckId: deckId,
			Bd:     db,
			NbCard: nbCard,
		}
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(<-cr)
		if err != nil {
			return
		}
	}
}

// ShowUndrawCard Affiche les cartes pas encore pigé
func ShowUndrawCard(w http.ResponseWriter, r *http.Request) {
	var mu sync.Mutex
	var vars = mux.Vars(r)
	var deckId, _ = uuid.Parse(vars["deckid"])
	var nbCard, _ = strconv.Atoi(vars["nbCard"])
	db, _ := database.DbCreation()
	cr := make(chan models.ShowDrawRequest)

	if nbCard > 0 {
		mu.Lock()
		go database.ShowUndrawCard(cr)
		cr <- models.ShowDrawRequest{
			DeckId: deckId,
			Bd:     db,
			NbCard: nbCard,
		}
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(<-cr)
		if err != nil {
			return
		}
	}
}

// ShowCard Montre les cartes
func ShowCard(w http.ResponseWriter, r *http.Request) {
	var mu sync.Mutex
	var vars = mux.Vars(r)
	var code, _ = vars["code"]
	db, _ := database.DbCreation()
	c := make(chan models.ShowCardRequest)

	if code == "back" {
		ShowBack(w, r)
		return
	}

	mu.Lock()
	go database.GetImage(c)
	c <- models.ShowCardRequest{
		Code: strings.Split(code, ".")[0],
		Bd:   db,
	}
	mu.Unlock()
	image := <-c
	if image.Error != "" {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(image.Error)
		return
	}

	path := strings.Replace(image.Image, "/", "", 1)

	w.Header().Set("Content-Type", "image/svg+xml")
	http.ServeFile(w, r, path)
}

// ShowBack Affiche la carte de dos
func ShowBack(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml")
	http.ServeFile(w, r, "static/back.svg")
}
