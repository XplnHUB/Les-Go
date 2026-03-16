package main

import (
	"bufio"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/XplnHUB/Les-Go/protocol"
)

// StartChatSession manages the active chat interaction using E2EE AES-GCM.
func StartChatSession(myID string, conn *websocket.Conn, session *ChatSession) {
	fmt.Printf("\n--- Connected to %s (E2EE Enabled) ---\n", session.PeerID)
	fmt.Println("Type messages and press Enter. Type 'exit' to quit.")

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

			var packet protocol.Packet
			if err := json.Unmarshal(p, &packet); err != nil {
				continue
			}

			if packet.Type == protocol.TypeMessage {
				decrypted, err := session.Decrypt(packet.Payload)
				if err != nil {
					fmt.Printf("\r<System>: Failed to decrypt message from %s (Tampered?)\n> ", packet.From)
					continue
				}
				fmt.Printf("\r<%s>: %s\n> ", packet.From, decrypted)
				
				// Send ACK
				conn.WriteJSON(protocol.Packet{
					Type:      protocol.TypeACK,
					From:      myID,
					To:        packet.From,
					MessageID: packet.MessageID,
				})
			} else if packet.Type == protocol.TypeDisconnect {
				fmt.Printf("\nPeer %s disconnected.\n", packet.From)
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

		// Encrypt message with AES-GCM
		encrypted, err := session.Encrypt(text)
		if err != nil {
			fmt.Println("Error: Failed to encrypt message:", err)
			continue
		}

		err = conn.WriteJSON(protocol.NewPacket(protocol.TypeMessage, myID, session.PeerID, encrypted))
		if err != nil {
			fmt.Println("Error: Failed to send message:", err)
			break
		}
		fmt.Print("> ")
	}

	fmt.Println("Exiting chat...")
}

// HandleIncomingRequest handles chat requests and the 4-step secure handshake.
func HandleIncomingRequest(myID string, conn *websocket.Conn, initPacket protocol.Packet, privKey *rsa.PrivateKey, pubKeyPEM string) {
	fmt.Printf("\nIncoming chat request from %s. Accept? (y/n):\n", initPacket.From)

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response == "y" || response == "yes" {
		// 1. Accept request
		conn.WriteJSON(protocol.NewPacket(protocol.TypeConnectAccept, myID, initPacket.From, ""))

		// 2. Send our public key
		conn.WriteJSON(protocol.NewPacket(protocol.TypePublicKey, myID, initPacket.From, pubKeyPEM))

		fmt.Println("Exchanging keys and establishing session...")

		for {
			_, p, err := conn.ReadMessage()
			if err != nil {
				fmt.Println("Handshake failed: disconnected.")
				return
			}
			var packet protocol.Packet
			if err := json.Unmarshal(p, &packet); err != nil {
				continue
			}

			if packet.Type == protocol.TypePublicKey {
				pub, err := PEMToPublicKey(packet.Payload)
				if err != nil {
					fmt.Println("Error: Invalid peer public key.")
					return
				}
				_ = pub // peerPubKey would be used for initiator-side AES key generation
				log.Printf("Received public key from %s", packet.From)
			}

			if packet.Type == protocol.TypeAESKey {
				// Decrypt AES key with our private RSA key
				aesKey, err := DecryptWithRSA(privKey, packet.Payload)
				if err != nil {
					fmt.Println("Error: Failed to decrypt session key.")
					return
				}

				// Start the chat session
				session := NewChatSession(packet.From, aesKey)
				StartChatSession(myID, conn, session)
				return
			}
		}
	} else {
		conn.WriteJSON(protocol.NewPacket(protocol.TypeConnectReject, myID, initPacket.From, ""))
		fmt.Println("Request rejected.")
	}
}
