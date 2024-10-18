package main

import (
	"TP1/database"
	"TP1/handlers"
	"fmt"
	_ "github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
)

func requestHandler() {
	r := mux.NewRouter()

	// Créer un nouveau paquet
	r.HandleFunc("/deck/new/{nbDeck}/{jokers:false}", handlers.NewDeck).Methods("GET")
	r.HandleFunc("/deck/new/{nbDeck}/{jokers:true}", handlers.NewDeck).Methods("GET")
	// Ajouter une carte
	r.HandleFunc("/deck/{deckid}/add", handlers.AddMoreCards).Methods("GET")
	// Piger une carte
	r.HandleFunc("/deck/{deckid}/draw/{nbCard}", handlers.Draw).Methods("GET")
	// Mélanger le paquet
	r.HandleFunc("/deck/{deckid}/shuffle", handlers.Shuffle).Methods("GET")
	// Afficher les cartes déjà pigé
	r.HandleFunc("/deck/{deckid}/show/0/{nbCard}", handlers.ShowDrawCard).Methods("GET")

	r.HandleFunc("/deck/{deckid}/show/1/{nbCard}", handlers.ShowUndrawCard).Methods("GET")
	r.HandleFunc("/static/{code}.svg", handlers.ShowCard).Methods("GET")

	log.Fatal(http.ListenAndServe(":8080", r))
}

func main() {
	_, err := database.DbCreation()
	if err == nil {
		fmt.Printf("DB Created\n")
	}

	requestHandler()
}
