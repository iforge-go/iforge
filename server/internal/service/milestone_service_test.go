package service

import (
	"testing"
	"time"

	"iforge/iforge/internal/model"
)

func TestMilestoneService_CreateMilestone(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewMilestoneService(db)

	// Create a repository first
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create a milestone
	dueDate := time.Now().Add(30 * 24 * time.Hour)
	milestone, err := service.CreateMilestone("testuser", "test-repo", "v1.0", "First release", &dueDate)
	if err != nil {
		t.Fatalf("CreateMilestone failed: %v", err)
	}

	if milestone.Title != "v1.0" {
		t.Errorf("Expected Title 'v1.0', got '%s'", milestone.Title)
	}
	if milestone.Description == nil || *milestone.Description != "First release" {
		t.Errorf("Expected Description 'First release', got %v", milestone.Description)
	}
	if milestone.MilestoneID == 0 {
		t.Errorf("Expected MilestoneID to be set, got 0")
	}
}

func TestMilestoneService_GetMilestone(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewMilestoneService(db)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create a milestone
	dueDate := time.Now().Add(30 * 24 * time.Hour)
	created, err := service.CreateMilestone("testuser", "test-repo", "v1.0", "First release", &dueDate)
	if err != nil {
		t.Fatalf("CreateMilestone failed: %v", err)
	}

	// Get the milestone
	milestone, err := service.GetMilestone("testuser", "test-repo", created.MilestoneID)
	if err != nil {
		t.Fatalf("GetMilestone failed: %v", err)
	}

	if milestone.MilestoneID != created.MilestoneID {
		t.Errorf("Expected MilestoneID %d, got %d", created.MilestoneID, milestone.MilestoneID)
	}
	if milestone.Title != "v1.0" {
		t.Errorf("Expected Title 'v1.0', got '%s'", milestone.Title)
	}
}

func TestMilestoneService_GetMilestone_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewMilestoneService(db)

	// Try to get non-existent milestone
	_, err := service.GetMilestone("testuser", "test-repo", 999)
	if err == nil {
		t.Fatal("Expected error for non-existent milestone, got nil")
	}
	if err != ErrMilestoneNotFound {
		t.Errorf("Expected ErrMilestoneNotFound, got %v", err)
	}
}

func TestMilestoneService_ListMilestones(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewMilestoneService(db)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create multiple milestones
	dueDate := time.Now().Add(30 * 24 * time.Hour)
	for i := 1; i <= 3; i++ {
		title := "v" + string(rune('0'+i)) + ".0"
		_, err := service.CreateMilestone("testuser", "test-repo", title, "Release "+title, &dueDate)
		if err != nil {
			t.Fatalf("CreateMilestone failed: %v", err)
		}
	}

	// List milestones
	result, err := service.ListMilestones("testuser", "test-repo")
	if err != nil {
		t.Fatalf("ListMilestones failed: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 milestones, got %d", len(result))
	}

	// Verify they are sorted by ID DESC (newest first)
	if result[0].Title != "v3.0" {
		t.Errorf("Expected first milestone 'v3.0', got '%s'", result[0].Title)
	}
	if result[2].Title != "v1.0" {
		t.Errorf("Expected last milestone 'v1.0', got '%s'", result[2].Title)
	}
}

func TestMilestoneService_UpdateMilestone(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewMilestoneService(db)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create a milestone
	dueDate := time.Now().Add(30 * 24 * time.Hour)
	created, err := service.CreateMilestone("testuser", "test-repo", "v1.0", "First release", &dueDate)
	if err != nil {
		t.Fatalf("CreateMilestone failed: %v", err)
	}

	// Update the milestone
	newDueDate := time.Now().Add(60 * 24 * time.Hour)
	err = service.UpdateMilestone("testuser", "test-repo", created.MilestoneID, "v1.1", "Updated release", &newDueDate)
	if err != nil {
		t.Fatalf("UpdateMilestone failed: %v", err)
	}

	// Verify update
	updated, err := service.GetMilestone("testuser", "test-repo", created.MilestoneID)
	if err != nil {
		t.Fatalf("GetMilestone failed: %v", err)
	}

	if updated.Title != "v1.1" {
		t.Errorf("Expected Title 'v1.1', got '%s'", updated.Title)
	}
	if updated.Description == nil || *updated.Description != "Updated release" {
		t.Errorf("Expected Description 'Updated release', got %v", updated.Description)
	}
}

func TestMilestoneService_DeleteMilestone(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewMilestoneService(db)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create a milestone
	dueDate := time.Now().Add(30 * 24 * time.Hour)
	created, err := service.CreateMilestone("testuser", "test-repo", "v1.0", "First release", &dueDate)
	if err != nil {
		t.Fatalf("CreateMilestone failed: %v", err)
	}

	// Delete the milestone
	err = service.DeleteMilestone("testuser", "test-repo", created.MilestoneID)
	if err != nil {
		t.Fatalf("DeleteMilestone failed: %v", err)
	}

	// Verify deletion
	_, err = service.GetMilestone("testuser", "test-repo", created.MilestoneID)
	if err == nil {
		t.Fatal("Expected error after deletion, got nil")
	}
	if err != ErrMilestoneNotFound {
		t.Errorf("Expected ErrMilestoneNotFound, got %v", err)
	}
}

func TestMilestoneService_DeleteMilestone_UnlinksIssues(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewMilestoneService(db)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create a milestone
	dueDate := time.Now().Add(30 * 24 * time.Hour)
	milestone, err := service.CreateMilestone("testuser", "test-repo", "v1.0", "First release", &dueDate)
	if err != nil {
		t.Fatalf("CreateMilestone failed: %v", err)
	}

	// Create an issue
	issueService := NewIssueService(db)
	issue, err := issueService.CreateIssue("testuser", "test-repo", "testuser", "Test Issue", nil, nil, nil, false)
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}

	// Link issue to milestone
	err = db.Model(&model.Issue{}).
		Where("issue_id = ?", issue.IssueID).
		Update("milestone_id", milestone.MilestoneID).Error
	if err != nil {
		t.Fatalf("Failed to link issue to milestone: %v", err)
	}

	// Delete the milestone
	err = service.DeleteMilestone("testuser", "test-repo", milestone.MilestoneID)
	if err != nil {
		t.Fatalf("DeleteMilestone failed: %v", err)
	}

	// Verify issue's milestone is unlinked
	var updatedIssue model.Issue
	err = db.Where("issue_id = ?", issue.IssueID).First(&updatedIssue).Error
	if err != nil {
		t.Fatalf("Failed to fetch issue: %v", err)
	}

	if updatedIssue.MilestoneID != nil {
		t.Errorf("Expected issue MilestoneID to be nil after milestone deletion, got %v", updatedIssue.MilestoneID)
	}
}

func TestMilestoneService_CloseMilestone(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewMilestoneService(db)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create a milestone
	dueDate := time.Now().Add(30 * 24 * time.Hour)
	created, err := service.CreateMilestone("testuser", "test-repo", "v1.0", "First release", &dueDate)
	if err != nil {
		t.Fatalf("CreateMilestone failed: %v", err)
	}

	// Close the milestone
	err = service.CloseMilestone("testuser", "test-repo", created.MilestoneID)
	if err != nil {
		t.Fatalf("CloseMilestone failed: %v", err)
	}

	// Verify milestone is closed
	closed, err := service.GetMilestone("testuser", "test-repo", created.MilestoneID)
	if err != nil {
		t.Fatalf("GetMilestone failed: %v", err)
	}

	if closed.ClosedDate == nil {
		t.Error("Expected ClosedDate to be set, got nil")
	}
}

func TestMilestoneService_ReopenMilestone(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewMilestoneService(db)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create a milestone
	dueDate := time.Now().Add(30 * 24 * time.Hour)
	created, err := service.CreateMilestone("testuser", "test-repo", "v1.0", "First release", &dueDate)
	if err != nil {
		t.Fatalf("CreateMilestone failed: %v", err)
	}

	// Close the milestone
	err = service.CloseMilestone("testuser", "test-repo", created.MilestoneID)
	if err != nil {
		t.Fatalf("CloseMilestone failed: %v", err)
	}

	// Reopen the milestone
	err = service.ReopenMilestone("testuser", "test-repo", created.MilestoneID)
	if err != nil {
		t.Fatalf("ReopenMilestone failed: %v", err)
	}

	// Verify milestone is reopened
	reopened, err := service.GetMilestone("testuser", "test-repo", created.MilestoneID)
	if err != nil {
		t.Fatalf("GetMilestone failed: %v", err)
	}

	if reopened.ClosedDate != nil {
		t.Errorf("Expected ClosedDate to be nil after reopen, got %v", reopened.ClosedDate)
	}
}

func TestMilestoneService_GetMilestoneIssues(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewMilestoneService(db)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create a milestone
	dueDate := time.Now().Add(30 * 24 * time.Hour)
	milestone, err := service.CreateMilestone("testuser", "test-repo", "v1.0", "First release", &dueDate)
	if err != nil {
		t.Fatalf("CreateMilestone failed: %v", err)
	}

	// Create multiple issues
	issueService := NewIssueService(db)
	for i := 1; i <= 3; i++ {
		issue, err := issueService.CreateIssue("testuser", "test-repo", "testuser", "Issue "+string(rune('0'+i)), nil, nil, nil, false)
		if err != nil {
			t.Fatalf("CreateIssue failed: %v", err)
		}

		// Link issue to milestone
		err = db.Model(&model.Issue{}).
			Where("issue_id = ?", issue.IssueID).
			Update("milestone_id", milestone.MilestoneID).Error
		if err != nil {
			t.Fatalf("Failed to link issue to milestone: %v", err)
		}
	}

	// Get milestone issues
	issues, err := service.GetMilestoneIssues("testuser", "test-repo", milestone.MilestoneID)
	if err != nil {
		t.Fatalf("GetMilestoneIssues failed: %v", err)
	}

	if len(issues) != 3 {
		t.Errorf("Expected 3 issues, got %d", len(issues))
	}
}
