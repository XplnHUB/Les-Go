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

## Installation & Setup

### 1. For Developers (via Go)
If you have Go installed, you can install the client directly:
```bash
go install github.com/XplnHUB/Les-Go/client@latest
# Note: Binaries are moved to $(go env GOPATH)/bin
```

### 2. For End-Users (CLI Download)
You can download the pre-compiled binaries from the [Releases](https://github.com/XplnHUB/Les-Go/releases) page.

**Using GitHub CLI:**
```bash
# Download the latest client for your architecture
gh release download v1.0.1 -p "*darwin_arm64.tar.gz" # for Mac M1/M2/M3
tar -xzf lesgo_*.tar.gz
chmod +x lesgo
```

**Using curl (Direct):**
```bash
# Example for Mac Silicon
curl -L -O https://github.com/XplnHUB/Les-Go/releases/download/v1.0.1/lesgo_1.0.1_darwin_arm64.tar.gz
tar -xzf lesgo_1.0.1_darwin_arm64.tar.gz
./lesgo id
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