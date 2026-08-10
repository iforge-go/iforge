package model

import "time"

// CustomField represents a custom field definition
type CustomField struct {
	FieldID                int       `gorm:"primaryKey;autoIncrement;column:field_id" json:"fieldId"`
	// Composite index on (user_name, repository_name) covers per-repository custom field queries
	UserName               string    `gorm:"column:user_name;index:idx_customfield_repo,priority:1" json:"userName"`
	RepositoryName         string    `gorm:"column:repository_name;index:idx_customfield_repo,priority:2" json:"repositoryName"`
	FieldName              string    `gorm:"column:field_name" json:"fieldName"`
	FieldType              string    `gorm:"column:field_type" json:"fieldType"`
	Constraints            *string   `gorm:"column:constraints" json:"constraints"`
	EnableForIssues        bool      `gorm:"column:enable_for_issues" json:"enableForIssues"`
	EnableForMergeRequests bool      `gorm:"column:enable_for_merge_requests" json:"enableForMergeRequests"`
	RegisteredDate         time.Time `gorm:"column:registered_date" json:"registeredDate"`
}

func (CustomField) TableName() string { return "custom_field" }

// IssueCustomField represents a custom field value for an issue
type IssueCustomField struct {
	UserName       string `gorm:"primaryKey;column:user_name" json:"userName"`
	RepositoryName string `gorm:"primaryKey;column:repository_name" json:"repositoryName"`
	IssueID        int    `gorm:"primaryKey;column:issue_id" json:"issueId"`
	FieldID        int    `gorm:"primaryKey;column:field_id" json:"fieldId"`
	Value          string `gorm:"column:value" json:"value"`
}

func (IssueCustomField) TableName() string { return "issue_custom_field" }
