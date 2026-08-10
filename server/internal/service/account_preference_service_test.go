package service

import (
	"iforge/iforge/internal/model"
	"testing"
)

func TestAccountPreferenceService_GetPreference_Default(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAccountPreferenceService(db)

	// Get preference for user without preferences (should return defaults)
	pref, err := service.GetPreference("testuser")
	if err != nil {
		t.Fatalf("GetPreference failed: %v", err)
	}

	if pref.UserName != "testuser" {
		t.Errorf("Expected UserName 'testuser', got '%s'", pref.UserName)
	}
	if pref.HighlighterTheme != "github-v2" {
		t.Errorf("Expected default HighlighterTheme 'github-v2', got '%s'", pref.HighlighterTheme)
	}
	if !pref.Notification {
		t.Error("Expected default Notification true, got false")
	}
	if pref.Timezone != "UTC" {
		t.Errorf("Expected default Timezone 'UTC', got '%s'", pref.Timezone)
	}
}

func TestAccountPreferenceService_UpdatePreference_Create(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAccountPreferenceService(db)

	// Create new preference
	pref := &model.AccountPreference{
		UserName:         "testuser",
		HighlighterTheme: "monokai",
		Notification:     false,
		Timezone:         "Asia/Shanghai",
	}

	if err := service.UpdatePreference(pref); err != nil {
		t.Fatalf("UpdatePreference failed: %v", err)
	}

	// Verify creation
	updated, err := service.GetPreference("testuser")
	if err != nil {
		t.Fatalf("GetPreference failed: %v", err)
	}

	if updated.HighlighterTheme != "monokai" {
		t.Errorf("Expected HighlighterTheme 'monokai', got '%s'", updated.HighlighterTheme)
	}
	if updated.Notification {
		t.Error("Expected Notification false, got true")
	}
	if updated.Timezone != "Asia/Shanghai" {
		t.Errorf("Expected Timezone 'Asia/Shanghai', got '%s'", updated.Timezone)
	}
}

func TestAccountPreferenceService_UpdatePreference_Update(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAccountPreferenceService(db)

	// Create initial preference
	pref := &model.AccountPreference{
		UserName:         "testuser",
		HighlighterTheme: "github-v2",
		Notification:     true,
		Timezone:         "UTC",
	}
	if err := service.UpdatePreference(pref); err != nil {
		t.Fatalf("UpdatePreference (create) failed: %v", err)
	}

	// Update preference
	pref.HighlighterTheme = "dracula"
	pref.Notification = false
	pref.Timezone = "America/New_York"
	if err := service.UpdatePreference(pref); err != nil {
		t.Fatalf("UpdatePreference (update) failed: %v", err)
	}

	// Verify update
	updated, err := service.GetPreference("testuser")
	if err != nil {
		t.Fatalf("GetPreference failed: %v", err)
	}

	if updated.HighlighterTheme != "dracula" {
		t.Errorf("Expected HighlighterTheme 'dracula', got '%s'", updated.HighlighterTheme)
	}
	if updated.Notification {
		t.Error("Expected Notification false, got true")
	}
	if updated.Timezone != "America/New_York" {
		t.Errorf("Expected Timezone 'America/New_York', got '%s'", updated.Timezone)
	}
}

func TestAccountPreferenceService_GetPreference_AfterUpdate(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAccountPreferenceService(db)

	// Create and update preference
	pref := &model.AccountPreference{
		UserName:         "testuser",
		HighlighterTheme: "solarized-dark",
		Notification:     true,
		Timezone:         "Europe/London",
	}
	if err := service.UpdatePreference(pref); err != nil {
		t.Fatalf("UpdatePreference failed: %v", err)
	}

	// Get preference multiple times to ensure consistency
	for i := 0; i < 3; i++ {
		updated, err := service.GetPreference("testuser")
		if err != nil {
			t.Fatalf("GetPreference iteration %d failed: %v", i, err)
		}
		if updated.HighlighterTheme != "solarized-dark" {
			t.Errorf("Iteration %d: Expected HighlighterTheme 'solarized-dark', got '%s'", i, updated.HighlighterTheme)
		}
	}
}
