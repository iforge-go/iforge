package service

import (
	"testing"
)

func TestCommitStatusService_CreateOrUpdateStatus(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCommitStatusService(db)

	// Create a new commit status
	targetURL := "https://example.com/build/123"
	description := "Build passed"
	status, err := service.CreateOrUpdateStatus(
		"testuser", "test-repo", "abc123", "ci/build",
		"success", &targetURL, &description, "testuser",
	)
	if err != nil {
		t.Fatalf("CreateOrUpdateStatus failed: %v", err)
	}

	if status.UserName != "testuser" {
		t.Errorf("Expected UserName 'testuser', got '%s'", status.UserName)
	}
	if status.RepositoryName != "test-repo" {
		t.Errorf("Expected RepositoryName 'test-repo', got '%s'", status.RepositoryName)
	}
	if status.CommitID != "abc123" {
		t.Errorf("Expected CommitID 'abc123', got '%s'", status.CommitID)
	}
	if status.Context != "ci/build" {
		t.Errorf("Expected Context 'ci/build', got '%s'", status.Context)
	}
	if status.State != "success" {
		t.Errorf("Expected State 'success', got '%s'", status.State)
	}
}

func TestCommitStatusService_CreateOrUpdateStatus_Update(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCommitStatusService(db)

	// Create initial status
	targetURL := "https://example.com/build/123"
	description := "Build started"
	_, err := service.CreateOrUpdateStatus(
		"testuser", "test-repo", "abc123", "ci/build",
		"pending", &targetURL, &description, "testuser",
	)
	if err != nil {
		t.Fatalf("CreateOrUpdateStatus (create) failed: %v", err)
	}

	// Update the status
	newDescription := "Build passed"
	updated, err := service.CreateOrUpdateStatus(
		"testuser", "test-repo", "abc123", "ci/build",
		"success", &targetURL, &newDescription, "testuser",
	)
	if err != nil {
		t.Fatalf("CreateOrUpdateStatus (update) failed: %v", err)
	}

	if updated.State != "success" {
		t.Errorf("Expected State 'success', got '%s'", updated.State)
	}
	if updated.Description == nil || *updated.Description != newDescription {
		t.Errorf("Expected Description '%s', got %v", newDescription, updated.Description)
	}
}

func TestCommitStatusService_GetStatusesForCommit(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCommitStatusService(db)

	// Create multiple statuses for the same commit
	for i, context := range []string{"ci/build", "ci/test", "ci/lint"} {
		_, err := service.CreateOrUpdateStatus(
			"testuser", "test-repo", "abc123", context,
			"success", nil, nil, "testuser",
		)
		if err != nil {
			t.Fatalf("CreateOrUpdateStatus %d failed: %v", i, err)
		}
	}

	// Get all statuses for the commit
	statuses, err := service.GetStatusesForCommit("testuser", "test-repo", "abc123")
	if err != nil {
		t.Fatalf("GetStatusesForCommit failed: %v", err)
	}
	if len(statuses) != 3 {
		t.Errorf("Expected 3 statuses, got %d", len(statuses))
	}
}

func TestCommitStatusService_GetStatus(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCommitStatusService(db)

	// Create a status
	targetURL := "https://example.com/build/123"
	description := "Build passed"
	_, err := service.CreateOrUpdateStatus(
		"testuser", "test-repo", "abc123", "ci/build",
		"success", &targetURL, &description, "testuser",
	)
	if err != nil {
		t.Fatalf("CreateOrUpdateStatus failed: %v", err)
	}

	// Get the specific status
	status, err := service.GetStatus("testuser", "test-repo", "abc123", "ci/build")
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}

	if status.Context != "ci/build" {
		t.Errorf("Expected Context 'ci/build', got '%s'", status.Context)
	}
	if status.State != "success" {
		t.Errorf("Expected State 'success', got '%s'", status.State)
	}
}

func TestCommitStatusService_GetStatus_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCommitStatusService(db)

	_, err := service.GetStatus("testuser", "test-repo", "abc123", "ci/nonexistent")
	if err == nil {
		t.Fatal("Expected error for non-existent status, got nil")
	}
	if err != ErrCommitStatusNotFound {
		t.Errorf("Expected ErrCommitStatusNotFound, got %v", err)
	}
}

func TestCommitStatusService_GetCombinedStatus_Success(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCommitStatusService(db)

	// Create all successful statuses
	for _, context := range []string{"ci/build", "ci/test", "ci/lint"} {
		_, err := service.CreateOrUpdateStatus(
			"testuser", "test-repo", "abc123", context,
			"success", nil, nil, "testuser",
		)
		if err != nil {
			t.Fatalf("CreateOrUpdateStatus failed: %v", err)
		}
	}

	// Get combined status
	combined, err := service.GetCombinedStatus("testuser", "test-repo", "abc123")
	if err != nil {
		t.Fatalf("GetCombinedStatus failed: %v", err)
	}

	if combined.State != "success" {
		t.Errorf("Expected combined State 'success', got '%s'", combined.State)
	}
	if len(combined.Statuses) != 3 {
		t.Errorf("Expected 3 statuses, got %d", len(combined.Statuses))
	}
}

func TestCommitStatusService_GetCombinedStatus_Failure(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCommitStatusService(db)

	// Create mixed statuses
	_, err := service.CreateOrUpdateStatus(
		"testuser", "test-repo", "abc123", "ci/build",
		"success", nil, nil, "testuser",
	)
	if err != nil {
		t.Fatalf("CreateOrUpdateStatus failed: %v", err)
	}

	_, err = service.CreateOrUpdateStatus(
		"testuser", "test-repo", "abc123", "ci/test",
		"failure", nil, nil, "testuser",
	)
	if err != nil {
		t.Fatalf("CreateOrUpdateStatus failed: %v", err)
	}

	// Get combined status
	combined, err := service.GetCombinedStatus("testuser", "test-repo", "abc123")
	if err != nil {
		t.Fatalf("GetCombinedStatus failed: %v", err)
	}

	if combined.State != "failure" {
		t.Errorf("Expected combined State 'failure', got '%s'", combined.State)
	}
}

func TestCommitStatusService_GetCombinedStatus_Pending(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCommitStatusService(db)

	// Create mixed statuses with pending
	_, err := service.CreateOrUpdateStatus(
		"testuser", "test-repo", "abc123", "ci/build",
		"success", nil, nil, "testuser",
	)
	if err != nil {
		t.Fatalf("CreateOrUpdateStatus failed: %v", err)
	}

	_, err = service.CreateOrUpdateStatus(
		"testuser", "test-repo", "abc123", "ci/test",
		"pending", nil, nil, "testuser",
	)
	if err != nil {
		t.Fatalf("CreateOrUpdateStatus failed: %v", err)
	}

	// Get combined status
	combined, err := service.GetCombinedStatus("testuser", "test-repo", "abc123")
	if err != nil {
		t.Fatalf("GetCombinedStatus failed: %v", err)
	}

	if combined.State != "pending" {
		t.Errorf("Expected combined State 'pending', got '%s'", combined.State)
	}
}

func TestCommitStatusService_GetCombinedStatus_NoStatuses(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCommitStatusService(db)

	// Get combined status for commit with no statuses
	combined, err := service.GetCombinedStatus("testuser", "test-repo", "abc123")
	if err != nil {
		t.Fatalf("GetCombinedStatus failed: %v", err)
	}

	if combined.State != "pending" {
		t.Errorf("Expected combined State 'pending' for no statuses, got '%s'", combined.State)
	}
	if len(combined.Statuses) != 0 {
		t.Errorf("Expected 0 statuses, got %d", len(combined.Statuses))
	}
}
