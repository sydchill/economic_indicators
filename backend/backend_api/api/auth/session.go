package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"time"
)

const sessionTTL = 7 * 24 * time.Hour

// GenerateSessionToken returns (rawToken, tokenHash).
// Store only the hash; return only the raw token to the client.
func GenerateSessionToken() (raw, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	return raw, HashSessionToken(raw), nil
}

// HashSessionToken is used both at creation and at verification time.
// SHA-256 is sufficient here because the input is 256 bits of random entropy
// (unlike passwords, which are low-entropy and need Argon2).
func HashSessionToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
