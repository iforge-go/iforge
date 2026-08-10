package service

import (
	"testing"
	"time"

	"iforge/iforge/internal/model"
)

func TestCreateMergeRequest(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewMergeRequestService(db, nil)

	// Create a repository first
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create a merge request
	mr, err := service.CreateMergeRequest(
		"testuser", "test-repo",
		"requestuser", "request-repo",
		"main", "feature-branch",
		"Test MR", "MR description",
		"abc123", "def456",
		false, "testuser",
	)
	if err != nil {
		t.Fatalf("CreateMergeRequest failed: %v", err)
	}

	if mr.IssueID != 1 {
		t.Errorf("Expected IssueID 1, got %d", mr.IssueID)
	}
	if mr.Branch != "main" {
		t.Errorf("Expected Branch 'main', got '%s'", mr.Branch)
	}
	if mr.RequestBranch != "feature-branch" {
		t.Errorf("Expected RequestBranch 'feature-branch', got '%s'", mr.RequestBranch)
	}
	if mr.IsDraft != false {
		t.Errorf("Expected IsDraft false, got %v", mr.IsDraft)
	}

	// Create another merge request to test ID increment
	mr2, err := service.CreateMergeRequest(
		"testuser", "test-repo",
		"requestuser", "request-repo",
		"main", "feature-branch-2",
		"Test MR 2", "MR description 2",
		"ghi789", "jkl012",
		true, "testuser",
	)
	if err != nil {
		t.Fatalf("Second CreateMergeRequest failed: %v", err)
	}

	if mr2.IssueID != 2 {
		t.Errorf("Expected IssueID 2, got %d", mr2.IssueID)
	}
}

func TestUpdateMergeRequest(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewMergeRequestService(db, nil)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create a merge request
	mr, err := service.CreateMergeRequest(
		"testuser", "test-repo",
		"requestuser", "request-repo",
		"main", "feature-branch",
		"Test MR", "MR description",
		"abc123", "def456",
		true, "testuser",
	)
	if err != nil {
		t.Fatalf("CreateMergeRequest failed: %v", err)
	}

	// Update the merge request
	updates := map[string]interface{}{
		"is_draft":       false,
		"commit_id_from": "new-abc123",
		"commit_id_to":   "new-def456",
	}
	err = service.UpdateMergeRequest("testuser", "test-repo", mr.IssueID, updates)
	if err != nil {
		t.Fatalf("UpdateMergeRequest failed: %v", err)
	}

	// Verify the update
	var updatedMR model.MergeRequest
	err = db.Where("user_name = ? AND repository_name = ? AND issue_id = ?",
		"testuser", "test-repo", mr.IssueID).First(&updatedMR).Error
	if err != nil {
		t.Fatalf("Failed to fetch updated MR: %v", err)
	}

	if updatedMR.IsDraft != false {
		t.Errorf("Expected IsDraft false after update, got %v", updatedMR.IsDraft)
	}
	if updatedMR.CommitIDFrom != "new-abc123" {
		t.Errorf("Expected CommitIDFrom 'new-abc123', got '%s'", updatedMR.CommitIDFrom)
	}
	if updatedMR.CommitIDTo != "new-def456" {
		t.Errorf("Expected CommitIDTo 'new-def456', got '%s'", updatedMR.CommitIDTo)
	}
}

func TestUpdateMergeRequest_EmptyUpdates(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewMergeRequestService(db, nil)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create a merge request
	mr, err := service.CreateMergeRequest(
		"testuser", "test-repo",
		"requestuser", "request-repo",
		"main", "feature-branch",
		"Test MR", "MR description",
		"abc123", "def456",
		false, "testuser",
	)
	if err != nil {
		t.Fatalf("CreateMergeRequest failed: %v", err)
	}

	// Update with empty map should be no-op
	err = service.UpdateMergeRequest("testuser", "test-repo", mr.IssueID, map[string]interface{}{})
	if err != nil {
		t.Fatalf("UpdateMergeRequest with empty map should not error: %v", err)
	}
}

func TestMergeMergeRequest(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewMergeRequestService(db, nil)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create a merge request
	mr, err := service.CreateMergeRequest(
		"testuser", "test-repo",
		"requestuser", "request-repo",
		"main", "feature-branch",
		"Test MR", "MR description",
		"abc123", "def456",
		false, "testuser",
	)
	if err != nil {
		t.Fatalf("CreateMergeRequest failed: %v", err)
	}

	// Merge the merge request
	mergedCommits := `[{"id":"commit1","message":"commit message"}]`
	mergedFileChanges := `[{"filename":"file.txt","status":"modified"}]`
	err = service.MergeMergeRequest(
		"testuser", "test-repo", mr.IssueID,
		"merge-commit-123",
		&mergedCommits,
		&mergedFileChanges,
	)
	if err != nil {
		t.Fatalf("MergeMergeRequest failed: %v", err)
	}

	// Verify the merge
	var updatedMR model.MergeRequest
	err = db.Where("user_name = ? AND repository_name = ? AND issue_id = ?",
		"testuser", "test-repo", mr.IssueID).First(&updatedMR).Error
	if err != nil {
		t.Fatalf("Failed to fetch merged MR: %v", err)
	}

	if updatedMR.MergedCommitIDs == nil || *updatedMR.MergedCommitIDs != "merge-commit-123" {
		t.Errorf("Expected MergedCommitIDs 'merge-commit-123', got %v", updatedMR.MergedCommitIDs)
	}
	if updatedMR.MergedCommits == nil || *updatedMR.MergedCommits != mergedCommits {
		t.Errorf("Expected MergedCommits to match, got %v", updatedMR.MergedCommits)
	}
	if updatedMR.MergedFileChanges == nil || *updatedMR.MergedFileChanges != mergedFileChanges {
		t.Errorf("Expected MergedFileChanges to match, got %v", updatedMR.MergedFileChanges)
	}

	// Verify the issue is closed
	var issue model.Issue
	err = db.Where("user_name = ? AND repository_name = ? AND issue_id = ?",
		"testuser", "test-repo", mr.IssueID).First(&issue).Error
	if err != nil {
		t.Fatalf("Failed to fetch issue: %v", err)
	}
	if !issue.Closed {
		t.Errorf("Expected issue to be closed after merge")
	}
}

func TestGetMergeRequest(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewMergeRequestService(db, nil)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create a merge request
	createdMR, err := service.CreateMergeRequest(
		"testuser", "test-repo",
		"requestuser", "request-repo",
		"main", "feature-branch",
		"Test MR", "MR description",
		"abc123", "def456",
		false, "testuser",
	)
	if err != nil {
		t.Fatalf("CreateMergeRequest failed: %v", err)
	}

	// Get the merge request
	mr, err := service.GetMergeRequest("testuser", "test-repo", createdMR.IssueID)
	if err != nil {
		t.Fatalf("GetMergeRequest failed: %v", err)
	}

	if mr.IssueID != createdMR.IssueID {
		t.Errorf("Expected IssueID %d, got %d", createdMR.IssueID, mr.IssueID)
	}
	if mr.Title != "Test MR" {
		t.Errorf("Expected Title 'Test MR', got '%s'", mr.Title)
	}
	if mr.State != "open" {
		t.Errorf("Expected State 'open', got '%s'", mr.State)
	}
	if mr.Branch != "main" {
		t.Errorf("Expected Branch 'main', got '%s'", mr.Branch)
	}
	if mr.RequestBranch != "feature-branch" {
		t.Errorf("Expected RequestBranch 'feature-branch', got '%s'", mr.RequestBranch)
	}
	if mr.IsDraft != false {
		t.Errorf("Expected IsDraft false, got %v", mr.IsDraft)
	}
}

func TestGetMergeRequest_AfterMerge(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewMergeRequestService(db, nil)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create a merge request
	createdMR, err := service.CreateMergeRequest(
		"testuser", "test-repo",
		"requestuser", "request-repo",
		"main", "feature-branch",
		"Test MR", "MR description",
		"abc123", "def456",
		false, "testuser",
	)
	if err != nil {
		t.Fatalf("CreateMergeRequest failed: %v", err)
	}

	// Merge the merge request
	mergedCommits := `[{"id":"commit1"}]`
	mergedFileChanges := `[{"filename":"file.txt"}]`
	err = service.MergeMergeRequest(
		"testuser", "test-repo", createdMR.IssueID,
		"merge-commit-123",
		&mergedCommits,
		&mergedFileChanges,
	)
	if err != nil {
		t.Fatalf("MergeMergeRequest failed: %v", err)
	}

	// Get the merge request after merge
	mr, err := service.GetMergeRequest("testuser", "test-repo", createdMR.IssueID)
	if err != nil {
		t.Fatalf("GetMergeRequest failed: %v", err)
	}

	if mr.State != "merged" {
		t.Errorf("Expected State 'merged', got '%s'", mr.State)
	}
	if !mr.Merged {
		t.Errorf("Expected Merged true, got false")
	}
	if mr.MergedCommitIDs == nil || *mr.MergedCommitIDs != "merge-commit-123" {
		t.Errorf("Expected MergedCommitIDs 'merge-commit-123', got %v", mr.MergedCommitIDs)
	}
}

func TestListMergeRequests(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewMergeRequestService(db, nil)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create multiple merge requests
	for i := 1; i <= 3; i++ {
		_, err := service.CreateMergeRequest(
			"testuser", "test-repo",
			"requestuser", "request-repo",
			"main", "feature-branch-"+string(rune('0'+i)),
			"Test MR "+string(rune('0'+i)), "MR description",
			"abc"+string(rune('0'+i)), "def"+string(rune('0'+i)),
			false, "testuser",
		)
		if err != nil {
			t.Fatalf("CreateMergeRequest %d failed: %v", i, err)
		}
	}

	// List all merge requests
	mrs, err := service.ListMergeRequests("testuser", "test-repo", "", 10, 0)
	if err != nil {
		t.Fatalf("ListMergeRequests failed: %v", err)
	}

	if len(mrs) != 3 {
		t.Errorf("Expected 3 merge requests, got %d", len(mrs))
	}

	// Verify all MRs are present (order may vary when created in same second)
	titles := make(map[string]bool)
	for _, mr := range mrs {
		titles[mr.Title] = true
	}
	for i := 1; i <= 3; i++ {
		expected := "Test MR " + string(rune('0'+i))
		if !titles[expected] {
			t.Errorf("Expected MR title '%s' not found", expected)
		}
	}
}

func TestListMergeRequests_FilterByState(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewMergeRequestService(db, nil)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create 3 merge requests
	mr1, err := service.CreateMergeRequest(
		"testuser", "test-repo",
		"requestuser", "request-repo",
		"main", "feature-branch-1",
		"Test MR 1", "MR description",
		"abc1", "def1",
		false, "testuser",
	)
	if err != nil {
		t.Fatalf("CreateMergeRequest 1 failed: %v", err)
	}

	_, err = service.CreateMergeRequest(
		"testuser", "test-repo",
		"requestuser", "request-repo",
		"main", "feature-branch-2",
		"Test MR 2", "MR description",
		"abc2", "def2",
		false, "testuser",
	)
	if err != nil {
		t.Fatalf("CreateMergeRequest 2 failed: %v", err)
	}

	mr3, err := service.CreateMergeRequest(
		"testuser", "test-repo",
		"requestuser", "request-repo",
		"main", "feature-branch-3",
		"Test MR 3", "MR description",
		"abc3", "def3",
		false, "testuser",
	)
	if err != nil {
		t.Fatalf("CreateMergeRequest 3 failed: %v", err)
	}

	// Merge MR 1
	mergedCommits := `[]`
	mergedFileChanges := `[]`
	err = service.MergeMergeRequest(
		"testuser", "test-repo", mr1.IssueID,
		"merge-commit-1",
		&mergedCommits,
		&mergedFileChanges,
	)
	if err != nil {
		t.Fatalf("MergeMergeRequest failed: %v", err)
	}

	// Close MR 3 (without merging)
	err = db.Model(&model.Issue{}).
		Where("user_name = ? AND repository_name = ? AND issue_id = ?",
			"testuser", "test-repo", mr3.IssueID).
		Updates(map[string]interface{}{
			"closed":       true,
			"updated_date": time.Now(),
		}).Error
	if err != nil {
		t.Fatalf("Failed to close MR 3: %v", err)
	}

	// List open merge requests
	openMRs, err := service.ListMergeRequests("testuser", "test-repo", "open", 10, 0)
	if err != nil {
		t.Fatalf("ListMergeRequests (open) failed: %v", err)
	}
	if len(openMRs) != 1 {
		t.Errorf("Expected 1 open merge request, got %d", len(openMRs))
	}
	if openMRs[0].IssueID != mr1.IssueID && openMRs[0].IssueID != 2 {
		// Should be MR 2 (the only one still open)
		t.Errorf("Expected open MR to be MR 2, got IssueID %d", openMRs[0].IssueID)
	}

	// List merged merge requests
	mergedMRs, err := service.ListMergeRequests("testuser", "test-repo", "merged", 10, 0)
	if err != nil {
		t.Fatalf("ListMergeRequests (merged) failed: %v", err)
	}
	if len(mergedMRs) != 1 {
		t.Errorf("Expected 1 merged merge request, got %d", len(mergedMRs))
	}
	if mergedMRs[0].IssueID != mr1.IssueID {
		t.Errorf("Expected merged MR to be MR 1, got IssueID %d", mergedMRs[0].IssueID)
	}

	// List closed merge requests (closed but not merged)
	closedMRs, err := service.ListMergeRequests("testuser", "test-repo", "closed", 10, 0)
	if err != nil {
		t.Fatalf("ListMergeRequests (closed) failed: %v", err)
	}
	if len(closedMRs) != 1 {
		t.Errorf("Expected 1 closed merge request, got %d", len(closedMRs))
	}
	if closedMRs[0].IssueID != mr3.IssueID {
		t.Errorf("Expected closed MR to be MR 3, got IssueID %d", closedMRs[0].IssueID)
	}
}
