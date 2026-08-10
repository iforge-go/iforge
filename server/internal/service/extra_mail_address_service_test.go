package service

import (
	"testing"
	"time"

	"iforge/iforge/internal/model"
)

func TestExtraMailAddressService_AddExtraMailAddress(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewExtraMailAddressService(db)

	// Add an extra mail address
	err := service.AddExtraMailAddress("testuser", "extra@example.com")
	if err != nil {
		t.Fatalf("AddExtraMailAddress failed: %v", err)
	}

	// Verify it was added
	addresses, err := service.ListExtraMailAddresses("testuser")
	if err != nil {
		t.Fatalf("ListExtraMailAddresses failed: %v", err)
	}

	if len(addresses) != 1 {
		t.Errorf("Expected 1 address, got %d", len(addresses))
	}
	if addresses[0] != "extra@example.com" {
		t.Errorf("Expected 'extra@example.com', got '%s'", addresses[0])
	}
}

func TestExtraMailAddressService_AddExtraMailAddress_Duplicate(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewExtraMailAddressService(db)

	// Add an address
	service.AddExtraMailAddress("testuser", "extra@example.com")

	// Try to add the same address again
	err := service.AddExtraMailAddress("testuser", "extra@example.com")
	if err == nil {
		t.Error("Expected error for duplicate address, got nil")
	}
	if err != ErrMailAddressExists {
		t.Errorf("Expected ErrMailAddressExists, got %v", err)
	}
}

func TestExtraMailAddressService_ListExtraMailAddresses(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewExtraMailAddressService(db)

	// Add multiple addresses
	service.AddExtraMailAddress("testuser", "extra1@example.com")
	service.AddExtraMailAddress("testuser", "extra2@example.com")
	service.AddExtraMailAddress("otheruser", "extra3@example.com")

	// List addresses for testuser
	addresses, err := service.ListExtraMailAddresses("testuser")
	if err != nil {
		t.Fatalf("ListExtraMailAddresses failed: %v", err)
	}

	if len(addresses) != 2 {
		t.Errorf("Expected 2 addresses for testuser, got %d", len(addresses))
	}
}

func TestExtraMailAddressService_DeleteExtraMailAddress(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewExtraMailAddressService(db)

	// Add an address
	service.AddExtraMailAddress("testuser", "extra@example.com")

	// Delete it
	err := service.DeleteExtraMailAddress("testuser", "extra@example.com")
	if err != nil {
		t.Fatalf("DeleteExtraMailAddress failed: %v", err)
	}

	// Verify deletion
	addresses, _ := service.ListExtraMailAddresses("testuser")
	if len(addresses) != 0 {
		t.Errorf("Expected 0 addresses after deletion, got %d", len(addresses))
	}
}

func TestExtraMailAddressService_DeleteExtraMailAddress_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewExtraMailAddressService(db)

	// Delete non-existent address
	err := service.DeleteExtraMailAddress("testuser", "nonexistent@example.com")
	if err == nil {
		t.Error("Expected error for non-existent address, got nil")
	}
	if err != ErrMailAddressNotFound {
		t.Errorf("Expected ErrMailAddressNotFound, got %v", err)
	}
}

func TestExtraMailAddressService_GetPrimaryMailAddress(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewExtraMailAddressService(db)

	// Create a user account
	account := &model.Account{
		UserName:       "testuser",
		Password:       "password",
		MailAddress:    "primary@example.com",
		RegisteredDate: time.Now(),
	}
	db.Create(account)

	// Get primary mail address
	primary, err := service.GetPrimaryMailAddress("testuser")
	if err != nil {
		t.Fatalf("GetPrimaryMailAddress failed: %v", err)
	}

	if primary != "primary@example.com" {
		t.Errorf("Expected 'primary@example.com', got '%s'", primary)
	}
}

func TestExtraMailAddressService_GetPrimaryMailAddress_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewExtraMailAddressService(db)

	// Get primary address for non-existent user
	_, err := service.GetPrimaryMailAddress("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent user, got nil")
	}
	if err != ErrMailAddressNotFound {
		t.Errorf("Expected ErrMailAddressNotFound, got %v", err)
	}
}

func TestExtraMailAddressService_MultipleUsers(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewExtraMailAddressService(db)

	// Add addresses for different users
	service.AddExtraMailAddress("user1", "user1-extra@example.com")
	service.AddExtraMailAddress("user2", "user2-extra@example.com")

	// Verify isolation
	user1Addresses, _ := service.ListExtraMailAddresses("user1")
	user2Addresses, _ := service.ListExtraMailAddresses("user2")

	if len(user1Addresses) != 1 {
		t.Errorf("Expected 1 address for user1, got %d", len(user1Addresses))
	}
	if len(user2Addresses) != 1 {
		t.Errorf("Expected 1 address for user2, got %d", len(user2Addresses))
	}

	if user1Addresses[0] != "user1-extra@example.com" {
		t.Errorf("Expected 'user1-extra@example.com', got '%s'", user1Addresses[0])
	}
	if user2Addresses[0] != "user2-extra@example.com" {
		t.Errorf("Expected 'user2-extra@example.com', got '%s'", user2Addresses[0])
	}
}
