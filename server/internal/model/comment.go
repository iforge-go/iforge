package model

import "time"

// CommitComment represents a comment on a commit
type CommitComment struct {
	UserName          string    `gorm:"column:user_name" json:"userName"`
	RepositoryName    string    `gorm:"column:repository_name" json:"repositoryName"`
	CommitID          string    `gorm:"column:commit_id" json:"commitId"`
	CommentID         int       `gorm:"primaryKey;autoIncrement;column:comment_id" json:"commentId"`
	CommentedUserName string    `gorm:"column:commented_user_name" json:"commentedUserName"`
	Content           string    `gorm:"column:content" json:"content"`
	FileName          *string   `gorm:"column:file_name" json:"fileName"`
	OldLine           *int      `gorm:"column:old_line_number" json:"oldLine"`
	NewLine           *int      `gorm:"column:new_line_number" json:"newLine"`
	RegisteredDate    time.Time `gorm:"column:registered_date" json:"registeredDate"`
	UpdatedDate       time.Time `gorm:"column:updated_date" json:"updatedDate"`
	IssueID           *int      `gorm:"column:issue_id" json:"issueId"`
	OriginalCommitID  string    `gorm:"column:original_commit_id" json:"originalCommitId"`
	OriginalOldLine   *int      `gorm:"column:original_old_line" json:"originalOldLine"`
	OriginalNewLine   *int      `gorm:"column:original_new_line" json:"originalNewLine"`
}

func (CommitComment) TableName() string { return "commit_comment" }
