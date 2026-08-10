package service

import (
	"testing"
)

func TestSystemSettingsService_GetSetSetting(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSystemSettingsService(db)

	// Set a setting
	err := service.SetSetting("test.key", "test.value")
	if err != nil {
		t.Fatalf("SetSetting failed: %v", err)
	}

	// Get the setting
	value, err := service.GetSetting("test.key")
	if err != nil {
		t.Fatalf("GetSetting failed: %v", err)
	}
	if value != "test.value" {
		t.Errorf("Expected 'test.value', got '%s'", value)
	}
}

func TestSystemSettingsService_GetSetting_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSystemSettingsService(db)

	// Get non-existent setting
	value, err := service.GetSetting("nonexistent.key")
	if err != nil {
		t.Fatalf("GetSetting failed: %v", err)
	}
	if value != "" {
		t.Errorf("Expected empty string for non-existent key, got '%s'", value)
	}
}

func TestSystemSettingsService_SetSetting_Update(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSystemSettingsService(db)

	// Set initial value
	err := service.SetSetting("test.key", "initial")
	if err != nil {
		t.Fatalf("SetSetting failed: %v", err)
	}

	// Update value
	err = service.SetSetting("test.key", "updated")
	if err != nil {
		t.Fatalf("SetSetting update failed: %v", err)
	}

	// Verify update
	value, err := service.GetSetting("test.key")
	if err != nil {
		t.Fatalf("GetSetting failed: %v", err)
	}
	if value != "updated" {
		t.Errorf("Expected 'updated', got '%s'", value)
	}
}

func TestSystemSettingsService_GetAllSettings(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSystemSettingsService(db)

	// Set multiple settings
	service.SetSetting("key1", "value1")
	service.SetSetting("key2", "value2")
	service.SetSetting("key3", "value3")

	// Get all settings
	settings, err := service.GetAllSettings()
	if err != nil {
		t.Fatalf("GetAllSettings failed: %v", err)
	}

	if len(settings) != 3 {
		t.Errorf("Expected 3 settings, got %d", len(settings))
	}

	if settings["key1"] != "value1" {
		t.Errorf("Expected key1='value1', got '%s'", settings["key1"])
	}
	if settings["key2"] != "value2" {
		t.Errorf("Expected key2='value2', got '%s'", settings["key2"])
	}
	if settings["key3"] != "value3" {
		t.Errorf("Expected key3='value3', got '%s'", settings["key3"])
	}
}

func TestSystemSettingsService_DeleteSetting(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSystemSettingsService(db)

	// Set a setting
	service.SetSetting("test.key", "test.value")

	// Delete it
	err := service.DeleteSetting("test.key")
	if err != nil {
		t.Fatalf("DeleteSetting failed: %v", err)
	}

	// Verify deletion
	value, err := service.GetSetting("test.key")
	if err != nil {
		t.Fatalf("GetSetting failed: %v", err)
	}
	if value != "" {
		t.Errorf("Expected empty string after deletion, got '%s'", value)
	}
}

func TestSystemSettingsService_SMTPSettings(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSystemSettingsService(db)

	// Set SMTP settings
	smtp := &SMTPSettings{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user@example.com",
		Password: "password123",
		From:     "noreply@example.com",
		SSL:      true,
	}

	err := service.SetSMTPSettings(smtp)
	if err != nil {
		t.Fatalf("SetSMTPSettings failed: %v", err)
	}

	// Get SMTP settings
	retrieved, err := service.GetSMTPSettings()
	if err != nil {
		t.Fatalf("GetSMTPSettings failed: %v", err)
	}

	if retrieved.Host != smtp.Host {
		t.Errorf("Expected Host '%s', got '%s'", smtp.Host, retrieved.Host)
	}
	if retrieved.Port != smtp.Port {
		t.Errorf("Expected Port %d, got %d", smtp.Port, retrieved.Port)
	}
	if retrieved.Username != smtp.Username {
		t.Errorf("Expected Username '%s', got '%s'", smtp.Username, retrieved.Username)
	}
	if retrieved.From != smtp.From {
		t.Errorf("Expected From '%s', got '%s'", smtp.From, retrieved.From)
	}
	if retrieved.SSL != smtp.SSL {
		t.Errorf("Expected SSL %v, got %v", smtp.SSL, retrieved.SSL)
	}
}

func TestSystemSettingsService_GeneralSettings(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSystemSettingsService(db)

	// Set general settings
	general := &GeneralSettings{
		SiteName:          "Test Site",
		Description:       "Test Description",
		AllowRegistration: true,
		AllowAnonymous:    false,
		DefaultBranch:     "main",
		Timezone:          "UTC",
	}

	err := service.SetGeneralSettings(general)
	if err != nil {
		t.Fatalf("SetGeneralSettings failed: %v", err)
	}

	// Get general settings
	retrieved, err := service.GetGeneralSettings()
	if err != nil {
		t.Fatalf("GetGeneralSettings failed: %v", err)
	}

	if retrieved.SiteName != general.SiteName {
		t.Errorf("Expected SiteName '%s', got '%s'", general.SiteName, retrieved.SiteName)
	}
	if retrieved.Description != general.Description {
		t.Errorf("Expected Description '%s', got '%s'", general.Description, retrieved.Description)
	}
	if retrieved.AllowRegistration != general.AllowRegistration {
		t.Errorf("Expected AllowRegistration %v, got %v", general.AllowRegistration, retrieved.AllowRegistration)
	}
	if retrieved.AllowAnonymous != general.AllowAnonymous {
		t.Errorf("Expected AllowAnonymous %v, got %v", general.AllowAnonymous, retrieved.AllowAnonymous)
	}
	if retrieved.DefaultBranch != general.DefaultBranch {
		t.Errorf("Expected DefaultBranch '%s', got '%s'", general.DefaultBranch, retrieved.DefaultBranch)
	}
	if retrieved.Timezone != general.Timezone {
		t.Errorf("Expected Timezone '%s', got '%s'", general.Timezone, retrieved.Timezone)
	}
}

func TestSystemSettingsService_AISettings(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSystemSettingsService(db)

	// Set AI settings
	ai := &AISettings{
		Enabled:   true,
		Provider:  "openai",
		BaseURL:   "https://api.openai.com/v1",
		APIKey:    "sk-test123",
		Model:     "gpt-4",
		Timeout:   60,
		MaxTokens: 2000,
	}

	err := service.SetAISettings(ai)
	if err != nil {
		t.Fatalf("SetAISettings failed: %v", err)
	}

	// Get AI settings
	retrieved, err := service.GetAISettings()
	if err != nil {
		t.Fatalf("GetAISettings failed: %v", err)
	}

	if retrieved.Enabled != ai.Enabled {
		t.Errorf("Expected Enabled %v, got %v", ai.Enabled, retrieved.Enabled)
	}
	if retrieved.Provider != ai.Provider {
		t.Errorf("Expected Provider '%s', got '%s'", ai.Provider, retrieved.Provider)
	}
	if retrieved.BaseURL != ai.BaseURL {
		t.Errorf("Expected BaseURL '%s', got '%s'", ai.BaseURL, retrieved.BaseURL)
	}
	if retrieved.Model != ai.Model {
		t.Errorf("Expected Model '%s', got '%s'", ai.Model, retrieved.Model)
	}
	if retrieved.Timeout != ai.Timeout {
		t.Errorf("Expected Timeout %d, got %d", ai.Timeout, retrieved.Timeout)
	}
	if retrieved.MaxTokens != ai.MaxTokens {
		t.Errorf("Expected MaxTokens %d, got %d", ai.MaxTokens, retrieved.MaxTokens)
	}
}
