package model

import "time"

// Notification represents a user notification
type Notification struct {
	NotificationID int    `gorm:"primaryKey;autoIncrement;column:notification_id" json:"-"`
	Slug           string `gorm:"-" json:"slug"`
	// Composite index (recipient_user_name, read) covers the two highest-frequency queries:
	// ListNotifications and GetUnreadCount
	RecipientUserName  string    `gorm:"column:recipient_user_name;index:idx_notif_recip_read,priority:1" json:"recipientUserName"`
	RepositoryUserName string    `gorm:"column:repository_user_name" json:"repositoryUserName"`
	RepositoryName     string    `gorm:"column:repository_name" json:"repositoryName"`
	NotificationType   string    `gorm:"column:notification_type" json:"notificationType"`
	IssueID            *int      `gorm:"column:issue_id" json:"issueId"`
	CommentID          *int      `gorm:"column:comment_id" json:"commentId"`
	ProjectID          *int      `gorm:"column:project_id" json:"-"`
	ProjectSlug        string    `gorm:"-" json:"projectSlug"`
	TaskID             *int      `gorm:"column:task_id" json:"taskId"`
	StoryID            *int      `gorm:"column:story_id" json:"storyId"`
	StorySlug          string    `gorm:"-" json:"storySlug"`
	Actor              string    `gorm:"column:actor" json:"actor"`
	Message            string    `gorm:"column:message" json:"message"`
	Read               bool      `gorm:"column:read;index:idx_notif_recip_read,priority:2" json:"read"`
	RegisteredDate     time.Time `gorm:"column:registered_date;index:idx_notif_registered" json:"registeredDate"`
}

func (Notification) TableName() string { return "notification" }
