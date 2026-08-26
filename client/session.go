package main

import "sync"

// ChatSession holds the security state for an active conversation.
type ChatSession struct {
	PeerID string
	AESKey []byte

	// pending tracks message IDs sent but not yet acknowledged by the peer.
	pending sync.Map // messageID (string) -> struct{}
}

// NewChatSession creates a new session.
func NewChatSession(peerID string, key []byte) *ChatSession {
	return &ChatSession{
		PeerID: peerID,
		AESKey: key,
	}
}

// Encrypt prepares a message for sending.
func (s *ChatSession) Encrypt(text string) (string, error) {
	return EncryptAES([]byte(text), s.AESKey)
}

// Decrypt processes an incoming message payload.
func (s *ChatSession) Decrypt(payload string) (string, error) {
	plain, err := DecryptAES(payload, s.AESKey)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// MarkPending records that messageID has been sent and is awaiting an ack.
func (s *ChatSession) MarkPending(messageID string) {
	if messageID == "" {
		return
	}
	s.pending.Store(messageID, struct{}{})
}

// AckReceived clears messageID from the pending set and reports whether it
// was still outstanding (false means it was already acked, or unknown).
func (s *ChatSession) AckReceived(messageID string) bool {
	if messageID == "" {
		return false
	}
	_, wasPending := s.pending.LoadAndDelete(messageID)
	return wasPending
}

// IsPending reports whether messageID is still awaiting an ack.
func (s *ChatSession) IsPending(messageID string) bool {
	_, ok := s.pending.Load(messageID)
	return ok
}
