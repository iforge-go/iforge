package handler

import (
	"net/http"
	"strconv"

	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/model"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

// TaskStatusHandler handles task status endpoints
type TaskStatusHandler struct {
	taskStatusService *service.TaskStatusService
	projectService    *service.ProjectService
}

// NewTaskStatusHandler creates a new TaskStatusHandler
func NewTaskStatusHandler(taskStatusService *service.TaskStatusService, projectService *service.ProjectService) *TaskStatusHandler {
	return &TaskStatusHandler{
		taskStatusService: taskStatusService,
		projectService:    projectService,
	}
}

// CreateTaskStatusRequest represents a create task status request
type CreateTaskStatusRequest struct {
	Name     string `json:"name" validate:"required"`
	Slug     string `json:"slug" validate:"required"`
	Color    string `json:"color"`
	Position int    `json:"position"`
	IsClosed bool   `json:"isClosed"`
	// Category is the three-state classification: todo / in_progress / done
	Category string `json:"category"`
	// WipLimit is the WIP limit for the kanban column; nil/0 means unlimited
	WipLimit *int `json:"wipLimit"`
}

// UpdateTaskStatusRequest represents an update task status request
type UpdateTaskStatusRequest struct {
	Name     *string `json:"name"`
	Slug     *string `json:"slug"`
	Color    *string `json:"color"`
	Position *int    `json:"position"`
	IsClosed *bool   `json:"isClosed"`
	// Category is the three-state classification; nil means no update
	Category *string `json:"category"`
	// WipLimit is the WIP limit; nil means no update, non-nil (including 0) means update (0 means unlimited)
	WipLimit *int `json:"wipLimit"`
}

// CreateTaskStatus creates a new task status
func (h *TaskStatusHandler) CreateTaskStatus(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	projectID, err := decodeID(c.Params("projectSlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid project ID")
	}

	// Check project access
	project, err := h.projectService.GetProject(projectID)
	if err != nil {
		if err == service.ErrProjectNotFound {
			return respondError(c, http.StatusNotFound, "Project not found")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to get project")
	}

	// Requires member role or higher (owner/admin/member)
	if _, err := h.projectService.RequireRole(project.ID, user.UserName, model.ProjectRoleMember); err != nil {
		return respondError(c, http.StatusForbidden, "Only project members can create task statuses")
	}

	var req CreateTaskStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid request body")
	}

	if req.Name == "" || req.Slug == "" {
		return respondError(c, http.StatusBadRequest, "Name and slug are required")
	}

	if req.Color == "" {
		req.Color = "#94a3b8"
	}

	status, err := h.taskStatusService.CreateTaskStatus(
		projectID, req.Name, req.Slug, req.Color, req.Category, req.Position, false, req.IsClosed, req.WipLimit,
	)
	if err != nil {
		if err == service.ErrTaskStatusExists {
			return respondError(c, http.StatusConflict, "Task status already exists")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to create task status")
	}

	status.ProjectSlug = encodeID(status.ProjectID)
	return c.Status(http.StatusCreated).JSON(status)
}

// ListTaskStatuses lists all task statuses for a project
func (h *TaskStatusHandler) ListTaskStatuses(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	projectID, err := decodeID(c.Params("projectSlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid project ID")
	}

	// Check project access
	project, err := h.projectService.GetProject(projectID)
	if err != nil {
		if err == service.ErrProjectNotFound {
			return respondError(c, http.StatusNotFound, "Project not found")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to get project")
	}

	// Private project read gate: viewer role or higher
	if project.IsPrivate {
		if _, err := h.projectService.RequireRole(project.ID, user.UserName, model.ProjectRoleViewer); err != nil {
			return respondError(c, http.StatusForbidden, "Forbidden")
		}
	}

	statuses, err := h.taskStatusService.ListTaskStatuses(projectID)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to list task statuses")
	}

	for i := range statuses {
		statuses[i].ProjectSlug = encodeID(statuses[i].ProjectID)
	}
	return c.Status(http.StatusOK).JSON(statuses)
}

// UpdateTaskStatus updates a task status
func (h *TaskStatusHandler) UpdateTaskStatus(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	statusID, err := strconv.Atoi(c.Params("statusId"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid status ID")
	}

	status, err := h.taskStatusService.GetTaskStatus(statusID)
	if err != nil {
		if err == service.ErrTaskStatusNotFound {
			return respondError(c, http.StatusNotFound, "Task status not found")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to get task status")
	}

	// Check project access
	project, err := h.projectService.GetProject(status.ProjectID)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to get project")
	}

	// Requires member role or higher
	if _, err := h.projectService.RequireRole(project.ID, user.UserName, model.ProjectRoleMember); err != nil {
		return respondError(c, http.StatusForbidden, "Only project members can update task statuses")
	}

	var req UpdateTaskStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid request body")
	}

	if req.Name != nil {
		status.Name = *req.Name
	}
	if req.Slug != nil {
		status.Slug = *req.Slug
	}
	if req.Color != nil {
		status.Color = *req.Color
	}
	if req.Position != nil {
		status.Position = *req.Position
	}
	if req.IsClosed != nil {
		status.IsClosed = *req.IsClosed
	}
	if req.Category != nil {
		status.Category = *req.Category
	}
	// WipLimit: nil means no update; non-nil (including 0) means update (0 or negative means unlimited, stored as nil)
	if req.WipLimit != nil {
		if *req.WipLimit <= 0 {
			status.WipLimit = nil
		} else {
			v := *req.WipLimit
			status.WipLimit = &v
		}
	}

	if err := h.taskStatusService.UpdateTaskStatus(status); err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to update task status")
	}

	status.ProjectSlug = encodeID(status.ProjectID)
	return c.Status(http.StatusOK).JSON(status)
}

// DeleteTaskStatus deletes a task status
func (h *TaskStatusHandler) DeleteTaskStatus(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	statusID, err := strconv.Atoi(c.Params("statusId"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid status ID")
	}

	status, err := h.taskStatusService.GetTaskStatus(statusID)
	if err != nil {
		if err == service.ErrTaskStatusNotFound {
			return respondError(c, http.StatusNotFound, "Task status not found")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to get task status")
	}

	// Check project access
	project, err := h.projectService.GetProject(status.ProjectID)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to get project")
	}

	// Requires member role or higher
	if _, err := h.projectService.RequireRole(project.ID, user.UserName, model.ProjectRoleMember); err != nil {
		return respondError(c, http.StatusForbidden, "Only project members can delete task statuses")
	}

	if err := h.taskStatusService.DeleteTaskStatus(statusID); err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to delete task status")
	}

	return c.SendStatus(http.StatusNoContent)
}

// SetTransitionsRequest represents a set of allowed status transitions
type SetTransitionsRequest struct {
	Transitions []service.StatusTransitionPair `json:"transitions"`
}

// ListTransitions lists all configured status transition rules for the project.
func (h *TaskStatusHandler) ListTransitions(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	projectID, err := decodeID(c.Params("projectSlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid project ID")
	}

	project, err := h.projectService.GetProject(projectID)
	if err != nil {
		if err == service.ErrProjectNotFound {
			return respondError(c, http.StatusNotFound, "Project not found")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to get project")
	}

	// Private project read gate: viewer role or higher
	if project.IsPrivate {
		if _, err := h.projectService.RequireRole(project.ID, user.UserName, model.ProjectRoleViewer); err != nil {
			return respondError(c, http.StatusForbidden, "Forbidden")
		}
	}

	transitions, err := h.taskStatusService.ListTransitions(projectID)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to list transitions")
	}

	for i := range transitions {
		transitions[i].ProjectSlug = encodeID(transitions[i].ProjectID)
	}
	return c.Status(http.StatusOK).JSON(transitions)
}

// SetTransitions replaces all status transition rules for the project.
// Empty transitions clears all rules, allowing all transitions.
func (h *TaskStatusHandler) SetTransitions(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	projectID, err := decodeID(c.Params("projectSlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid project ID")
	}

	project, err := h.projectService.GetProject(projectID)
	if err != nil {
		if err == service.ErrProjectNotFound {
			return respondError(c, http.StatusNotFound, "Project not found")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to get project")
	}

	// Requires member role or higher
	if _, err := h.projectService.RequireRole(project.ID, user.UserName, model.ProjectRoleMember); err != nil {
		return respondError(c, http.StatusForbidden, "Only project members can configure workflow transitions")
	}

	var req SetTransitionsRequest
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid request body")
	}

	if err := h.taskStatusService.SetTransitions(projectID, req.Transitions); err != nil {
		if err == service.ErrTransitionStatusNotFound {
			return respondError(c, http.StatusBadRequest, "Transition references unknown status slug")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to set transitions")
	}

	// Return updated rules for frontend refresh
	transitions, err := h.taskStatusService.ListTransitions(projectID)
	if err != nil {
		return c.SendStatus(http.StatusOK)
	}
	for i := range transitions {
		transitions[i].ProjectSlug = encodeID(transitions[i].ProjectID)
	}
	return c.Status(http.StatusOK).JSON(transitions)
}
