package service

import (
	"iforge/iforge/internal/model"
	"testing"
)

func TestNotificationService_CreateNotification(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewNotificationService(db)

	// Create a notification
	issueID := 1
	commentID := 10
	err := service.CreateNotification(
		"recipient",
		"repo_owner",
		"test-repo",
		"mention",
		&issueID,
		&commentID,
		"actor",
		"You were mentioned in a comment",
	)
	if err != nil {
		t.Fatalf("CreateNotification failed: %v", err)
	}

	// Verify the notification was created
	notifications, err := service.ListNotifications("recipient", false)
	if err != nil {
		t.Fatalf("ListNotifications failed: %v", err)
	}
	if len(notifications) != 1 {
		t.Errorf("Expected 1 notification, got %d", len(notifications))
	}

	n := notifications[0]
	if n.RecipientUserName != "recipient" {
		t.Errorf("Expected RecipientUserName 'recipient', got '%s'", n.RecipientUserName)
	}
	if n.RepositoryUserName != "repo_owner" {
		t.Errorf("Expected RepositoryUserName 'repo_owner', got '%s'", n.RepositoryUserName)
	}
	if n.RepositoryName != "test-repo" {
		t.Errorf("Expected RepositoryName 'test-repo', got '%s'", n.RepositoryName)
	}
	if n.NotificationType != "mention" {
		t.Errorf("Expected NotificationType 'mention', got '%s'", n.NotificationType)
	}
	if n.IssueID == nil || *n.IssueID != issueID {
		t.Errorf("Expected IssueID %d, got %v", issueID, n.IssueID)
	}
	if n.CommentID == nil || *n.CommentID != commentID {
		t.Errorf("Expected CommentID %d, got %v", commentID, n.CommentID)
	}
	if n.Actor != "actor" {
		t.Errorf("Expected Actor 'actor', got '%s'", n.Actor)
	}
	if n.Message != "You were mentioned in a comment" {
		t.Errorf("Expected Message 'You were mentioned in a comment', got '%s'", n.Message)
	}
	if n.Read {
		t.Error("Expected Read false, got true")
	}
}

func TestNotificationService_ListNotifications(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewNotificationService(db)

	// Create multiple notifications
	for i := 0; i < 3; i++ {
		err := service.CreateNotification(
			"recipient",
			"repo_owner",
			"test-repo",
			"mention",
			nil,
			nil,
			"actor",
			"Test notification",
		)
		if err != nil {
			t.Fatalf("CreateNotification %d failed: %v", i, err)
		}
	}

	// List all notifications
	notifications, err := service.ListNotifications("recipient", false)
	if err != nil {
		t.Fatalf("ListNotifications failed: %v", err)
	}
	if len(notifications) != 3 {
		t.Errorf("Expected 3 notifications, got %d", len(notifications))
	}

	// List unread notifications
	unread, err := service.ListNotifications("recipient", true)
	if err != nil {
		t.Fatalf("ListNotifications (unreadOnly) failed: %v", err)
	}
	if len(unread) != 3 {
		t.Errorf("Expected 3 unread notifications, got %d", len(unread))
	}
}

func TestNotificationService_ListNotifications_Limit(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewNotificationService(db)

	// Create 60 notifications (more than the limit of 50)
	for i := 0; i < 60; i++ {
		err := service.CreateNotification(
			"recipient",
			"repo_owner",
			"test-repo",
			"mention",
			nil,
			nil,
			"actor",
			"Test notification",
		)
		if err != nil {
			t.Fatalf("CreateNotification %d failed: %v", i, err)
		}
	}

	// List should be limited to 50
	notifications, err := service.ListNotifications("recipient", false)
	if err != nil {
		t.Fatalf("ListNotifications failed: %v", err)
	}
	if len(notifications) != 50 {
		t.Errorf("Expected 50 notifications (limit), got %d", len(notifications))
	}
}

func TestNotificationService_GetUnreadCount(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewNotificationService(db)

	// Create 5 notifications
	for i := 0; i < 5; i++ {
		err := service.CreateNotification(
			"recipient",
			"repo_owner",
			"test-repo",
			"mention",
			nil,
			nil,
			"actor",
			"Test notification",
		)
		if err != nil {
			t.Fatalf("CreateNotification %d failed: %v", i, err)
		}
	}

	// Get unread count
	count, err := service.GetUnreadCount("recipient")
	if err != nil {
		t.Fatalf("GetUnreadCount failed: %v", err)
	}
	if count != 5 {
		t.Errorf("Expected 5 unread, got %d", count)
	}

	// Mark one as read
	notifications, _ := service.ListNotifications("recipient", false)
	if err := service.MarkAsRead(notifications[0].NotificationID, "recipient"); err != nil {
		t.Fatalf("MarkAsRead failed: %v", err)
	}

	// Get unread count again
	count, err = service.GetUnreadCount("recipient")
	if err != nil {
		t.Fatalf("GetUnreadCount failed: %v", err)
	}
	if count != 4 {
		t.Errorf("Expected 4 unread after marking one, got %d", count)
	}
}

func TestNotificationService_MarkAsRead(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewNotificationService(db)

	// Create a notification
	err := service.CreateNotification(
		"recipient",
		"repo_owner",
		"test-repo",
		"mention",
		nil,
		nil,
		"actor",
		"Test notification",
	)
	if err != nil {
		t.Fatalf("CreateNotification failed: %v", err)
	}

	// Get the notification
	notifications, _ := service.ListNotifications("recipient", false)
	if len(notifications) != 1 {
		t.Fatalf("Expected 1 notification, got %d", len(notifications))
	}

	// Mark as read
	if err := service.MarkAsRead(notifications[0].NotificationID, "recipient"); err != nil {
		t.Fatalf("MarkAsRead failed: %v", err)
	}

	// Verify it's marked as read
	unread, _ := service.ListNotifications("recipient", true)
	if len(unread) != 0 {
		t.Errorf("Expected 0 unread after marking, got %d", len(unread))
	}

	// Verify it still appears in all notifications
	all, _ := service.ListNotifications("recipient", false)
	if len(all) != 1 {
		t.Errorf("Expected 1 notification in all list, got %d", len(all))
	}
	if !all[0].Read {
		t.Error("Expected notification to be marked as read")
	}
}

func TestNotificationService_MarkAsRead_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewNotificationService(db)

	// Try to mark non-existent notification
	err := service.MarkAsRead(999, "recipient")
	if err == nil {
		t.Fatal("Expected error for non-existent notification, got nil")
	}
}

func TestNotificationService_MarkAllAsRead(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewNotificationService(db)

	// Create 5 notifications
	for i := 0; i < 5; i++ {
		err := service.CreateNotification(
			"recipient",
			"repo_owner",
			"test-repo",
			"mention",
			nil,
			nil,
			"actor",
			"Test notification",
		)
		if err != nil {
			t.Fatalf("CreateNotification %d failed: %v", i, err)
		}
	}

	// Verify all are unread
	unread, _ := service.ListNotifications("recipient", true)
	if len(unread) != 5 {
		t.Errorf("Expected 5 unread, got %d", len(unread))
	}

	// Mark all as read
	if err := service.MarkAllAsRead("recipient"); err != nil {
		t.Fatalf("MarkAllAsRead failed: %v", err)
	}

	// Verify all are read
	unread, _ = service.ListNotifications("recipient", true)
	if len(unread) != 0 {
		t.Errorf("Expected 0 unread after marking all, got %d", len(unread))
	}

	// Verify all still exist
	all, _ := service.ListNotifications("recipient", false)
	if len(all) != 5 {
		t.Errorf("Expected 5 notifications in all list, got %d", len(all))
	}
}

func TestNotificationService_DeleteNotification(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewNotificationService(db)

	// Create a notification
	err := service.CreateNotification(
		"recipient",
		"repo_owner",
		"test-repo",
		"mention",
		nil,
		nil,
		"actor",
		"Test notification",
	)
	if err != nil {
		t.Fatalf("CreateNotification failed: %v", err)
	}

	// Get the notification
	notifications, _ := service.ListNotifications("recipient", false)
	if len(notifications) != 1 {
		t.Fatalf("Expected 1 notification, got %d", len(notifications))
	}

	// Delete the notification
	if err := service.DeleteNotification(notifications[0].NotificationID, "recipient"); err != nil {
		t.Fatalf("DeleteNotification failed: %v", err)
	}

	// Verify it's deleted
	notifications, _ = service.ListNotifications("recipient", false)
	if len(notifications) != 0 {
		t.Errorf("Expected 0 notifications after deletion, got %d", len(notifications))
	}
}

func TestNotificationService_DeleteNotification_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewNotificationService(db)

	// Try to delete non-existent notification
	err := service.DeleteNotification(999, "recipient")
	if err == nil {
		t.Fatal("Expected error for non-existent notification, got nil")
	}
}

func TestNotificationService_DeleteNotification_WrongUser(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewNotificationService(db)

	// Create a notification for user1
	err := service.CreateNotification(
		"user1",
		"repo_owner",
		"test-repo",
		"mention",
		nil,
		nil,
		"actor",
		"Test notification",
	)
	if err != nil {
		t.Fatalf("CreateNotification failed: %v", err)
	}

	// Get the notification
	notifications, _ := service.ListNotifications("user1", false)
	if len(notifications) != 1 {
		t.Fatalf("Expected 1 notification, got %d", len(notifications))
	}

	// Try to delete with wrong user
	err = service.DeleteNotification(notifications[0].NotificationID, "user2")
	if err == nil {
		t.Fatal("Expected error when deleting with wrong user, got nil")
	}

	// Verify notification still exists for user1
	notifications, _ = service.ListNotifications("user1", false)
	if len(notifications) != 1 {
		t.Errorf("Expected 1 notification to still exist, got %d", len(notifications))
	}
}

func TestNotificationService_BatchCreateNotifications(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewNotificationService(db)

	// Create batch of notifications
	notifications := []*model.Notification{
		{
			RecipientUserName:  "user1",
			RepositoryUserName: "repo_owner",
			RepositoryName:     "test-repo",
			NotificationType:   "mention",
			Actor:              "actor",
			Message:            "Notification 1",
		},
		{
			RecipientUserName:  "user2",
			RepositoryUserName: "repo_owner",
			RepositoryName:     "test-repo",
			NotificationType:   "mention",
			Actor:              "actor",
			Message:            "Notification 2",
		},
		{
			RecipientUserName:  "user3",
			RepositoryUserName: "repo_owner",
			RepositoryName:     "test-repo",
			NotificationType:   "mention",
			Actor:              "actor",
			Message:            "Notification 3",
		},
	}

	// Batch create
	if err := service.BatchCreateNotifications(notifications); err != nil {
		t.Fatalf("BatchCreateNotifications failed: %v", err)
	}

	// Verify each user has their notification
	for i, user := range []string{"user1", "user2", "user3"} {
		notifications, err := service.ListNotifications(user, false)
		if err != nil {
			t.Fatalf("ListNotifications for %s failed: %v", user, err)
		}
		if len(notifications) != 1 {
			t.Errorf("Expected 1 notification for %s, got %d", user, len(notifications))
		}
		if notifications[0].Message != "Notification "+string(rune('1'+i)) {
			t.Errorf("Expected message 'Notification %d' for %s, got '%s'", i+1, user, notifications[0].Message)
		}
	}
}

func TestNotificationService_BatchCreateNotifications_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewNotificationService(db)

	// Batch create with empty slice
	if err := service.BatchCreateNotifications([]*model.Notification{}); err != nil {
		t.Fatalf("BatchCreateNotifications with empty slice failed: %v", err)
	}
}

func TestNotificationService_CreateTaskNotification(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewNotificationService(db)

	projectID := 1
	taskID := 10

	// Create task notification
	err := service.CreateTaskNotification(
		"recipient",
		"task_assigned",
		"actor",
		"You were assigned to a task",
		&projectID,
		&taskID,
	)
	if err != nil {
		t.Fatalf("CreateTaskNotification failed: %v", err)
	}

	// Verify the notification was created
	notifications, err := service.ListNotifications("recipient", false)
	if err != nil {
		t.Fatalf("ListNotifications failed: %v", err)
	}
	if len(notifications) != 1 {
		t.Errorf("Expected 1 notification, got %d", len(notifications))
	}

	n := notifications[0]
	if n.NotificationType != "task_assigned" {
		t.Errorf("Expected NotificationType 'task_assigned', got '%s'", n.NotificationType)
	}
	if n.ProjectID == nil || *n.ProjectID != projectID {
		t.Errorf("Expected ProjectID %d, got %v", projectID, n.ProjectID)
	}
	if n.TaskID == nil || *n.TaskID != taskID {
		t.Errorf("Expected TaskID %d, got %v", taskID, n.TaskID)
	}
}

func TestNotificationService_CreateStoryNotification(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewNotificationService(db)

	projectID := 1
	storyID := 5

	// Create story notification
	err := service.CreateStoryNotification(
		"recipient",
		"story_updated",
		"actor",
		"A story was updated",
		&projectID,
		&storyID,
	)
	if err != nil {
		t.Fatalf("CreateStoryNotification failed: %v", err)
	}

	// Verify the notification was created
	notifications, err := service.ListNotifications("recipient", false)
	if err != nil {
		t.Fatalf("ListNotifications failed: %v", err)
	}
	if len(notifications) != 1 {
		t.Errorf("Expected 1 notification, got %d", len(notifications))
	}

	n := notifications[0]
	if n.NotificationType != "story_updated" {
		t.Errorf("Expected NotificationType 'story_updated', got '%s'", n.NotificationType)
	}
	if n.ProjectID == nil || *n.ProjectID != projectID {
		t.Errorf("Expected ProjectID %d, got %v", projectID, n.ProjectID)
	}
	if n.StoryID == nil || *n.StoryID != storyID {
		t.Errorf("Expected StoryID %d, got %v", storyID, n.StoryID)
	}
}

func TestNotificationService_MultipleUsers(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewNotificationService(db)

	// Create notifications for different users
	for i := 0; i < 3; i++ {
		user := "user" + string(rune('1'+i))
		err := service.CreateNotification(
			user,
			"repo_owner",
			"test-repo",
			"mention",
			nil,
			nil,
			"actor",
			"Test notification",
		)
		if err != nil {
			t.Fatalf("CreateNotification for %s failed: %v", user, err)
		}
	}

	// Verify each user has only their own notification
	for i := 0; i < 3; i++ {
		user := "user" + string(rune('1'+i))
		notifications, err := service.ListNotifications(user, false)
		if err != nil {
			t.Fatalf("ListNotifications for %s failed: %v", user, err)
		}
		if len(notifications) != 1 {
			t.Errorf("Expected 1 notification for %s, got %d", user, len(notifications))
		}
	}

	// Verify unread counts
	for i := 0; i < 3; i++ {
		user := "user" + string(rune('1'+i))
		count, err := service.GetUnreadCount(user)
		if err != nil {
			t.Fatalf("GetUnreadCount for %s failed: %v", user, err)
		}
		if count != 1 {
			t.Errorf("Expected 1 unread for %s, got %d", user, count)
		}
	}
}
