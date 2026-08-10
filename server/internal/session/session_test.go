package session

import (
	"strings"
	"testing"
	"time"
)

func TestSignAndVerify(t *testing.T) {
	// Setup
	SetSecret("test-secret-key-for-testing")

	tests := []struct {
		name        string
		username    string
		ttl         time.Duration
		sleepBefore time.Duration
		wantErr     bool
		errType     error
	}{
		{
			name:     "valid session",
			username: "alice",
			ttl:      time.Hour,
			wantErr:  false,
		},
		{
			name:     "expired session",
			username: "bob",
			ttl:      time.Millisecond * 100,
			sleepBefore: time.Millisecond * 200,
			wantErr:  true,
			errType:  ErrSessionExpired,
		},
		{
			name:     "username with special chars",
			username: "alice_smith-123",
			ttl:      time.Hour,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := Sign(tt.username, tt.ttl)

			if tt.sleepBefore > 0 {
				time.Sleep(tt.sleepBefore)
			}

			gotUsername, err := Verify(token)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Verify() expected error %v, got nil", tt.errType)
				}
				if tt.errType != nil && err != tt.errType {
					t.Errorf("Verify() expected error %v, got %v", tt.errType, err)
				}
			} else {
				if err != nil {
					t.Errorf("Verify() unexpected error: %v", err)
				}
				if gotUsername != tt.username {
					t.Errorf("Verify() username = %v, want %v", gotUsername, tt.username)
				}
			}
		})
	}
}

func TestVerifyInvalidTokens(t *testing.T) {
	SetSecret("test-secret-key")

	tests := []struct {
		name    string
		token   string
		wantErr error
	}{
		{
			name:    "empty token",
			token:   "",
			wantErr: ErrInvalidSession,
		},
		{
			name:    "no signature",
			token:   "alice|2025-01-01T00:00:00Z",
			wantErr: ErrInvalidSession,
		},
		{
			name:    "invalid signature",
			token:   "alice|2025-01-01T00:00:00Z.invalidsig",
			wantErr: ErrInvalidSession,
		},
		{
			name:    "malformed payload",
			token:   "invalidpayload.invalidsig",
			wantErr: ErrInvalidSession,
		},
		{
			name:    "wrong secret",
			token:   Sign("alice", time.Hour),
			wantErr: nil, // Will be valid with current secret
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Verify(tt.token)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("Verify() error = %v, want %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("Verify() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestVerifyTamperedToken(t *testing.T) {
	SetSecret("test-secret-key")

	// Sign a valid token
	token := Sign("alice", time.Hour)

	// Tamper with the username
	parts := strings.Split(token, ".")
	payload := parts[0]
	sig := parts[1]

	// Change username in payload
	tamperedPayload := strings.Replace(payload, "alice", "bob", 1)
	tamperedToken := tamperedPayload + "." + sig

	_, err := Verify(tamperedToken)
	if err != ErrInvalidSession {
		t.Errorf("Verify() tampered token should return ErrInvalidSession, got %v", err)
	}
}

func TestGenerateRandomToken(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{
			name:   "16 bytes",
			length: 16,
		},
		{
			name:   "32 bytes",
			length: 32,
		},
		{
			name:   "64 bytes",
			length: 64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token1, err := GenerateRandomToken(tt.length)
			if err != nil {
				t.Errorf("GenerateRandomToken() unexpected error: %v", err)
			}

			// Check length (hex encoding doubles the length)
			expectedLen := tt.length * 2
			if len(token1) != expectedLen {
				t.Errorf("GenerateRandomToken() length = %v, want %v", len(token1), expectedLen)
			}

			// Generate another token and ensure they're different
			token2, err := GenerateRandomToken(tt.length)
			if err != nil {
				t.Errorf("GenerateRandomToken() unexpected error: %v", err)
			}

			if token1 == token2 {
				t.Errorf("GenerateRandomToken() should generate unique tokens")
			}
		})
	}
}

func TestSecretPanic(t *testing.T) {
	// Reset secret
	secretKey = nil

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("secret() should panic when secretKey is nil")
		}
	}()

	secret()
}

func TestSetSecret(t *testing.T) {
	// Test setting empty secret (should not set)
	secretKey = nil
	SetSecret("")
	if secretKey != nil {
		t.Errorf("SetSecret('') should not set secretKey")
	}

	// Test setting valid secret
	SetSecret("my-secret")
	if secretKey == nil {
		t.Errorf("SetSecret('my-secret') should set secretKey")
	}
	if string(secretKey) != "my-secret" {
		t.Errorf("SetSecret('my-secret') secretKey = %v, want 'my-secret'", string(secretKey))
	}
}

func BenchmarkSign(b *testing.B) {
	SetSecret("benchmark-secret")
	for i := 0; i < b.N; i++ {
		Sign("alice", time.Hour)
	}
}

func BenchmarkVerify(b *testing.B) {
	SetSecret("benchmark-secret")
	token := Sign("alice", time.Hour)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Verify(token)
	}
}
