# Les-Go Device Connection Architecture

## Purpose

This document defines how **Les-Go** establishes a secure connection between two devices and
maintains a real-time encrypted chat session through a relay.

---

# Core Connection Model

Les-Go does **not** use localhost for production communication. Each client connects to a relay,
which forwards encrypted packets between the two ends of a chat. The relay never stores messages
and never decrypts them.

```text
Device A ─────┐
              │
            Relay
              │
Device B ─────┘
```

---

# Two relay implementations, one protocol

There are **two** interchangeable relay implementations in this repository, both speaking the
same packet protocol (`protocol/protocol.go`, frozen in [`protocol.md`](./protocol.md)):

| | Production default | Self-host option |
|---|---|---|
| **Implementation** | `worker.js` — a Cloudflare Worker + Durable Object | `server/main.go` — a standalone Go binary |
| **Deployed at** | `wss://lesgo.backend.xplnhub.tech/ws` (per `wrangler.toml`) | wherever you run it |
| **Deployment** | `wrangler deploy`, automated via `.github/workflows/deploy-worker.yml` on every push to `main` that touches `worker.js`/`wrangler.toml` | manual — `go run ./server/main.go`, the published Docker image, or a GoReleaser binary |
| **State** | In-memory inside one Durable Object instance | In-memory `sync.Map`s |
| **Also serves** | An embedded browser chat client at `/` (see below) | Nothing beyond `/ws` and `/health` |

Clients default to the Cloudflare Worker (`lesgo.backend.xplnhub.tech`) and fall back to
`localhost:80` if it's unreachable; `LESGO_SERVER` overrides this entirely (see
[`SETUP.md`](./SETUP.md)). **Both relays must behave identically** — that's enforced two ways:

1. `scripts/check_protocol_sync.js` runs in CI (`.github/workflows/go.yml`) and fails the build
   if the Go relay's packet-type constants and the Worker's `RELAY_PACKET_TYPES` ever diverge.
2. Both implementations independently enforce the same three behaviors described below
   (sender-identity binding, heartbeat-timeout cleanup with peer disconnect notification, and
   basic rate limiting) — see each source file for the concrete logic.

There is no shared runtime code between them (Go and JavaScript can't share a binary), so keeping
them behaviorally identical is a discipline the sync check and this document exist to support, not
something enforced by the type system.

---

# Connection Flow

## Step 1 — Client connects to the relay

```text
wss://lesgo.backend.xplnhub.tech/ws   (production, Cloudflare)
ws://localhost:80/ws                  (local Go relay)
```

## Step 2 — Device Registration

```bash
lesgo online
```

Client sends:

```json
{ "type": "register", "from": "1234567890" }
```

The relay records `1234567890 → this connection` and, from this point on, will only accept
further packets on this connection whose `from` is `1234567890` — see **Sender identity** below.

---

# Step 3 — Connection Request

```bash
lesgo connect 9876543210
```

```json
{ "type": "connect_request", "from": "1234567890", "to": "9876543210" }
```

If `9876543210` isn't currently registered, the relay replies to the *sender* with:

```json
{ "type": "error", "from": "relay", "to": "1234567890", "payload": "target_offline" }
```

---

# Step 4 — Target Device Accepts

```text
Incoming request from 1234567890
Accept? (y/n)
```

```json
{ "type": "connect_accept", "from": "9876543210", "to": "1234567890" }
```

The relay also records this pair as an active chat (`server/main.go`'s `activeChats` map /
`worker.js`'s `pairs` map), which is what lets it deliver a `disconnect` notification to the
right peer later.

---

# Step 5 — RSA Public Key Exchange

```json
{ "type": "public_key", "from": "1234567890", "payload": "BASE64_RSA_PUBLIC_KEY" }
```

---

# Step 6 — AES Session Key Establishment

The connect **initiator** generates the AES-256 session key, wraps it with the receiver's RSA
public key (RSA-OAEP), and sends it:

```json
{ "type": "aes_key", "from": "1234567890", "to": "9876543210", "payload": "RSA_ENCRYPTED_AES_KEY" }
```

---

# Step 7 — Real-Time Chat

```json
{ "type": "message", "from": "1234567890", "to": "9876543210", "payload": "AES_ENCRYPTED_MESSAGE", "message_id": "..." }
```

The relay forwards the packet unmodified; the receiving client decrypts it and replies with an
`ack` carrying the same `message_id`. If no `ack` arrives within a few seconds, the CLI client
prints a "no delivery confirmation" hint — it's a UX signal, not a delivery guarantee (there's no
retry or store-and-forward for offline peers).

---

# Sender Identity

Both relays bind a connection to the ID it registered with: any packet whose `from` doesn't match
the ID that connection actually sent in its `register` packet is dropped, not forwarded. This
stops one connected client from putting an arbitrary ID in `from` to impersonate another device's
presence, requests, or messages. It does **not** replace end-to-end encryption — a spoofed sender
still can't decrypt real message content without the right private key — it closes a separate,
lower-severity social-engineering-style gap (e.g. a forged `connect_request` appearing to come
from someone it didn't).

---

# Heartbeat & Dead-Session Cleanup

Every online client sends a `heartbeat` every 15 seconds (`client/heartbeat.go`).

```json
{ "type": "heartbeat", "from": "1234567890" }
```

Both relays track each connection's last-seen time and run a periodic sweep:

| | Go relay (`server/main.go`) | Cloudflare relay (`worker.js`) |
|---|---|---|
| Sweep interval | 30s (`time.Ticker`) | 30s (Durable Object Alarm) |
| Dead-session threshold | 45s since last message | 45s since last message |
| On timeout | Closes the connection, which triggers `handleDisconnect` | Calls `dropSession`, closing the socket directly |

Either way, a session that goes quiet for more than 45 seconds is dropped, and if it was in an
active chat, its peer receives a `disconnect` packet.

---

# Disconnection Handling

Whenever a connection closes — whether the client exited cleanly, the network dropped, or the
heartbeat timeout fired — the relay removes it from the online set and, if it was paired in an
active chat, notifies the peer:

```json
{ "type": "disconnect", "from": "1234567890" }
```

Both relay implementations do this identically (this used to only be true of the Go relay; see
`understanding.md` for that history if you're reading an older mirror of these docs).

---

# Rate Limiting

Both relays apply a simple fixed-window counter per connected client: more than 20
non-heartbeat, non-register packets in a 1-second window gets the excess packets dropped, with an
`error`/`rate_limited` reply sent back to the sender. This is a basic abuse guard, not a
production-grade limiter — see "Not yet built" below.

---

# The Embedded Web Client

`worker.js` also serves a full browser-based chat UI at `/` (with its JS at `/app.js`), using the
Web Crypto API for the same RSA-2048 + AES-256-GCM scheme as the CLI client. It talks to the same
relay over the same protocol. **Its handshake behavior differs subtly from the CLI client's**: on
receiving a peer's `public_key`, the browser client generates its own AES session key if it
doesn't have one yet, rather than only the connect-initiator doing so. Whether a full CLI↔browser
handshake reliably completes as a result has not been verified end-to-end — treat mixed CLI/browser
sessions as unverified until that's tested.

---

# Required Packet Structure

```go
type Packet struct {
    Type      string `json:"type"`
    From      string `json:"from"`
    To        string `json:"to"`
    Payload   string `json:"payload,omitempty"`
    Timestamp int64  `json:"timestamp,omitempty"`
    MessageID string `json:"message_id,omitempty"`
}
```

See [`protocol.md`](./protocol.md) for the full type list, including `error`.

---

# Security Rules

## Mandatory (implemented)

* Never store plaintext messages — neither relay ever does.
* Never store private keys server-side — keys are generated and held client-side only.
* Validate packet sender — enforced by both relays (see **Sender Identity** above).
* Limit invalid/excessive connection attempts — basic rate limiting is in place (see above); this
  is intentionally minimal and could be hardened further (Cloudflare's native rate-limiting rules
  would be the lowest-effort next step for the Worker).

---

# Not Yet Built (future scaling)

The following describes a **possible future direction**, not the current architecture. Nothing
below is implemented today:

## Self-hosted deployment behind a reverse proxy

If self-hosting `server/main.go` outside of Docker, a typical production stack would put Nginx (or
similar) in front of it for TLS termination and reverse proxying:

```text
Client → Nginx → Go Relay Server
```

Nothing in this repository automates that Nginx configuration — it would need to be set up by
whoever operates that deployment.

## Multi-Relay Scaling

The Cloudflare relay currently runs as a **single global Durable Object instance**
(`env.RELAY.idFromName("global-relay")`) — every user worldwide is served by the same object
instance, which is single-threaded and pinned to one location. That's a deliberate simplicity
trade-off (it avoids any cross-instance coordination problem) at the cost of being a scalability
ceiling and a single point of failure for the production backend. If that ever becomes a real
constraint, the options are:

```text
Load Balancer
↓
Relay Nodes (sharded by ID range/hash)
↓
Shared session registry (e.g. Redis, or multiple Durable Object instances)
```

This is explicitly **not** recommended to build preemptively — it adds real complexity (session
routing across shards) for a scaling problem the project doesn't currently have evidence of
facing.

---

# Final Goal

Les-Go connection should behave like:

* instant peer discovery
* secure handshake
* encrypted message relay
* zero persistence
* real-time terminal (or browser) chat
* consistent behavior regardless of which relay implementation is actually handling the session
