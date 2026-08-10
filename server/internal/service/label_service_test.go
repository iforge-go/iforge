package service

import (
	"iforge/iforge/internal/model"
	"testing"
)

func TestLabelService_CreateLabel(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewLabelService(db)

	// Create a label
	label, err := service.CreateLabel("owner", "repo", "bug", "#ff0000")
	if err != nil {
		t.Fatalf("CreateLabel failed: %v", err)
	}

	if label.UserName != "owner" {
		t.Errorf("Expected UserName 'owner', got '%s'", label.UserName)
	}
	if label.RepositoryName != "repo" {
		t.Errorf("Expected RepositoryName 'repo', got '%s'", label.RepositoryName)
	}
	if label.LabelName != "bug" {
		t.Errorf("Expected LabelName 'bug', got '%s'", label.LabelName)
	}
	if label.Color != "#ff0000" {
		t.Errorf("Expected Color '#ff0000', got '%s'", label.Color)
	}
}

func TestLabelService_CreateLabel_InvalidName(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewLabelService(db)

	// Try to create label with empty name
	_, err := service.CreateLabel("owner", "repo", "", "#ff0000")
	if err == nil {
		t.Fatal("Expected error for empty label name, got nil")
	}
	if err != ErrInvalidLabelName {
		t.Errorf("Expected ErrInvalidLabelName, got %v", err)
	}

	// Try to create label with name too long
	longName := ""
	for i := 0; i < 101; i++ {
		longName += "a"
	}
	_, err = service.CreateLabel("owner", "repo", longName, "#ff0000")
	if err == nil {
		t.Fatal("Expected error for label name too long, got nil")
	}
	if err != ErrInvalidLabelName {
		t.Errorf("Expected ErrInvalidLabelName, got %v", err)
	}
}

func TestLabelService_ListLabels(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewLabelService(db)

	// Create labels
	_, err := service.CreateLabel("owner", "repo", "bug", "#ff0000")
	if err != nil {
		t.Fatalf("CreateLabel failed: %v", err)
	}
	_, err = service.CreateLabel("owner", "repo", "feature", "#00ff00")
	if err != nil {
		t.Fatalf("CreateLabel failed: %v", err)
	}

	// List labels
	labels, err := service.ListLabels("owner", "repo")
	if err != nil {
		t.Fatalf("ListLabels failed: %v", err)
	}

	if len(labels) != 2 {
		t.Errorf("Expected 2 labels, got %d", len(labels))
	}
}

func TestLabelService_GetLabel(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewLabelService(db)

	// Create a label
	created, err := service.CreateLabel("owner", "repo", "bug", "#ff0000")
	if err != nil {
		t.Fatalf("CreateLabel failed: %v", err)
	}

	// Get the label
	label, err := service.GetLabel("owner", "repo", created.LabelID)
	if err != nil {
		t.Fatalf("GetLabel failed: %v", err)
	}

	if label.LabelName != "bug" {
		t.Errorf("Expected LabelName 'bug', got '%s'", label.LabelName)
	}
}

func TestLabelService_GetLabel_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewLabelService(db)

	// Try to get non-existent label
	_, err := service.GetLabel("owner", "repo", 999)
	if err == nil {
		t.Fatal("Expected error for non-existent label, got nil")
	}
	if err != ErrLabelNotFound {
		t.Errorf("Expected ErrLabelNotFound, got %v", err)
	}
}

func TestLabelService_UpdateLabel(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewLabelService(db)

	// Create a label
	created, err := service.CreateLabel("owner", "repo", "bug", "#ff0000")
	if err != nil {
		t.Fatalf("CreateLabel failed: %v", err)
	}

	// Update the label
	err = service.UpdateLabel("owner", "repo", created.LabelID, "defect", "#0000ff")
	if err != nil {
		t.Fatalf("UpdateLabel failed: %v", err)
	}

	// Verify update
	label, err := service.GetLabel("owner", "repo", created.LabelID)
	if err != nil {
		t.Fatalf("GetLabel failed: %v", err)
	}

	if label.LabelName != "defect" {
		t.Errorf("Expected LabelName 'defect', got '%s'", label.LabelName)
	}
	if label.Color != "#0000ff" {
		t.Errorf("Expected Color '#0000ff', got '%s'", label.Color)
	}
}

func TestLabelService_DeleteLabel(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewLabelService(db)

	// Create a label
	created, err := service.CreateLabel("owner", "repo", "bug", "#ff0000")
	if err != nil {
		t.Fatalf("CreateLabel failed: %v", err)
	}

	// Delete the label
	err = service.DeleteLabel("owner", "repo", created.LabelID)
	if err != nil {
		t.Fatalf("DeleteLabel failed: %v", err)
	}

	// Verify deletion
	_, err = service.GetLabel("owner", "repo", created.LabelID)
	if err == nil {
		t.Fatal("Expected error after deletion, got nil")
	}
}

func TestLabelService_AddLabelToIssue(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewLabelService(db)

	// Create a label
	label, err := service.CreateLabel("owner", "repo", "bug", "#ff0000")
	if err != nil {
		t.Fatalf("CreateLabel failed: %v", err)
	}

	// Add label to issue
	err = service.AddLabelToIssue("owner", "repo", 1, label.LabelID)
	if err != nil {
		t.Fatalf("AddLabelToIssue failed: %v", err)
	}

	// Verify by getting issue labels
	labels, err := service.GetIssueLabels("owner", "repo", 1)
	if err != nil {
		t.Fatalf("GetIssueLabels failed: %v", err)
	}

	if len(labels) != 1 {
		t.Errorf("Expected 1 label, got %d", len(labels))
	}
}

func TestLabelService_AddLabelToIssue_Duplicate(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewLabelService(db)

	// Create a label
	label, err := service.CreateLabel("owner", "repo", "bug", "#ff0000")
	if err != nil {
		t.Fatalf("CreateLabel failed: %v", err)
	}

	// Add label to issue
	err = service.AddLabelToIssue("owner", "repo", 1, label.LabelID)
	if err != nil {
		t.Fatalf("AddLabelToIssue failed: %v", err)
	}

	// Try to add again (should not error, just skip)
	err = service.AddLabelToIssue("owner", "repo", 1, label.LabelID)
	if err != nil {
		t.Fatalf("AddLabelToIssue (duplicate) failed: %v", err)
	}

	// Verify only one label
	labels, err := service.GetIssueLabels("owner", "repo", 1)
	if err != nil {
		t.Fatalf("GetIssueLabels failed: %v", err)
	}

	if len(labels) != 1 {
		t.Errorf("Expected 1 label, got %d", len(labels))
	}
}

func TestLabelService_RemoveLabelFromIssue(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewLabelService(db)

	// Create a label
	label, err := service.CreateLabel("owner", "repo", "bug", "#ff0000")
	if err != nil {
		t.Fatalf("CreateLabel failed: %v", err)
	}

	// Add label to issue
	err = service.AddLabelToIssue("owner", "repo", 1, label.LabelID)
	if err != nil {
		t.Fatalf("AddLabelToIssue failed: %v", err)
	}

	// Remove label from issue
	err = service.RemoveLabelFromIssue("owner", "repo", 1, label.LabelID)
	if err != nil {
		t.Fatalf("RemoveLabelFromIssue failed: %v", err)
	}

	// Verify removal
	labels, err := service.GetIssueLabels("owner", "repo", 1)
	if err != nil {
		t.Fatalf("GetIssueLabels failed: %v", err)
	}

	if len(labels) != 0 {
		t.Errorf("Expected 0 labels, got %d", len(labels))
	}
}

func TestLabelService_GetIssueLabels(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewLabelService(db)

	// Create labels
	label1, err := service.CreateLabel("owner", "repo", "bug", "#ff0000")
	if err != nil {
		t.Fatalf("CreateLabel failed: %v", err)
	}
	label2, err := service.CreateLabel("owner", "repo", "feature", "#00ff00")
	if err != nil {
		t.Fatalf("CreateLabel failed: %v", err)
	}

	// Add labels to issue
	err = service.AddLabelToIssue("owner", "repo", 1, label1.LabelID)
	if err != nil {
		t.Fatalf("AddLabelToIssue failed: %v", err)
	}
	err = service.AddLabelToIssue("owner", "repo", 1, label2.LabelID)
	if err != nil {
		t.Fatalf("AddLabelToIssue failed: %v", err)
	}

	// Get issue labels
	labels, err := service.GetIssueLabels("owner", "repo", 1)
	if err != nil {
		t.Fatalf("GetIssueLabels failed: %v", err)
	}

	if len(labels) != 2 {
		t.Errorf("Expected 2 labels, got %d", len(labels))
	}
}

func TestValidateLabelName(t *testing.T) {
	// Test valid name
	err := validateLabelName("bug")
	if err != nil {
		t.Errorf("Expected no error for valid name, got %v", err)
	}

	// Test empty name
	err = validateLabelName("")
	if err == nil {
		t.Error("Expected error for empty name, got nil")
	}
	if err != ErrInvalidLabelName {
		t.Errorf("Expected ErrInvalidLabelName, got %v", err)
	}

	// Test name too long
	longName := ""
	for i := 0; i < 101; i++ {
		longName += "a"
	}
	err = validateLabelName(longName)
	if err == nil {
		t.Error("Expected error for name too long, got nil")
	}
	if err != ErrInvalidLabelName {
		t.Errorf("Expected ErrInvalidLabelName, got %v", err)
	}
}

func TestLabelService_CreateLabel_Duplicate(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewLabelService(db)

	// Create first label
	_, err := service.CreateLabel("owner", "repo", "bug", "#ff0000")
	if err != nil {
		t.Fatalf("First CreateLabel failed: %v", err)
	}

	// Try to create duplicate (same name, same repo)
	_, err = service.CreateLabel("owner", "repo", "bug", "#00ff00")
	if err == nil {
		t.Fatal("Expected error for duplicate label, got nil")
	}
}

func TestLabelService_ListLabels_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewLabelService(db)

	// List labels for non-existent repo
	labels, err := service.ListLabels("owner", "nonexistent")
	if err != nil {
		t.Fatalf("ListLabels failed: %v", err)
	}

	if len(labels) != 0 {
		t.Errorf("Expected 0 labels, got %d", len(labels))
	}
}

func TestLabelService_GetIssueLabels_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewLabelService(db)

	// Get labels for issue with no labels
	labels, err := service.GetIssueLabels("owner", "repo", 999)
	if err != nil {
		t.Fatalf("GetIssueLabels failed: %v", err)
	}

	if len(labels) != 0 {
		t.Errorf("Expected 0 labels, got %d", len(labels))
	}
}

func TestLabelService_DeleteLabel_WithIssueLabels(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewLabelService(db)

	// Create a label
	label, err := service.CreateLabel("owner", "repo", "bug", "#ff0000")
	if err != nil {
		t.Fatalf("CreateLabel failed: %v", err)
	}

	// Add label to issue
	err = service.AddLabelToIssue("owner", "repo", 1, label.LabelID)
	if err != nil {
		t.Fatalf("AddLabelToIssue failed: %v", err)
	}

	// Delete the label (should also delete issue_label entries)
	err = service.DeleteLabel("owner", "repo", label.LabelID)
	if err != nil {
		t.Fatalf("DeleteLabel failed: %v", err)
	}

	// Verify issue has no labels
	labels, err := service.GetIssueLabels("owner", "repo", 1)
	if err != nil {
		t.Fatalf("GetIssueLabels failed: %v", err)
	}

	if len(labels) != 0 {
		t.Errorf("Expected 0 labels after deletion, got %d", len(labels))
	}
}

func TestLabelService_MultipleRepos(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewLabelService(db)

	// Create labels in different repos
	_, err := service.CreateLabel("owner1", "repo1", "bug", "#ff0000")
	if err != nil {
		t.Fatalf("CreateLabel failed: %v", err)
	}
	_, err = service.CreateLabel("owner2", "repo2", "bug", "#00ff00")
	if err != nil {
		t.Fatalf("CreateLabel failed: %v", err)
	}

	// List labels for each repo
	labels1, err := service.ListLabels("owner1", "repo1")
	if err != nil {
		t.Fatalf("ListLabels failed: %v", err)
	}
	if len(labels1) != 1 {
		t.Errorf("Expected 1 label in repo1, got %d", len(labels1))
	}

	labels2, err := service.ListLabels("owner2", "repo2")
	if err != nil {
		t.Fatalf("ListLabels failed: %v", err)
	}
	if len(labels2) != 1 {
		t.Errorf("Expected 1 label in repo2, got %d", len(labels2))
	}
}

func TestLabelService_UpdateLabel_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewLabelService(db)

	// Try to update non-existent label
	err := service.UpdateLabel("owner", "repo", 999, "newname", "#000000")
	if err != nil {
		t.Fatalf("UpdateLabel failed: %v", err)
	}
	// Note: GORM doesn't error on update of non-existent record, it just affects 0 rows
}

func TestLabelService_RemoveLabelFromIssue_NotExists(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewLabelService(db)

	// Try to remove non-existent label-issue relationship
	err := service.RemoveLabelFromIssue("owner", "repo", 1, 999)
	if err != nil {
		t.Fatalf("RemoveLabelFromIssue failed: %v", err)
	}
	// Note: GORM doesn't error on delete of non-existent record
}

func TestLabelService_ColorFormats(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewLabelService(db)

	// Test different color formats
	colors := []string{
		"#ff0000",
		"#FF0000",
		"#f00",
		"red",
		"rgb(255, 0, 0)",
	}

	for i, color := range colors {
		label, err := service.CreateLabel("owner", "repo", "label"+string(rune('0'+i)), color)
		if err != nil {
			t.Fatalf("CreateLabel with color %s failed: %v", color, err)
		}
		if label.Color != color {
			t.Errorf("Expected color %s, got %s", color, label.Color)
		}
	}
}

func TestLabelService_LabelNameMaxLength(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewLabelService(db)

	// Test exactly 100 characters (should succeed)
	name100 := ""
	for i := 0; i < 100; i++ {
		name100 += "a"
	}
	_, err := service.CreateLabel("owner", "repo", name100, "#ff0000")
	if err != nil {
		t.Errorf("Expected no error for 100 character name, got %v", err)
	}

	// Test 101 characters (should fail)
	name101 := name100 + "a"
	_, err = service.CreateLabel("owner", "repo", name101, "#ff0000")
	if err == nil {
		t.Error("Expected error for 101 character name, got nil")
	}
	if err != ErrInvalidLabelName {
		t.Errorf("Expected ErrInvalidLabelName, got %v", err)
	}
}

func TestLabelService_SpecialCharacters(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewLabelService(db)

	// Test special characters in label names
	names := []string{
		"bug-fix",
		"feature_request",
		"priority:high",
		"status/pending",
		"label with spaces",
		"中文标签",
		"タグ",
	}

	for _, name := range names {
		label, err := service.CreateLabel("owner", "repo", name, "#ff0000")
		if err != nil {
			t.Errorf("CreateLabel with name %s failed: %v", name, err)
			continue
		}
		if label.LabelName != name {
			t.Errorf("Expected name %s, got %s", name, label.LabelName)
		}
	}
}

func TestLabelService_CaseSensitivity(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewLabelService(db)

	// Create label with lowercase
	_, err := service.CreateLabel("owner", "repo", "bug", "#ff0000")
	if err != nil {
		t.Fatalf("CreateLabel failed: %v", err)
	}

	// Try to create same label with uppercase (should succeed if case-sensitive)
	_, err = service.CreateLabel("owner", "repo", "BUG", "#00ff00")
	if err != nil {
		// If it fails, the database is case-insensitive
		t.Logf("Database appears to be case-insensitive: %v", err)
	}
}

func TestLabelService_MultipleIssuesSameLabel(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewLabelService(db)

	// Create a label
	label, err := service.CreateLabel("owner", "repo", "bug", "#ff0000")
	if err != nil {
		t.Fatalf("CreateLabel failed: %v", err)
	}

	// Add same label to multiple issues
	for i := 1; i <= 3; i++ {
		err = service.AddLabelToIssue("owner", "repo", i, label.LabelID)
		if err != nil {
			t.Fatalf("AddLabelToIssue for issue %d failed: %v", i, err)
		}
	}

	// Verify each issue has the label
	for i := 1; i <= 3; i++ {
		labels, err := service.GetIssueLabels("owner", "repo", i)
		if err != nil {
			t.Fatalf("GetIssueLabels for issue %d failed: %v", i, err)
		}
		if len(labels) != 1 {
			t.Errorf("Expected 1 label for issue %d, got %d", i, len(labels))
		}
	}
}

func TestLabelService_IssueMultipleLabels(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewLabelService(db)

	// Create multiple labels
	labelIDs := make([]int, 0)
	for i := 0; i < 5; i++ {
		label, err := service.CreateLabel("owner", "repo", "label"+string(rune('0'+i)), "#ff0000")
		if err != nil {
			t.Fatalf("CreateLabel failed: %v", err)
		}
		labelIDs = append(labelIDs, label.LabelID)
	}

	// Add all labels to same issue
	for _, labelID := range labelIDs {
		err := service.AddLabelToIssue("owner", "repo", 1, labelID)
		if err != nil {
			t.Fatalf("AddLabelToIssue failed: %v", err)
		}
	}

	// Verify issue has all labels
	labels, err := service.GetIssueLabels("owner", "repo", 1)
	if err != nil {
		t.Fatalf("GetIssueLabels failed: %v", err)
	}
	if len(labels) != 5 {
		t.Errorf("Expected 5 labels, got %d", len(labels))
	}
}

func TestLabelService_EmptyColor(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewLabelService(db)

	// Create label with empty color
	label, err := service.CreateLabel("owner", "repo", "bug", "")
	if err != nil {
		t.Fatalf("CreateLabel with empty color failed: %v", err)
	}
	if label.Color != "" {
		t.Errorf("Expected empty color, got %s", label.Color)
	}
}

func TestLabelService_Ordering(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewLabelService(db)

	// Create labels in non-alphabetical order
	names := []string{"zebra", "apple", "mango", "banana"}
	for _, name := range names {
		_, err := service.CreateLabel("owner", "repo", name, "#ff0000")
		if err != nil {
			t.Fatalf("CreateLabel failed: %v", err)
		}
	}

	// List labels (should be ordered by label_name)
	labels, err := service.ListLabels("owner", "repo")
	if err != nil {
		t.Fatalf("ListLabels failed: %v", err)
	}

	if len(labels) != 4 {
		t.Errorf("Expected 4 labels, got %d", len(labels))
	}

	// Verify ordering
	expectedOrder := []string{"apple", "banana", "mango", "zebra"}
	for i, label := range labels {
		if label.LabelName != expectedOrder[i] {
			t.Errorf("Expected label %d to be %s, got %s", i, expectedOrder[i], label.LabelName)
		}
	}
}

func TestLabelService_DeleteNonExistentLabel(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewLabelService(db)

	// Try to delete non-existent label
	err := service.DeleteLabel("owner", "repo", 999)
	if err != nil {
		t.Fatalf("DeleteLabel failed: %v", err)
	}
	// Note: GORM doesn't error on delete of non-existent record
}

func TestLabelService_ConcurrentOperations(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewLabelService(db)

	// Create a label
	label, err := service.CreateLabel("owner", "repo", "bug", "#ff0000")
	if err != nil {
		t.Fatalf("CreateLabel failed: %v", err)
	}

	// Simulate concurrent add/remove operations
	done := make(chan bool)

	go func() {
		for i := 0; i < 10; i++ {
			_ = service.AddLabelToIssue("owner", "repo", 1, label.LabelID)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			_ = service.RemoveLabelFromIssue("owner", "repo", 1, label.LabelID)
		}
		done <- true
	}()

	<-done
	<-done

	// Verify final state (should have 0 or 1 label depending on race)
	labels, err := service.GetIssueLabels("owner", "repo", 1)
	if err != nil {
		t.Fatalf("GetIssueLabels failed: %v", err)
	}
	if len(labels) > 1 {
		t.Errorf("Expected at most 1 label, got %d", len(labels))
	}
}

func TestLabelService_ModelFields(t *testing.T) {
	// Test that model fields are correctly set
	label := &model.Label{
		UserName:       "owner",
		RepositoryName: "repo",
		LabelName:      "bug",
		Color:          "#ff0000",
	}

	if label.UserName != "owner" {
		t.Errorf("Expected UserName 'owner', got '%s'", label.UserName)
	}
	if label.RepositoryName != "repo" {
		t.Errorf("Expected RepositoryName 'repo', got '%s'", label.RepositoryName)
	}
	if label.LabelName != "bug" {
		t.Errorf("Expected LabelName 'bug', got '%s'", label.LabelName)
	}
	if label.Color != "#ff0000" {
		t.Errorf("Expected Color '#ff0000', got '%s'", label.Color)
	}
}

func TestLabelService_IssueLabelModelFields(t *testing.T) {
	// Test that issue label model fields are correctly set
	issueLabel := &model.IssueLabel{
		UserName:       "owner",
		RepositoryName: "repo",
		IssueID:        1,
		LabelID:        1,
	}

	if issueLabel.UserName != "owner" {
		t.Errorf("Expected UserName 'owner', got '%s'", issueLabel.UserName)
	}
	if issueLabel.RepositoryName != "repo" {
		t.Errorf("Expected RepositoryName 'repo', got '%s'", issueLabel.RepositoryName)
	}
	if issueLabel.IssueID != 1 {
		t.Errorf("Expected IssueID 1, got %d", issueLabel.IssueID)
	}
	if issueLabel.LabelID != 1 {
		t.Errorf("Expected LabelID 1, got %d", issueLabel.LabelID)
	}
}
