package model

import "time"

// Issue represents an issue or merge request
type Issue struct {
	// Composite index (user_name, repository_name, closed, merge_request) covers the high-frequency ListIssues query
	UserName       string    `gorm:"primaryKey;column:user_name;index:idx_issue_repo_closed,priority:1" json:"userName"`
	RepositoryName string    `gorm:"primaryKey;column:repository_name;index:idx_issue_repo_closed,priority:2" json:"repositoryName"`
	IssueID        int       `gorm:"primaryKey;column:issue_id" json:"issueId"`
	OpenedUserName string    `gorm:"column:opened_user_name;index" json:"openedUserName"`
	MilestoneID    *int      `gorm:"column:milestone_id;index" json:"milestoneId"`
	PriorityID     *int      `gorm:"column:priority_id" json:"priorityId"`
	Title          string    `gorm:"column:title" json:"title"`
	Content        *string   `gorm:"column:content" json:"content"`
	Closed         bool      `gorm:"column:closed;index:idx_issue_repo_closed,priority:3" json:"closed"`
	CloseReason    *string   `gorm:"column:close_reason" json:"closeReason"`
	Locked         bool      `gorm:"column:locked" json:"locked"`
	RegisteredDate time.Time `gorm:"column:registered_date;index" json:"registeredDate"`
	UpdatedDate    time.Time `gorm:"column:updated_date;index:idx_issue_updated" json:"updatedDate"`
	IsMergeRequest bool      `gorm:"column:merge_request;index:idx_issue_repo_closed,priority:4" json:"isMergeRequest"`
}

func (Issue) TableName() string { return "issue" }

// IssueIDCounter represents the issue ID counter for a repository
type IssueIDCounter struct {
	UserName       string `gorm:"primaryKey;column:user_name" json:"userName"`
	RepositoryName string `gorm:"primaryKey;column:repository_name" json:"repositoryName"`
	IssueID        int    `gorm:"column:issue_id" json:"issueId"`
}

func (IssueIDCounter) TableName() string { return "issue_id" }

// IssueComment represents a comment on an issue
type IssueComment struct {
	// Composite index (user_name, repository_name, issue_id) covers GetComments
	UserName          string    `gorm:"column:user_name;index:idx_issue_comment_repo,priority:1" json:"userName"`
	RepositoryName    string    `gorm:"column:repository_name;index:idx_issue_comment_repo,priority:2" json:"repositoryName"`
	IssueID           int       `gorm:"column:issue_id;index:idx_issue_comment_repo,priority:3" json:"issueId"`
	CommentID         int       `gorm:"primaryKey;autoIncrement;column:comment_id" json:"commentId"`
	Action            string    `gorm:"column:action" json:"action"`
	CommentedUserName string    `gorm:"column:commented_user_name" json:"commentedUserName"`
	Content           string    `gorm:"column:content" json:"content"`
	RegisteredDate    time.Time `gorm:"column:registered_date;index:idx_issue_comment_registered" json:"registeredDate"`
	UpdatedDate       time.Time `gorm:"column:updated_date" json:"updatedDate"`
}

func (IssueComment) TableName() string { return "issue_comment" }

// Label represents an issue label
type Label struct {
	UserName       string `gorm:"column:user_name;index:idx_label_repo,priority:1" json:"userName"`
	RepositoryName string `gorm:"column:repository_name;index:idx_label_repo,priority:2" json:"repositoryName"`
	LabelID        int    `gorm:"primaryKey;autoIncrement;column:label_id" json:"labelId"`
	LabelName      string `gorm:"column:label_name" json:"labelName"`
	Color          string `gorm:"column:color" json:"color"`
}

func (Label) TableName() string { return "label" }

// Milestone represents a milestone
type Milestone struct {
	UserName       string     `gorm:"column:user_name;index:idx_milestone_repo,priority:1" json:"userName"`
	RepositoryName string     `gorm:"column:repository_name;index:idx_milestone_repo,priority:2" json:"repositoryName"`
	MilestoneID    int        `gorm:"primaryKey;autoIncrement;column:milestone_id" json:"milestoneId"`
	Title          string     `gorm:"column:title" json:"title"`
	Description    *string    `gorm:"column:description" json:"description"`
	DueDate        *time.Time `gorm:"column:due_date" json:"dueDate"`
	ClosedDate     *time.Time `gorm:"column:closed_date" json:"closedDate"`
}

func (Milestone) TableName() string { return "milestone" }

// Priority represents an issue priority
type Priority struct {
	UserName       string  `gorm:"column:user_name" json:"userName"`
	RepositoryName string  `gorm:"column:repository_name" json:"repositoryName"`
	PriorityID     int     `gorm:"primaryKey;autoIncrement;column:priority_id" json:"priorityId"`
	PriorityName   string  `gorm:"column:priority_name" json:"priorityName"`
	Description    *string `gorm:"column:description" json:"description"`
	Color          string  `gorm:"column:color" json:"color"`
}

func (Priority) TableName() string { return "priority" }

// IssueLabel represents the many-to-many relationship between issues and labels
type IssueLabel struct {
	UserName       string `gorm:"primaryKey;column:user_name" json:"userName"`
	RepositoryName string `gorm:"primaryKey;column:repository_name" json:"repositoryName"`
	IssueID        int    `gorm:"primaryKey;column:issue_id" json:"issueId"`
	LabelID        int    `gorm:"primaryKey;column:label_id" json:"labelId"`
}

func (IssueLabel) TableName() string { return "issue_label" }

// IssueAssignment represents the many-to-many relationship between issues and assignees
type IssueAssignment struct {
	UserName         string `gorm:"primaryKey;column:user_name" json:"userName"`
	RepositoryName   string `gorm:"primaryKey;column:repository_name" json:"repositoryName"`
	IssueID          int    `gorm:"primaryKey;column:issue_id" json:"issueId"`
	AssigneeUserName string `gorm:"primaryKey;column:assignee_user_name" json:"assigneeUserName"`
}

func (IssueAssignment) TableName() string { return "issue_assignment" }
