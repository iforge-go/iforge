package service

import (
	"errors"

	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

var (
	ErrCollaboratorNotFound = errors.New("collaborator not found")
	ErrCollaboratorExists   = errors.New("collaborator already exists")
)

// CollaboratorService handles collaborator operations
type CollaboratorService struct {
	db *gorm.DB
}

// NewCollaboratorService creates a new CollaboratorService
func NewCollaboratorService(db *gorm.DB) *CollaboratorService {
	return &CollaboratorService{db: db}
}

// AddCollaborator adds a collaborator to a repository
func (s *CollaboratorService) AddCollaborator(owner, repo, collaboratorName, role string) (*model.Collaborator, error) {
	// Check if collaborator already exists
	var count int64
	err := s.db.Model(&model.Collaborator{}).
		Where("user_name = ? AND repository_name = ? AND collaborator_name = ?", owner, repo, collaboratorName).
		Count(&count).Error

	if err != nil {
		return nil, err
	}

	if count > 0 {
		return nil, ErrCollaboratorExists
	}

	collaborator := &model.Collaborator{
		UserName:         owner,
		RepositoryName:   repo,
		CollaboratorName: collaboratorName,
		Role:             role,
	}

	err = s.db.Create(collaborator).Error
	if err != nil {
		return nil, err
	}

	return collaborator, nil
}

// ListCollaborators lists all collaborators for a repository, joined with account info
func (s *CollaboratorService) ListCollaborators(owner, repo string) ([]*model.CollaboratorWithUser, error) {
	var collaborators []*model.CollaboratorWithUser
	err := s.db.Table("collaborator").
		Select("collaborator.user_name, collaborator.repository_name, collaborator.collaborator_name, collaborator.role, account.full_name, account.image, account.mail_address, account.is_organization").
		Joins("LEFT JOIN account ON account.user_name = collaborator.collaborator_name").
		Where("collaborator.user_name = ? AND collaborator.repository_name = ?", owner, repo).
		Find(&collaborators).Error
	return collaborators, err
}

// RemoveCollaborator removes a collaborator from a repository
func (s *CollaboratorService) RemoveCollaborator(owner, repo, collaboratorName string) error {
	result := s.db.
		Where("user_name = ? AND repository_name = ? AND collaborator_name = ?", owner, repo, collaboratorName).
		Delete(&model.Collaborator{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrCollaboratorNotFound
	}

	return nil
}

// IsCollaborator checks if a user is a collaborator of a repository
func (s *CollaboratorService) IsCollaborator(owner, repo, userName string) (bool, error) {
	var count int64
	err := s.db.Model(&model.Collaborator{}).
		Where("user_name = ? AND repository_name = ? AND collaborator_name = ?", owner, repo, userName).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetCollaboratorRole returns the role of a user for the given repository.
// Returns empty string (no error) if the user is not a collaborator.
func (s *CollaboratorService) GetCollaboratorRole(owner, repo, userName string) (string, error) {
	var collaborator model.Collaborator
	err := s.db.
		Where("user_name = ? AND repository_name = ? AND collaborator_name = ?", owner, repo, userName).
		Select("role").
		First(&collaborator).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return collaborator.Role, nil
}

// UpdateCollaboratorRole updates a collaborator's role
func (s *CollaboratorService) UpdateCollaboratorRole(owner, repo, collaboratorName, role string) error {
	result := s.db.Model(&model.Collaborator{}).
		Where("user_name = ? AND repository_name = ? AND collaborator_name = ?", owner, repo, collaboratorName).
		Update("role", role)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrCollaboratorNotFound
	}

	return nil
}
