package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // Allow all origins for dev
}

// Client wraps the websocket connection and user info
type Client struct {
	Hub      *Hub
	Conn     *websocket.Conn
	Username string
	Send     chan []byte
}

// Event structure for incoming/outgoing messages
type Event struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
	To      string `json:"to,omitempty"`
	From    string `json:"from,omitempty"`
}

// Hub maintains the set of active clients
type Hub struct {
	clients    map[string]*Client // Username -> Client
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[string]*Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			// If already connected, close old
			if oldClient, ok := h.clients[client.Username]; ok {
				close(oldClient.Send)
			}
			h.clients[client.Username] = client
			h.mu.Unlock()
			log.Printf("Client connected: %s", client.Username)

			// Send current online users to the new client
			online := h.GetOnlineUsers()
			event := Event{
				Type:    "ONLINE_USERS",
				Payload: strings.Join(online, ","),
			}
			data, _ := json.Marshal(event)
			client.Send <- data

			h.broadcastPresence(client.Username, "USER_ONLINE")

		case client := <-h.unregister:
			h.mu.Lock()
			if c, ok := h.clients[client.Username]; ok && c == client {
				delete(h.clients, client.Username)
				close(client.Send)
				log.Printf("Client disconnected: %s", client.Username)
				h.mu.Unlock() // Unlock before broadcasting to avoid deadlock if broadcastPresence needs lock
				h.broadcastPresence(client.Username, "USER_OFFLINE")
			} else {
				h.mu.Unlock()
			}

		case message := <-h.broadcast:
			// Simply broadcast to all for now (not used for E2EE DMs yet)
			h.mu.RLock()
			for _, client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client.Username)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Forward direct message to a specific user if online
func (h *Hub) Forward(to string, event Event) bool {
	h.mu.RLock()
	client, ok := h.clients[to]
	h.mu.RUnlock()

	if !ok {
		return false
	}

	data, err := json.Marshal(event)
	if err != nil {
		return false
	}

	select {
	case client.Send <- data:
		return true
	default:
		return false
	}
}

func (h *Hub) broadcastPresence(username, eventType string) {
	event := Event{
		Type:    eventType,
		Payload: username,
	}
	data, _ := json.Marshal(event)

	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, client := range h.clients {
		// Don't send status update to the user themselves for now,
		// or do, doesn't matter much for a TUI.
		select {
		case client.Send <- data:
		default:
			// If client is slow, we skip them for this broadcast
		}
	}
}

func (h *Hub) GetOnlineUsers() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	var online []string
	for username := range h.clients {
		online = append(online, username)
	}
	return online
}

// ServeWS handles websocket requests
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request, username string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{
		Hub:      h,
		Conn:     conn,
		Username: username,
		Send:     make(chan []byte, 256), // Buffered channel
	}

	client.Hub.register <- client

	// Start pump goroutines
	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		// In Les'Go, client -> server messages will usually be sent via REST (as defined in SYSTEM_DESIGN)
		// Or via WebSocket if we want to bypass REST. For this simple iteration, we'll route sends via REST
		// and use WS *primarily* for pushing incoming messages.
	}
}

func (c *Client) writePump() {
	defer c.Conn.Close()
	for {
		message, ok := <-c.Send
		if !ok {
			c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}

		w, err := c.Conn.NextWriter(websocket.TextMessage)
		if err != nil {
			return
		}
		w.Write(message)

		// Add queued chat messages to the current websocket message.
		n := len(c.Send)
		for i := 0; i < n; i++ {
			w.Write([]byte{'\n'})
			w.Write(<-c.Send)
		}

		if err := w.Close(); err != nil {
			return
		}
	}
}
