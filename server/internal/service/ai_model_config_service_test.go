package service

import (
	"testing"
	"time"

	"iforge/iforge/internal/model"
)

func TestAIModelConfigService_Create(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAIModelConfigService(db)

	cfg := &model.AIModelConfig{
		Name:      "test-config",
		Provider:  "openai",
		BaseURL:   "https://api.openai.com/v1",
		APIKey:    "sk-test123",
		Model:     "gpt-4",
		Timeout:   60,
		MaxTokens: 2000,
		Enabled:   true,
		IsDefault: false,
	}

	created, err := service.Create(cfg)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if created.ID == 0 {
		t.Error("Expected ID to be set, got 0")
	}
	if created.Name != "test-config" {
		t.Errorf("Expected Name 'test-config', got '%s'", created.Name)
	}
	if created.Provider != "openai" {
		t.Errorf("Expected Provider 'openai', got '%s'", created.Provider)
	}
}

func TestAIModelConfigService_Create_Default(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAIModelConfigService(db)

	// Create first config as default
	cfg1 := &model.AIModelConfig{
		Name:      "config1",
		Provider:  "openai",
		IsDefault: true,
	}
	service.Create(cfg1)

	// Create second config as default (should unmark first)
	cfg2 := &model.AIModelConfig{
		Name:      "config2",
		Provider:  "deepseek",
		IsDefault: true,
	}
	service.Create(cfg2)

	// Verify only one default
	defaultCfg, err := service.GetDefault()
	if err != nil {
		t.Fatalf("GetDefault failed: %v", err)
	}
	if defaultCfg.Name != "config2" {
		t.Errorf("Expected default to be 'config2', got '%s'", defaultCfg.Name)
	}

	// Verify first config is no longer default
	retrieved1, _ := service.Get(cfg1.ID)
	if retrieved1.IsDefault {
		t.Error("Expected config1.IsDefault to be false after config2 became default")
	}
}

func TestAIModelConfigService_Create_NameConflict(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAIModelConfigService(db)

	cfg1 := &model.AIModelConfig{
		Name:     "duplicate-name",
		Provider: "openai",
	}
	service.Create(cfg1)

	cfg2 := &model.AIModelConfig{
		Name:     "duplicate-name",
		Provider: "deepseek",
	}
	_, err := service.Create(cfg2)
	if err == nil {
		t.Error("Expected error for duplicate name, got nil")
	}
	if err != ErrAIModelConfigNameConflict {
		t.Errorf("Expected ErrAIModelConfigNameConflict, got %v", err)
	}
}

func TestAIModelConfigService_Get(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAIModelConfigService(db)

	cfg := &model.AIModelConfig{
		Name:     "test-config",
		Provider: "openai",
	}
	created, _ := service.Create(cfg)

	retrieved, err := service.Get(created.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.Name != "test-config" {
		t.Errorf("Expected Name 'test-config', got '%s'", retrieved.Name)
	}
}

func TestAIModelConfigService_Get_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAIModelConfigService(db)

	_, err := service.Get(999)
	if err == nil {
		t.Error("Expected error for non-existent config, got nil")
	}
	if err != ErrAIModelConfigNotFound {
		t.Errorf("Expected ErrAIModelConfigNotFound, got %v", err)
	}
}

func TestAIModelConfigService_GetDefault(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAIModelConfigService(db)

	// Create configs
	cfg1 := &model.AIModelConfig{
		Name:      "config1",
		Provider:  "openai",
		IsDefault: false,
	}
	cfg2 := &model.AIModelConfig{
		Name:      "config2",
		Provider:  "deepseek",
		IsDefault: true,
	}
	service.Create(cfg1)
	service.Create(cfg2)

	// Get default
	defaultCfg, err := service.GetDefault()
	if err != nil {
		t.Fatalf("GetDefault failed: %v", err)
	}

	if defaultCfg.Name != "config2" {
		t.Errorf("Expected default to be 'config2', got '%s'", defaultCfg.Name)
	}
}

func TestAIModelConfigService_GetDefault_NoDefault(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAIModelConfigService(db)

	// Create config without default
	cfg := &model.AIModelConfig{
		Name:      "config1",
		Provider:  "openai",
		IsDefault: false,
	}
	service.Create(cfg)

	// Get default should fail
	_, err := service.GetDefault()
	if err == nil {
		t.Error("Expected error when no default config, got nil")
	}
	if err != ErrAIModelConfigNoDefault {
		t.Errorf("Expected ErrAIModelConfigNoDefault, got %v", err)
	}
}

func TestAIModelConfigService_List(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAIModelConfigService(db)

	// Create multiple configs
	service.Create(&model.AIModelConfig{Name: "config1", Provider: "openai"})
	service.Create(&model.AIModelConfig{Name: "config2", Provider: "deepseek"})
	service.Create(&model.AIModelConfig{Name: "config3", Provider: "qwen"})

	// List all
	configs, err := service.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(configs) != 3 {
		t.Errorf("Expected 3 configs, got %d", len(configs))
	}
}

func TestAIModelConfigService_Update(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAIModelConfigService(db)

	cfg := &model.AIModelConfig{
		Name:      "original",
		Provider:  "openai",
		APIKey:    "original-key",
		Timeout:   60,
		MaxTokens: 2000,
	}
	created, _ := service.Create(cfg)

	// Update config
	updates := &model.AIModelConfig{
		Name:      "updated",
		Provider:  "deepseek",
		APIKey:    "updated-key",
		Timeout:   120,
		MaxTokens: 4000,
	}
	updated, err := service.Update(created.ID, updates)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.Name != "updated" {
		t.Errorf("Expected Name 'updated', got '%s'", updated.Name)
	}
	if updated.Provider != "deepseek" {
		t.Errorf("Expected Provider 'deepseek', got '%s'", updated.Provider)
	}
	if updated.Timeout != 120 {
		t.Errorf("Expected Timeout 120, got %d", updated.Timeout)
	}
}

func TestAIModelConfigService_Update_PreserveAPIKey(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAIModelConfigService(db)

	cfg := &model.AIModelConfig{
		Name:   "test",
		APIKey: "original-secret-key",
	}
	created, _ := service.Create(cfg)

	// Update with empty API key (should preserve original)
	updates := &model.AIModelConfig{
		Name:   "updated",
		APIKey: "",
	}
	service.Update(created.ID, updates)

	retrieved, _ := service.Get(created.ID)
	if retrieved.APIKey != "original-secret-key" {
		t.Errorf("Expected APIKey to be preserved as 'original-secret-key', got '%s'", retrieved.APIKey)
	}

	// Update with masked API key (should preserve original)
	updates2 := &model.AIModelConfig{
		Name:   "updated2",
		APIKey: "sk-****1234",
	}
	service.Update(created.ID, updates2)

	retrieved2, _ := service.Get(created.ID)
	if retrieved2.APIKey != "original-secret-key" {
		t.Errorf("Expected APIKey to be preserved with mask, got '%s'", retrieved2.APIKey)
	}
}

func TestAIModelConfigService_Update_SetDefault(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAIModelConfigService(db)

	cfg1 := &model.AIModelConfig{
		Name:      "config1",
		IsDefault: true,
	}
	cfg2 := &model.AIModelConfig{
		Name:      "config2",
		IsDefault: false,
	}
	created1, _ := service.Create(cfg1)
	created2, _ := service.Create(cfg2)

	// Update config2 to be default
	updates := &model.AIModelConfig{
		Name:      "config2",
		IsDefault: true,
	}
	service.Update(created2.ID, updates)

	// Verify config2 is now default
	retrieved2, _ := service.Get(created2.ID)
	if !retrieved2.IsDefault {
		t.Error("Expected config2.IsDefault to be true")
	}

	// Verify config1 is no longer default
	retrieved1, _ := service.Get(created1.ID)
	if retrieved1.IsDefault {
		t.Error("Expected config1.IsDefault to be false after config2 became default")
	}
}

func TestAIModelConfigService_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAIModelConfigService(db)

	cfg := &model.AIModelConfig{
		Name:     "to-delete",
		Provider: "openai",
	}
	created, _ := service.Create(cfg)

	// Delete config
	err := service.Delete(created.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deletion
	_, err = service.Get(created.ID)
	if err != ErrAIModelConfigNotFound {
		t.Errorf("Expected ErrAIModelConfigNotFound after deletion, got %v", err)
	}
}

func TestAIModelConfigService_Delete_LastDefault(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAIModelConfigService(db)

	// Create two configs, one as default
	cfg1 := &model.AIModelConfig{
		Name:      "config1",
		IsDefault: true,
	}
	cfg2 := &model.AIModelConfig{
		Name:      "config2",
		IsDefault: false,
	}
	created1, _ := service.Create(cfg1)
	service.Create(cfg2)

	// Try to delete default config (should fail)
	err := service.Delete(created1.ID)
	if err == nil {
		t.Error("Expected error when deleting last default config, got nil")
	}
	if err != ErrAIModelConfigLastDefault {
		t.Errorf("Expected ErrAIModelConfigLastDefault, got %v", err)
	}
}

func TestAIModelConfigService_Delete_OnlyConfig(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAIModelConfigService(db)

	// Create only one config as default
	cfg := &model.AIModelConfig{
		Name:      "only-config",
		IsDefault: true,
	}
	created, _ := service.Create(cfg)

	// Delete should succeed (only config, no conflict)
	err := service.Delete(created.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deletion
	_, err = service.Get(created.ID)
	if err != ErrAIModelConfigNotFound {
		t.Errorf("Expected ErrAIModelConfigNotFound after deletion, got %v", err)
	}
}

func TestAIModelConfigService_SetDefault(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAIModelConfigService(db)

	cfg1 := &model.AIModelConfig{
		Name:      "config1",
		IsDefault: true,
		Enabled:   true,
	}
	cfg2 := &model.AIModelConfig{
		Name:      "config2",
		IsDefault: false,
		Enabled:   false,
	}
	created1, _ := service.Create(cfg1)
	created2, _ := service.Create(cfg2)

	// Set config2 as default
	result, err := service.SetDefault(created2.ID)
	if err != nil {
		t.Fatalf("SetDefault failed: %v", err)
	}

	if !result.IsDefault {
		t.Error("Expected config2.IsDefault to be true")
	}
	if !result.Enabled {
		t.Error("Expected config2.Enabled to be true (default must be enabled)")
	}

	// Verify config1 is no longer default
	retrieved1, _ := service.Get(created1.ID)
	if retrieved1.IsDefault {
		t.Error("Expected config1.IsDefault to be false after config2 became default")
	}
}

func TestAIModelConfigService_UpdateTestResult(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAIModelConfigService(db)

	cfg := &model.AIModelConfig{
		Name:     "test-config",
		Provider: "openai",
	}
	created, _ := service.Create(cfg)

	// Update test result to success
	err := service.UpdateTestResult(created.ID, true)
	if err != nil {
		t.Fatalf("UpdateTestResult failed: %v", err)
	}

	retrieved, _ := service.Get(created.ID)
	if retrieved.TestStatus != "success" {
		t.Errorf("Expected TestStatus 'success', got '%s'", retrieved.TestStatus)
	}
	if retrieved.TestedAt == nil {
		t.Error("Expected TestedAt to be set")
	}

	// Update test result to failure
	err = service.UpdateTestResult(created.ID, false)
	if err != nil {
		t.Fatalf("UpdateTestResult failed: %v", err)
	}

	retrieved2, _ := service.Get(created.ID)
	if retrieved2.TestStatus != "failed" {
		t.Errorf("Expected TestStatus 'failed', got '%s'", retrieved2.TestStatus)
	}
}

func TestIsDuplicateKeyError(t *testing.T) {
	tests := []struct {
		err      error
		expected bool
	}{
		{ErrAIModelConfigNotFound, false},
		{ErrAIModelConfigNameConflict, false},
	}

	for _, tt := range tests {
		result := isDuplicateKeyError(tt.err)
		if result != tt.expected {
			t.Errorf("isDuplicateKeyError(%v) = %v, expected %v", tt.err, result, tt.expected)
		}
	}
}

func TestAIModelConfigService_Timestamps(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAIModelConfigService(db)

	before := time.Now()
	cfg := &model.AIModelConfig{
		Name:     "test",
		Provider: "openai",
	}
	created, _ := service.Create(cfg)
	after := time.Now()

	if created.CreatedAt.Before(before) || created.CreatedAt.After(after) {
		t.Error("Expected CreatedAt to be set to current time")
	}
	if created.UpdatedAt.Before(before) || created.UpdatedAt.After(after) {
		t.Error("Expected UpdatedAt to be set to current time")
	}

	// Update and verify UpdatedAt changes
	time.Sleep(10 * time.Millisecond)
	updates := &model.AIModelConfig{
		Name: "updated",
	}
	updated, _ := service.Update(created.ID, updates)

	if !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Error("Expected UpdatedAt to be updated after Update")
	}
}
