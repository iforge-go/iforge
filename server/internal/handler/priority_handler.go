package handler

import (
	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/event"
	"iforge/iforge/internal/model"
	"iforge/iforge/internal/service"
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type PriorityHandler struct {
	priorityService *service.PriorityService
	repoService     *service.RepositoryService
	eventBus        *event.Bus
}

func NewPriorityHandler(priorityService *service.PriorityService, repoService *service.RepositoryService, eventBus *event.Bus) *PriorityHandler {
	return &PriorityHandler{
		priorityService: priorityService,
		repoService:     repoService,
		eventBus:        eventBus,
	}
}

func (h *PriorityHandler) ListPriorities(c *fiber.Ctx) error {
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

	priorities, err := h.priorityService.ListPriorities(owner, repo)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(priorities)
	return nil
}

func (h *PriorityHandler) GetPriority(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	priorityID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid priority ID")
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

	priority, err := h.priorityService.GetPriority(owner, repo, priorityID)
	if err != nil {
		respondError(c, http.StatusNotFound, "Priority not found")
		return nil
	}

	c.Status(http.StatusOK).JSON(priority)
	return nil
}

func (h *PriorityHandler) CreatePriority(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	user, repository, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	var req struct {
		PriorityName string `json:"priorityName" validate:"required"`
		Description  string `json:"description"`
		Color        string `json:"color" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	priority := &model.Priority{
		UserName:       owner,
		RepositoryName: repo,
		PriorityName:   req.PriorityName,
		Description:    &req.Description,
		Color:          req.Color,
	}

	createdPriority, err := h.priorityService.CreatePriority(priority)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	// Publish event: ActivitySubscriber records create_priority activity.
	h.eventBus.Publish(event.NewPriorityCreatedEvent(repository, user, req.PriorityName))

	c.Status(http.StatusCreated).JSON(createdPriority)
	return nil
}

// UpdatePriority updates a priority
func (h *PriorityHandler) UpdatePriority(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	priorityID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid priority ID")
		return nil
	}

	// Check write permission
	_, _, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	var req struct {
		PriorityName string `json:"priorityName"`
		Description  string `json:"description"`
		Color        string `json:"color"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	updates := map[string]interface{}{}
	if req.PriorityName != "" {
		updates["priorityName"] = req.PriorityName
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Color != "" {
		updates["color"] = req.Color
	}

	err = h.priorityService.UpdatePriority(owner, repo, priorityID, updates)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Priority updated successfully"})
	return nil
}

// DeletePriority deletes a priority
func (h *PriorityHandler) DeletePriority(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	priorityID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid priority ID")
		return nil
	}

	// Check admin permission
	_, _, ok := getUserAndCheckPermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	err = h.priorityService.DeletePriority(owner, repo, priorityID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Priority deleted successfully"})
	return nil
}

// ReorderPriorities reorders priorities
func (h *PriorityHandler) ReorderPriorities(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	// Check write permission
	_, _, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	var req struct {
		PriorityIDs []int `json:"priorityIds" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	err := h.priorityService.ReorderPriorities(owner, repo, req.PriorityIDs)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Priorities reordered successfully"})
	return nil
}

// SetDefaultPriority sets the default priority for a repository
func (h *PriorityHandler) SetDefaultPriority(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	// Check write permission
	_, _, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	var req struct {
		PriorityID *int `json:"priorityId"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	err := h.priorityService.SetDefaultPriority(owner, repo, req.PriorityID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Default priority set successfully"})
	return nil
}

// GetDefaultPriority gets the default priority for a repository
func (h *PriorityHandler) GetDefaultPriority(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	// Check repository access
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

	priorityID, err := h.priorityService.GetDefaultPriority(owner, repo)
	if err != nil {
		respondError(c, http.StatusNotFound, "No default priority set")
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"priorityId": priorityID})
	return nil
}
