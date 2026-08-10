package service

import (
	"errors"

	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

var (
	ErrLabelNotFound    = errors.New("label not found")
	ErrInvalidLabelName = errors.New("invalid label name")
)

// LabelService handles label operations
type LabelService struct {
	db *gorm.DB
}

// NewLabelService creates a new LabelService
func NewLabelService(db *gorm.DB) *LabelService {
	return &LabelService{db: db}
}

// CreateLabel creates a new label
func (s *LabelService) CreateLabel(owner, repo, name, color string) (*model.Label, error) {
	if err := validateLabelName(name); err != nil {
		return nil, err
	}

	label := &model.Label{
		UserName:       owner,
		RepositoryName: repo,
		LabelName:      name,
		Color:          color,
	}

	if err := s.db.Create(label).Error; err != nil {
		return nil, err
	}

	return label, nil
}

// ListLabels lists all labels for a repository
func (s *LabelService) ListLabels(owner, repo string) ([]*model.Label, error) {
	var labels []*model.Label
	err := s.db.
		Where("user_name = ? AND repository_name = ?", owner, repo).
		Order("label_name").
		Find(&labels).Error
	return labels, err
}

// GetLabel gets a label by ID
func (s *LabelService) GetLabel(owner, repo string, labelID int) (*model.Label, error) {
	var label model.Label
	err := s.db.
		Where("user_name = ? AND repository_name = ? AND label_id = ?", owner, repo, labelID).
		First(&label).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrLabelNotFound
		}
		return nil, err
	}
	return &label, nil
}

// UpdateLabel updates a label
func (s *LabelService) UpdateLabel(owner, repo string, labelID int, name, color string) error {
	return s.db.Model(&model.Label{}).
		Where("user_name = ? AND repository_name = ? AND label_id = ?", owner, repo, labelID).
		Updates(map[string]interface{}{"label_name": name, "color": color}).Error
}

// DeleteLabel deletes a label
func (s *LabelService) DeleteLabel(owner, repo string, labelID int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Where("user_name = ? AND repository_name = ? AND label_id = ?", owner, repo, labelID).
			Delete(&model.IssueLabel{}).Error; err != nil {
			return err
		}
		return tx.
			Where("user_name = ? AND repository_name = ? AND label_id = ?", owner, repo, labelID).
			Delete(&model.Label{}).Error
	})
}

// AddLabelToIssue adds a label to an issue
func (s *LabelService) AddLabelToIssue(owner, repo string, issueID, labelID int) error {
	// Check if already exists
	var count int64
	s.db.Model(&model.IssueLabel{}).
		Where("user_name = ? AND repository_name = ? AND issue_id = ? AND label_id = ?",
			owner, repo, issueID, labelID).
		Count(&count)
	if count > 0 {
		return nil
	}

	issueLabel := &model.IssueLabel{
		UserName:       owner,
		RepositoryName: repo,
		IssueID:        issueID,
		LabelID:        labelID,
	}
	return s.db.Create(issueLabel).Error
}

// RemoveLabelFromIssue removes a label from an issue
func (s *LabelService) RemoveLabelFromIssue(owner, repo string, issueID, labelID int) error {
	return s.db.
		Where("user_name = ? AND repository_name = ? AND issue_id = ? AND label_id = ?",
			owner, repo, issueID, labelID).
		Delete(&model.IssueLabel{}).Error
}

// GetIssueLabels gets all labels for an issue
func (s *LabelService) GetIssueLabels(owner, repo string, issueID int) ([]*model.Label, error) {
	var labels []*model.Label
	err := s.db.
		Table("label").
		Select("label.*").
		Joins("INNER JOIN issue_label ON label.label_id = issue_label.label_id").
		Where("issue_label.user_name = ? AND issue_label.repository_name = ? AND issue_label.issue_id = ?",
			owner, repo, issueID).
		Order("label.label_name").
		Find(&labels).Error
	return labels, err
}

// validateLabelName validates a label name
func validateLabelName(name string) error {
	if name == "" {
		return ErrInvalidLabelName
	}
	if len(name) > 100 {
		return ErrInvalidLabelName
	}
	return nil
}
