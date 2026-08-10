package service

import (
	"testing"
)

func TestReviewService_CreateReview(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewReviewService(db)

	// Create a review
	content := "This looks good"
	review, err := service.CreateReview("testuser", "test-repo", 1, "reviewer", "approved", &content)
	if err != nil {
		t.Fatalf("CreateReview failed: %v", err)
	}

	if review.Reviewer != "reviewer" {
		t.Errorf("Expected Reviewer 'reviewer', got '%s'", review.Reviewer)
	}
	if review.Status != "approved" {
		t.Errorf("Expected Status 'approved', got '%s'", review.Status)
	}
	if review.Content == nil || *review.Content != content {
		t.Errorf("Expected Content '%s', got %v", content, review.Content)
	}
}

func TestReviewService_GetReview(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewReviewService(db)

	// Create a review
	content := "Test review"
	created, err := service.CreateReview("testuser", "test-repo", 1, "reviewer", "approved", &content)
	if err != nil {
		t.Fatalf("CreateReview failed: %v", err)
	}

	// Get the review
	review, err := service.GetReview(created.ReviewID)
	if err != nil {
		t.Fatalf("GetReview failed: %v", err)
	}

	if review.ReviewID != created.ReviewID {
		t.Errorf("Expected ReviewID %d, got %d", created.ReviewID, review.ReviewID)
	}
	if review.Reviewer != "reviewer" {
		t.Errorf("Expected Reviewer 'reviewer', got '%s'", review.Reviewer)
	}
}

func TestReviewService_GetReview_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewReviewService(db)

	_, err := service.GetReview(999)
	if err == nil {
		t.Fatal("Expected error for non-existent review, got nil")
	}
	if err != ErrReviewNotFound {
		t.Errorf("Expected ErrReviewNotFound, got %v", err)
	}
}

func TestReviewService_ListReviews(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewReviewService(db)

	// Create multiple reviews
	for i := 1; i <= 3; i++ {
		content := "Review"
		_, err := service.CreateReview("testuser", "test-repo", 1, "reviewer", "approved", &content)
		if err != nil {
			t.Fatalf("CreateReview %d failed: %v", i, err)
		}
	}

	// List reviews
	reviews, err := service.ListReviews("testuser", "test-repo", 1)
	if err != nil {
		t.Fatalf("ListReviews failed: %v", err)
	}
	if len(reviews) != 3 {
		t.Errorf("Expected 3 reviews, got %d", len(reviews))
	}
}

func TestReviewService_UpdateReview(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewReviewService(db)

	// Create a review
	content := "Original content"
	review, err := service.CreateReview("testuser", "test-repo", 1, "reviewer", "pending", &content)
	if err != nil {
		t.Fatalf("CreateReview failed: %v", err)
	}

	// Update the review
	newContent := "Updated content"
	if err := service.UpdateReview(review.ReviewID, "approved", &newContent); err != nil {
		t.Fatalf("UpdateReview failed: %v", err)
	}

	// Verify update
	updated, err := service.GetReview(review.ReviewID)
	if err != nil {
		t.Fatalf("GetReview failed: %v", err)
	}
	if updated.Status != "approved" {
		t.Errorf("Expected Status 'approved', got '%s'", updated.Status)
	}
	if updated.Content == nil || *updated.Content != newContent {
		t.Errorf("Expected Content '%s', got %v", newContent, updated.Content)
	}
}

func TestReviewService_DeleteReview(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewReviewService(db)

	// Create a review
	content := "Test review"
	review, err := service.CreateReview("testuser", "test-repo", 1, "reviewer", "approved", &content)
	if err != nil {
		t.Fatalf("CreateReview failed: %v", err)
	}

	// Delete the review
	if err := service.DeleteReview(review.ReviewID); err != nil {
		t.Fatalf("DeleteReview failed: %v", err)
	}

	// Verify deletion
	_, err = service.GetReview(review.ReviewID)
	if err != ErrReviewNotFound {
		t.Errorf("Expected ErrReviewNotFound after deletion, got %v", err)
	}
}

func TestReviewService_CreateReviewComment(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewReviewService(db)

	// Create a review first
	content := "Test review"
	review, err := service.CreateReview("testuser", "test-repo", 1, "reviewer", "approved", &content)
	if err != nil {
		t.Fatalf("CreateReview failed: %v", err)
	}

	// Create a review comment
	comment, err := service.CreateReviewComment(review.ReviewID, "testuser", "test-repo", 1, "commenter", "file.go", 10, "This line needs fixing")
	if err != nil {
		t.Fatalf("CreateReviewComment failed: %v", err)
	}

	if comment.Commenter != "commenter" {
		t.Errorf("Expected Commenter 'commenter', got '%s'", comment.Commenter)
	}
	if comment.FilePath != "file.go" {
		t.Errorf("Expected FilePath 'file.go', got '%s'", comment.FilePath)
	}
	if comment.Line != 10 {
		t.Errorf("Expected Line 10, got %d", comment.Line)
	}
	if comment.Content != "This line needs fixing" {
		t.Errorf("Expected Content 'This line needs fixing', got '%s'", comment.Content)
	}
}

func TestReviewService_GetReviewComment(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewReviewService(db)

	// Create a review
	content := "Test review"
	review, err := service.CreateReview("testuser", "test-repo", 1, "reviewer", "approved", &content)
	if err != nil {
		t.Fatalf("CreateReview failed: %v", err)
	}

	// Create a review comment
	created, err := service.CreateReviewComment(review.ReviewID, "testuser", "test-repo", 1, "commenter", "file.go", 10, "Comment")
	if err != nil {
		t.Fatalf("CreateReviewComment failed: %v", err)
	}

	// Get the comment
	comment, err := service.GetReviewComment(created.CommentID)
	if err != nil {
		t.Fatalf("GetReviewComment failed: %v", err)
	}

	if comment.CommentID != created.CommentID {
		t.Errorf("Expected CommentID %d, got %d", created.CommentID, comment.CommentID)
	}
	if comment.Content != "Comment" {
		t.Errorf("Expected Content 'Comment', got '%s'", comment.Content)
	}
}

func TestReviewService_GetReviewComment_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewReviewService(db)

	_, err := service.GetReviewComment(999)
	if err == nil {
		t.Fatal("Expected error for non-existent comment, got nil")
	}
	if err != ErrReviewNotFound {
		t.Errorf("Expected ErrReviewNotFound, got %v", err)
	}
}

func TestReviewService_ListReviewComments(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewReviewService(db)

	// Create a review
	content := "Test review"
	review, err := service.CreateReview("testuser", "test-repo", 1, "reviewer", "approved", &content)
	if err != nil {
		t.Fatalf("CreateReview failed: %v", err)
	}

	// Create multiple comments
	for i := 1; i <= 3; i++ {
		_, err := service.CreateReviewComment(review.ReviewID, "testuser", "test-repo", 1, "commenter", "file.go", i*10, "Comment")
		if err != nil {
			t.Fatalf("CreateReviewComment %d failed: %v", i, err)
		}
	}

	// List comments
	comments, err := service.ListReviewComments(review.ReviewID)
	if err != nil {
		t.Fatalf("ListReviewComments failed: %v", err)
	}
	if len(comments) != 3 {
		t.Errorf("Expected 3 comments, got %d", len(comments))
	}
}

func TestReviewService_UpdateReviewComment(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewReviewService(db)

	// Create a review
	content := "Test review"
	review, err := service.CreateReview("testuser", "test-repo", 1, "reviewer", "approved", &content)
	if err != nil {
		t.Fatalf("CreateReview failed: %v", err)
	}

	// Create a comment
	comment, err := service.CreateReviewComment(review.ReviewID, "testuser", "test-repo", 1, "commenter", "file.go", 10, "Original")
	if err != nil {
		t.Fatalf("CreateReviewComment failed: %v", err)
	}

	// Update the comment
	if err := service.UpdateReviewComment(comment.CommentID, "Updated"); err != nil {
		t.Fatalf("UpdateReviewComment failed: %v", err)
	}

	// Verify update
	updated, err := service.GetReviewComment(comment.CommentID)
	if err != nil {
		t.Fatalf("GetReviewComment failed: %v", err)
	}
	if updated.Content != "Updated" {
		t.Errorf("Expected Content 'Updated', got '%s'", updated.Content)
	}
}

func TestReviewService_DeleteReviewComment(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewReviewService(db)

	// Create a review
	content := "Test review"
	review, err := service.CreateReview("testuser", "test-repo", 1, "reviewer", "approved", &content)
	if err != nil {
		t.Fatalf("CreateReview failed: %v", err)
	}

	// Create a comment
	comment, err := service.CreateReviewComment(review.ReviewID, "testuser", "test-repo", 1, "commenter", "file.go", 10, "Comment")
	if err != nil {
		t.Fatalf("CreateReviewComment failed: %v", err)
	}

	// Delete the comment
	if err := service.DeleteReviewComment(comment.CommentID); err != nil {
		t.Fatalf("DeleteReviewComment failed: %v", err)
	}

	// Verify deletion
	_, err = service.GetReviewComment(comment.CommentID)
	if err == nil {
		t.Fatal("Expected error after deletion, got nil")
	}
}

func TestReviewService_DeleteReview_WithComments(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewReviewService(db)

	// Create a review
	content := "Test review"
	review, err := service.CreateReview("testuser", "test-repo", 1, "reviewer", "approved", &content)
	if err != nil {
		t.Fatalf("CreateReview failed: %v", err)
	}

	// Create comments
	for i := 1; i <= 3; i++ {
		_, err := service.CreateReviewComment(review.ReviewID, "testuser", "test-repo", 1, "commenter", "file.go", i*10, "Comment")
		if err != nil {
			t.Fatalf("CreateReviewComment %d failed: %v", i, err)
		}
	}

	// Delete the review (should cascade delete comments)
	if err := service.DeleteReview(review.ReviewID); err != nil {
		t.Fatalf("DeleteReview failed: %v", err)
	}

	// Verify review is deleted
	_, err = service.GetReview(review.ReviewID)
	if err != ErrReviewNotFound {
		t.Errorf("Expected ErrReviewNotFound after deletion, got %v", err)
	}

	// Verify comments are deleted
	comments, err := service.ListReviewComments(review.ReviewID)
	if err != nil {
		t.Fatalf("ListReviewComments failed: %v", err)
	}
	if len(comments) != 0 {
		t.Errorf("Expected 0 comments after review deletion, got %d", len(comments))
	}
}
