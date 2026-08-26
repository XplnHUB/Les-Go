package main

import (
	"strings"
	"testing"
)

func TestRSAKeyPairAndPEMRoundTrip(t *testing.T) {
	priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	pemStr := PublicKeyToPEM(&priv.PublicKey)
	if !strings.Contains(pemStr, "-----BEGIN PUBLIC KEY-----") {
		t.Errorf("PEM output missing expected 'PUBLIC KEY' label: %s", pemStr)
	}

	pub, err := PEMToPublicKey(pemStr)
	if err != nil {
		t.Fatalf("PEMToPublicKey() error = %v", err)
	}
	if pub.N.Cmp(priv.PublicKey.N) != 0 {
		t.Errorf("decoded public key modulus does not match original")
	}
}

func TestPEMToPublicKeyRejectsGarbage(t *testing.T) {
	if _, err := PEMToPublicKey("not a pem block"); err == nil {
		t.Error("PEMToPublicKey(garbage) error = nil, want error")
	}
}

func TestRSAOAEPRoundTrip(t *testing.T) {
	priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	plaintext := []byte("a 32-byte AES-256 session key!!")
	ciphertextB64, err := EncryptWithRSA(&priv.PublicKey, plaintext)
	if err != nil {
		t.Fatalf("EncryptWithRSA() error = %v", err)
	}

	decrypted, err := DecryptWithRSA(priv, ciphertextB64)
	if err != nil {
		t.Fatalf("DecryptWithRSA() error = %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Errorf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

func TestRSADecryptRejectsBadBase64(t *testing.T) {
	priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}
	if _, err := DecryptWithRSA(priv, "!!!not-base64!!!"); err == nil {
		t.Error("DecryptWithRSA(bad base64) error = nil, want error")
	}
}

func TestRSADecryptWithWrongKeyFails(t *testing.T) {
	privA, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}
	privB, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	ciphertextB64, err := EncryptWithRSA(&privA.PublicKey, []byte("secret"))
	if err != nil {
		t.Fatalf("EncryptWithRSA() error = %v", err)
	}
	if _, err := DecryptWithRSA(privB, ciphertextB64); err == nil {
		t.Error("DecryptWithRSA(wrong key) error = nil, want error")
	}
}

func TestAESGCMRoundTrip(t *testing.T) {
	key, err := GenerateAESKey()
	if err != nil {
		t.Fatalf("GenerateAESKey() error = %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("GenerateAESKey() returned %d bytes, want 32", len(key))
	}

	plaintext := []byte("hello, encrypted world")
	ciphertextB64, err := EncryptAES(plaintext, key)
	if err != nil {
		t.Fatalf("EncryptAES() error = %v", err)
	}

	decrypted, err := DecryptAES(ciphertextB64, key)
	if err != nil {
		t.Fatalf("DecryptAES() error = %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Errorf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

func TestAESGCMUsesFreshNonceEachTime(t *testing.T) {
	key, err := GenerateAESKey()
	if err != nil {
		t.Fatalf("GenerateAESKey() error = %v", err)
	}

	c1, err := EncryptAES([]byte("same message"), key)
	if err != nil {
		t.Fatalf("EncryptAES() error = %v", err)
	}
	c2, err := EncryptAES([]byte("same message"), key)
	if err != nil {
		t.Fatalf("EncryptAES() error = %v", err)
	}
	if c1 == c2 {
		t.Error("two encryptions of the same plaintext produced identical ciphertext; nonce is not varying")
	}
}

func TestAESGCMRejectsTamperedCiphertext(t *testing.T) {
	key, err := GenerateAESKey()
	if err != nil {
		t.Fatalf("GenerateAESKey() error = %v", err)
	}
	ciphertextB64, err := EncryptAES([]byte("integrity matters"), key)
	if err != nil {
		t.Fatalf("EncryptAES() error = %v", err)
	}

	tampered := []byte(ciphertextB64)
	// Flip a character well past the nonce prefix so we're mutating the ciphertext body.
	idx := len(tampered) - 2
	if tampered[idx] == 'A' {
		tampered[idx] = 'B'
	} else {
		tampered[idx] = 'A'
	}

	if _, err := DecryptAES(string(tampered), key); err == nil {
		t.Error("DecryptAES(tampered) error = nil, want authentication failure")
	}
}

func TestAESGCMRejectsWrongKey(t *testing.T) {
	keyA, err := GenerateAESKey()
	if err != nil {
		t.Fatalf("GenerateAESKey() error = %v", err)
	}
	keyB, err := GenerateAESKey()
	if err != nil {
		t.Fatalf("GenerateAESKey() error = %v", err)
	}

	ciphertextB64, err := EncryptAES([]byte("for A's eyes only"), keyA)
	if err != nil {
		t.Fatalf("EncryptAES() error = %v", err)
	}
	if _, err := DecryptAES(ciphertextB64, keyB); err == nil {
		t.Error("DecryptAES(wrong key) error = nil, want error")
	}
}

func TestAESGCMRejectsShortCiphertext(t *testing.T) {
	key, err := GenerateAESKey()
	if err != nil {
		t.Fatalf("GenerateAESKey() error = %v", err)
	}
	if _, err := DecryptAES("dG9vc2hvcnQ=", key); err == nil {
		t.Error("DecryptAES(too short) error = nil, want error")
	}
}

func TestAESGCMRejectsBadBase64(t *testing.T) {
	key, err := GenerateAESKey()
	if err != nil {
		t.Fatalf("GenerateAESKey() error = %v", err)
	}
	if _, err := DecryptAES("!!!not-base64!!!", key); err == nil {
		t.Error("DecryptAES(bad base64) error = nil, want error")
	}
}
