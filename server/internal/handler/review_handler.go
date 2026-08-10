package handler

import (
	"net/http"
	"strconv"

	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type ReviewHandler struct {
	reviewService *service.ReviewService
	repoService   *service.RepositoryService
}

func NewReviewHandler(reviewService *service.ReviewService, repoService *service.RepositoryService) *ReviewHandler {
	return &ReviewHandler{
		reviewService: reviewService,
		repoService:   repoService,
	}
}

type CreateReviewRequest struct {
	Status  string  `json:"status" validate:"required,oneof=approved changes_requested commented"`
	Content *string `json:"content"`
}

func (h *ReviewHandler) CreateReview(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	owner := c.Params("owner")
	repo := c.Params("repo")
	issueID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid issue ID")
		return nil
	}

	// Check repository access
	repository, err := h.repoService.GetRepository(owner, repo)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	if !h.repoService.HasViewerRole(repository, user) {
		respondError(c, http.StatusForbidden, "Forbidden")
		return nil
	}

	var req CreateReviewRequest
	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	// Developer role required for approved or changes_requested reviews
	if (req.Status == "approved" || req.Status == "changes_requested") && !h.repoService.HasMemberRole(repository, user) {
		respondError(c, http.StatusForbidden, "Forbidden: Developer access required to submit reviews")
		return nil
	}

	review, err := h.reviewService.CreateReview(owner, repo, issueID, user.UserName, req.Status, req.Content)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusCreated).JSON(review)
	return nil
}

func (h *ReviewHandler) ListReviews(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	issueID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid issue ID")
		return nil
	}

	// Check repository access for private repos
	_, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	reviews, err := h.reviewService.ListReviews(owner, repo, issueID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(reviews)
	return nil
}

func (h *ReviewHandler) GetReview(c *fiber.Ctx) error {
	reviewID, err := strconv.Atoi(c.Params("reviewId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid review ID")
		return nil
	}

	review, err := h.reviewService.GetReview(reviewID)
	if err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(review)
	return nil
}

type UpdateReviewRequest struct {
	Status  string  `json:"status" validate:"required,oneof=approved changes_requested commented"`
	Content *string `json:"content"`
}

func (h *ReviewHandler) UpdateReview(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	reviewID, err := strconv.Atoi(c.Params("reviewId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid review ID")
		return nil
	}

	// Check if user owns the review
	review, err := h.reviewService.GetReview(reviewID)
	if err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return nil
	}

	if review.Reviewer != user.UserName {
		respondError(c, http.StatusForbidden, "You can only update your own reviews")
		return nil
	}

	var req UpdateReviewRequest
	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	if err := h.reviewService.UpdateReview(reviewID, req.Status, req.Content); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Review updated"})
	return nil
}

// DeleteReview deletes a review
func (h *ReviewHandler) DeleteReview(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	reviewID, err := strconv.Atoi(c.Params("reviewId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid review ID")
		return nil
	}

	// Check if user owns the review
	review, err := h.reviewService.GetReview(reviewID)
	if err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return nil
	}

	if review.Reviewer != user.UserName {
		respondError(c, http.StatusForbidden, "You can only delete your own reviews")
		return nil
	}

	if err := h.reviewService.DeleteReview(reviewID); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Review deleted"})
	return nil
}

// CreateReviewCommentRequest represents a create review comment request
type CreateReviewCommentRequest struct {
	FilePath string `json:"filePath" validate:"required"`
	Line     int    `json:"line" validate:"required"`
	Content  string `json:"content" validate:"required"`
}

// CreateReviewComment creates a review comment
func (h *ReviewHandler) CreateReviewComment(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	owner := c.Params("owner")
	repo := c.Params("repo")
	issueID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid issue ID")
		return nil
	}

	reviewID, err := strconv.Atoi(c.Params("reviewId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid review ID")
		return nil
	}

	// Check repository access
	repository, err := h.repoService.GetRepository(owner, repo)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	if !h.repoService.HasViewerRole(repository, user) {
		respondError(c, http.StatusForbidden, "Forbidden")
		return nil
	}

	var req CreateReviewCommentRequest
	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	comment, err := h.reviewService.CreateReviewComment(reviewID, owner, repo, issueID, user.UserName, req.FilePath, req.Line, req.Content)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusCreated).JSON(comment)
	return nil
}

// ListReviewComments lists review comments
func (h *ReviewHandler) ListReviewComments(c *fiber.Ctx) error {
	reviewID, err := strconv.Atoi(c.Params("reviewId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid review ID")
		return nil
	}

	// Get review to access repository info
	review, err := h.reviewService.GetReview(reviewID)
	if err != nil {
		respondError(c, http.StatusNotFound, "Review not found")
		return nil
	}

	// Check repository access for private repos
	_, _, ok := getRepoAndCheckAccess(c, h.repoService, review.UserName, review.RepositoryName)
	if !ok {
		return nil
	}

	comments, err := h.reviewService.ListReviewComments(reviewID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(comments)
	return nil
}

// UpdateReviewCommentRequest represents an update review comment request
type UpdateReviewCommentRequest struct {
	Content string `json:"content" validate:"required"`
}

// UpdateReviewComment updates a review comment
func (h *ReviewHandler) UpdateReviewComment(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	commentID, err := strconv.Atoi(c.Params("commentId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid comment ID")
		return nil
	}

	// Check if user owns the comment
	comment, err := h.reviewService.GetReviewComment(commentID)
	if err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return nil
	}

	if comment.Commenter != user.UserName {
		respondError(c, http.StatusForbidden, "You can only update your own comments")
		return nil
	}

	var req UpdateReviewCommentRequest
	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	if err := h.reviewService.UpdateReviewComment(commentID, req.Content); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Comment updated"})
	return nil
}

// DeleteReviewComment deletes a review comment
func (h *ReviewHandler) DeleteReviewComment(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	commentID, err := strconv.Atoi(c.Params("commentId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid comment ID")
		return nil
	}

	// Check if user owns the comment
	comment, err := h.reviewService.GetReviewComment(commentID)
	if err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return nil
	}

	if comment.Commenter != user.UserName {
		respondError(c, http.StatusForbidden, "You can only delete your own comments")
		return nil
	}

	if err := h.reviewService.DeleteReviewComment(commentID); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Comment deleted"})
	return nil
}
