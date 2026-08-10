package handler

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/event"
	"iforge/iforge/internal/git"
	"iforge/iforge/internal/model"
	"iforge/iforge/internal/service"
	"iforge/iforge/internal/util"

	"github.com/gofiber/fiber/v2"
)

func parseAssignees(s *string) []string {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil
	}
	parts := strings.Split(*s, ",")
	var result []string
	for _, p := range parts {
		name := strings.TrimSpace(p)
		if name != "" {
			result = append(result, name)
		}
	}
	return result
}

func attachTaskSlugs(task *model.Task) {
	task.Slug = encodeID(task.TaskID)
	task.ProjectSlug = encodeID(task.ProjectID)
	if task.SprintID != nil {
		slug := encodeID(*task.SprintID)
		task.SprintSlug = &slug
	}
	if task.UserStoryID != nil {
		slug := encodeID(*task.UserStoryID)
		task.UserStorySlug = &slug
	}
}

func findAddedAssignees(current, newAssignees []string) []string {
	currentSet := make(map[string]bool, len(current))
	for _, u := range current {
		currentSet[u] = true
	}
	var added []string
	for _, u := range newAssignees {
		if !currentSet[u] {
			added = append(added, u)
		}
	}
	return added
}

func (h *TaskHandler) notifyNewAssignees(user *model.Account, projectID, taskID int, taskTitle string, added []string) {
	if h.notificationService == nil || len(added) == 0 {
		return
	}
	for _, assignee := range added {
		if assignee == user.UserName {
			continue
		}
		msg := fmt.Sprintf("%s assigned you to task #%d: %s", user.UserName, taskID, taskTitle)
		_ = h.notificationService.CreateTaskNotification(
			assignee,
			"task_assigned",
			user.UserName,
			msg,
			&projectID,
			&taskID,
		)
	}
}

// notifyMentionedUsers parses @username mentions in content and notifies mentioned users.
// Skips the actor; batch-validates users via accountService then inserts notifications in one query.
// Best-effort: errors do not affect the comment creation result.
func (h *TaskHandler) notifyMentionedUsers(actor *model.Account, projectID, taskID int, taskTitle, content string) {
	if h.notificationService == nil || h.accountService == nil {
		return
	}
	mentions := util.ParseMentions(content)
	if len(mentions) == 0 {
		return
	}
	// Filter out the actor themselves (ParseMentions already deduplicates)
	filtered := make([]string, 0, len(mentions))
	for _, u := range mentions {
		if u != actor.UserName {
			filtered = append(filtered, u)
		}
	}
	if len(filtered) == 0 {
		return
	}
	existing, err := h.accountService.FilterExistingUserNames(filtered)
	if err != nil || len(existing) == 0 {
		return
	}
	now := time.Now()
	msg := fmt.Sprintf("%s mentioned you in task #%d: %s", actor.UserName, taskID, taskTitle)
	notifications := make([]*model.Notification, 0, len(existing))
	for _, username := range existing {
		notifications = append(notifications, &model.Notification{
			RecipientUserName: username,
			NotificationType:  "mentioned",
			Actor:             actor.UserName,
			Message:           msg,
			ProjectID:         &projectID,
			TaskID:            &taskID,
			RegisteredDate:    now,
		})
	}
	_ = h.notificationService.BatchCreateNotifications(notifications)
}

func (h *TaskHandler) logTaskFieldChange(userName string, projectID, taskID int, activityType, field, oldValue, newValue string) {
	if oldValue == newValue {
		return
	}
	_ = h.activityService.LogActivity(userName, projectID, "task", taskID, activityType, field, oldValue, newValue)
}

func (h *TaskHandler) broadcastTaskUpdate(user *model.Account, projectID int, task *model.Task) {
	if h.notificationService == nil {
		return
	}
	recipients := make([]string, 0, 4)
	if project, err := h.projectService.GetProject(projectID); err == nil {
		recipients = append(recipients, project.OwnerName)
	}
	if members, err := h.projectService.GetMembers(projectID); err == nil {
		for _, m := range members {
			recipients = append(recipients, m.UserName)
		}
	}
	seen := make(map[string]bool, len(recipients))
	filtered := make([]string, 0, len(recipients))
	for _, u := range recipients {
		if u == "" || seen[u] {
			continue
		}
		seen[u] = true
		filtered = append(filtered, u)
	}
	if len(filtered) == 0 {
		return
	}
	payload := fiber.Map{
		"task":        task,
		"projectSlug": encodeID(projectID),
		"actor":       user.UserName,
	}
	h.notificationService.PushToUsers(filtered, "task_updated", payload)
}

type taskSnapshot struct {
	title          string
	description    *string
	status         string
	priority       string
	taskType       string
	storyPoints    *int
	estimatedHours *float64
	assigneeName   *string
	sprintID       *int
	userStoryID    *int
}

func snapshotTask(t *model.Task) taskSnapshot {
	return taskSnapshot{
		title:          t.Title,
		description:    t.Description,
		status:         t.Status,
		priority:       t.Priority,
		taskType:       t.TaskType,
		storyPoints:    t.StoryPoints,
		estimatedHours: t.EstimatedHours,
		assigneeName:   t.AssigneeName,
		sprintID:       t.SprintID,
		userStoryID:    t.UserStoryID,
	}
}

func (h *TaskHandler) logTaskUpdateActivities(user *model.Account, task *model.Task, snap taskSnapshot) {
	if snap.status != task.Status {
		_ = h.taskService.RecordStatusHistory(task.ProjectID, task.TaskID, snap.status, task.Status, user.UserName, nil)
	}

	if h.activityService == nil {
		return
	}
	h.logTaskFieldChange(user.UserName, task.ProjectID, task.TaskID, "updated", "title", snap.title, task.Title)
	h.logTaskFieldChange(user.UserName, task.ProjectID, task.TaskID, "updated", "description", ptrStringValue(snap.description), ptrStringValue(task.Description))
	h.logTaskFieldChange(user.UserName, task.ProjectID, task.TaskID, "status_changed", "status", snap.status, task.Status)
	h.logTaskFieldChange(user.UserName, task.ProjectID, task.TaskID, "updated", "priority", snap.priority, task.Priority)
	h.logTaskFieldChange(user.UserName, task.ProjectID, task.TaskID, "updated", "taskType", snap.taskType, task.TaskType)
	h.logTaskFieldChange(user.UserName, task.ProjectID, task.TaskID, "updated", "storyPoints", ptrIntValue(snap.storyPoints), ptrIntValue(task.StoryPoints))
	h.logTaskFieldChange(user.UserName, task.ProjectID, task.TaskID, "updated", "estimatedHours", ptrFloat64Value(snap.estimatedHours), ptrFloat64Value(task.EstimatedHours))
	h.logTaskFieldChange(user.UserName, task.ProjectID, task.TaskID, "updated", "assignee", ptrStringValue(snap.assigneeName), ptrStringValue(task.AssigneeName))
	oldSprintVal, newSprintVal := "", ""
	if snap.sprintID != nil {
		oldSprintVal = h.taskService.GetSprintTitle(*snap.sprintID)
	}
	if task.SprintID != nil {
		newSprintVal = h.taskService.GetSprintTitle(*task.SprintID)
	}
	h.logTaskFieldChange(user.UserName, task.ProjectID, task.TaskID, "updated", "sprintId", oldSprintVal, newSprintVal)
	// Log sprint scope changes for Sprint Report
	if !ptrIntEqual(snap.sprintID, task.SprintID) {
		if snap.sprintID != nil {
			_ = h.activityService.LogActivity(user.UserName, task.ProjectID, "sprint", *snap.sprintID, "sprint_scope_removed", "task", "", task.Title)
		}
		if task.SprintID != nil {
			_ = h.activityService.LogActivity(user.UserName, task.ProjectID, "sprint", *task.SprintID, "sprint_scope_added", "task", "", task.Title)
		}
	}
	// UserStory: resolve IDs to titles for readability
	oldStoryVal, newStoryVal := "", ""
	if snap.userStoryID != nil {
		oldStoryVal = h.taskService.GetUserStoryTitle(*snap.userStoryID)
	}
	if task.UserStoryID != nil {
		newStoryVal = h.taskService.GetUserStoryTitle(*task.UserStoryID)
	}
	h.logTaskFieldChange(user.UserName, task.ProjectID, task.TaskID, "updated", "userStoryId", oldStoryVal, newStoryVal)
}

// getTaskFromContext retrieves a task by (projectId, taskId) from URL params.
// Tasks are addressed via the project-scoped taskId (composite primary key).
func (h *TaskHandler) getTaskFromContext(c *fiber.Ctx) (*model.Task, int, int, error) {
	projectID, err := decodeID(c.Params("projectSlug"))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("invalid project ID")
	}
	taskID, err := strconv.Atoi(c.Params("taskId"))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("invalid task ID")
	}
	task, err := h.taskService.GetTask(projectID, taskID)
	if err != nil {
		return nil, 0, 0, err
	}
	return task, projectID, taskID, nil
}

// TaskHandler handles task endpoints
type TaskHandler struct {
	taskService         *service.TaskService
	projectService      *service.ProjectService
	notificationService *service.NotificationService
	accountService      *service.AccountService
	activityService     *service.ScrumActivityService
	gitClient           *git.Client
	repoService         *service.RepositoryService
	eventBus            *event.Bus
	aiService           *service.AIService
}

// NewTaskHandler creates a new TaskHandler
func NewTaskHandler(taskService *service.TaskService, projectService *service.ProjectService, notificationService *service.NotificationService, accountService *service.AccountService, activityService *service.ScrumActivityService, gitClient *git.Client, repoService *service.RepositoryService, eventBus *event.Bus, aiService *service.AIService) *TaskHandler {
	return &TaskHandler{
		taskService:         taskService,
		projectService:      projectService,
		notificationService: notificationService,
		accountService:      accountService,
		activityService:     activityService,
		gitClient:           gitClient,
		repoService:         repoService,
		eventBus:            eventBus,
		aiService:           aiService,
	}
}

// CreateTaskRequest represents a create task request
type CreateTaskRequest struct {
	Title          string   `json:"title" validate:"required"`
	Description    *string  `json:"description"`
	Status         string   `json:"status" validate:"required"`   // todo, in_progress, review, done
	Priority       string   `json:"priority" validate:"required"` // low, medium, high, urgent
	TaskType       string   `json:"taskType" validate:"required"` // bug, feature, improvement, task
	StoryPoints    *int     `json:"storyPoints"`
	EstimatedHours *float64 `json:"estimatedHours"`
	AssigneeName   *string  `json:"assigneeName"`
	ReporterName   *string  `json:"reporterName"`
	SprintSlug     *string  `json:"sprintSlug"`
	UserStorySlug  *string  `json:"userStorySlug"`
	ParentID       *int     `json:"parentId"`
	Position       int      `json:"position"`
	DueDate        *string  `json:"dueDate"` // ISO date string, e.g. "2026-07-20"
}

// UpdateTaskRequest represents an update task request
type UpdateTaskRequest struct {
	Title          *string  `json:"title"`
	Description    *string  `json:"description"`
	Status         *string  `json:"status"`
	Priority       *string  `json:"priority"`
	TaskType       *string  `json:"taskType"`
	StoryPoints    *int     `json:"storyPoints"`
	EstimatedHours *float64 `json:"estimatedHours"`
	AssigneeName   *string  `json:"assigneeName"`
	Assignees      []string `json:"assignees"`
	SprintSlug     *string  `json:"sprintSlug"`
	UserStorySlug  *string  `json:"userStorySlug"`
	Position       *int     `json:"position"`
	DueDate        *string  `json:"dueDate"` // ISO date string; empty string clears
}

// CreateTask creates a new task
func (h *TaskHandler) CreateTask(c *fiber.Ctx) error {
	projectID, err := decodeID(c.Params("projectSlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid project ID")
	}

	user, ok := checkProjectWriteAccess(c, h.projectService, projectID)
	if !ok {
		return nil
	}

	var req CreateTaskRequest
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

	var sprintIDPtr, userStoryIDPtr *int
	if req.SprintSlug != nil && *req.SprintSlug != "" {
		if id, err := decodeID(*req.SprintSlug); err == nil {
			sprintIDPtr = &id
		}
	}
	if req.UserStorySlug != nil && *req.UserStorySlug != "" {
		if id, err := decodeID(*req.UserStorySlug); err == nil {
			userStoryIDPtr = &id
		}
	}

	// Parse due_date string to time.Time pointer
	var dueDatePtr *time.Time
	if req.DueDate != nil && *req.DueDate != "" {
		if parsed, err := time.Parse("2006-01-02", *req.DueDate); err == nil {
			dueDatePtr = &parsed
		}
	}

	task, err := h.taskService.CreateTask(
		projectID, req.Title, req.Status, req.Priority, req.TaskType,
		req.Description, req.StoryPoints, req.EstimatedHours, req.AssigneeName, &reporterName,
		sprintIDPtr, userStoryIDPtr, req.ParentID, req.Position, dueDatePtr,
	)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to create task")
	}

	attachTaskSlugs(task)

	// Record creation activity
	if h.activityService != nil {
		_ = h.activityService.LogActivity(user.UserName, task.ProjectID, "task", task.TaskID, "created", "", "", task.Title)
		// Log sprint scope when task is created with a sprint
		if task.SprintID != nil {
			_ = h.activityService.LogActivity(user.UserName, task.ProjectID, "sprint", *task.SprintID, "sprint_scope_added", "task", "", task.Title)
		}
	}
	return respondSuccess(c, http.StatusCreated, task)
}

// GetTask retrieves a task by project-scoped taskId
func (h *TaskHandler) GetTask(c *fiber.Ctx) error {
	task, projectID, taskID, err := h.getTaskFromContext(c)
	if err != nil {
		if err == service.ErrTaskNotFound {
			return respondError(c, http.StatusNotFound, "Task not found")
		}
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	if !checkProjectReadAccess(c, h.projectService, task.ProjectID) {
		return nil
	}

	assignees, _ := h.taskService.GetAssignees(projectID, taskID)
	if assignees == nil {
		assignees = []*model.Account{}
	}

	subtaskTotalMap, subtaskDoneMap, _ := h.taskService.GetSubtaskCounts(projectID, []int{taskID})
	blockingInProgress, _ := h.taskService.GetBlockingTasks(projectID, taskID, "in_progress")
	blockingDone, _ := h.taskService.GetBlockingTasks(projectID, taskID, "done")
	blockingTasks := append(blockingInProgress, blockingDone...)
	if blockingTasks == nil {
		blockingTasks = []service.BlockedTaskInfo{}
	}

	attachTaskSlugs(task)

	// Sideload work logs + actual hours for time tracking
	workLogs, _ := h.taskService.GetWorkLogs(projectID, taskID)
	if workLogs == nil {
		workLogs = []*model.TaskWorkLog{}
	}
	actualHours, _ := h.taskService.GetActualHours(projectID, taskID)
	for i := range workLogs {
		workLogs[i].ProjectSlug = encodeID(workLogs[i].ProjectID)
	}

	return respondSuccess(c, http.StatusOK, fiber.Map{
		"task":                  task,
		"assignees":             assignees,
		"subtaskCount":          subtaskTotalMap[taskID],
		"subtaskCompletedCount": subtaskDoneMap[taskID],
		"blockedByCount":        len(blockingTasks),
		"blockingTasks":         blockingTasks,
		"workLogs":              workLogs,
		"actualHours":           actualHours,
	})
}

// UpdateTask updates a task
func (h *TaskHandler) UpdateTask(c *fiber.Ctx) error {
	task, projectID, taskID, err := h.getTaskFromContext(c)
	if err != nil {
		if err == service.ErrTaskNotFound {
			return respondError(c, http.StatusNotFound, "Task not found")
		}
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	user, ok := checkProjectWriteAccess(c, h.projectService, task.ProjectID)
	if !ok {
		return nil
	}

	var req UpdateTaskRequest
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid request body")
	}

	// Snapshot old values for activity logging
	snap := snapshotTask(task)

	if req.Title != nil {
		task.Title = *req.Title
	}
	if req.Description != nil {
		task.Description = req.Description
	}
	if req.Status != nil {
		// Validate workflow transition rules when status actually changes
		if *req.Status != snap.status {
			allowed, err := h.taskService.IsStatusTransitionAllowed(projectID, snap.status, *req.Status)
			if err != nil {
				return respondError(c, http.StatusInternalServerError, "Failed to validate status transition")
			}
			if !allowed {
				return respondError(c, http.StatusUnprocessableEntity, "Status transition not allowed by project workflow rules")
			}
			// Block status transitions to in-progress if dependencies are incomplete
			if err := h.taskService.CheckBlock(projectID, taskID, *req.Status); err != nil {
				var blockErr *service.BlockedError
				if errors.As(err, &blockErr) {
					return c.Status(http.StatusUnprocessableEntity).JSON(fiber.Map{
						"error":        "Task is blocked by unfinished dependencies",
						"reason":       "task_blocked",
						"blockedTasks": blockErr.BlockingTasks,
					})
				}
				return respondError(c, http.StatusInternalServerError, "Failed to check task dependencies")
			}
		}
		task.Status = *req.Status
	}
	if req.Priority != nil {
		task.Priority = *req.Priority
	}
	if req.TaskType != nil {
		task.TaskType = *req.TaskType
	}
	if req.StoryPoints != nil {
		task.StoryPoints = req.StoryPoints
	}
	if req.EstimatedHours != nil {
		task.EstimatedHours = req.EstimatedHours
	}
	// Handle multi-assignees (new approach)
	if req.Assignees != nil {
		currentAssignees, _ := h.taskService.GetAssignees(projectID, taskID)
		currentUsernames := make([]string, 0, len(currentAssignees))
		for _, a := range currentAssignees {
			currentUsernames = append(currentUsernames, a.UserName)
		}
		added := findAddedAssignees(currentUsernames, req.Assignees)

		// Auto-add new non-member assignees as project members
		for _, userName := range added {
			if err := h.projectService.EnsureMember(projectID, userName, model.ProjectRoleMember); err != nil {
				fmt.Printf("Warning: failed to ensure project membership for task assignee %s: %v\n", userName, err)
			}
		}

		if err := h.taskService.SetAssignees(projectID, taskID, req.Assignees); err != nil {
			return respondError(c, http.StatusInternalServerError, "Failed to update assignees")
		}

		h.notifyNewAssignees(user, projectID, task.TaskID, task.Title, added)
	} else if req.AssigneeName != nil {
		// Legacy single-assignee support (backward compatibility)
		oldAssignees := parseAssignees(task.AssigneeName)
		task.AssigneeName = req.AssigneeName
		newAssignees := parseAssignees(req.AssigneeName)
		added := findAddedAssignees(oldAssignees, newAssignees)

		// Auto-add new non-member assignees as project members
		for _, userName := range added {
			if err := h.projectService.EnsureMember(projectID, userName, model.ProjectRoleMember); err != nil {
				fmt.Printf("Warning: failed to ensure project membership for task assignee %s: %v\n", userName, err)
			}
		}

		h.notifyNewAssignees(user, projectID, task.TaskID, task.Title, added)
	}
	if req.SprintSlug != nil {
		if *req.SprintSlug == "" {
			task.SprintID = nil
		} else if sprintID, err := decodeID(*req.SprintSlug); err == nil {
			task.SprintID = &sprintID
		}
	}
	if req.UserStorySlug != nil {
		if *req.UserStorySlug == "" {
			task.UserStoryID = nil
		} else if userStoryID, err := decodeID(*req.UserStorySlug); err == nil {
			task.UserStoryID = &userStoryID
		}
	}
	if req.Position != nil {
		task.Position = *req.Position
	}
	// Handle due_date: empty string clears, ISO date parses
	if req.DueDate != nil {
		if *req.DueDate == "" {
			task.DueDate = nil
		} else if parsed, err := time.Parse("2006-01-02", *req.DueDate); err == nil {
			task.DueDate = &parsed
		}
	}

	if err := h.taskService.UpdateTask(task); err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to update task")
	}

	// Record status history + field-change activities
	h.logTaskUpdateActivities(user, task, snap)

	// Attach assignees for frontend consistency
	assignees, _ := h.taskService.GetAssignees(projectID, taskID)
	if assignees == nil {
		assignees = []*model.Account{}
	}

	attachTaskSlugs(task)
	// Broadcast task update to other online project members
	h.broadcastTaskUpdate(user, task.ProjectID, task)
	return respondSuccess(c, http.StatusOK, fiber.Map{
		"task":      task,
		"assignees": assignees,
	})
}

// DeleteTask deletes a task
func (h *TaskHandler) DeleteTask(c *fiber.Ctx) error {
	task, projectID, taskID, err := h.getTaskFromContext(c)
	if err != nil {
		if err == service.ErrTaskNotFound {
			return respondError(c, http.StatusNotFound, "Task not found")
		}
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	user, ok := checkProjectWriteAccess(c, h.projectService, task.ProjectID)
	if !ok {
		return nil
	}

	if err := h.taskService.DeleteTask(projectID, taskID); err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to delete task")
	}

	// Record deletion activity
	if h.activityService != nil {
		_ = h.activityService.LogActivity(user.UserName, task.ProjectID, "task", task.TaskID, "deleted", "", "", task.Title)
	}

	return c.SendStatus(http.StatusNoContent)
}

// ListTasks lists tasks for a project
func (h *TaskHandler) ListTasks(c *fiber.Ctx) error {
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
	taskType := c.Query("task_type")
	sprintSlugStr := c.Query("sprint_slug")
	userStorySlugStr := c.Query("user_story_slug")
	q := c.Query("q")

	var statusPtr, taskTypePtr, qPtr *string
	var sprintIDPtr, userStoryIDPtr *int

	if status != "" {
		statusPtr = &status
	}
	if taskType != "" {
		taskTypePtr = &taskType
	}
	if q != "" {
		qPtr = &q
	}
	if sprintSlugStr != "" {
		if sprintID, err := decodeID(sprintSlugStr); err == nil {
			sprintIDPtr = &sprintID
		}
	}
	if userStorySlugStr != "" {
		if userStoryID, err := decodeID(userStorySlugStr); err == nil {
			userStoryIDPtr = &userStoryID
		}
	}

	limit, _ := strconv.Atoi(c.Query("limit", "30"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	tasks, err := h.taskService.ListTasks(projectID, statusPtr, taskTypePtr, sprintIDPtr, userStoryIDPtr, qPtr, limit, offset)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to list tasks")
	}

	// Return total count for pagination
	total, err := h.taskService.CountTasks(projectID, statusPtr, taskTypePtr, sprintIDPtr, userStoryIDPtr, qPtr)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to count tasks")
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

	// Sideload: fetch assignee usernames in a single query
	taskIDs := make([]int, len(tasks))
	for i, task := range tasks {
		taskIDs[i] = task.TaskID
	}
	assigneeMap, _ := h.taskService.BatchGetAssignees(projectID, taskIDs)
	subtaskTotalMap, subtaskDoneMap, _ := h.taskService.GetSubtaskCounts(projectID, taskIDs)
	epicMap, _ := h.taskService.GetTaskEpics(projectID, taskIDs)
	storyMap, _ := h.taskService.GetTaskUserStories(projectID, taskIDs)
	blockedByCountMap, _ := h.taskService.BatchGetBlockedByCounts(projectID, taskIDs)

	// Collect deduplicated usernames across all tasks
	seen := make(map[string]bool)
	var userNames []string
	for _, id := range taskIDs {
		for _, name := range assigneeMap[id] {
			if !seen[name] {
				seen[name] = true
				userNames = append(userNames, name)
			}
		}
	}
	avatarInfo, _ := h.accountService.GetUserAvatarInfo(userNames)
	assignees := h.accountService.BuildParticipants(avatarInfo, userNames)

	// Build task items: carry only assigneeUserNames (references);
	// hide legacy single-assignee field to avoid redundancy in sideload response.
	type taskListItem struct {
		*model.Task
		AssigneeName          *string  `json:"-"`
		AssigneeUserNames     []string `json:"assigneeUserNames"`
		SubtaskCount          int      `json:"subtaskCount"`
		SubtaskCompletedCount int      `json:"subtaskCompletedCount"`
		EpicSlug              *string  `json:"epicSlug"`
		EpicTitle             *string  `json:"epicTitle"`
		UserStoryTitle        *string  `json:"userStoryTitle"`
		BlockedByCount        int      `json:"blockedByCount"`
	}

	items := make([]taskListItem, 0, len(tasks))
	for _, task := range tasks {
		names := assigneeMap[task.TaskID]
		if names == nil {
			names = []string{}
		}
		item := taskListItem{
			Task:                  task,
			AssigneeUserNames:     names,
			SubtaskCount:          subtaskTotalMap[task.TaskID],
			SubtaskCompletedCount: subtaskDoneMap[task.TaskID],
			BlockedByCount:        blockedByCountMap[task.TaskID],
		}
		// Populate epic information
		if info, ok := epicMap[task.TaskID]; ok {
			epicSlug := encodeID(info.EpicID)
			epicTitle := info.EpicTitle
			item.EpicSlug = &epicSlug
			item.EpicTitle = &epicTitle
		}
		// Populate user story title
		if info, ok := storyMap[task.TaskID]; ok {
			title := info.UserStoryTitle
			item.UserStoryTitle = &title
		}
		items = append(items, item)
	}

	return respondSuccess(c, http.StatusOK, fiber.Map{
		"tasks":     items,
		"assignees": assignees,
		"total":     total,
	})
}

// CreateTaskCommentRequest represents a create comment request for tasks
type CreateTaskCommentRequest struct {
	Content string `json:"content" validate:"required"`
}

// CreateComment creates a comment on a task
func (h *TaskHandler) CreateComment(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	task, projectID, taskID, err := h.getTaskFromContext(c)
	if err != nil {
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	if _, err := h.projectService.RequireRole(task.ProjectID, user.UserName, model.ProjectRoleMember); err != nil {
		return respondError(c, http.StatusForbidden, "Only project members can comment")
	}

	var req CreateTaskCommentRequest
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid request body")
	}

	comment, err := h.taskService.AddComment(projectID, taskID, user.UserName, req.Content)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to create comment")
	}

	// Parse @mentions and send notifications (best-effort)
	h.notifyMentionedUsers(user, projectID, taskID, task.Title, req.Content)

	comment.ProjectSlug = encodeID(comment.ProjectID)
	return respondSuccess(c, http.StatusCreated, comment)
}

// GetComments retrieves all comments for a task
func (h *TaskHandler) GetComments(c *fiber.Ctx) error {
	task, projectID, taskID, err := h.getTaskFromContext(c)
	if err != nil {
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	if !checkProjectReadAccess(c, h.projectService, task.ProjectID) {
		return nil
	}

	comments, err := h.taskService.GetComments(projectID, taskID)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to get comments")
	}

	for i := range comments {
		comments[i].ProjectSlug = encodeID(comments[i].ProjectID)
	}
	return respondSuccess(c, http.StatusOK, comments)
}

// UpdateComment updates a task comment
func (h *TaskHandler) UpdateComment(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	commentID, err := strconv.Atoi(c.Params("commentId"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid comment ID")
	}

	comment, err := h.taskService.GetComment(commentID)
	if err != nil {
		return respondError(c, http.StatusNotFound, "Comment not found")
	}

	if _, err := h.projectService.RequireRole(comment.ProjectID, user.UserName, model.ProjectRoleMember); err != nil {
		return respondError(c, http.StatusForbidden, "Only project members can edit comments")
	}

	// Check if user is comment author
	if comment.AuthorName != user.UserName {
		return respondError(c, http.StatusForbidden, "Only comment author can update")
	}

	var req struct {
		Content string `json:"content" validate:"required"`
	}
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid request body")
	}

	comment.Content = req.Content
	if err := h.taskService.UpdateComment(comment); err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to update comment")
	}

	comment.ProjectSlug = encodeID(comment.ProjectID)
	return respondSuccess(c, http.StatusOK, comment)
}

// DeleteComment deletes a task comment
func (h *TaskHandler) DeleteComment(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	commentID, err := strconv.Atoi(c.Params("commentId"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid comment ID")
	}

	comment, err := h.taskService.GetComment(commentID)
	if err != nil {
		return respondError(c, http.StatusNotFound, "Comment not found")
	}

	if _, err := h.projectService.RequireRole(comment.ProjectID, user.UserName, model.ProjectRoleMember); err != nil {
		return respondError(c, http.StatusForbidden, "Only project members can delete comments")
	}

	// Check if user is comment author
	if comment.AuthorName != user.UserName {
		return respondError(c, http.StatusForbidden, "Only comment author can delete")
	}

	if err := h.taskService.DeleteComment(commentID); err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to delete comment")
	}

	return c.SendStatus(http.StatusNoContent)
}

// CreateWorkLogRequest represents a create/update work log request
type CreateWorkLogRequest struct {
	Hours       float64 `json:"hours" validate:"required"`
	Description string  `json:"description"`
}

// CreateWorkLog creates a work log entry on a task (time tracking)
func (h *TaskHandler) CreateWorkLog(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	task, projectID, taskID, err := h.getTaskFromContext(c)
	if err != nil {
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	if _, err := h.projectService.RequireRole(task.ProjectID, user.UserName, model.ProjectRoleMember); err != nil {
		return respondError(c, http.StatusForbidden, "Only project members can log work")
	}

	var req CreateWorkLogRequest
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid request body")
	}

	if req.Hours <= 0 {
		return respondError(c, http.StatusBadRequest, "Hours must be greater than 0")
	}

	log, err := h.taskService.AddWorkLog(projectID, taskID, user.UserName, req.Hours, req.Description)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to create work log")
	}

	log.ProjectSlug = encodeID(log.ProjectID)
	return respondSuccess(c, http.StatusCreated, log)
}

// GetWorkLogs retrieves all work logs for a task
func (h *TaskHandler) GetWorkLogs(c *fiber.Ctx) error {
	task, projectID, taskID, err := h.getTaskFromContext(c)
	if err != nil {
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	if !checkProjectReadAccess(c, h.projectService, task.ProjectID) {
		return nil
	}

	logs, err := h.taskService.GetWorkLogs(projectID, taskID)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to get work logs")
	}
	if logs == nil {
		logs = []*model.TaskWorkLog{}
	}

	actualHours, _ := h.taskService.GetActualHours(projectID, taskID)
	for i := range logs {
		logs[i].ProjectSlug = encodeID(logs[i].ProjectID)
	}

	return respondSuccess(c, http.StatusOK, fiber.Map{
		"workLogs":    logs,
		"actualHours": actualHours,
	})
}

// UpdateWorkLog updates a work log entry
func (h *TaskHandler) UpdateWorkLog(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	logID, err := strconv.ParseInt(c.Params("logId"), 10, 64)
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid log ID")
	}

	log, err := h.taskService.GetWorkLog(logID)
	if err != nil {
		return respondError(c, http.StatusNotFound, "Work log not found")
	}

	if _, err := h.projectService.RequireRole(log.ProjectID, user.UserName, model.ProjectRoleMember); err != nil {
		return respondError(c, http.StatusForbidden, "Only project members can edit work logs")
	}

	// Check if user is log author
	if log.UserName != user.UserName {
		return respondError(c, http.StatusForbidden, "Only log author can update")
	}

	var req CreateWorkLogRequest
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid request body")
	}

	if req.Hours <= 0 {
		return respondError(c, http.StatusBadRequest, "Hours must be greater than 0")
	}

	log.Hours = req.Hours
	log.Description = req.Description
	if err := h.taskService.UpdateWorkLog(log); err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to update work log")
	}

	log.ProjectSlug = encodeID(log.ProjectID)
	return respondSuccess(c, http.StatusOK, log)
}

// DeleteWorkLog deletes a work log entry
func (h *TaskHandler) DeleteWorkLog(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	logID, err := strconv.ParseInt(c.Params("logId"), 10, 64)
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid log ID")
	}

	log, err := h.taskService.GetWorkLog(logID)
	if err != nil {
		return respondError(c, http.StatusNotFound, "Work log not found")
	}

	if _, err := h.projectService.RequireRole(log.ProjectID, user.UserName, model.ProjectRoleMember); err != nil {
		return respondError(c, http.StatusForbidden, "Only project members can delete work logs")
	}

	// Check if user is log author
	if log.UserName != user.UserName {
		return respondError(c, http.StatusForbidden, "Only log author can delete")
	}

	if err := h.taskService.DeleteWorkLog(logID); err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to delete work log")
	}

	return c.SendStatus(http.StatusNoContent)
}

// AddLabel adds a label to a task
func (h *TaskHandler) AddLabel(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	task, projectID, taskID, err := h.getTaskFromContext(c)
	if err != nil {
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	if _, err := h.projectService.RequireRole(task.ProjectID, user.UserName, model.ProjectRoleMember); err != nil {
		return respondError(c, http.StatusForbidden, "Only project members can manage labels")
	}

	var req struct {
		LabelID int `json:"labelId" validate:"required"`
	}
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid request body")
	}

	if err := h.taskService.AddLabel(projectID, taskID, req.LabelID); err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to add label")
	}

	return c.SendStatus(http.StatusCreated)
}

// RemoveLabel removes a label from a task
func (h *TaskHandler) RemoveLabel(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	task, projectID, taskID, err := h.getTaskFromContext(c)
	if err != nil {
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	if _, err := h.projectService.RequireRole(task.ProjectID, user.UserName, model.ProjectRoleMember); err != nil {
		return respondError(c, http.StatusForbidden, "Only project members can manage labels")
	}

	labelID, err := strconv.Atoi(c.Params("labelId"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid label ID")
	}

	if err := h.taskService.RemoveLabel(projectID, taskID, labelID); err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to remove label")
	}

	return c.SendStatus(http.StatusNoContent)
}

// GetLabels retrieves all labels for a task
func (h *TaskHandler) GetLabels(c *fiber.Ctx) error {
	task, projectID, taskID, err := h.getTaskFromContext(c)
	if err != nil {
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	if !checkProjectReadAccess(c, h.projectService, task.ProjectID) {
		return nil
	}

	labels, err := h.taskService.GetLabels(projectID, taskID)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to get labels")
	}

	for i := range labels {
		labels[i].ProjectSlug = encodeID(labels[i].ProjectID)
	}
	return respondSuccess(c, http.StatusOK, labels)
}

// CreateLabel creates a new task label
func (h *TaskHandler) CreateLabel(c *fiber.Ctx) error {
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
		return respondError(c, http.StatusForbidden, "Only project members can create labels")
	}

	var req struct {
		Name  string `json:"name" validate:"required"`
		Color string `json:"color" validate:"required"`
	}
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid request body")
	}

	label, err := h.taskService.CreateLabel(projectID, req.Name, req.Color)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to create label")
	}

	return respondSuccess(c, http.StatusCreated, label)
}

// GetProjectLabels retrieves all labels for a project
func (h *TaskHandler) GetProjectLabels(c *fiber.Ctx) error {
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

	labels, err := h.taskService.GetProjectLabels(projectID)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to get labels")
	}

	for i := range labels {
		labels[i].ProjectSlug = encodeID(labels[i].ProjectID)
	}
	return respondSuccess(c, http.StatusOK, labels)
}

// UpdatePosition updates task position (for kanban board)
func (h *TaskHandler) UpdatePosition(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	_, projectID, taskID, err := h.getTaskFromContext(c)
	if err != nil {
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	var req struct {
		Position int `json:"position" validate:"required"`
	}
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid request body")
	}

	if err := h.taskService.UpdatePosition(projectID, taskID, req.Position); err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to update position")
	}

	return c.JSON(fiber.Map{"message": "OK"})
}

// GetSubtasks retrieves all subtasks of a task
func (h *TaskHandler) GetSubtasks(c *fiber.Ctx) error {
	task, projectID, taskID, err := h.getTaskFromContext(c)
	if err != nil {
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	if !checkProjectReadAccess(c, h.projectService, task.ProjectID) {
		return nil
	}

	tasks, err := h.taskService.GetSubtasks(projectID, taskID)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to get subtasks")
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
	return respondSuccess(c, http.StatusOK, tasks)
}

// ListAssignees lists all assignees for a task
func (h *TaskHandler) ListAssignees(c *fiber.Ctx) error {
	_, projectID, taskID, err := h.getTaskFromContext(c)
	if err != nil {
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	assignees, err := h.taskService.GetAssignees(projectID, taskID)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to get assignees")
	}

	if assignees == nil {
		assignees = []*model.Account{}
	}

	return respondSuccess(c, http.StatusOK, fiber.Map{"assignees": assignees})
}

// AddAssignee assigns a user to a task
func (h *TaskHandler) AddAssignee(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	_, projectID, taskID, err := h.getTaskFromContext(c)
	if err != nil {
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	username := c.Params("username")
	if username == "" {
		return respondError(c, http.StatusBadRequest, "Username is required")
	}

	if err := h.taskService.AddAssignee(projectID, taskID, username); err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to add assignee")
	}

	assignees, _ := h.taskService.GetAssignees(projectID, taskID)
	c.Status(http.StatusOK).JSON(fiber.Map{"assignees": assignees})
	return nil
}

// RemoveAssignee removes an assignee from a task
func (h *TaskHandler) RemoveAssignee(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	_, projectID, taskID, err := h.getTaskFromContext(c)
	if err != nil {
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	username := c.Params("username")
	if username == "" {
		return respondError(c, http.StatusBadRequest, "Username is required")
	}

	if err := h.taskService.RemoveAssignee(projectID, taskID, username); err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to remove assignee")
	}

	assignees, _ := h.taskService.GetAssignees(projectID, taskID)
	c.Status(http.StatusOK).JSON(fiber.Map{"assignees": assignees})
	return nil
}

// GetStatusHistory retrieves the status change history for a task
func (h *TaskHandler) GetStatusHistory(c *fiber.Ctx) error {
	_, projectID, taskID, err := h.getTaskFromContext(c)
	if err != nil {
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	history, err := h.taskService.GetStatusHistory(projectID, taskID)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to get status history")
	}

	for i := range history {
		history[i].ProjectSlug = encodeID(history[i].ProjectID)
	}
	return respondSuccess(c, http.StatusOK, history)
}

// -----------------------------------------------------------------------------
// Task-Branch association handlers
// -----------------------------------------------------------------------------

// parseRepoFullName splits a "owner/repo" string into owner and repo.
// Returns ok=false if the format is invalid.
func parseRepoFullName(repoFullName string) (owner, repo string, ok bool) {
	parts := strings.SplitN(repoFullName, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || strings.Contains(parts[1], "/") {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// AssociateBranchRequest links an existing branch to a task.
type AssociateBranchRequest struct {
	RepoFullName string `json:"repoFullName" validate:"required"`
	BranchName   string `json:"branchName" validate:"required"`
}

// CreateAndAssociateBranchRequest creates a new branch in a repo and links it to a task.
type CreateAndAssociateBranchRequest struct {
	RepoFullName string `json:"repoFullName" validate:"required"`
	BranchName   string `json:"branchName" validate:"required"`
	From         string `json:"from"` // base branch; empty → repo default branch
}

// ListTaskBranches lists all branches associated with a task.
func (h *TaskHandler) ListTaskBranches(c *fiber.Ctx) error {
	task, projectID, taskID, err := h.getTaskFromContext(c)
	if err != nil {
		if err == service.ErrTaskNotFound {
			return respondError(c, http.StatusNotFound, "Task not found")
		}
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	if !checkProjectReadAccess(c, h.projectService, task.ProjectID) {
		return nil
	}

	branches, err := h.taskService.ListTaskBranches(projectID, taskID)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to list task branches")
	}
	if branches == nil {
		branches = []*model.TaskBranch{}
	}
	for i := range branches {
		branches[i].ProjectSlug = encodeID(branches[i].ProjectID)
	}
	return respondSuccess(c, http.StatusOK, fiber.Map{"branches": branches})
}

// GetTaskCommits lists commits on branches associated with a task.
//
// For each branch, returns commits unique to that branch — i.e. commits made
// since the branch diverged from the repo's default branch (its merge base).
// This shows only the commits unique to the branch: for a branch created from the
// default branch, the merge base is exactly the branch creation point, so only
// commits made on the task branch are shown. Up to 50 commits per branch.
//
// Used by the PMS task drawer to display the commit log for the task's
// associated branch.
func (h *TaskHandler) GetTaskCommits(c *fiber.Ctx) error {
	task, projectID, taskID, err := h.getTaskFromContext(c)
	if err != nil {
		if err == service.ErrTaskNotFound {
			return respondError(c, http.StatusNotFound, "Task not found")
		}
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	if !checkProjectReadAccess(c, h.projectService, task.ProjectID) {
		return nil
	}

	branches, err := h.taskService.ListTaskBranches(projectID, taskID)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to list task branches")
	}

	type branchCommits struct {
		RepoFullName string               `json:"repoFullName"`
		BranchName   string               `json:"branchName"`
		Commits      []git.PushCommitInfo `json:"commits"`
	}

	result := make([]branchCommits, 0, len(branches))
	for _, tb := range branches {
		// Parse "owner/repo" from RepoFullName
		owner, repoName, ok := parseRepoFullName(tb.RepoFullName)
		if !ok {
			continue
		}

		// Get the repo's default branch for comparison
		repo, err := h.repoService.GetRepository(owner, repoName)
		if err != nil {
			continue
		}
		defaultBranch := repo.DefaultBranch
		if defaultBranch == "" {
			defaultBranch = "main"
		}

		// Resolve SHAs for default branch and task branch
		defaultSHA, err := h.gitClient.ResolveRef(owner, repoName, defaultBranch)
		if err != nil {
			continue
		}
		branchSHA, err := h.gitClient.ResolveRef(owner, repoName, tb.BranchName)
		if err != nil {
			continue
		}

		// Skip if branch is at the same commit as default (no unique commits)
		if defaultSHA == branchSHA {
			result = append(result, branchCommits{
				RepoFullName: tb.RepoFullName,
				BranchName:   tb.BranchName,
				Commits:      []git.PushCommitInfo{},
			})
			continue
		}

		// Determine base SHA for listing branch-unique commits with 3-level fallback
		baseSHA := tb.BaseSHA
		if baseSHA == "" || baseSHA == branchSHA {
			mb, mbErr := h.gitClient.MergeBase(owner, repoName, defaultSHA, branchSHA)
			if mbErr == nil {
				baseSHA = mb
			} else {
				baseSHA = defaultSHA
			}
		}

		if baseSHA == branchSHA {
			// Post-merge collapse: fallback to listing recent commits filtered by creation time
			allCommits, lcErr := h.gitClient.ListCommitsBetween(
				owner, repoName,
				"0000000000000000000000000000000000000000",
				branchSHA, 50,
			)
			if lcErr != nil {
				allCommits = []git.PushCommitInfo{}
			}
			filtered := make([]git.PushCommitInfo, 0, len(allCommits))
			for _, cm := range allCommits {
				if !cm.Time.Before(tb.CreatedAt) {
					filtered = append(filtered, cm)
				}
			}
			result = append(result, branchCommits{
				RepoFullName: tb.RepoFullName,
				BranchName:   tb.BranchName,
				Commits:      filtered,
			})
			continue
		}

		// List commits unique to the task branch (ahead of base)
		commits, err := h.gitClient.ListCommitsBetween(owner, repoName, baseSHA, branchSHA, 50)
		if err != nil {
			commits = []git.PushCommitInfo{}
		}
		result = append(result, branchCommits{
			RepoFullName: tb.RepoFullName,
			BranchName:   tb.BranchName,
			Commits:      commits,
		})
	}

	return respondSuccess(c, http.StatusOK, fiber.Map{"branches": result})
}

// AssociateBranch links an existing branch to a task.
// The branch must exist in the repository and the user must have read access to it.
func (h *TaskHandler) AssociateBranch(c *fiber.Ctx) error {
	task, projectID, taskID, err := h.getTaskFromContext(c)
	if err != nil {
		if err == service.ErrTaskNotFound {
			return respondError(c, http.StatusNotFound, "Task not found")
		}
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	user, ok := checkProjectWriteAccess(c, h.projectService, task.ProjectID)
	if !ok {
		return nil
	}

	var req AssociateBranchRequest
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid request body")
	}
	if req.RepoFullName == "" || req.BranchName == "" {
		return respondError(c, http.StatusBadRequest, "repoFullName and branchName are required")
	}

	owner, repo, ok := parseRepoFullName(req.RepoFullName)
	if !ok {
		return respondError(c, http.StatusBadRequest, "Invalid repoFullName, expected 'owner/repo'")
	}

	// Verify repository exists and user has read access (prevent linking branches from unauthorized private repos)
	repository, err := h.repoService.GetRepository(owner, repo)
	if err != nil {
		return respondError(c, http.StatusNotFound, "Repository not found")
	}
	if !h.repoService.HasViewerRole(repository, user) {
		return respondError(c, http.StatusForbidden, "You do not have access to this repository")
	}

	// Verify branch actually exists (avoid storing invalid branch names)
	branches, err := h.gitClient.ListBranches(owner, repo, repository.DefaultBranch)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to list repository branches")
	}
	branchExists := false
	for _, b := range branches {
		if b.Name == req.BranchName {
			branchExists = true
			break
		}
	}
	if !branchExists {
		return respondError(c, http.StatusBadRequest, "Branch does not exist in this repository")
	}

	// Record base SHA for GetTaskCommits. When linking an existing branch, the fork point is the merge base
	// with the default branch; if it cannot be calculated, fallback to the default branch's current HEAD.
	var baseSHA string
	if defaultSHA, err := h.gitClient.ResolveRef(owner, repo, repository.DefaultBranch); err == nil {
		baseSHA = defaultSHA
		if branchSHA, err := h.gitClient.ResolveRef(owner, repo, req.BranchName); err == nil {
			if mb, err := h.gitClient.MergeBase(owner, repo, defaultSHA, branchSHA); err == nil {
				baseSHA = mb
			}
		}
	}

	tb, err := h.taskService.AddTaskBranch(projectID, taskID, req.RepoFullName, req.BranchName, baseSHA, user.UserName)
	if err != nil {
		if err == service.ErrTaskBranchAlreadyLinked {
			return respondError(c, http.StatusConflict, "Branch already linked to this task")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to link branch")
	}
	tb.ProjectSlug = encodeID(tb.ProjectID)

	if h.activityService != nil {
		_ = h.activityService.LogActivity(user.UserName, task.ProjectID, "task", task.TaskID,
			"branch_linked", "", "", fmt.Sprintf("%s:%s", req.RepoFullName, req.BranchName))
	}
	return respondSuccess(c, http.StatusCreated, tb)
}

// CreateAndAssociateBranch creates a new branch in a repository and links it to a task.
// Requires write access to the repository (developer role).
func (h *TaskHandler) CreateAndAssociateBranch(c *fiber.Ctx) error {
	task, projectID, taskID, err := h.getTaskFromContext(c)
	if err != nil {
		if err == service.ErrTaskNotFound {
			return respondError(c, http.StatusNotFound, "Task not found")
		}
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	user, ok := checkProjectWriteAccess(c, h.projectService, task.ProjectID)
	if !ok {
		return nil
	}

	var req CreateAndAssociateBranchRequest
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid request body")
	}
	if req.RepoFullName == "" || req.BranchName == "" {
		return respondError(c, http.StatusBadRequest, "repoFullName and branchName are required")
	}

	owner, repo, ok := parseRepoFullName(req.RepoFullName)
	if !ok {
		return respondError(c, http.StatusBadRequest, "Invalid repoFullName, expected 'owner/repo'")
	}

	// Creating a branch requires repository write access (developer+)
	repository, err := h.repoService.GetRepository(owner, repo)
	if err != nil {
		return respondError(c, http.StatusNotFound, "Repository not found")
	}
	if !h.repoService.HasMemberRole(repository, user) {
		return respondError(c, http.StatusForbidden, "Write access to repository required to create a branch")
	}

	// If no base branch specified, use repository default branch
	fromBranch := req.From
	if fromBranch == "" {
		fromBranch = repository.DefaultBranch
	}

	// If target branch already exists, link it directly (avoid duplicate creation errors)
	existing, _ := h.gitClient.ListBranches(owner, repo, repository.DefaultBranch)
	alreadyExists := false
	if existing != nil {
		for _, b := range existing {
			if b.Name == req.BranchName {
				alreadyExists = true
				break
			}
		}
	}
	if !alreadyExists {
		if err := h.gitClient.CreateBranch(owner, repo, req.BranchName, fromBranch); err != nil {
			return respondError(c, http.StatusInternalServerError, "Failed to create branch: "+err.Error())
		}
	}

	// Record base SHA (fromBranch HEAD at creation time) for GetTaskCommits.
	// This fixed BaseSHA is immune to MR merge effects that would otherwise collapse the merge base.
	baseSHA, _ := h.gitClient.ResolveRef(owner, repo, fromBranch)

	tb, err := h.taskService.AddTaskBranch(projectID, taskID, req.RepoFullName, req.BranchName, baseSHA, user.UserName)
	if err != nil {
		if err == service.ErrTaskBranchAlreadyLinked {
			return respondError(c, http.StatusConflict, "Branch already linked to this task")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to link branch")
	}
	tb.ProjectSlug = encodeID(tb.ProjectID)

	// Publish event: subscribers handle side effects asynchronously (Scrum activity log + assignee notification)
	// Keeps core transaction synchronous while decoupling side effects.
	if h.eventBus != nil {
		h.eventBus.Publish(event.NewTaskBranchCreatedEvent(
			user, task.ProjectID, task.TaskID, task.Title, req.RepoFullName, req.BranchName,
		))
	}
	return respondSuccess(c, http.StatusCreated, tb)
}

// RemoveTaskBranch unlinks a branch from a task (does NOT delete the git branch).
func (h *TaskHandler) RemoveTaskBranch(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	task, projectID, taskID, err := h.getTaskFromContext(c)
	if err != nil {
		if err == service.ErrTaskNotFound {
			return respondError(c, http.StatusNotFound, "Task not found")
		}
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	// Write access: member role or above
	if _, err := h.projectService.RequireRole(task.ProjectID, user.UserName, model.ProjectRoleMember); err != nil {
		return respondError(c, http.StatusForbidden, "Only project members can unlink branches")
	}

	branchID, err := strconv.Atoi(c.Params("branchId"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid branch id")
	}

	if err := h.taskService.RemoveTaskBranch(projectID, taskID, branchID); err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to unlink branch")
	}

	if h.activityService != nil {
		_ = h.activityService.LogActivity(user.UserName, task.ProjectID, "task", task.TaskID,
			"branch_unlinked", "", "", strconv.Itoa(branchID))
	}
	return c.SendStatus(http.StatusNoContent)
}

// --- Task Dependency handlers ---

// AddDependencyRequest represents a request to create a dependency: task depends on dependsOnTaskId.
type AddDependencyRequest struct {
	DependsOnTaskID int    `json:"dependsOnTaskId" validate:"required"`
	DependencyType  string `json:"dependencyType"`
}

// ListDependencies lists the predecessor dependencies of a task (A depends on B → returns B).
func (h *TaskHandler) ListDependencies(c *fiber.Ctx) error {
	task, projectID, taskID, err := h.getTaskFromContext(c)
	if err != nil {
		if err == service.ErrTaskNotFound {
			return respondError(c, http.StatusNotFound, "Task not found")
		}
		return respondError(c, http.StatusBadRequest, err.Error())
	}
	if !checkProjectReadAccess(c, h.projectService, task.ProjectID) {
		return nil
	}
	deps, err := h.taskService.ListDependencies(projectID, taskID)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to list dependencies")
	}
	if deps == nil {
		deps = []service.DependencyTask{}
	}
	return respondSuccess(c, http.StatusOK, fiber.Map{"dependencies": deps})
}

// ListBlocks lists the successor tasks that depend on the current task (who is blocked by this task).
func (h *TaskHandler) ListBlocks(c *fiber.Ctx) error {
	task, projectID, taskID, err := h.getTaskFromContext(c)
	if err != nil {
		if err == service.ErrTaskNotFound {
			return respondError(c, http.StatusNotFound, "Task not found")
		}
		return respondError(c, http.StatusBadRequest, err.Error())
	}
	if !checkProjectReadAccess(c, h.projectService, task.ProjectID) {
		return nil
	}
	blocks, err := h.taskService.ListBlocks(projectID, taskID)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to list blocks")
	}
	if blocks == nil {
		blocks = []service.DependencyTask{}
	}
	return respondSuccess(c, http.StatusOK, fiber.Map{"blocks": blocks})
}

// AddDependency creates a directed dependency: task → dependsOnTaskId (A depends on B).
// Error codes: self→400, notfound→404, alreadyexists→409, circular→422.
func (h *TaskHandler) AddDependency(c *fiber.Ctx) error {
	task, projectID, taskID, err := h.getTaskFromContext(c)
	if err != nil {
		if err == service.ErrTaskNotFound {
			return respondError(c, http.StatusNotFound, "Task not found")
		}
		return respondError(c, http.StatusBadRequest, err.Error())
	}
	user, ok := checkProjectWriteAccess(c, h.projectService, task.ProjectID)
	if !ok {
		return nil
	}
	var req AddDependencyRequest
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid request body")
	}
	if req.DependsOnTaskID <= 0 {
		return respondError(c, http.StatusBadRequest, "dependsOnTaskId is required")
	}
	depType := req.DependencyType
	if depType == "" {
		depType = "fs"
	}
	if err := h.taskService.AddDependency(projectID, taskID, req.DependsOnTaskID, depType); err != nil {
		switch err {
		case service.ErrTaskSelfDependency:
			return respondError(c, http.StatusBadRequest, "Task cannot depend on itself")
		case service.ErrTaskNotFound:
			return respondError(c, http.StatusNotFound, "Dependency task not found")
		case service.ErrTaskDependencyAlreadyExists:
			return respondError(c, http.StatusConflict, "Dependency already exists")
		case service.ErrCircularDependency:
			return respondError(c, http.StatusUnprocessableEntity, "Circular dependency detected")
		case service.ErrInvalidDependencyType:
			return respondError(c, http.StatusBadRequest, "Invalid dependency type")
		default:
			return respondError(c, http.StatusInternalServerError, "Failed to add dependency")
		}
	}
	if h.activityService != nil {
		_ = h.activityService.LogActivity(user.UserName, task.ProjectID, "task", task.TaskID,
			"dependency_added", "", "", strconv.Itoa(req.DependsOnTaskID))
	}
	return respondSuccess(c, http.StatusCreated, fiber.Map{
		"taskId":          taskID,
		"dependsOnTaskId": req.DependsOnTaskID,
		"dependencyType":  depType,
	})
}

// RemoveDependency removes a directed dependency: task → dependsOnTaskId.
func (h *TaskHandler) RemoveDependency(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}
	task, projectID, taskID, err := h.getTaskFromContext(c)
	if err != nil {
		if err == service.ErrTaskNotFound {
			return respondError(c, http.StatusNotFound, "Task not found")
		}
		return respondError(c, http.StatusBadRequest, err.Error())
	}
	if _, err := h.projectService.RequireRole(task.ProjectID, user.UserName, model.ProjectRoleMember); err != nil {
		return respondError(c, http.StatusForbidden, "Only project members can remove dependencies")
	}
	dependsOnTaskID, err := strconv.Atoi(c.Params("dependsOnTaskId"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid dependsOnTaskId")
	}
	if err := h.taskService.RemoveDependency(projectID, taskID, dependsOnTaskID); err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to remove dependency")
	}
	if h.activityService != nil {
		_ = h.activityService.LogActivity(user.UserName, task.ProjectID, "task", task.TaskID,
			"dependency_removed", "", "", strconv.Itoa(dependsOnTaskID))
	}
	return c.SendStatus(http.StatusNoContent)
}

// AIOptimizeTask optimizes a task draft using LLM via SSE streaming.
// Accepts the current draft (title/description) in the request body and returns
// optimized content + suggested priority/taskType/storyPoints.
// Mirrors AIOptimizeUserStory but tailored for task structure (no acceptanceCriteria, has taskType).
func (h *TaskHandler) AIOptimizeTask(c *fiber.Ctx) error {
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

	// Parse request body: current draft content (tasks only have title/description, no acceptanceCriteria)
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid request body")
	}
	if req.Title == "" {
		return respondError(c, http.StatusBadRequest, "Title is required")
	}

	lang := c.Query("lang", "en")

	// Set SSE response headers
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

		result, err := h.aiService.OptimizeTaskStream(
			req.Title, req.Description, lang,
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
