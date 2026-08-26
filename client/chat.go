package main

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/XplnHUB/Les-Go/protocol"
	"github.com/gorilla/websocket"
)

// ackTimeout is how long we wait for an ack before warning the user that a
// message may not have been delivered.
const ackTimeout = 8 * time.Second

// newMessageID returns a short random hex ID used to correlate a message
// with its ack.
func newMessageID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

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

			switch packet.Type {
			case protocol.TypeMessage:
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

			case protocol.TypeACK:
				session.AckReceived(packet.MessageID)

			case protocol.TypeError:
				fmt.Printf("\r<System>: Server error: %s\n> ", packet.Payload)

			case protocol.TypeDisconnect:
				fmt.Printf("\nPeer %s disconnected.\n", packet.From)
				return
			}
		}
	}()

	// Read stdin on its own goroutine so we're never blocked waiting on a
	// keypress when the peer disconnects or the socket errors out.
	inputCh := make(chan string)
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			inputCh <- scanner.Text()
		}
		close(inputCh)
	}()

	fmt.Print("> ")
	for {
		select {
		case <-done:
			fmt.Println("Exiting chat...")
			return

		case text, ok := <-inputCh:
			if !ok {
				fmt.Println("Exiting chat...")
				return
			}

			if strings.ToLower(text) == "exit" {
				fmt.Println("Exiting chat...")
				return
			}

			if text == "" {
				fmt.Print("> ")
				continue
			}

			encrypted, err := session.Encrypt(text)
			if err != nil {
				fmt.Println("Error: Failed to encrypt message:", err)
				continue
			}

			msgID := newMessageID()
			pkt := protocol.NewPacket(protocol.TypeMessage, myID, session.PeerID, encrypted)
			pkt.MessageID = msgID
			session.MarkPending(msgID)

			if err := conn.WriteJSON(pkt); err != nil {
				fmt.Println("Error: Failed to send message:", err)
				return
			}

			go warnIfUndelivered(session, msgID)

			fmt.Print("> ")
		}
	}
}

// warnIfUndelivered surfaces a hint if no ack arrives for msgID within
// ackTimeout. It's a best-effort UX signal, not a delivery guarantee.
func warnIfUndelivered(session *ChatSession, msgID string) {
	time.Sleep(ackTimeout)
	if session.IsPending(msgID) {
		fmt.Printf("\r<System>: No delivery confirmation for your last message yet.\n> ")
	}
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

			if packet.Type == protocol.TypeError {
				fmt.Printf("Server error during handshake: %s\n", packet.Payload)
				return
			}

			if packet.Type == protocol.TypeDisconnect {
				fmt.Printf("Peer %s disconnected before the handshake finished.\n", packet.From)
				return
			}
		}
	} else {
		conn.WriteJSON(protocol.NewPacket(protocol.TypeConnectReject, myID, initPacket.From, ""))
		fmt.Println("Request rejected.")
	}
}
