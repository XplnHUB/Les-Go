package main

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
)

const idFile = "device.txt"

// GetOrGenerateID retrieves the 10-digit ID from LESGO_ID env, device.txt or generates a new one.
func GetOrGenerateID() (string, error) {
	// Check environment variable first (useful for testing multiple clients on one machine)
	if envID := os.Getenv("LESGO_ID"); envID != "" {
		if len(envID) == 10 {
			return envID, nil
		}
	}

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
