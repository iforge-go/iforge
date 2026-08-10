package model

import "time"

// UserStory represents a user story within a project (Product Backlog Item).
// Sprint planning uses Story as the unit (aligned with Jira / Scrum Guide):
// SprintID is nullable; when non-null the Story is planned into that Sprint.
// Its Tasks inherit Sprint membership via the parent Story's sprint_id (no separate assignment needed).
// Standalone Tasks (no user_story_id) can still enter a Sprint directly via task.sprint_id.
//
// Epic association: EpicID is nullable; orphan Stories in the Backlog belong to no Epic.
// EpicSlug/EpicTitle are non-persisted; filled in bulk by the handler layer (avoids N+1).
// SprintSlug is non-persisted; filled by the handler layer using encodeID(sprintID).
type UserStory struct {
	ID int `gorm:"primaryKey;autoIncrement;column:id" json:"-"`
	// Composite index (project_id, status) covers Backlog queries by project + status
	ProjectID          int     `gorm:"column:project_id;index:idx_story_project_status,priority:1" json:"-"`
	ProjectSlug        string  `gorm:"-" json:"projectSlug"`
	Title              string  `gorm:"column:title" json:"title"`
	Description        *string `gorm:"column:description" json:"description"`
	Slug               string  `gorm:"-" json:"slug"`
	Status             string  `gorm:"column:status;index:idx_story_project_status,priority:2" json:"status"`
	Priority           string  `gorm:"column:priority" json:"priority"`
	StoryPoints        *int    `gorm:"column:story_points" json:"storyPoints"`
	AcceptanceCriteria *string `gorm:"column:acceptance_criteria" json:"acceptanceCriteria"`
	AssigneeName       *string `gorm:"column:assignee_name" json:"assigneeName"`
	ReporterName       string  `gorm:"column:reporter_name" json:"reporterName"`
	// EpicID is a nullable FK to Epic. Set to NULL when the Epic is deleted (soft-disassociate)
	EpicID    *int    `gorm:"column:epic_id;index:idx_story_epic" json:"-"`
	EpicSlug  *string `gorm:"-" json:"epicSlug"`
	EpicTitle *string `gorm:"-" json:"epicTitle"`
	// SprintID is a nullable FK to Sprint. Non-null means the Story is planned into that Sprint (Story-level Sprint planning)
	SprintID   *int    `gorm:"column:sprint_id;index:idx_story_sprint" json:"-"`
	SprintSlug *string `gorm:"-" json:"sprintSlug"`
	// TaskCount is non-persisted; filled in bulk by the handler layer (avoids N+1);
	// used to display task-count badges on Backlog story rows
	TaskCount int        `gorm:"-" json:"taskCount"`
	CreatedAt time.Time  `gorm:"column:created_at;index:idx_story_created" json:"createdAt"`
	UpdatedAt time.Time  `gorm:"column:updated_at" json:"updatedAt"`
	ClosedAt  *time.Time `gorm:"column:closed_at" json:"closedAt"`
}

func (UserStory) TableName() string { return "user_story" }
