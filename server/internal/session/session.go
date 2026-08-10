package session

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidSession = errors.New("invalid session")
	ErrSessionExpired = errors.New("session expired")
)

// secretKey holds the HMAC secret key. Must be set via SetSecret() before use.
// In production, set IFORGE_SESSION_SECRET env var or rely on DB-stored random secret.
var secretKey []byte

// secret returns the HMAC secret key. Panics if SetSecret() was never called.
func secret() []byte {
	if secretKey == nil {
		panic("session: secret not set — call SetSecret() during app initialization")
	}
	return secretKey
}

// SetSecret sets the secret key (call during app init).
func SetSecret(s string) {
	if s != "" {
		secretKey = []byte(s)
	}
}

// Sign creates a signed session token: username|expiry.hmac
func Sign(username string, ttl time.Duration) string {
	expiry := time.Now().Add(ttl).UTC().Format(time.RFC3339)
	payload := username + "|" + expiry

	mac := hmac.New(sha256.New, secret())
	mac.Write([]byte(payload))
	signature := hex.EncodeToString(mac.Sum(nil))

	return payload + "." + signature
}

// Verify validates a signed session token and returns the username.
func Verify(token string) (string, error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return "", ErrInvalidSession
	}

	payload := parts[0]
	signature := parts[1]

	mac := hmac.New(sha256.New, secret())
	mac.Write([]byte(payload))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
		return "", ErrInvalidSession
	}

	// Parse username|expiry
	idx := strings.LastIndex(payload, "|")
	if idx < 0 {
		return "", ErrInvalidSession
	}

	username := payload[:idx]
	expiryStr := payload[idx+1:]

	expiry, err := time.Parse(time.RFC3339, expiryStr)
	if err != nil {
		return "", ErrInvalidSession
	}

	if time.Now().After(expiry) {
		return "", ErrSessionExpired
	}

	return username, nil
}

// GenerateRandomToken generates a cryptographically random hex token.
func GenerateRandomToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
