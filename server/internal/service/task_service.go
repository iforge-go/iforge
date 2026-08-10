package service

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

var (
	ErrTaskNotFound = errors.New("task not found")
)

// EpicInfo represents summary info of Epic associated with a Task (indirectly via user_story)
// Used to populate epic_slug/epic_title in taskListItem for displaying Epic color tags on kanban cards
type EpicInfo struct {
	EpicID    int
	EpicTitle string
}

// TaskService handles task-related operations
type TaskService struct {
	db *gorm.DB
}

// NewTaskService creates a new TaskService
func NewTaskService(db *gorm.DB) *TaskService {
	return &TaskService{
		db: db,
	}
}

// CreateTask creates a new task with project-scoped task_id via TaskIDCounter (mirrors CreateIssue)
func (s *TaskService) CreateTask(projectID int, title, status, priority, taskType string, description *string, storyPoints *int, estimatedHours *float64, assigneeName, reporterName *string, sprintID, userStoryID, parentID *int, position int, dueDate *time.Time) (*model.Task, error) {
	var createdTask *model.Task
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Acquire next task_id from counter (same pattern as IssueIDCounter)
		var counter model.TaskIDCounter
		err := tx.Where("project_id = ?", projectID).First(&counter).Error

		var taskID int
		if err == gorm.ErrRecordNotFound {
			taskID = 1
			counter = model.TaskIDCounter{
				ProjectID: projectID,
				TaskID:    1,
			}
			if err := tx.Create(&counter).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			taskID = counter.TaskID + 1
			if err := tx.Model(&counter).Update("task_id", taskID).Error; err != nil {
				return err
			}
		}

		now := time.Now()
		task := &model.Task{
			ProjectID:      projectID,
			TaskID:         taskID,
			SprintID:       sprintID,
			UserStoryID:    userStoryID,
			Title:          title,
			Description:    description,
			Status:         status,
			Priority:       priority,
			TaskType:       taskType,
			StoryPoints:    storyPoints,
			EstimatedHours: estimatedHours,
			AssigneeName:   assigneeName,
			ReporterName:   *reporterName,
			ParentID:       parentID,
			Position:       position,
			DueDate:        dueDate,
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		// Set root_id: root task points to itself; subtask inherits parent's root_id
		if parentID != nil {
			var parent model.Task
			if err := tx.Where("project_id = ? AND task_id = ?", projectID, *parentID).First(&parent).Error; err == nil && parent.RootID != nil {
				task.RootID = parent.RootID
			} else {
				task.RootID = parentID
			}
		} else {
			task.RootID = &taskID
		}

		if err := tx.Create(task).Error; err != nil {
			return err
		}

		createdTask = task
		return nil
	})
	if err != nil {
		return nil, err
	}
	return createdTask, nil
}

// GetTask retrieves a task by project ID and project-scoped task_id
func (s *TaskService) GetTask(projectID, taskID int) (*model.Task, error) {
	var task model.Task
	err := s.db.Where("project_id = ? AND task_id = ?", projectID, taskID).First(&task).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}
	return &task, nil
}

// UpdateTask updates a task
func (s *TaskService) UpdateTask(task *model.Task) error {
	task.UpdatedAt = time.Now()
	return s.db.Save(task).Error
}

// buildListTasksQuery builds the base WHERE query for listing/counting root tasks.
// Reuse same filter for ListTasks and CountTasks to ensure pagination total matches list data exactly.
func (s *TaskService) buildListTasksQuery(projectID int, status, taskType *string, sprintID, userStoryID *int, q *string) *gorm.DB {
	query := s.db.Model(&model.Task{}).Where("project_id = ? AND parent_id IS NULL", projectID)

	if status != nil && *status != "" {
		query = query.Where("status = ?", *status)
	}
	if taskType != nil && *taskType != "" {
		query = query.Where("task_type = ?", *taskType)
	}
	if sprintID != nil {
		// Match tasks directly assigned to the sprint (task.sprint_id)
		// or tasks whose parent Story belongs to the sprint (story.sprint_id, inherited during story-level planning).
		query = query.Where("sprint_id = ? OR user_story_id IN (SELECT id FROM user_story WHERE sprint_id = ?)", *sprintID, *sprintID)
	}
	if userStoryID != nil {
		query = query.Where("user_story_id = ?", *userStoryID)
	}
	// Title fuzzy search + ID exact match: search by task_id when input is numeric
	if q != nil && *q != "" {
		qLower := strings.ToLower(*q)
		if id, err := strconv.Atoi(*q); err == nil {
			query = query.Where("LOWER(title) LIKE ? OR task_id = ?", "%"+qLower+"%", id)
		} else {
			query = query.Where("LOWER(title) LIKE ?", "%"+qLower+"%")
		}
	}

	return query
}

// ListTasks lists tasks for a project (root tasks only; subtasks are excluded)
func (s *TaskService) ListTasks(projectID int, status, taskType *string, sprintID, userStoryID *int, q *string, limit, offset int) ([]*model.Task, error) {
	var tasks []*model.Task
	query := s.buildListTasksQuery(projectID, status, taskType, sprintID, userStoryID, q)

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Order("position ASC, created_at DESC").Find(&tasks).Error
	return tasks, err
}

// CountTasks counts total root tasks matching the same filter as ListTasks.
// Used for pagination total returned by backend; frontend calculates total pages based on this.
// Shares buildListTasksQuery with ListTasks to ensure consistency.
func (s *TaskService) CountTasks(projectID int, status, taskType *string, sprintID, userStoryID *int, q *string) (int64, error) {
	var count int64
	query := s.buildListTasksQuery(projectID, status, taskType, sprintID, userStoryID, q)
	err := query.Count(&count).Error
	return count, err
}

// GetSubtaskCounts returns total subtask count and completed subtask count for each task
func (s *TaskService) GetSubtaskCounts(projectID int, taskIDs []int) (map[int]int, map[int]int, error) {
	totalMap := make(map[int]int)
	completedMap := make(map[int]int)
	if len(taskIDs) == 0 {
		return totalMap, completedMap, nil
	}

	type countResult struct {
		ParentID int `gorm:"column:parent_id"`
		Total    int `gorm:"column:total"`
		Done     int `gorm:"column:done"`
	}

	var results []countResult
	// LEFT JOIN task_status:use is_closed for configured projects; fallback to hardcoded slug list
	// fallback to hardcoded slug list (consistent with inferCategorySlug).
	// Consistent with GetBlockingTasks/BatchGetBlockedByCounts JOIN pattern.
	err := s.db.Raw(`
		SELECT t.parent_id, COUNT(*) as total,
		       SUM(CASE WHEN COALESCE(ts.is_closed, 0) = 1
		                   OR t.status IN ('done', 'completed', 'closed', 'archived')
		                THEN 1 ELSE 0 END) as done
		FROM task t
		LEFT JOIN task_status ts ON ts.project_id = t.project_id AND ts.slug = t.status
		WHERE t.project_id = ? AND t.parent_id IN ?
		GROUP BY t.parent_id`, projectID, taskIDs).Scan(&results).Error
	if err != nil {
		return nil, nil, err
	}

	for _, r := range results {
		totalMap[r.ParentID] = r.Total
		completedMap[r.ParentID] = r.Done
	}
	return totalMap, completedMap, nil
}

// GetTaskEpics batch queries Epic info associated with tasks (task -> user_story -> epic three-table JOIN)
// Returns map[taskID]EpicInfo; tasks without associated Epic are not in map (frontend handles as missing)
// Used to display Epic color tags on kanban task cards, enhancing Epic visibility on daily pages
func (s *TaskService) GetTaskEpics(projectID int, taskIDs []int) (map[int]EpicInfo, error) {
	result := make(map[int]EpicInfo)
	if len(taskIDs) == 0 {
		return result, nil
	}

	type row struct {
		TaskID    int    `gorm:"column:task_id"`
		EpicID    int    `gorm:"column:epic_id"`
		EpicTitle string `gorm:"column:epic_title"`
	}

	var rows []row
	// task JOIN user_story JOIN epic: only returns tasks with associated Epic
	err := s.db.Raw(`
		SELECT t.task_id, e.id as epic_id, e.title as epic_title
		FROM task t
		JOIN user_story us ON t.user_story_id = us.id
		JOIN epic e ON us.epic_id = e.id
		WHERE t.project_id = ? AND t.task_id IN ?`,
		projectID, taskIDs).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, r := range rows {
		result[r.TaskID] = EpicInfo{
			EpicID:    r.EpicID,
			EpicTitle: r.EpicTitle,
		}
	}
	return result, nil
}

// UserStoryInfo represents summary info of UserStory associated with Task
// Used to populate userStoryTitle in taskListItem for displaying story badge on kanban cards
type UserStoryInfo struct {
	UserStoryID    int
	UserStoryTitle string
}

// GetTaskUserStories batch queries UserStory info associated with tasks (task → user_story two-table JOIN)
// Returns map[taskID]UserStoryInfo; tasks without associated Story are not in map (frontend handles as missing)
// Used to display story identifier on kanban task cards, enhancing Story visibility on daily pages
// Note: task.UserStorySlug is already populated in ListTasks handler using encodeID; this only fills title
func (s *TaskService) GetTaskUserStories(projectID int, taskIDs []int) (map[int]UserStoryInfo, error) {
	result := make(map[int]UserStoryInfo)
	if len(taskIDs) == 0 {
		return result, nil
	}

	type row struct {
		TaskID         int    `gorm:"column:task_id"`
		UserStoryID    int    `gorm:"column:user_story_id"`
		UserStoryTitle string `gorm:"column:user_story_title"`
	}

	var rows []row
	// task JOIN user_story: only returns tasks with associated Story
	err := s.db.Raw(`
		SELECT t.task_id, us.id as user_story_id, us.title as user_story_title
		FROM task t
		JOIN user_story us ON t.user_story_id = us.id
		WHERE t.project_id = ? AND t.task_id IN ?`,
		projectID, taskIDs).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, r := range rows {
		result[r.TaskID] = UserStoryInfo{
			UserStoryID:    r.UserStoryID,
			UserStoryTitle: r.UserStoryTitle,
		}
	}
	return result, nil
}

// DeleteTask deletes a task by (projectID, taskID)
func (s *TaskService) DeleteTask(projectID, taskID int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Delete task comments
		if err := tx.Where("project_id = ? AND task_id = ?", projectID, taskID).Delete(&model.TaskComment{}).Error; err != nil {
			return err
		}

		// Delete task label assignments
		if err := tx.Where("project_id = ? AND task_id = ?", projectID, taskID).Delete(&model.TaskLabelAssignment{}).Error; err != nil {
			return err
		}

		// Delete task assignees
		if err := tx.Where("project_id = ? AND task_id = ?", projectID, taskID).Delete(&model.TaskAssignment{}).Error; err != nil {
			return err
		}

		// Unlink subtasks
		if err := tx.Model(&model.Task{}).Where("project_id = ? AND parent_id = ?", projectID, taskID).Update("parent_id", nil).Error; err != nil {
			return err
		}

		// Delete task
		if err := tx.Where("project_id = ? AND task_id = ?", projectID, taskID).Delete(&model.Task{}).Error; err != nil {
			return err
		}

		return nil
	})
}

// UpdateStatus updates task status
func (s *TaskService) UpdateStatus(projectID, taskID int, status string, closedByName *string) error {
	// Workflow transition validation (aligned with Jira workflow transitions):
	// Query current task's oldStatus first, then check if oldStatus→status is allowed by project rules.
	// Backward compatible: unknown slug / empty rule set / oldStatus==newStatus all pass through, not breaking projects without configured workflows.
	var task model.Task
	if err := s.db.Select("status").Where("project_id = ? AND task_id = ?", projectID, taskID).First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrTaskNotFound
		}
		return err
	}
	if task.Status != status {
		allowed, err := s.IsStatusTransitionAllowed(projectID, task.Status, status)
		if err != nil {
			return err
		}
		if !allowed {
			return ErrInvalidTransition
		}
		// Hard block: before transitioning to in_progress-like status, verify prerequisites are completed.
		// CheckBlock triggers only when target status category==in_progress; returns *BlockedError for handler to convert to 422.
		if err := s.CheckBlock(projectID, taskID, status); err != nil {
			return err
		}
	}

	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	// Any status with isClosed=true is considered closed, set closed_at/closed_by_name
	// Configurable: query task_status.is_closed, fallback to isClosedStatusSlug (compatible with legacy slugs)
	isClosedStatus := s.IsClosedStatusByConfig(projectID, status)
	if isClosedStatus {
		now := time.Now()
		updates["closed_at"] = &now
		updates["closed_by_name"] = closedByName
	} else {
		updates["closed_at"] = nil
		updates["closed_by_name"] = nil
	}

	return s.db.Model(&model.Task{}).Where("project_id = ? AND task_id = ?", projectID, taskID).Updates(updates).Error
}

// IsStatusTransitionAllowed implements workflow validation by direct table lookup in TaskService,
// avoiding injection of TaskStatusService dependency (keeping constructor signature stable).
// Semantics consistent with TaskStatusService.IsTransitionAllowed (backward compatible):
//   - fromSlug or toSlug not found in task_status table (legacy slug): allow
//   - Project has no configured rules (empty table): allow
//   - Rules configured but (fromID→toID) doesn't match: deny
//
// Public so handler layer (UpdateTask status change) and service layer (UpdateStatus) share same validation logic.
func (s *TaskService) IsStatusTransitionAllowed(projectID int, fromSlug, toSlug string) (bool, error) {
	if fromSlug == toSlug {
		return true, nil
	}
	var from, to model.TaskStatus
	if err := s.db.Where("project_id = ? AND slug = ?", projectID, fromSlug).First(&from).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return true, nil
		}
		return false, err
	}
	if err := s.db.Where("project_id = ? AND slug = ?", projectID, toSlug).First(&to).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return true, nil
		}
		return false, err
	}
	var count int64
	if err := s.db.Model(&model.TaskStatusTransition{}).Where("project_id = ?", projectID).Count(&count).Error; err != nil {
		return false, err
	}
	if count == 0 {
		return true, nil
	}
	var matched int64
	if err := s.db.Model(&model.TaskStatusTransition{}).
		Where("project_id = ? AND from_status_id = ? AND to_status_id = ?", projectID, from.ID, to.ID).
		Count(&matched).Error; err != nil {
		return false, err
	}
	return matched > 0, nil
}

// UpdateStatusIfCurrent atomically transitions a task's status using optimistic
// locking: the update only applies when the current DB status matches
// expectedFrom. Returns true when the row was actually changed.
//
// This prevents duplicate activity records when two concurrent writers (e.g. the
// MR-merged subscriber and a manual PUT /tasks/:id) both observe the same
// pre-transition status and each emit a "status_changed" activity. The loser of
// the race sees RowsAffected == 0 and skips logging.
func (s *TaskService) UpdateStatusIfCurrent(projectID, taskID int, expectedFrom, status string, closedByName *string) (bool, error) {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	isClosedStatus := s.IsClosedStatusByConfig(projectID, status)
	if isClosedStatus {
		now := time.Now()
		updates["closed_at"] = &now
		updates["closed_by_name"] = closedByName
	} else {
		updates["closed_at"] = nil
		updates["closed_by_name"] = nil
	}

	res := s.db.Model(&model.Task{}).
		Where("project_id = ? AND task_id = ? AND status = ?", projectID, taskID, expectedFrom).
		Updates(updates)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// RecordStatusHistory records a status transition for audit/traceability.
// oldStatus may be empty when the task is first created with a non-default status.
//
// Deduplication: identical status transitions (oldStatus → newStatus) for the same task are only recorded once within a short window.
// Prevents concurrent races (e.g., MR merge subscriber and manual PUT /tasks/:id both triggering review→done) from creating duplicate entries.
// Atomic deduplication via existence check within transaction (SQLite writes are serialized; transaction ensures check+insert are not interleaved).
func (s *TaskService) RecordStatusHistory(projectID, taskID int, oldStatus, newStatus, changedByName string, comment *string) error {
	history := &model.TaskStatusHistory{
		ProjectID:     projectID,
		TaskID:        taskID,
		OldStatus:     oldStatus,
		NewStatus:     newStatus,
		ChangedByName: changedByName,
		Comment:       comment,
		CreatedAt:     time.Now(),
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Skip if identical (oldStatus → newStatus) record exists within 5-second window
		windowStart := history.CreatedAt.Add(-5 * time.Second)
		var count int64
		if err := tx.Model(&model.TaskStatusHistory{}).
			Where("project_id = ? AND task_id = ? AND old_status = ? AND new_status = ? AND created_at > ?",
				projectID, taskID, oldStatus, newStatus, windowStart).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
		return tx.Create(history).Error
	})
}

// GetStatusHistory retrieves the status change history for a task, newest first.
func (s *TaskService) GetStatusHistory(projectID, taskID int) ([]*model.TaskStatusHistory, error) {
	var history []*model.TaskStatusHistory
	err := s.db.Where("project_id = ? AND task_id = ?", projectID, taskID).
		Order("created_at DESC").Find(&history).Error
	return history, err
}

// GetSprintTitle returns the sprint title for a given sprint ID, fallback to "#id" if not found
func (s *TaskService) GetSprintTitle(sprintID int) string {
	var sprint model.Sprint
	if err := s.db.Select("title").First(&sprint, sprintID).Error; err != nil {
		return "#" + strconv.Itoa(sprintID)
	}
	return sprint.Title
}

// GetUserStoryTitle returns the user story title for a given story ID, fallback to "#id" if not found
func (s *TaskService) GetUserStoryTitle(storyID int) string {
	var story model.UserStory
	if err := s.db.Select("title").First(&story, storyID).Error; err != nil {
		return "#" + strconv.Itoa(storyID)
	}
	return story.Title
}

// AddComment adds a comment to a task
func (s *TaskService) AddComment(projectID, taskID int, authorName, content string) (*model.TaskComment, error) {
	now := time.Now()
	comment := &model.TaskComment{
		ProjectID:  projectID,
		TaskID:     taskID,
		AuthorName: authorName,
		Content:    content,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.db.Create(comment).Error; err != nil {
		return nil, err
	}

	// Update task updated_at
	if err := s.db.Model(&model.Task{}).Where("project_id = ? AND task_id = ?", projectID, taskID).Update("updated_at", now).Error; err != nil {
		return nil, err
	}

	return comment, nil
}

// GetComments retrieves all comments for a task
func (s *TaskService) GetComments(projectID, taskID int) ([]*model.TaskComment, error) {
	var comments []*model.TaskComment
	err := s.db.Where("project_id = ? AND task_id = ?", projectID, taskID).Order("created_at ASC").Find(&comments).Error
	return comments, err
}

// UpdateComment updates a task comment
func (s *TaskService) UpdateComment(comment *model.TaskComment) error {
	comment.UpdatedAt = time.Now()
	return s.db.Save(comment).Error
}

// DeleteComment deletes a task comment
func (s *TaskService) DeleteComment(commentID int) error {
	return s.db.Where("comment_id = ?", commentID).Delete(&model.TaskComment{}).Error
}

// GetComment retrieves a comment by ID
func (s *TaskService) GetComment(commentID int) (*model.TaskComment, error) {
	var comment model.TaskComment
	err := s.db.Where("comment_id = ?", commentID).First(&comment).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

// AddWorkLog adds a work log entry to a task (time tracking)
func (s *TaskService) AddWorkLog(projectID, taskID int, userName string, hours float64, description string) (*model.TaskWorkLog, error) {
	now := time.Now()
	log := &model.TaskWorkLog{
		ProjectID:   projectID,
		TaskID:      taskID,
		UserName:    userName,
		Hours:       hours,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.db.Create(log).Error; err != nil {
		return nil, err
	}
	// Update task updated_at
	if err := s.db.Model(&model.Task{}).Where("project_id = ? AND task_id = ?", projectID, taskID).Update("updated_at", now).Error; err != nil {
		return nil, err
	}
	return log, nil
}

// GetWorkLogs retrieves all work logs for a task
func (s *TaskService) GetWorkLogs(projectID, taskID int) ([]*model.TaskWorkLog, error) {
	var logs []*model.TaskWorkLog
	err := s.db.Where("project_id = ? AND task_id = ?", projectID, taskID).Order("created_at ASC").Find(&logs).Error
	return logs, err
}

// UpdateWorkLog updates a work log entry
func (s *TaskService) UpdateWorkLog(log *model.TaskWorkLog) error {
	log.UpdatedAt = time.Now()
	return s.db.Save(log).Error
}

// DeleteWorkLog deletes a work log entry
func (s *TaskService) DeleteWorkLog(logID int64) error {
	return s.db.Where("log_id = ?", logID).Delete(&model.TaskWorkLog{}).Error
}

// GetWorkLog retrieves a work log by ID
func (s *TaskService) GetWorkLog(logID int64) (*model.TaskWorkLog, error) {
	var log model.TaskWorkLog
	err := s.db.Where("log_id = ?", logID).First(&log).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// GetActualHours returns the sum of all work log hours for a task
func (s *TaskService) GetActualHours(projectID, taskID int) (float64, error) {
	var total float64
	err := s.db.Model(&model.TaskWorkLog{}).
		Where("project_id = ? AND task_id = ?", projectID, taskID).
		Select("COALESCE(SUM(hours), 0)").Scan(&total).Error
	return total, err
}

// AddLabel adds a label to a task
func (s *TaskService) AddLabel(projectID, taskID, labelID int) error {
	assignment := &model.TaskLabelAssignment{
		ProjectID: projectID,
		TaskID:    taskID,
		LabelID:   labelID,
	}
	return s.db.Create(assignment).Error
}

// RemoveLabel removes a label from a task
func (s *TaskService) RemoveLabel(projectID, taskID, labelID int) error {
	return s.db.Where("project_id = ? AND task_id = ? AND label_id = ?", projectID, taskID, labelID).
		Delete(&model.TaskLabelAssignment{}).Error
}

// GetLabels retrieves all labels for a task
func (s *TaskService) GetLabels(projectID, taskID int) ([]*model.TaskLabel, error) {
	var labels []*model.TaskLabel
	err := s.db.Joins("JOIN task_label_assignment ON task_label.id = task_label_assignment.label_id").
		Where("task_label_assignment.project_id = ? AND task_label_assignment.task_id = ?", projectID, taskID).
		Find(&labels).Error
	return labels, err
}

// CreateLabel creates a new task label
func (s *TaskService) CreateLabel(projectID int, name, color string) (*model.TaskLabel, error) {
	label := &model.TaskLabel{
		ProjectID: projectID,
		Name:      name,
		Color:     color,
	}
	if err := s.db.Create(label).Error; err != nil {
		return nil, err
	}
	return label, nil
}

// GetProjectLabels retrieves all labels for a project
func (s *TaskService) GetProjectLabels(projectID int) ([]*model.TaskLabel, error) {
	var labels []*model.TaskLabel
	err := s.db.Where("project_id = ?", projectID).Find(&labels).Error
	return labels, err
}

// UpdatePosition updates task position (for kanban board)
func (s *TaskService) UpdatePosition(projectID, taskID, position int) error {
	return s.db.Model(&model.Task{}).Where("project_id = ? AND task_id = ?", projectID, taskID).
		Updates(map[string]interface{}{
			"position":   position,
			"updated_at": time.Now(),
		}).Error
}

// GetSubtasks retrieves all subtasks of a task
func (s *TaskService) GetSubtasks(projectID, parentID int) ([]*model.Task, error) {
	var tasks []*model.Task
	err := s.db.Where("project_id = ? AND parent_id = ?", projectID, parentID).Order("position ASC, created_at DESC").Find(&tasks).Error
	return tasks, err
}

// AddAssignee assigns a user to a task
func (s *TaskService) AddAssignee(projectID, taskID int, assigneeUserName string) error {
	assignment := &model.TaskAssignment{
		ProjectID:        projectID,
		TaskID:           taskID,
		AssigneeUserName: assigneeUserName,
	}
	return s.db.Create(assignment).Error
}

// RemoveAssignee removes an assignee from a task
func (s *TaskService) RemoveAssignee(projectID, taskID int, assigneeUserName string) error {
	return s.db.Where("project_id = ? AND task_id = ? AND assignee_user_name = ?", projectID, taskID, assigneeUserName).
		Delete(&model.TaskAssignment{}).Error
}

// SetAssignees replaces all assignees of a task
func (s *TaskService) SetAssignees(projectID, taskID int, assigneeUserNames []string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("project_id = ? AND task_id = ?", projectID, taskID).
			Delete(&model.TaskAssignment{}).Error; err != nil {
			return err
		}
		for _, username := range assigneeUserNames {
			assignment := &model.TaskAssignment{
				ProjectID:        projectID,
				TaskID:           taskID,
				AssigneeUserName: username,
			}
			if err := tx.Create(assignment).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetAssignees retrieves all assignees (as Account) for a task
func (s *TaskService) GetAssignees(projectID, taskID int) ([]*model.Account, error) {
	var accounts []*model.Account
	err := s.db.Joins("JOIN task_assignment ON account.user_name = task_assignment.assignee_user_name").
		Where("task_assignment.project_id = ? AND task_assignment.task_id = ?", projectID, taskID).
		Find(&accounts).Error
	return accounts, err
}

// BatchGetAssignees fetches assignee usernames for multiple tasks in a single query (sideload).
// Returns a map of taskID -> []userName (references only; resolve via AccountService.BuildParticipants).
func (s *TaskService) BatchGetAssignees(projectID int, taskIDs []int) (map[int][]string, error) {
	result := make(map[int][]string)
	if len(taskIDs) == 0 {
		return result, nil
	}

	type taskAssignmentRow struct {
		TaskID           int    `gorm:"column:task_id"`
		AssigneeUserName string `gorm:"column:assignee_user_name"`
	}

	var rows []taskAssignmentRow
	// Use Model(&TaskAssignment{}) instead of Table("task_assignment") to go through model abstraction layer
	err := s.db.Model(&model.TaskAssignment{}).
		Select("task_assignment.task_id, task_assignment.assignee_user_name").
		Where("task_assignment.project_id = ? AND task_assignment.task_id IN ?", projectID, taskIDs).
		Scan(&rows).Error
	if err != nil {
		return result, err
	}

	for _, row := range rows {
		result[row.TaskID] = append(result[row.TaskID], row.AssigneeUserName)
	}
	return result, nil
}

// -----------------------------------------------------------------------------
// Task-Branch association (loose coupling: only records branch physical location, no VCS table references)
// -----------------------------------------------------------------------------

var ErrTaskBranchAlreadyLinked = errors.New("branch already linked to this task")

// ListTaskBranches retrieves all branches associated with a task, ordered by creation time.
func (s *TaskService) ListTaskBranches(projectID, taskID int) ([]*model.TaskBranch, error) {
	var branches []*model.TaskBranch
	err := s.db.Where("project_id = ? AND task_id = ?", projectID, taskID).
		Order("created_at ASC").Find(&branches).Error
	return branches, err
}

// FindTaskBranchesByRepoBranch performs a reverse lookup: given a repo full
// name ("owner/repo") and branch name, returns all task-branch associations
// across all projects. Used by the push event subscriber to find tasks linked
// to the branch that was just pushed to.
func (s *TaskService) FindTaskBranchesByRepoBranch(repoFullName, branchName string) ([]*model.TaskBranch, error) {
	var branches []*model.TaskBranch
	err := s.db.Where("repo_full_name = ? AND branch_name = ?", repoFullName, branchName).
		Find(&branches).Error
	return branches, err
}

// AddTaskBranch associates a branch with a task. Returns ErrTaskBranchAlreadyLinked
// if the (project, task, repoFullName, branchName) tuple is already linked.
// baseSHA is the commit SHA of the base branch at creation time (used by GetTaskCommits
// to list branch-unique commits even after the branch is merged into the default branch).
// Uses the pre-check pattern (consistent with RepositoryService) since the codebase
// does not rely on DB-specific unique-violation error detection.
func (s *TaskService) AddTaskBranch(projectID, taskID int, repoFullName, branchName, baseSHA, userName string) (*model.TaskBranch, error) {
	var count int64
	if err := s.db.Model(&model.TaskBranch{}).
		Where("project_id = ? AND task_id = ? AND repo_full_name = ? AND branch_name = ?",
			projectID, taskID, repoFullName, branchName).
		Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrTaskBranchAlreadyLinked
	}

	tb := &model.TaskBranch{
		ProjectID:    projectID,
		TaskID:       taskID,
		RepoFullName: repoFullName,
		BranchName:   branchName,
		BaseSHA:      baseSHA,
		UserName:     userName,
	}
	if err := s.db.Create(tb).Error; err != nil {
		return nil, err
	}
	return tb, nil
}

// RemoveTaskBranch removes a task-branch association by id, scoped to (projectID, taskID)
// to prevent cross-task deletion.
func (s *TaskService) RemoveTaskBranch(projectID, taskID, branchID int) error {
	return s.db.Where("id = ? AND project_id = ? AND task_id = ?", branchID, projectID, taskID).
		Delete(&model.TaskBranch{}).Error
}

// --- Task Dependency (hard block + cycle prevention) ---

var (
	ErrTaskDependencyAlreadyExists = errors.New("dependency already exists")
	ErrCircularDependency          = errors.New("circular dependency detected")
	ErrTaskSelfDependency          = errors.New("task cannot depend on itself")
	ErrInvalidDependencyType       = errors.New("invalid dependency type, must be one of: fs, ss, ff, sf")
)

// DependencyTask represents one end of a dependency relationship (for frontend to display id/title/status/type).
// ListDependencies returns predecessor tasks (B), ListBlocks returns successor tasks (A); both share this structure.
type DependencyTask struct {
	TaskID         int    `gorm:"column:task_id" json:"taskId"`
	Title          string `gorm:"column:title" json:"title"`
	Status         string `gorm:"column:status" json:"status"`
	DependencyType string `gorm:"column:dependency_type" json:"dependencyType"`
}

// BlockedTaskInfo describes a prerequisite task (incomplete) that blocks the current task
type BlockedTaskInfo struct {
	TaskID         int    `json:"taskId"`
	Title          string `json:"title"`
	Status         string `json:"status"`
	DependencyType string `json:"dependencyType"`
}

// BlockedError is returned when a task cannot transition to in-progress status due to unfinished prerequisites.
// Handler layer uses errors.As to extract BlockingTasks and generate structured 422 response.
type BlockedError struct {
	BlockingTasks []BlockedTaskInfo
}

func (e *BlockedError) Error() string {
	return fmt.Sprintf("task is blocked by %d unfinished dependencies", len(e.BlockingTasks))
}

// isClosedStatusSlug checks if status slug is a legacy closed state (aligned with UpdateStatus isClosedStatus)
func isClosedStatusSlug(slug string) bool {
	return slug == "done" || slug == "closed" || slug == "completed" || slug == "archived"
}

// inferCategorySlug infers three-state category from status slug (fallback when no task_status config exists).
// Aligned with TaskStatus.Category: todo / in_progress / done.
func inferCategorySlug(slug string, isClosed bool) string {
	if isClosed || isClosedStatusSlug(slug) {
		return "done"
	}
	if slug == "in_progress" || slug == "active" || slug == "doing" || slug == "review" {
		return "in_progress"
	}
	return "todo"
}

// GetStatusSlugByCategory returns the first status slug for a given category in a project (ordered by position ASC).
// Used for VCS auto status transition scenarios: branch push → in_progress, MR merged → done, etc.
// Backward compatible: when project has no task_status row configured, returns the category string itself —
// "todo"/"in_progress"/"done" are both category names and legacy hardcoded slugs; fallback aligns with old behavior.
func (s *TaskService) GetStatusSlugByCategory(projectID int, category string) string {
	var ts model.TaskStatus
	if err := s.db.Select("slug").
		Where("project_id = ? AND category = ?", projectID, category).
		Order("position ASC, id ASC").
		First(&ts).Error; err == nil {
		return ts.Slug
	}
	return category
}

// GetReviewStatusSlug returns the slug for "in review" status in project, resolution priority:
//  1. Explicit slug="review" (default config)
//  2. Second in_progress-like status (review follows in_progress by default)
//  3. The only in_progress-like status (fallback when project has only one in_progress column)
//  4. Hardcoded "review" (fallback when project has no task_status config, aligned with original behavior)
func (s *TaskService) GetReviewStatusSlug(projectID int) string {
	var review model.TaskStatus
	if err := s.db.Where("project_id = ? AND slug = ?", projectID, "review").First(&review).Error; err == nil {
		return review.Slug
	}
	var statuses []model.TaskStatus
	if err := s.db.Where("project_id = ? AND category = ?", projectID, "in_progress").
		Order("position ASC, id ASC").Find(&statuses).Error; err == nil {
		if len(statuses) >= 2 {
			return statuses[1].Slug
		}
		if len(statuses) == 1 {
			return statuses[0].Slug
		}
	}
	return "review"
}

// GetInProgressStatusSlug is a convenience wrapper for GetStatusSlugByCategory(projectID, "in_progress").
func (s *TaskService) GetInProgressStatusSlug(projectID int) string {
	return s.GetStatusSlugByCategory(projectID, "in_progress")
}

// GetStatusCategory returns the category (todo/in_progress/done) for a slug in project.
// Backward compatible: uses inferCategorySlug fallback when slug not found in task_status table.
func (s *TaskService) GetStatusCategory(projectID int, slug string) string {
	var ts model.TaskStatus
	if err := s.db.Select("category").
		Where("project_id = ? AND slug = ?", projectID, slug).
		First(&ts).Error; err == nil {
		return ts.Category
	}
	return inferCategorySlug(slug, false)
}

// IsClosedStatusByConfig determines if a slug is a closed state under project configuration.
// Replaces hardcoded isClosedStatus check in UpdateStatus/UpdateStatusIfCurrent.
// Backward compatible: falls back to isClosedStatusSlug when slug is not found.
func (s *TaskService) IsClosedStatusByConfig(projectID int, slug string) bool {
	var ts model.TaskStatus
	if err := s.db.Select("is_closed").
		Where("project_id = ? AND slug = ?", projectID, slug).
		First(&ts).Error; err == nil {
		return ts.IsClosed
	}
	return isClosedStatusSlug(slug)
}

// isBlockingDep determines if a dependency constitutes blocking during target status transition.
// FS: when A→in_progress, B must be completed; SS: when A→in_progress, B must be started;
// FF: when A→done, B must be completed; SF: when A→done, B must be started.
func isBlockingDep(depType, bCategory, targetCategory string) bool {
	bDone := bCategory == "done"
	bStarted := bCategory == "in_progress" || bCategory == "done"
	switch depType {
	case "fs":
		return targetCategory == "in_progress" && !bDone
	case "ss":
		return targetCategory == "in_progress" && !bStarted
	case "ff":
		return targetCategory == "done" && !bDone
	case "sf":
		return targetCategory == "done" && !bStarted
	default:
		return false
	}
}

// isPrerequisiteUnmet checks if dependency prerequisite is unmet (for card blocking count, ignores target status).
// FS/FF: B not completed; SS/SF: B not started.
func isPrerequisiteUnmet(depType, bCategory string) bool {
	bDone := bCategory == "done"
	bStarted := bCategory == "in_progress" || bCategory == "done"
	switch depType {
	case "fs", "ff":
		return !bDone
	case "ss", "sf":
		return !bStarted
	default:
		return false
	}
}

// ListDependencies returns prerequisite tasks that task depends on (A depends on B -> returns B summary),
// JOINs task table to fill title/status for frontend TaskDrawer to display prerequisites.
func (s *TaskService) ListDependencies(projectID, taskID int) ([]DependencyTask, error) {
	var tasks []DependencyTask
	err := s.db.Raw(`
		SELECT t.task_id, t.title, t.status, td.dependency_type
		FROM task_dependency td
		JOIN task t ON t.project_id = td.project_id AND t.task_id = td.depends_on_task_id
		WHERE td.project_id = ? AND td.task_id = ?
		ORDER BY td.created_at ASC`, projectID, taskID).Scan(&tasks).Error
	return tasks, err
}

// ListBlocks returns successor tasks that depend on current task (who is blocked by current task), symmetric query.
// Used by frontend TaskDrawer to display "blocks whom" read-only section.
func (s *TaskService) ListBlocks(projectID, taskID int) ([]DependencyTask, error) {
	var tasks []DependencyTask
	err := s.db.Raw(`
		SELECT t.task_id, t.title, t.status, td.dependency_type
		FROM task_dependency td
		JOIN task t ON t.project_id = td.project_id AND t.task_id = td.task_id
		WHERE td.project_id = ? AND td.depends_on_task_id = ?
		ORDER BY td.created_at ASC`, projectID, taskID).Scan(&tasks).Error
	return tasks, err
}

// AddDependency creates directed dependency taskID -> dependsOnTaskID (A depends on B), depType specifies type (fs/ss/ff/sf).
// Validation: type legality/self/existence/dedup/cycle prevention, follows AddTaskBranch pre-check pattern.
func (s *TaskService) AddDependency(projectID, taskID, dependsOnTaskID int, depType string) error {
	if depType == "" {
		depType = "fs"
	}
	if depType != "fs" && depType != "ss" && depType != "ff" && depType != "sf" {
		return ErrInvalidDependencyType
	}
	if taskID == dependsOnTaskID {
		return ErrTaskSelfDependency
	}
	// Verify both tasks exist (same project)
	var existCount int64
	if err := s.db.Model(&model.Task{}).
		Where("project_id = ? AND task_id IN ?", projectID, []int{taskID, dependsOnTaskID}).
		Count(&existCount).Error; err != nil {
		return err
	}
	if existCount < 2 {
		return ErrTaskNotFound
	}
	// Dedup: reject if same dependency already exists
	var dupCount int64
	if err := s.db.Model(&model.TaskDependency{}).
		Where("project_id = ? AND task_id = ? AND depends_on_task_id = ?", projectID, taskID, dependsOnTaskID).
		Count(&dupCount).Error; err != nil {
		return err
	}
	if dupCount > 0 {
		return ErrTaskDependencyAlreadyExists
	}
	// Cycle prevention: after adding taskID->dependsOnTaskID, if dependsOnTaskID can reach taskID along depends_on direction, cycle exists
	cycle, err := s.wouldCreateCycle(projectID, taskID, dependsOnTaskID)
	if err != nil {
		return err
	}
	if cycle {
		return ErrCircularDependency
	}
	dep := &model.TaskDependency{
		ProjectID:       projectID,
		TaskID:          taskID,
		DependsOnTaskID: dependsOnTaskID,
		DependencyType:  depType,
	}
	return s.db.Create(dep).Error
}

// wouldCreateCycle checks if adding taskID→dependsOnTaskID edge would form a cycle.
// Algorithm: fetch all project dependency edges to build adjacency list, DFS from dependsOnTaskID along depends_on direction,
// if can reach taskID then B (indirectly) depends on A, adding A→B would create a cycle.
func (s *TaskService) wouldCreateCycle(projectID, taskID, dependsOnTaskID int) (bool, error) {
	var edges []model.TaskDependency
	if err := s.db.Select("task_id, depends_on_task_id").
		Where("project_id = ?", projectID).Find(&edges).Error; err != nil {
		return false, err
	}
	adj := make(map[int][]int)
	for _, e := range edges {
		adj[e.TaskID] = append(adj[e.TaskID], e.DependsOnTaskID)
	}
	visited := make(map[int]bool)
	var dfs func(node int) bool
	dfs = func(node int) bool {
		if node == taskID {
			return true
		}
		if visited[node] {
			return false
		}
		visited[node] = true
		for _, next := range adj[node] {
			if dfs(next) {
				return true
			}
		}
		return false
	}
	return dfs(dependsOnTaskID), nil
}

// RemoveDependency deletes directed dependency taskID -> dependsOnTaskID, composite primary key restricts to project scope.
func (s *TaskService) RemoveDependency(projectID, taskID, dependsOnTaskID int) error {
	return s.db.Where("project_id = ? AND task_id = ? AND depends_on_task_id = ?",
		projectID, taskID, dependsOnTaskID).Delete(&model.TaskDependency{}).Error
}

// GetBlockingTasks returns predecessor dependencies that block current task during target status transition.
// targetCategory is the target status category (in_progress/done), determines blocking by dependency type (fs/ss/ff/sf).
func (s *TaskService) GetBlockingTasks(projectID, taskID int, targetCategory string) ([]BlockedTaskInfo, error) {
	type row struct {
		TaskID         int    `gorm:"column:task_id"`
		Title          string `gorm:"column:title"`
		Status         string `gorm:"column:status"`
		IsClosed       bool   `gorm:"column:is_closed"`
		BCategory      string `gorm:"column:b_category"`
		DependencyType string `gorm:"column:dependency_type"`
	}
	var rows []row
	err := s.db.Raw(`
		SELECT t.task_id, t.title, t.status,
		       COALESCE(ts.is_closed, 0) as is_closed,
		       COALESCE(ts.category, '') as b_category,
		       td.dependency_type
		FROM task_dependency td
		JOIN task t ON t.project_id = td.project_id AND t.task_id = td.depends_on_task_id
		LEFT JOIN task_status ts ON ts.project_id = t.project_id AND ts.slug = t.status
		WHERE td.project_id = ? AND td.task_id = ?`, projectID, taskID).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	var blocking []BlockedTaskInfo
	for _, r := range rows {
		bCat := r.BCategory
		if bCat == "" {
			bCat = inferCategorySlug(r.Status, r.IsClosed)
		}
		if isBlockingDep(r.DependencyType, bCat, targetCategory) {
			blocking = append(blocking, BlockedTaskInfo{
				TaskID:         r.TaskID,
				Title:          r.Title,
				Status:         r.Status,
				DependencyType: r.DependencyType,
			})
		}
	}
	return blocking, nil
}

// CheckBlock public read-only validation: when target status category is in_progress or done, query GetBlockingTasks,
// returns *BlockedError if blocking prerequisites exist. Shared by UpdateStatus and UpdateTask handler,
// covers kanban drag, TaskDrawer inline, edit save all status change paths.
// Backward compatible: when target status not found (legacy slug), in_progress/done-like slugs trigger check.
func (s *TaskService) CheckBlock(projectID, taskID int, targetStatusSlug string) error {
	var ts model.TaskStatus
	err := s.db.Select("category").Where("project_id = ? AND slug = ?", projectID, targetStatusSlug).First(&ts).Error
	targetCategory := ""
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Legacy slug fallback: in_progress-like -> in_progress, closed-like -> done
			if targetStatusSlug == "in_progress" {
				targetCategory = "in_progress"
			} else if isClosedStatusSlug(targetStatusSlug) {
				targetCategory = "done"
			} else {
				return nil
			}
		} else {
			return err
		}
	} else {
		targetCategory = ts.Category
	}
	// Only in_progress and done transitions can be blocked (FS/SS blocks start, FF/SF blocks completion)
	if targetCategory != "in_progress" && targetCategory != "done" {
		return nil
	}

	blocking, err := s.GetBlockingTasks(projectID, taskID, targetCategory)
	if err != nil {
		return err
	}
	if len(blocking) > 0 {
		return &BlockedError{BlockingTasks: blocking}
	}
	return nil
}

// BatchGetBlockedByCounts batch queries unmet prerequisite dependency count for each task, for kanban sideload.
// Judged by dependency type: FS/FF requires B completed, SS/SF requires B started. Ignores target status (for card indicator).
// Returns map[taskID]count; tasks without dependencies or with all dependencies met are not in map (frontend treats as 0).
func (s *TaskService) BatchGetBlockedByCounts(projectID int, taskIDs []int) (map[int]int, error) {
	result := make(map[int]int)
	if len(taskIDs) == 0 {
		return result, nil
	}
	type row struct {
		TaskID         int    `gorm:"column:task_id"`
		Status         string `gorm:"column:status"`
		IsClosed       bool   `gorm:"column:is_closed"`
		BCategory      string `gorm:"column:b_category"`
		DependencyType string `gorm:"column:dependency_type"`
	}
	var rows []row
	err := s.db.Raw(`
		SELECT td.task_id, t.status,
		       COALESCE(ts.is_closed, 0) as is_closed,
		       COALESCE(ts.category, '') as b_category,
		       td.dependency_type
		FROM task_dependency td
		JOIN task t ON t.project_id = td.project_id AND t.task_id = td.depends_on_task_id
		LEFT JOIN task_status ts ON ts.project_id = t.project_id AND ts.slug = t.status
		WHERE td.project_id = ? AND td.task_id IN ?`, projectID, taskIDs).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		bCat := r.BCategory
		if bCat == "" {
			bCat = inferCategorySlug(r.Status, r.IsClosed)
		}
		if isPrerequisiteUnmet(r.DependencyType, bCat) {
			result[r.TaskID]++
		}
	}
	return result, nil
}
