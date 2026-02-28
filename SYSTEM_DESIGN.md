# Les'Go - System Design Document

## 1. Introduction

**Les'Go** is a secure, terminal-based messaging system built in Go. It provides authenticated, end-to-end encrypted (E2EE) communication between users, with real-time delivery, unread message tracking, and encrypted message persistence. This document outlines the technical architecture, data flow, and components required to build the system.

## 2. High-Level Architecture

The system follows a standard **Client-Server** architecture over a secure network:

```text
+-------------------+          HTTPS / WSS          +-------------------+
|                   |   (REST & WebSockets)         |                   |
|   CLI Client      | <===========================> |     Server        |
|  (Bubbletea/Go)   |                               |    (Go API)       |
|                   |                               |                   |
+--------+----------+                               +--------+----------+
         |                                                   |
         | Local Storage                                     | TCP
         v                                                   v
+-------------------+                               +-------------------+
|                   |                               |                   |
|  Keys & Config    |                               |    Database       |
|  (SQLite/Files)   |                               | (Postgres/SQLite) |
+-------------------+                               +-------------------+
```

## 3. Core Components

### 3.1 Client Layer
- **UI Framework**: Utilizes a terminal UI library (e.g., `charmbracelet/bubbletea`) to handle the chat interface, input inputs, and navigation without leaving the terminal.
- **Crypto Engine**: Handles local key pair generation (e.g., RSA or Curve25519). Public keys are shared with the server, while private keys remain securely stored on the local machine (optionally protected by a passphrase).
- **Network Interface**:
  - **REST Client**: Used for one-off operations like `/register`, `/login`, and fetching user public keys.
  - **WebSocket Client**: Maintains a persistent connection to the server for receiving and sending real-time messages.
- **Local State**: Caches current active chat, recent history, and other users' public keys to minimize server calls.

### 3.2 Server Layer
- **API Service (REST)**: Handles user onboarding, authentication, and directory synchronization (finding other users and their public keys).
- **Connection Hub (WebSockets)**: 
  - Maintains a registry of active connections (`map[string]*websocket.Conn`).
  - Handles presence (online/offline status updates).
- **Message Router**: 
  - Intercepts incoming messages.
  - Checks if the recipient is online in the Connection Hub.
  - Submits the message directly to the recipient's WebSocket if online, or flags it as unread in the Database if offline.
- **Security Context**: Validates JWTs on both REST endpoint access and WebSocket connection establishment.

### 3.3 Storage Layer
- **Relational Structure**: Uses PostgreSQL (for production/scale) or SQLite (for single-binary localized deployments).
- **Message Persistence**: The server *only* stores encrypted blobs. It cannot read the chat messages. It only uses metadata (Sender, Receiver, Timestamp) to route and sync messages.

## 4. Data Model (Schema)

### `Users` Table
| Field | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID/String | Primary Key |
| `username` | String | Unique username |
| `password_hash`| String | Bcrypt hash for login |
| `public_key` | String | User's public key for E2EE |
| `created_at` | Timestamp | Account creation date |

### `Messages` Table
| Field | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID/String | Primary Key |
| `sender_id` | UUID/String | Foreign Key -> Users.id |
| `receiver_id` | UUID/String | Foreign Key -> Users.id |
| `encrypted_data`| Text | The E2E encrypted message payload |
| `is_read` | Boolean | Tracks unread status for offline delivery |
| `created_at` | Timestamp | Time message was handled by server |

*(Note: Depending on E2EE implementation like the Double Ratchet Algorithm, session states or pre-keys might require additional tables, but standard PKI can use a simpler model).*

## 5. Communication Protocols & Flows

### 5.1 Registration & Key Exchange Flow
1. **Client** generates a Private / Public key pair.
2. **Client** sends `POST /api/register` with `{ "username": "alice", "password": "...", "public_key": "..." }`.
3. **Server** hashes the password, stores the user and public key in the database, and returns a success status.

### 5.2 Sending a Message Flow
1. **Alice (Client)** wants to message **Bob**.
2. **Alice** queries **Bob's** public key via `GET /api/users/bob/key`.
3. **Alice** encrypts the message plaintext using **Bob's** public key.
4. **Alice** sends a WebSocket event:
   ```json
   {
     "type": "CHAT_MESSAGE",
     "payload": {
       "to": "bob",
       "data": "<encrypted_blob>"
     }
   }
   ```
5. **Server** receives the event, saves the `<encrypted_blob>` into the `Messages` table.
6. **Server** checks if **Bob** is online. If yes, forwards the event to Bob's WS connection. If no, the message remains `is_read=false`.

### 5.3 Offline Synchronization
1. **Bob** logs in and connects via WebSocket.
2. **Bob** requests unread messages: `GET /api/messages/unread`.
3. **Server** returns all unread `<encrypted_blobs>` for Bob.
4. **Bob (Client)** decrypts messages locally using his Private Key and updates the UI.
5. **Bob** acknowledges receipt: `POST /api/messages/ack`, and the server marks them `is_read=true`.

## 6. Scaling & Future Proofing
- **Stateless API**: The REST API can be heavily load-balanced as it relies entirely on the DB.
- **WebSocket Scaling**: Currently, the "Hub" is in-memory on a single Go process. To scale horizontally, a **Pub/Sub Broker (e.g., Redis)** must be introduced so that if Alice connects to Server A and Bob connects to Server B, a message from Alice to Bob is published via Redis and routed to Server B.

## 7. Recommended Tech Stack Choices
- **CLI Framework**: `charmbracelet/bubbletea` (for Elm-inspired state handling) + `charmbracelet/lipgloss` (for styling).
- **Go Web Framework**: `gin-gonic/gin` or standard `net/http` for API routes.
- **WebSocket**: `gorilla/websocket`.
- **Database ORM**: `gorm` or `sqlc` for type-safe database queries.
- **Security**: `golang.org/x/crypto/bcrypt` (passwords), `golang.org/x/crypto/nacl/box` (E2E Encryption).
