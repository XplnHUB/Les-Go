package main

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gorilla/websocket"
)

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

func main() {
	var command string
	var targetID string

	if len(os.Args) < 2 {
		// Default behavior: Go online
		command = "online"
	} else {
		arg1 := os.Args[1]
		if len(arg1) == 10 && isNumeric(arg1) {
			// If it looks like an ID, default to connect
			command = "connect"
			targetID = arg1
		} else {
			command = arg1
			if command == "connect" && len(os.Args) >= 3 {
				targetID = os.Args[2]
			}
		}
	}

	id, err := GetOrGenerateID()
	if err != nil {
		log.Fatalf("ID error: %v", err)
	}

	// Generate keys for this session
	privKey, err := GenerateKeyPair()
	if err != nil {
		log.Fatalf("Crypto error: %v", err)
	}
	pubKeyPEM := PublicKeyToPEM(&privKey.PublicKey)

	switch command {
	case "id":
		fmt.Printf("Device ID: %s\n", id)

	case "online":
		runOnline(id, privKey, pubKeyPEM)

	case "connect":
		if targetID == "" {
			fmt.Println("Please specify target ID: lesgo connect <id> or just lesgo <id>")
			return
		}
		runConnect(id, targetID, privKey, pubKeyPEM)

	case "help":
		printUsage()

	case "exit":
		os.Exit(0)

	default:
		printUsage()
	}
}

func runOnline(myID string, privKey *rsa.PrivateKey, pubKeyPEM string) {
	conn := connectToServer(myID)
	if conn == nil {
		return
	}
	defer conn.Close()

	fmt.Printf("You are online as [%s]. Waiting for incoming requests...\n", myID)

	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("\nDisconnected from server.")
			break
		}

		var msg Message
		if err := json.Unmarshal(p, &msg); err != nil {
			continue
		}

		if msg.Type == TypeChatRequest {
			HandleIncomingRequest(myID, conn, msg, privKey, pubKeyPEM)
			fmt.Printf("\nYou are online as [%s]. Waiting for incoming requests...\n", myID)
		}
	}
}

func runConnect(myID, targetID string, privKey *rsa.PrivateKey, pubKeyPEM string) {
	if len(targetID) != 10 {
		fmt.Println("Invalid ID. Target ID must be 10 digits.")
		return
	}

	conn := connectToServer(myID)
	if conn == nil {
		return
	}
	defer conn.Close()

	fmt.Printf("Sending chat request to %s...\n", targetID)
	err := conn.WriteJSON(Message{
		Type: TypeChatRequest,
		From: myID,
		To:   targetID,
	})
	if err != nil {
		log.Fatalf("Failed to send request: %v", err)
	}

	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Disconnected from server.")
			break
		}

		var msg Message
		if err := json.Unmarshal(p, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case TypeChatAccept:
			// Initiator receives acceptor's public key
			peerPubKey, err := PEMToPublicKey(msg.Data)
			if err != nil {
				fmt.Println("Invalid public key from peer.")
				return
			}
			// Send our public key to complete exchange
			conn.WriteJSON(Message{
				Type: TypeKeyExchange,
				From: myID,
				To:   targetID,
				Data: pubKeyPEM,
			})
			StartChatSession(myID, targetID, conn, privKey, peerPubKey)
			return
		case TypeChatReject:
			fmt.Printf("Request rejected by %s.\n", targetID)
			return
		case "ERROR":
			fmt.Printf("Error: %s\n", msg.Data)
			return
		}
	}
}

func connectToServer(myID string) *websocket.Conn {
	serverAddr := os.Getenv("LESGO_SERVER")
	if serverAddr == "" {
		serverAddr = "lesgo.xplnhub.com"
	}
	var u string
	if serverAddr == "localhost:8080" || serverAddr == "127.0.0.1:8080" {
		u = fmt.Sprintf("ws://%s/ws", serverAddr)
	} else if !containsPort(serverAddr) {
		// Default to wss for public domains if no port is specified
		u = fmt.Sprintf("wss://%s/ws", serverAddr)
	} else {
		u = fmt.Sprintf("ws://%s/ws", serverAddr)
	}
	conn, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		// Fallback logic for out-of-the-box experience
		if serverAddr == "lesgo.xplnhub.com" {
			fmt.Printf("Public server [%s] unavailable. Trying local fallback (localhost:80)...\n", u)
			uLocal := "ws://localhost:80/ws"
			connLocal, _, errLocal := websocket.DefaultDialer.Dial(uLocal, nil)
			if errLocal == nil {
				fmt.Println("Connected to local relay server.")
				registerAtServer(connLocal, myID)
				return connLocal
			}
		}

		fmt.Printf("Server unavailable at %s. Use export LESGO_SERVER=IP:PORT to change it.\n", u)
		return nil
	}

	registerAtServer(conn, myID)
	return conn
}

func registerAtServer(conn *websocket.Conn, myID string) {
	// Register immediately
	conn.WriteJSON(Message{
		Type: TypeRegister,
		From: myID,
	})
}

func containsPort(host string) bool {
	return strings.Contains(host, ":")
}

func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func printUsage() {
	fmt.Println("Les'Go CLI - Chat system for the terminal")
	fmt.Println("Usage:")
	fmt.Println("  lesgo               Go online (default)")
	fmt.Println("  lesgo <id>          Connect to a peer (default)")
	fmt.Println("  lesgo id            Display your device ID")
	fmt.Println("  lesgo online        Go online and wait for requests")
	fmt.Println("  lesgo connect <id>  Connect to a peer")
	fmt.Println("  lesgo exit          Close application")
}
