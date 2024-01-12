package main

import (
	"log"
	"net/http"
)

func main() {
	manager := NewManager()

	http.Handle("/", http.FileServer(http.Dir("./frontend")))
	http.HandleFunc("/createGame", manager.CreateGame)
	http.HandleFunc("/ws", manager.serveWS)

	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}
