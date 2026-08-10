package handler

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/model"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type SprintHandler struct {
	sprintService   *service.SprintService
	projectService  *service.ProjectService
	activityService *service.ScrumActivityService
	aiService       *service.AIService
}

func NewSprintHandler(sprintService *service.SprintService, projectService *service.ProjectService, activityService *service.ScrumActivityService, aiService *service.AIService) *SprintHandler {
	return &SprintHandler{
		sprintService:   sprintService,
		projectService:  projectService,
		activityService: activityService,
		aiService:       aiService,
	}
}

type CreateSprintRequest struct {
	Title       string  `json:"title" validate:"required"`
	Description *string `json:"description"`
	Status      string  `json:"status" validate:"required"` // open, active, closed
	Goal        *string `json:"goal"`
	StartDate   *string `json:"startDate"`
	EndDate     *string `json:"endDate"`
}

type UpdateSprintRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	Goal        *string `json:"goal"`
	StartDate   *string `json:"startDate"`
	EndDate     *string `json:"endDate"`
}

// CreateSprint creates a new sprint
func (h *SprintHandler) CreateSprint(c *fiber.Ctx) error {
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

	// Write access: member role or above
	if _, err := h.projectService.RequireRole(project.ID, user.UserName, model.ProjectRoleMember); err != nil {
		return respondError(c, http.StatusForbidden, "Only project members can create sprints")
	}

	var req CreateSprintRequest
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid request body")
	}

	if req.Title == "" {
		return respondError(c, http.StatusBadRequest, "Title is required")
	}

	sprint, err := h.sprintService.CreateSprint(projectID, req.Title, req.Status, req.Description, req.Goal, nil, nil, user.UserName)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to create sprint")
	}

	sprint.Slug = encodeID(sprint.ID)
	sprint.ProjectSlug = encodeID(sprint.ProjectID)

	// Record creation activity
	if h.activityService != nil {
		_ = h.activityService.LogActivity(user.UserName, sprint.ProjectID, "sprint", sprint.ID, "created", "", "", sprint.Title)
	}
	return c.Status(http.StatusCreated).JSON(sprint)
}

// GetSprint retrieves a sprint by ID
func (h *SprintHandler) GetSprint(c *fiber.Ctx) error {
	id, err := decodeID(c.Params("sprintSlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid sprint ID")
	}

	sprint, err := h.sprintService.GetSprint(id)
	if err != nil {
		if err == service.ErrSprintNotFound {
			return respondError(c, http.StatusNotFound, "Sprint not found")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to get sprint")
	}

	sprint.Slug = encodeID(sprint.ID)
	sprint.ProjectSlug = encodeID(sprint.ProjectID)
	return c.Status(http.StatusOK).JSON(sprint)
}

// UpdateSprint updates a sprint
func (h *SprintHandler) UpdateSprint(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	id, err := decodeID(c.Params("sprintSlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid sprint ID")
	}

	sprint, err := h.sprintService.GetSprint(id)
	if err != nil {
		if err == service.ErrSprintNotFound {
			return respondError(c, http.StatusNotFound, "Sprint not found")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to get sprint")
	}

	// Check project access
	project, err := h.projectService.GetProject(sprint.ProjectID)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to get project")
	}

	// Write access: member role or above
	if _, err := h.projectService.RequireRole(project.ID, user.UserName, model.ProjectRoleMember); err != nil {
		return respondError(c, http.StatusForbidden, "Only project members can update sprints")
	}

	var req UpdateSprintRequest
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid request body")
	}

	// Snapshot old values for activity logging
	oldTitle := sprint.Title
	oldDescription := sprint.Description
	oldStatus := sprint.Status
	oldGoal := sprint.Goal
	oldStartDate := sprint.StartDate
	oldEndDate := sprint.EndDate

	if req.Title != nil {
		sprint.Title = *req.Title
	}
	if req.Description != nil {
		sprint.Description = req.Description
	}
	if req.Status != nil {
		sprint.Status = *req.Status
	}
	if req.Goal != nil {
		sprint.Goal = req.Goal
	}
	if req.StartDate != nil {
		if *req.StartDate == "" {
			sprint.StartDate = nil
		} else if parsed, err := time.Parse("2006-01-02", *req.StartDate); err == nil {
			sprint.StartDate = &parsed
		}
	}
	if req.EndDate != nil {
		if *req.EndDate == "" {
			sprint.EndDate = nil
		} else if parsed, err := time.Parse("2006-01-02", *req.EndDate); err == nil {
			sprint.EndDate = &parsed
		}
	}

	if err := h.sprintService.UpdateSprint(sprint); err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to update sprint")
	}

	// Record field changes for activity logging
	if h.activityService != nil {
		if oldTitle != sprint.Title {
			_ = h.activityService.LogActivity(user.UserName, sprint.ProjectID, "sprint", sprint.ID, "updated", "title", oldTitle, sprint.Title)
		}
		if ptrStringValue(oldDescription) != ptrStringValue(sprint.Description) {
			_ = h.activityService.LogActivity(user.UserName, sprint.ProjectID, "sprint", sprint.ID, "updated", "description", ptrStringValue(oldDescription), ptrStringValue(sprint.Description))
		}
		if oldStatus != sprint.Status {
			_ = h.activityService.LogActivity(user.UserName, sprint.ProjectID, "sprint", sprint.ID, "status_changed", "status", oldStatus, sprint.Status)
		}
		if ptrStringValue(oldGoal) != ptrStringValue(sprint.Goal) {
			_ = h.activityService.LogActivity(user.UserName, sprint.ProjectID, "sprint", sprint.ID, "updated", "goal", ptrStringValue(oldGoal), ptrStringValue(sprint.Goal))
		}
		if ptrTimeValue(oldStartDate) != ptrTimeValue(sprint.StartDate) {
			_ = h.activityService.LogActivity(user.UserName, sprint.ProjectID, "sprint", sprint.ID, "updated", "startDate", ptrTimeValue(oldStartDate), ptrTimeValue(sprint.StartDate))
		}
		if ptrTimeValue(oldEndDate) != ptrTimeValue(sprint.EndDate) {
			_ = h.activityService.LogActivity(user.UserName, sprint.ProjectID, "sprint", sprint.ID, "updated", "endDate", ptrTimeValue(oldEndDate), ptrTimeValue(sprint.EndDate))
		}
	}

	sprint.Slug = encodeID(sprint.ID)
	sprint.ProjectSlug = encodeID(sprint.ProjectID)
	return c.Status(http.StatusOK).JSON(sprint)
}

// DeleteSprint deletes a sprint
func (h *SprintHandler) DeleteSprint(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	id, err := decodeID(c.Params("sprintSlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid sprint ID")
	}

	sprint, err := h.sprintService.GetSprint(id)
	if err != nil {
		if err == service.ErrSprintNotFound {
			return respondError(c, http.StatusNotFound, "Sprint not found")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to get sprint")
	}

	// Check project access
	project, err := h.projectService.GetProject(sprint.ProjectID)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to get project")
	}

	// Write access: member role or above
	if _, err := h.projectService.RequireRole(project.ID, user.UserName, model.ProjectRoleMember); err != nil {
		return respondError(c, http.StatusForbidden, "Only project members can delete sprints")
	}

	if err := h.sprintService.DeleteSprint(id); err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to delete sprint")
	}

	// Record deletion activity
	if h.activityService != nil {
		_ = h.activityService.LogActivity(user.UserName, sprint.ProjectID, "sprint", sprint.ID, "deleted", "", "", sprint.Title)
	}

	return c.SendStatus(http.StatusNoContent)
}

// ListSprints lists sprints for a project
func (h *SprintHandler) ListSprints(c *fiber.Ctx) error {
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

	// Read access: viewer role or above for private projects
	if project.IsPrivate {
		if _, err := h.projectService.RequireRole(project.ID, user.UserName, model.ProjectRoleViewer); err != nil {
			return respondError(c, http.StatusForbidden, "Forbidden")
		}
	}

	limit, _ := strconv.Atoi(c.Query("limit", "30"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	sprints, err := h.sprintService.ListSprints(projectID, limit, offset)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to list sprints")
	}

	for i := range sprints {
		sprints[i].Slug = encodeID(sprints[i].ID)
		sprints[i].ProjectSlug = encodeID(sprints[i].ProjectID)
	}

	return c.Status(http.StatusOK).JSON(sprints)
}

// GetSprintTasks retrieves all tasks in a sprint
func (h *SprintHandler) GetSprintTasks(c *fiber.Ctx) error {
	id, err := decodeID(c.Params("sprintSlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid sprint ID")
	}

	tasks, err := h.sprintService.GetSprintTasks(id)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to get sprint tasks")
	}

	// Ensure empty results serialize as [] not null
	if tasks == nil {
		tasks = []*model.Task{}
	}

	for i := range tasks {
		tasks[i].Slug = encodeID(tasks[i].TaskID)
		tasks[i].ProjectSlug = encodeID(tasks[i].ProjectID)
		if tasks[i].SprintID != nil {
			slug := encodeID(*tasks[i].SprintID)
			tasks[i].SprintSlug = &slug
		}
		if tasks[i].UserStoryID != nil {
			slug := encodeID(*tasks[i].UserStoryID)
			tasks[i].UserStorySlug = &slug
		}
	}

	return c.Status(http.StatusOK).JSON(tasks)
}

// GetSprintStories retrieves user stories that have tasks in this sprint.
// Stories are not directly assigned to sprints; they appear here if any of their tasks are.
func (h *SprintHandler) GetSprintStories(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	id, err := decodeID(c.Params("sprintSlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid sprint ID")
	}

	sprint, err := h.sprintService.GetSprint(id)
	if err != nil {
		if err == service.ErrSprintNotFound {
			return respondError(c, http.StatusNotFound, "Sprint not found")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to get sprint")
	}

	stories, err := h.sprintService.GetSprintStories(sprint.ProjectID, id)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to get sprint stories")
	}

	// Ensure empty results serialize as [] not null
	if stories == nil {
		stories = []*model.UserStory{}
	}

	for i := range stories {
		stories[i].Slug = encodeID(stories[i].ID)
		stories[i].ProjectSlug = encodeID(stories[i].ProjectID)
	}

	return c.Status(http.StatusOK).JSON(stories)
}

// AIOptimizeSprint handles SSE streaming for AI sprint draft optimization.
// Accepts the current draft (title/description/goal) and returns optimized content.
func (h *SprintHandler) AIOptimizeSprint(c *fiber.Ctx) error {
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

	// Write access gate (member+)
	if _, ok := checkProjectWriteAccess(c, h.projectService, project.ID); !ok {
		return respondError(c, http.StatusForbidden, "Only project members can use AI optimize")
	}

	if h.aiService == nil {
		return respondError(c, http.StatusServiceUnavailable, "AI service not configured")
	}

	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Goal        string `json:"goal"`
	}
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid request body")
	}
	if req.Title == "" {
		return respondError(c, http.StatusBadRequest, "Title is required")
	}

	lang := c.Query("lang", "en")

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		writeEvent := func(event string, data interface{}) {
			jsonBytes, _ := json.Marshal(data)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, string(jsonBytes))
			w.Flush()
		}

		result, err := h.aiService.OptimizeSprintStream(
			req.Title, req.Description, req.Goal, lang,
			func(prompt string) {
				writeEvent("prompt", map[string]string{"content": prompt})
			},
			func(thinking string) {
				writeEvent("thinking", map[string]string{"content": thinking})
			},
			func(delta string) {
				writeEvent("delta", map[string]string{"content": delta})
			},
		)

		if err != nil {
			reason := ""
			if err == service.ErrAIDisabled || err == service.ErrAIMisconfigured {
				reason = "ai_not_configured"
			}
			writeEvent("error", map[string]string{
				"error":  err.Error(),
				"reason": reason,
			})
			return
		}

		writeEvent("done", result)
	})

	return nil
}

// GetVelocity returns velocity data (committed vs completed points) for the last N closed sprints.
func (h *SprintHandler) GetVelocity(c *fiber.Ctx) error {
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

	// Read access: viewer role or above for private projects
	if project.IsPrivate {
		if _, err := h.projectService.RequireRole(project.ID, user.UserName, model.ProjectRoleViewer); err != nil {
			return respondError(c, http.StatusForbidden, "Forbidden")
		}
	}

	count, _ := strconv.Atoi(c.Query("count", "6"))
	data, err := h.sprintService.GetVelocity(projectID, count)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to get velocity data")
	}

	// Populate sprint slugs (handler computes hashid, service layer doesn't depend on it)
	for i := range data {
		data[i].SprintSlug = encodeID(data[i].SprintID)
	}

	return c.Status(http.StatusOK).JSON(data)
}

// GetSprintReport returns sprint report data: scope change timeline and committed vs completed stats.
func (h *SprintHandler) GetSprintReport(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	sprintID, err := decodeID(c.Params("sprintSlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid sprint ID")
	}

	sprint, err := h.sprintService.GetSprint(sprintID)
	if err != nil {
		if err == service.ErrSprintNotFound {
			return respondError(c, http.StatusNotFound, "Sprint not found")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to get sprint")
	}

	project, err := h.projectService.GetProject(sprint.ProjectID)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to get project")
	}
	if project.IsPrivate {
		if _, err := h.projectService.RequireRole(project.ID, user.UserName, model.ProjectRoleViewer); err != nil {
			return respondError(c, http.StatusForbidden, "Forbidden")
		}
	}

	report, err := h.sprintService.GetSprintReport(sprint.ProjectID, sprintID)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to get sprint report")
	}

	return c.Status(http.StatusOK).JSON(report)
}
