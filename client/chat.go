package main

import (
	"bufio"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/gorilla/websocket"
)

// StartChatSession manages the active chat interaction.
func StartChatSession(myID, peerID string, conn *websocket.Conn, privKey *rsa.PrivateKey, peerPubKey *rsa.PublicKey) {
	fmt.Printf("\n--- Connected to %s ---\n", peerID)
	fmt.Println("Type messages and press Enter. Type 'exit' to quit.")

	// Channel to signal exit
	done := make(chan struct{})

	// Read from WebSocket goroutine
	go func() {
		defer close(done)
		for {
			_, p, err := conn.ReadMessage()
			if err != nil {
				fmt.Println("\nDisconnected from server.")
				return
			}

			var msg Message
			if err := json.Unmarshal(p, &msg); err != nil {
				continue
			}

			if msg.Type == TypeMessage {
				// Decrypt message
				decrypted, err := Decrypt(privKey, msg.Data)
				if err != nil {
					fmt.Printf("\r<System>: Failed to decrypt message from %s\n> ", msg.From)
					continue
				}
				fmt.Printf("\r<%s>: %s\n> ", msg.From, decrypted)
			} else if msg.Type == TypeDisconnect {
				fmt.Printf("\nPeer %s disconnected.\n", msg.From)
				return
			}
		}
	}()

	// Read from Stdin goroutine
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")
	for scanner.Scan() {
		text := scanner.Text()
		if strings.ToLower(text) == "exit" {
			break
		}

		if text == "" {
			fmt.Print("> ")
			continue
		}

		// Encrypt message
		encrypted, err := Encrypt(peerPubKey, text)
		if err != nil {
			fmt.Println("Failed to encrypt message:", err)
			continue
		}

		err = conn.WriteJSON(Message{
			Type: TypeMessage,
			From: myID,
			To:   peerID,
			Data: encrypted,
		})
		if err != nil {
			fmt.Println("Failed to send message:", err)
			break
		}
		fmt.Print("> ")
	}

	fmt.Println("Exiting chat...")
}

// HandleIncomingRequest handles chat requests from other peers.
func HandleIncomingRequest(myID string, conn *websocket.Conn, initialMsg Message, privKey *rsa.PrivateKey, pubKeyPEM string) {
	fmt.Printf("\nIncoming chat request from %s. Accept? (y/n):\n", initialMsg.From)

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response == "y" || response == "yes" {
		// Accept and send our public key
		conn.WriteJSON(Message{
			Type: TypeChatAccept,
			From: myID,
			To:   initialMsg.From,
			Data: pubKeyPEM,
		})

		// Wait for initiator's public key
		fmt.Println("Exchanging keys...")
		for {
			_, p, err := conn.ReadMessage()
			if err != nil {
				fmt.Println("Failed to receive peer public key.")
				return
			}
			var msg Message
			if err := json.Unmarshal(p, &msg); err != nil {
				continue
			}
			if msg.Type == TypeKeyExchange {
				peerPubKey, err := PEMToPublicKey(msg.Data)
				if err != nil {
					fmt.Println("Invalid public key received.")
					return
				}
				StartChatSession(myID, initialMsg.From, conn, privKey, peerPubKey)
				return
			}
		}
	} else {
		conn.WriteJSON(Message{
			Type: TypeChatReject,
			From: myID,
			To:   initialMsg.From,
		})
		fmt.Println("Request rejected.")
	}
}
