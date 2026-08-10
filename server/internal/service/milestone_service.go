package service

import (
	"errors"
	"time"

	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

var (
	ErrMilestoneNotFound = errors.New("milestone not found")
	ErrInvalidMilestone  = errors.New("invalid milestone")
)

// MilestoneService handles milestone operations
type MilestoneService struct {
	db *gorm.DB
}

// NewMilestoneService creates a new MilestoneService
func NewMilestoneService(db *gorm.DB) *MilestoneService {
	return &MilestoneService{db: db}
}

// CreateMilestone creates a new milestone
func (s *MilestoneService) CreateMilestone(owner, repo, title, description string, dueDate *time.Time) (*model.Milestone, error) {
	milestone := &model.Milestone{
		UserName:       owner,
		RepositoryName: repo,
		Title:          title,
		Description:    &description,
		DueDate:        dueDate,
	}

	if err := s.db.Create(milestone).Error; err != nil {
		return nil, err
	}

	return milestone, nil
}

// ListMilestones lists all milestones for a repository
func (s *MilestoneService) ListMilestones(owner, repo string) ([]*model.Milestone, error) {
	var milestones []*model.Milestone
	err := s.db.
		Where("user_name = ? AND repository_name = ?", owner, repo).
		Order("milestone_id DESC").
		Find(&milestones).Error
	return milestones, err
}

// GetMilestone gets a milestone by ID
func (s *MilestoneService) GetMilestone(owner, repo string, milestoneID int) (*model.Milestone, error) {
	milestone := &model.Milestone{}
	err := s.db.
		Where("user_name = ? AND repository_name = ? AND milestone_id = ?", owner, repo, milestoneID).
		First(milestone).Error

	if err == gorm.ErrRecordNotFound {
		return nil, ErrMilestoneNotFound
	}
	if err != nil {
		return nil, err
	}

	return milestone, nil
}

// UpdateMilestone updates a milestone
func (s *MilestoneService) UpdateMilestone(owner, repo string, milestoneID int, title, description string, dueDate *time.Time) error {
	return s.db.
		Model(&model.Milestone{}).
		Where("user_name = ? AND repository_name = ? AND milestone_id = ?", owner, repo, milestoneID).
		Updates(map[string]interface{}{
			"title":       title,
			"description": description,
			"due_date":    dueDate,
		}).Error
}

// DeleteMilestone deletes a milestone
func (s *MilestoneService) DeleteMilestone(owner, repo string, milestoneID int) error {
	// Use transaction to ensure atomicity
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Unlink issues from this milestone
		if err := tx.Model(&model.Issue{}).
			Where("user_name = ? AND repository_name = ? AND milestone_id = ?", owner, repo, milestoneID).
			Update("milestone_id", nil).Error; err != nil {
			return err
		}

		// Delete the milestone
		return tx.
			Where("user_name = ? AND repository_name = ? AND milestone_id = ?", owner, repo, milestoneID).
			Delete(&model.Milestone{}).Error
	})
}

// CloseMilestone closes a milestone
func (s *MilestoneService) CloseMilestone(owner, repo string, milestoneID int) error {
	now := time.Now()
	return s.db.
		Model(&model.Milestone{}).
		Where("user_name = ? AND repository_name = ? AND milestone_id = ?", owner, repo, milestoneID).
		Update("closed_date", now).Error
}

// ReopenMilestone reopens a milestone
func (s *MilestoneService) ReopenMilestone(owner, repo string, milestoneID int) error {
	return s.db.
		Model(&model.Milestone{}).
		Where("user_name = ? AND repository_name = ? AND milestone_id = ?", owner, repo, milestoneID).
		Update("closed_date", nil).Error
}

// GetMilestoneIssues gets all issues for a milestone
func (s *MilestoneService) GetMilestoneIssues(owner, repo string, milestoneID int) ([]*model.Issue, error) {
	var issues []*model.Issue
	err := s.db.
		Where("user_name = ? AND repository_name = ? AND milestone_id = ?", owner, repo, milestoneID).
		Order("registered_date DESC").
		Find(&issues).Error
	return issues, err
}
