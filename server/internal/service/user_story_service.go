package service

import (
	"errors"
	"time"

	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

var (
	ErrUserStoryNotFound = errors.New("user story not found")
)

// UserStoryService handles user story-related operations
type UserStoryService struct {
	db *gorm.DB
}

// NewUserStoryService creates a new UserStoryService
func NewUserStoryService(db *gorm.DB) *UserStoryService {
	return &UserStoryService{
		db: db,
	}
}

// CreateUserStory creates a new user story in the product backlog.
// epicID is nullable: orphan Stories in Backlog don't belong to any Epic.
// sprintID is nullable: non-nil means planned into that Sprint at creation time (Story-level Sprint planning).
func (s *UserStoryService) CreateUserStory(projectID int, title, status, priority string, description, acceptanceCriteria *string, storyPoints *int, assigneeName, reporterName *string, epicID, sprintID *int) (*model.UserStory, error) {
	now := time.Now()
	userStory := &model.UserStory{
		ProjectID:          projectID,
		Title:              title,
		Description:        description,
		Status:             status,
		Priority:           priority,
		StoryPoints:        storyPoints,
		AcceptanceCriteria: acceptanceCriteria,
		AssigneeName:       assigneeName,
		ReporterName:       *reporterName,
		EpicID:             epicID,
		SprintID:           sprintID,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := s.db.Create(userStory).Error; err != nil {
		return nil, err
	}

	return userStory, nil
}

// GetUserStory retrieves a user story by ID
func (s *UserStoryService) GetUserStory(id int) (*model.UserStory, error) {
	var userStory model.UserStory
	err := s.db.Where("id = ?", id).First(&userStory).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrUserStoryNotFound
		}
		return nil, err
	}
	return &userStory, nil
}

// UpdateUserStory updates a user story
func (s *UserStoryService) UpdateUserStory(userStory *model.UserStory) error {
	userStory.UpdatedAt = time.Now()
	return s.db.Save(userStory).Error
}

// ListUserStories lists user stories for a project.
// epicID filter: when non-nil, only returns Stories for that Epic; when nil, returns all Stories.
// sprintID filter: when non-nil, only returns Stories planned into that Sprint (Story-level Sprint planning).
// backlog:true returns only Stories with sprint_id IS NULL (Stories in requirements pool not planned into any Sprint).
func (s *UserStoryService) ListUserStories(projectID int, status *string, epicID, sprintID *int, backlog bool, limit, offset int) ([]*model.UserStory, error) {
	var userStories []*model.UserStory
	query := s.db.Where("project_id = ?", projectID)

	if status != nil && *status != "" {
		query = query.Where("status = ?", *status)
	}
	if epicID != nil {
		query = query.Where("epic_id = ?", *epicID)
	}
	if backlog {
		query = query.Where("sprint_id IS NULL")
	} else if sprintID != nil {
		query = query.Where("sprint_id = ?", *sprintID)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Order("created_at DESC").Find(&userStories).Error
	return userStories, err
}

// DeleteUserStory deletes a user story
func (s *UserStoryService) DeleteUserStory(id int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Unlink tasks from this user story
		if err := tx.Model(&model.Task{}).Where("user_story_id = ?", id).Update("user_story_id", nil).Error; err != nil {
			return err
		}

		// Delete user story
		if err := tx.Where("id = ?", id).Delete(&model.UserStory{}).Error; err != nil {
			return err
		}

		return nil
	})
}

// UpdateStatus updates user story status
func (s *UserStoryService) UpdateStatus(id int, status string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	if status == "closed" {
		now := time.Now()
		updates["closed_at"] = &now
	} else {
		updates["closed_at"] = nil
	}

	return s.db.Model(&model.UserStory{}).Where("id = ?", id).Updates(updates).Error
}

// GetUserStoryTasks retrieves all tasks for a user story
func (s *UserStoryService) GetUserStoryTasks(userStoryID int) ([]*model.Task, error) {
	var tasks []*model.Task
	err := s.db.Where("user_story_id = ?", userStoryID).Order("position ASC, created_at DESC").Find(&tasks).Error
	return tasks, err
}

// GetStoryTaskCounts batch-queries task counts for user_stories.
// Returns map[storyID]int; stories without tasks are omitted (callers treat as 0).
// Used to display task count badges and breakdown entry hints on backlog story rows, avoiding N+1.
func (s *UserStoryService) GetStoryTaskCounts(projectID int, storyIDs []int) (map[int]int, error) {
	result := make(map[int]int)
	if len(storyIDs) == 0 {
		return result, nil
	}

	type row struct {
		UserStoryID int `gorm:"column:user_story_id"`
		Total       int `gorm:"column:total"`
	}

	var rows []row
	err := s.db.Raw(`
		SELECT user_story_id, COUNT(*) as total
		FROM task
		WHERE project_id = ? AND user_story_id IN ?
		GROUP BY user_story_id`,
		projectID, storyIDs).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, r := range rows {
		result[r.UserStoryID] = r.Total
	}
	return result, nil
}

// GetStoriesWithTasksInSprint retrieves user stories that have at least one task in the given sprint.
// Stories themselves are not assigned to sprints; this finds stories whose tasks are in the sprint.
func (s *UserStoryService) GetStoriesWithTasksInSprint(projectID, sprintID int) ([]*model.UserStory, error) {
	var stories []*model.UserStory
	err := s.db.Raw(`
		SELECT DISTINCT us.* FROM user_story us
		INNER JOIN task t ON t.user_story_id = us.id
		WHERE us.project_id = ? AND t.sprint_id = ?
		ORDER BY us.created_at DESC
	`, projectID, sprintID).Scan(&stories).Error
	return stories, err
}

// ListStoriesByEpic lists all stories belonging to an epic
func (s *UserStoryService) ListStoriesByEpic(projectID, epicID int) ([]*model.UserStory, error) {
	var stories []*model.UserStory
	err := s.db.Where("project_id = ? AND epic_id = ?", projectID, epicID).
		Order("created_at DESC").
		Find(&stories).Error
	return stories, err
}

// UnlinkStoriesFromEpic sets epic_id=NULL for all stories of an epic.
// Used in transactions: called internally by DeleteEpic to ensure atomic soft-delete of associations.
func (s *UserStoryService) UnlinkStoriesFromEpic(epicID int) error {
	return s.db.Model(&model.UserStory{}).
		Where("epic_id = ?", epicID).
		Update("epic_id", nil).Error
}
