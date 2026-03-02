# Les'Go - Product Requirements

## 1. Core Mission
**Les'Go** is a secure, terminal-based messaging system designed for developers who prioritize speed, security, and minimal interfaces. It provides a robust, end-to-end encrypted (E2EE) communication platform that operates entirely within the terminal.

## 2. Functional Requirements

### 2.1 User Management & Authentication
- **User Registration**: Users must be able to register with a unique username and a secure password.
- **Key Generation**: Upon registration, the client must automatically generate a cryptographic key pair (Private/Public).
- **Public Key Storage**: The server must store the user's public key for encryption by other users.
- **Secure Login**: Users must be able to log in securely. Passwords must be hashed using `Bcrypt` before storage.
- **Session Management**: Authenticated sessions must be managed using JSON Web Tokens (JWT).

### 2.2 End-to-End Encryption (E2EE)
- **Client-Side Encryption**: All message content must be encrypted on the sender's client before being sent to the server.
- **Client-Side Decryption**: Only the recipient's client, holding the corresponding private key, should be able to decrypt the message content.
- **Private Key Security**: Private keys must never leave the user's local machine.
- **Public Key Retrieval**: The client must be able to fetch the public key of any registered user from the server.

### 2.3 Real-Time Messaging
- **WebSocket Communication**: The system must support real-time message delivery via persistent WebSocket connections.
- **Persistent Routing**: The server must route messages from the sender to the intended recipient if they are online.
- **Presence Tracking**: The system should track and reflect the online/offline status of users.

### 2.4 Message Persistence & Synchronization
- **Encrypted Storage**: The server must store encrypted message blobs to allow for offline delivery and history retrieval.
- **Unread Tracking**: The system must track unread messages for each user.
- **Offline Sync**: Upon logging in, the client should automatically fetch all unread messages from the server.
- **Message Acknowledgment**: The client must notify the server when a message has been successfully received and decrypted.

### 2.5 Terminal UI (CLI)
- **Interactive Interface**: A rich terminal UI (TUI) built using `bubbletea` for chatting, navigating, and managing contacts.
- **Command Support**: Support for slash commands (e.g., `/register`, `/login`, `/send`, `/history`, `/logout`).

## 3. Technical Requirements
- **Language**: Go (Golang) for both client and server.
- **Frameworks**: `charmbracelet/bubbletea` (TUI), `gin-gonic/gin` (API), `gorilla/websocket` (Real-time).
- **Database**: PostgreSQL (Production) or SQLite (Development/Local).
- **Security**: JWT for auth, Bcrypt for passwords, NaCl/Box or similar for E2EE.

## 4. Future Roadmap
- [ ] **Group Chats**: Support for encrypted multi-user conversations.
- [ ] **File Transfers**: Securely sending files within the terminal interface.
- [ ] **Custom Themes**: Ability to customize the TUI appearance.
- [ ] **Peer-to-Peer Mode**: Optional direct client-to-client communication.
- [ ] **Multi-Device Support**: Synchronizing message history across different machines.
