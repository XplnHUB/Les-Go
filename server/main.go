package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/XplnHUB/Les-Go/protocol"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Client represents an online device and its connection state.
type Client struct {
	ID       string
	Conn     *websocket.Conn
	LastSeen time.Time
	Mu       sync.Mutex
}

// Global state (In-memory only)
var (
	onlineUsers sync.Map // ID (string) -> *Client
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

		var packet protocol.Packet
		if err := json.Unmarshal(p, &packet); err != nil {
			log.Printf("JSON unmarshal error: %v", err)
			continue
		}

		// Update LastSeen on every valid packet
		if client, ok := onlineUsers.Load(myID); ok && myID != "" {
			client.(*Client).Mu.Lock()
			client.(*Client).LastSeen = time.Now()
			client.(*Client).Mu.Unlock()
		}

		switch packet.Type {
		case protocol.TypeRegister:
			myID = packet.From
			client := &Client{
				ID:       myID,
				Conn:     conn,
				LastSeen: time.Now(),
			}
			onlineUsers.Store(myID, client)
			log.Printf("User registered: %s", myID)

		case protocol.TypeHeartbeat:
			if myID != "" {
				// LastSeen updated above switch
				log.Printf("Heartbeat received from: %s", myID)
			}

		case protocol.TypeConnectRequest, protocol.TypeConnectAccept, protocol.TypeConnectReject,
			protocol.TypePublicKey, protocol.TypeAESKey, protocol.TypeMessage, protocol.TypeACK:
			
			targetID := packet.To
			if targetCl, ok := onlineUsers.Load(targetID); ok {
				targetCl.(*Client).Mu.Lock()
				err := targetCl.(*Client).Conn.WriteJSON(packet)
				targetCl.(*Client).Mu.Unlock()
				
				if err != nil {
					log.Printf("Failed to forward %s to %s: %v", packet.Type, targetID, err)
				}
				
				if packet.Type == protocol.TypeConnectAccept {
					activeChats.Store(packet.From, packet.To)
					activeChats.Store(packet.To, packet.From)
					log.Printf("Chat session established: %s <-> %s", packet.From, packet.To)
				}
			} else {
				log.Printf("Target %s offline, cannot forward %s", targetID, packet.Type)
				// Optionally send error packet back
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

		if peerCl, ok := onlineUsers.Load(peerID.(string)); ok {
			peerCl.(*Client).Mu.Lock()
			peerCl.(*Client).Conn.WriteJSON(protocol.Packet{
				Type: protocol.TypeDisconnect,
				From: id,
			})
			peerCl.(*Client).Mu.Unlock()
		}
	}
}

func startCleanupLoop() {
	ticker := time.NewTicker(30 * time.Second)
	for range ticker.C {
		now := time.Now()
		onlineUsers.Range(func(key, value interface{}) bool {
			client := value.(*Client)
			client.Mu.Lock()
			if now.Sub(client.LastSeen) > 45*time.Second {
				log.Printf("Cleaning up dead session: %s", client.ID)
				client.Conn.Close()
				// handleDisconnect will be triggered by handleWS loop exit
			}
			client.Mu.Unlock()
			return true
		})
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}
	addr := ":" + port

	// Start session cleanup loop
	go startCleanupLoop()

	http.HandleFunc("/ws", handleWS)
	log.Printf("Relay Server starting on %s...", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
