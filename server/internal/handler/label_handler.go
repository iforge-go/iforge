package handler

import (
	"net/http"
	"strconv"

	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

// LabelHandler handles label endpoints
type LabelHandler struct {
	labelService *service.LabelService
	repoService  *service.RepositoryService
	issueService *service.IssueService
}

// NewLabelHandler creates a new LabelHandler
func NewLabelHandler(labelService *service.LabelService, repoService *service.RepositoryService, issueService *service.IssueService) *LabelHandler {
	return &LabelHandler{
		labelService: labelService,
		repoService:  repoService,
		issueService: issueService,
	}
}

// CreateLabelRequest represents a create label request
type CreateLabelRequest struct {
	Name  string `json:"name" validate:"required"`
	Color string `json:"color" validate:"required"`
}

// UpdateLabelRequest represents an update label request
type UpdateLabelRequest struct {
	Name  string `json:"name" validate:"required"`
	Color string `json:"color" validate:"required"`
}

// ListLabels lists labels for a repository
func (h *LabelHandler) ListLabels(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	repository, err := h.repoService.GetRepository(owner, repo)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	user := contextutil.GetUserFromContext(c)
	if repository.IsPrivate {
		if user == nil || !h.repoService.HasViewerRole(repository, user) {
			respondError(c, http.StatusNotFound, "Repository not found")
			return nil
		}
	}

	labels, err := h.labelService.ListLabels(owner, repo)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list labels")
		return nil
	}

	c.Status(http.StatusOK).JSON(labels)
	return nil
}

// CreateLabel creates a label
// Requires Developer role or higher (aligned with GitBucket: writableUsersOnly)
func (h *LabelHandler) CreateLabel(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	_, _, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	var req CreateLabelRequest
	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body")
		return nil
	}

	label, err := h.labelService.CreateLabel(owner, repo, req.Name, req.Color)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to create label")
		return nil
	}

	c.Status(http.StatusCreated).JSON(label)
	return nil
}

// UpdateLabel updates a label
// Requires Developer role or higher (aligned with GitBucket: writableUsersOnly)
func (h *LabelHandler) UpdateLabel(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	labelID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid label ID")
		return nil
	}

	_, _, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	var req UpdateLabelRequest
	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body")
		return nil
	}

	if err := h.labelService.UpdateLabel(owner, repo, labelID, req.Name, req.Color); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to update label")
		return nil
	}

	c.Status(http.StatusNoContent)
	return nil
}

// DeleteLabel deletes a label
// Requires Developer role or higher (aligned with GitBucket: writableUsersOnly)
func (h *LabelHandler) DeleteLabel(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	labelID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid label ID")
		return nil
	}

	_, _, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	if err := h.labelService.DeleteLabel(owner, repo, labelID); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to delete label")
		return nil
	}

	c.Status(http.StatusNoContent)
	return nil
}

// AddLabelToIssue adds a label to an issue
// Requires Developer role or higher (aligned with GitBucket: writableUsersOnly)
func (h *LabelHandler) AddLabelToIssue(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	issueID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid issue ID")
		return nil
	}

	labelID, err := strconv.Atoi(c.Params("labelId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid label ID")
		return nil
	}

	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	_, _, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	if err := h.labelService.AddLabelToIssue(owner, repo, issueID, labelID); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to add label to issue")
		return nil
	}

	// Record activity
	label, err := h.labelService.GetLabel(owner, repo, labelID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to get label")
		return nil
	}

	action := "add_label"
	content := label.LabelName + "|" + label.Color
	h.issueService.AddComment(owner, repo, issueID, user.UserName, action, content)

	c.Status(http.StatusCreated).JSON(label)
	return nil
}

// RemoveLabelFromIssue removes a label from an issue
// Requires Developer role or higher (aligned with GitBucket: writableUsersOnly)
func (h *LabelHandler) RemoveLabelFromIssue(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	issueID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid issue ID")
		return nil
	}

	labelID, err := strconv.Atoi(c.Params("labelId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid label ID")
		return nil
	}

	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	_, _, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	label, err := h.labelService.GetLabel(owner, repo, labelID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to get label")
		return nil
	}

	if err := h.labelService.RemoveLabelFromIssue(owner, repo, issueID, labelID); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to remove label from issue")
		return nil
	}

	// Record activity
	action := "remove_label"
	content := label.LabelName + "|" + label.Color
	h.issueService.AddComment(owner, repo, issueID, user.UserName, action, content)

	c.Status(http.StatusNoContent)
	return nil
}

// GetIssueLabels gets labels for an issue
func (h *LabelHandler) GetIssueLabels(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	issueID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid issue ID")
		return nil
	}

	repository, err := h.repoService.GetRepository(owner, repo)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	user := contextutil.GetUserFromContext(c)
	if repository.IsPrivate {
		if user == nil || !h.repoService.HasViewerRole(repository, user) {
			respondError(c, http.StatusNotFound, "Repository not found")
			return nil
		}
	}

	labels, err := h.labelService.GetIssueLabels(owner, repo, issueID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to get issue labels")
		return nil
	}

	c.Status(http.StatusOK).JSON(labels)
	return nil
}
