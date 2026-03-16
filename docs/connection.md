# Les-Go Device Connection Architecture

## Purpose

This document defines how **Les-Go** establishes a secure connection between two devices and maintains a real-time encrypted chat session using a centralized relay server.

---

# Core Connection Model

Les-Go does **not** use localhost for production communication.

Each client must connect to a **public relay server**.

```text
Device A ─────┐
              │
         Relay Server
              │
Device B ─────┘
```

The relay server only forwards encrypted packets.
It does not store messages.
It does not decrypt messages.

---

# Production Connection Flow

## Step 1 — Start Relay Server

The relay server must listen on all interfaces.

```go
http.ListenAndServe("0.0.0.0:8080", nil)
```

Never bind only to localhost.

Incorrect:

```go
localhost:8080
```

Correct:

```go
0.0.0.0:8080
```

---

## Step 2 — Deploy Relay Publicly

Recommended deployment targets:

* AWS EC2
* Google Cloud VM
* DigitalOcean Droplet
* Render
* Railway

Production websocket endpoint:

```text
wss://relay.lesgo.chat/ws
```

Development endpoint:

```text
ws://PUBLIC_IP:8080/ws
```

---

# Step 3 — Device Registration

When client starts:

```bash
lesgo online
```

Client sends:

```json
{
  "type": "register",
  "from": "1234567890"
}
```

Server stores:

```text
1234567890 → active websocket connection
```

---

# Step 4 — Connection Request

User initiates:

```bash
lesgo connect 9876543210
```

Client sends packet:

```json
{
  "type": "connect_request",
  "from": "1234567890",
  "to": "9876543210"
}
```

---

# Step 5 — Target Device Accepts

Target receives:

```text
Incoming request from 1234567890
Accept? (y/n)
```

If accepted:

```json
{
  "type": "connect_accept",
  "from": "9876543210",
  "to": "1234567890"
}
```

---

# Step 6 — RSA Public Key Exchange

Both clients exchange public keys.

Packet:

```json
{
  "type": "public_key",
  "from": "1234567890",
  "payload": "BASE64_RSA_PUBLIC_KEY"
}
```

---

# Step 7 — AES Session Key Establishment

After public keys are exchanged:

* initiator generates AES key
* encrypts AES key using receiver RSA public key
* sends encrypted AES key

Packet:

```json
{
  "type": "aes_key",
  "from": "1234567890",
  "to": "9876543210",
  "payload": "RSA_ENCRYPTED_AES_KEY"
}
```

Receiver decrypts AES key using private RSA key.

---

# Step 8 — Real-Time Chat Begins

All messages now use AES encryption.

Packet:

```json
{
  "type": "message",
  "from": "1234567890",
  "to": "9876543210",
  "payload": "AES_ENCRYPTED_MESSAGE"
}
```

Relay server forwards packet directly.

---

# Real-Time Messaging Rules

## Client Requirements

Each client must run:

* one goroutine for websocket reading
* one goroutine for terminal input

Example:

```go
go listenIncoming()
go readTerminalInput()
```

---

## Server Requirements

Use concurrent-safe session registry.

```go
var clients = make(map[string]*websocket.Conn)
var mu sync.RWMutex
```

---

# Required Packet Structure

```go
type Packet struct {
    Type    string `json:"type"`
    From    string `json:"from"`
    To      string `json:"to"`
    Payload string `json:"payload"`
}
```

---

# Recommended Packet Types

* register
* connect_request
* connect_accept
* public_key
* aes_key
* message
* disconnect
* heartbeat

---

# Realtime Stability

## Heartbeat System

Every client sends heartbeat every 15 seconds.

```json
{
  "type": "heartbeat",
  "from": "1234567890"
}
```

Server removes dead sessions automatically.

---

# Disconnection Handling

If websocket closes:

* remove device from active sessions
* notify connected peer

Packet:

```json
{
  "type": "disconnect",
  "from": "1234567890"
}
```

---

# Production Network Layer

Recommended stack:

```text
Client → Nginx → Go Relay Server
```

Nginx handles:

* TLS
* websocket upgrade
* rate limiting
* reverse proxy

---

# Security Rules

## Mandatory

* Never store plaintext messages
* Never store private keys server-side
* Validate packet sender
* Limit invalid connection attempts

---

# Future Scaling

## Multi-Relay Scaling

Use Redis session registry.

```text
Load Balancer
↓
Relay Nodes
↓
Redis
```

This allows multiple relay servers.

---

# Final Goal

Les-Go connection should behave like:

* instant peer discovery
* secure handshake
* encrypted message relay
* zero persistence
* real-time terminal chat
