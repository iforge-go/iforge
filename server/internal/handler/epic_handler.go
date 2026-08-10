package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/model"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type EpicHandler struct {
	epicService     *service.EpicService
	projectService  *service.ProjectService
	activityService *service.ScrumActivityService
}

func NewEpicHandler(epicService *service.EpicService, projectService *service.ProjectService, activityService *service.ScrumActivityService) *EpicHandler {
	return &EpicHandler{
		epicService:     epicService,
		projectService:  projectService,
		activityService: activityService,
	}
}

type CreateEpicRequest struct {
	Title       string  `json:"title" validate:"required"`
	Description *string `json:"description"`
	Status      string  `json:"status"`
	Priority    string  `json:"priority"`
	Goal        *string `json:"goal"`
	StartDate   *string `json:"startDate"`
	TargetDate  *string `json:"targetDate"`
	OwnerName   *string `json:"ownerName"`
}

type UpdateEpicRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	Priority    *string `json:"priority"`
	Goal        *string `json:"goal"`
	StartDate   *string `json:"startDate"`
	TargetDate  *string `json:"targetDate"`
	OwnerName   *string `json:"ownerName"`
}

// CreateEpic creates a new epic
func (h *EpicHandler) CreateEpic(c *fiber.Ctx) error {
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

	if _, err := h.projectService.RequireRole(project.ID, user.UserName, model.ProjectRoleMember); err != nil {
		return respondError(c, http.StatusForbidden, "Only project members can create epics")
	}

	var req CreateEpicRequest
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid request body")
	}

	if req.Title == "" {
		return respondError(c, http.StatusBadRequest, "Title is required")
	}

	status := req.Status
	if status == "" {
		status = "open"
	}
	priority := req.Priority
	if priority == "" {
		priority = "medium"
	}

	var startDate *time.Time
	if req.StartDate != nil && *req.StartDate != "" {
		if parsed, err := time.Parse("2006-01-02", *req.StartDate); err == nil {
			startDate = &parsed
		}
	}

	var targetDate *time.Time
	if req.TargetDate != nil && *req.TargetDate != "" {
		if parsed, err := time.Parse("2006-01-02", *req.TargetDate); err == nil {
			targetDate = &parsed
		}
	}

	epic, err := h.epicService.CreateEpic(
		projectID, req.Title, status, priority,
		req.Description, req.Goal, startDate, targetDate,
		req.OwnerName, &user.UserName,
	)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to create epic")
	}

	epic.Slug = encodeID(epic.ID)
	epic.ProjectSlug = encodeID(epic.ProjectID)

	// Record creation activity
	if h.activityService != nil {
		_ = h.activityService.LogActivity(user.UserName, epic.ProjectID, "epic", epic.ID, "created", "", "", epic.Title)
	}
	return c.Status(http.StatusCreated).JSON(epic)
}

// GetEpic retrieves an epic by ID with aggregated progress
func (h *EpicHandler) GetEpic(c *fiber.Ctx) error {
	id, err := decodeID(c.Params("epicSlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid epic ID")
	}

	epic, err := h.epicService.GetEpic(id)
	if err != nil {
		if err == service.ErrEpicNotFound {
			return respondError(c, http.StatusNotFound, "Epic not found")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to get epic")
	}

	epic.Slug = encodeID(epic.ID)
	epic.ProjectSlug = encodeID(epic.ProjectID)
	return c.Status(http.StatusOK).JSON(epic)
}

// UpdateEpic updates an epic
func (h *EpicHandler) UpdateEpic(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	id, err := decodeID(c.Params("epicSlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid epic ID")
	}

	epic, err := h.epicService.GetEpic(id)
	if err != nil {
		if err == service.ErrEpicNotFound {
			return respondError(c, http.StatusNotFound, "Epic not found")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to get epic")
	}

	// Check project access
	project, err := h.projectService.GetProject(epic.ProjectID)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to get project")
	}

	if _, err := h.projectService.RequireRole(project.ID, user.UserName, model.ProjectRoleMember); err != nil {
		return respondError(c, http.StatusForbidden, "Only project members can update epics")
	}

	var req UpdateEpicRequest
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid request body")
	}

	// Snapshot old values for activity logging
	oldTitle := epic.Title
	oldDescription := epic.Description
	oldStatus := epic.Status
	oldPriority := epic.Priority
	oldGoal := epic.Goal
	oldStartDate := epic.StartDate
	oldTargetDate := epic.TargetDate
	oldOwnerName := epic.OwnerName

	if req.Title != nil {
		epic.Title = *req.Title
	}
	if req.Description != nil {
		epic.Description = req.Description
	}
	if req.Status != nil {
		epic.Status = *req.Status
		epic.StatusManual = true
		if *req.Status == "closed" {
			now := time.Now()
			epic.ClosedAt = &now
		} else {
			epic.ClosedAt = nil
		}
	}
	if req.Priority != nil {
		epic.Priority = *req.Priority
	}
	if req.Goal != nil {
		epic.Goal = req.Goal
	}
	if req.StartDate != nil {
		if *req.StartDate == "" {
			epic.StartDate = nil
		} else if parsed, err := time.Parse("2006-01-02", *req.StartDate); err == nil {
			epic.StartDate = &parsed
		}
	}
	if req.TargetDate != nil {
		if *req.TargetDate == "" {
			epic.TargetDate = nil
		} else if parsed, err := time.Parse("2006-01-02", *req.TargetDate); err == nil {
			epic.TargetDate = &parsed
		}
	}
	if req.OwnerName != nil {
		if *req.OwnerName == "" {
			epic.OwnerName = nil
		} else {
			epic.OwnerName = req.OwnerName
			if err := h.projectService.EnsureMember(project.ID, *req.OwnerName, model.ProjectRoleMember); err != nil {
				fmt.Printf("Warning: failed to ensure project membership for epic owner %s: %v\n", *req.OwnerName, err)
			}
		}
	}

	if err := h.epicService.UpdateEpic(epic); err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to update epic")
	}

	// Record field changes for activity logging
	if h.activityService != nil {
		if oldTitle != epic.Title {
			_ = h.activityService.LogActivity(user.UserName, epic.ProjectID, "epic", epic.ID, "updated", "title", oldTitle, epic.Title)
		}
		if ptrStringValue(oldDescription) != ptrStringValue(epic.Description) {
			_ = h.activityService.LogActivity(user.UserName, epic.ProjectID, "epic", epic.ID, "updated", "description", ptrStringValue(oldDescription), ptrStringValue(epic.Description))
		}
		if oldStatus != epic.Status {
			_ = h.activityService.LogActivity(user.UserName, epic.ProjectID, "epic", epic.ID, "status_changed", "status", oldStatus, epic.Status)
		}
		if oldPriority != epic.Priority {
			_ = h.activityService.LogActivity(user.UserName, epic.ProjectID, "epic", epic.ID, "updated", "priority", oldPriority, epic.Priority)
		}
		if ptrStringValue(oldGoal) != ptrStringValue(epic.Goal) {
			_ = h.activityService.LogActivity(user.UserName, epic.ProjectID, "epic", epic.ID, "updated", "goal", ptrStringValue(oldGoal), ptrStringValue(epic.Goal))
		}
		if ptrTimeValue(oldStartDate) != ptrTimeValue(epic.StartDate) {
			_ = h.activityService.LogActivity(user.UserName, epic.ProjectID, "epic", epic.ID, "updated", "startDate", ptrTimeValue(oldStartDate), ptrTimeValue(epic.StartDate))
		}
		if ptrTimeValue(oldTargetDate) != ptrTimeValue(epic.TargetDate) {
			_ = h.activityService.LogActivity(user.UserName, epic.ProjectID, "epic", epic.ID, "updated", "targetDate", ptrTimeValue(oldTargetDate), ptrTimeValue(epic.TargetDate))
		}
		if ptrStringValue(oldOwnerName) != ptrStringValue(epic.OwnerName) {
			_ = h.activityService.LogActivity(user.UserName, epic.ProjectID, "epic", epic.ID, "updated", "owner", ptrStringValue(oldOwnerName), ptrStringValue(epic.OwnerName))
		}
	}

	// Re-fetch to get latest aggregated progress
	updated, _ := h.epicService.GetEpic(epic.ID)
	if updated != nil {
		epic = updated
	}
	epic.Slug = encodeID(epic.ID)
	epic.ProjectSlug = encodeID(epic.ProjectID)
	return c.Status(http.StatusOK).JSON(epic)
}

// DeleteEpic deletes an epic with soft-delete of associated stories
func (h *EpicHandler) DeleteEpic(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	id, err := decodeID(c.Params("epicSlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid epic ID")
	}

	epic, err := h.epicService.GetEpic(id)
	if err != nil {
		if err == service.ErrEpicNotFound {
			return respondError(c, http.StatusNotFound, "Epic not found")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to get epic")
	}

	// Check project access
	project, err := h.projectService.GetProject(epic.ProjectID)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to get project")
	}

	if _, err := h.projectService.RequireRole(project.ID, user.UserName, model.ProjectRoleMember); err != nil {
		return respondError(c, http.StatusForbidden, "Only project members can delete epics")
	}

	if err := h.epicService.DeleteEpic(id); err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to delete epic")
	}

	// Record deletion activity
	if h.activityService != nil {
		_ = h.activityService.LogActivity(user.UserName, epic.ProjectID, "epic", epic.ID, "deleted", "", "", epic.Title)
	}

	return c.SendStatus(http.StatusNoContent)
}

// ListEpics lists epics for a project
func (h *EpicHandler) ListEpics(c *fiber.Ctx) error {
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

	if project.IsPrivate {
		if _, err := h.projectService.RequireRole(project.ID, user.UserName, model.ProjectRoleViewer); err != nil {
			return respondError(c, http.StatusForbidden, "Forbidden")
		}
	}

	status := c.Query("status")
	var statusPtr *string
	if status != "" {
		statusPtr = &status
	}

	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	epics, err := h.epicService.ListEpics(projectID, statusPtr, limit, offset)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to list epics")
	}

	if epics == nil {
		epics = []*model.Epic{}
	}

	for i := range epics {
		epics[i].Slug = encodeID(epics[i].ID)
		epics[i].ProjectSlug = encodeID(epics[i].ProjectID)
	}

	return c.Status(http.StatusOK).JSON(epics)
}

// ListEpicStories lists all stories belonging to an epic
func (h *EpicHandler) ListEpicStories(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	id, err := decodeID(c.Params("epicSlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid epic ID")
	}

	epic, err := h.epicService.GetEpic(id)
	if err != nil {
		if err == service.ErrEpicNotFound {
			return respondError(c, http.StatusNotFound, "Epic not found")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to get epic")
	}

	project, err := h.projectService.GetProject(epic.ProjectID)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to get project")
	}
	if project.IsPrivate {
		if _, err := h.projectService.RequireRole(project.ID, user.UserName, model.ProjectRoleViewer); err != nil {
			return respondError(c, http.StatusForbidden, "Forbidden")
		}
	}

	stories, err := h.epicService.ListEpicStories(epic.ProjectID, id)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to list epic stories")
	}

	if stories == nil {
		stories = []*model.UserStory{}
	}

	for i := range stories {
		stories[i].Slug = encodeID(stories[i].ID)
		stories[i].ProjectSlug = encodeID(stories[i].ProjectID)
		if stories[i].EpicID != nil {
			slug := encodeID(*stories[i].EpicID)
			stories[i].EpicSlug = &slug
			titleCopy := epic.Title
			stories[i].EpicTitle = &titleCopy
		}
	}

	return c.Status(http.StatusOK).JSON(stories)
}

// GetEpicProgress returns aggregated progress for an epic
func (h *EpicHandler) GetEpicProgress(c *fiber.Ctx) error {
	id, err := decodeID(c.Params("epicSlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid epic ID")
	}

	epic, err := h.epicService.GetEpic(id)
	if err != nil {
		if err == service.ErrEpicNotFound {
			return respondError(c, http.StatusNotFound, "Epic not found")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to get epic")
	}

	return c.Status(http.StatusOK).JSON(map[string]interface{}{
		"totalStories":     epic.TotalStories,
		"doneStories":      epic.DoneStories,
		"totalStoryPoints": epic.TotalStoryPoints,
		"doneStoryPoints":  epic.DoneStoryPoints,
		"progressPercent":  epic.ProgressPercent,
		"sprintCount":      epic.SprintCount,
	})
}

// GetEpicSprints returns the sprints that this epic spans (via task → story → epic chain)
func (h *EpicHandler) GetEpicSprints(c *fiber.Ctx) error {
	id, err := decodeID(c.Params("epicSlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid epic ID")
	}

	epic, err := h.epicService.GetEpic(id)
	if err != nil {
		if err == service.ErrEpicNotFound {
			return respondError(c, http.StatusNotFound, "Epic not found")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to get epic")
	}

	infos, err := h.epicService.GetEpicSprints(epic.ProjectID, id)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to get epic sprints")
	}

	for i := range infos {
		infos[i].SprintSlug = encodeID(infos[i].SprintID)
	}

	if infos == nil {
		infos = []service.EpicSprintInfo{}
	}

	return c.Status(http.StatusOK).JSON(infos)
}

// AssignStoryToEpicRequest represents an assign story to epic request
type AssignStoryToEpicRequest struct {
	StorySlug string `json:"storySlug" validate:"required"`
}

// AssignStoryToEpic associates a story with an epic
func (h *EpicHandler) AssignStoryToEpic(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	epicID, err := decodeID(c.Params("epicSlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid epic ID")
	}

	epic, err := h.epicService.GetEpic(epicID)
	if err != nil {
		if err == service.ErrEpicNotFound {
			return respondError(c, http.StatusNotFound, "Epic not found")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to get epic")
	}

	project, err := h.projectService.GetProject(epic.ProjectID)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to get project")
	}
	if _, err := h.projectService.RequireRole(project.ID, user.UserName, model.ProjectRoleMember); err != nil {
		return respondError(c, http.StatusForbidden, "Only project members can assign stories to epics")
	}

	var req AssignStoryToEpicRequest
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid request body")
	}
	if req.StorySlug == "" {
		return respondError(c, http.StatusBadRequest, "storySlug is required")
	}

	storyID, err := decodeID(req.StorySlug)
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid story slug")
	}

	if err := h.epicService.AssignStoryToEpic(epic.ProjectID, epicID, storyID); err != nil {
		if err == service.ErrUserStoryNotFound {
			return respondError(c, http.StatusNotFound, "Story not found")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to assign story to epic")
	}

	if h.activityService != nil {
		_ = h.activityService.LogActivity(user.UserName, epic.ProjectID, "epic", epic.ID, "story_added", "", "", req.StorySlug)
	}

	return c.SendStatus(http.StatusNoContent)
}

// RemoveStoryFromEpic removes a story from an epic
func (h *EpicHandler) RemoveStoryFromEpic(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	epicID, err := decodeID(c.Params("epicSlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid epic ID")
	}

	storyID, err := decodeID(c.Params("storySlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid story slug")
	}

	epic, err := h.epicService.GetEpic(epicID)
	if err != nil {
		if err == service.ErrEpicNotFound {
			return respondError(c, http.StatusNotFound, "Epic not found")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to get epic")
	}

	project, err := h.projectService.GetProject(epic.ProjectID)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to get project")
	}
	if _, err := h.projectService.RequireRole(project.ID, user.UserName, model.ProjectRoleMember); err != nil {
		return respondError(c, http.StatusForbidden, "Only project members can remove stories from epics")
	}

	if err := h.epicService.RemoveStoryFromEpic(epic.ProjectID, storyID); err != nil {
		if err == service.ErrUserStoryNotFound {
			return respondError(c, http.StatusNotFound, "Story not found")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to remove story from epic")
	}

	if h.activityService != nil {
		_ = h.activityService.LogActivity(user.UserName, epic.ProjectID, "epic", epic.ID, "story_removed", "", "", c.Params("storySlug"))
	}

	return c.SendStatus(http.StatusNoContent)
}
