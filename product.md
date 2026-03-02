# Les'Go - Product Requirements (Final Production Version)

## 1. Core Mission
**Les'Go** is a production-ready, CLI-based peer messaging system built in Go. It operates without a traditional UI, using pure terminal interaction, and prioritizes privacy by using in-memory-only server storage and device-based identity.

## 2. Functional Requirements

### 2.1 Device Identity
- **Permanent ID**: On first run, the client generates a random 10-digit numeric ID.
- **Persistence**: The ID is stored locally in `device.txt` and reused on subsequent runs.
- **Command**: `lesgo id` prints the device ID.

### 2.2 Connectivity
- **Online Mode**: `lesgo online` connects to the central relay server via WebSockets and registers the device ID.
- **In-Memory Storage**: The server stores online users and active chat sessions in memory only (no database).

### 2.3 Chat Request Flow
- **Connect Command**: `lesgo connect <10-digit-id>` initiates a chat request.
- **Request Forwarding**: The server forwards the request to the target device if online.
- **Peer Acceptance**: The recipient sees an interactive prompt: `"Incoming chat request from <ID>. Accept? (y/n)"`.
- **Active Mapping**: Upon acceptance, the server manages the session between both users in an `activeChats` map.

### 2.4 Real-time E2EE Messaging
- **Instant Delivery**: Messages typed in the CLI are sent instantly to the peer.
- **End-to-End Encryption (E2EE)**: RSA-2048 encryption is used for all peer-to-peer messages.
- **Privacy**: Public keys are automatically exchanged upon chat acceptance. The server only sees Base64-encoded ciphertext.
- **Concurrent Interaction**: Simultaneous goroutines handle socket listening and terminal input.
- **Display Format**: Messages are shown as `<ID>: message text`.

### 2.5 Disconnection Handling
- **Cleanup**: On disconnect, the server removes the user from `onlineUsers` and cleans up any `activeChats`.
- **Notification**: The remaining participant is notified when their peer leaves the chat.

## 3. Technical Requirements

### 3.1 Server
- **Framework**: Gorilla WebSocket.
- **Concurrency**: Thread-safe `sync.Map` for state management.
- **Zero-Knowledge**: No access to decryption keys or message plaintext.

### 3.2 Client
- **Interaction**: `bufio.Scanner` for terminal UI.
- **Security**: 
    - `crypto/rsa` for 2048-bit key pairs.
    - `encoding/base64` for safe transit of binary ciphertext.
- **Structure**:
    - `device.go`: Identity management.
    - `crypto.go`: RSA & Base64 logic.
    - `chat.go`: session flow & message loop.
    - `main.go`: CLI command routing.

## 4. CLI Commands
- `lesgo id`: Prints the device ID.
- `lesgo online`: Goes online and waits for incoming requests.
- `lesgo connect <id>`: Attempts to establish a secure chat with a peer.
- `lesgo exit`: Gracefully terminates the application.
