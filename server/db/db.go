package db

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           string
	Username     string
	PasswordHash string
	PublicKey    string
}

type Message struct {
	ID        string
	Sender    string
	Receiver  string
	Encrypted string
	IsRead    bool
	CreatedAt time.Time
}

type InMemoryDB struct {
	users    map[string]*User     // username -> user
	messages map[string][]Message // receiver -> slice of messages
	mu       sync.RWMutex
}

func NewInMemoryDB() *InMemoryDB {
	return &InMemoryDB{
		users:    make(map[string]*User),
		messages: make(map[string][]Message),
	}
}

func (db *InMemoryDB) CreateUser(username, password, publicKey string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	if existing, exists := db.users[username]; exists {
		existing.PublicKey = publicKey
		existing.PasswordHash = string(hash)
		return nil
	}

	db.users[username] = &User{
		ID:           uuid.NewString(),
		Username:     username,
		PasswordHash: string(hash),
		PublicKey:    publicKey,
	}
	return nil
}

func (db *InMemoryDB) Authenticate(username, password string) (*User, error) {
	db.mu.RLock()
	user, exists := db.users[username]
	db.mu.RUnlock()

	if !exists {
		return nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	return user, nil
}

func (db *InMemoryDB) GetPublicKey(username string) (string, error) {
	db.mu.RLock()
	user, exists := db.users[username]
	db.mu.RUnlock()

	if !exists {
		return "", errors.New("user not found")
	}

	return user.PublicKey, nil
}

func (db *InMemoryDB) StoreMessage(sender, receiver, encrypted string) (Message, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	msg := Message{
		ID:        uuid.NewString(),
		Sender:    sender,
		Receiver:  receiver,
		Encrypted: encrypted,
		IsRead:    false,
		CreatedAt: time.Now(),
	}

	db.messages[receiver] = append(db.messages[receiver], msg)
	return msg, nil
}

func (db *InMemoryDB) GetUnreadMessages(username string) []Message {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var unread []Message
	if msgs, exists := db.messages[username]; exists {
		for _, msg := range msgs {
			if !msg.IsRead {
				unread = append(unread, msg)
			}
		}
	}
	return unread
}

func (db *InMemoryDB) MarkMessageAsRead(receiver, messageID string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if msgs, exists := db.messages[receiver]; exists {
		for i := range msgs {
			if msgs[i].ID == messageID {
				msgs[i].IsRead = true
				return nil
			}
		}
	}
	return errors.New("message not found")
}
