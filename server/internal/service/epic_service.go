package service

import (
	"errors"
	"time"

	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

var (
	ErrEpicNotFound = errors.New("epic not found")
)

// EpicSprintInfo summarizes a Sprint linked to an Epic (aggregated via task.sprint_id).
type EpicSprintInfo struct {
	SprintID   int        `json:"sprintId"`
	SprintSlug string     `json:"sprintSlug"`
	Title      string     `json:"title"`
	Status     string     `json:"status"`
	TaskCount  int        `json:"taskCount"`
	StartDate  *time.Time `json:"startDate"`
	EndDate    *time.Time `json:"endDate"`
}

// EpicService handles epic-related operations
type EpicService struct {
	db *gorm.DB
}

// NewEpicService creates a new EpicService
func NewEpicService(db *gorm.DB) *EpicService {
	return &EpicService{db: db}
}

// CreateEpic creates a new epic
func (s *EpicService) CreateEpic(projectID int, title, status, priority string, description, goal *string, startDate, targetDate *time.Time, ownerName, reporterName *string) (*model.Epic, error) {
	now := time.Now()
	epic := &model.Epic{
		ProjectID:    projectID,
		Title:        title,
		Description:  description,
		Status:       status,
		StatusManual: false, // auto-derived by default
		Priority:     priority,
		Goal:         goal,
		StartDate:    startDate,
		TargetDate:   targetDate,
		OwnerName:    ownerName,
		ReporterName: derefString(reporterName),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.db.Create(epic).Error; err != nil {
		return nil, err
	}
	return epic, nil
}

// GetEpic retrieves an epic by ID, with aggregated progress filled in.
func (s *EpicService) GetEpic(id int) (*model.Epic, error) {
	var epic model.Epic
	err := s.db.Where("id = ?", id).First(&epic).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrEpicNotFound
		}
		return nil, err
	}
	s.fillAggregates(&epic)
	return &epic, nil
}

// UpdateEpic updates an epic
func (s *EpicService) UpdateEpic(epic *model.Epic) error {
	epic.UpdatedAt = time.Now()
	return s.db.Save(epic).Error
}

// ListEpics lists epics for a project, with aggregated progress filled in.
func (s *EpicService) ListEpics(projectID int, statusFilter *string, limit, offset int) ([]*model.Epic, error) {
	var epics []*model.Epic
	query := s.db.Where("project_id = ?", projectID)
	if statusFilter != nil && *statusFilter != "" {
		query = query.Where("status = ?", *statusFilter)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	err := query.Order("created_at DESC").Find(&epics).Error
	if err != nil {
		return nil, err
	}
	for _, e := range epics {
		s.fillAggregates(e)
	}
	return epics, nil
}

// DeleteEpic deletes an epic; within a transaction, sets related Story.epic_id to NULL (soft unlink).
func (s *EpicService) DeleteEpic(id int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Unlink stories from this epic
		if err := tx.Model(&model.UserStory{}).Where("epic_id = ?", id).Update("epic_id", nil).Error; err != nil {
			return err
		}
		// Delete epic
		if err := tx.Where("id = ?", id).Delete(&model.Epic{}).Error; err != nil {
			return err
		}
		return nil
	})
}

// AssignStoryToEpic associates a story with an epic (overwrites the previous epic_id).
func (s *EpicService) AssignStoryToEpic(projectID, epicID, storyID int) error {
	result := s.db.Model(&model.UserStory{}).
		Where("id = ? AND project_id = ?", storyID, projectID).
		Update("epic_id", epicID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserStoryNotFound
	}
	return nil
}

// RemoveStoryFromEpic removes a story from its epic (sets epic_id to NULL).
func (s *EpicService) RemoveStoryFromEpic(projectID, storyID int) error {
	result := s.db.Model(&model.UserStory{}).
		Where("id = ? AND project_id = ?", storyID, projectID).
		Update("epic_id", nil)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserStoryNotFound
	}
	return nil
}

// ListEpicStories lists all stories belonging to an epic
func (s *EpicService) ListEpicStories(projectID, epicID int) ([]*model.UserStory, error) {
	var stories []*model.UserStory
	err := s.db.Where("project_id = ? AND epic_id = ?", projectID, epicID).
		Order("created_at DESC").
		Find(&stories).Error
	return stories, err
}

// GetEpicSprints aggregates the Sprint list linked to an Epic (via task.user_story_id -> story.epic_id).
// Returns each Sprint's summary plus the task count of the Epic within that Sprint.
func (s *EpicService) GetEpicSprints(projectID, epicID int) ([]EpicSprintInfo, error) {
	var infos []EpicSprintInfo
	err := s.db.Raw(`
		SELECT s.id AS sprint_id, s.title, s.status, s.start_date, s.end_date,
		       COUNT(*) AS task_count
		FROM sprint s
		INNER JOIN task t ON t.sprint_id = s.id
		INNER JOIN user_story us ON us.id = t.user_story_id
		WHERE us.project_id = ? AND us.epic_id = ?
		GROUP BY s.id, s.title, s.status, s.start_date, s.end_date
		ORDER BY s.start_date DESC, s.id DESC
	`, projectID, epicID).Scan(&infos).Error
	if err != nil {
		return nil, err
	}
	// SprintSlug is filled by the handler layer (hashid)
	return infos, nil
}

// fillAggregates fills the Epic's aggregated fields (TotalStories/DoneStories/Points/Progress/SprintCount)
// and lazily derives Status (only when StatusManual=false).
func (s *EpicService) fillAggregates(epic *model.Epic) {
	type storyAgg struct {
		TotalStories int
		DoneStories  int
		TotalPoints  int
		DonePoints   int
	}
	var agg storyAgg
	// both done and closed count as completed
	err := s.db.Model(&model.UserStory{}).
		Select(`
			COUNT(*) AS total_stories,
			SUM(CASE WHEN status IN ('done','closed') THEN 1 ELSE 0 END) AS done_stories,
			COALESCE(SUM(story_points), 0) AS total_points,
			COALESCE(SUM(CASE WHEN status IN ('done','closed') THEN COALESCE(story_points,0) ELSE 0 END), 0) AS done_points
		`).
		Where("epic_id = ?", epic.ID).
		Scan(&agg).Error
	if err == nil {
		epic.TotalStories = agg.TotalStories
		epic.DoneStories = agg.DoneStories
		epic.TotalStoryPoints = agg.TotalPoints
		epic.DoneStoryPoints = agg.DonePoints
		if epic.TotalStories > 0 {
			epic.ProgressPercent = epic.DoneStories * 100 / epic.TotalStories
		} else {
			epic.ProgressPercent = 0
		}
	}

	// SprintCount: how many distinct Sprints the Epic's stories' tasks span
	var sprintCount int64
	s.db.Raw(`
		SELECT COUNT(DISTINCT t.sprint_id)
		FROM task t
		INNER JOIN user_story us ON us.id = t.user_story_id
		WHERE us.epic_id = ? AND t.sprint_id IS NOT NULL
	`, epic.ID).Scan(&sprintCount)
	epic.SprintCount = int(sprintCount)

	// Lazily derive Status: only when not manually set
	if !epic.StatusManual {
		var statuses []string
		s.db.Model(&model.UserStory{}).
			Where("epic_id = ?", epic.ID).
			Pluck("status", &statuses)
		epic.Status = s.deriveStatus(statuses)
	}
}

// deriveStatus derives the Epic status from its stories' statuses:
//   - All stories done/closed -> done
//   - Any story in_progress -> in_progress
//   - All stories closed -> closed
//   - Otherwise -> open
//
// Note: returns open when there are no stories (newly created empty Epic).
func (s *EpicService) deriveStatus(storyStatuses []string) string {
	if len(storyStatuses) == 0 {
		return "open"
	}
	allClosed := true
	allDone := true
	for _, st := range storyStatuses {
		if st == "in_progress" {
			return "in_progress"
		}
		if st != "closed" {
			allClosed = false
		}
		if st != "done" && st != "closed" {
			allDone = false
		}
	}
	if allClosed {
		return "closed"
	}
	if allDone {
		return "done"
	}
	return "open"
}

// derefString safely dereferences a *string, returning "" if nil.
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
