package main

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
)

const idFile = "device.txt"

// GetOrGenerateID retrieves the 10-digit ID from device.txt or generates a new one.
func GetOrGenerateID() (string, error) {
	data, err := os.ReadFile(idFile)
	if err == nil {
		id := strings.TrimSpace(string(data))
		if len(id) == 10 {
			return id, nil
		}
	}

	// Generate new 10-digit ID
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	id := fmt.Sprintf("%010d", r.Int63n(10000000000))

	err = os.WriteFile(idFile, []byte(id), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to save device ID: %w", err)
	}

	return id, nil
}
