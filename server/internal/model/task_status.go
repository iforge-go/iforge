package model

import "time"

// TaskStatus represents a configurable task status for kanban columns
type TaskStatus struct {
	ID          int       `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	ProjectID   int       `gorm:"column:project_id;not null" json:"-"`
	ProjectSlug string    `gorm:"-" json:"projectSlug"`
	Name        string    `gorm:"column:name;not null" json:"name"`
	Slug        string    `gorm:"column:slug;not null" json:"slug"`
	Color       string    `gorm:"column:color;default:'#94a3b8'" json:"color"`
	Position    int       `gorm:"column:position;default:0" json:"position"`
	IsDefault   bool      `gorm:"column:is_default;default:false" json:"isDefault"`
	IsClosed    bool      `gorm:"column:is_closed;default:false" json:"isClosed"`
	// Category is a three-state classification (aligned with Jira status category):
	// todo / in_progress / done.
	// Drives reporting (burndown/velocity chart "done" determination) and kanban column grouping.
	// Populated from existing data by MigrateStatusCategory.
	Category string `gorm:"column:category;default:'todo'" json:"category"`
	// WipLimit is the kanban column's work-in-progress limit (aligned with Jira Kanban WIP).
	// nil = unlimited; >0 shows N/Limit in column header and highlights in red when exceeded.
	WipLimit *int `gorm:"column:wip_limit" json:"wipLimit"`
	CreatedAt   time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"column:updated_at" json:"updatedAt"`
}

func (TaskStatus) TableName() string { return "task_status" }

// TaskStatusTransition represents project-configured allowed status transition rules
// (workflow constraints, aligned with Jira workflow transitions).
// Semantics: empty table = all transitions allowed (backward compatible; no config means no constraint);
// when non-empty, a task status change from->to must match a row in this table.
// Uses IDs (not slugs) for stable references; the handler layer converts slug<->ID and fills
// FromSlug/ToSlug for frontend display.
type TaskStatusTransition struct {
	ProjectID    int    `gorm:"primaryKey;column:project_id" json:"-"`
	ProjectSlug  string `gorm:"-" json:"projectSlug"`
	FromStatusID int    `gorm:"primaryKey;column:from_status_id" json:"fromStatusId"`
	ToStatusID   int    `gorm:"primaryKey;column:to_status_id" json:"toStatusId"`
	// Non-persisted: filled by handler; frontend can display directly without reverse lookup
	FromSlug string `gorm:"-" json:"fromSlug"`
	ToSlug   string `gorm:"-" json:"toSlug"`
}

func (TaskStatusTransition) TableName() string { return "task_status_transition" }
