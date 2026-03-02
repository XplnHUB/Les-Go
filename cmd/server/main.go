package main

import (
	"log"
	"net/http"

	"github.com/XplnHUB/Les-Go/server/api"
	"github.com/XplnHUB/Les-Go/server/db"
	"github.com/XplnHUB/Les-Go/server/ws"
)

func main() {
	database := db.NewInMemoryDB()
	hub := ws.NewHub()

	go hub.Run()

	server := &api.Server{
		DB:  database,
		Hub: hub,
	}

	http.HandleFunc("/api/register", server.HandleRegister())
	http.HandleFunc("/api/login", server.HandleLogin())
	http.HandleFunc("/api/users/", server.HandleGetKey()) // /api/users/{username}/key
	http.HandleFunc("/api/messages/send", server.HandleSendMessage())
	http.HandleFunc("/api/messages/unread", server.HandleGetUnread())

	http.HandleFunc("/ws", server.HandleWS())

	log.Println("Starting server on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
