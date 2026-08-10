package service

import (
	"testing"
)

func TestAccessTokenService_CreateAccessToken(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAccessTokenService(db)

	// Create an access token
	token, tokenStr, err := service.CreateAccessToken("testuser", "Test token")
	if err != nil {
		t.Fatalf("CreateAccessToken failed: %v", err)
	}

	if token.UserName != "testuser" {
		t.Errorf("Expected UserName 'testuser', got '%s'", token.UserName)
	}
	if token.Note != "Test token" {
		t.Errorf("Expected Note 'Test token', got '%s'", token.Note)
	}
	if tokenStr == "" {
		t.Error("Expected token string to be non-empty")
	}
	if token.AccessTokenID == 0 {
		t.Error("Expected AccessTokenID to be set")
	}
}

func TestAccessTokenService_GetAccessToken(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAccessTokenService(db)

	// Create an access token
	created, _, err := service.CreateAccessToken("testuser", "Test token")
	if err != nil {
		t.Fatalf("CreateAccessToken failed: %v", err)
	}

	// Get the token
	token, err := service.GetAccessToken("testuser", created.AccessTokenID)
	if err != nil {
		t.Fatalf("GetAccessToken failed: %v", err)
	}

	if token.AccessTokenID != created.AccessTokenID {
		t.Errorf("Expected AccessTokenID %d, got %d", created.AccessTokenID, token.AccessTokenID)
	}
	if token.UserName != "testuser" {
		t.Errorf("Expected UserName 'testuser', got '%s'", token.UserName)
	}
}

func TestAccessTokenService_GetAccessToken_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAccessTokenService(db)

	_, err := service.GetAccessToken("testuser", 999)
	if err == nil {
		t.Fatal("Expected error for non-existent token, got nil")
	}
}

func TestAccessTokenService_ListAccessTokens(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAccessTokenService(db)

	// Create multiple tokens
	for i := 1; i <= 3; i++ {
		_, _, err := service.CreateAccessToken("testuser", "Token")
		if err != nil {
			t.Fatalf("CreateAccessToken %d failed: %v", i, err)
		}
	}

	// List tokens
	tokens, err := service.ListAccessTokens("testuser")
	if err != nil {
		t.Fatalf("ListAccessTokens failed: %v", err)
	}
	if len(tokens) != 3 {
		t.Errorf("Expected 3 tokens, got %d", len(tokens))
	}
}

func TestAccessTokenService_DeleteAccessToken(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAccessTokenService(db)

	// Create a token
	token, _, err := service.CreateAccessToken("testuser", "Test token")
	if err != nil {
		t.Fatalf("CreateAccessToken failed: %v", err)
	}

	// Delete the token
	if err := service.DeleteAccessToken(token.AccessTokenID, "testuser"); err != nil {
		t.Fatalf("DeleteAccessToken failed: %v", err)
	}

	// Verify deletion
	_, err = service.GetAccessToken("testuser", token.AccessTokenID)
	if err == nil {
		t.Fatal("Expected error after deletion, got nil")
	}
}

func TestAccessTokenService_DeleteAccessToken_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAccessTokenService(db)

	err := service.DeleteAccessToken(999, "testuser")
	if err == nil {
		t.Fatal("Expected error for non-existent token, got nil")
	}
}

func TestAccessTokenService_ValidateAccessToken(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAccessTokenService(db)

	// Create a user account first
	accountService := NewAccountService(db, nil)
	_, err := accountService.CreateAccount("testuser", "password123", "Test User", "test@example.com", false, nil, nil)
	if err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}

	// Create an access token
	_, tokenStr, err := service.CreateAccessToken("testuser", "Test token")
	if err != nil {
		t.Fatalf("CreateAccessToken failed: %v", err)
	}

	// Validate the token
	account, err := service.ValidateAccessToken(tokenStr)
	if err != nil {
		t.Fatalf("ValidateAccessToken failed: %v", err)
	}

	if account.UserName != "testuser" {
		t.Errorf("Expected UserName 'testuser', got '%s'", account.UserName)
	}
}

func TestAccessTokenService_ValidateAccessToken_Invalid(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAccessTokenService(db)

	_, err := service.ValidateAccessToken("invalid-token")
	if err == nil {
		t.Fatal("Expected error for invalid token, got nil")
	}
}

func TestAccessTokenService_ValidateAccessToken_UserRemoved(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAccessTokenService(db)

	// Create a user account
	accountService := NewAccountService(db, nil)
	account, err := accountService.CreateAccount("testuser", "password123", "Test User", "test@example.com", false, nil, nil)
	if err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}

	// Create an access token
	_, tokenStr, err := service.CreateAccessToken("testuser", "Test token")
	if err != nil {
		t.Fatalf("CreateAccessToken failed: %v", err)
	}

	// Soft delete the user
	if err := db.Model(account).Update("removed", true).Error; err != nil {
		t.Fatalf("Failed to soft delete user: %v", err)
	}

	// Validate should fail for removed user
	_, err = service.ValidateAccessToken(tokenStr)
	if err == nil {
		t.Fatal("Expected error for removed user's token, got nil")
	}
}
