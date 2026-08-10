package service

import (
	"testing"
	"time"

	"iforge/iforge/internal/model"
)

func TestCustomFieldService_CreateCustomField(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCustomFieldService(db)

	field := &model.CustomField{
		UserName:               "testuser",
		RepositoryName:         "testrepo",
		FieldName:              "Priority",
		FieldType:              "select",
		EnableForIssues:        true,
		EnableForMergeRequests: false,
		RegisteredDate:         time.Now(),
	}

	created, err := service.CreateCustomField(field)
	if err != nil {
		t.Fatalf("CreateCustomField failed: %v", err)
	}

	if created.FieldID == 0 {
		t.Error("Expected FieldID to be set, got 0")
	}
	if created.FieldName != "Priority" {
		t.Errorf("Expected FieldName 'Priority', got '%s'", created.FieldName)
	}
}

func TestCustomFieldService_ListCustomFields(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCustomFieldService(db)

	// Create multiple fields
	for i := 1; i <= 3; i++ {
		field := &model.CustomField{
			UserName:       "testuser",
			RepositoryName: "testrepo",
			FieldName:      "Field" + string(rune('0'+i)),
			FieldType:      "text",
			RegisteredDate: time.Now(),
		}
		service.CreateCustomField(field)
	}

	// List fields
	fields, err := service.ListCustomFields("testuser", "testrepo")
	if err != nil {
		t.Fatalf("ListCustomFields failed: %v", err)
	}

	if len(fields) != 3 {
		t.Errorf("Expected 3 fields, got %d", len(fields))
	}
}

func TestCustomFieldService_GetCustomField(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCustomFieldService(db)

	// Create a field
	field := &model.CustomField{
		UserName:       "testuser",
		RepositoryName: "testrepo",
		FieldName:      "TestField",
		FieldType:      "text",
		RegisteredDate: time.Now(),
	}
	created, _ := service.CreateCustomField(field)

	// Get the field
	retrieved, err := service.GetCustomField("testuser", "testrepo", created.FieldID)
	if err != nil {
		t.Fatalf("GetCustomField failed: %v", err)
	}

	if retrieved.FieldName != "TestField" {
		t.Errorf("Expected FieldName 'TestField', got '%s'", retrieved.FieldName)
	}
}

func TestCustomFieldService_GetCustomField_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCustomFieldService(db)

	// Get non-existent field
	_, err := service.GetCustomField("testuser", "testrepo", 999)
	if err == nil {
		t.Error("Expected error for non-existent field, got nil")
	}
	if err != ErrCustomFieldNotFound {
		t.Errorf("Expected ErrCustomFieldNotFound, got %v", err)
	}
}

func TestCustomFieldService_UpdateCustomField(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCustomFieldService(db)

	// Create a field
	field := &model.CustomField{
		UserName:       "testuser",
		RepositoryName: "testrepo",
		FieldName:      "OldName",
		FieldType:      "text",
		RegisteredDate: time.Now(),
	}
	created, _ := service.CreateCustomField(field)

	// Update the field
	updates := map[string]interface{}{
		"fieldName": "NewName",
		"fieldType": "select",
	}
	err := service.UpdateCustomField("testuser", "testrepo", created.FieldID, updates)
	if err != nil {
		t.Fatalf("UpdateCustomField failed: %v", err)
	}

	// Verify update
	retrieved, _ := service.GetCustomField("testuser", "testrepo", created.FieldID)
	if retrieved.FieldName != "NewName" {
		t.Errorf("Expected FieldName 'NewName', got '%s'", retrieved.FieldName)
	}
	if retrieved.FieldType != "select" {
		t.Errorf("Expected FieldType 'select', got '%s'", retrieved.FieldType)
	}
}

func TestCustomFieldService_DeleteCustomField(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCustomFieldService(db)

	// Create a field
	field := &model.CustomField{
		UserName:       "testuser",
		RepositoryName: "testrepo",
		FieldName:      "ToDelete",
		FieldType:      "text",
		RegisteredDate: time.Now(),
	}
	created, _ := service.CreateCustomField(field)

	// Delete the field
	err := service.DeleteCustomField("testuser", "testrepo", created.FieldID)
	if err != nil {
		t.Fatalf("DeleteCustomField failed: %v", err)
	}

	// Verify deletion
	_, err = service.GetCustomField("testuser", "testrepo", created.FieldID)
	if err != ErrCustomFieldNotFound {
		t.Errorf("Expected ErrCustomFieldNotFound after deletion, got %v", err)
	}
}

func TestCustomFieldService_SetIssueCustomFieldValue(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCustomFieldService(db)

	// Create a field first
	field := &model.CustomField{
		UserName:       "testuser",
		RepositoryName: "testrepo",
		FieldName:      "Priority",
		FieldType:      "select",
		RegisteredDate: time.Now(),
	}
	created, _ := service.CreateCustomField(field)

	// Set a value for an issue
	err := service.SetIssueCustomFieldValue("testuser", "testrepo", 1, created.FieldID, "High")
	if err != nil {
		t.Fatalf("SetIssueCustomFieldValue failed: %v", err)
	}

	// Get the value
	value, err := service.GetIssueCustomFieldValue("testuser", "testrepo", 1, created.FieldID)
	if err != nil {
		t.Fatalf("GetIssueCustomFieldValue failed: %v", err)
	}

	if value != "High" {
		t.Errorf("Expected value 'High', got '%s'", value)
	}
}

func TestCustomFieldService_SetIssueCustomFieldValue_Update(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCustomFieldService(db)

	// Create a field
	field := &model.CustomField{
		UserName:       "testuser",
		RepositoryName: "testrepo",
		FieldName:      "Priority",
		FieldType:      "select",
		RegisteredDate: time.Now(),
	}
	created, _ := service.CreateCustomField(field)

	// Set initial value
	service.SetIssueCustomFieldValue("testuser", "testrepo", 1, created.FieldID, "High")

	// Update the value
	err := service.SetIssueCustomFieldValue("testuser", "testrepo", 1, created.FieldID, "Low")
	if err != nil {
		t.Fatalf("SetIssueCustomFieldValue update failed: %v", err)
	}

	// Verify update
	value, _ := service.GetIssueCustomFieldValue("testuser", "testrepo", 1, created.FieldID)
	if value != "Low" {
		t.Errorf("Expected value 'Low', got '%s'", value)
	}
}

func TestCustomFieldService_GetIssueCustomFieldValues(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCustomFieldService(db)

	// Create multiple fields
	field1 := &model.CustomField{
		UserName:       "testuser",
		RepositoryName: "testrepo",
		FieldName:      "Priority",
		FieldType:      "select",
		RegisteredDate: time.Now(),
	}
	field2 := &model.CustomField{
		UserName:       "testuser",
		RepositoryName: "testrepo",
		FieldName:      "Status",
		FieldType:      "text",
		RegisteredDate: time.Now(),
	}
	created1, _ := service.CreateCustomField(field1)
	created2, _ := service.CreateCustomField(field2)

	// Set values for an issue
	service.SetIssueCustomFieldValue("testuser", "testrepo", 1, created1.FieldID, "High")
	service.SetIssueCustomFieldValue("testuser", "testrepo", 1, created2.FieldID, "In Progress")

	// Get all values for the issue
	values, err := service.GetIssueCustomFieldValues("testuser", "testrepo", 1)
	if err != nil {
		t.Fatalf("GetIssueCustomFieldValues failed: %v", err)
	}

	if len(values) != 2 {
		t.Errorf("Expected 2 values, got %d", len(values))
	}
}

func TestCustomFieldService_GetIssueCustomFieldValue_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCustomFieldService(db)

	// Get non-existent value
	value, err := service.GetIssueCustomFieldValue("testuser", "testrepo", 1, 999)
	if err != nil {
		t.Fatalf("GetIssueCustomFieldValue failed: %v", err)
	}

	// Should return empty string for non-existent value
	if value != "" {
		t.Errorf("Expected empty string for non-existent value, got '%s'", value)
	}
}
