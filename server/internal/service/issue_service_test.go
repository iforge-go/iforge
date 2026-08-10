package service

import (
	"testing"
)

func TestIssueService_CreateIssue(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewIssueService(db)

	// Create an issue
	content := "Test issue content"
	issue, err := service.CreateIssue("testuser", "test-repo", "testuser", "Test Issue", &content, nil, nil, false)
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}

	if issue.IssueID != 1 {
		t.Errorf("Expected IssueID 1, got %d", issue.IssueID)
	}
	if issue.Title != "Test Issue" {
		t.Errorf("Expected Title 'Test Issue', got '%s'", issue.Title)
	}
	if issue.Closed {
		t.Errorf("Expected Closed false, got true")
	}
}

func TestIssueService_CreateIssue_IncrementID(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewIssueService(db)

	// Create first issue
	issue1, err := service.CreateIssue("testuser", "test-repo", "testuser", "Issue 1", nil, nil, nil, false)
	if err != nil {
		t.Fatalf("CreateIssue 1 failed: %v", err)
	}
	if issue1.IssueID != 1 {
		t.Errorf("Expected first IssueID 1, got %d", issue1.IssueID)
	}

	// Create second issue
	issue2, err := service.CreateIssue("testuser", "test-repo", "testuser", "Issue 2", nil, nil, nil, false)
	if err != nil {
		t.Fatalf("CreateIssue 2 failed: %v", err)
	}
	if issue2.IssueID != 2 {
		t.Errorf("Expected second IssueID 2, got %d", issue2.IssueID)
	}
}

func TestIssueService_GetIssue(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewIssueService(db)

	// Create an issue
	created, err := service.CreateIssue("testuser", "test-repo", "testuser", "Test Issue", nil, nil, nil, false)
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}

	// Get the issue
	issue, err := service.GetIssue("testuser", "test-repo", created.IssueID)
	if err != nil {
		t.Fatalf("GetIssue failed: %v", err)
	}

	if issue.IssueID != created.IssueID {
		t.Errorf("Expected IssueID %d, got %d", created.IssueID, issue.IssueID)
	}
	if issue.Title != "Test Issue" {
		t.Errorf("Expected Title 'Test Issue', got '%s'", issue.Title)
	}
}

func TestIssueService_GetIssue_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewIssueService(db)

	_, err := service.GetIssue("testuser", "test-repo", 999)
	if err == nil {
		t.Fatal("Expected error for non-existent issue, got nil")
	}
	if err != ErrIssueNotFound {
		t.Errorf("Expected ErrIssueNotFound, got %v", err)
	}
}

func TestIssueService_UpdateIssue(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewIssueService(db)

	// Create an issue
	issue, err := service.CreateIssue("testuser", "test-repo", "testuser", "Original Title", nil, nil, nil, false)
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}

	// Update the issue
	issue.Title = "Updated Title"
	if err := service.UpdateIssue(issue); err != nil {
		t.Fatalf("UpdateIssue failed: %v", err)
	}

	// Verify update
	updated, err := service.GetIssue("testuser", "test-repo", issue.IssueID)
	if err != nil {
		t.Fatalf("GetIssue failed: %v", err)
	}
	if updated.Title != "Updated Title" {
		t.Errorf("Expected Title 'Updated Title', got '%s'", updated.Title)
	}
}

func TestIssueService_ListIssues(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewIssueService(db)

	// Create issues
	for i := 1; i <= 3; i++ {
		_, err := service.CreateIssue("testuser", "test-repo", "testuser", "Issue", nil, nil, nil, false)
		if err != nil {
			t.Fatalf("CreateIssue %d failed: %v", i, err)
		}
	}

	// List open issues
	issues, err := service.ListIssues("testuser", "test-repo", false, 10, 0)
	if err != nil {
		t.Fatalf("ListIssues failed: %v", err)
	}
	if len(issues) != 3 {
		t.Errorf("Expected 3 issues, got %d", len(issues))
	}

	// List closed issues (should be empty)
	closedIssues, err := service.ListIssues("testuser", "test-repo", true, 10, 0)
	if err != nil {
		t.Fatalf("ListIssues (closed) failed: %v", err)
	}
	if len(closedIssues) != 0 {
		t.Errorf("Expected 0 closed issues, got %d", len(closedIssues))
	}
}

func TestIssueService_ListIssues_Pagination(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewIssueService(db)

	// Create 5 issues
	for i := 1; i <= 5; i++ {
		_, err := service.CreateIssue("testuser", "test-repo", "testuser", "Issue", nil, nil, nil, false)
		if err != nil {
			t.Fatalf("CreateIssue %d failed: %v", i, err)
		}
	}

	// List with limit
	issues, err := service.ListIssues("testuser", "test-repo", false, 2, 0)
	if err != nil {
		t.Fatalf("ListIssues failed: %v", err)
	}
	if len(issues) != 2 {
		t.Errorf("Expected 2 issues with limit, got %d", len(issues))
	}

	// List with offset
	issues2, err := service.ListIssues("testuser", "test-repo", false, 10, 3)
	if err != nil {
		t.Fatalf("ListIssues with offset failed: %v", err)
	}
	if len(issues2) != 2 {
		t.Errorf("Expected 2 issues with offset, got %d", len(issues2))
	}
}

func TestIssueService_AddComment(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewIssueService(db)

	// Create an issue
	issue, err := service.CreateIssue("testuser", "test-repo", "testuser", "Test Issue", nil, nil, nil, false)
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}

	// Add a comment
	comment, err := service.AddComment("testuser", "test-repo", issue.IssueID, "commenter", "comment", "This is a comment")
	if err != nil {
		t.Fatalf("AddComment failed: %v", err)
	}

	if comment.Content != "This is a comment" {
		t.Errorf("Expected Content 'This is a comment', got '%s'", comment.Content)
	}
	if comment.CommentedUserName != "commenter" {
		t.Errorf("Expected CommentedUserName 'commenter', got '%s'", comment.CommentedUserName)
	}
}

func TestIssueService_GetComments(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewIssueService(db)

	// Create an issue
	issue, err := service.CreateIssue("testuser", "test-repo", "testuser", "Test Issue", nil, nil, nil, false)
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}

	// Add comments
	for i := 1; i <= 3; i++ {
		_, err := service.AddComment("testuser", "test-repo", issue.IssueID, "commenter", "comment", "Comment")
		if err != nil {
			t.Fatalf("AddComment %d failed: %v", i, err)
		}
	}

	// Get comments
	comments, err := service.GetComments("testuser", "test-repo", issue.IssueID)
	if err != nil {
		t.Fatalf("GetComments failed: %v", err)
	}
	if len(comments) != 3 {
		t.Errorf("Expected 3 comments, got %d", len(comments))
	}
}

func TestIssueService_LockUnlockIssue(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewIssueService(db)

	// Create an issue
	issue, err := service.CreateIssue("testuser", "test-repo", "testuser", "Test Issue", nil, nil, nil, false)
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}

	// Lock the issue
	if err := service.LockIssue("testuser", "test-repo", issue.IssueID); err != nil {
		t.Fatalf("LockIssue failed: %v", err)
	}

	// Verify locked
	locked, err := service.GetIssue("testuser", "test-repo", issue.IssueID)
	if err != nil {
		t.Fatalf("GetIssue failed: %v", err)
	}
	if !locked.Locked {
		t.Errorf("Expected Locked true, got false")
	}

	// Unlock the issue
	if err := service.UnlockIssue("testuser", "test-repo", issue.IssueID); err != nil {
		t.Fatalf("UnlockIssue failed: %v", err)
	}

	// Verify unlocked
	unlocked, err := service.GetIssue("testuser", "test-repo", issue.IssueID)
	if err != nil {
		t.Fatalf("GetIssue failed: %v", err)
	}
	if unlocked.Locked {
		t.Errorf("Expected Locked false, got true")
	}
}

func TestIssueService_AddRemoveLabel(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewIssueService(db)

	// Create a label
	label, err := service.CreateLabel("testuser", "test-repo", "bug", "#ff0000")
	if err != nil {
		t.Fatalf("CreateLabel failed: %v", err)
	}

	// Create an issue
	issue, err := service.CreateIssue("testuser", "test-repo", "testuser", "Test Issue", nil, nil, nil, false)
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}

	// Add label to issue
	if err := service.AddLabel("testuser", "test-repo", issue.IssueID, label.LabelID); err != nil {
		t.Fatalf("AddLabel failed: %v", err)
	}

	// Get labels for issue
	labels, err := service.GetLabels("testuser", "test-repo", issue.IssueID)
	if err != nil {
		t.Fatalf("GetLabels failed: %v", err)
	}
	if len(labels) != 1 {
		t.Errorf("Expected 1 label, got %d", len(labels))
	}

	// Remove label from issue
	if err := service.RemoveLabel("testuser", "test-repo", issue.IssueID, label.LabelID); err != nil {
		t.Fatalf("RemoveLabel failed: %v", err)
	}

	// Verify removal
	labels2, err := service.GetLabels("testuser", "test-repo", issue.IssueID)
	if err != nil {
		t.Fatalf("GetLabels failed: %v", err)
	}
	if len(labels2) != 0 {
		t.Errorf("Expected 0 labels after removal, got %d", len(labels2))
	}
}

func TestIssueService_AddRemoveAssignee(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewIssueService(db)

	// Create a user account (needed for JOIN in GetAssignees)
	accountService := NewAccountService(db, nil)
	_, err := accountService.CreateAccount("assignee1", "password123", "Assignee One", "a1@example.com", false, nil, nil)
	if err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}

	// Create an issue
	issue, err := service.CreateIssue("testuser", "test-repo", "testuser", "Test Issue", nil, nil, nil, false)
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}

	// Add assignee
	if err := service.AddAssignee("testuser", "test-repo", issue.IssueID, "assignee1"); err != nil {
		t.Fatalf("AddAssignee failed: %v", err)
	}

	// Get assignees
	assignees, err := service.GetAssignees("testuser", "test-repo", issue.IssueID)
	if err != nil {
		t.Fatalf("GetAssignees failed: %v", err)
	}
	if len(assignees) != 1 {
		t.Errorf("Expected 1 assignee, got %d", len(assignees))
	}

	// Remove assignee
	if err := service.RemoveAssignee("testuser", "test-repo", issue.IssueID, "assignee1"); err != nil {
		t.Fatalf("RemoveAssignee failed: %v", err)
	}

	// Verify removal
	assignees2, err := service.GetAssignees("testuser", "test-repo", issue.IssueID)
	if err != nil {
		t.Fatalf("GetAssignees failed: %v", err)
	}
	if len(assignees2) != 0 {
		t.Errorf("Expected 0 assignees after removal, got %d", len(assignees2))
	}
}

func TestIssueService_SetAssignees(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewIssueService(db)

	// Create user accounts
	accountService := NewAccountService(db, nil)
	for _, name := range []string{"user1", "user2", "user3"} {
		_, err := accountService.CreateAccount(name, "password123", name, name+"@example.com", false, nil, nil)
		if err != nil {
			t.Fatalf("CreateAccount(%s) failed: %v", name, err)
		}
	}

	// Create an issue
	issue, err := service.CreateIssue("testuser", "test-repo", "testuser", "Test Issue", nil, nil, nil, false)
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}

	// Set initial assignees
	if err := service.SetAssignees("testuser", "test-repo", issue.IssueID, []string{"user1", "user2"}); err != nil {
		t.Fatalf("SetAssignees failed: %v", err)
	}

	assignees, err := service.GetAssignees("testuser", "test-repo", issue.IssueID)
	if err != nil {
		t.Fatalf("GetAssignees failed: %v", err)
	}
	if len(assignees) != 2 {
		t.Errorf("Expected 2 assignees, got %d", len(assignees))
	}

	// Replace assignees
	if err := service.SetAssignees("testuser", "test-repo", issue.IssueID, []string{"user3"}); err != nil {
		t.Fatalf("SetAssignees (replace) failed: %v", err)
	}

	assignees2, err := service.GetAssignees("testuser", "test-repo", issue.IssueID)
	if err != nil {
		t.Fatalf("GetAssignees failed: %v", err)
	}
	if len(assignees2) != 1 {
		t.Errorf("Expected 1 assignee after replace, got %d", len(assignees2))
	}
}

func TestIssueService_CreateLabel(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewIssueService(db)

	label, err := service.CreateLabel("testuser", "test-repo", "bug", "#ff0000")
	if err != nil {
		t.Fatalf("CreateLabel failed: %v", err)
	}

	if label.LabelName != "bug" {
		t.Errorf("Expected LabelName 'bug', got '%s'", label.LabelName)
	}
	if label.Color != "#ff0000" {
		t.Errorf("Expected Color '#ff0000', got '%s'", label.Color)
	}
}

func TestIssueService_GetRepositoryLabels(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewIssueService(db)

	// Create labels
	for _, name := range []string{"bug", "feature", "enhancement"} {
		_, err := service.CreateLabel("testuser", "test-repo", name, "#000000")
		if err != nil {
			t.Fatalf("CreateLabel(%s) failed: %v", name, err)
		}
	}

	labels, err := service.GetRepositoryLabels("testuser", "test-repo")
	if err != nil {
		t.Fatalf("GetRepositoryLabels failed: %v", err)
	}
	if len(labels) != 3 {
		t.Errorf("Expected 3 labels, got %d", len(labels))
	}
}

func TestIssueService_DeleteIssue(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewIssueService(db)

	// Create an issue with comment and label
	issue, err := service.CreateIssue("testuser", "test-repo", "testuser", "Test Issue", nil, nil, nil, false)
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}

	// Add a comment
	_, err = service.AddComment("testuser", "test-repo", issue.IssueID, "user", "comment", "content")
	if err != nil {
		t.Fatalf("AddComment failed: %v", err)
	}

	// Add a label
	label, err := service.CreateLabel("testuser", "test-repo", "bug", "#ff0000")
	if err != nil {
		t.Fatalf("CreateLabel failed: %v", err)
	}
	if err := service.AddLabel("testuser", "test-repo", issue.IssueID, label.LabelID); err != nil {
		t.Fatalf("AddLabel failed: %v", err)
	}

	// Delete the issue
	if err := service.DeleteIssue("testuser", "test-repo", issue.IssueID); err != nil {
		t.Fatalf("DeleteIssue failed: %v", err)
	}

	// Verify issue is deleted
	_, err = service.GetIssue("testuser", "test-repo", issue.IssueID)
	if err != ErrIssueNotFound {
		t.Errorf("Expected ErrIssueNotFound after deletion, got %v", err)
	}

	// Verify comments are deleted
	comments, err := service.GetComments("testuser", "test-repo", issue.IssueID)
	if err != nil {
		t.Fatalf("GetComments failed: %v", err)
	}
	if len(comments) != 0 {
		t.Errorf("Expected 0 comments after issue deletion, got %d", len(comments))
	}
}

func TestIssueService_DeleteComment(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewIssueService(db)

	// Create an issue
	issue, err := service.CreateIssue("testuser", "test-repo", "testuser", "Test Issue", nil, nil, nil, false)
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}

	// Add a comment
	comment, err := service.AddComment("testuser", "test-repo", issue.IssueID, "user", "comment", "content")
	if err != nil {
		t.Fatalf("AddComment failed: %v", err)
	}

	// Delete the comment
	if err := service.DeleteComment("testuser", "test-repo", comment.CommentID); err != nil {
		t.Fatalf("DeleteComment failed: %v", err)
	}

	// Verify deletion
	comments, err := service.GetComments("testuser", "test-repo", issue.IssueID)
	if err != nil {
		t.Fatalf("GetComments failed: %v", err)
	}
	if len(comments) != 0 {
		t.Errorf("Expected 0 comments after deletion, got %d", len(comments))
	}
}

func TestIssueService_BatchUpdateIssues(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewIssueService(db)

	// Create issues
	var issueIDs []int
	for i := 1; i <= 3; i++ {
		issue, err := service.CreateIssue("testuser", "test-repo", "testuser", "Issue", nil, nil, nil, false)
		if err != nil {
			t.Fatalf("CreateIssue %d failed: %v", i, err)
		}
		issueIDs = append(issueIDs, issue.IssueID)
	}

	// Batch close issues
	closed := true
	if err := service.BatchUpdateIssues("testuser", "test-repo", issueIDs, &closed, nil, nil, nil, nil); err != nil {
		t.Fatalf("BatchUpdateIssues failed: %v", err)
	}

	// Verify all issues are closed
	for _, id := range issueIDs {
		issue, err := service.GetIssue("testuser", "test-repo", id)
		if err != nil {
			t.Fatalf("GetIssue(%d) failed: %v", id, err)
		}
		if !issue.Closed {
			t.Errorf("Expected issue %d to be closed", id)
		}
	}
}

func TestIssueService_BatchUpdateIssues_AddLabels(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewIssueService(db)

	// Create issues
	var issueIDs []int
	for i := 1; i <= 2; i++ {
		issue, err := service.CreateIssue("testuser", "test-repo", "testuser", "Issue", nil, nil, nil, false)
		if err != nil {
			t.Fatalf("CreateIssue %d failed: %v", i, err)
		}
		issueIDs = append(issueIDs, issue.IssueID)
	}

	// Create labels
	label1, err := service.CreateLabel("testuser", "test-repo", "bug", "#ff0000")
	if err != nil {
		t.Fatalf("CreateLabel failed: %v", err)
	}
	label2, err := service.CreateLabel("testuser", "test-repo", "urgent", "#00ff00")
	if err != nil {
		t.Fatalf("CreateLabel failed: %v", err)
	}

	// Batch add labels
	if err := service.BatchUpdateIssues("testuser", "test-repo", issueIDs, nil, nil, nil, []int{label1.LabelID, label2.LabelID}, nil); err != nil {
		t.Fatalf("BatchUpdateIssues (add labels) failed: %v", err)
	}

	// Verify labels were added to all issues
	for _, id := range issueIDs {
		labels, err := service.GetLabels("testuser", "test-repo", id)
		if err != nil {
			t.Fatalf("GetLabels(%d) failed: %v", id, err)
		}
		if len(labels) != 2 {
			t.Errorf("Expected 2 labels for issue %d, got %d", id, len(labels))
		}
	}
}

func TestIssueService_ListIssuesWithLabels(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewIssueService(db)

	// Create labels
	label1, err := service.CreateLabel("testuser", "test-repo", "bug", "#ff0000")
	if err != nil {
		t.Fatalf("CreateLabel failed: %v", err)
	}
	label2, err := service.CreateLabel("testuser", "test-repo", "feature", "#00ff00")
	if err != nil {
		t.Fatalf("CreateLabel failed: %v", err)
	}

	// Create issues
	issue1, err := service.CreateIssue("testuser", "test-repo", "testuser", "Issue 1", nil, nil, nil, false)
	if err != nil {
		t.Fatalf("CreateIssue 1 failed: %v", err)
	}
	issue2, err := service.CreateIssue("testuser", "test-repo", "testuser", "Issue 2", nil, nil, nil, false)
	if err != nil {
		t.Fatalf("CreateIssue 2 failed: %v", err)
	}

	// Add labels to issues
	if err := service.AddLabel("testuser", "test-repo", issue1.IssueID, label1.LabelID); err != nil {
		t.Fatalf("AddLabel failed: %v", err)
	}
	if err := service.AddLabel("testuser", "test-repo", issue1.IssueID, label2.LabelID); err != nil {
		t.Fatalf("AddLabel failed: %v", err)
	}
	if err := service.AddLabel("testuser", "test-repo", issue2.IssueID, label1.LabelID); err != nil {
		t.Fatalf("AddLabel failed: %v", err)
	}

	// List issues with labels
	issuesWithLabels, err := service.ListIssuesWithLabels("testuser", "test-repo", false, 10, 0)
	if err != nil {
		t.Fatalf("ListIssuesWithLabels failed: %v", err)
	}
	if len(issuesWithLabels) != 2 {
		t.Errorf("Expected 2 issues with labels, got %d", len(issuesWithLabels))
	}
}

func TestIssueService_GetIssueAuthors(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewIssueService(db)

	// Create user accounts
	accountService := NewAccountService(db, nil)
	for _, name := range []string{"author1", "author2"} {
		_, err := accountService.CreateAccount(name, "password123", name, name+"@example.com", false, nil, nil)
		if err != nil {
			t.Fatalf("CreateAccount(%s) failed: %v", name, err)
		}
	}

	// Create issues by different authors
	_, err := service.CreateIssue("testuser", "test-repo", "author1", "Issue 1", nil, nil, nil, false)
	if err != nil {
		t.Fatalf("CreateIssue 1 failed: %v", err)
	}
	_, err = service.CreateIssue("testuser", "test-repo", "author2", "Issue 2", nil, nil, nil, false)
	if err != nil {
		t.Fatalf("CreateIssue 2 failed: %v", err)
	}
	_, err = service.CreateIssue("testuser", "test-repo", "author1", "Issue 3", nil, nil, nil, false)
	if err != nil {
		t.Fatalf("CreateIssue 3 failed: %v", err)
	}

	// Get issue authors
	authors, err := service.GetIssueAuthors("testuser", "test-repo", false)
	if err != nil {
		t.Fatalf("GetIssueAuthors failed: %v", err)
	}
	if len(authors) != 2 {
		t.Errorf("Expected 2 distinct authors, got %d", len(authors))
	}
}

func TestIssueService_UpdateComment(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewIssueService(db)

	// Create an issue
	issue, err := service.CreateIssue("testuser", "test-repo", "testuser", "Test Issue", nil, nil, nil, false)
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}

	// Add a comment
	comment, err := service.AddComment("testuser", "test-repo", issue.IssueID, "user", "comment", "Original content")
	if err != nil {
		t.Fatalf("AddComment failed: %v", err)
	}

	// Update the comment
	comment.Content = "Updated content"
	if err := service.UpdateComment(comment); err != nil {
		t.Fatalf("UpdateComment failed: %v", err)
	}

	// Verify update
	updated, err := service.GetComment("testuser", "test-repo", comment.CommentID)
	if err != nil {
		t.Fatalf("GetComment failed: %v", err)
	}
	if updated.Content != "Updated content" {
		t.Errorf("Expected Content 'Updated content', got '%s'", updated.Content)
	}
}
