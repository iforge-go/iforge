package service

import (
	"testing"

	"iforge/iforge/internal/model"
)

func TestSetupService_NewSetupService(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	// Create service without initialized setting
	service := NewSetupService(db)

	// Should not be initialized by default
	if service.IsInitialized() {
		t.Error("Expected IsInitialized to be false by default")
	}
}

func TestSetupService_NewSetupService_Initialized(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	// Create initialized setting
	setting := &model.SystemSetting{
		Key:   "initialized",
		Value: "true",
	}
	db.Create(setting)

	// Create service
	service := NewSetupService(db)

	// Should be initialized
	if !service.IsInitialized() {
		t.Error("Expected IsInitialized to be true after setting initialized=true")
	}
}

func TestSetupService_IsInitialized(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	service := NewSetupService(db)

	// Initially not initialized
	if service.IsInitialized() {
		t.Error("Expected IsInitialized to be false initially")
	}

	// Create initialized setting
	setting := &model.SystemSetting{
		Key:   "initialized",
		Value: "true",
	}
	db.Create(setting)

	// Still not initialized (cache not refreshed)
	if service.IsInitialized() {
		t.Error("Expected IsInitialized to be false before refresh")
	}

	// Refresh
	service.RefreshInitialized()

	// Now should be initialized
	if !service.IsInitialized() {
		t.Error("Expected IsInitialized to be true after refresh")
	}
}

func TestSetupService_RefreshInitialized_FalseToTrue(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	// Create service without initialized setting
	service := NewSetupService(db)
	if service.IsInitialized() {
		t.Error("Expected IsInitialized to be false initially")
	}

	// Create initialized setting
	setting := &model.SystemSetting{
		Key:   "initialized",
		Value: "true",
	}
	db.Create(setting)

	// Refresh
	service.RefreshInitialized()

	// Should now be initialized
	if !service.IsInitialized() {
		t.Error("Expected IsInitialized to be true after refresh")
	}
}

func TestSetupService_RefreshInitialized_TrueToFalse(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	// Create initialized setting
	setting := &model.SystemSetting{
		Key:   "initialized",
		Value: "true",
	}
	db.Create(setting)

	// Create service
	service := NewSetupService(db)
	if !service.IsInitialized() {
		t.Error("Expected IsInitialized to be true initially")
	}

	// Update setting to false
	db.Model(&model.SystemSetting{}).Where("`key` = ?", "initialized").Update("value", "false")

	// Refresh
	service.RefreshInitialized()

	// Should now be not initialized
	if service.IsInitialized() {
		t.Error("Expected IsInitialized to be false after refresh")
	}
}

func TestSetupService_RefreshInitialized_DeleteSetting(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	// Create initialized setting
	setting := &model.SystemSetting{
		Key:   "initialized",
		Value: "true",
	}
	db.Create(setting)

	// Create service
	service := NewSetupService(db)
	if !service.IsInitialized() {
		t.Error("Expected IsInitialized to be true initially")
	}

	// Delete setting
	db.Where("`key` = ?", "initialized").Delete(&model.SystemSetting{})

	// Refresh
	service.RefreshInitialized()

	// Should now be not initialized
	if service.IsInitialized() {
		t.Error("Expected IsInitialized to be false after setting deletion")
	}
}

func TestSetupService_MultipleRefresh(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	service := NewSetupService(db)

	// Initially false
	if service.IsInitialized() {
		t.Error("Expected false initially")
	}

	// Set to true
	db.Create(&model.SystemSetting{Key: "initialized", Value: "true"})
	service.RefreshInitialized()
	if !service.IsInitialized() {
		t.Error("Expected true after first refresh")
	}

	// Set to false
	db.Model(&model.SystemSetting{}).Where("`key` = ?", "initialized").Update("value", "false")
	service.RefreshInitialized()
	if service.IsInitialized() {
		t.Error("Expected false after second refresh")
	}

	// Set to true again
	db.Model(&model.SystemSetting{}).Where("`key` = ?", "initialized").Update("value", "true")
	service.RefreshInitialized()
	if !service.IsInitialized() {
		t.Error("Expected true after third refresh")
	}
}

func TestSetupService_ConcurrentAccess(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	// Create initialized setting first
	db.Create(&model.SystemSetting{Key: "initialized", Value: "true"})

	service := NewSetupService(db)

	// Concurrent reads
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_ = service.IsInitialized()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic and should be initialized
	if !service.IsInitialized() {
		t.Error("Expected IsInitialized to be true")
	}
}

func TestSetupService_RefreshInitialized_InvalidValue(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	// Create setting with invalid value
	setting := &model.SystemSetting{
		Key:   "initialized",
		Value: "invalid",
	}
	db.Create(setting)

	// Create service
	service := NewSetupService(db)

	// Should not be initialized (only "true" is considered initialized)
	if service.IsInitialized() {
		t.Error("Expected IsInitialized to be false for invalid value")
	}

	// Refresh
	service.RefreshInitialized()

	// Still should not be initialized
	if service.IsInitialized() {
		t.Error("Expected IsInitialized to be false for invalid value after refresh")
	}
}

func TestSetupService_RefreshInitialized_EmptyValue(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	// Create setting with empty value
	setting := &model.SystemSetting{
		Key:   "initialized",
		Value: "",
	}
	db.Create(setting)

	// Create service
	service := NewSetupService(db)

	// Should not be initialized
	if service.IsInitialized() {
		t.Error("Expected IsInitialized to be false for empty value")
	}

	// Refresh
	service.RefreshInitialized()

	// Still should not be initialized
	if service.IsInitialized() {
		t.Error("Expected IsInitialized to be false for empty value after refresh")
	}
}
