package service

import (
	"testing"
	"time"

	"iforge/iforge/internal/model"
)

func TestPasswordResetService_GenerateToken(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewPasswordResetService(db)

	// Generate a token
	token, err := service.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	// Token should be 64 characters (32 bytes hex encoded)
	if len(token) != 64 {
		t.Errorf("Expected token length 64, got %d", len(token))
	}

	// Generate another token and verify they're different
	token2, err := service.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken second call failed: %v", err)
	}
	if token == token2 {
		t.Error("Expected different tokens, got same")
	}
}

func TestPasswordResetService_CreateResetToken(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewPasswordResetService(db)

	// Create a reset token
	token, err := service.CreateResetToken("testuser")
	if err != nil {
		t.Fatalf("CreateResetToken failed: %v", err)
	}

	if len(token) != 64 {
		t.Errorf("Expected token length 64, got %d", len(token))
	}

	// Verify token exists in database
	var count int64
	db.Model(&model.PasswordResetToken{}).Where("token = ?", token).Count(&count)
	if count != 1 {
		t.Errorf("Expected 1 token in database, got %d", count)
	}
}

func TestPasswordResetService_ValidateToken(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewPasswordResetService(db)

	// Create a reset token
	token, err := service.CreateResetToken("testuser")
	if err != nil {
		t.Fatalf("CreateResetToken failed: %v", err)
	}

	// Validate the token
	username, err := service.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	if username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", username)
	}
}

func TestPasswordResetService_ValidateToken_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewPasswordResetService(db)

	// Validate non-existent token
	_, err := service.ValidateToken("nonexistent-token")
	if err == nil {
		t.Error("Expected error for non-existent token, got nil")
	}
	if err != ErrTokenNotFound {
		t.Errorf("Expected ErrTokenNotFound, got %v", err)
	}
}

func TestPasswordResetService_ValidateToken_Expired(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewPasswordResetService(db)

	// Create an expired token manually
	expiredToken := &model.PasswordResetToken{
		Token:     "expired-token-123",
		UserName:  "testuser",
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
		CreatedAt: time.Now().Add(-25 * time.Hour),
	}
	db.Create(expiredToken)

	// Validate expired token
	_, err := service.ValidateToken("expired-token-123")
	if err == nil {
		t.Error("Expected error for expired token, got nil")
	}
	if err != ErrTokenExpired {
		t.Errorf("Expected ErrTokenExpired, got %v", err)
	}
}

func TestPasswordResetService_ResetPassword(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewPasswordResetService(db)

	// Create a test user
	user := &model.Account{
		UserName:       "testuser",
		Password:       "oldpassword",
		MailAddress:    "test@example.com",
		RegisteredDate: time.Now(),
	}
	db.Create(user)

	// Create a reset token
	token, err := service.CreateResetToken("testuser")
	if err != nil {
		t.Fatalf("CreateResetToken failed: %v", err)
	}

	// Reset password
	err = service.ResetPassword(token, "newpassword123")
	if err != nil {
		t.Fatalf("ResetPassword failed: %v", err)
	}

	// Verify password was updated
	var updatedUser model.Account
	db.Where("user_name = ?", "testuser").First(&updatedUser)
	if updatedUser.Password == "oldpassword" {
		t.Error("Expected password to be updated, but it wasn't changed")
	}
	if updatedUser.Password == "newpassword123" {
		t.Error("Password should be hashed, not stored in plain text")
	}

	// Verify token was deleted
	var count int64
	db.Model(&model.PasswordResetToken{}).Where("token = ?", token).Count(&count)
	if count != 0 {
		t.Errorf("Expected token to be deleted, but found %d", count)
	}
}

func TestPasswordResetService_DeleteToken(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewPasswordResetService(db)

	// Create a reset token
	token, err := service.CreateResetToken("testuser")
	if err != nil {
		t.Fatalf("CreateResetToken failed: %v", err)
	}

	// Delete the token
	err = service.DeleteToken(token)
	if err != nil {
		t.Fatalf("DeleteToken failed: %v", err)
	}

	// Verify token was deleted
	var count int64
	db.Model(&model.PasswordResetToken{}).Where("token = ?", token).Count(&count)
	if count != 0 {
		t.Errorf("Expected token to be deleted, but found %d", count)
	}
}

func TestPasswordResetService_CleanupExpiredTokens(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewPasswordResetService(db)

	// Create an expired token
	expiredToken := &model.PasswordResetToken{
		Token:     "expired-token",
		UserName:  "user1",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
		CreatedAt: time.Now().Add(-25 * time.Hour),
	}
	db.Create(expiredToken)

	// Create a valid token
	validToken := &model.PasswordResetToken{
		Token:     "valid-token",
		UserName:  "user2",
		ExpiresAt: time.Now().Add(23 * time.Hour),
		CreatedAt: time.Now(),
	}
	db.Create(validToken)

	// Cleanup expired tokens
	err := service.CleanupExpiredTokens()
	if err != nil {
		t.Fatalf("CleanupExpiredTokens failed: %v", err)
	}

	// Verify expired token was deleted
	var count int64
	db.Model(&model.PasswordResetToken{}).Where("token = ?", "expired-token").Count(&count)
	if count != 0 {
		t.Errorf("Expected expired token to be deleted, but found %d", count)
	}

	// Verify valid token still exists
	db.Model(&model.PasswordResetToken{}).Where("token = ?", "valid-token").Count(&count)
	if count != 1 {
		t.Errorf("Expected valid token to exist, but found %d", count)
	}
}
