package service

import (
	"errors"
	"testing"
	"time"
)

func TestCreateEpic(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewEpicService(db)

	// Create a project first
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create an epic
	desc := "Epic description"
	goal := "Epic goal"
	now := time.Now()
	startDate := now
	targetDate := now.AddDate(0, 3, 0)
	owner := "testuser"
	reporter := "testuser"

	epic, err := service.CreateEpic(project.ID, "Epic 1", "open", "high", &desc, &goal, &startDate, &targetDate, &owner, &reporter)
	if err != nil {
		t.Fatalf("CreateEpic failed: %v", err)
	}

	if epic.ProjectID != project.ID {
		t.Errorf("Expected ProjectID %d, got %d", project.ID, epic.ProjectID)
	}
	if epic.Title != "Epic 1" {
		t.Errorf("Expected title 'Epic 1', got '%s'", epic.Title)
	}
	if epic.Status != "open" {
		t.Errorf("Expected status 'open', got '%s'", epic.Status)
	}
	if epic.Priority != "high" {
		t.Errorf("Expected priority 'high', got '%s'", epic.Priority)
	}
	if epic.Description == nil || *epic.Description != desc {
		t.Errorf("Expected description '%s', got %v", desc, epic.Description)
	}
	if epic.Goal == nil || *epic.Goal != goal {
		t.Errorf("Expected goal '%s', got %v", goal, epic.Goal)
	}
	if epic.OwnerName == nil || *epic.OwnerName != owner {
		t.Errorf("Expected owner '%s', got %v", owner, epic.OwnerName)
	}
	if epic.ReporterName != reporter {
		t.Errorf("Expected reporter '%s', got '%s'", reporter, epic.ReporterName)
	}
}

func TestCreateEpic_MinimalFields(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewEpicService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create an epic with minimal fields
	reporter := "testuser"
	epic, err := service.CreateEpic(project.ID, "Epic 1", "open", "medium", nil, nil, nil, nil, nil, &reporter)
	if err != nil {
		t.Fatalf("CreateEpic failed: %v", err)
	}

	if epic.Description != nil {
		t.Errorf("Expected nil description, got %v", epic.Description)
	}
	if epic.Goal != nil {
		t.Errorf("Expected nil goal, got %v", epic.Goal)
	}
	if epic.OwnerName != nil {
		t.Errorf("Expected nil owner, got %v", epic.OwnerName)
	}
}

func TestGetEpic(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewEpicService(db)

	// Create a project and epic
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	reporter := "testuser"
	epic, err := service.CreateEpic(project.ID, "Epic 1", "open", "high", nil, nil, nil, nil, nil, &reporter)
	if err != nil {
		t.Fatalf("CreateEpic failed: %v", err)
	}

	// Get the epic
	retrieved, err := service.GetEpic(epic.ID)
	if err != nil {
		t.Fatalf("GetEpic failed: %v", err)
	}

	if retrieved.ID != epic.ID {
		t.Errorf("Expected ID %d, got %d", epic.ID, retrieved.ID)
	}
	if retrieved.Title != epic.Title {
		t.Errorf("Expected title '%s', got '%s'", epic.Title, retrieved.Title)
	}
}

func TestGetEpic_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewEpicService(db)

	_, err := service.GetEpic(999)
	if err == nil {
		t.Fatal("Expected error for non-existent epic, got nil")
	}
	if !errors.Is(err, ErrEpicNotFound) {
		t.Errorf("Expected ErrEpicNotFound, got %v", err)
	}
}

func TestUpdateEpic(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewEpicService(db)

	// Create a project and epic
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	reporter := "testuser"
	epic, err := service.CreateEpic(project.ID, "Epic 1", "open", "high", nil, nil, nil, nil, nil, &reporter)
	if err != nil {
		t.Fatalf("CreateEpic failed: %v", err)
	}

	// Update the epic
	newTitle := "Updated Epic"
	newStatus := "in_progress"
	epic.Title = newTitle
	epic.Status = newStatus
	epic.StatusManual = true // Manually set status to avoid being overridden by automatic derivation

	if err := service.UpdateEpic(epic); err != nil {
		t.Fatalf("UpdateEpic failed: %v", err)
	}

	// Retrieve and verify
	updated, err := service.GetEpic(epic.ID)
	if err != nil {
		t.Fatalf("GetEpic failed: %v", err)
	}

	if updated.Title != newTitle {
		t.Errorf("Expected title '%s', got '%s'", newTitle, updated.Title)
	}
	if updated.Status != newStatus {
		t.Errorf("Expected status '%s', got '%s'", newStatus, updated.Status)
	}
}

func TestListEpics(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewEpicService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create multiple epics
	reporter := "testuser"
	for i := 1; i <= 3; i++ {
		_, err := service.CreateEpic(project.ID, "Epic "+string(rune('0'+i)), "open", "high", nil, nil, nil, nil, nil, &reporter)
		if err != nil {
			t.Fatalf("CreateEpic failed: %v", err)
		}
	}

	// List epics
	epics, err := service.ListEpics(project.ID, nil, 0, 0)
	if err != nil {
		t.Fatalf("ListEpics failed: %v", err)
	}

	if len(epics) != 3 {
		t.Errorf("Expected 3 epics, got %d", len(epics))
	}
}

func TestListEpics_WithLimit(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewEpicService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create multiple epics
	reporter := "testuser"
	for i := 1; i <= 5; i++ {
		_, err := service.CreateEpic(project.ID, "Epic "+string(rune('0'+i)), "open", "high", nil, nil, nil, nil, nil, &reporter)
		if err != nil {
			t.Fatalf("CreateEpic failed: %v", err)
		}
	}

	// List with limit
	epics, err := service.ListEpics(project.ID, nil, 2, 0)
	if err != nil {
		t.Fatalf("ListEpics failed: %v", err)
	}

	if len(epics) != 2 {
		t.Errorf("Expected 2 epics with limit, got %d", len(epics))
	}
}

func TestListEpics_FilterByStatus(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewEpicService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create epics with different statuses
	reporter := "testuser"
	_, err = service.CreateEpic(project.ID, "Epic 1", "open", "high", nil, nil, nil, nil, nil, &reporter)
	if err != nil {
		t.Fatalf("CreateEpic failed: %v", err)
	}
	_, err = service.CreateEpic(project.ID, "Epic 2", "in_progress", "high", nil, nil, nil, nil, nil, &reporter)
	if err != nil {
		t.Fatalf("CreateEpic failed: %v", err)
	}
	_, err = service.CreateEpic(project.ID, "Epic 3", "open", "high", nil, nil, nil, nil, nil, &reporter)
	if err != nil {
		t.Fatalf("CreateEpic failed: %v", err)
	}

	// Filter by status
	status := "open"
	epics, err := service.ListEpics(project.ID, &status, 0, 0)
	if err != nil {
		t.Fatalf("ListEpics failed: %v", err)
	}

	if len(epics) != 2 {
		t.Errorf("Expected 2 open epics, got %d", len(epics))
	}
}

func TestDeleteEpic(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewEpicService(db)

	// Create a project and epic
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	reporter := "testuser"
	epic, err := service.CreateEpic(project.ID, "Epic 1", "open", "high", nil, nil, nil, nil, nil, &reporter)
	if err != nil {
		t.Fatalf("CreateEpic failed: %v", err)
	}

	// Delete the epic
	if err := service.DeleteEpic(epic.ID); err != nil {
		t.Fatalf("DeleteEpic failed: %v", err)
	}

	// Verify epic is deleted
	_, err = service.GetEpic(epic.ID)
	if !errors.Is(err, ErrEpicNotFound) {
		t.Errorf("Expected ErrEpicNotFound after deletion, got %v", err)
	}
}

func TestDeleteEpic_UnlinksStories(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewEpicService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create an epic
	reporter := "testuser"
	epic, err := service.CreateEpic(project.ID, "Epic 1", "open", "high", nil, nil, nil, nil, nil, &reporter)
	if err != nil {
		t.Fatalf("CreateEpic failed: %v", err)
	}

	// Create user stories linked to the epic
	userStoryService := NewUserStoryService(db)
	for i := 1; i <= 2; i++ {
		_, err := userStoryService.CreateUserStory(project.ID, "Story "+string(rune('0'+i)), "open", "medium", nil, nil, nil, nil, &reporter, &epic.ID, nil)
		if err != nil {
			t.Fatalf("CreateUserStory failed: %v", err)
		}
	}

	// Delete the epic
	if err := service.DeleteEpic(epic.ID); err != nil {
		t.Fatalf("DeleteEpic failed: %v", err)
	}

	// Verify stories are unlinked (epic_id = NULL)
	stories, err := service.ListEpicStories(project.ID, epic.ID)
	if err != nil {
		t.Fatalf("ListEpicStories failed: %v", err)
	}

	if len(stories) != 0 {
		t.Errorf("Expected 0 stories for epic after deletion, got %d", len(stories))
	}
}

func TestAssignStoryToEpic(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewEpicService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create an epic
	reporter := "testuser"
	epic, err := service.CreateEpic(project.ID, "Epic 1", "open", "high", nil, nil, nil, nil, nil, &reporter)
	if err != nil {
		t.Fatalf("CreateEpic failed: %v", err)
	}

	// Create a user story
	userStoryService := NewUserStoryService(db)
	story, err := userStoryService.CreateUserStory(project.ID, "Story 1", "open", "medium", nil, nil, nil, nil, &reporter, nil, nil)
	if err != nil {
		t.Fatalf("CreateUserStory failed: %v", err)
	}

	// Assign story to epic
	if err := service.AssignStoryToEpic(project.ID, epic.ID, story.ID); err != nil {
		t.Fatalf("AssignStoryToEpic failed: %v", err)
	}

	// Verify story is assigned
	stories, err := service.ListEpicStories(project.ID, epic.ID)
	if err != nil {
		t.Fatalf("ListEpicStories failed: %v", err)
	}

	if len(stories) != 1 {
		t.Errorf("Expected 1 story for epic, got %d", len(stories))
	}
	if stories[0].ID != story.ID {
		t.Errorf("Expected story ID %d, got %d", story.ID, stories[0].ID)
	}
}

func TestAssignStoryToEpic_StoryNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewEpicService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create an epic
	reporter := "testuser"
	epic, err := service.CreateEpic(project.ID, "Epic 1", "open", "high", nil, nil, nil, nil, nil, &reporter)
	if err != nil {
		t.Fatalf("CreateEpic failed: %v", err)
	}

	// Try to assign non-existent story
	err = service.AssignStoryToEpic(project.ID, epic.ID, 999)
	if !errors.Is(err, ErrUserStoryNotFound) {
		t.Errorf("Expected ErrUserStoryNotFound, got %v", err)
	}
}

func TestRemoveStoryFromEpic(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewEpicService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create an epic
	reporter := "testuser"
	epic, err := service.CreateEpic(project.ID, "Epic 1", "open", "high", nil, nil, nil, nil, nil, &reporter)
	if err != nil {
		t.Fatalf("CreateEpic failed: %v", err)
	}

	// Create a user story linked to the epic
	userStoryService := NewUserStoryService(db)
	story, err := userStoryService.CreateUserStory(project.ID, "Story 1", "open", "medium", nil, nil, nil, nil, &reporter, &epic.ID, nil)
	if err != nil {
		t.Fatalf("CreateUserStory failed: %v", err)
	}

	// Remove story from epic
	if err := service.RemoveStoryFromEpic(project.ID, story.ID); err != nil {
		t.Fatalf("RemoveStoryFromEpic failed: %v", err)
	}

	// Verify story is unlinked
	stories, err := service.ListEpicStories(project.ID, epic.ID)
	if err != nil {
		t.Fatalf("ListEpicStories failed: %v", err)
	}

	if len(stories) != 0 {
		t.Errorf("Expected 0 stories for epic after removal, got %d", len(stories))
	}
}

func TestRemoveStoryFromEpic_StoryNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewEpicService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Try to remove non-existent story
	err = service.RemoveStoryFromEpic(project.ID, 999)
	if !errors.Is(err, ErrUserStoryNotFound) {
		t.Errorf("Expected ErrUserStoryNotFound, got %v", err)
	}
}

func TestListEpicStories(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewEpicService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create an epic
	reporter := "testuser"
	epic, err := service.CreateEpic(project.ID, "Epic 1", "open", "high", nil, nil, nil, nil, nil, &reporter)
	if err != nil {
		t.Fatalf("CreateEpic failed: %v", err)
	}

	// Create user stories linked to the epic
	userStoryService := NewUserStoryService(db)
	for i := 1; i <= 3; i++ {
		_, err := userStoryService.CreateUserStory(project.ID, "Story "+string(rune('0'+i)), "open", "medium", nil, nil, nil, nil, &reporter, &epic.ID, nil)
		if err != nil {
			t.Fatalf("CreateUserStory failed: %v", err)
		}
	}

	// List epic stories
	stories, err := service.ListEpicStories(project.ID, epic.ID)
	if err != nil {
		t.Fatalf("ListEpicStories failed: %v", err)
	}

	if len(stories) != 3 {
		t.Errorf("Expected 3 stories for epic, got %d", len(stories))
	}
}

func TestGetEpicSprints(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewEpicService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create an epic
	reporter := "testuser"
	epic, err := service.CreateEpic(project.ID, "Epic 1", "open", "high", nil, nil, nil, nil, nil, &reporter)
	if err != nil {
		t.Fatalf("CreateEpic failed: %v", err)
	}

	// Create a sprint
	sprintService := NewSprintService(db)
	sprint, err := sprintService.CreateSprint(project.ID, "Sprint 1", "active", nil, nil, nil, nil, "testuser")
	if err != nil {
		t.Fatalf("CreateSprint failed: %v", err)
	}

	// Create a user story linked to the epic
	userStoryService := NewUserStoryService(db)
	story, err := userStoryService.CreateUserStory(project.ID, "Story 1", "open", "medium", nil, nil, nil, nil, &reporter, &epic.ID, nil)
	if err != nil {
		t.Fatalf("CreateUserStory failed: %v", err)
	}

	// Create a task linked to the story and sprint
	taskService := NewTaskService(db)
	storyID := story.ID
	_, err = taskService.CreateTask(project.ID, "Task 1", "open", "medium", "task", nil, nil, nil, nil, &reporter, &sprint.ID, &storyID, nil, 0, nil)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// Get epic sprints
	sprints, err := service.GetEpicSprints(project.ID, epic.ID)
	if err != nil {
		t.Fatalf("GetEpicSprints failed: %v", err)
	}

	if len(sprints) != 1 {
		t.Errorf("Expected 1 sprint for epic, got %d", len(sprints))
	}
	if sprints[0].SprintID != sprint.ID {
		t.Errorf("Expected sprint ID %d, got %d", sprint.ID, sprints[0].SprintID)
	}
	if sprints[0].TaskCount != 1 {
		t.Errorf("Expected task count 1, got %d", sprints[0].TaskCount)
	}
}

func TestFillAggregates(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewEpicService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create an epic
	reporter := "testuser"
	epic, err := service.CreateEpic(project.ID, "Epic 1", "open", "high", nil, nil, nil, nil, nil, &reporter)
	if err != nil {
		t.Fatalf("CreateEpic failed: %v", err)
	}

	// Create user stories with different statuses and story points
	userStoryService := NewUserStoryService(db)
	points1 := 5
	points2 := 8
	points3 := 3

	_, err = userStoryService.CreateUserStory(project.ID, "Story 1", "done", "medium", nil, nil, &points1, nil, &reporter, &epic.ID, nil)
	if err != nil {
		t.Fatalf("CreateUserStory failed: %v", err)
	}
	_, err = userStoryService.CreateUserStory(project.ID, "Story 2", "in_progress", "medium", nil, nil, &points2, nil, &reporter, &epic.ID, nil)
	if err != nil {
		t.Fatalf("CreateUserStory failed: %v", err)
	}
	_, err = userStoryService.CreateUserStory(project.ID, "Story 3", "open", "medium", nil, nil, &points3, nil, &reporter, &epic.ID, nil)
	if err != nil {
		t.Fatalf("CreateUserStory failed: %v", err)
	}

	// Get epic (should fill aggregates)
	retrieved, err := service.GetEpic(epic.ID)
	if err != nil {
		t.Fatalf("GetEpic failed: %v", err)
	}

	// Verify aggregates
	if retrieved.TotalStories != 3 {
		t.Errorf("Expected TotalStories 3, got %d", retrieved.TotalStories)
	}
	if retrieved.DoneStories != 1 {
		t.Errorf("Expected DoneStories 1, got %d", retrieved.DoneStories)
	}
	if retrieved.TotalStoryPoints != 16 {
		t.Errorf("Expected TotalStoryPoints 16, got %d", retrieved.TotalStoryPoints)
	}
	if retrieved.DoneStoryPoints != 5 {
		t.Errorf("Expected DoneStoryPoints 5, got %d", retrieved.DoneStoryPoints)
	}
	if retrieved.ProgressPercent != 33 {
		t.Errorf("Expected ProgressPercent 33, got %d", retrieved.ProgressPercent)
	}
}

func TestDeriveStatus(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewEpicService(db)

	tests := []struct {
		name     string
		statuses []string
		expected string
	}{
		{
			name:     "empty statuses returns open",
			statuses: []string{},
			expected: "open",
		},
		{
			name:     "all done returns done",
			statuses: []string{"done", "done", "done"},
			expected: "done",
		},
		{
			name:     "all closed returns closed",
			statuses: []string{"closed", "closed"},
			expected: "closed",
		},
		{
			name:     "mixed done and closed returns done",
			statuses: []string{"done", "closed", "done"},
			expected: "done",
		},
		{
			name:     "any in_progress returns in_progress",
			statuses: []string{"open", "in_progress", "done"},
			expected: "in_progress",
		},
		{
			name:     "all open returns open",
			statuses: []string{"open", "open"},
			expected: "open",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.deriveStatus(tt.statuses)
			if result != tt.expected {
				t.Errorf("deriveStatus(%v) = %s, want %s", tt.statuses, result, tt.expected)
			}
		})
	}
}
