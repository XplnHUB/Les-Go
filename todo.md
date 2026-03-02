# Les'Go - Project TODO List

## 1. Backend Implementation (Server)

### 1.1 User Management & Auth
- [x] Implement User Registration API (`/api/register`)
- [x] Implement Password Hashing (Bcrypt)
- [x] Implement Secure Login API (`/api/login`)
- [x] Implement JWT Authentication Middleware
- [x] Implement Public Key Storage & Retrieval API (`/api/users/:username/key`)

### 1.2 Messaging & Real-Time
- [x] Implement WebSocket Hub for connection management
- [x] Implement Real-Time message forwarding via WebSockets
- [x] Implement Message Persistence for offline delivery (Encrypted blobs)
- [x] Implement Unread Message synchronization API (`/api/messages/unread`)
- [x] Implement Presence Tracking (Online/Offline status updates)
- [x] Implement Message Acknowledgment API (`/api/messages/ack`)

## 2. Client Implementation (CLI - Bubbletea)

### 2.1 Foundational UI
- [/] Implement Interactive Login/Register views
- [/] Implement Main Chat view with message input
- [ ] Implement Sidebar for contact list (optional but recommended)

### 2.2 Security & Crypto
- [x] Implement Client-Side Key Pair Generation (RSA/Curve25519)
- [/] Implement E2EE: Encrypting messages with recipient's public key
- [/] Implement E2EE: Decrypting messages with sender's private key
- [ ] Implement Secure Local Storage for Private Keys (Passphrase protected)

### 2.3 Network & Interaction
- [x] Implement REST client for API interaction
- [x] Implement WebSocket client for real-time updates
- [ ] Implement Command handling (Slash commands like `/send`, `/history`, `/logout`)
- [ ] Implement Auto-fetch of unread messages on login

## 3. Deployment & DevEx
- [ ] Configure PostgreSQL for production deployment
- [ ] Implement Docker configuration for easy setup
- [ ] Create detailed Installation & Usage guide in README.md

## 4. Future Roadmap (Post-v1)
- [ ] Group Messaging support
- [ ] File Transfer mechanism
- [ ] Custom TUI Themes
- [ ] Peer-to-Peer (P2P) mode
- [ ] Multi-device history synchronization
