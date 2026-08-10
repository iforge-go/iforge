package service

import (
	"testing"

	"iforge/iforge/internal/model"
)

func TestScrumActivityService_LogActivity(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewScrumActivityService(db)

	// Log an activity
	err := service.LogActivity("testuser", 1, "task", 100, "created", "", "", "")
	if err != nil {
		t.Fatalf("LogActivity failed: %v", err)
	}

	// Verify activity was logged
	activities, err := service.GetActivities(1, "task", 100)
	if err != nil {
		t.Fatalf("GetActivities failed: %v", err)
	}
	if len(activities) != 1 {
		t.Fatalf("Expected 1 activity, got %d", len(activities))
	}

	activity := activities[0]
	if activity.UserName != "testuser" {
		t.Errorf("Expected UserName 'testuser', got '%s'", activity.UserName)
	}
	if activity.ProjectID != 1 {
		t.Errorf("Expected ProjectID 1, got %d", activity.ProjectID)
	}
	if activity.EntityType != "task" {
		t.Errorf("Expected EntityType 'task', got '%s'", activity.EntityType)
	}
	if activity.EntityID != 100 {
		t.Errorf("Expected EntityID 100, got %d", activity.EntityID)
	}
	if activity.Action != "created" {
		t.Errorf("Expected Action 'created', got '%s'", activity.Action)
	}
}

func TestScrumActivityService_LogActivity_WithFieldChanges(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewScrumActivityService(db)

	// Log activity with field changes
	err := service.LogActivity("testuser", 1, "task", 100, "updated", "status", "open", "closed")
	if err != nil {
		t.Fatalf("LogActivity failed: %v", err)
	}

	// Verify activity
	activities, err := service.GetActivities(1, "task", 100)
	if err != nil {
		t.Fatalf("GetActivities failed: %v", err)
	}
	if len(activities) != 1 {
		t.Fatalf("Expected 1 activity, got %d", len(activities))
	}

	activity := activities[0]
	if activity.Field != "status" {
		t.Errorf("Expected Field 'status', got '%s'", activity.Field)
	}
	if activity.OldValue != "open" {
		t.Errorf("Expected OldValue 'open', got '%s'", activity.OldValue)
	}
	if activity.NewValue != "closed" {
		t.Errorf("Expected NewValue 'closed', got '%s'", activity.NewValue)
	}
}

func TestScrumActivityService_GetActivities_MultipleActivities(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewScrumActivityService(db)

	// Log multiple activities
	service.LogActivity("user1", 1, "task", 100, "created", "", "", "")
	service.LogActivity("user2", 1, "task", 100, "updated", "status", "open", "in_progress")
	service.LogActivity("user1", 1, "task", 100, "updated", "assignee", "", "user2")

	// Get activities
	activities, err := service.GetActivities(1, "task", 100)
	if err != nil {
		t.Fatalf("GetActivities failed: %v", err)
	}
	if len(activities) != 3 {
		t.Fatalf("Expected 3 activities, got %d", len(activities))
	}

	// Verify order (newest first)
	if activities[0].UserName != "user1" || activities[0].Field != "assignee" {
		t.Errorf("Expected first activity to be assignee update by user1")
	}
	if activities[2].UserName != "user1" || activities[2].Action != "created" {
		t.Errorf("Expected last activity to be creation by user1")
	}
}

func TestScrumActivityService_GetActivities_DifferentEntities(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewScrumActivityService(db)

	// Log activities for different entities
	service.LogActivity("user1", 1, "task", 100, "created", "", "", "")
	service.LogActivity("user1", 1, "task", 200, "created", "", "", "")
	service.LogActivity("user1", 1, "story", 100, "created", "", "", "")

	// Get activities for task 100
	activities, err := service.GetActivities(1, "task", 100)
	if err != nil {
		t.Fatalf("GetActivities failed: %v", err)
	}
	if len(activities) != 1 {
		t.Fatalf("Expected 1 activity for task 100, got %d", len(activities))
	}

	// Get activities for task 200
	activities, err = service.GetActivities(1, "task", 200)
	if err != nil {
		t.Fatalf("GetActivities failed: %v", err)
	}
	if len(activities) != 1 {
		t.Fatalf("Expected 1 activity for task 200, got %d", len(activities))
	}

	// Get activities for story 100
	activities, err = service.GetActivities(1, "story", 100)
	if err != nil {
		t.Fatalf("GetActivities failed: %v", err)
	}
	if len(activities) != 1 {
		t.Fatalf("Expected 1 activity for story 100, got %d", len(activities))
	}
}

func TestScrumActivityService_GetActivities_DifferentProjects(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewScrumActivityService(db)

	// Log activities for different projects (same entity ID)
	service.LogActivity("user1", 1, "task", 100, "created", "", "", "")
	service.LogActivity("user1", 2, "task", 100, "created", "", "", "")

	// Get activities for project 1
	activities, err := service.GetActivities(1, "task", 100)
	if err != nil {
		t.Fatalf("GetActivities failed: %v", err)
	}
	if len(activities) != 1 {
		t.Fatalf("Expected 1 activity for project 1, got %d", len(activities))
	}
	if activities[0].ProjectID != 1 {
		t.Errorf("Expected ProjectID 1, got %d", activities[0].ProjectID)
	}

	// Get activities for project 2
	activities, err = service.GetActivities(2, "task", 100)
	if err != nil {
		t.Fatalf("GetActivities failed: %v", err)
	}
	if len(activities) != 1 {
		t.Fatalf("Expected 1 activity for project 2, got %d", len(activities))
	}
	if activities[0].ProjectID != 2 {
		t.Errorf("Expected ProjectID 2, got %d", activities[0].ProjectID)
	}
}

func TestScrumActivityService_GetActivities_NoActivities(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewScrumActivityService(db)

	// Get activities for non-existent entity
	activities, err := service.GetActivities(1, "task", 999)
	if err != nil {
		t.Fatalf("GetActivities failed: %v", err)
	}
	if len(activities) != 0 {
		t.Errorf("Expected 0 activities, got %d", len(activities))
	}
}

func TestScrumActivityService_EnrichActivityValues_SprintId(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewScrumActivityService(db)

	// Create sprints
	goal1 := "Goal 1"
	goal2 := "Goal 2"
	sprint1 := &model.Sprint{
		ProjectID: 1,
		Title:     "Sprint 1",
		Goal:      &goal1,
		Status:    "active",
	}
	sprint2 := &model.Sprint{
		ProjectID: 1,
		Title:     "Sprint 2",
		Goal:      &goal2,
		Status:    "planned",
	}
	db.Create(sprint1)
	db.Create(sprint2)

	// Log activity with sprintId field changes
	err := service.LogActivity("user1", 1, "task", 100, "updated", "sprintId", "1", "2")
	if err != nil {
		t.Fatalf("LogActivity failed: %v", err)
	}

	// Get activities (should enrich sprint IDs to names)
	activities, err := service.GetActivities(1, "task", 100)
	if err != nil {
		t.Fatalf("GetActivities failed: %v", err)
	}
	if len(activities) != 1 {
		t.Fatalf("Expected 1 activity, got %d", len(activities))
	}

	activity := activities[0]
	if activity.OldValue != "Sprint 1" {
		t.Errorf("Expected OldValue 'Sprint 1', got '%s'", activity.OldValue)
	}
	if activity.NewValue != "Sprint 2" {
		t.Errorf("Expected NewValue 'Sprint 2', got '%s'", activity.NewValue)
	}
}

func TestScrumActivityService_EnrichActivityValues_UserStoryId(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewScrumActivityService(db)

	// Create user stories
	desc1 := "Description 1"
	desc2 := "Description 2"
	story1 := &model.UserStory{
		ProjectID:   1,
		Title:       "Story 1",
		Description: &desc1,
		Priority:    "high",
		Status:      "open",
	}
	story2 := &model.UserStory{
		ProjectID:   1,
		Title:       "Story 2",
		Description: &desc2,
		Priority:    "medium",
		Status:      "in_progress",
	}
	db.Create(story1)
	db.Create(story2)

	// Log activity with userStoryId field changes
	err := service.LogActivity("user1", 1, "task", 100, "updated", "userStoryId", "1", "2")
	if err != nil {
		t.Fatalf("LogActivity failed: %v", err)
	}

	// Get activities (should enrich story IDs to titles)
	activities, err := service.GetActivities(1, "task", 100)
	if err != nil {
		t.Fatalf("GetActivities failed: %v", err)
	}
	if len(activities) != 1 {
		t.Fatalf("Expected 1 activity, got %d", len(activities))
	}

	activity := activities[0]
	if activity.OldValue != "Story 1" {
		t.Errorf("Expected OldValue 'Story 1', got '%s'", activity.OldValue)
	}
	if activity.NewValue != "Story 2" {
		t.Errorf("Expected NewValue 'Story 2', got '%s'", activity.NewValue)
	}
}

func TestScrumActivityService_EnrichActivityValues_InvalidIDs(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewScrumActivityService(db)

	// Log activity with invalid sprintId (non-numeric)
	err := service.LogActivity("user1", 1, "task", 100, "updated", "sprintId", "invalid", "999")
	if err != nil {
		t.Fatalf("LogActivity failed: %v", err)
	}

	// Get activities (should not crash, invalid IDs remain unchanged)
	activities, err := service.GetActivities(1, "task", 100)
	if err != nil {
		t.Fatalf("GetActivities failed: %v", err)
	}
	if len(activities) != 1 {
		t.Fatalf("Expected 1 activity, got %d", len(activities))
	}

	activity := activities[0]
	// Invalid ID should remain unchanged
	if activity.OldValue != "invalid" {
		t.Errorf("Expected OldValue 'invalid', got '%s'", activity.OldValue)
	}
	// Non-existent ID should remain as-is (not enriched)
	if activity.NewValue != "999" {
		t.Errorf("Expected NewValue '999', got '%s'", activity.NewValue)
	}
}
