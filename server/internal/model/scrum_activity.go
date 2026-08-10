package model

import "time"

// ScrumActivity represents a Scrum activity log entry (task/story/sprint changes)
type ScrumActivity struct {
	ID         int       `gorm:"primaryKey;autoIncrement" json:"-"`
	Slug       string    `gorm:"-" json:"slug"`
	ProjectID  int       `gorm:"column:project_id;index" json:"-"`
	UserName   string    `gorm:"column:user_name" json:"userName"`
	EntityType string    `gorm:"column:entity_type;index" json:"entityType"` // task, story, sprint
	EntityID   int       `gorm:"column:entity_id;index" json:"entityId"`
	Action     string    `gorm:"column:action" json:"action"` // created, updated, status_changed, deleted
	Field      string    `gorm:"column:field" json:"field"`   // status, priority, assignee, title, etc.
	OldValue   string    `gorm:"column:old_value" json:"oldValue"`
	NewValue   string    `gorm:"column:new_value" json:"newValue"`
	CreatedAt  time.Time `gorm:"column:created_at" json:"createdAt"`
}

func (ScrumActivity) TableName() string { return "scrum_activity" }
