package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Message types
const (
	TypeRegister    = "REGISTER"
	TypeChatRequest = "CHAT_REQUEST"
	TypeChatAccept  = "CHAT_ACCEPT"
	TypeChatReject  = "CHAT_REJECT"
	TypeKeyExchange = "KEY_EXCHANGE"
	TypeMessage     = "MESSAGE"
	TypeDisconnect  = "DISCONNECT"
)

type Message struct {
	Type string `json:"type"`
	From string `json:"from"`
	To   string `json:"to"`
	Data string `json:"data"`
}

// Global state (In-memory only)
var (
	onlineUsers sync.Map // ID (string) -> *websocket.Conn
	activeChats sync.Map // ID (string) -> ID (string)
)

func handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Upgrade error: %v", err)
		return
	}
	defer conn.Close()

	var myID string

	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			if myID != "" {
				handleDisconnect(myID)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(p, &msg); err != nil {
			log.Printf("JSON unmarshal error: %v", err)
			continue
		}

		switch msg.Type {
		case TypeRegister:
			myID = msg.From
			onlineUsers.Store(myID, conn)
			log.Printf("User registered: %s", myID)

		case TypeChatRequest, TypeChatAccept, TypeChatReject, TypeKeyExchange, TypeMessage:
			targetID := msg.To
			log.Printf("Routing %s from %s to %s", msg.Type, msg.From, targetID)
			if targetConn, ok := onlineUsers.Load(targetID); ok {
				err := targetConn.(*websocket.Conn).WriteJSON(msg)
				if err != nil {
					log.Printf("Failed to write %s to %s: %v", msg.Type, targetID, err)
				}
				if msg.Type == TypeChatAccept {
					activeChats.Store(msg.From, msg.To)
					activeChats.Store(msg.To, msg.From)
					log.Printf("Chat started: %s <-> %s", msg.From, msg.To)
				}
			} else {
				log.Printf("Target %s not found (offline)", targetID)
				// Optionally notify sender that user is offline
				conn.WriteJSON(Message{
					Type: "ERROR",
					Data: fmt.Sprintf("User %s is offline", targetID),
				})
			}
		}
	}
}

func handleDisconnect(id string) {
	onlineUsers.Delete(id)
	log.Printf("User disconnected: %s", id)

	if peerID, ok := activeChats.Load(id); ok {
		activeChats.Delete(id)
		activeChats.Delete(peerID.(string))

		if peerConn, ok := onlineUsers.Load(peerID.(string)); ok {
			peerConn.(*websocket.Conn).WriteJSON(Message{
				Type: TypeDisconnect,
				From: id,
				Data: "Peer disconnected",
			})
		}
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}
	addr := ":" + port

	http.HandleFunc("/ws", handleWS)
	log.Printf("Relay Server starting on %s...", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
