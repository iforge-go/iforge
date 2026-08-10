package service

import (
	"errors"
	"testing"
	"time"

	"iforge/iforge/internal/model"
)

func TestCreateSprint(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSprintService(db)

	// Create a project first
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create a sprint
	desc := "Sprint description"
	goal := "Sprint goal"
	now := time.Now()
	startDate := now
	endDate := now.AddDate(0, 0, 14)

	sprint, err := service.CreateSprint(project.ID, "Sprint 1", "open", &desc, &goal, &startDate, &endDate, "testuser")
	if err != nil {
		t.Fatalf("CreateSprint failed: %v", err)
	}

	if sprint.ProjectID != project.ID {
		t.Errorf("Expected ProjectID %d, got %d", project.ID, sprint.ProjectID)
	}
	if sprint.Title != "Sprint 1" {
		t.Errorf("Expected title 'Sprint 1', got '%s'", sprint.Title)
	}
	if sprint.Status != "open" {
		t.Errorf("Expected status 'open', got '%s'", sprint.Status)
	}
	if sprint.Description == nil || *sprint.Description != desc {
		t.Errorf("Expected description '%s', got %v", desc, sprint.Description)
	}
	if sprint.Goal == nil || *sprint.Goal != goal {
		t.Errorf("Expected goal '%s', got %v", goal, sprint.Goal)
	}
	if sprint.CreatedBy != "testuser" {
		t.Errorf("Expected createdBy 'testuser', got '%s'", sprint.CreatedBy)
	}
}

func TestCreateSprint_MinimalFields(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSprintService(db)

	// Create a project first
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create a sprint with minimal fields
	sprint, err := service.CreateSprint(project.ID, "Sprint 1", "open", nil, nil, nil, nil, "testuser")
	if err != nil {
		t.Fatalf("CreateSprint failed: %v", err)
	}

	if sprint.Description != nil {
		t.Errorf("Expected nil description, got %v", sprint.Description)
	}
	if sprint.Goal != nil {
		t.Errorf("Expected nil goal, got %v", sprint.Goal)
	}
}

func TestGetSprint(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSprintService(db)

	// Create a project and sprint
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	sprint, err := service.CreateSprint(project.ID, "Sprint 1", "open", nil, nil, nil, nil, "testuser")
	if err != nil {
		t.Fatalf("CreateSprint failed: %v", err)
	}

	// Get the sprint
	retrieved, err := service.GetSprint(sprint.ID)
	if err != nil {
		t.Fatalf("GetSprint failed: %v", err)
	}

	if retrieved.ID != sprint.ID {
		t.Errorf("Expected ID %d, got %d", sprint.ID, retrieved.ID)
	}
	if retrieved.Title != sprint.Title {
		t.Errorf("Expected title '%s', got '%s'", sprint.Title, retrieved.Title)
	}
}

func TestGetSprint_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSprintService(db)

	_, err := service.GetSprint(999)
	if err == nil {
		t.Fatal("Expected error for non-existent sprint, got nil")
	}
	if !errors.Is(err, ErrSprintNotFound) {
		t.Errorf("Expected ErrSprintNotFound, got %v", err)
	}
}

func TestUpdateSprint(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSprintService(db)

	// Create a project and sprint
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	sprint, err := service.CreateSprint(project.ID, "Sprint 1", "open", nil, nil, nil, nil, "testuser")
	if err != nil {
		t.Fatalf("CreateSprint failed: %v", err)
	}

	// Update the sprint
	newTitle := "Updated Sprint"
	sprint.Title = newTitle
	sprint.Status = "active"

	if err := service.UpdateSprint(sprint); err != nil {
		t.Fatalf("UpdateSprint failed: %v", err)
	}

	// Retrieve and verify
	updated, err := service.GetSprint(sprint.ID)
	if err != nil {
		t.Fatalf("GetSprint failed: %v", err)
	}

	if updated.Title != newTitle {
		t.Errorf("Expected title '%s', got '%s'", newTitle, updated.Title)
	}
	if updated.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", updated.Status)
	}
}

func TestListSprints(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSprintService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create multiple sprints
	for i := 1; i <= 3; i++ {
		_, err := service.CreateSprint(project.ID, "Sprint "+string(rune('0'+i)), "open", nil, nil, nil, nil, "testuser")
		if err != nil {
			t.Fatalf("CreateSprint failed: %v", err)
		}
	}

	// List sprints
	sprints, err := service.ListSprints(project.ID, 0, 0)
	if err != nil {
		t.Fatalf("ListSprints failed: %v", err)
	}

	if len(sprints) != 3 {
		t.Errorf("Expected 3 sprints, got %d", len(sprints))
	}
}

func TestListSprints_WithLimit(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSprintService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create multiple sprints
	for i := 1; i <= 5; i++ {
		_, err := service.CreateSprint(project.ID, "Sprint "+string(rune('0'+i)), "open", nil, nil, nil, nil, "testuser")
		if err != nil {
			t.Fatalf("CreateSprint failed: %v", err)
		}
	}

	// List with limit
	sprints, err := service.ListSprints(project.ID, 2, 0)
	if err != nil {
		t.Fatalf("ListSprints failed: %v", err)
	}

	if len(sprints) != 2 {
		t.Errorf("Expected 2 sprints with limit, got %d", len(sprints))
	}
}

func TestDeleteSprint(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSprintService(db)

	// Create a project and sprint
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	sprint, err := service.CreateSprint(project.ID, "Sprint 1", "open", nil, nil, nil, nil, "testuser")
	if err != nil {
		t.Fatalf("CreateSprint failed: %v", err)
	}

	// Delete the sprint
	if err := service.DeleteSprint(sprint.ID); err != nil {
		t.Fatalf("DeleteSprint failed: %v", err)
	}

	// Verify sprint is deleted
	_, err = service.GetSprint(sprint.ID)
	if !errors.Is(err, ErrSprintNotFound) {
		t.Errorf("Expected ErrSprintNotFound after deletion, got %v", err)
	}
}

func TestDeleteSprint_UnlinksTasks(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSprintService(db)

	// Create a project and sprint
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	sprint, err := service.CreateSprint(project.ID, "Sprint 1", "open", nil, nil, nil, nil, "testuser")
	if err != nil {
		t.Fatalf("CreateSprint failed: %v", err)
	}

	// Create a task in the sprint
	taskService := NewTaskService(db)
	reporter := "testuser"
	task, err := taskService.CreateTask(project.ID, "Task 1", "open", "medium", "task", nil, nil, nil, nil, &reporter, &sprint.ID, nil, nil, 0, nil)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// Delete the sprint
	if err := service.DeleteSprint(sprint.ID); err != nil {
		t.Fatalf("DeleteSprint failed: %v", err)
	}

	// Verify task's sprint_id is unlinked
	var updatedTask model.Task
	if err := db.Where("project_id = ? AND task_id = ?", project.ID, task.TaskID).First(&updatedTask).Error; err != nil {
		t.Fatalf("Failed to retrieve task: %v", err)
	}
	if updatedTask.SprintID != nil {
		t.Errorf("Expected task sprint_id to be nil after sprint deletion, got %v", updatedTask.SprintID)
	}
}

func TestUpdateStatus(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSprintService(db)

	// Create a project and sprint
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	sprint, err := service.CreateSprint(project.ID, "Sprint 1", "open", nil, nil, nil, nil, "testuser")
	if err != nil {
		t.Fatalf("CreateSprint failed: %v", err)
	}

	// Update status to active
	if err := service.UpdateStatus(sprint.ID, "active"); err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	// Verify status is updated
	updated, err := service.GetSprint(sprint.ID)
	if err != nil {
		t.Fatalf("GetSprint failed: %v", err)
	}
	if updated.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", updated.Status)
	}
	if updated.CompletedDate != nil {
		t.Errorf("Expected CompletedDate to be nil for active sprint, got %v", updated.CompletedDate)
	}
}

func TestUpdateStatus_Closed(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSprintService(db)

	// Create a project and sprint
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	sprint, err := service.CreateSprint(project.ID, "Sprint 1", "active", nil, nil, nil, nil, "testuser")
	if err != nil {
		t.Fatalf("CreateSprint failed: %v", err)
	}

	// Update status to closed
	if err := service.UpdateStatus(sprint.ID, "closed"); err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	// Verify status and completed_date
	updated, err := service.GetSprint(sprint.ID)
	if err != nil {
		t.Fatalf("GetSprint failed: %v", err)
	}
	if updated.Status != "closed" {
		t.Errorf("Expected status 'closed', got '%s'", updated.Status)
	}
	if updated.CompletedDate == nil {
		t.Errorf("Expected CompletedDate to be set for closed sprint, got nil")
	}
}

func TestUpdateStatus_Completed(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSprintService(db)

	// Create a project and sprint
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	sprint, err := service.CreateSprint(project.ID, "Sprint 1", "active", nil, nil, nil, nil, "testuser")
	if err != nil {
		t.Fatalf("CreateSprint failed: %v", err)
	}

	// Update status to completed (should also set completed_date)
	if err := service.UpdateStatus(sprint.ID, "completed"); err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	// Verify completed_date is set
	updated, err := service.GetSprint(sprint.ID)
	if err != nil {
		t.Fatalf("GetSprint failed: %v", err)
	}
	if updated.CompletedDate == nil {
		t.Errorf("Expected CompletedDate to be set for completed sprint, got nil")
	}
}

func TestGetVelocity(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSprintService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create closed sprints with stories
	for i := 1; i <= 3; i++ {
		sprint, err := service.CreateSprint(project.ID, "Sprint "+string(rune('0'+i)), "closed", nil, nil, nil, nil, "testuser")
		if err != nil {
			t.Fatalf("CreateSprint failed: %v", err)
		}

		// Add user stories
		points := i * 5
		story := &model.UserStory{
			ProjectID:    project.ID,
			SprintID:     &sprint.ID,
			Title:        "Story " + string(rune('0'+i)),
			Status:       "done",
			Priority:     "medium",
			StoryPoints:  &points,
			ReporterName: "testuser",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		if err := db.Create(story).Error; err != nil {
			t.Fatalf("Failed to create story: %v", err)
		}

		// Set completed_date
		if err := service.UpdateStatus(sprint.ID, "closed"); err != nil {
			t.Fatalf("UpdateStatus failed: %v", err)
		}
	}

	// Get velocity data
	velocity, err := service.GetVelocity(project.ID, 6)
	if err != nil {
		t.Fatalf("GetVelocity failed: %v", err)
	}

	if len(velocity) != 3 {
		t.Errorf("Expected 3 velocity data points, got %d", len(velocity))
	}

	// Verify data is in chronological order (oldest first)
	for i := 0; i < len(velocity); i++ {
		if velocity[i].CommittedPoints == 0 {
			t.Errorf("Expected non-zero committed points for velocity data point %d", i)
		}
		if velocity[i].CompletedPoints == 0 {
			t.Errorf("Expected non-zero completed points for velocity data point %d", i)
		}
	}
}

func TestGetVelocity_DefaultCount(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSprintService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Get velocity with count <= 0 (should default to 6)
	velocity, err := service.GetVelocity(project.ID, 0)
	if err != nil {
		t.Fatalf("GetVelocity failed: %v", err)
	}

	// Should return empty list (no sprints)
	if len(velocity) != 0 {
		t.Errorf("Expected 0 velocity data points, got %d", len(velocity))
	}
}

func TestGetSprintReport(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSprintService(db)

	// Create a project and sprint
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	sprint, err := service.CreateSprint(project.ID, "Sprint 1", "active", nil, nil, nil, nil, "testuser")
	if err != nil {
		t.Fatalf("CreateSprint failed: %v", err)
	}

	// Add user stories
	points1 := 5
	points2 := 8
	story1 := &model.UserStory{
		ProjectID:    project.ID,
		SprintID:     &sprint.ID,
		Title:        "Story 1",
		Status:       "done",
		Priority:     "medium",
		StoryPoints:  &points1,
		ReporterName: "testuser",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	story2 := &model.UserStory{
		ProjectID:    project.ID,
		SprintID:     &sprint.ID,
		Title:        "Story 2",
		Status:       "open",
		Priority:     "high",
		StoryPoints:  &points2,
		ReporterName: "testuser",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := db.Create(story1).Error; err != nil {
		t.Fatalf("Failed to create story: %v", err)
	}
	if err := db.Create(story2).Error; err != nil {
		t.Fatalf("Failed to create story: %v", err)
	}

	// Get sprint report
	report, err := service.GetSprintReport(project.ID, sprint.ID)
	if err != nil {
		t.Fatalf("GetSprintReport failed: %v", err)
	}

	// Verify committed points (all stories)
	expectedCommitted := points1 + points2
	if report.CommittedPoints != expectedCommitted {
		t.Errorf("Expected committed points %d, got %d", expectedCommitted, report.CommittedPoints)
	}

	// Verify completed points (only done stories)
	if report.CompletedPoints != points1 {
		t.Errorf("Expected completed points %d, got %d", points1, report.CompletedPoints)
	}
}

func TestGetSprintTasks(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSprintService(db)

	// Create a project and sprint
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	sprint, err := service.CreateSprint(project.ID, "Sprint 1", "active", nil, nil, nil, nil, "testuser")
	if err != nil {
		t.Fatalf("CreateSprint failed: %v", err)
	}

	// Create tasks in the sprint
	taskService := NewTaskService(db)
	reporter := "testuser"
	for i := 1; i <= 3; i++ {
		_, err := taskService.CreateTask(project.ID, "Task "+string(rune('0'+i)), "open", "medium", "task", nil, nil, nil, nil, &reporter, &sprint.ID, nil, nil, 0, nil)
		if err != nil {
			t.Fatalf("CreateTask failed: %v", err)
		}
	}

	// Get sprint tasks
	tasks, err := service.GetSprintTasks(sprint.ID)
	if err != nil {
		t.Fatalf("GetSprintTasks failed: %v", err)
	}

	if len(tasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(tasks))
	}
}

func TestGetSprintStories(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSprintService(db)

	// Create a project and sprint
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	sprint, err := service.CreateSprint(project.ID, "Sprint 1", "active", nil, nil, nil, nil, "testuser")
	if err != nil {
		t.Fatalf("CreateSprint failed: %v", err)
	}

	// Create user stories in the sprint
	points := 5
	for i := 1; i <= 3; i++ {
		story := &model.UserStory{
			ProjectID:    project.ID,
			SprintID:     &sprint.ID,
			Title:        "Story " + string(rune('0'+i)),
			Status:       "open",
			Priority:     "medium",
			StoryPoints:  &points,
			ReporterName: "testuser",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		if err := db.Create(story).Error; err != nil {
			t.Fatalf("Failed to create story: %v", err)
		}
	}

	// Get sprint stories
	stories, err := service.GetSprintStories(project.ID, sprint.ID)
	if err != nil {
		t.Fatalf("GetSprintStories failed: %v", err)
	}

	if len(stories) != 3 {
		t.Errorf("Expected 3 stories, got %d", len(stories))
	}
}
