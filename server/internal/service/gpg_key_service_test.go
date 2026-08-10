package service

import (
	"testing"
	"time"
)

func TestGPGKeyService_AddGPGKey(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewGPGKeyService(db)

	// Add a GPG key
	key, err := service.AddGPGKey("testuser", "Test Key", "-----BEGIN PGP PUBLIC KEY BLOCK-----\ntest-key-data-12345678901234567890\n-----END PGP PUBLIC KEY BLOCK-----")
	if err != nil {
		t.Fatalf("AddGPGKey failed: %v", err)
	}

	if key.KeyID == 0 {
		t.Error("Expected KeyID to be set, got 0")
	}
	if key.UserName != "testuser" {
		t.Errorf("Expected UserName 'testuser', got '%s'", key.UserName)
	}
	if key.Title != "Test Key" {
		t.Errorf("Expected Title 'Test Key', got '%s'", key.Title)
	}
}

func TestGPGKeyService_ListGPGKeys(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewGPGKeyService(db)

	// Add multiple keys with time.Sleep to ensure unique keyIDs (UnixNano)
	service.AddGPGKey("testuser", "Key 1", "key1-data-12345678901234567890")
	time.Sleep(time.Millisecond)
	service.AddGPGKey("testuser", "Key 2", "key2-data-12345678901234567890")
	time.Sleep(time.Millisecond)
	service.AddGPGKey("otheruser", "Key 3", "key3-data-12345678901234567890")

	// List keys for testuser
	keys, err := service.ListGPGKeys("testuser")
	if err != nil {
		t.Fatalf("ListGPGKeys failed: %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("Expected 2 keys for testuser, got %d", len(keys))
	}
}

func TestGPGKeyService_GetGPGKey(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewGPGKeyService(db)

	// Add a key
	added, _ := service.AddGPGKey("testuser", "Test Key", "test-key-data-12345678901234567890")

	// Get the key
	retrieved, err := service.GetGPGKey("testuser", added.KeyID)
	if err != nil {
		t.Fatalf("GetGPGKey failed: %v", err)
	}

	if retrieved.Title != "Test Key" {
		t.Errorf("Expected Title 'Test Key', got '%s'", retrieved.Title)
	}
}

func TestGPGKeyService_GetGPGKey_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewGPGKeyService(db)

	// Get non-existent key
	_, err := service.GetGPGKey("testuser", 999)
	if err == nil {
		t.Error("Expected error for non-existent key, got nil")
	}
	if err != ErrGPGKeyNotFound {
		t.Errorf("Expected ErrGPGKeyNotFound, got %v", err)
	}
}

func TestGPGKeyService_DeleteGPGKey(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewGPGKeyService(db)

	// Add a key
	added, _ := service.AddGPGKey("testuser", "Test Key", "test-key-data-12345678901234567890")

	// Delete the key
	err := service.DeleteGPGKey("testuser", added.KeyID)
	if err != nil {
		t.Fatalf("DeleteGPGKey failed: %v", err)
	}

	// Verify deletion
	keys, _ := service.ListGPGKeys("testuser")
	if len(keys) != 0 {
		t.Errorf("Expected 0 keys after deletion, got %d", len(keys))
	}
}

func TestGPGKeyService_VerifyCommitSignature(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewGPGKeyService(db)

	// VerifyCommitSignature is not implemented yet, should return false
	verified, signer, err := service.VerifyCommitSignature("commit-123")
	if err != nil {
		t.Fatalf("VerifyCommitSignature failed: %v", err)
	}

	if verified {
		t.Error("Expected verified to be false (not implemented)")
	}
	if signer != "" {
		t.Errorf("Expected empty signer, got '%s'", signer)
	}
}

func TestGenerateGPGKeyID(t *testing.T) {
	// Test with short key
	shortKey := "short"
	id := generateGPGKeyID(shortKey)
	if id != "0000000000000000" {
		t.Errorf("Expected placeholder ID for short key, got '%s'", id)
	}

	// Test with long key
	longKey := "this-is-a-very-long-gpg-public-key-data"
	id = generateGPGKeyID(longKey)
	if len(id) != 16 {
		t.Errorf("Expected ID length 16, got %d", len(id))
	}
	if id != "this-is-a-very-l" {
		t.Errorf("Expected first 16 chars, got '%s'", id)
	}
}

func TestGPGKeyService_KeyIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewGPGKeyService(db)

	// Add keys for different users
	service.AddGPGKey("user1", "User1 Key", "user1-key-data-12345678901234567890")
	service.AddGPGKey("user2", "User2 Key", "user2-key-data-12345678901234567890")

	// Verify isolation
	user1Keys, _ := service.ListGPGKeys("user1")
	user2Keys, _ := service.ListGPGKeys("user2")

	if len(user1Keys) != 1 {
		t.Errorf("Expected 1 key for user1, got %d", len(user1Keys))
	}
	if len(user2Keys) != 1 {
		t.Errorf("Expected 1 key for user2, got %d", len(user2Keys))
	}

	if user1Keys[0].Title != "User1 Key" {
		t.Errorf("Expected 'User1 Key', got '%s'", user1Keys[0].Title)
	}
	if user2Keys[0].Title != "User2 Key" {
		t.Errorf("Expected 'User2 Key', got '%s'", user2Keys[0].Title)
	}
}

func TestGPGKeyService_RegisteredDate(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewGPGKeyService(db)

	before := time.Now()
	key, _ := service.AddGPGKey("testuser", "Test Key", "test-key-data-12345678901234567890")
	after := time.Now()

	if key.RegisteredDate.Before(before) || key.RegisteredDate.After(after) {
		t.Error("Expected RegisteredDate to be set to current time")
	}
}
