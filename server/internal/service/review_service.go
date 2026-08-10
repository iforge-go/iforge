package service

import (
	"errors"
	"time"

	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

var (
	ErrReviewNotFound = errors.New("review not found")
)

// ReviewService handles review operations
type ReviewService struct {
	db *gorm.DB
}

// NewReviewService creates a new ReviewService
func NewReviewService(db *gorm.DB) *ReviewService {
	return &ReviewService{db: db}
}

// GetReview gets a review by ID
func (s *ReviewService) GetReview(reviewID int) (*model.Review, error) {
	var review model.Review
	err := s.db.Where("review_id = ?", reviewID).First(&review).Error

	if err == gorm.ErrRecordNotFound {
		return nil, ErrReviewNotFound
	}
	if err != nil {
		return nil, err
	}

	return &review, nil
}

// ListReviews lists reviews for a merge request
func (s *ReviewService) ListReviews(userName, repoName string, issueID int) ([]*model.Review, error) {
	var reviews []*model.Review
	err := s.db.
		Where("user_name = ? AND repository_name = ? AND issue_id = ?", userName, repoName, issueID).
		Order("registered_date ASC").
		Find(&reviews).Error

	return reviews, err
}

// CreateReview creates a new review
func (s *ReviewService) CreateReview(userName, repoName string, issueID int, reviewer, status string, content *string) (*model.Review, error) {
	now := time.Now()
	review := &model.Review{
		UserName:       userName,
		RepositoryName: repoName,
		IssueID:        issueID,
		Reviewer:       reviewer,
		Status:         status,
		Content:        content,
		RegisteredDate: now,
		UpdatedDate:    now,
	}

	if err := s.db.Create(review).Error; err != nil {
		return nil, err
	}

	return review, nil
}

// UpdateReview updates a review
func (s *ReviewService) UpdateReview(reviewID int, status string, content *string) error {
	return s.db.
		Model(&model.Review{}).
		Where("review_id = ?", reviewID).
		Updates(map[string]interface{}{
			"status":       status,
			"content":      content,
			"updated_date": time.Now(),
		}).Error
}

// DeleteReview deletes a review
func (s *ReviewService) DeleteReview(reviewID int) error {
	// First delete all review comments
	if err := s.db.Where("review_id = ?", reviewID).Delete(&model.ReviewComment{}).Error; err != nil {
		return err
	}

	// Then delete the review
	return s.db.Where("review_id = ?", reviewID).Delete(&model.Review{}).Error
}

// GetReviewComment gets a review comment by ID
func (s *ReviewService) GetReviewComment(commentID int) (*model.ReviewComment, error) {
	var comment model.ReviewComment
	err := s.db.Where("comment_id = ?", commentID).First(&comment).Error

	if err == gorm.ErrRecordNotFound {
		return nil, ErrReviewNotFound
	}
	if err != nil {
		return nil, err
	}

	return &comment, nil
}

// ListReviewComments lists review comments for a review
func (s *ReviewService) ListReviewComments(reviewID int) ([]*model.ReviewComment, error) {
	var comments []*model.ReviewComment
	err := s.db.
		Where("review_id = ?", reviewID).
		Order("registered_date ASC").
		Find(&comments).Error

	return comments, err
}

// CreateReviewComment creates a new review comment
func (s *ReviewService) CreateReviewComment(reviewID int, userName, repoName string, issueID int, commenter, filePath string, line int, content string) (*model.ReviewComment, error) {
	now := time.Now()
	comment := &model.ReviewComment{
		ReviewID:       reviewID,
		UserName:       userName,
		RepositoryName: repoName,
		IssueID:        issueID,
		Commenter:      commenter,
		FilePath:       filePath,
		Line:           line,
		Content:        content,
		RegisteredDate: now,
		UpdatedDate:    now,
	}

	if err := s.db.Create(comment).Error; err != nil {
		return nil, err
	}

	return comment, nil
}

// UpdateReviewComment updates a review comment
func (s *ReviewService) UpdateReviewComment(commentID int, content string) error {
	return s.db.
		Model(&model.ReviewComment{}).
		Where("comment_id = ?", commentID).
		Updates(map[string]interface{}{
			"content":      content,
			"updated_date": time.Now(),
		}).Error
}

// DeleteReviewComment deletes a review comment
func (s *ReviewService) DeleteReviewComment(commentID int) error {
	return s.db.Where("comment_id = ?", commentID).Delete(&model.ReviewComment{}).Error
}
