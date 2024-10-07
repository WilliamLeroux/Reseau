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

//TODO: /deck/{deckid}/shuffle

//TODO: /deck/{deckid}/show/0/{nbrCarte:1}

//TODO: /deck/{deckid}/show/1/{nbrCarte:1}

//TODO: /static/{code}.svg

//TODO: /static/back.svg

func requestHandler() {
	r := mux.NewRouter()
	r.HandleFunc("/deck/new/{nbDeck}/{jokers:false}", handlers.NewDeck).Methods("GET")
	r.HandleFunc("/deck/{deckid}/add", handlers.AddMoreCards).Methods("GET")
	r.HandleFunc("/deck/{deckid}/draw", handlers.Draw).Methods("GET")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func main() {
	_, err := database.DbCreation()
	if err == nil {
		fmt.Printf("DB Created\n")
	}
	requestHandler()
}
