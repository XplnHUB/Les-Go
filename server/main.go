package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/XplnHUB/Les-Go/protocol"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Tunable in tests to keep them fast; production keeps these defaults.
var (
	heartbeatTimeout = 45 * time.Second
	cleanupInterval  = 30 * time.Second

	rateLimitMax    = 20
	rateLimitWindow = time.Second
)

// Client represents an online device and its connection state.
type Client struct {
	ID       string
	Conn     *websocket.Conn
	LastSeen time.Time
	Mu       sync.Mutex

	rateCount       int
	rateWindowStart time.Time
}

// allowRate reports whether the client is still within its rate limit,
// bumping its counter as a side effect. A simple fixed-window counter is
// enough to blunt basic spam/abuse without adding real complexity.
func (c *Client) allowRate() bool {
	c.Mu.Lock()
	defer c.Mu.Unlock()

	now := time.Now()
	if now.Sub(c.rateWindowStart) > rateLimitWindow {
		c.rateWindowStart = now
		c.rateCount = 0
	}
	c.rateCount++
	return c.rateCount <= rateLimitMax
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

		// Sender-identity binding: a connection may only ever speak as the ID
		// it registered with. This doesn't touch the E2EE payload itself, but
		// it stops one connected client from forging the `from` field to
		// impersonate presence/requests/messages from another device.
		if packet.Type == protocol.TypeRegister {
			if myID != "" && packet.From != myID {
				log.Printf("Rejected re-register attempt: connection %q tried to become %q", myID, packet.From)
				continue
			}
		} else if myID == "" || packet.From != myID {
			log.Printf("Rejected spoofed packet: claimed from=%q on connection registered as %q", packet.From, myID)
			continue
		}

		var client *Client
		if myID != "" {
			if c, ok := onlineUsers.Load(myID); ok {
				client = c.(*Client)
				client.Mu.Lock()
				client.LastSeen = time.Now()
				client.Mu.Unlock()
			}
		}

		if client != nil && packet.Type != protocol.TypeHeartbeat {
			if !client.allowRate() {
				log.Printf("Rate limit exceeded for %s", myID)
				conn.WriteJSON(protocol.NewPacket(protocol.TypeError, "relay", myID, "rate_limited"))
				continue
			}
		}

		switch packet.Type {
		case protocol.TypeRegister:
			myID = packet.From
			newClient := &Client{
				ID:       myID,
				Conn:     conn,
				LastSeen: time.Now(),
			}
			onlineUsers.Store(myID, newClient)
			log.Printf("User registered: %s", myID)

		case protocol.TypeHeartbeat:
			log.Printf("Heartbeat received from: %s", myID)

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
				conn.WriteJSON(protocol.NewPacket(protocol.TypeError, "relay", myID, "target_offline"))
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

// startCleanupLoop sweeps onlineUsers every cleanupInterval, closing any
// connection that's gone quiet for longer than heartbeatTimeout.
// handleDisconnect (and any peer notification) is triggered naturally when
// that closed connection's handleWS read loop exits.
//
// stop is nil in production (never fires, matching the previous
// run-forever behavior) and a real channel in tests, so a test's cleanup
// goroutine doesn't leak into later tests and race their state resets.
func startCleanupLoop(stop <-chan struct{}) {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			onlineUsers.Range(func(key, value interface{}) bool {
				client := value.(*Client)
				client.Mu.Lock()
				if now.Sub(client.LastSeen) > heartbeatTimeout {
					log.Printf("Cleaning up dead session: %s", client.ID)
					client.Conn.Close()
				}
				client.Mu.Unlock()
				return true
			})
		case <-stop:
			return
		}
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "time": time.Now().String()})
}

// newServeMux builds the HTTP handler set on a fresh mux (rather than
// registering onto http.DefaultServeMux) so it can be exercised directly
// with httptest in tests without leaking global handler state.
func newServeMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/ws", handleWS)
	return mux
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}
	addr := ":" + port

	// Start session cleanup loop (nil stop channel: runs for the process lifetime)
	go startCleanupLoop(nil)

	log.Printf("Relay Server starting on %s...", addr)
	if err := http.ListenAndServe(addr, newServeMux()); err != nil {
		log.Fatal(err)
	}
}
