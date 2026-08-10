package service

import (
	"errors"

	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

var (
	ErrCustomFieldNotFound = errors.New("custom field not found")
)

type CustomFieldService struct {
	db *gorm.DB
}

func NewCustomFieldService(db *gorm.DB) *CustomFieldService {
	return &CustomFieldService{db: db}
}

// ListCustomFields lists all custom fields for a repository
func (s *CustomFieldService) ListCustomFields(owner, repo string) ([]*model.CustomField, error) {
	var fields []*model.CustomField
	err := s.db.
		Where("user_name = ? AND repository_name = ?", owner, repo).
		Order("field_id").
		Find(&fields).Error

	return fields, err
}

// GetCustomField gets a specific custom field
func (s *CustomFieldService) GetCustomField(owner, repo string, fieldID int) (*model.CustomField, error) {
	field := &model.CustomField{}
	err := s.db.
		Where("user_name = ? AND repository_name = ? AND field_id = ?", owner, repo, fieldID).
		First(field).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCustomFieldNotFound
		}
		return nil, err
	}

	return field, nil
}

// CreateCustomField creates a new custom field
func (s *CustomFieldService) CreateCustomField(field *model.CustomField) (*model.CustomField, error) {
	if err := s.db.Create(field).Error; err != nil {
		return nil, err
	}

	return field, nil
}

// UpdateCustomField updates a custom field
func (s *CustomFieldService) UpdateCustomField(owner, repo string, fieldID int, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	gormUpdates := make(map[string]interface{})

	if name, ok := updates["fieldName"].(string); ok {
		gormUpdates["field_name"] = name
	}
	if fieldType, ok := updates["fieldType"].(string); ok {
		gormUpdates["field_type"] = fieldType
	}
	if constraints, ok := updates["constraints"].(string); ok {
		gormUpdates["constraints"] = constraints
	}
	if enableForIssues, ok := updates["enableForIssues"].(bool); ok {
		gormUpdates["enable_for_issues"] = enableForIssues
	}
	if enableForPRs, ok := updates["enableForMergeRequests"].(bool); ok {
		gormUpdates["enable_for_merge_requests"] = enableForPRs
	}

	if len(gormUpdates) == 0 {
		return nil
	}

	return s.db.
		Model(&model.CustomField{}).
		Where("user_name = ? AND repository_name = ? AND field_id = ?", owner, repo, fieldID).
		Updates(gormUpdates).Error
}

// DeleteCustomField deletes a custom field
func (s *CustomFieldService) DeleteCustomField(owner, repo string, fieldID int) error {
	// First, remove all values for this field
	if err := s.db.
		Where("user_name = ? AND repository_name = ? AND field_id = ?", owner, repo, fieldID).
		Delete(&model.IssueCustomField{}).Error; err != nil {
		return err
	}

	// Then delete the field definition
	return s.db.
		Where("user_name = ? AND repository_name = ? AND field_id = ?", owner, repo, fieldID).
		Delete(&model.CustomField{}).Error
}

// GetIssueCustomFieldValue gets a custom field value for an issue
func (s *CustomFieldService) GetIssueCustomFieldValue(owner, repo string, issueID, fieldID int) (string, error) {
	value := &model.IssueCustomField{}
	err := s.db.
		Where("user_name = ? AND repository_name = ? AND issue_id = ? AND field_id = ?", owner, repo, issueID, fieldID).
		First(value).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}

	return value.Value, nil
}

// SetIssueCustomFieldValue sets a custom field value for an issue
func (s *CustomFieldService) SetIssueCustomFieldValue(owner, repo string, issueID, fieldID int, value string) error {
	// Check if the value already exists
	existing := &model.IssueCustomField{}
	err := s.db.
		Where("user_name = ? AND repository_name = ? AND issue_id = ? AND field_id = ?", owner, repo, issueID, fieldID).
		First(existing).Error

	if err == nil {
		// Update existing value
		return s.db.
			Model(&model.IssueCustomField{}).
			Where("user_name = ? AND repository_name = ? AND issue_id = ? AND field_id = ?", owner, repo, issueID, fieldID).
			Update("value", value).Error
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		// Insert new value
		newValue := &model.IssueCustomField{
			UserName:       owner,
			RepositoryName: repo,
			IssueID:        issueID,
			FieldID:        fieldID,
			Value:          value,
		}
		return s.db.Create(newValue).Error
	}

	return err
}

// GetIssueCustomFieldValues gets all custom field values for an issue
func (s *CustomFieldService) GetIssueCustomFieldValues(owner, repo string, issueID int) ([]*model.IssueCustomField, error) {
	var values []*model.IssueCustomField
	err := s.db.
		Where("user_name = ? AND repository_name = ? AND issue_id = ?", owner, repo, issueID).
		Find(&values).Error
	return values, err
}
