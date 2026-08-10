package service

import (
	"testing"
)

func TestActivityService_CreateActivity(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewActivityService(db)

	// Create an activity
	additionalInfo := "test info"
	activity, err := service.CreateActivity(
		"testuser", "test-repo", "testuser",
		"commit", "Test commit message",
		&additionalInfo,
	)
	if err != nil {
		t.Fatalf("CreateActivity failed: %v", err)
	}

	if activity.UserName != "testuser" {
		t.Errorf("Expected UserName 'testuser', got '%s'", activity.UserName)
	}
	if activity.RepositoryName != "test-repo" {
		t.Errorf("Expected RepositoryName 'test-repo', got '%s'", activity.RepositoryName)
	}
	if activity.ActivityUserName != "testuser" {
		t.Errorf("Expected ActivityUserName 'testuser', got '%s'", activity.ActivityUserName)
	}
	if activity.ActivityType != "commit" {
		t.Errorf("Expected ActivityType 'commit', got '%s'", activity.ActivityType)
	}
	if activity.Message != "Test commit message" {
		t.Errorf("Expected Message 'Test commit message', got '%s'", activity.Message)
	}
	if activity.AdditionalInfo == nil || *activity.AdditionalInfo != "test info" {
		t.Errorf("Expected AdditionalInfo 'test info', got %v", activity.AdditionalInfo)
	}
	if activity.ActivityID == "" {
		t.Error("Expected ActivityID to be set")
	}
}

func TestActivityService_GetActivities(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewActivityService(db)

	// Create multiple activities
	for i := 1; i <= 3; i++ {
		_, err := service.CreateActivity(
			"testuser", "test-repo", "testuser",
			"commit", "Test commit",
			nil,
		)
		if err != nil {
			t.Fatalf("CreateActivity %d failed: %v", i, err)
		}
	}

	// Get activities
	activities, err := service.GetActivities("testuser", "test-repo", 10, 0)
	if err != nil {
		t.Fatalf("GetActivities failed: %v", err)
	}
	if len(activities) != 3 {
		t.Errorf("Expected 3 activities, got %d", len(activities))
	}
}

func TestActivityService_GetActivities_Pagination(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewActivityService(db)

	// Create 5 activities
	for i := 1; i <= 5; i++ {
		_, err := service.CreateActivity(
			"testuser", "test-repo", "testuser",
			"commit", "Test commit",
			nil,
		)
		if err != nil {
			t.Fatalf("CreateActivity %d failed: %v", i, err)
		}
	}

	// Get with limit
	activities, err := service.GetActivities("testuser", "test-repo", 2, 0)
	if err != nil {
		t.Fatalf("GetActivities with limit failed: %v", err)
	}
	if len(activities) != 2 {
		t.Errorf("Expected 2 activities with limit, got %d", len(activities))
	}

	// Get with offset
	activities2, err := service.GetActivities("testuser", "test-repo", 10, 3)
	if err != nil {
		t.Fatalf("GetActivities with offset failed: %v", err)
	}
	if len(activities2) != 2 {
		t.Errorf("Expected 2 activities with offset, got %d", len(activities2))
	}
}

func TestActivityService_GetUserActivities(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewActivityService(db)

	// Create activities in different repos for the same user
	for i := 1; i <= 3; i++ {
		_, err := service.CreateActivity(
			"testuser", "repo"+string(rune('0'+i)), "testuser",
			"commit", "Test commit",
			nil,
		)
		if err != nil {
			t.Fatalf("CreateActivity %d failed: %v", i, err)
		}
	}

	// Get user activities across all repos
	activities, err := service.GetUserActivities("testuser", 10, 0)
	if err != nil {
		t.Fatalf("GetUserActivities failed: %v", err)
	}
	if len(activities) != 3 {
		t.Errorf("Expected 3 activities, got %d", len(activities))
	}
}

func TestActivityService_GetUserContributionDays(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewActivityService(db)

	// Create activities on different days
	for i := 0; i < 3; i++ {
		_, err := service.CreateActivity(
			"testuser", "test-repo", "testuser",
			"commit", "Test commit",
			nil,
		)
		if err != nil {
			t.Fatalf("CreateActivity %d failed: %v", i, err)
		}
	}

	// Get contribution days
	days, err := service.GetUserContributionDays("testuser")
	if err != nil {
		t.Fatalf("GetUserContributionDays failed: %v", err)
	}
	if len(days) == 0 {
		t.Error("Expected at least 1 contribution day, got 0")
	}
}

func TestActivityService_GetRecentActivities(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewActivityService(db)

	// Create activities
	for i := 1; i <= 3; i++ {
		_, err := service.CreateActivity(
			"testuser", "test-repo", "testuser",
			"commit", "Test commit",
			nil,
		)
		if err != nil {
			t.Fatalf("CreateActivity %d failed: %v", i, err)
		}
	}

	// Get recent activities
	activities, err := service.GetRecentActivities("testuser", 10, 0)
	if err != nil {
		t.Fatalf("GetRecentActivities failed: %v", err)
	}
	if len(activities) != 3 {
		t.Errorf("Expected 3 activities, got %d", len(activities))
	}
}

func TestActivityService_GetRecentActivities_WithOrganization(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewActivityService(db)

	// Create organization
	accountService := NewAccountService(db, nil)
	_, err := accountService.CreateOrganization("testuser", "testorg", "Test organization")
	if err != nil {
		t.Fatalf("CreateOrganization failed: %v", err)
	}

	// Create activity in org repo
	_, err = service.CreateActivity(
		"testorg", "test-repo", "testuser",
		"commit", "Test commit",
		nil,
	)
	if err != nil {
		t.Fatalf("CreateActivity failed: %v", err)
	}

	// Get recent activities for user (should include org activities)
	activities, err := service.GetRecentActivities("testuser", 10, 0)
	if err != nil {
		t.Fatalf("GetRecentActivities failed: %v", err)
	}
	if len(activities) == 0 {
		t.Error("Expected at least 1 activity from organization, got 0")
	}
}

func TestActivityService_GenerateActivityID_Unique(t *testing.T) {
	// Test that generateActivityID generates unique IDs
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := generateActivityID()
		if ids[id] {
			t.Errorf("Duplicate activity ID generated: %s", id)
		}
		ids[id] = true
	}
}
