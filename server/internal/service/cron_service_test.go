package service

import (
	"errors"
	"testing"
	"time"
)

func TestCronService_CreateCron(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCronService(db, nil, nil, nil)

	// Create a cron schedule
	cron, err := service.CreateCron("testuser", "testrepo", "daily-build", "0 2 * * *", "main", "", "testuser")
	if err != nil {
		t.Fatalf("CreateCron failed: %v", err)
	}

	if cron.ID == 0 {
		t.Error("Expected ID to be set, got 0")
	}
	if cron.UserName != "testuser" {
		t.Errorf("Expected UserName 'testuser', got '%s'", cron.UserName)
	}
	if cron.RepositoryName != "testrepo" {
		t.Errorf("Expected RepositoryName 'testrepo', got '%s'", cron.RepositoryName)
	}
	if cron.Name != "daily-build" {
		t.Errorf("Expected Name 'daily-build', got '%s'", cron.Name)
	}
	if cron.Schedule != "0 2 * * *" {
		t.Errorf("Expected Schedule '0 2 * * *', got '%s'", cron.Schedule)
	}
	if cron.Branch != "main" {
		t.Errorf("Expected Branch 'main', got '%s'", cron.Branch)
	}
	if !cron.Enabled {
		t.Error("Expected Enabled true, got false")
	}
	if cron.NextRunAt == nil {
		t.Error("Expected NextRunAt to be set")
	}
}

func TestCronService_CreateCron_InvalidCronExpr(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCronService(db, nil, nil, nil)

	// Try to create with invalid cron expression
	_, err := service.CreateCron("testuser", "testrepo", "invalid", "invalid-cron", "main", "", "testuser")
	if err == nil {
		t.Error("Expected error for invalid cron expression, got nil")
	}
	// Error should wrap ErrInvalidCronExpr
	if !errors.Is(err, ErrInvalidCronExpr) {
		t.Errorf("Expected error to wrap ErrInvalidCronExpr, got %v", err)
	}
}

func TestCronService_CreateCron_NameConflict(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCronService(db, nil, nil, nil)

	// Create first cron
	service.CreateCron("testuser", "testrepo", "daily-build", "0 2 * * *", "main", "", "testuser")

	// Try to create with same name
	_, err := service.CreateCron("testuser", "testrepo", "daily-build", "0 3 * * *", "main", "", "testuser")
	if err == nil {
		t.Error("Expected error for duplicate name, got nil")
	}
	if err != ErrCronNameExists {
		t.Errorf("Expected ErrCronNameExists, got %v", err)
	}
}

func TestCronService_ListCrons(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCronService(db, nil, nil, nil)

	// Create multiple crons with time gaps to ensure distinct timestamps
	service.CreateCron("testuser", "testrepo", "cron1", "0 1 * * *", "main", "", "testuser")
	time.Sleep(10 * time.Millisecond)
	service.CreateCron("testuser", "testrepo", "cron2", "0 2 * * *", "main", "", "testuser")
	time.Sleep(10 * time.Millisecond)
	service.CreateCron("otheruser", "otherrepo", "cron3", "0 3 * * *", "main", "", "otheruser")

	// List crons for testuser/testrepo
	crons, err := service.ListCrons("testuser", "testrepo")
	if err != nil {
		t.Fatalf("ListCrons failed: %v", err)
	}

	if len(crons) != 2 {
		t.Errorf("Expected 2 crons, got %d", len(crons))
	}

	// Verify order (newest first)
	if crons[0].Name != "cron2" {
		t.Errorf("Expected first cron 'cron2', got '%s'", crons[0].Name)
	}
	if crons[1].Name != "cron1" {
		t.Errorf("Expected second cron 'cron1', got '%s'", crons[1].Name)
	}
}

func TestCronService_GetCron(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCronService(db, nil, nil, nil)

	// Create a cron
	created, _ := service.CreateCron("testuser", "testrepo", "daily-build", "0 2 * * *", "main", "", "testuser")

	// Get the cron
	cron, err := service.GetCron(created.ID)
	if err != nil {
		t.Fatalf("GetCron failed: %v", err)
	}

	if cron.Name != "daily-build" {
		t.Errorf("Expected Name 'daily-build', got '%s'", cron.Name)
	}
}

func TestCronService_GetCron_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCronService(db, nil, nil, nil)

	// Get non-existent cron
	_, err := service.GetCron(999)
	if err == nil {
		t.Error("Expected error for non-existent cron, got nil")
	}
	if err != ErrCronNotFound {
		t.Errorf("Expected ErrCronNotFound, got %v", err)
	}
}

func TestCronService_UpdateCron(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCronService(db, nil, nil, nil)

	// Create a cron
	created, _ := service.CreateCron("testuser", "testrepo", "daily-build", "0 2 * * *", "main", "", "testuser")

	// Update the cron
	updated, err := service.UpdateCron(created.ID, "0 3 * * *", "develop", "yaml-config", false)
	if err != nil {
		t.Fatalf("UpdateCron failed: %v", err)
	}

	if updated.Schedule != "0 3 * * *" {
		t.Errorf("Expected Schedule '0 3 * * *', got '%s'", updated.Schedule)
	}
	if updated.Branch != "develop" {
		t.Errorf("Expected Branch 'develop', got '%s'", updated.Branch)
	}
	if updated.YAMLConfig != "yaml-config" {
		t.Errorf("Expected YAMLConfig 'yaml-config', got '%s'", updated.YAMLConfig)
	}
	if updated.Enabled {
		t.Error("Expected Enabled false, got true")
	}
}

func TestCronService_UpdateCron_InvalidCronExpr(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCronService(db, nil, nil, nil)

	// Create a cron
	created, _ := service.CreateCron("testuser", "testrepo", "daily-build", "0 2 * * *", "main", "", "testuser")

	// Try to update with invalid cron expression
	_, err := service.UpdateCron(created.ID, "invalid-cron", "main", "", true)
	if err == nil {
		t.Error("Expected error for invalid cron expression, got nil")
	}
	// Error should wrap ErrInvalidCronExpr
	if !errors.Is(err, ErrInvalidCronExpr) {
		t.Errorf("Expected error to wrap ErrInvalidCronExpr, got %v", err)
	}
}

func TestCronService_UpdateCron_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCronService(db, nil, nil, nil)

	// Try to update non-existent cron
	_, err := service.UpdateCron(999, "0 3 * * *", "main", "", true)
	if err == nil {
		t.Error("Expected error for non-existent cron, got nil")
	}
	if err != ErrCronNotFound {
		t.Errorf("Expected ErrCronNotFound, got %v", err)
	}
}

func TestCronService_DeleteCron(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCronService(db, nil, nil, nil)

	// Create a cron
	created, _ := service.CreateCron("testuser", "testrepo", "daily-build", "0 2 * * *", "main", "", "testuser")

	// Delete the cron
	err := service.DeleteCron(created.ID)
	if err != nil {
		t.Fatalf("DeleteCron failed: %v", err)
	}

	// Verify deletion
	_, err = service.GetCron(created.ID)
	if err != ErrCronNotFound {
		t.Errorf("Expected ErrCronNotFound after deletion, got %v", err)
	}
}

func TestCronService_DeleteCron_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCronService(db, nil, nil, nil)

	// Try to delete non-existent cron
	err := service.DeleteCron(999)
	if err == nil {
		t.Error("Expected error for non-existent cron, got nil")
	}
	if err != ErrCronNotFound {
		t.Errorf("Expected ErrCronNotFound, got %v", err)
	}
}

func TestCronService_CreateCron_WithYAMLConfig(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCronService(db, nil, nil, nil)

	yamlConfig := `
name: daily-build
on:
  schedule:
    - cron: '0 2 * * *'
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
`

	// Create a cron with YAML config
	cron, err := service.CreateCron("testuser", "testrepo", "daily-build", "0 2 * * *", "main", yamlConfig, "testuser")
	if err != nil {
		t.Fatalf("CreateCron failed: %v", err)
	}

	if cron.YAMLConfig != yamlConfig {
		t.Errorf("Expected YAMLConfig to match, got '%s'", cron.YAMLConfig)
	}
}

func TestCronService_UpdateCron_ScheduleChange(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCronService(db, nil, nil, nil)

	// Create a cron
	created, _ := service.CreateCron("testuser", "testrepo", "daily-build", "0 2 * * *", "main", "", "testuser")
	originalNextRun := created.NextRunAt

	// Wait a bit to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Update with new schedule
	updated, err := service.UpdateCron(created.ID, "0 3 * * *", "main", "", true)
	if err != nil {
		t.Fatalf("UpdateCron failed: %v", err)
	}

	// NextRunAt should be updated
	if updated.NextRunAt == nil {
		t.Error("Expected NextRunAt to be set")
	}
	if originalNextRun != nil && updated.NextRunAt.Equal(*originalNextRun) {
		t.Error("Expected NextRunAt to be updated after schedule change")
	}
}

func TestCronService_CreateCron_DifferentRepos(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCronService(db, nil, nil, nil)

	// Create crons with same name in different repos
	cron1, err := service.CreateCron("user1", "repo1", "daily-build", "0 2 * * *", "main", "", "user1")
	if err != nil {
		t.Fatalf("CreateCron for repo1 failed: %v", err)
	}

	cron2, err := service.CreateCron("user2", "repo2", "daily-build", "0 3 * * *", "main", "", "user2")
	if err != nil {
		t.Fatalf("CreateCron for repo2 failed: %v", err)
	}

	// Both should succeed (different repos)
	if cron1.ID == cron2.ID {
		t.Error("Expected different IDs for crons in different repos")
	}

	// Verify isolation
	crons1, _ := service.ListCrons("user1", "repo1")
	crons2, _ := service.ListCrons("user2", "repo2")

	if len(crons1) != 1 {
		t.Errorf("Expected 1 cron for repo1, got %d", len(crons1))
	}
	if len(crons2) != 1 {
		t.Errorf("Expected 1 cron for repo2, got %d", len(crons2))
	}
}

func TestCronService_UpdateCron_EnableDisable(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCronService(db, nil, nil, nil)

	// Create a cron (enabled by default)
	created, _ := service.CreateCron("testuser", "testrepo", "daily-build", "0 2 * * *", "main", "", "testuser")
	if !created.Enabled {
		t.Error("Expected cron to be enabled by default")
	}

	// Disable the cron
	disabled, err := service.UpdateCron(created.ID, "0 2 * * *", "main", "", false)
	if err != nil {
		t.Fatalf("UpdateCron failed: %v", err)
	}
	if disabled.Enabled {
		t.Error("Expected cron to be disabled")
	}

	// Re-enable the cron
	enabled, err := service.UpdateCron(created.ID, "0 2 * * *", "main", "", true)
	if err != nil {
		t.Fatalf("UpdateCron failed: %v", err)
	}
	if !enabled.Enabled {
		t.Error("Expected cron to be enabled")
	}
}
