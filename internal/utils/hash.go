package utils

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/google/uuid"
)

// SHA256Checksum computes the SHA256 hash of the given content and returns it as a hex string
func SHA256Checksum(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

// GenerateUUID generates a new UUID v4 and returns it as a string
func GenerateUUID() string {
	return uuid.New().String()
}
