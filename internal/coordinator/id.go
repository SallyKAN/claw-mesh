package coordinator

import (
	"crypto/rand"
	"errors"
	"fmt"
)

const maxIDRetries = 3

// generateID creates a random node ID like "node-a1b2c3d4e5f6a7b8".
// It retries up to maxIDRetries times on rand.Read failure.
func generateID() (string, error) {
	b := make([]byte, 8)
	var lastErr error
	for i := 0; i < maxIDRetries; i++ {
		if _, err := rand.Read(b); err != nil {
			lastErr = err
			continue
		}
		return fmt.Sprintf("node-%x", b), nil
	}
	return "", fmt.Errorf("generating node ID after %d attempts: %w", maxIDRetries, lastErr)
}

// generateToken creates a random token for per-node authentication.
func generateToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating token: %w", err)
	}
	return fmt.Sprintf("%x", b), nil
}

// generateUniqueID generates an ID that doesn't collide with existing keys.
// exists is a function that returns true if the ID is already taken.
func generateUniqueID(exists func(string) bool) (string, error) {
	for i := 0; i < maxIDRetries; i++ {
		id, err := generateID()
		if err != nil {
			return "", err
		}
		if !exists(id) {
			return id, nil
		}
	}
	return "", errors.New("failed to generate unique node ID: too many collisions")
}
