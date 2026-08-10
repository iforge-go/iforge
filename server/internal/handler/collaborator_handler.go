package handler

import (
	"net/http"

	"iforge/iforge/internal/event"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type CollaboratorHandler struct {
	collaboratorService *service.CollaboratorService
	repositoryService   *service.RepositoryService
	eventBus            *event.Bus
}

func NewCollaboratorHandler(collaboratorService *service.CollaboratorService, repositoryService *service.RepositoryService, eventBus *event.Bus) *CollaboratorHandler {
	return &CollaboratorHandler{
		collaboratorService: collaboratorService,
		repositoryService:   repositoryService,
		eventBus:            eventBus,
	}
}

func (h *CollaboratorHandler) AddCollaborator(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	user, repo, ok := getUserAndCheckPermission(c, h.repositoryService, owner, repoName)
	if !ok {
		return nil
	}

	var req struct {
		CollaboratorName string `json:"collaboratorName" validate:"required"`
		Role             string `json:"role" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	collaborator, err := h.collaboratorService.AddCollaborator(owner, repoName, req.CollaboratorName, req.Role)
	if err != nil {
		if err == service.ErrCollaboratorExists {
			respondError(c, http.StatusConflict, err.Error())
			return nil
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	h.eventBus.Publish(event.NewCollaboratorAddedEvent(repo, user, req.CollaboratorName, req.Role))

	LogAudit(c, "collaborator.add", "repository", owner+"/"+repoName,
		map[string]interface{}{"collaborator": req.CollaboratorName, "role": req.Role}, true)

	c.Status(http.StatusCreated).JSON(collaborator)
	return nil
}

func (h *CollaboratorHandler) ListCollaborators(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	_, _, ok := getUserAndCheckPermission(c, h.repositoryService, owner, repoName)
	if !ok {
		return nil
	}

	collaborators, err := h.collaboratorService.ListCollaborators(owner, repoName)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(collaborators)
	return nil
}

func (h *CollaboratorHandler) RemoveCollaborator(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")
	collaboratorName := c.Params("collaborator")

	_, _, ok := getUserAndCheckPermission(c, h.repositoryService, owner, repoName)
	if !ok {
		return nil
	}

	err := h.collaboratorService.RemoveCollaborator(owner, repoName, collaboratorName)
	if err != nil {
		if err == service.ErrCollaboratorNotFound {
			respondError(c, http.StatusNotFound, err.Error())
			return nil
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Collaborator removed"})
	return nil
}

func (h *CollaboratorHandler) UpdateCollaboratorRole(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")
	collaboratorName := c.Params("collaborator")

	_, _, ok := getUserAndCheckPermission(c, h.repositoryService, owner, repoName)
	if !ok {
		return nil
	}

	var req struct {
		Role string `json:"role" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	err := h.collaboratorService.UpdateCollaboratorRole(owner, repoName, collaboratorName, req.Role)
	if err != nil {
		if err == service.ErrCollaboratorNotFound {
			respondError(c, http.StatusNotFound, err.Error())
			return nil
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	LogAudit(c, "collaborator.role_update", "repository", owner+"/"+repoName,
		map[string]interface{}{"collaborator": collaboratorName, "role": req.Role}, true)

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Collaborator role updated"})
	return nil
}
