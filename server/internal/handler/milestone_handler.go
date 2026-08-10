package handler

import (
	"net/http"
	"strconv"
	"time"

	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type MilestoneHandler struct {
	milestoneService *service.MilestoneService
	repoService      *service.RepositoryService
}

func NewMilestoneHandler(milestoneService *service.MilestoneService, repoService *service.RepositoryService) *MilestoneHandler {
	return &MilestoneHandler{
		milestoneService: milestoneService,
		repoService:      repoService,
	}
}

type CreateMilestoneRequest struct {
	Title       string  `json:"title" validate:"required"`
	Description string  `json:"description"`
	DueDate     *string `json:"dueDate"`
}

type UpdateMilestoneRequest struct {
	Title       string  `json:"title" validate:"required"`
	Description string  `json:"description"`
	DueDate     *string `json:"dueDate"`
}

func (h *MilestoneHandler) ListMilestones(c *fiber.Ctx) error {
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

	milestones, err := h.milestoneService.ListMilestones(owner, repo)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list milestones")
		return nil
	}

	c.Status(http.StatusOK).JSON(milestones)
	return nil
}

func (h *MilestoneHandler) CreateMilestone(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	_, _, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	var req CreateMilestoneRequest
	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body")
		return nil
	}

	var dueDate *time.Time
	if req.DueDate != nil {
		t, err := time.Parse(time.RFC3339, *req.DueDate)
		if err != nil {
			respondError(c, http.StatusBadRequest, "Invalid due date format")
			return nil
		}
		dueDate = &t
	}

	milestone, err := h.milestoneService.CreateMilestone(owner, repo, req.Title, req.Description, dueDate)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to create milestone")
		return nil
	}

	c.Status(http.StatusCreated).JSON(milestone)
	return nil
}

func (h *MilestoneHandler) GetMilestone(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid milestone ID")
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

	milestone, err := h.milestoneService.GetMilestone(owner, repo, id)
	if err != nil {
		if err == service.ErrMilestoneNotFound {
			respondError(c, http.StatusNotFound, "Milestone not found")
			return nil
		}
		respondError(c, http.StatusInternalServerError, "Failed to get milestone")
		return nil
	}

	c.Status(http.StatusOK).JSON(milestone)
	return nil
}

func (h *MilestoneHandler) UpdateMilestone(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid milestone ID")
		return nil
	}

	_, _, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	var req UpdateMilestoneRequest
	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body")
		return nil
	}

	var dueDate *time.Time
	if req.DueDate != nil {
		t, err := time.Parse(time.RFC3339, *req.DueDate)
		if err != nil {
			respondError(c, http.StatusBadRequest, "Invalid due date format")
			return nil
		}
		dueDate = &t
	}

	if err := h.milestoneService.UpdateMilestone(owner, repo, id, req.Title, req.Description, dueDate); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to update milestone")
		return nil
	}

	c.Status(http.StatusNoContent)
	return nil
}

func (h *MilestoneHandler) DeleteMilestone(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid milestone ID")
		return nil
	}

	_, _, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	if err := h.milestoneService.DeleteMilestone(owner, repo, id); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to delete milestone")
		return nil
	}

	c.Status(http.StatusNoContent)
	return nil
}

func (h *MilestoneHandler) CloseMilestone(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid milestone ID")
		return nil
	}

	_, _, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	if err := h.milestoneService.CloseMilestone(owner, repo, id); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to close milestone")
		return nil
	}

	c.Status(http.StatusNoContent)
	return nil
}

func (h *MilestoneHandler) ReopenMilestone(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid milestone ID")
		return nil
	}

	_, _, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	if err := h.milestoneService.ReopenMilestone(owner, repo, id); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to reopen milestone")
		return nil
	}

	c.Status(http.StatusNoContent)
	return nil
}
