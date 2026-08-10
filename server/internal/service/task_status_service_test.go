package service

import (
	"testing"
)

func TestTaskStatusService_CreateTaskStatus(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewTaskStatusService(db)

	// Create a task status
	status, err := service.CreateTaskStatus(1, "To Do", "todo", "#daa724", "todo", 0, false, false, nil)
	if err != nil {
		t.Fatalf("CreateTaskStatus failed: %v", err)
	}

	if status.Name != "To Do" {
		t.Errorf("Expected Name 'To Do', got '%s'", status.Name)
	}
	if status.Slug != "todo" {
		t.Errorf("Expected Slug 'todo', got '%s'", status.Slug)
	}
	if status.Category != "todo" {
		t.Errorf("Expected Category 'todo', got '%s'", status.Category)
	}
}

func TestTaskStatusService_CreateTaskStatus_Duplicate(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewTaskStatusService(db)

	// Create first status
	_, err := service.CreateTaskStatus(1, "To Do", "todo", "#daa724", "todo", 0, false, false, nil)
	if err != nil {
		t.Fatalf("CreateTaskStatus failed: %v", err)
	}

	// Try to create duplicate
	_, err = service.CreateTaskStatus(1, "To Do 2", "todo", "#daa724", "todo", 1, false, false, nil)
	if err == nil {
		t.Fatal("Expected error for duplicate status, got nil")
	}
	if err != ErrTaskStatusExists {
		t.Errorf("Expected ErrTaskStatusExists, got %v", err)
	}
}

func TestTaskStatusService_GetTaskStatus(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewTaskStatusService(db)

	// Create a status
	created, err := service.CreateTaskStatus(1, "To Do", "todo", "#daa724", "todo", 0, false, false, nil)
	if err != nil {
		t.Fatalf("CreateTaskStatus failed: %v", err)
	}

	// Get the status
	status, err := service.GetTaskStatus(created.ID)
	if err != nil {
		t.Fatalf("GetTaskStatus failed: %v", err)
	}

	if status.ID != created.ID {
		t.Errorf("Expected ID %d, got %d", created.ID, status.ID)
	}
	if status.Name != "To Do" {
		t.Errorf("Expected Name 'To Do', got '%s'", status.Name)
	}
}

func TestTaskStatusService_GetTaskStatus_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewTaskStatusService(db)

	_, err := service.GetTaskStatus(999)
	if err == nil {
		t.Fatal("Expected error for non-existent status, got nil")
	}
	if err != ErrTaskStatusNotFound {
		t.Errorf("Expected ErrTaskStatusNotFound, got %v", err)
	}
}

func TestTaskStatusService_GetTaskStatusBySlug(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewTaskStatusService(db)

	// Create a status
	_, err := service.CreateTaskStatus(1, "To Do", "todo", "#daa724", "todo", 0, false, false, nil)
	if err != nil {
		t.Fatalf("CreateTaskStatus failed: %v", err)
	}

	// Get by slug
	status, err := service.GetTaskStatusBySlug(1, "todo")
	if err != nil {
		t.Fatalf("GetTaskStatusBySlug failed: %v", err)
	}

	if status.Slug != "todo" {
		t.Errorf("Expected Slug 'todo', got '%s'", status.Slug)
	}
}

func TestTaskStatusService_ListTaskStatuses(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewTaskStatusService(db)

	// Create multiple statuses
	for i, slug := range []string{"todo", "in_progress", "done"} {
		_, err := service.CreateTaskStatus(1, slug, slug, "#000000", "todo", i, false, false, nil)
		if err != nil {
			t.Fatalf("CreateTaskStatus(%s) failed: %v", slug, err)
		}
	}

	// List statuses
	statuses, err := service.ListTaskStatuses(1)
	if err != nil {
		t.Fatalf("ListTaskStatuses failed: %v", err)
	}
	if len(statuses) != 3 {
		t.Errorf("Expected 3 statuses, got %d", len(statuses))
	}
}

func TestTaskStatusService_UpdateTaskStatus(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewTaskStatusService(db)

	// Create a status
	status, err := service.CreateTaskStatus(1, "To Do", "todo", "#daa724", "todo", 0, false, false, nil)
	if err != nil {
		t.Fatalf("CreateTaskStatus failed: %v", err)
	}

	// Update the status
	status.Name = "Updated To Do"
	status.Color = "#ff0000"
	if err := service.UpdateTaskStatus(status); err != nil {
		t.Fatalf("UpdateTaskStatus failed: %v", err)
	}

	// Verify update
	updated, err := service.GetTaskStatus(status.ID)
	if err != nil {
		t.Fatalf("GetTaskStatus failed: %v", err)
	}
	if updated.Name != "Updated To Do" {
		t.Errorf("Expected Name 'Updated To Do', got '%s'", updated.Name)
	}
	if updated.Color != "#ff0000" {
		t.Errorf("Expected Color '#ff0000', got '%s'", updated.Color)
	}
}

func TestTaskStatusService_DeleteTaskStatus(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewTaskStatusService(db)

	// Create a status
	status, err := service.CreateTaskStatus(1, "To Do", "todo", "#daa724", "todo", 0, false, false, nil)
	if err != nil {
		t.Fatalf("CreateTaskStatus failed: %v", err)
	}

	// Delete the status
	if err := service.DeleteTaskStatus(status.ID); err != nil {
		t.Fatalf("DeleteTaskStatus failed: %v", err)
	}

	// Verify deletion
	_, err = service.GetTaskStatus(status.ID)
	if err != ErrTaskStatusNotFound {
		t.Errorf("Expected ErrTaskStatusNotFound after deletion, got %v", err)
	}
}

func TestTaskStatusService_InitializeDefaultStatuses(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewTaskStatusService(db)

	// Initialize default statuses
	if err := service.InitializeDefaultStatuses(1); err != nil {
		t.Fatalf("InitializeDefaultStatuses failed: %v", err)
	}

	// List statuses
	statuses, err := service.ListTaskStatuses(1)
	if err != nil {
		t.Fatalf("ListTaskStatuses failed: %v", err)
	}
	if len(statuses) != 4 {
		t.Errorf("Expected 4 default statuses, got %d", len(statuses))
	}

	// Verify default statuses
	expectedSlugs := []string{"todo", "in_progress", "review", "done"}
	for i, slug := range expectedSlugs {
		if statuses[i].Slug != slug {
			t.Errorf("Expected status[%d].Slug '%s', got '%s'", i, slug, statuses[i].Slug)
		}
	}
}

func TestTaskStatusService_SetTransitions(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewTaskStatusService(db)

	// Initialize default statuses
	if err := service.InitializeDefaultStatuses(1); err != nil {
		t.Fatalf("InitializeDefaultStatuses failed: %v", err)
	}

	// Set transitions
	pairs := []StatusTransitionPair{
		{From: "todo", To: "in_progress"},
		{From: "in_progress", To: "review"},
		{From: "review", To: "done"},
	}
	if err := service.SetTransitions(1, pairs); err != nil {
		t.Fatalf("SetTransitions failed: %v", err)
	}

	// List transitions
	transitions, err := service.ListTransitions(1)
	if err != nil {
		t.Fatalf("ListTransitions failed: %v", err)
	}
	if len(transitions) != 3 {
		t.Errorf("Expected 3 transitions, got %d", len(transitions))
	}
}

func TestTaskStatusService_IsTransitionAllowed(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewTaskStatusService(db)

	// Initialize default statuses
	if err := service.InitializeDefaultStatuses(1); err != nil {
		t.Fatalf("InitializeDefaultStatuses failed: %v", err)
	}

	// Set transitions
	pairs := []StatusTransitionPair{
		{From: "todo", To: "in_progress"},
		{From: "in_progress", To: "review"},
	}
	if err := service.SetTransitions(1, pairs); err != nil {
		t.Fatalf("SetTransitions failed: %v", err)
	}

	// Check allowed transition
	allowed, err := service.IsTransitionAllowed(1, "todo", "in_progress")
	if err != nil {
		t.Fatalf("IsTransitionAllowed failed: %v", err)
	}
	if !allowed {
		t.Errorf("Expected transition todo->in_progress to be allowed")
	}

	// Check disallowed transition
	allowed, err = service.IsTransitionAllowed(1, "todo", "done")
	if err != nil {
		t.Fatalf("IsTransitionAllowed failed: %v", err)
	}
	if allowed {
		t.Errorf("Expected transition todo->done to be disallowed")
	}

	// Check self-transition (always allowed)
	allowed, err = service.IsTransitionAllowed(1, "todo", "todo")
	if err != nil {
		t.Fatalf("IsTransitionAllowed failed: %v", err)
	}
	if !allowed {
		t.Errorf("Expected self-transition to be allowed")
	}
}

func TestTaskStatusService_CategoryNormalization(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewTaskStatusService(db)

	// Create status with invalid category
	status, err := service.CreateTaskStatus(1, "Test", "test", "#000000", "invalid", 0, false, false, nil)
	if err != nil {
		t.Fatalf("CreateTaskStatus failed: %v", err)
	}

	// Should normalize to "todo" for non-closed status
	if status.Category != "todo" {
		t.Errorf("Expected Category 'todo' for non-closed status, got '%s'", status.Category)
	}

	// Create closed status with invalid category
	closedStatus, err := service.CreateTaskStatus(1, "Closed", "closed", "#000000", "invalid", 1, true, true, nil)
	if err != nil {
		t.Fatalf("CreateTaskStatus failed: %v", err)
	}

	// Should normalize to "done" for closed status
	if closedStatus.Category != "done" {
		t.Errorf("Expected Category 'done' for closed status, got '%s'", closedStatus.Category)
	}
}
