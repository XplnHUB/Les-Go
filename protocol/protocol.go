package protocol

import "time"

// Packet Types
const (
	TypeRegister       = "register"
	TypeConnectRequest = "connect_request"
	TypeConnectAccept  = "connect_accept"
	TypeConnectReject  = "connect_reject"
	TypePublicKey      = "public_key"
	TypeAESKey         = "aes_key"
	TypeMessage        = "message"
	TypeACK            = "ack"
	TypeHeartbeat      = "heartbeat"
	TypeDisconnect     = "disconnect"
	// TypeError is sent by a relay back to the sender when a request cannot be
	// fulfilled (e.g. Payload "target_offline" or "rate_limited"). It is never
	// sent client-to-client.
	TypeError = "error"
)

// Packet represents the standard message structure.
type Packet struct {
	Type      string `json:"type"`
	From      string `json:"from"`
	To        string `json:"to"`
	Payload   string `json:"payload,omitempty"` // Base64 encoded data
	Timestamp int64  `json:"timestamp,omitempty"`
	MessageID string `json:"message_id,omitempty"`
}

// NewPacket creates a new packet with a timestamp.
func NewPacket(pType, from, to, payload string) Packet {
	return Packet{
		Type:      pType,
		From:      from,
		To:        to,
		Payload:   payload,
		Timestamp: time.Now().Unix(),
	}
}
