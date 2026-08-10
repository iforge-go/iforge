package model

import "time"

// Task represents a task within a project (mirrors Issue's composite-key pattern)
type Task struct {
	// project_id is the prefix of all queries; first column of multiple composite indexes
	ProjectID   int    `gorm:"primaryKey;column:project_id;index:idx_task_sprint,priority:1;index:idx_task_story,priority:1;index:idx_task_parent,priority:1;index:idx_task_status,priority:1" json:"-"`
	ProjectSlug string `gorm:"-" json:"projectSlug"`
	TaskID      int    `gorm:"primaryKey;column:task_id" json:"taskId"`
	// Index on sprint_id covers GetSprintTasks
	SprintID   *int    `gorm:"column:sprint_id;index:idx_task_sprint,priority:2" json:"-"`
	SprintSlug *string `gorm:"-" json:"sprintSlug"`
	// Index on user_story_id covers per-story task queries
	UserStoryID   *int    `gorm:"column:user_story_id;index:idx_task_story,priority:2" json:"-"`
	UserStorySlug *string `gorm:"-" json:"userStorySlug"`
	Title         string  `gorm:"column:title" json:"title"`
	Description   *string `gorm:"column:description" json:"description"`
	Slug          string  `gorm:"-" json:"slug"`
	// Index on status covers status-filter queries
	Status         string   `gorm:"column:status;index:idx_task_status,priority:2" json:"status"`
	Priority       string   `gorm:"column:priority" json:"priority"`
	TaskType       string   `gorm:"column:task_type" json:"taskType"`
	StoryPoints    *int     `gorm:"column:story_points" json:"storyPoints"`
	EstimatedHours *float64 `gorm:"column:estimated_hours" json:"estimatedHours"`
	AssigneeName   *string  `gorm:"column:assignee_name" json:"-"`
	ReporterName   string   `gorm:"column:reporter_name" json:"reporterName"`
	// Index on parent_id covers GetSubtasks / GetSubtaskCounts
	ParentID *int       `gorm:"column:parent_id;index:idx_task_parent,priority:2" json:"parentId"`
	RootID   *int       `gorm:"column:root_id" json:"rootId"`
	Position int        `gorm:"column:position" json:"position"`
	DueDate  *time.Time `gorm:"column:due_date" json:"dueDate"`
	// Index on created_at covers ORDER BY position, created_at
	CreatedAt    time.Time  `gorm:"column:created_at;index:idx_task_created" json:"createdAt"`
	UpdatedAt    time.Time  `gorm:"column:updated_at" json:"updatedAt"`
	ClosedAt     *time.Time `gorm:"column:closed_at" json:"closedAt"`
	ClosedByName *string    `gorm:"column:closed_by_name" json:"closedByName"`
}

func (Task) TableName() string { return "task" }

// TaskIDCounter represents the task ID counter for a project (mirrors IssueIDCounter)
type TaskIDCounter struct {
	ProjectID   int    `gorm:"primaryKey;column:project_id" json:"-"`
	ProjectSlug string `gorm:"-" json:"projectSlug"`
	TaskID      int    `gorm:"column:task_id" json:"taskId"`
}

func (TaskIDCounter) TableName() string { return "task_id_counter" }

// TaskComment represents a comment on a task
type TaskComment struct {
	CommentID int `gorm:"primaryKey;autoIncrement;column:comment_id" json:"commentId"`
	// Composite index (project_id, task_id) covers GetComments
	ProjectID   int       `gorm:"column:project_id;index:idx_task_comment,priority:1" json:"-"`
	ProjectSlug string    `gorm:"-" json:"projectSlug"`
	TaskID      int       `gorm:"column:task_id;index:idx_task_comment,priority:2" json:"taskId"`
	AuthorName  string    `gorm:"column:author_name" json:"authorName"`
	Content     string    `gorm:"column:content" json:"content"`
	CreatedAt   time.Time `gorm:"column:created_at;index:idx_task_comment_created" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"column:updated_at" json:"updatedAt"`
}

func (TaskComment) TableName() string { return "task_comment" }

// TaskWorkLog represents a work log entry on a task (time tracking)
type TaskWorkLog struct {
	LogID       int64     `gorm:"primaryKey;autoIncrement;column:log_id" json:"logId"`
	ProjectID   int       `gorm:"column:project_id;index:idx_task_worklog,priority:1" json:"-"`
	ProjectSlug string    `gorm:"-" json:"projectSlug"`
	TaskID      int       `gorm:"column:task_id;index:idx_task_worklog,priority:2" json:"taskId"`
	UserName    string    `gorm:"column:user_name" json:"userName"`
	Hours       float64   `gorm:"column:hours" json:"hours"`
	Description string    `gorm:"column:description" json:"description"`
	CreatedAt   time.Time `gorm:"column:created_at;index:idx_task_worklog_created" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"column:updated_at" json:"updatedAt"`
}

func (TaskWorkLog) TableName() string { return "task_work_log" }

// TaskLabel represents a label for tasks
type TaskLabel struct {
	ID          int    `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	ProjectID   int    `gorm:"column:project_id" json:"-"`
	ProjectSlug string `gorm:"-" json:"projectSlug"`
	Name        string `gorm:"column:name" json:"name"`
	Color       string `gorm:"column:color" json:"color"`
}

func (TaskLabel) TableName() string { return "task_label" }

// TaskLabelAssignment represents the assignment of labels to tasks
type TaskLabelAssignment struct {
	ProjectID   int    `gorm:"primaryKey;column:project_id" json:"-"`
	ProjectSlug string `gorm:"-" json:"projectSlug"`
	TaskID      int    `gorm:"primaryKey;column:task_id" json:"taskId"`
	LabelID     int    `gorm:"primaryKey;column:label_id" json:"labelId"`
}

func (TaskLabelAssignment) TableName() string { return "task_label_assignment" }

// TaskAssignment represents the many-to-many relationship between tasks and assignees
type TaskAssignment struct {
	ProjectID        int    `gorm:"primaryKey;column:project_id" json:"-"`
	ProjectSlug      string `gorm:"-" json:"projectSlug"`
	TaskID           int    `gorm:"primaryKey;column:task_id" json:"taskId"`
	AssigneeUserName string `gorm:"primaryKey;column:assignee_user_name" json:"assigneeUserName"`
}

func (TaskAssignment) TableName() string { return "task_assignment" }

// TaskDependency is a directed dependency: task_id depends on depends_on_task_id (B is a predecessor of A).
// Composite primary key (project_id, task_id, depends_on_task_id) prevents duplicates and scopes to a project,
// mirroring the structure of TaskAssignment.
type TaskDependency struct {
	ProjectID           int       `gorm:"primaryKey;column:project_id" json:"-"`
	ProjectSlug         string    `gorm:"-" json:"projectSlug"`
	TaskID              int       `gorm:"primaryKey;column:task_id" json:"taskId"`
	DependsOnTaskID     int       `gorm:"primaryKey;column:depends_on_task_id" json:"dependsOnTaskId"`
	DependencyType      string    `gorm:"column:dependency_type;not null;default:fs" json:"dependencyType"`
	DependsOnTaskTitle  string    `gorm:"-" json:"dependsOnTaskTitle,omitempty"`
	DependsOnTaskStatus string    `gorm:"-" json:"dependsOnTaskStatus,omitempty"`
	CreatedAt           time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
}

func (TaskDependency) TableName() string { return "task_dependency" }

// TaskStatusHistory records every status transition of a task for traceability
type TaskStatusHistory struct {
	ID int `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	// Composite index (project_id, task_id, created_at) covers GetStatusHistory and deduplication checks
	ProjectID     int       `gorm:"column:project_id;index:idx_task_hist,priority:1" json:"-"`
	ProjectSlug   string    `gorm:"-" json:"projectSlug"`
	TaskID        int       `gorm:"column:task_id;index:idx_task_hist,priority:2" json:"taskId"`
	OldStatus     string    `gorm:"column:old_status" json:"oldStatus"`
	NewStatus     string    `gorm:"column:new_status" json:"newStatus"`
	ChangedByName string    `gorm:"column:changed_by_name" json:"changedByName"`
	Comment       *string   `gorm:"column:comment" json:"comment"`
	CreatedAt     time.Time `gorm:"column:created_at;index:idx_task_hist,priority:3" json:"createdAt"`
}

func (TaskStatusHistory) TableName() string { return "task_status_history" }
