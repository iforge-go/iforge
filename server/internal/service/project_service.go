package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

var (
	ErrProjectNotFound       = errors.New("project not found")
	ErrProjectMemberNotFound = errors.New("project member not found")
	// ErrProjectForbidden indicates insufficient user role level (returned by RequireRole).
	ErrProjectForbidden = errors.New("forbidden: insufficient project role")
	// ErrInvalidRole indicates the provided role is not one of the four valid roles.
	ErrInvalidRole = errors.New("invalid project role")
)

// ProjectService handles project-related operations
type ProjectService struct {
	db *gorm.DB
}

// NewProjectService creates a new ProjectService
func NewProjectService(db *gorm.DB) *ProjectService {
	return &ProjectService{
		db: db,
	}
}

// CreateProject creates a new project
func (s *ProjectService) CreateProject(ownerName, name string, description *string, isPrivate bool) (*model.Project, error) {
	now := time.Now()
	project := &model.Project{
		OwnerName:      ownerName,
		Name:           name,
		Description:    description,
		IsPrivate:      isPrivate,
		RegisteredDate: now,
		UpdatedDate:    now,
	}

	if err := s.db.Create(project).Error; err != nil {
		return nil, err
	}

	return project, nil
}

// GetProject retrieves a project by ID
func (s *ProjectService) GetProject(id int) (*model.Project, error) {
	var project model.Project
	err := s.db.Where("id = ?", id).First(&project).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}
	return &project, nil
}

// UpdateProject updates a project
func (s *ProjectService) UpdateProject(project *model.Project) error {
	project.UpdatedDate = time.Now()
	return s.db.Save(project).Error
}

// ListProjects lists projects for an owner
func (s *ProjectService) ListProjects(ownerName string, limit, offset int) ([]*model.Project, error) {
	var projects []*model.Project
	query := s.db.Where("owner_name = ? OR id IN (SELECT project_id FROM project_member WHERE user_name = ?)", ownerName, ownerName)

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Order("updated_date DESC").Find(&projects).Error
	return projects, err
}

// CountProjects returns the total number of projects
func (s *ProjectService) CountProjects() (int64, error) {
	var count int64
	err := s.db.Model(&model.Project{}).Count(&count).Error
	return count, err
}

// FillProjectCounts batch-fills member count, sprint count, and story count for projects.
// Uses GROUP BY to avoid N+1 queries.
func (s *ProjectService) FillProjectCounts(projects []*model.Project) error {
	if len(projects) == 0 {
		return nil
	}
	projectIDs := make([]int, len(projects))
	for i, p := range projects {
		projectIDs[i] = p.ID
	}

	type countResult struct {
		ProjectID int   `gorm:"column:project_id"`
		Count     int64 `gorm:"column:cnt"`
	}

	// Count members
	var memberCounts []countResult
	if err := s.db.Model(&model.ProjectMember{}).
		Select("project_id, count(*) as cnt").
		Where("project_id IN ?", projectIDs).
		Group("project_id").
		Scan(&memberCounts).Error; err != nil {
		return fmt.Errorf("failed to count members: %w", err)
	}

	// Count sprints
	var sprintCounts []countResult
	if err := s.db.Model(&model.Sprint{}).
		Select("project_id, count(*) as cnt").
		Where("project_id IN ?", projectIDs).
		Group("project_id").
		Scan(&sprintCounts).Error; err != nil {
		return fmt.Errorf("failed to count sprints: %w", err)
	}

	// Count user stories
	var storyCounts []countResult
	if err := s.db.Model(&model.UserStory{}).
		Select("project_id, count(*) as cnt").
		Where("project_id IN ?", projectIDs).
		Group("project_id").
		Scan(&storyCounts).Error; err != nil {
		return fmt.Errorf("failed to count user stories: %w", err)
	}

	memberMap := make(map[int]int, len(memberCounts))
	for _, r := range memberCounts {
		memberMap[r.ProjectID] = int(r.Count)
	}
	sprintMap := make(map[int]int, len(sprintCounts))
	for _, r := range sprintCounts {
		sprintMap[r.ProjectID] = int(r.Count)
	}
	storyMap := make(map[int]int, len(storyCounts))
	for _, r := range storyCounts {
		storyMap[r.ProjectID] = int(r.Count)
	}

	for _, p := range projects {
		p.MemberCount = memberMap[p.ID]
		p.SprintCount = sprintMap[p.ID]
		p.UserStoryCount = storyMap[p.ID]
	}
	return nil
}

// DeleteProject deletes a project
func (s *ProjectService) DeleteProject(id int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Delete project members
		if err := tx.Where("project_id = ?", id).Delete(&model.ProjectMember{}).Error; err != nil {
			return err
		}

		// Delete project repositories
		if err := tx.Where("project_id = ?", id).Delete(&model.ProjectRepository{}).Error; err != nil {
			return err
		}

		// Delete sprints
		if err := tx.Where("project_id = ?", id).Delete(&model.Sprint{}).Error; err != nil {
			return err
		}

		// Delete user stories
		if err := tx.Where("project_id = ?", id).Delete(&model.UserStory{}).Error; err != nil {
			return err
		}

		// Delete tasks
		if err := tx.Where("project_id = ?", id).Delete(&model.Task{}).Error; err != nil {
			return err
		}

		// Delete project
		if err := tx.Where("id = ?", id).Delete(&model.Project{}).Error; err != nil {
			return err
		}

		return nil
	})
}

// AddMember adds a member to a project.
// role must be one of the valid four levels (owner/admin/member/viewer).
// Note: owner is typically created internally by CreateProject; external AddMember handlers
// should reject user-submitted role=owner before calling.
func (s *ProjectService) AddMember(projectID int, userName, role string) error {
	if !model.ValidProjectRole(role) {
		return ErrInvalidRole
	}
	member := &model.ProjectMember{
		ProjectID:  projectID,
		UserName:   userName,
		Role:       strings.ToLower(role),
		JoinedDate: time.Now(),
	}
	return s.db.Create(member).Error
}

// EnsureMember ensures the user is a project member, auto-joining if not.
// Used for assignee auto-join scenarios.
// If user is already a member, no changes are made and original role is preserved.
func (s *ProjectService) EnsureMember(projectID int, userName, role string) error {
	if userName == "" {
		return nil
	}
	// Already a member, return without modifying role
	if s.GetMemberRole(projectID, userName) != "" {
		return nil
	}
	// Auto-join
	return s.AddMember(projectID, userName, role)
}

// RemoveMember removes a member from a project
func (s *ProjectService) RemoveMember(projectID int, userName string) error {
	return s.db.Where("project_id = ? AND user_name = ?", projectID, userName).
		Delete(&model.ProjectMember{}).Error
}

// UpdateMemberRole updates the role of a project member.
// role must be one of admin/member/viewer (cannot assign owner via this method).
// Returns ErrProjectForbidden if target member is currently owner (owner cannot be modified).
func (s *ProjectService) UpdateMemberRole(projectID int, userName, role string) error {
	normalized := strings.ToLower(role)
	if normalized != model.ProjectRoleAdmin && normalized != model.ProjectRoleMember && normalized != model.ProjectRoleViewer {
		return ErrInvalidRole
	}
	// Reject modifying owner (owner can only be created by project creator)
	// Check if user is project owner (in project table)
	var project model.Project
	if err := s.db.Select("owner_name").Where("id = ?", projectID).First(&project).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrProjectNotFound
		}
		return err
	}
	if project.OwnerName == userName {
		return ErrProjectForbidden
	}
	// Query member record
	var current model.ProjectMember
	if err := s.db.Select("role").Where("project_id = ? AND user_name = ?", projectID, userName).First(&current).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrProjectMemberNotFound
		}
		return err
	}
	result := s.db.Model(&model.ProjectMember{}).
		Where("project_id = ? AND user_name = ?", projectID, userName).
		Update("role", normalized)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProjectMemberNotFound
	}
	return nil
}

// GetMembers retrieves all members of a project
func (s *ProjectService) GetMembers(projectID int) ([]*model.ProjectMember, error) {
	var members []*model.ProjectMember
	err := s.db.Where("project_id = ?", projectID).Find(&members).Error
	return members, err
}

// GetMemberRole returns the user's role in the project.
// Returns "owner" for project creator; returns the member's role (normalized to lowercase) for members;
// returns empty string for non-members or when not found.
//
// This is the unified data source for all role-based permission checks, replacing the old IsProjectMember/IsProjectAdmin.
func (s *ProjectService) GetMemberRole(projectID int, userName string) string {
	if userName == "" {
		return ""
	}
	// Owner check (project creator, separate from project_member table)
	var project model.Project
	if err := s.db.Select("owner_name").Where("id = ?", projectID).First(&project).Error; err != nil {
		return ""
	}
	if project.OwnerName == userName {
		return model.ProjectRoleOwner
	}
	// Query member row
	var member model.ProjectMember
	if err := s.db.Select("role").Where("project_id = ? AND user_name = ?", projectID, userName).First(&member).Error; err != nil {
		return ""
	}
	role := strings.ToLower(member.Role)
	// Defensive: treat unknown role as no role (migrations will clean up, but this is a safety net)
	if !model.ValidProjectRole(role) {
		return ""
	}
	return role
}

// RequireRole checks if user's role level >= required.
// Returns actual role if satisfied, ErrProjectForbidden otherwise.
// Role levels: owner=4 > admin=3 > member=2 > viewer=1 > none=0.
func (s *ProjectService) RequireRole(projectID int, userName, required string) (string, error) {
	role := s.GetMemberRole(projectID, userName)
	if !model.HasProjectRoleAtLeast(role, required) {
		return "", ErrProjectForbidden
	}
	return role, nil
}

// AddRepository adds a repository to a project
func (s *ProjectService) AddRepository(projectID int, userName, repoName string) error {
	pr := &model.ProjectRepository{
		ProjectID:      projectID,
		UserName:       userName,
		RepositoryName: repoName,
	}
	return s.db.Create(pr).Error
}

// RemoveRepository removes a repository from a project
func (s *ProjectService) RemoveRepository(projectID int, userName, repoName string) error {
	return s.db.Where("project_id = ? AND user_name = ? AND repository_name = ?",
		projectID, userName, repoName).
		Delete(&model.ProjectRepository{}).Error
}

// GetRepositories retrieves all repositories of a project
func (s *ProjectService) GetRepositories(projectID int) ([]*model.ProjectRepository, error) {
	var repos []*model.ProjectRepository
	err := s.db.Where("project_id = ?", projectID).Find(&repos).Error
	return repos, err
}
