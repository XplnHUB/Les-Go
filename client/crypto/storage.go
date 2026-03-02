package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/crypto/pbkdf2"
)

type encryptedKeys struct {
	PublicKey  []byte `json:"public_key"`
	PrivateKey []byte `json:"private_key"`
	Salt       []byte `json:"salt"`
	Nonce      []byte `json:"nonce"`
}

func getStoragePath(username string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".lesgo", "keys", username+".json")
}

func deriveKey(password string, salt []byte) []byte {
	return pbkdf2.Key([]byte(password), salt, 4096, 32, sha256.New)
}

// SaveKeys encrypts and saves the key pair to a local file
func SaveKeys(username, password string, keys *KeyPair) error {
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return err
	}

	key := deriveKey(password, salt)
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}

	encryptedPriv := gcm.Seal(nil, nonce, keys.PrivateKey[:], nil)

	data := encryptedKeys{
		PublicKey:  keys.PublicKey[:],
		PrivateKey: encryptedPriv,
		Salt:       salt,
		Nonce:      nonce,
	}

	path := getStoragePath(username)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(data)
}

// LoadKeys loads and decrypts the key pair from a local file
func LoadKeys(username, password string) (*KeyPair, error) {
	path := getStoragePath(username)
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no keys found for user %s", username)
		}
		return nil, err
	}
	defer file.Close()

	var data encryptedKeys
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return nil, err
	}

	key := deriveKey(password, data.Salt)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	decryptedPriv, err := gcm.Open(nil, data.Nonce, data.PrivateKey, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt private key (wrong password?)")
	}

	var pub [32]byte
	var priv [32]byte
	copy(pub[:], data.PublicKey)
	copy(priv[:], decryptedPriv)

	return &KeyPair{
		PublicKey:  &pub,
		PrivateKey: &priv,
	}, nil
}
