package service

import (
	"errors"
	"testing"

	"iforge/iforge/internal/model"
)

func TestCreatePriority(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewPriorityService(db)

	// Create a repository first
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create a priority
	desc := "High priority issue"
	priority := &model.Priority{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		PriorityName:   "High",
		Color:          "#ff0000",
		Description:    &desc,
	}

	result, err := service.CreatePriority(priority)
	if err != nil {
		t.Fatalf("CreatePriority failed: %v", err)
	}

	if result.PriorityID == 0 {
		t.Errorf("Expected PriorityID to be set, got 0")
	}
	if result.PriorityName != "High" {
		t.Errorf("Expected PriorityName 'High', got '%s'", result.PriorityName)
	}
	if result.Color != "#ff0000" {
		t.Errorf("Expected Color '#ff0000', got '%s'", result.Color)
	}
}

func TestGetPriority(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewPriorityService(db)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create a priority
	desc := "Medium priority issue"
	priority := &model.Priority{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		PriorityName:   "Medium",
		Color:          "#ffaa00",
		Description:    &desc,
	}

	created, err := service.CreatePriority(priority)
	if err != nil {
		t.Fatalf("CreatePriority failed: %v", err)
	}

	// Get the priority
	retrieved, err := service.GetPriority("testuser", "test-repo", created.PriorityID)
	if err != nil {
		t.Fatalf("GetPriority failed: %v", err)
	}

	if retrieved.PriorityID != created.PriorityID {
		t.Errorf("Expected PriorityID %d, got %d", created.PriorityID, retrieved.PriorityID)
	}
	if retrieved.PriorityName != "Medium" {
		t.Errorf("Expected PriorityName 'Medium', got '%s'", retrieved.PriorityName)
	}
}

func TestGetPriority_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewPriorityService(db)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Try to get non-existent priority
	_, err := service.GetPriority("testuser", "test-repo", 999)
	if err == nil {
		t.Fatal("Expected error for non-existent priority, got nil")
	}
	if !errors.Is(err, ErrPriorityNotFound) {
		t.Errorf("Expected ErrPriorityNotFound, got %v", err)
	}
}

func TestListPriorities(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewPriorityService(db)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create multiple priorities
	priorities := []struct {
		name  string
		color string
	}{
		{"High", "#ff0000"},
		{"Medium", "#ffaa00"},
		{"Low", "#00ff00"},
	}

	for _, p := range priorities {
		desc := p.name + " priority"
		priority := &model.Priority{
			UserName:       "testuser",
			RepositoryName: "test-repo",
			PriorityName:   p.name,
			Color:          p.color,
			Description:    &desc,
		}
		_, err := service.CreatePriority(priority)
		if err != nil {
			t.Fatalf("CreatePriority failed: %v", err)
		}
	}

	// List priorities
	result, err := service.ListPriorities("testuser", "test-repo")
	if err != nil {
		t.Fatalf("ListPriorities failed: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 priorities, got %d", len(result))
	}
}

func TestUpdatePriority(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewPriorityService(db)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create a priority
	desc := "Low priority"
	priority := &model.Priority{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		PriorityName:   "Low",
		Color:          "#00ff00",
		Description:    &desc,
	}

	created, err := service.CreatePriority(priority)
	if err != nil {
		t.Fatalf("CreatePriority failed: %v", err)
	}

	// Update the priority
	updates := map[string]interface{}{
		"priorityName": "Very Low",
		"color":        "#00aa00",
		"description":  "Very low priority issue",
	}

	err = service.UpdatePriority("testuser", "test-repo", created.PriorityID, updates)
	if err != nil {
		t.Fatalf("UpdatePriority failed: %v", err)
	}

	// Verify update
	retrieved, err := service.GetPriority("testuser", "test-repo", created.PriorityID)
	if err != nil {
		t.Fatalf("GetPriority failed: %v", err)
	}

	if retrieved.PriorityName != "Very Low" {
		t.Errorf("Expected PriorityName 'Very Low', got '%s'", retrieved.PriorityName)
	}
	if retrieved.Color != "#00aa00" {
		t.Errorf("Expected Color '#00aa00', got '%s'", retrieved.Color)
	}
}

func TestDeletePriority(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewPriorityService(db)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create a priority
	desc := "Test priority"
	priority := &model.Priority{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		PriorityName:   "Test",
		Color:          "#000000",
		Description:    &desc,
	}

	created, err := service.CreatePriority(priority)
	if err != nil {
		t.Fatalf("CreatePriority failed: %v", err)
	}

	// Delete the priority
	err = service.DeletePriority("testuser", "test-repo", created.PriorityID)
	if err != nil {
		t.Fatalf("DeletePriority failed: %v", err)
	}

	// Verify deletion
	_, err = service.GetPriority("testuser", "test-repo", created.PriorityID)
	if err == nil {
		t.Fatal("Expected error after deletion, got nil")
	}
	if !errors.Is(err, ErrPriorityNotFound) {
		t.Errorf("Expected ErrPriorityNotFound, got %v", err)
	}
}

func TestDeletePriority_UnlinksIssues(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewPriorityService(db)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create a priority
	desc := "High priority"
	priority := &model.Priority{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		PriorityName:   "High",
		Color:          "#ff0000",
		Description:    &desc,
	}

	created, err := service.CreatePriority(priority)
	if err != nil {
		t.Fatalf("CreatePriority failed: %v", err)
	}

	// Create an issue with this priority
	issueService := NewIssueService(db)
	issue, err := issueService.CreateIssue("testuser", "test-repo", "testuser", "Test Issue", nil, &created.PriorityID, nil, false)
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}

	// Delete the priority
	err = service.DeletePriority("testuser", "test-repo", created.PriorityID)
	if err != nil {
		t.Fatalf("DeletePriority failed: %v", err)
	}

	// Verify issue's priority is unlinked
	retrievedIssue, err := issueService.GetIssue("testuser", "test-repo", issue.IssueID)
	if err != nil {
		t.Fatalf("GetIssue failed: %v", err)
	}

	if retrievedIssue.PriorityID != nil {
		t.Errorf("Expected issue PriorityID to be nil after priority deletion, got %v", retrievedIssue.PriorityID)
	}
}

func TestReorderPriorities(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewPriorityService(db)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create multiple priorities
	var priorityIDs []int
	for i := 1; i <= 3; i++ {
		desc := "Priority " + string(rune('0'+i))
		priority := &model.Priority{
			UserName:       "testuser",
			RepositoryName: "test-repo",
			PriorityName:   "Priority " + string(rune('0'+i)),
			Color:          "#000000",
			Description:    &desc,
		}
		created, err := service.CreatePriority(priority)
		if err != nil {
			t.Fatalf("CreatePriority failed: %v", err)
		}
		priorityIDs = append(priorityIDs, created.PriorityID)
	}

	// Reorder priorities (reverse order)
	reorderedIDs := []int{priorityIDs[2], priorityIDs[1], priorityIDs[0]}
	err := service.ReorderPriorities("testuser", "test-repo", reorderedIDs)
	if err != nil {
		t.Fatalf("ReorderPriorities failed: %v", err)
	}
}

func TestReorderPriorities_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewPriorityService(db)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Try to reorder with non-existent ID
	err := service.ReorderPriorities("testuser", "test-repo", []int{999})
	if err == nil {
		t.Fatal("Expected error for non-existent priority ID, got nil")
	}
}

func TestSetDefaultPriority(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewPriorityService(db)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create a priority
	desc := "Medium priority"
	priority := &model.Priority{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		PriorityName:   "Medium",
		Color:          "#ffaa00",
		Description:    &desc,
	}

	created, err := service.CreatePriority(priority)
	if err != nil {
		t.Fatalf("CreatePriority failed: %v", err)
	}

	// Set as default priority
	err = service.SetDefaultPriority("testuser", "test-repo", &created.PriorityID)
	if err != nil {
		t.Fatalf("SetDefaultPriority failed: %v", err)
	}

	// Verify default priority is set
	defaultID, err := service.GetDefaultPriority("testuser", "test-repo")
	if err != nil {
		t.Fatalf("GetDefaultPriority failed: %v", err)
	}

	if defaultID == nil {
		t.Fatal("Expected default priority to be set, got nil")
	}
	if *defaultID != created.PriorityID {
		t.Errorf("Expected default priority ID %d, got %d", created.PriorityID, *defaultID)
	}
}

func TestSetDefaultPriority_Nil(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewPriorityService(db)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create a priority
	desc := "Medium priority"
	priority := &model.Priority{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		PriorityName:   "Medium",
		Color:          "#ffaa00",
		Description:    &desc,
	}

	created, err := service.CreatePriority(priority)
	if err != nil {
		t.Fatalf("CreatePriority failed: %v", err)
	}

	// Set as default priority
	err = service.SetDefaultPriority("testuser", "test-repo", &created.PriorityID)
	if err != nil {
		t.Fatalf("SetDefaultPriority failed: %v", err)
	}

	// Remove default priority
	err = service.SetDefaultPriority("testuser", "test-repo", nil)
	if err != nil {
		t.Fatalf("SetDefaultPriority with nil failed: %v", err)
	}

	// Verify default priority is removed
	defaultID, err := service.GetDefaultPriority("testuser", "test-repo")
	if err != nil {
		t.Fatalf("GetDefaultPriority failed: %v", err)
	}

	if defaultID != nil {
		t.Errorf("Expected default priority to be nil, got %v", defaultID)
	}
}

func TestGetDefaultPriority(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewPriorityService(db)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Get default priority before setting
	defaultID, err := service.GetDefaultPriority("testuser", "test-repo")
	if err != nil {
		t.Fatalf("GetDefaultPriority failed: %v", err)
	}

	if defaultID != nil {
		t.Errorf("Expected default priority to be nil initially, got %v", defaultID)
	}
}
