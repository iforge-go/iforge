package service

import (
	"testing"
)

func TestSSHKeyService_CreateSSHKey(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSSHKeyService(db)

	// Create a user first
	accountService := NewAccountService(db, nil)
	_, err := accountService.CreateAccount("testuser", "password123", "Test User", "test@example.com", false, nil, nil)
	if err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}

	// Create an SSH key
	publicKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHqWyG1nexvEDMHYhmXhuqiukucxCYL6u62W9PHLrUnH test@example.com"
	key, err := service.CreateSSHKey("testuser", "Test Key", publicKey)
	if err != nil {
		t.Fatalf("CreateSSHKey failed: %v", err)
	}

	if key.Title != "Test Key" {
		t.Errorf("Expected Title 'Test Key', got '%s'", key.Title)
	}
	if key.UserName != "testuser" {
		t.Errorf("Expected UserName 'testuser', got '%s'", key.UserName)
	}
	if key.SSHKeyID == 0 {
		t.Errorf("Expected SSHKeyID to be set, got 0")
	}
}

func TestSSHKeyService_CreateSSHKey_Duplicate(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSSHKeyService(db)

	// Create a user
	accountService := NewAccountService(db, nil)
	_, err := accountService.CreateAccount("testuser", "password123", "Test User", "test@example.com", false, nil, nil)
	if err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}

	// Create first SSH key
	publicKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHqWyG1nexvEDMHYhmXhuqiukucxCYL6u62W9PHLrUnH test@example.com"
	_, err = service.CreateSSHKey("testuser", "Key 1", publicKey)
	if err != nil {
		t.Fatalf("First CreateSSHKey failed: %v", err)
	}

	// Try to create duplicate
	_, err = service.CreateSSHKey("testuser", "Key 2", publicKey)
	if err == nil {
		t.Fatal("Expected error for duplicate SSH key, got nil")
	}
}

func TestSSHKeyService_ListSSHKeys(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSSHKeyService(db)

	// Create a user
	accountService := NewAccountService(db, nil)
	_, err := accountService.CreateAccount("testuser", "password123", "Test User", "test@example.com", false, nil, nil)
	if err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}

	// Create multiple SSH keys
	publicKeys := []string{
		"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHqWyG1nexvEDMHYhmXhuqiukucxCYL6u62W9PHLrUnH test@example.com",
		"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCwcUfs/p/4NSV5AzVhYhJUomddqX54j1UixN0uFC7vVLF+hN7rS8vsIlLT/i+cYMnGLSQijp/7SBR5/2jEbLO1+j35ZAGOIkI1LyEb7o0QsNc/Fs/WhqLkCz4MT3w37ys6Xhuz5QSAifSIbPIItGSPjwz9GFXiLsl91R/WCx4jG4PlpBhmyCqfpVF1ubP6rfUNOsN+hY3oRaamZWSTjenU2Xsjlen/tdWK/SfkMGgnmP6gQASfBU2YZL/L4zmiaidID1q+XRBjPFlLRm5n+8fOu0B/Xo71W0+vmsVXIHE/3/kZuMGIZmnmZN8/tdgo2sItVDamWSrs9VFfTSRsITTX test@example.com",
		"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBOCHhUHR1CaeAPXP+V0T4iyWfCtGD4T4LcK5H8fHniRLuBgpAbPTFN355FhSIXRd+SQsKTdDNShR6+N9zHHQLT4= test@example.com",
	}
	for i, publicKey := range publicKeys {
		_, err := service.CreateSSHKey("testuser", "Key "+string(rune('1'+i)), publicKey)
		if err != nil {
			t.Fatalf("CreateSSHKey failed: %v", err)
		}
	}

	// List SSH keys
	keys, err := service.ListSSHKeys("testuser")
	if err != nil {
		t.Fatalf("ListSSHKeys failed: %v", err)
	}

	if len(keys) != 3 {
		t.Errorf("Expected 3 SSH keys, got %d", len(keys))
	}
}

func TestSSHKeyService_GetSSHKey(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSSHKeyService(db)

	// Create a user
	accountService := NewAccountService(db, nil)
	_, err := accountService.CreateAccount("testuser", "password123", "Test User", "test@example.com", false, nil, nil)
	if err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}

	// Create an SSH key
	publicKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHqWyG1nexvEDMHYhmXhuqiukucxCYL6u62W9PHLrUnH test@example.com"
	created, err := service.CreateSSHKey("testuser", "Test Key", publicKey)
	if err != nil {
		t.Fatalf("CreateSSHKey failed: %v", err)
	}

	// Get the SSH key
	key, err := service.GetSSHKey("testuser", created.SSHKeyID)
	if err != nil {
		t.Fatalf("GetSSHKey failed: %v", err)
	}

	if key.SSHKeyID != created.SSHKeyID {
		t.Errorf("Expected SSHKeyID %d, got %d", created.SSHKeyID, key.SSHKeyID)
	}
	if key.Title != "Test Key" {
		t.Errorf("Expected Title 'Test Key', got '%s'", key.Title)
	}
}

func TestSSHKeyService_GetSSHKey_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSSHKeyService(db)

	// Try to get non-existent SSH key
	_, err := service.GetSSHKey("testuser", 999)
	if err == nil {
		t.Fatal("Expected error for non-existent SSH key, got nil")
	}
}

func TestSSHKeyService_DeleteSSHKey(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSSHKeyService(db)

	// Create a user
	accountService := NewAccountService(db, nil)
	_, err := accountService.CreateAccount("testuser", "password123", "Test User", "test@example.com", false, nil, nil)
	if err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}

	// Create an SSH key
	publicKey := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCwcUfs/p/4NSV5AzVhYhJUomddqX54j1UixN0uFC7vVLF+hN7rS8vsIlLT/i+cYMnGLSQijp/7SBR5/2jEbLO1+j35ZAGOIkI1LyEb7o0QsNc/Fs/WhqLkCz4MT3w37ys6Xhuz5QSAifSIbPIItGSPjwz9GFXiLsl91R/WCx4jG4PlpBhmyCqfpVF1ubP6rfUNOsN+hY3oRaamZWSTjenU2Xsjlen/tdWK/SfkMGgnmP6gQASfBU2YZL/L4zmiaidID1q+XRBjPFlLRm5n+8fOu0B/Xo71W0+vmsVXIHE/3/kZuMGIZmnmZN8/tdgo2sItVDamWSrs9VFfTSRsITTX test@example.com"
	created, err := service.CreateSSHKey("testuser", "Test Key", publicKey)
	if err != nil {
		t.Fatalf("CreateSSHKey failed: %v", err)
	}

	// Delete the SSH key
	err = service.DeleteSSHKey("testuser", created.SSHKeyID)
	if err != nil {
		t.Fatalf("DeleteSSHKey failed: %v", err)
	}

	// Verify deletion
	_, err = service.GetSSHKey("testuser", created.SSHKeyID)
	if err == nil {
		t.Fatal("Expected error after deletion, got nil")
	}
}

func TestSSHKeyService_DeleteSSHKey_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSSHKeyService(db)

	// Try to delete non-existent SSH key
	err := service.DeleteSSHKey("testuser", 999)
	if err == nil {
		t.Fatal("Expected error for non-existent SSH key, got nil")
	}
}

func TestSSHKeyService_GetAccountByPublicKey(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSSHKeyService(db)

	// Create a user
	accountService := NewAccountService(db, nil)
	_, err := accountService.CreateAccount("testuser", "password123", "Test User", "test@example.com", false, nil, nil)
	if err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}

	// Create an SSH key
	publicKey := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCwcUfs/p/4NSV5AzVhYhJUomddqX54j1UixN0uFC7vVLF+hN7rS8vsIlLT/i+cYMnGLSQijp/7SBR5/2jEbLO1+j35ZAGOIkI1LyEb7o0QsNc/Fs/WhqLkCz4MT3w37ys6Xhuz5QSAifSIbPIItGSPjwz9GFXiLsl91R/WCx4jG4PlpBhmyCqfpVF1ubP6rfUNOsN+hY3oRaamZWSTjenU2Xsjlen/tdWK/SfkMGgnmP6gQASfBU2YZL/L4zmiaidID1q+XRBjPFlLRm5n+8fOu0B/Xo71W0+vmsVXIHE/3/kZuMGIZmnmZN8/tdgo2sItVDamWSrs9VFfTSRsITTX test@example.com"
	_, err = service.CreateSSHKey("testuser", "Test Key", publicKey)
	if err != nil {
		t.Fatalf("CreateSSHKey failed: %v", err)
	}

	// Get account by public key
	account, err := service.GetAccountByPublicKey(publicKey)
	if err != nil {
		t.Fatalf("GetAccountByPublicKey failed: %v", err)
	}

	if account.UserName != "testuser" {
		t.Errorf("Expected UserName 'testuser', got '%s'", account.UserName)
	}
}

func TestSSHKeyService_GetAccountByPublicKey_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSSHKeyService(db)

	// Try to get account by non-existent public key (using a valid but different key)
	_, err := service.GetAccountByPublicKey("ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBOCHhUHR1CaeAPXP+V0T4iyWfCtGD4T4LcK5H8fHniRLuBgpAbPTFN355FhSIXRd+SQsKTdDNShR6+N9zHHQLT4= notfound@example.com")
	if err == nil {
		t.Fatal("Expected error for non-existent public key, got nil")
	}
}
