# Les'Go - WhatsApp for Terminal (CLI)

Les'Go is a production-ready, peer messaging system written in Go. It features a centralized in-memory relay server and terminal-based clients with End-to-End Encryption (E2EE).

## Features

- **Pure CLI Interaction**: No TUI, just pure terminal input/output.
- **Persistent Device Identity**: 10-digit unique ID generated on first run.
- **End-to-End Encryption (E2EE)**: RSA-2048 encryption for all messages.
- **In-Memory Relay**: Zero-knowledge server that only forwards encrypted data.
- **Anonymous**: No accounts, no database, no personal data stored.

## Project Structure

```text
.
├── client/              # CLI Client logic
│   ├── chat.go          # Chat session & input handling
│   ├── crypto.go        # RSA Encryption/Decryption utilities
│   ├── device.go        # Identity generation & persistence
│   └── main.go          # CLI Entry point & command routing
├── server/              # Relay Server logic
│   └── main.go          # WebSocket relay & session manager
├── README.md            # You are here
├── product.md           # Product requirements
├── todo.md              # Project status & development log
└── go.mod               # Dependencies
```

## Getting Started

### 1. Installation
Ensure you have Go installed, then clone the repository.

### 2. Run the Relay Server
Start the server in a terminal:
```bash
go run ./server/main.go
# Server listens on :8080
```

### 3. Run the Client
In separate terminals, you can run multiple clients:

**Display your ID:**
```bash
go run ./client/*.go id
```

**Go Online (Listen for requests):**
```bash
go run ./client/*.go online
```

**Connect to a Peer:**
```bash
go run ./client/*.go connect <Target_ID>
```

## Security (E2EE)
Les'Go uses RSA-2048 for end-to-end encryption. 
1.  On startup, the client generates a temporary public/private key pair.
2.  Public keys are exchanged automatically once a connection is accepted.
3.  Messages are encrypted locally and only decrypted by the recipient.
4.  The server only sees Base64-encoded ciphertext.

## Requirements
- Go 1.20+
- `github.com/gorilla/websocket`

## License
MIT