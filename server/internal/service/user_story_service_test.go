package service

import (
	"errors"
	"testing"
)

func TestCreateUserStory(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewUserStoryService(db)

	// Create a project first
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create a user story
	desc := "User story description"
	acceptance := "Acceptance criteria"
	points := 5
	reporter := "testuser"
	assignee := "developer"

	story, err := service.CreateUserStory(project.ID, "User Story 1", "open", "medium", &desc, &acceptance, &points, &assignee, &reporter, nil, nil)
	if err != nil {
		t.Fatalf("CreateUserStory failed: %v", err)
	}

	if story.ProjectID != project.ID {
		t.Errorf("Expected ProjectID %d, got %d", project.ID, story.ProjectID)
	}
	if story.Title != "User Story 1" {
		t.Errorf("Expected title 'User Story 1', got '%s'", story.Title)
	}
	if story.Status != "open" {
		t.Errorf("Expected status 'open', got '%s'", story.Status)
	}
	if story.Priority != "medium" {
		t.Errorf("Expected priority 'medium', got '%s'", story.Priority)
	}
	if story.Description == nil || *story.Description != desc {
		t.Errorf("Expected description '%s', got %v", desc, story.Description)
	}
	if story.AcceptanceCriteria == nil || *story.AcceptanceCriteria != acceptance {
		t.Errorf("Expected acceptance criteria '%s', got %v", acceptance, story.AcceptanceCriteria)
	}
	if story.StoryPoints == nil || *story.StoryPoints != points {
		t.Errorf("Expected story points %d, got %v", points, story.StoryPoints)
	}
	if story.AssigneeName == nil || *story.AssigneeName != assignee {
		t.Errorf("Expected assignee '%s', got %v", assignee, story.AssigneeName)
	}
	if story.ReporterName != reporter {
		t.Errorf("Expected reporter '%s', got '%s'", reporter, story.ReporterName)
	}
}

func TestCreateUserStory_WithEpic(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewUserStoryService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Note: Epic creation would require epic_service, but for this test we'll just use a fake epic ID
	epicID := 1
	reporter := "testuser"

	story, err := service.CreateUserStory(project.ID, "User Story 1", "open", "medium", nil, nil, nil, nil, &reporter, &epicID, nil)
	if err != nil {
		t.Fatalf("CreateUserStory failed: %v", err)
	}

	if story.EpicID == nil || *story.EpicID != epicID {
		t.Errorf("Expected EpicID %d, got %v", epicID, story.EpicID)
	}
}

func TestCreateUserStory_WithSprint(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewUserStoryService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create a sprint
	sprintService := NewSprintService(db)
	sprint, err := sprintService.CreateSprint(project.ID, "Sprint 1", "active", nil, nil, nil, nil, "testuser")
	if err != nil {
		t.Fatalf("CreateSprint failed: %v", err)
	}

	reporter := "testuser"
	story, err := service.CreateUserStory(project.ID, "User Story 1", "open", "medium", nil, nil, nil, nil, &reporter, nil, &sprint.ID)
	if err != nil {
		t.Fatalf("CreateUserStory failed: %v", err)
	}

	if story.SprintID == nil || *story.SprintID != sprint.ID {
		t.Errorf("Expected SprintID %d, got %v", sprint.ID, story.SprintID)
	}
}

func TestGetUserStory(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewUserStoryService(db)

	// Create a project and user story
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	reporter := "testuser"
	story, err := service.CreateUserStory(project.ID, "User Story 1", "open", "medium", nil, nil, nil, nil, &reporter, nil, nil)
	if err != nil {
		t.Fatalf("CreateUserStory failed: %v", err)
	}

	// Get the user story
	retrieved, err := service.GetUserStory(story.ID)
	if err != nil {
		t.Fatalf("GetUserStory failed: %v", err)
	}

	if retrieved.ID != story.ID {
		t.Errorf("Expected ID %d, got %d", story.ID, retrieved.ID)
	}
	if retrieved.Title != story.Title {
		t.Errorf("Expected title '%s', got '%s'", story.Title, retrieved.Title)
	}
}

func TestGetUserStory_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewUserStoryService(db)

	_, err := service.GetUserStory(999)
	if err == nil {
		t.Fatal("Expected error for non-existent user story, got nil")
	}
	if !errors.Is(err, ErrUserStoryNotFound) {
		t.Errorf("Expected ErrUserStoryNotFound, got %v", err)
	}
}

func TestUpdateUserStory(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewUserStoryService(db)

	// Create a project and user story
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	reporter := "testuser"
	story, err := service.CreateUserStory(project.ID, "User Story 1", "open", "medium", nil, nil, nil, nil, &reporter, nil, nil)
	if err != nil {
		t.Fatalf("CreateUserStory failed: %v", err)
	}

	// Update the user story
	newTitle := "Updated User Story"
	newStatus := "in_progress"
	story.Title = newTitle
	story.Status = newStatus

	if err := service.UpdateUserStory(story); err != nil {
		t.Fatalf("UpdateUserStory failed: %v", err)
	}

	// Retrieve and verify
	updated, err := service.GetUserStory(story.ID)
	if err != nil {
		t.Fatalf("GetUserStory failed: %v", err)
	}

	if updated.Title != newTitle {
		t.Errorf("Expected title '%s', got '%s'", newTitle, updated.Title)
	}
	if updated.Status != newStatus {
		t.Errorf("Expected status '%s', got '%s'", newStatus, updated.Status)
	}
}

func TestListUserStories(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewUserStoryService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create multiple user stories
	reporter := "testuser"
	for i := 1; i <= 3; i++ {
		_, err := service.CreateUserStory(project.ID, "User Story "+string(rune('0'+i)), "open", "medium", nil, nil, nil, nil, &reporter, nil, nil)
		if err != nil {
			t.Fatalf("CreateUserStory failed: %v", err)
		}
	}

	// List user stories
	stories, err := service.ListUserStories(project.ID, nil, nil, nil, false, 0, 0)
	if err != nil {
		t.Fatalf("ListUserStories failed: %v", err)
	}

	if len(stories) != 3 {
		t.Errorf("Expected 3 user stories, got %d", len(stories))
	}
}

func TestListUserStories_WithLimit(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewUserStoryService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create multiple user stories
	reporter := "testuser"
	for i := 1; i <= 5; i++ {
		_, err := service.CreateUserStory(project.ID, "User Story "+string(rune('0'+i)), "open", "medium", nil, nil, nil, nil, &reporter, nil, nil)
		if err != nil {
			t.Fatalf("CreateUserStory failed: %v", err)
		}
	}

	// List with limit
	stories, err := service.ListUserStories(project.ID, nil, nil, nil, false, 2, 0)
	if err != nil {
		t.Fatalf("ListUserStories failed: %v", err)
	}

	if len(stories) != 2 {
		t.Errorf("Expected 2 user stories with limit, got %d", len(stories))
	}
}

func TestListUserStories_FilterByStatus(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewUserStoryService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create user stories with different statuses
	reporter := "testuser"
	_, err = service.CreateUserStory(project.ID, "User Story 1", "open", "medium", nil, nil, nil, nil, &reporter, nil, nil)
	if err != nil {
		t.Fatalf("CreateUserStory failed: %v", err)
	}
	_, err = service.CreateUserStory(project.ID, "User Story 2", "in_progress", "medium", nil, nil, nil, nil, &reporter, nil, nil)
	if err != nil {
		t.Fatalf("CreateUserStory failed: %v", err)
	}
	_, err = service.CreateUserStory(project.ID, "User Story 3", "open", "medium", nil, nil, nil, nil, &reporter, nil, nil)
	if err != nil {
		t.Fatalf("CreateUserStory failed: %v", err)
	}

	// Filter by status
	status := "open"
	stories, err := service.ListUserStories(project.ID, &status, nil, nil, false, 0, 0)
	if err != nil {
		t.Fatalf("ListUserStories failed: %v", err)
	}

	if len(stories) != 2 {
		t.Errorf("Expected 2 open user stories, got %d", len(stories))
	}
}

func TestListUserStories_Backlog(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewUserStoryService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create a sprint
	sprintService := NewSprintService(db)
	sprint, err := sprintService.CreateSprint(project.ID, "Sprint 1", "active", nil, nil, nil, nil, "testuser")
	if err != nil {
		t.Fatalf("CreateSprint failed: %v", err)
	}

	// Create user stories (some in sprint, some not)
	reporter := "testuser"
	_, err = service.CreateUserStory(project.ID, "User Story 1", "open", "medium", nil, nil, nil, nil, &reporter, nil, nil)
	if err != nil {
		t.Fatalf("CreateUserStory failed: %v", err)
	}
	_, err = service.CreateUserStory(project.ID, "User Story 2", "open", "medium", nil, nil, nil, nil, &reporter, nil, &sprint.ID)
	if err != nil {
		t.Fatalf("CreateUserStory failed: %v", err)
	}
	_, err = service.CreateUserStory(project.ID, "User Story 3", "open", "medium", nil, nil, nil, nil, &reporter, nil, nil)
	if err != nil {
		t.Fatalf("CreateUserStory failed: %v", err)
	}

	// List backlog (stories without sprint)
	stories, err := service.ListUserStories(project.ID, nil, nil, nil, true, 0, 0)
	if err != nil {
		t.Fatalf("ListUserStories failed: %v", err)
	}

	if len(stories) != 2 {
		t.Errorf("Expected 2 backlog user stories, got %d", len(stories))
	}
}

func TestDeleteUserStory(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewUserStoryService(db)

	// Create a project and user story
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	reporter := "testuser"
	story, err := service.CreateUserStory(project.ID, "User Story 1", "open", "medium", nil, nil, nil, nil, &reporter, nil, nil)
	if err != nil {
		t.Fatalf("CreateUserStory failed: %v", err)
	}

	// Delete the user story
	if err := service.DeleteUserStory(story.ID); err != nil {
		t.Fatalf("DeleteUserStory failed: %v", err)
	}

	// Verify user story is deleted
	_, err = service.GetUserStory(story.ID)
	if !errors.Is(err, ErrUserStoryNotFound) {
		t.Errorf("Expected ErrUserStoryNotFound after deletion, got %v", err)
	}
}

func TestDeleteUserStory_UnlinksTasks(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewUserStoryService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create a user story
	reporter := "testuser"
	story, err := service.CreateUserStory(project.ID, "User Story 1", "open", "medium", nil, nil, nil, nil, &reporter, nil, nil)
	if err != nil {
		t.Fatalf("CreateUserStory failed: %v", err)
	}

	// Create a task linked to the user story
	taskService := NewTaskService(db)
	storyID := story.ID
	_, err = taskService.CreateTask(project.ID, "Task 1", "open", "medium", "task", nil, nil, nil, nil, &reporter, nil, &storyID, nil, 0, nil)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// Delete the user story
	if err := service.DeleteUserStory(story.ID); err != nil {
		t.Fatalf("DeleteUserStory failed: %v", err)
	}

	// Verify task's user_story_id is unlinked
	tasks, err := service.GetUserStoryTasks(story.ID)
	if err != nil {
		t.Fatalf("GetUserStoryTasks failed: %v", err)
	}
	// Should return empty list since story is deleted
	if len(tasks) != 0 {
		t.Errorf("Expected 0 tasks after story deletion, got %d", len(tasks))
	}
}

func TestUserStoryUpdateStatus(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewUserStoryService(db)

	// Create a project and user story
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	reporter := "testuser"
	story, err := service.CreateUserStory(project.ID, "User Story 1", "open", "medium", nil, nil, nil, nil, &reporter, nil, nil)
	if err != nil {
		t.Fatalf("CreateUserStory failed: %v", err)
	}

	// Update status
	if err := service.UpdateStatus(story.ID, "in_progress"); err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	// Verify status is updated
	updated, err := service.GetUserStory(story.ID)
	if err != nil {
		t.Fatalf("GetUserStory failed: %v", err)
	}
	if updated.Status != "in_progress" {
		t.Errorf("Expected status 'in_progress', got '%s'", updated.Status)
	}
	if updated.ClosedAt != nil {
		t.Errorf("Expected ClosedAt to be nil for in_progress status, got %v", updated.ClosedAt)
	}
}

func TestUserStoryUpdateStatus_Closed(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewUserStoryService(db)

	// Create a project and user story
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	reporter := "testuser"
	story, err := service.CreateUserStory(project.ID, "User Story 1", "open", "medium", nil, nil, nil, nil, &reporter, nil, nil)
	if err != nil {
		t.Fatalf("CreateUserStory failed: %v", err)
	}

	// Update status to closed
	if err := service.UpdateStatus(story.ID, "closed"); err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	// Verify status and closed_at
	updated, err := service.GetUserStory(story.ID)
	if err != nil {
		t.Fatalf("GetUserStory failed: %v", err)
	}
	if updated.Status != "closed" {
		t.Errorf("Expected status 'closed', got '%s'", updated.Status)
	}
	if updated.ClosedAt == nil {
		t.Errorf("Expected ClosedAt to be set for closed status, got nil")
	}
}

func TestGetUserStoryTasks(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewUserStoryService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create a user story
	reporter := "testuser"
	story, err := service.CreateUserStory(project.ID, "User Story 1", "open", "medium", nil, nil, nil, nil, &reporter, nil, nil)
	if err != nil {
		t.Fatalf("CreateUserStory failed: %v", err)
	}

	// Create tasks for the user story
	taskService := NewTaskService(db)
	storyID := story.ID
	for i := 1; i <= 3; i++ {
		_, err := taskService.CreateTask(project.ID, "Task "+string(rune('0'+i)), "open", "medium", "task", nil, nil, nil, nil, &reporter, nil, &storyID, nil, 0, nil)
		if err != nil {
			t.Fatalf("CreateTask failed: %v", err)
		}
	}

	// Get user story tasks
	tasks, err := service.GetUserStoryTasks(story.ID)
	if err != nil {
		t.Fatalf("GetUserStoryTasks failed: %v", err)
	}

	if len(tasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(tasks))
	}
}

func TestGetStoryTaskCounts(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewUserStoryService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create user stories
	reporter := "testuser"
	story1, err := service.CreateUserStory(project.ID, "User Story 1", "open", "medium", nil, nil, nil, nil, &reporter, nil, nil)
	if err != nil {
		t.Fatalf("CreateUserStory failed: %v", err)
	}
	story2, err := service.CreateUserStory(project.ID, "User Story 2", "open", "medium", nil, nil, nil, nil, &reporter, nil, nil)
	if err != nil {
		t.Fatalf("CreateUserStory failed: %v", err)
	}

	// Create tasks for story1
	taskService := NewTaskService(db)
	story1ID := story1.ID
	for i := 1; i <= 2; i++ {
		_, err := taskService.CreateTask(project.ID, "Task "+string(rune('0'+i)), "open", "medium", "task", nil, nil, nil, nil, &reporter, nil, &story1ID, nil, 0, nil)
		if err != nil {
			t.Fatalf("CreateTask failed: %v", err)
		}
	}

	// Create tasks for story2
	story2ID := story2.ID
	_, err = taskService.CreateTask(project.ID, "Task 3", "open", "medium", "task", nil, nil, nil, nil, &reporter, nil, &story2ID, nil, 0, nil)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// Get story task counts
	counts, err := service.GetStoryTaskCounts(project.ID, []int{story1.ID, story2.ID})
	if err != nil {
		t.Fatalf("GetStoryTaskCounts failed: %v", err)
	}

	if counts[story1.ID] != 2 {
		t.Errorf("Expected 2 tasks for story1, got %d", counts[story1.ID])
	}
	if counts[story2.ID] != 1 {
		t.Errorf("Expected 1 task for story2, got %d", counts[story2.ID])
	}
}

func TestGetStoryTaskCounts_EmptyList(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewUserStoryService(db)

	// Get story task counts with empty list
	counts, err := service.GetStoryTaskCounts(1, []int{})
	if err != nil {
		t.Fatalf("GetStoryTaskCounts failed: %v", err)
	}

	if len(counts) != 0 {
		t.Errorf("Expected empty counts map, got %d items", len(counts))
	}
}

func TestListStoriesByEpic(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewUserStoryService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create user stories with epic ID
	epicID := 1
	reporter := "testuser"
	for i := 1; i <= 3; i++ {
		_, err := service.CreateUserStory(project.ID, "User Story "+string(rune('0'+i)), "open", "medium", nil, nil, nil, nil, &reporter, &epicID, nil)
		if err != nil {
			t.Fatalf("CreateUserStory failed: %v", err)
		}
	}

	// List stories by epic
	stories, err := service.ListStoriesByEpic(project.ID, epicID)
	if err != nil {
		t.Fatalf("ListStoriesByEpic failed: %v", err)
	}

	if len(stories) != 3 {
		t.Errorf("Expected 3 stories for epic, got %d", len(stories))
	}
}

func TestUnlinkStoriesFromEpic(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewUserStoryService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create user stories with epic ID
	epicID := 1
	reporter := "testuser"
	for i := 1; i <= 3; i++ {
		_, err := service.CreateUserStory(project.ID, "User Story "+string(rune('0'+i)), "open", "medium", nil, nil, nil, nil, &reporter, &epicID, nil)
		if err != nil {
			t.Fatalf("CreateUserStory failed: %v", err)
		}
	}

	// Unlink stories from epic
	if err := service.UnlinkStoriesFromEpic(epicID); err != nil {
		t.Fatalf("UnlinkStoriesFromEpic failed: %v", err)
	}

	// Verify stories are unlinked
	stories, err := service.ListStoriesByEpic(project.ID, epicID)
	if err != nil {
		t.Fatalf("ListStoriesByEpic failed: %v", err)
	}

	if len(stories) != 0 {
		t.Errorf("Expected 0 stories for epic after unlinking, got %d", len(stories))
	}
}
