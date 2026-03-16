# Les-Go - AI Context

## Product Overview
**Les-Go** is a production-ready, peer-to-peer messaging system written in Go. It is designed for terminal-based users who prioritize privacy and simplicity. It features a centralized in-memory relay server that facilitates connections between clients without storing any persistent data.

## Core Mission
To provide a secure, anonymous, and lightweight "WhatsApp for Terminal" experience with zero-knowledge server infrastructure.

## Key Features
- **Pure CLI Interaction**: No TUI/GUI; interaction is through standard terminal input/output.
- **Persistent Device Identity**: Generates a unique 10-digit ID on the first run, stored locally in `device.txt`.
- **End-to-End Encryption (E2EE)**: Uses RSA-2048 encryption for all messages. Public keys are exchanged automatically upon chat acceptance.
- **In-Memory Relay Server**: The server only routes encrypted data and maintains active session mappings in memory. No database is used.
- **Anonymity**: No accounts or personal data are required or stored.

## Technical Stack
- **Languages**: Go (1.20+)
- **Communication**: WebSockets (via `github.com/gorilla/websocket`)
- **Security**: RSA-2048 (E2EE), Base64 encoding for transit.
- **Concurrency**: Goroutines for handling socket listening and terminal input.

## Project Structure
- `client/`: Core client logic.
  - `chat.go`: Manages chat sessions, input handling, and the message loop.
  - `crypto.go`: RSA encryption/decryption and key generation.
  - `device.go`: Identity generation and persistence.
  - `main.go`: CLI entry point and command routing.
- `server/`: Relay server logic.
  - `main.go`: WebSocket relay, session manager, and in-memory state.
- `README.md`: General overview and installation instructions.
- `product.md`: Detailed product requirements and specs.
- `SETUP.md`: Guide for multi-device and local network setup.

## CLI Commands
- `lesgo id`: Display the local device ID.
- `lesgo online`: Register with the relay server and wait for incoming requests.
- `lesgo connect <10-digit-id>`: Initiate a secure chat with another online peer.
- `lesgo exit`: Gracefully terminate the client.

## Development Context
- The project emphasizes "Zero-Knowledge" architecture.
- Any message content passing through the server is encrypted and inaccessible to the server.
- The system is designed to work out-of-the-box with a public relay or in isolated local networks.
