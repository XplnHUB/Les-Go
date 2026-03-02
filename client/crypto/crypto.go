package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/nacl/box"
)

// KeyPair holds a generated public/private key pair
type KeyPair struct {
	PublicKey  *[32]byte
	PrivateKey *[32]byte
}

// GenerateKeyPair creates a new Curve25519 key pair for E2EE
func GenerateKeyPair() (*KeyPair, error) {
	pub, priv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	return &KeyPair{
		PublicKey:  pub,
		PrivateKey: priv,
	}, nil
}

// EncryptMessage encrypts a plaintext message for a specific recipient
func EncryptMessage(plaintext []byte, recipientPubKey *[32]byte, senderPrivKey *[32]byte) (string, error) {
	var nonce [24]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	encrypted := box.Seal(nonce[:], plaintext, &nonce, recipientPubKey, senderPrivKey)
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// DecryptMessage decrypts an encrypted message from a specific sender
func DecryptMessage(encryptedBase64 string, senderPubKey *[32]byte, recipientPrivKey *[32]byte) ([]byte, error) {
	encrypted, err := base64.StdEncoding.DecodeString(encryptedBase64)
	if err != nil || len(encrypted) < 24 {
		return nil, fmt.Errorf("invalid ciphertext format: %w", err)
	}

	var nonce [24]byte
	copy(nonce[:], encrypted[:24])

	plaintext, ok := box.Open(nil, encrypted[24:], &nonce, senderPubKey, recipientPrivKey)
	if !ok {
		return nil, fmt.Errorf("failed to decrypt message")
	}

	return plaintext, nil
}
