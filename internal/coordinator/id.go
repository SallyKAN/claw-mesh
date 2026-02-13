package coordinator

import (
	"crypto/rand"
	"fmt"
)

// generateID creates a random node ID like "node-a1b2c3d4e5f6a7b8".
func generateID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating node ID: %w", err)
	}
	return fmt.Sprintf("node-%x", b), nil
}
