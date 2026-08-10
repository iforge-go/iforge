package model

import "time"

// Project represents a project management entity (corresponds to gitscrum's ProductBacklog)
type Project struct {
	ID int `gorm:"primaryKey;autoIncrement;column:id" json:"-"`
	// Index on owner_name covers ListProjects queries by owner
	OwnerName      string    `gorm:"column:owner_name;index:idx_project_owner" json:"ownerName"`
	Name           string    `gorm:"column:name" json:"name"`
	Slug           string    `gorm:"-" json:"slug"`
	Description    *string   `gorm:"column:description" json:"description"`
	IsPrivate      bool      `gorm:"column:is_private" json:"isPrivate"`
	RegisteredDate time.Time `gorm:"column:registered_date" json:"registeredDate"`
	UpdatedDate    time.Time `gorm:"column:updated_date;index:idx_project_updated" json:"updatedDate"`
	// MyRole is the current requesting user's role in this project (owner/admin/member/viewer).
	// Non-persisted; filled by the handler layer for list-view display.
	MyRole string `gorm:"-" json:"myRole,omitempty"`
	// Statistics counters (non-persisted; filled in bulk by the service layer)
	MemberCount    int `gorm:"-" json:"memberCount"`
	SprintCount    int `gorm:"-" json:"sprintCount"`
	UserStoryCount int `gorm:"-" json:"userStoryCount"`
}

func (Project) TableName() string { return "project" }

// ProjectMember represents a project member
type ProjectMember struct {
	ProjectID   int    `gorm:"primaryKey;column:project_id" json:"-"`
	ProjectSlug string `gorm:"-" json:"projectSlug"`
	// Single-column index on user_name supports reverse lookup of user's projects (used by ListProjects subquery)
	UserName   string    `gorm:"primaryKey;column:user_name;index:idx_pmember_user" json:"userName"`
	Role       string    `gorm:"column:role" json:"role"`
	JoinedDate time.Time `gorm:"column:joined_date" json:"joinedDate"`
	FullName   string    `gorm:"-" json:"fullName"`
	Image      *string   `gorm:"-" json:"image"`
}

func (ProjectMember) TableName() string { return "project_member" }

// ProjectRepository represents the relationship between project and repository
type ProjectRepository struct {
	ProjectID      int    `gorm:"primaryKey;column:project_id" json:"-"`
	ProjectSlug    string `gorm:"-" json:"projectSlug"`
	UserName       string `gorm:"primaryKey;column:user_name" json:"userName"`
	RepositoryName string `gorm:"primaryKey;column:repository_name" json:"repositoryName"`
}

func (ProjectRepository) TableName() string { return "project_repository" }
