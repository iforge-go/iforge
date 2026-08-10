package model

import "time"

// Sprint represents a sprint/iteration within a project
type Sprint struct {
	ID            int        `gorm:"primaryKey;autoIncrement;column:id" json:"-"`
	// Composite index (project_id, status) covers Sprint list queries by project + status
	ProjectID     int        `gorm:"column:project_id;index:idx_sprint_project_status,priority:1" json:"-"`
	ProjectSlug   string     `gorm:"-" json:"projectSlug"`
	Title         string     `gorm:"column:title" json:"title"`
	Description   *string    `gorm:"column:description" json:"description"`
	Slug          string     `gorm:"-" json:"slug"`
	Status        string     `gorm:"column:status;index:idx_sprint_project_status,priority:2" json:"status"`
	Goal          *string    `gorm:"column:goal" json:"goal"`
	StartDate     *time.Time `gorm:"column:start_date" json:"startDate"`
	EndDate       *time.Time `gorm:"column:end_date" json:"endDate"`
	CompletedDate *time.Time `gorm:"column:completed_date" json:"completedDate"`
	CreatedBy     string     `gorm:"column:created_by" json:"createdBy"`
	CreatedAt     time.Time  `gorm:"column:created_at;index:idx_sprint_created" json:"createdAt"`
	UpdatedAt     time.Time  `gorm:"column:updated_at" json:"updatedAt"`

	// Statistics fields (non-persisted; batch-computed by ListSprints; aligned with Story-level Sprint planning)
	// TaskCount: root tasks in this Sprint (direct task.sprint_id + inherited tasks via parent Story)
	// StoryCount: Stories directly associated with this Sprint (story.sprint_id)
	// TotalStoryPoints: sum of story points of Stories in this Sprint
	TaskCount        int `gorm:"-" json:"taskCount"`
	StoryCount       int `gorm:"-" json:"storyCount"`
	TotalStoryPoints int `gorm:"-" json:"totalStoryPoints"`
}

func (Sprint) TableName() string { return "sprint" }
