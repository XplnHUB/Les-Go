# Les'Go

> Les’Go is a secure, terminal-based messaging system built in Go that provides authenticated, end-to-end encrypted communication between users, with real-time delivery, unread tracking, and encrypted message persistence.

## Overview

Les'Go is designed for developers who prefer speed, control, and minimal interfaces. It eliminates unnecessary UI layers and focuses on performance, security, and simplicity.

### Core Features

- Authenticated user accounts
- End-to-end encrypted messaging
- Real-time message delivery
- Unread message tracking
- Persistent chat history
- Fully terminal-based interface

## Architecture

```text
Client (CLI)  <---->  Server  <---->  Database
     |                  |                 |
 Encryption       Auth + Routing     Persistent Storage
```

### Components

#### Client
- Terminal-based user interface
- Handles encryption/decryption
- Sends and receives messages via WebSocket

#### Server
- Authentication and session management
- Message routing
- Real-time communication handling
- Secure message storage

#### Storage Layer
- Persists users and messages
- Enables history retrieval and unread tracking

## Security Model

- Token-based authentication (JWT)
- Password hashing using bcrypt
- End-to-end encrypted message exchange
- TLS-ready server configuration
- Optional encrypted message storage

## Message Storage

Messages are stored server-side to allow:
- Viewing past conversations
- Tracking unread messages
- Resuming sessions across restarts

**Supported databases:**
- PostgreSQL
- MongoDB
- SQLite (lightweight deployments)

## Tech Stack

- **Core:** Go
- **Networking:** `net/http`, `gorilla/websocket`
- **Security:** JWT, bcrypt
- **Database:** PostgreSQL / MongoDB / SQLite

## Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/XplnHUB/Les-Go.git
   cd Les-Go
   ```

2. **Build the server**
   ```bash
   go build -o server ./server
   ```

3. **Build the client**
   ```bash
   go build -o lesgo ./client
   ```

## Running the Application

1. **Start the server**
   ```bash
   ./server
   ```

2. **Start the client**
   ```bash
   ./lesgo
   ```

## Example CLI Commands

- `/register`
- `/login`
- `/send <username> <message>`
- `/history <username>`
- `/unread`
- `/logout`

## Roadmap

- [ ] Group chats
- [ ] File transfer support
- [ ] Offline message synchronization
- [ ] CLI themes and customization
- [ ] Self-hosted deployment scripts
- [ ] Peer-to-peer mode
- [ ] Multi-device support

## Contributing

Contributions are welcome! Please ensure code quality, security, and performance standards are maintained.

## License

MIT License