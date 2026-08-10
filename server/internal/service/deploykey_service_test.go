package service

import (
	"testing"

	"iforge/iforge/internal/model"
)

func TestDeployKeyService_CreateDeployKey(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewDeployKeyService(db)

	// Create a repository first
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create a deploy key
	publicKey := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ..."
	dk, err := service.CreateDeployKey("testuser", "test-repo", "Test Key", publicKey, false)
	if err != nil {
		t.Fatalf("CreateDeployKey failed: %v", err)
	}

	if dk.Title != "Test Key" {
		t.Errorf("Expected Title 'Test Key', got '%s'", dk.Title)
	}
	if dk.PublicKey != publicKey {
		t.Errorf("Expected PublicKey '%s', got '%s'", publicKey, dk.PublicKey)
	}
	if dk.AllowWrite != false {
		t.Errorf("Expected AllowWrite false, got %v", dk.AllowWrite)
	}
	if dk.DeployKeyID == 0 {
		t.Errorf("Expected DeployKeyID to be set, got 0")
	}
}

func TestDeployKeyService_ListDeployKeys(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewDeployKeyService(db)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create multiple deploy keys
	for i := 1; i <= 3; i++ {
		publicKey := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ" + string(rune('0'+i))
		_, err := service.CreateDeployKey("testuser", "test-repo", "Key "+string(rune('0'+i)), publicKey, false)
		if err != nil {
			t.Fatalf("CreateDeployKey failed: %v", err)
		}
	}

	// List deploy keys
	keys, err := service.ListDeployKeys("testuser", "test-repo")
	if err != nil {
		t.Fatalf("ListDeployKeys failed: %v", err)
	}

	if len(keys) != 3 {
		t.Errorf("Expected 3 deploy keys, got %d", len(keys))
	}
}

func TestDeployKeyService_GetDeployKey(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewDeployKeyService(db)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create a deploy key
	publicKey := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ..."
	created, err := service.CreateDeployKey("testuser", "test-repo", "Test Key", publicKey, false)
	if err != nil {
		t.Fatalf("CreateDeployKey failed: %v", err)
	}

	// Get the deploy key
	dk, err := service.GetDeployKey("testuser", "test-repo", created.DeployKeyID)
	if err != nil {
		t.Fatalf("GetDeployKey failed: %v", err)
	}

	if dk.DeployKeyID != created.DeployKeyID {
		t.Errorf("Expected DeployKeyID %d, got %d", created.DeployKeyID, dk.DeployKeyID)
	}
	if dk.Title != "Test Key" {
		t.Errorf("Expected Title 'Test Key', got '%s'", dk.Title)
	}
}

func TestDeployKeyService_GetDeployKey_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewDeployKeyService(db)

	// Try to get non-existent deploy key
	_, err := service.GetDeployKey("testuser", "test-repo", 999)
	if err == nil {
		t.Fatal("Expected error for non-existent deploy key, got nil")
	}
	if err != ErrDeployKeyNotFound {
		t.Errorf("Expected ErrDeployKeyNotFound, got %v", err)
	}
}

func TestDeployKeyService_DeleteDeployKey(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewDeployKeyService(db)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create a deploy key
	publicKey := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ..."
	created, err := service.CreateDeployKey("testuser", "test-repo", "Test Key", publicKey, false)
	if err != nil {
		t.Fatalf("CreateDeployKey failed: %v", err)
	}

	// Delete the deploy key
	err = service.DeleteDeployKey("testuser", "test-repo", created.DeployKeyID)
	if err != nil {
		t.Fatalf("DeleteDeployKey failed: %v", err)
	}

	// Verify deletion
	_, err = service.GetDeployKey("testuser", "test-repo", created.DeployKeyID)
	if err == nil {
		t.Fatal("Expected error after deletion, got nil")
	}
	if err != ErrDeployKeyNotFound {
		t.Errorf("Expected ErrDeployKeyNotFound, got %v", err)
	}
}

func TestDeployKeyService_DeleteDeployKey_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewDeployKeyService(db)

	// Try to delete non-existent deploy key
	err := service.DeleteDeployKey("testuser", "test-repo", 999)
	if err == nil {
		t.Fatal("Expected error for non-existent deploy key, got nil")
	}
	if err != ErrDeployKeyNotFound {
		t.Errorf("Expected ErrDeployKeyNotFound, got %v", err)
	}
}

func TestDeployKeyService_GetDeployKeyByPublicKey(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewDeployKeyService(db)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create a deploy key
	publicKey := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ..."
	created, err := service.CreateDeployKey("testuser", "test-repo", "Test Key", publicKey, false)
	if err != nil {
		t.Fatalf("CreateDeployKey failed: %v", err)
	}

	// Get deploy key by public key
	dk, err := service.GetDeployKeyByPublicKey(publicKey)
	if err != nil {
		t.Fatalf("GetDeployKeyByPublicKey failed: %v", err)
	}

	if dk.DeployKeyID != created.DeployKeyID {
		t.Errorf("Expected DeployKeyID %d, got %d", created.DeployKeyID, dk.DeployKeyID)
	}
	if dk.PublicKey != publicKey {
		t.Errorf("Expected PublicKey '%s', got '%s'", publicKey, dk.PublicKey)
	}
}

func TestDeployKeyService_GetDeployKeyByPublicKey_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewDeployKeyService(db)

	// Try to get deploy key by non-existent public key
	_, err := service.GetDeployKeyByPublicKey("non-existent-key")
	if err == nil {
		t.Fatal("Expected error for non-existent public key, got nil")
	}
	if err != ErrDeployKeyNotFound {
		t.Errorf("Expected ErrDeployKeyNotFound, got %v", err)
	}
}

func TestDeployKeyService_CreateDeployKey_WithWritePermission(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewDeployKeyService(db)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create a deploy key with write permission
	publicKey := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ..."
	dk, err := service.CreateDeployKey("testuser", "test-repo", "Write Key", publicKey, true)
	if err != nil {
		t.Fatalf("CreateDeployKey failed: %v", err)
	}

	if dk.AllowWrite != true {
		t.Errorf("Expected AllowWrite true, got %v", dk.AllowWrite)
	}
}
