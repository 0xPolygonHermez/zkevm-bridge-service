package utils

import (
	"crypto/sha256"
	"math/rand"
	"time"
)

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))] //nolint:gosec
	}
	return string(b)
}

// GenerateRandomHash generates a random hash.
func GenerateRandomHash() [sha256.Size]byte {
	rand.Seed(time.Now().UnixNano())
	rs := generateRandomString(10) //nolint:gomnd
	return sha256.Sum256([]byte(rs))
}
