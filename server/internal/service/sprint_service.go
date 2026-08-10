package service

import (
	"errors"
	"time"

	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

var (
	ErrSprintNotFound = errors.New("sprint not found")
)

// SprintService handles sprint-related operations
type SprintService struct {
	db *gorm.DB
}

// NewSprintService creates a new SprintService
func NewSprintService(db *gorm.DB) *SprintService {
	return &SprintService{
		db: db,
	}
}

// CreateSprint creates a new sprint
func (s *SprintService) CreateSprint(projectID int, title, status string, description, goal *string, startDate, endDate *time.Time, createdBy string) (*model.Sprint, error) {
	now := time.Now()
	sprint := &model.Sprint{
		ProjectID:   projectID,
		Title:       title,
		Description: description,
		Status:      status,
		Goal:        goal,
		StartDate:   startDate,
		EndDate:     endDate,
		CreatedBy:   createdBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.db.Create(sprint).Error; err != nil {
		return nil, err
	}

	return sprint, nil
}

// GetSprint retrieves a sprint by ID
func (s *SprintService) GetSprint(id int) (*model.Sprint, error) {
	var sprint model.Sprint
	err := s.db.Where("id = ?", id).First(&sprint).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrSprintNotFound
		}
		return nil, err
	}
	return &sprint, nil
}

// UpdateSprint updates a sprint
func (s *SprintService) UpdateSprint(sprint *model.Sprint) error {
	sprint.UpdatedAt = time.Now()
	return s.db.Save(sprint).Error
}

// ListSprints lists sprints for a project.
// Batch-fill statistics (TaskCount/StoryCount/TotalStoryPoints) for each sprint, consistent with story-level sprint planning:
// TaskCount includes inherited tasks from parent stories in this sprint, StoryCount/TotalStoryPoints based on direct story.sprint_id association.
func (s *SprintService) ListSprints(projectID int, limit, offset int) ([]*model.Sprint, error) {
	var sprints []*model.Sprint
	query := s.db.Where("project_id = ?", projectID)

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Order("created_at DESC").Find(&sprints).Error; err != nil {
		return sprints, err
	}

	// Batch-fill statistics. Sprint count is usually small (<20), so per-sprint aggregation is acceptable;
	// task_count uses compatible scope (direct assignment + parent story inheritance), excludes subtasks (parent_id IS NULL, consistent with ListTasks).
	for _, sp := range sprints {
		var stats struct {
			TaskCount  int
			StoryCount int
			Points     int
		}
		s.db.Raw(`SELECT
			(SELECT COUNT(*) FROM task WHERE parent_id IS NULL AND (sprint_id = ? OR user_story_id IN (SELECT id FROM user_story WHERE sprint_id = ?))) AS task_count,
			(SELECT COUNT(*) FROM user_story WHERE sprint_id = ?) AS story_count,
			(SELECT COALESCE(SUM(story_points), 0) FROM user_story WHERE sprint_id = ?) AS points`,
			sp.ID, sp.ID, sp.ID, sp.ID).Scan(&stats)
		sp.TaskCount = stats.TaskCount
		sp.StoryCount = stats.StoryCount
		sp.TotalStoryPoints = stats.Points
	}

	return sprints, nil
}

// DeleteSprint deletes a sprint
func (s *SprintService) DeleteSprint(id int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Unlink tasks from this sprint
		if err := tx.Model(&model.Task{}).Where("sprint_id = ?", id).Update("sprint_id", nil).Error; err != nil {
			return err
		}

		// Delete sprint
		if err := tx.Where("id = ?", id).Delete(&model.Sprint{}).Error; err != nil {
			return err
		}

		return nil
	})
}

// UpdateStatus updates sprint status.
// Accepts both "closed" and "completed" for backward compatibility.
func (s *SprintService) UpdateStatus(id int, status string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	if status == "closed" || status == "completed" {
		now := time.Now()
		updates["completed_date"] = &now
	}

	return s.db.Model(&model.Sprint{}).Where("id = ?", id).Updates(updates).Error
}

// VelocityDataPoint represents velocity data for a closed sprint (committed vs completed points).
type VelocityDataPoint struct {
	SprintSlug      string     `json:"sprintSlug"`
	Title           string     `json:"title"`
	CommittedPoints int        `json:"committedPoints"` // Total story points in sprint
	CompletedPoints int        `json:"completedPoints"` // Completed story points (status=done/closed/completed)
	CompletedDate   *time.Time `json:"completedDate"`
	SprintID        int        `json:"-"`
}

// GetVelocity returns velocity data for the last N closed sprints, ordered chronologically.
func (s *SprintService) GetVelocity(projectID int, count int) ([]VelocityDataPoint, error) {
	if count <= 0 {
		count = 6
	}
	var sprints []*model.Sprint
	err := s.db.Where("project_id = ? AND status IN ('closed', 'completed')", projectID).
		Order("completed_date DESC").
		Limit(count).
		Find(&sprints).Error
	if err != nil {
		return nil, err
	}

	// Reverse to chronological order (oldest to newest) for chart x-axis left-to-right
	result := make([]VelocityDataPoint, 0, len(sprints))
	for i := len(sprints) - 1; i >= 0; i-- {
		sp := sprints[i]
		var stats struct {
			Committed int
			Completed int
		}
		s.db.Raw(`SELECT
			(SELECT COALESCE(SUM(story_points), 0) FROM user_story WHERE sprint_id = ?) AS committed,
			(SELECT COALESCE(SUM(story_points), 0) FROM user_story WHERE sprint_id = ? AND status IN ('done', 'closed', 'completed')) AS completed`,
			sp.ID, sp.ID).Scan(&stats)
		result = append(result, VelocityDataPoint{
			SprintID:        sp.ID,
			Title:           sp.Title,
			CommittedPoints: stats.Committed,
			CompletedPoints: stats.Completed,
			CompletedDate:   sp.CompletedDate,
		})
	}
	return result, nil
}

// SprintScopeChange represents a sprint scope change event (item added/removed from sprint).
// Aligned with Jira Sprint Report scope change timeline.
type SprintScopeChange struct {
	UserName  string    `json:"userName"`
	ItemType  string    `json:"itemType"` // "story" or "task"
	ItemTitle string    `json:"itemTitle"`
	Action    string    `json:"action"` // sprint_scope_added / sprint_scope_removed
	CreatedAt time.Time `json:"createdAt"`
}

// SprintReport summarizes scope changes and committed/completed statistics for a sprint.
type SprintReport struct {
	CommittedPoints int                 `json:"committedPoints"`
	CompletedPoints int                 `json:"completedPoints"`
	ScopeChanges    []SprintScopeChange `json:"scopeChanges"`
}

// GetSprintReport returns sprint report data: committed vs completed points + scope change timeline.
func (s *SprintService) GetSprintReport(projectID, sprintID int) (*SprintReport, error) {
	var stats struct {
		Committed int
		Completed int
	}
	s.db.Raw(`SELECT
		(SELECT COALESCE(SUM(story_points), 0) FROM user_story WHERE sprint_id = ?) AS committed,
		(SELECT COALESCE(SUM(story_points), 0) FROM user_story WHERE sprint_id = ? AND status IN ('done', 'closed', 'completed')) AS completed`,
		sprintID, sprintID).Scan(&stats)

	var activities []*model.ScrumActivity
	s.db.Where("entity_type = ? AND entity_id = ? AND action IN ?",
		"sprint", sprintID, []string{"sprint_scope_added", "sprint_scope_removed"}).
		Order("created_at DESC, id DESC").
		Find(&activities)

	changes := make([]SprintScopeChange, 0, len(activities))
	for _, a := range activities {
		changes = append(changes, SprintScopeChange{
			UserName:  a.UserName,
			ItemType:  a.Field,
			ItemTitle: a.NewValue,
			Action:    a.Action,
			CreatedAt: a.CreatedAt,
		})
	}

	return &SprintReport{
		CommittedPoints: stats.Committed,
		CompletedPoints: stats.Completed,
		ScopeChanges:    changes,
	}, nil
}

// GetSprintTasks retrieves all tasks in a sprint.
// Supports two sprint assignment scopes: task directly belongs to sprint (task.sprint_id),
// or task's parent story belongs to sprint (story.sprint_id, task inherits during story-level planning).
func (s *SprintService) GetSprintTasks(sprintID int) ([]*model.Task, error) {
	var tasks []*model.Task
	err := s.db.Where("sprint_id = ? OR user_story_id IN (SELECT id FROM user_story WHERE sprint_id = ?)", sprintID, sprintID).
		Order("position ASC, created_at DESC").Find(&tasks).Error
	return tasks, err
}

// GetSprintStories retrieves user stories directly assigned to the given sprint.
// Story-level sprint planning: story.sprint_id directly associates with sprint (aligned with Jira).
func (s *SprintService) GetSprintStories(projectID, sprintID int) ([]*model.UserStory, error) {
	var stories []*model.UserStory
	err := s.db.Where("project_id = ? AND sprint_id = ?", projectID, sprintID).
		Order("created_at DESC").
		Find(&stories).Error
	return stories, err
}
