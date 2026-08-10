package model

import "time"

// Epic represents a large requirement within a project, used to aggregate and manage
// User Stories across multiple Sprints. Epic -> UserStory -> Task is the three-layer
// hierarchy, with Epic at the top.
//
// Design notes:
//   - Slug uses hashid (aligned with Sprint); non-persisted; filled by handler's encodeID
//   - Status is lazily derived: when StatusManual=false, GetEpic/ListEpics compute it
//     on the fly by aggregating associated Story statuses
//   - Deleting an Epic sets associated Stories' epic_id to NULL (soft-disassociate, no cascade)
//   - Aggregate fields (TotalStories/DoneStories/...) are non-persisted; filled by
//     service-layer fillAggregates
type Epic struct {
	ID          int     `gorm:"primaryKey;autoIncrement;column:id" json:"-"`
	ProjectID   int     `gorm:"column:project_id;index:idx_epic_project_status,priority:1" json:"-"`
	ProjectSlug string  `gorm:"-" json:"projectSlug"`
	Title       string  `gorm:"column:title" json:"title"`
	Description *string `gorm:"column:description;type:text" json:"description"`
	Slug        string  `gorm:"-" json:"slug"`
	// Status: open, in_progress, done, closed.
	// When StatusManual=true the user set it manually (skip derivation);
	// when false, deriveStatus computes it in real time.
	Status       string     `gorm:"column:status;index:idx_epic_project_status,priority:2" json:"status"`
	StatusManual bool       `gorm:"column:status_manual" json:"statusManual"`
	Priority     string     `gorm:"column:priority" json:"priority"`
	Goal         *string    `gorm:"column:goal;type:text" json:"goal"`
	StartDate    *time.Time `gorm:"column:start_date" json:"startDate"`
	TargetDate   *time.Time `gorm:"column:target_date" json:"targetDate"`
	OwnerName    *string    `gorm:"column:owner_name" json:"ownerName"`
	ReporterName string     `gorm:"column:reporter_name" json:"reporterName"`
	CreatedAt    time.Time  `gorm:"column:created_at;index:idx_epic_created" json:"createdAt"`
	UpdatedAt    time.Time  `gorm:"column:updated_at" json:"updatedAt"`
	ClosedAt     *time.Time `gorm:"column:closed_at" json:"closedAt"`

	// Aggregate fields (non-persisted; filled by service/handler)
	TotalStories     int `gorm:"-" json:"totalStories"`
	DoneStories      int `gorm:"-" json:"doneStories"`
	TotalStoryPoints int `gorm:"-" json:"totalStoryPoints"`
	DoneStoryPoints  int `gorm:"-" json:"doneStoryPoints"`
	ProgressPercent  int `gorm:"-" json:"progressPercent"`
	SprintCount      int `gorm:"-" json:"sprintCount"`
}

func (Epic) TableName() string { return "epic" }
