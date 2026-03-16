# Les-Go Packet Protocol Definition (v1.0)

This document freezes the message formats for Les-Go to ensure production stability.

## Packet Structure
Every packet is a JSON object with the following base structure:

```json
{
  "type": "TYPE",
  "from": "SENDER_ID",
  "to": "RECEIVER_ID",
  "payload": "BASE64_ENCODED_DATA",
  "timestamp": 1712341234,
  "message_id": "optional_id"
}
```

## Packet Types

### 1. `register`
Sent by client upon connection to register its ID with the relay.
- `from`: Client 10-digit ID.
- `to`: (empty)

### 2. `connect_request`
Sent by initiator to request a chat with a peer.
- `from`: Initiator ID.
- `to`: Target ID.

### 3. `connect_accept`
Sent by receiver to accept a chat request.
- `from`: Receiver ID.
- `to`: Initiator ID.

### 4. `connect_reject`
Sent by receiver to reject a chat request.
- `from`: Receiver ID.
- `to`: Initiator ID.

### 5. `public_key`
Exchanges RSA public key for AES key wrapping.
- `payload`: PEM-encoded RSA-2048 public key.

### 6. `aes_key`
The initiator sends the session AES key encrypted with the receiver's RSA public key.
- `payload`: RSA-OAEP encrypted 32-byte AES key.

### 7. `message`
Main chat message.
- `payload`: AES-256-GCM encrypted content.
- `format`: `nonce (12 bytes) + ciphertext`.
- `message_id`: Random unique ID for ACK tracking.

### 8. `ack`
Acknowledgement of a received message.
- `message_id`: The ID of the message being acknowledged.

### 9. `heartbeat`
Keep-alive signal sent every 15 seconds.
- `timestamp`: Current Unix timestamp.

### 10. `disconnect`
Notification that a peer has disconnected.

## Handshake Flow
1. **Initiator** sends `connect_request`.
2. **Receiver** sends `connect_accept`.
3. **Receiver** sends `public_key`.
4. **Initiator** sends `public_key`.
5. **Initiator** generates AES-256 key, sends `aes_key` (RSA-encrypted).
6. **Chat Session** established.
