package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	manager := NewManager()

	http.Handle("/", http.FileServer(http.Dir("./frontend")))
	http.HandleFunc("/createGame", manager.CreateGame)
	http.HandleFunc("/ws", manager.serveWS)

	deployed := os.Getenv("DEPLOYED")
	if deployed == "true" {
		log.Fatal(http.ListenAndServe(":8080", nil))
	} else {
		log.Fatal(http.ListenAndServeTLS(":8080", "server.crt", "server.key", nil))
	}
}
