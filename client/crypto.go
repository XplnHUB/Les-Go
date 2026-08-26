package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
)

// --- RSA Section ---

// GenerateKeyPair creates a new RSA key pair for the session.
func GenerateKeyPair() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, 2048)
}

// PublicKeyToPEM converts an RSA public key to PEM format string.
func PublicKeyToPEM(pub *rsa.PublicKey) string {
	pubASN1, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return ""
	}
	// "PUBLIC KEY" is the conventional PEM label for PKIX/SPKI-encoded keys
	// (what MarshalPKIXPublicKey produces); "RSA PUBLIC KEY" is reserved for
	// PKCS#1, which this is not.
	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubASN1,
	})
	return string(pubPEM)
}

// PEMToPublicKey converts a PEM format string back to an RSA public key.
func PEMToPublicKey(pubPEM string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pubPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block containing the public key")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return pub.(*rsa.PublicKey), nil
}

// EncryptWithRSA encrypts data (like an AES key) with a public key using RSA-OAEP.
func EncryptWithRSA(pub *rsa.PublicKey, data []byte) (string, error) {
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, data, nil)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptWithRSA decrypts a base64 ciphertext with a private key using RSA-OAEP.
func DecryptWithRSA(priv *rsa.PrivateKey, ciphertextB64 string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return nil, err
	}
	plaintext, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

// --- AES Section (AES-256-GCM) ---

// GenerateAESKey creates a 32-byte (256-bit) key for AES-256.
func GenerateAESKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

// EncryptAES encrypts plaintext using AES-256-GCM with a new nonce.
// Returns nonce + ciphertext as a base64 string.
func EncryptAES(plainText []byte, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Create a new nonce for every encryption
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Seal appends the ciphertext to the nonce
	ciphertext := aesGCM.Seal(nonce, nonce, plainText, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptAES decrypts a base64 string containing nonce + ciphertext using AES-256-GCM.
func DecryptAES(cipherTextB64 string, key []byte) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(cipherTextB64)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return aesGCM.Open(nil, nonce, ciphertext, nil)
}
