package main

// ChatSession holds the security state for an active conversation.
type ChatSession struct {
	PeerID string
	AESKey []byte
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
