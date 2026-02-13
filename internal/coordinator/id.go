package coordinator

import (
	"crypto/rand"
	"fmt"
)

// generateID creates a short random node ID like "node-a1b2c3".
func generateID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("node-%x", b)
}
