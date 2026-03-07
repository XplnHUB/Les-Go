# Les'Go - WhatsApp for Terminal (CLI)

Les'Go is a production-ready, peer messaging system written in Go. It features a centralized in-memory relay server and terminal-based clients with End-to-End Encryption (E2EE).

## Features

- **Pure CLI Interaction**: No TUI, just pure terminal input/output.
- **Persistent Device Identity**: 10-digit unique ID generated on first run.
- **End-to-End Encryption (E2EE)**: RSA-2048 encryption for all messages.
- **In-Memory Relay**: Zero-knowledge server that only forwards encrypted data.
- **Anonymous**: No accounts, no database, no personal data stored.
- **Easy Setup**: Works out-of-the-box with public relay or private LAN (see [SETUP.md](./SETUP.md)).

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

### 1. From Source (Recommended for Developers)
If you want to run or modify the code:
```bash
# Clone the repository
git clone https://github.com/XplnHUB/Les-Go.git
cd Les-Go

# Install dependencies
go mod download

# Run the Server
go run ./server/main.go

# Run the Client
go run ./client/*.go online
```

### 2. Global Installation (via Go)
If you just want to use the client:
```bash
go install github.com/XplnHUB/Les-Go/client@latest
# The 'lesgo' command will be available if your GOBIN is in your PATH
```

### 3. Homebrew (macOS/Linux - Pop!_OS)
Les'Go works perfectly on Linux (including Pop!_OS). If you have Homebrew installed on Linux:
```bash
brew tap XplnHUB/tap
brew install lesgo
```

### 4. Direct Binary (Linux x86_64)
For Pop!_OS users without Homebrew:
```bash
# Download for Intel/AMD Linux
curl -L -O https://github.com/XplnHUB/Les-Go/releases/download/v1.0.15/lesgo_1.0.15_linux_amd64.tar.gz
tar -xzf lesgo_*.tar.gz
sudo mv lesgo /usr/local/bin/
```

**Using GitHub CLI:**
```bash
# Download the latest client for your architecture
gh release download v1.0.1 -p "*darwin_arm64.tar.gz"
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

## Multi-Device testing (Two Laptops)

To test Les'Go between two different machines:

### 1. Identify Server IP
On the machine running the server, find your local network IP:
- **Mac/Linux**: `ifconfig | grep "inet "`
- **Windows**: `ipconfig`
*Assume the IP is `192.168.1.10`.*

### 2. Configure Client on Second Laptop
Before running the client on the second machine, point it to the first machine's IP:
```bash
# Set the environment variable
export LESGO_SERVER=192.168.1.10:80

# Now run the client
go run ./client/*.go online
```

### 3. Connect as usual
Once both are online, use `lesgo connect <id>` as normal. The messages will be encrypted and routed across your local network.

## Requirements
- Go 1.20+
- `github.com/gorilla/websocket`

## Troubleshooting

### Error: "Server unavailable" (on another Mac)
1. **Firewall**: On the Server Mac, go to `System Settings -> Network -> Firewall`. Ensure it's off or allows `server_bin` / `go`.
2. **Network Isolation**: If you are on a public/office Wi-Fi (like "Guest" networks), they often block devices from talking to each other. Try using a mobile hotspot.
3.  **Ping Test**: From the *other* Mac, try to reach the server:
    ```bash
    ping 10.98.42.31
    ```
    If ping fails, the laptops cannot see each other on the network.
4. **Incorrect Server Address**: Ensure you ran `export LESGO_SERVER=10.98.42.31:80` before starting the client.

### Error: "address already in use" (port issue)
This means another server is already running on port 8080. 
To fix it, kill the existing process:
- **Mac/Linux**: `lsof -i :80 -t | xargs kill -9`
- **Windows**: `netstat -ano | findstr :80` (then kill the PID via Task Manager)

## Docker Usage (Server Only)

If you want to run the relay server on any machine without installing Go:

### 1. Build and Run locally
```bash
docker build -t lesgo-server -f server/Dockerfile .
docker run -d -p 80:80 lesgo-server
```

### 2. Pull from GHCR (Production)
Once you have pushed a tag and the GitHub Action has finished:
```bash
docker pull ghcr.io/xplnhub/lesgo-server:latest
docker run -d -p 80:80 ghcr.io/xplnhub/lesgo-server:latest
```

## License
MIT
