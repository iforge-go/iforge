package handler

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/model"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type UserStoryHandler struct {
	userStoryService    *service.UserStoryService
	projectService      *service.ProjectService
	activityService     *service.ScrumActivityService
	aiService           *service.AIService
	notificationService *service.NotificationService
	epicService         *service.EpicService
}

func NewUserStoryHandler(userStoryService *service.UserStoryService, projectService *service.ProjectService, activityService *service.ScrumActivityService, aiService *service.AIService, notificationService *service.NotificationService, epicService *service.EpicService) *UserStoryHandler {
	return &UserStoryHandler{
		userStoryService:    userStoryService,
		projectService:      projectService,
		activityService:     activityService,
		aiService:           aiService,
		notificationService: notificationService,
		epicService:         epicService,
	}
}

type CreateUserStoryRequest struct {
	Title              string  `json:"title" validate:"required"`
	Description        *string `json:"description"`
	Status             string  `json:"status" validate:"required"`   // open, in_progress, done, closed
	Priority           string  `json:"priority" validate:"required"` // low, medium, high, urgent
	StoryPoints        *int    `json:"storyPoints"`
	AcceptanceCriteria *string `json:"acceptanceCriteria"`
	AssigneeName       *string `json:"assigneeName"`
	ReporterName       *string `json:"reporterName"`
	EpicSlug           *string `json:"epicSlug"`
	SprintSlug         *string `json:"sprintSlug"`
}

type UpdateUserStoryRequest struct {
	Title              *string `json:"title"`
	Description        *string `json:"description"`
	Status             *string `json:"status"`
	Priority           *string `json:"priority"`
	StoryPoints        *int    `json:"storyPoints"`
	AcceptanceCriteria *string `json:"acceptanceCriteria"`
	AssigneeName       *string `json:"assigneeName"`
	EpicSlug           *string `json:"epicSlug"`
	SprintSlug         *string `json:"sprintSlug"`
}

func (h *UserStoryHandler) CreateUserStory(c *fiber.Ctx) error {
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
		return respondError(c, http.StatusForbidden, "Only project members can create user stories")
	}

	var req CreateUserStoryRequest
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid request body")
	}

	if req.Title == "" {
		return respondError(c, http.StatusBadRequest, "Title is required")
	}

	reporterName := user.UserName
	if req.ReporterName != nil {
		reporterName = *req.ReporterName
	}

	var epicID *int
	if req.EpicSlug != nil && *req.EpicSlug != "" {
		eid, err := decodeID(*req.EpicSlug)
		if err != nil {
			return respondError(c, http.StatusBadRequest, "Invalid epic slug")
		}
		epicID = &eid
	}

	var sprintID *int
	if req.SprintSlug != nil && *req.SprintSlug != "" {
		sid, err := decodeID(*req.SprintSlug)
		if err != nil {
			return respondError(c, http.StatusBadRequest, "Invalid sprint slug")
		}
		sprintID = &sid
	}

	userStory, err := h.userStoryService.CreateUserStory(
		projectID, req.Title, req.Status, req.Priority,
		req.Description, req.AcceptanceCriteria, req.StoryPoints,
		req.AssigneeName, &reporterName, epicID, sprintID,
	)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to create user story")
	}

	userStory.Slug = encodeID(userStory.ID)
	userStory.ProjectSlug = encodeID(userStory.ProjectID)
	h.fillEpicInfo(userStory)
	h.fillSprintInfo(userStory)

	if h.activityService != nil {
		_ = h.activityService.LogActivity(user.UserName, userStory.ProjectID, "story", userStory.ID, "created", "", "", userStory.Title)
		if userStory.SprintID != nil {
			_ = h.activityService.LogActivity(user.UserName, userStory.ProjectID, "sprint", *userStory.SprintID, "sprint_scope_added", "story", "", userStory.Title)
		}
	}
	return c.Status(http.StatusCreated).JSON(userStory)
}

func (h *UserStoryHandler) fillEpicInfo(story *model.UserStory) {
	if story == nil || story.EpicID == nil || h.epicService == nil {
		return
	}
	slug := encodeID(*story.EpicID)
	story.EpicSlug = &slug
	if epic, err := h.epicService.GetEpic(*story.EpicID); err == nil {
		titleCopy := epic.Title
		story.EpicTitle = &titleCopy
	}
}

func (h *UserStoryHandler) fillEpicInfoBatch(stories []*model.UserStory) {
	if h.epicService == nil {
		return
	}
	epicIDSet := make(map[int]struct{})
	for _, s := range stories {
		if s != nil && s.EpicID != nil {
			epicIDSet[*s.EpicID] = struct{}{}
		}
	}
	if len(epicIDSet) == 0 {
		return
	}
	epicTitles := make(map[int]string)
	for epicID := range epicIDSet {
		if epic, err := h.epicService.GetEpic(epicID); err == nil {
			epicTitles[epicID] = epic.Title
		}
	}
	for _, s := range stories {
		if s != nil && s.EpicID != nil {
			slug := encodeID(*s.EpicID)
			s.EpicSlug = &slug
			if title, ok := epicTitles[*s.EpicID]; ok {
				titleCopy := title
				s.EpicTitle = &titleCopy
			}
		}
	}
}

func (h *UserStoryHandler) fillTaskCountBatch(stories []*model.UserStory) {
	if h.userStoryService == nil || len(stories) == 0 {
		return
	}
	storyIDs := make([]int, 0, len(stories))
	var projectID int
	for _, s := range stories {
		if s != nil {
			storyIDs = append(storyIDs, s.ID)
			projectID = s.ProjectID
		}
	}
	if len(storyIDs) == 0 {
		return
	}
	counts, err := h.userStoryService.GetStoryTaskCounts(projectID, storyIDs)
	if err != nil {
		return
	}
	for _, s := range stories {
		if s != nil {
			s.TaskCount = counts[s.ID]
		}
	}
}

func (h *UserStoryHandler) fillSprintInfo(story *model.UserStory) {
	if story == nil || story.SprintID == nil {
		return
	}
	slug := encodeID(*story.SprintID)
	story.SprintSlug = &slug
}

func (h *UserStoryHandler) fillSprintInfoBatch(stories []*model.UserStory) {
	for _, s := range stories {
		if s != nil && s.SprintID != nil {
			slug := encodeID(*s.SprintID)
			s.SprintSlug = &slug
		}
	}
}

func (h *UserStoryHandler) GetUserStory(c *fiber.Ctx) error {
	id, err := decodeID(c.Params("userStorySlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid user story ID")
	}

	userStory, err := h.userStoryService.GetUserStory(id)
	if err != nil {
		if err == service.ErrUserStoryNotFound {
			return respondError(c, http.StatusNotFound, "User story not found")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to get user story")
	}

	userStory.Slug = encodeID(userStory.ID)
	userStory.ProjectSlug = encodeID(userStory.ProjectID)
	h.fillEpicInfo(userStory)
	h.fillSprintInfo(userStory)
	return c.Status(http.StatusOK).JSON(userStory)
}

func (h *UserStoryHandler) UpdateUserStory(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	id, err := decodeID(c.Params("userStorySlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid user story ID")
	}

	userStory, err := h.userStoryService.GetUserStory(id)
	if err != nil {
		if err == service.ErrUserStoryNotFound {
			return respondError(c, http.StatusNotFound, "User story not found")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to get user story")
	}

	project, err := h.projectService.GetProject(userStory.ProjectID)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to get project")
	}

	if _, err := h.projectService.RequireRole(project.ID, user.UserName, model.ProjectRoleMember); err != nil {
		return respondError(c, http.StatusForbidden, "Only project members can update user stories")
	}

	var req UpdateUserStoryRequest
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid request body")
	}

	oldTitle := userStory.Title
	oldDescription := userStory.Description
	oldStatus := userStory.Status
	oldPriority := userStory.Priority
	oldStoryPoints := userStory.StoryPoints
	oldAcceptanceCriteria := userStory.AcceptanceCriteria
	oldAssigneeName := userStory.AssigneeName
	oldEpicID := userStory.EpicID
	oldSprintID := userStory.SprintID

	if req.Title != nil {
		userStory.Title = *req.Title
	}
	if req.Description != nil {
		userStory.Description = req.Description
	}
	if req.Status != nil {
		userStory.Status = *req.Status
	}
	if req.Priority != nil {
		userStory.Priority = *req.Priority
	}
	if req.StoryPoints != nil {
		userStory.StoryPoints = req.StoryPoints
	}
	if req.AcceptanceCriteria != nil {
		userStory.AcceptanceCriteria = req.AcceptanceCriteria
	}
	if req.EpicSlug != nil {
		if *req.EpicSlug == "" {
			userStory.EpicID = nil
		} else {
			eid, err := decodeID(*req.EpicSlug)
			if err != nil {
				return respondError(c, http.StatusBadRequest, "Invalid epic slug")
			}
			userStory.EpicID = &eid
		}
	}
	if req.SprintSlug != nil {
		if *req.SprintSlug == "" {
			userStory.SprintID = nil
		} else {
			sid, err := decodeID(*req.SprintSlug)
			if err != nil {
				return respondError(c, http.StatusBadRequest, "Invalid sprint slug")
			}
			userStory.SprintID = &sid
		}
	}
	if req.AssigneeName != nil {
		if *req.AssigneeName == "" {
			userStory.AssigneeName = nil
		} else {
			userStory.AssigneeName = req.AssigneeName
			if err := h.projectService.EnsureMember(project.ID, *req.AssigneeName, model.ProjectRoleMember); err != nil {
				fmt.Printf("Warning: failed to ensure project membership for story assignee %s: %v\n", *req.AssigneeName, err)
			}
			newAssignee := *req.AssigneeName
			oldAssigneeStr := ""
			if oldAssigneeName != nil {
				oldAssigneeStr = *oldAssigneeName
			}
			if newAssignee != user.UserName && newAssignee != oldAssigneeStr {
				msg := fmt.Sprintf("%s assigned you to story #%d: %s", user.UserName, userStory.ID, userStory.Title)
				if h.notificationService != nil {
					_ = h.notificationService.CreateStoryNotification(
						newAssignee,
						"story_assigned",
						user.UserName,
						msg,
						&project.ID,
						&userStory.ID,
					)
				}
			}
		}
	}

	if err := h.userStoryService.UpdateUserStory(userStory); err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to update user story")
	}

	if h.activityService != nil {
		if oldTitle != userStory.Title {
			_ = h.activityService.LogActivity(user.UserName, userStory.ProjectID, "story", userStory.ID, "updated", "title", oldTitle, userStory.Title)
		}
		if ptrStringValue(oldDescription) != ptrStringValue(userStory.Description) {
			_ = h.activityService.LogActivity(user.UserName, userStory.ProjectID, "story", userStory.ID, "updated", "description", ptrStringValue(oldDescription), ptrStringValue(userStory.Description))
		}
		if oldStatus != userStory.Status {
			_ = h.activityService.LogActivity(user.UserName, userStory.ProjectID, "story", userStory.ID, "status_changed", "status", oldStatus, userStory.Status)
		}
		if oldPriority != userStory.Priority {
			_ = h.activityService.LogActivity(user.UserName, userStory.ProjectID, "story", userStory.ID, "updated", "priority", oldPriority, userStory.Priority)
		}
		if ptrIntValue(oldStoryPoints) != ptrIntValue(userStory.StoryPoints) {
			_ = h.activityService.LogActivity(user.UserName, userStory.ProjectID, "story", userStory.ID, "updated", "storyPoints", ptrIntValue(oldStoryPoints), ptrIntValue(userStory.StoryPoints))
		}
		if ptrStringValue(oldAcceptanceCriteria) != ptrStringValue(userStory.AcceptanceCriteria) {
			_ = h.activityService.LogActivity(user.UserName, userStory.ProjectID, "story", userStory.ID, "updated", "acceptanceCriteria", ptrStringValue(oldAcceptanceCriteria), ptrStringValue(userStory.AcceptanceCriteria))
		}
		if ptrStringValue(oldAssigneeName) != ptrStringValue(userStory.AssigneeName) {
			_ = h.activityService.LogActivity(user.UserName, userStory.ProjectID, "story", userStory.ID, "updated", "assignee", ptrStringValue(oldAssigneeName), ptrStringValue(userStory.AssigneeName))
		}
		if !ptrIntEqual(oldEpicID, userStory.EpicID) {
			oldEpicSlug := ""
			if oldEpicID != nil {
				oldEpicSlug = encodeID(*oldEpicID)
			}
			newEpicSlug := ""
			if userStory.EpicID != nil {
				newEpicSlug = encodeID(*userStory.EpicID)
			}
			_ = h.activityService.LogActivity(user.UserName, userStory.ProjectID, "story", userStory.ID, "updated", "epic", oldEpicSlug, newEpicSlug)
		}
		if !ptrIntEqual(oldSprintID, userStory.SprintID) {
			oldSprintSlug := ""
			if oldSprintID != nil {
				oldSprintSlug = encodeID(*oldSprintID)
			}
			newSprintSlug := ""
			if userStory.SprintID != nil {
				newSprintSlug = encodeID(*userStory.SprintID)
			}
			_ = h.activityService.LogActivity(user.UserName, userStory.ProjectID, "story", userStory.ID, "updated", "sprint", oldSprintSlug, newSprintSlug)
			if oldSprintID != nil {
				_ = h.activityService.LogActivity(user.UserName, userStory.ProjectID, "sprint", *oldSprintID, "sprint_scope_removed", "story", "", userStory.Title)
			}
			if userStory.SprintID != nil {
				_ = h.activityService.LogActivity(user.UserName, userStory.ProjectID, "sprint", *userStory.SprintID, "sprint_scope_added", "story", "", userStory.Title)
			}
		}
	}

	userStory.Slug = encodeID(userStory.ID)
	userStory.ProjectSlug = encodeID(userStory.ProjectID)
	h.fillEpicInfo(userStory)
	h.fillSprintInfo(userStory)
	return c.Status(http.StatusOK).JSON(userStory)
}

// DeleteUserStory deletes a user story
func (h *UserStoryHandler) DeleteUserStory(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	id, err := decodeID(c.Params("userStorySlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid user story ID")
	}

	userStory, err := h.userStoryService.GetUserStory(id)
	if err != nil {
		if err == service.ErrUserStoryNotFound {
			return respondError(c, http.StatusNotFound, "User story not found")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to get user story")
	}

	// Check project access
	project, err := h.projectService.GetProject(userStory.ProjectID)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to get project")
	}

	if _, err := h.projectService.RequireRole(project.ID, user.UserName, model.ProjectRoleMember); err != nil {
		return respondError(c, http.StatusForbidden, "Only project members can delete user stories")
	}

	if err := h.userStoryService.DeleteUserStory(id); err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to delete user story")
	}

	// Record deletion activity
	if h.activityService != nil {
		_ = h.activityService.LogActivity(user.UserName, userStory.ProjectID, "story", userStory.ID, "deleted", "", "", userStory.Title)
	}

	return c.SendStatus(http.StatusNoContent)
}

// ListUserStories lists user stories for a project
func (h *UserStoryHandler) ListUserStories(c *fiber.Ctx) error {
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

	var epicIDPtr *int
	if epicSlug := c.Query("epicSlug"); epicSlug != "" {
		eid, err := decodeID(epicSlug)
		if err != nil {
			return respondError(c, http.StatusBadRequest, "Invalid epic slug")
		}
		epicIDPtr = &eid
	}

	var sprintIDPtr *int
	if sprintSlug := c.Query("sprintSlug"); sprintSlug != "" {
		sid, err := decodeID(sprintSlug)
		if err != nil {
			return respondError(c, http.StatusBadRequest, "Invalid sprint slug")
		}
		sprintIDPtr = &sid
	}
	// backlog=true returns only stories not assigned to any sprint
	backlogFlag := c.Query("backlog") == "true"

	limit, _ := strconv.Atoi(c.Query("limit", "30"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	userStories, err := h.userStoryService.ListUserStories(projectID, statusPtr, epicIDPtr, sprintIDPtr, backlogFlag, limit, offset)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to list user stories")
	}

	for i := range userStories {
		userStories[i].Slug = encodeID(userStories[i].ID)
		userStories[i].ProjectSlug = encodeID(userStories[i].ProjectID)
	}
	h.fillEpicInfoBatch(userStories)
	h.fillSprintInfoBatch(userStories)
	h.fillTaskCountBatch(userStories)

	return c.Status(http.StatusOK).JSON(userStories)
}

// GetUserStoryTasks retrieves all tasks for a user story
func (h *UserStoryHandler) GetUserStoryTasks(c *fiber.Ctx) error {
	id, err := decodeID(c.Params("userStorySlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid user story ID")
	}

	tasks, err := h.userStoryService.GetUserStoryTasks(id)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to get user story tasks")
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

// AIDecomposeUserStory generates 3-8 suggested tasks for a user story using LLM.
// Uses SSE (Server-Sent Events) to stream token deltas to the client in real-time,
// then sends a final "done" event with the structured task list.
// Does NOT create tasks; the client decides which suggestions to persist via createTask.
func (h *UserStoryHandler) AIDecomposeUserStory(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	id, err := decodeID(c.Params("userStorySlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid user story ID")
	}

	story, err := h.userStoryService.GetUserStory(id)
	if err != nil {
		if err == service.ErrUserStoryNotFound {
			return respondError(c, http.StatusNotFound, "User story not found")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to get user story")
	}

	if _, ok := checkProjectWriteAccess(c, h.projectService, story.ProjectID); !ok {
		return nil
	}

	if h.aiService == nil {
		return respondError(c, http.StatusServiceUnavailable, "AI service not configured")
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

		result, err := h.aiService.DecomposeUserStoryStream(
			story, lang,
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

// AIOptimizeUserStory optimizes a user story draft using LLM via SSE streaming.
// Accepts the current draft (title/description/acceptanceCriteria) in the request body
// and returns optimized content + suggested priority/storyPoints.
// Unlike AIDecomposeUserStory, this does NOT require an existing story — it works on draft content.
func (h *UserStoryHandler) AIOptimizeUserStory(c *fiber.Ctx) error {
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
		Title              string `json:"title"`
		Description        string `json:"description"`
		AcceptanceCriteria string `json:"acceptanceCriteria"`
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

		result, err := h.aiService.OptimizeUserStoryStream(
			req.Title, req.Description, req.AcceptanceCriteria, lang,
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
