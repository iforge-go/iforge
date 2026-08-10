package service

import (
	"encoding/json"
	"fmt"
	"time"

	"iforge/iforge/internal/hashid"
	"iforge/iforge/internal/model"
	"iforge/iforge/internal/realtime"

	"gorm.io/gorm"
)

type NotificationService struct {
	db  *gorm.DB
	hub *realtime.Hub // Can be nil (skip push when uninitialized, backward compatible)
}

func NewNotificationService(db *gorm.DB) *NotificationService {
	return &NotificationService{db: db}
}

// SetHub injects the hub for real-time push notifications
func (s *NotificationService) SetHub(h *realtime.Hub) {
	s.hub = h
}

// ListNotifications retrieves notifications for a user
func (s *NotificationService) ListNotifications(userName string, unreadOnly bool) ([]*model.Notification, error) {
	var notifications []*model.Notification

	query := s.db.Where("recipient_user_name = ?", userName)
	if unreadOnly {
		query = query.Where("`read` = FALSE")
	}

	err := query.Order("registered_date DESC").Limit(50).Find(&notifications).Error
	if err != nil {
		return nil, err
	}

	return notifications, nil
}

// GetUnreadCount returns the count of unread notifications
func (s *NotificationService) GetUnreadCount(userName string) (int, error) {
	var count int64
	err := s.db.Model(&model.Notification{}).
		Where("recipient_user_name = ? AND `read` = FALSE", userName).
		Count(&count).Error
	if err != nil {
		return 0, err
	}

	return int(count), nil
}

// MarkAsRead marks a notification as read
func (s *NotificationService) MarkAsRead(notificationID int, userName string) error {
	result := s.db.Model(&model.Notification{}).
		Where("notification_id = ? AND recipient_user_name = ?", notificationID, userName).
		Update("`read`", true)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("notification not found")
	}

	return nil
}

// MarkAllAsRead marks all notifications as read for a user
func (s *NotificationService) MarkAllAsRead(userName string) error {
	return s.db.Model(&model.Notification{}).
		Where("recipient_user_name = ? AND `read` = FALSE", userName).
		Update("`read`", true).Error
}

// CreateNotification creates a new notification
func (s *NotificationService) CreateNotification(
	recipientUserName, repositoryUserName, repositoryName, notificationType string,
	issueID, commentID *int,
	actor, message string,
) error {
	notification := &model.Notification{
		RecipientUserName:  recipientUserName,
		RepositoryUserName: repositoryUserName,
		RepositoryName:     repositoryName,
		NotificationType:   notificationType,
		IssueID:            issueID,
		CommentID:          commentID,
		Actor:              actor,
		Message:            message,
		RegisteredDate:     time.Now(),
	}
	if err := s.db.Create(notification).Error; err != nil {
		return err
	}
	// Push notification to recipient in real-time (if online)
	s.pushNotification(notification)
	return nil
}

// BatchCreateNotifications creates multiple notifications in a single INSERT to eliminate N+1.
// Caller is responsible for deduplication and excluding the actor. Real-time push is executed per notification (best-effort, does not affect persistence).
// Used for scenarios like @mentions where multiple users need to be notified at once.
func (s *NotificationService) BatchCreateNotifications(notifications []*model.Notification) error {
	if len(notifications) == 0 {
		return nil
	}
	// GORM's Create on a slice generates a single batch INSERT statement
	if err := s.db.Create(&notifications).Error; err != nil {
		return err
	}
	// Push notification to each online recipient (best-effort)
	for _, n := range notifications {
		s.pushNotification(n)
	}
	return nil
}

// CreateTaskNotification creates a notification for a task-related event.
func (s *NotificationService) CreateTaskNotification(
	recipientUserName, notificationType, actor, message string,
	projectID, taskID *int,
) error {
	notification := &model.Notification{
		RecipientUserName: recipientUserName,
		NotificationType:  notificationType,
		ProjectID:         projectID,
		TaskID:            taskID,
		Actor:             actor,
		Message:           message,
		RegisteredDate:    time.Now(),
	}
	if err := s.db.Create(notification).Error; err != nil {
		return err
	}
	// Push notification to recipient in real-time (if online)
	s.pushNotification(notification)
	return nil
}

// CreateStoryNotification creates a notification for a story-related event.
func (s *NotificationService) CreateStoryNotification(
	recipientUserName, notificationType, actor, message string,
	projectID, storyID *int,
) error {
	notification := &model.Notification{
		RecipientUserName: recipientUserName,
		NotificationType:  notificationType,
		ProjectID:         projectID,
		StoryID:           storyID,
		Actor:             actor,
		Message:           message,
		RegisteredDate:    time.Now(),
	}
	if err := s.db.Create(notification).Error; err != nil {
		return err
	}
	// Push notification to recipient in real-time (if online)
	s.pushNotification(notification)
	return nil
}

// pushNotification pushes the newly created notification to all online WebSocket connections of the recipient.
// Skips if hub is nil (backward compatible).
//
// Before pushing, fills in Slug/ProjectSlug (both are gorm:"-" and empty after DB Create),
// so the frontend toast can directly jump to the corresponding entity when clicked - consistent with
// the fields returned by REST in notification_handler.go::ListNotifications.
func (s *NotificationService) pushNotification(n *model.Notification) {
	if s.hub == nil {
		return
	}
	n.Slug = hashid.Encode(n.NotificationID)
	if n.ProjectID != nil {
		n.ProjectSlug = hashid.Encode(*n.ProjectID)
	}
	if n.StoryID != nil {
		n.StorySlug = hashid.Encode(*n.StoryID)
	}
	msg, err := json.Marshal(map[string]interface{}{
		"type":    "notification",
		"payload": n,
	})
	if err != nil {
		return
	}
	s.hub.SendToUser(n.RecipientUserName, msg)
}

// PushToUsers broadcasts real-time events to multiple users (no DB persistence, only pushes to online connections).
// Used for collaborative events like task_updated that don't need persistence.
// Skips if hub is nil (backward compatible).
func (s *NotificationService) PushToUsers(usernames []string, msgType string, payload interface{}) {
	if s.hub == nil {
		return
	}
	msg, err := json.Marshal(map[string]interface{}{
		"type":    msgType,
		"payload": payload,
	})
	if err != nil {
		return
	}
	s.hub.SendToUsers(usernames, msg)
}

// DeleteNotification deletes a notification
func (s *NotificationService) DeleteNotification(notificationID int, userName string) error {
	result := s.db.
		Where("notification_id = ? AND recipient_user_name = ?", notificationID, userName).
		Delete(&model.Notification{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("notification not found")
	}

	return nil
}