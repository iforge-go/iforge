package model

import "time"

// Review represents a code review on a merge request
type Review struct {
	ReviewID       int       `gorm:"primaryKey;autoIncrement;column:review_id" json:"reviewId"`
	UserName       string    `gorm:"column:user_name" json:"userName"`
	RepositoryName string    `gorm:"column:repository_name" json:"repositoryName"`
	IssueID        int       `gorm:"column:issue_id" json:"issueId"`
	Reviewer       string    `gorm:"column:reviewer" json:"reviewer"`
	Status         string    `gorm:"column:status" json:"status"`
	Content        *string   `gorm:"column:content" json:"content"`
	RegisteredDate time.Time `gorm:"column:registered_date" json:"registeredDate"`
	UpdatedDate    time.Time `gorm:"column:updated_date" json:"updatedDate"`
}

func (Review) TableName() string { return "review" }

// ReviewComment represents a comment on a specific line of code in a review
type ReviewComment struct {
	CommentID      int       `gorm:"primaryKey;autoIncrement;column:comment_id" json:"commentId"`
	ReviewID       int       `gorm:"column:review_id" json:"reviewId"`
	UserName       string    `gorm:"column:user_name" json:"userName"`
	RepositoryName string    `gorm:"column:repository_name" json:"repositoryName"`
	IssueID        int       `gorm:"column:issue_id" json:"issueId"`
	Commenter      string    `gorm:"column:commenter" json:"commenter"`
	FilePath       string    `gorm:"column:file_path" json:"filePath"`
	Line           int       `gorm:"column:line" json:"line"`
	Content        string    `gorm:"column:content" json:"content"`
	RegisteredDate time.Time `gorm:"column:registered_date" json:"registeredDate"`
	UpdatedDate    time.Time `gorm:"column:updated_date" json:"updatedDate"`
}

func (ReviewComment) TableName() string { return "review_comment" }
