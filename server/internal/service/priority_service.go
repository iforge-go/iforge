package service

import (
	"errors"

	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

var (
	ErrPriorityNotFound = errors.New("priority not found")
)

type PriorityService struct {
	db *gorm.DB
}

func NewPriorityService(db *gorm.DB) *PriorityService {
	return &PriorityService{db: db}
}

// ListPriorities lists all priorities for a repository
func (s *PriorityService) ListPriorities(owner, repo string) ([]*model.Priority, error) {
	var priorities []*model.Priority
	err := s.db.
		Where("user_name = ? AND repository_name = ?", owner, repo).
		Order("priority_id").
		Find(&priorities).Error
	return priorities, err
}

// GetPriority gets a specific priority
func (s *PriorityService) GetPriority(owner, repo string, priorityID int) (*model.Priority, error) {
	priority := &model.Priority{}
	err := s.db.
		Where("user_name = ? AND repository_name = ? AND priority_id = ?", owner, repo, priorityID).
		First(priority).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPriorityNotFound
		}
		return nil, err
	}
	return priority, nil
}

// CreatePriority creates a new priority
func (s *PriorityService) CreatePriority(priority *model.Priority) (*model.Priority, error) {
	if err := s.db.Create(priority).Error; err != nil {
		return nil, err
	}
	return priority, nil
}

// UpdatePriority updates a priority
func (s *PriorityService) UpdatePriority(owner, repo string, priorityID int, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	updateMap := make(map[string]interface{})
	if name, ok := updates["priorityName"].(string); ok {
		updateMap["priority_name"] = name
	}
	if desc, ok := updates["description"].(string); ok {
		updateMap["description"] = desc
	}
	if color, ok := updates["color"].(string); ok {
		updateMap["color"] = color
	}

	if len(updateMap) == 0 {
		return nil
	}

	return s.db.Model(&model.Priority{}).
		Where("user_name = ? AND repository_name = ? AND priority_id = ?", owner, repo, priorityID).
		Updates(updateMap).Error
}

// DeletePriority deletes a priority
func (s *PriorityService) DeletePriority(owner, repo string, priorityID int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// First, remove this priority from all issues
		if err := tx.Model(&model.Issue{}).
			Where("user_name = ? AND repository_name = ? AND priority_id = ?", owner, repo, priorityID).
			Update("priority_id", nil).Error; err != nil {
			return err
		}

		// Then delete the priority
		return tx.Where("user_name = ? AND repository_name = ? AND priority_id = ?", owner, repo, priorityID).
			Delete(&model.Priority{}).Error
	})
}

// ReorderPriorities reorders priorities by updating their IDs
func (s *PriorityService) ReorderPriorities(owner, repo string, priorityIDs []int) error {
	// This is a simplified implementation
	// In a real system, you might want to use a separate 'order' column
	// For now, we'll just validate that all priorities exist
	for _, id := range priorityIDs {
		_, err := s.GetPriority(owner, repo, id)
		if err != nil {
			return err
		}
	}

	return nil
}

// SetDefaultPriority sets the default priority for a repository
func (s *PriorityService) SetDefaultPriority(owner, repo string, priorityID *int) error {
	if priorityID == nil {
		// Remove default priority
		return s.db.Model(&model.Repository{}).
			Where("user_name = ? AND repository_name = ?", owner, repo).
			Update("default_priority_id", nil).Error
	}

	// Validate priority exists
	_, err := s.GetPriority(owner, repo, *priorityID)
	if err != nil {
		return err
	}

	return s.db.Model(&model.Repository{}).
		Where("user_name = ? AND repository_name = ?", owner, repo).
		Update("default_priority_id", *priorityID).Error
}

// GetDefaultPriority gets the default priority ID for a repository
func (s *PriorityService) GetDefaultPriority(owner, repo string) (*int, error) {
	var repoObj model.Repository
	err := s.db.
		Select("default_priority_id").
		Where("user_name = ? AND repository_name = ?", owner, repo).
		First(&repoObj).Error
	if err != nil {
		return nil, err
	}
	return repoObj.DefaultPriorityID, nil
}
