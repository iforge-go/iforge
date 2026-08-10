package service

import (
	"strconv"
	"time"

	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

// ScrumActivityService handles Scrum activity log operations
type ScrumActivityService struct {
	db *gorm.DB
}

// NewScrumActivityService creates a new ScrumActivityService
func NewScrumActivityService(db *gorm.DB) *ScrumActivityService {
	return &ScrumActivityService{db: db}
}

// LogActivity records a Scrum activity
func (s *ScrumActivityService) LogActivity(userName string, projectID int, entityType string, entityID int, action string, field string, oldValue string, newValue string) error {
	activity := &model.ScrumActivity{
		UserName:   userName,
		ProjectID:  projectID,
		EntityType: entityType,
		EntityID:   entityID,
		Action:     action,
		Field:      field,
		OldValue:   oldValue,
		NewValue:   newValue,
		CreatedAt:  time.Now(),
	}
	return s.db.Create(activity).Error
}

// GetActivities retrieves activities for a specific entity scoped to a project.
// projectID is required because entity_id for tasks is a composite key
// (project_id + task_id); filtering by project_id avoids cross-project bleed-through.
//
// Ordering uses `created_at DESC, id DESC`: the id tiebreaker ensures that
// activities inserted in the same second (e.g. an MR event and the status change
// it triggers) appear in a deterministic, causally-correct order — the later-
// inserted row (the effect, e.g. status_changed) always sorts above the earlier-
// inserted row (the cause, e.g. mr_merged). Without the tiebreaker, same-second
// rows sort non-deterministically and the timeline can show an effect above its
// cause or vice versa.
func (s *ScrumActivityService) GetActivities(projectID int, entityType string, entityID int) ([]*model.ScrumActivity, error) {
	var activities []*model.ScrumActivity
	err := s.db.Where("project_id = ? AND entity_type = ? AND entity_id = ?", projectID, entityType, entityID).
		Order("created_at DESC, id DESC").Find(&activities).Error
	if activities == nil {
		activities = make([]*model.ScrumActivity, 0)
	}
	if err != nil {
		return activities, err
	}
	// Enrich sprintId/userStoryId fields: convert numeric IDs to readable titles
	s.enrichActivityValues(activities)
	return activities, nil
}

// enrichActivityValues converts numeric ID values for sprintId/userStoryId fields to readable titles (batch query)
func (s *ScrumActivityService) enrichActivityValues(activities []*model.ScrumActivity) {
	sprintIDs := make(map[int]bool)
	storyIDs := make(map[int]bool)
	for _, a := range activities {
		if a.Field == "sprintId" {
			if id, err := strconv.Atoi(a.OldValue); err == nil {
				sprintIDs[id] = true
			}
			if id, err := strconv.Atoi(a.NewValue); err == nil {
				sprintIDs[id] = true
			}
		}
		if a.Field == "userStoryId" {
			if id, err := strconv.Atoi(a.OldValue); err == nil {
				storyIDs[id] = true
			}
			if id, err := strconv.Atoi(a.NewValue); err == nil {
				storyIDs[id] = true
			}
		}
	}
	// Batch query sprint titles
	sprintTitles := make(map[int]string)
	if len(sprintIDs) > 0 {
		var sprints []model.Sprint
		ids := make([]int, 0, len(sprintIDs))
		for id := range sprintIDs {
			ids = append(ids, id)
		}
		if err := s.db.Select("id, title").Where("id IN ?", ids).Find(&sprints).Error; err == nil {
			for _, sp := range sprints {
				sprintTitles[sp.ID] = sp.Title
			}
		}
	}
	// Batch query story titles
	storyTitles := make(map[int]string)
	if len(storyIDs) > 0 {
		var stories []model.UserStory
		ids := make([]int, 0, len(storyIDs))
		for id := range storyIDs {
			ids = append(ids, id)
		}
		if err := s.db.Select("id, title").Where("id IN ?", ids).Find(&stories).Error; err == nil {
			for _, st := range stories {
				storyTitles[st.ID] = st.Title
			}
		}
	}
	// Replace numeric values with titles
	for _, a := range activities {
		if a.Field == "sprintId" {
			if id, err := strconv.Atoi(a.OldValue); err == nil {
				if title, ok := sprintTitles[id]; ok {
					a.OldValue = title
				}
			}
			if id, err := strconv.Atoi(a.NewValue); err == nil {
				if title, ok := sprintTitles[id]; ok {
					a.NewValue = title
				}
			}
		}
		if a.Field == "userStoryId" {
			if id, err := strconv.Atoi(a.OldValue); err == nil {
				if title, ok := storyTitles[id]; ok {
					a.OldValue = title
				}
			}
			if id, err := strconv.Atoi(a.NewValue); err == nil {
				if title, ok := storyTitles[id]; ok {
					a.NewValue = title
				}
			}
		}
	}
}
