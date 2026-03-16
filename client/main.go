package main

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/XplnHUB/Les-Go/protocol"
)

var Version = "v1.0.16"

func main() {
	var command string
	var targetID string

	if len(os.Args) < 2 {
		command = "online"
	} else {
		arg1 := os.Args[1]
		if arg1 == "-v" || arg1 == "--v" || arg1 == "-version" || arg1 == "--version" {
			fmt.Printf("Les'Go version %s\n", GetVersion())
			return
		}
		if len(arg1) == 10 && isNumeric(arg1) {
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

	// Generate RSA keys for this session (for AES key exchange)
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go StartHeartbeat(ctx, conn, myID)

	fmt.Printf("You are online as [%s]. Waiting for incoming requests...\n", myID)

	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("\nDisconnected from server.")
			break
		}

		var packet protocol.Packet
		if err := json.Unmarshal(p, &packet); err != nil {
			continue
		}

		if packet.Type == protocol.TypeConnectRequest {
			HandleIncomingRequest(myID, conn, packet, privKey, pubKeyPEM)
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go StartHeartbeat(ctx, conn, myID)

	fmt.Printf("Sending chat request to %s...\n", targetID)
	err := conn.WriteJSON(protocol.NewPacket(protocol.TypeConnectRequest, myID, targetID, ""))
	if err != nil {
		log.Fatalf("Failed to send request: %v", err)
	}

	var peerPubKey *rsa.PublicKey

	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Disconnected from server.")
			break
		}

		var packet protocol.Packet
		if err := json.Unmarshal(p, &packet); err != nil {
			continue
		}

		switch packet.Type {
		case protocol.TypeConnectAccept:
			// Initiator receives accept, waits for public key or sends ours?
			// Handshake: A req -> B accept -> B pubkey -> A pubkey -> A aeskey
			fmt.Printf("Request accepted by %s. Exchanging keys...\n", targetID)

		case protocol.TypePublicKey:
			if packet.From == targetID {
				pub, err := PEMToPublicKey(packet.Payload)
				if err != nil {
					fmt.Println("Error: Invalid peer public key.")
					return
				}
				peerPubKey = pub
				
				// Send our public key
				conn.WriteJSON(protocol.NewPacket(protocol.TypePublicKey, myID, targetID, pubKeyPEM))

				// Now we (the initiator) generate the AES key
				aesKey, err := GenerateAESKey()
				if err != nil {
					fmt.Println("Error: Failed to generate session key.")
					return
				}

				// Encrypt AES key with peer's RSA public key
				encryptedKeyB64, err := EncryptWithRSA(peerPubKey, aesKey)
				if err != nil {
					fmt.Println("Error: Failed to encrypt session key.")
					return
				}

				// Send AES key
				conn.WriteJSON(protocol.NewPacket(protocol.TypeAESKey, myID, targetID, encryptedKeyB64))

				// Start the chat
				session := NewChatSession(targetID, aesKey)
				StartChatSession(myID, conn, session)
				return
			}

		case protocol.TypeConnectReject:
			fmt.Printf("Request rejected by %s.\n", targetID)
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
		u = fmt.Sprintf("wss://%s/ws", serverAddr)
	} else {
		u = fmt.Sprintf("ws://%s/ws", serverAddr)
	}
	conn, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
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
	conn.WriteJSON(protocol.NewPacket(protocol.TypeRegister, myID, "", ""))
}

func GetVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
	}
	return Version
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
	fmt.Println("Les'Go CLI - " + Version)
	fmt.Println("Usage:")
	fmt.Println("  lesgo               Go online (default)")
	fmt.Println("  lesgo <id>          Connect to a peer (default)")
	fmt.Println("  lesgo id            Display your device ID")
	fmt.Println("  lesgo online        Go online and wait for requests")
	fmt.Println("  lesgo connect <id>  Connect to a peer")
	fmt.Println("  lesgo exit          Close application")
}
