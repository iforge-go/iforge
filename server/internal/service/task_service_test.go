package service

import (
	"errors"
	"testing"
	"time"

	"iforge/iforge/internal/model"
)

func TestCreateTask(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewTaskService(db)

	// Create a project first
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create a task
	desc := "Task description"
	storyPoints := 5
	estimatedHours := 8.0
	reporter := "testuser"
	task, err := service.CreateTask(project.ID, "Task 1", "open", "medium", "task", &desc, &storyPoints, &estimatedHours, nil, &reporter, nil, nil, nil, 0, nil)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	if task.ProjectID != project.ID {
		t.Errorf("Expected ProjectID %d, got %d", project.ID, task.ProjectID)
	}
	if task.Title != "Task 1" {
		t.Errorf("Expected title 'Task 1', got '%s'", task.Title)
	}
	if task.Status != "open" {
		t.Errorf("Expected status 'open', got '%s'", task.Status)
	}
	if task.Priority != "medium" {
		t.Errorf("Expected priority 'medium', got '%s'", task.Priority)
	}
	if task.TaskType != "task" {
		t.Errorf("Expected taskType 'task', got '%s'", task.TaskType)
	}
	if task.Description == nil || *task.Description != desc {
		t.Errorf("Expected description '%s', got %v", desc, task.Description)
	}
	if task.StoryPoints == nil || *task.StoryPoints != storyPoints {
		t.Errorf("Expected storyPoints %d, got %v", storyPoints, task.StoryPoints)
	}
	if task.TaskID != 1 {
		t.Errorf("Expected TaskID 1, got %d", task.TaskID)
	}
}

func TestCreateTask_MultipleTasks(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewTaskService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create multiple tasks
	reporter := "testuser"
	for i := 1; i <= 3; i++ {
		task, err := service.CreateTask(project.ID, "Task "+string(rune('0'+i)), "open", "medium", "task", nil, nil, nil, nil, &reporter, nil, nil, nil, 0, nil)
		if err != nil {
			t.Fatalf("CreateTask failed: %v", err)
		}
		if task.TaskID != i {
			t.Errorf("Expected TaskID %d, got %d", i, task.TaskID)
		}
	}
}

func TestCreateTask_WithSprint(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewTaskService(db)

	// Create a project and sprint
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	sprintService := NewSprintService(db)
	sprint, err := sprintService.CreateSprint(project.ID, "Sprint 1", "active", nil, nil, nil, nil, "testuser")
	if err != nil {
		t.Fatalf("CreateSprint failed: %v", err)
	}

	// Create a task in the sprint
	reporter := "testuser"
	task, err := service.CreateTask(project.ID, "Task 1", "open", "medium", "task", nil, nil, nil, nil, &reporter, &sprint.ID, nil, nil, 0, nil)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	if task.SprintID == nil || *task.SprintID != sprint.ID {
		t.Errorf("Expected SprintID %d, got %v", sprint.ID, task.SprintID)
	}
}

func TestCreateTask_Subtask(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewTaskService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create parent task
	reporter := "testuser"
	parentTask, err := service.CreateTask(project.ID, "Parent Task", "open", "medium", "task", nil, nil, nil, nil, &reporter, nil, nil, nil, 0, nil)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// Create subtask
	parentID := parentTask.TaskID
	subtask, err := service.CreateTask(project.ID, "Subtask", "open", "low", "subtask", nil, nil, nil, nil, &reporter, nil, nil, &parentID, 0, nil)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	if subtask.ParentID == nil || *subtask.ParentID != parentID {
		t.Errorf("Expected ParentID %d, got %v", parentID, subtask.ParentID)
	}
	// Subtask should inherit parent's root_id
	if subtask.RootID == nil || *subtask.RootID != *parentTask.RootID {
		t.Errorf("Expected RootID %d, got %v", *parentTask.RootID, subtask.RootID)
	}
}

func TestGetTask(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewTaskService(db)

	// Create a project and task
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	reporter := "testuser"
	task, err := service.CreateTask(project.ID, "Task 1", "open", "medium", "task", nil, nil, nil, nil, &reporter, nil, nil, nil, 0, nil)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// Get the task
	retrieved, err := service.GetTask(project.ID, task.TaskID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if retrieved.TaskID != task.TaskID {
		t.Errorf("Expected TaskID %d, got %d", task.TaskID, retrieved.TaskID)
	}
	if retrieved.Title != task.Title {
		t.Errorf("Expected title '%s', got '%s'", task.Title, retrieved.Title)
	}
}

func TestGetTask_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewTaskService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	_, err = service.GetTask(project.ID, 999)
	if err == nil {
		t.Fatal("Expected error for non-existent task, got nil")
	}
	if !errors.Is(err, ErrTaskNotFound) {
		t.Errorf("Expected ErrTaskNotFound, got %v", err)
	}
}

func TestUpdateTask(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewTaskService(db)

	// Create a project and task
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	reporter := "testuser"
	task, err := service.CreateTask(project.ID, "Task 1", "open", "medium", "task", nil, nil, nil, nil, &reporter, nil, nil, nil, 0, nil)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// Update the task
	newTitle := "Updated Task"
	task.Title = newTitle
	task.Status = "in_progress"

	if err := service.UpdateTask(task); err != nil {
		t.Fatalf("UpdateTask failed: %v", err)
	}

	// Retrieve and verify
	updated, err := service.GetTask(project.ID, task.TaskID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if updated.Title != newTitle {
		t.Errorf("Expected title '%s', got '%s'", newTitle, updated.Title)
	}
	if updated.Status != "in_progress" {
		t.Errorf("Expected status 'in_progress', got '%s'", updated.Status)
	}
}

func TestListTasks(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewTaskService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create multiple tasks
	reporter := "testuser"
	for i := 1; i <= 3; i++ {
		_, err := service.CreateTask(project.ID, "Task "+string(rune('0'+i)), "open", "medium", "task", nil, nil, nil, nil, &reporter, nil, nil, nil, 0, nil)
		if err != nil {
			t.Fatalf("CreateTask failed: %v", err)
		}
	}

	// List tasks
	tasks, err := service.ListTasks(project.ID, nil, nil, nil, nil, nil, 0, 0)
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}

	if len(tasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(tasks))
	}
}

func TestListTasks_WithLimit(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewTaskService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create multiple tasks
	reporter := "testuser"
	for i := 1; i <= 5; i++ {
		_, err := service.CreateTask(project.ID, "Task "+string(rune('0'+i)), "open", "medium", "task", nil, nil, nil, nil, &reporter, nil, nil, nil, 0, nil)
		if err != nil {
			t.Fatalf("CreateTask failed: %v", err)
		}
	}

	// List with limit
	tasks, err := service.ListTasks(project.ID, nil, nil, nil, nil, nil, 2, 0)
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}

	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks with limit, got %d", len(tasks))
	}
}

func TestListTasks_FilterByStatus(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewTaskService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create tasks with different statuses
	reporter := "testuser"
	_, err = service.CreateTask(project.ID, "Task 1", "open", "medium", "task", nil, nil, nil, nil, &reporter, nil, nil, nil, 0, nil)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}
	_, err = service.CreateTask(project.ID, "Task 2", "in_progress", "medium", "task", nil, nil, nil, nil, &reporter, nil, nil, nil, 0, nil)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}
	_, err = service.CreateTask(project.ID, "Task 3", "open", "medium", "task", nil, nil, nil, nil, &reporter, nil, nil, nil, 0, nil)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// Filter by status
	status := "open"
	tasks, err := service.ListTasks(project.ID, &status, nil, nil, nil, nil, 0, 0)
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}

	if len(tasks) != 2 {
		t.Errorf("Expected 2 open tasks, got %d", len(tasks))
	}
}

func TestCountTasks(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewTaskService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create multiple tasks
	reporter := "testuser"
	for i := 1; i <= 4; i++ {
		_, err := service.CreateTask(project.ID, "Task "+string(rune('0'+i)), "open", "medium", "task", nil, nil, nil, nil, &reporter, nil, nil, nil, 0, nil)
		if err != nil {
			t.Fatalf("CreateTask failed: %v", err)
		}
	}

	// Count tasks
	count, err := service.CountTasks(project.ID, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CountTasks failed: %v", err)
	}

	if count != 4 {
		t.Errorf("Expected count 4, got %d", count)
	}
}

func TestDeleteTask(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewTaskService(db)

	// Create a project and task
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	reporter := "testuser"
	task, err := service.CreateTask(project.ID, "Task 1", "open", "medium", "task", nil, nil, nil, nil, &reporter, nil, nil, nil, 0, nil)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// Delete the task
	if err := service.DeleteTask(project.ID, task.TaskID); err != nil {
		t.Fatalf("DeleteTask failed: %v", err)
	}

	// Verify task is deleted
	_, err = service.GetTask(project.ID, task.TaskID)
	if !errors.Is(err, ErrTaskNotFound) {
		t.Errorf("Expected ErrTaskNotFound after deletion, got %v", err)
	}
}

func TestDeleteTask_UnlinksSubtasks(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewTaskService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create parent task
	reporter := "testuser"
	parentTask, err := service.CreateTask(project.ID, "Parent Task", "open", "medium", "task", nil, nil, nil, nil, &reporter, nil, nil, nil, 0, nil)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// Create subtask
	parentID := parentTask.TaskID
	subtask, err := service.CreateTask(project.ID, "Subtask", "open", "low", "subtask", nil, nil, nil, nil, &reporter, nil, nil, &parentID, 0, nil)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// Delete parent task
	if err := service.DeleteTask(project.ID, parentTask.TaskID); err != nil {
		t.Fatalf("DeleteTask failed: %v", err)
	}

	// Verify subtask's parent_id is unlinked
	var updatedSubtask model.Task
	if err := db.Where("project_id = ? AND task_id = ?", project.ID, subtask.TaskID).First(&updatedSubtask).Error; err != nil {
		t.Fatalf("Failed to retrieve subtask: %v", err)
	}
	if updatedSubtask.ParentID != nil {
		t.Errorf("Expected subtask parent_id to be nil after parent deletion, got %v", updatedSubtask.ParentID)
	}
}

func TestTaskUpdateStatus(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewTaskService(db)

	// Create a project and task
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	reporter := "testuser"
	task, err := service.CreateTask(project.ID, "Task 1", "open", "medium", "task", nil, nil, nil, nil, &reporter, nil, nil, nil, 0, nil)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// Update status
	if err := service.UpdateStatus(project.ID, task.TaskID, "in_progress", nil); err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	// Verify status is updated
	updated, err := service.GetTask(project.ID, task.TaskID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if updated.Status != "in_progress" {
		t.Errorf("Expected status 'in_progress', got '%s'", updated.Status)
	}
}

func TestTaskUpdateStatus_ClosedStatus(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewTaskService(db)

	// Create a project and task
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	reporter := "testuser"
	closedBy := "testuser"
	task, err := service.CreateTask(project.ID, "Task 1", "open", "medium", "task", nil, nil, nil, nil, &reporter, nil, nil, nil, 0, nil)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// Update status to done (closed status)
	if err := service.UpdateStatus(project.ID, task.TaskID, "done", &closedBy); err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	// Verify status and closed_at are set
	updated, err := service.GetTask(project.ID, task.TaskID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if updated.Status != "done" {
		t.Errorf("Expected status 'done', got '%s'", updated.Status)
	}
	if updated.ClosedAt == nil {
		t.Errorf("Expected ClosedAt to be set for done status, got nil")
	}
	if updated.ClosedByName == nil || *updated.ClosedByName != closedBy {
		t.Errorf("Expected ClosedByName '%s', got %v", closedBy, updated.ClosedByName)
	}
}

func TestUpdateStatus_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewTaskService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	err = service.UpdateStatus(project.ID, 999, "in_progress", nil)
	if err == nil {
		t.Fatal("Expected error for non-existent task, got nil")
	}
	if !errors.Is(err, ErrTaskNotFound) {
		t.Errorf("Expected ErrTaskNotFound, got %v", err)
	}
}

func TestRecordStatusHistory(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewTaskService(db)

	// Create a project and task
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	reporter := "testuser"
	task, err := service.CreateTask(project.ID, "Task 1", "open", "medium", "task", nil, nil, nil, nil, &reporter, nil, nil, nil, 0, nil)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// Record status history
	comment := "Status changed"
	if err := service.RecordStatusHistory(project.ID, task.TaskID, "open", "in_progress", "testuser", &comment); err != nil {
		t.Fatalf("RecordStatusHistory failed: %v", err)
	}

	// Verify history is recorded
	history, err := service.GetStatusHistory(project.ID, task.TaskID)
	if err != nil {
		t.Fatalf("GetStatusHistory failed: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("Expected 1 history record, got %d", len(history))
	}
	if history[0].OldStatus != "open" {
		t.Errorf("Expected OldStatus 'open', got '%s'", history[0].OldStatus)
	}
	if history[0].NewStatus != "in_progress" {
		t.Errorf("Expected NewStatus 'in_progress', got '%s'", history[0].NewStatus)
	}
	if history[0].ChangedByName != "testuser" {
		t.Errorf("Expected ChangedByName 'testuser', got '%s'", history[0].ChangedByName)
	}
}

func TestRecordStatusHistory_Deduplication(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewTaskService(db)

	// Create a project and task
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	reporter := "testuser"
	task, err := service.CreateTask(project.ID, "Task 1", "open", "medium", "task", nil, nil, nil, nil, &reporter, nil, nil, nil, 0, nil)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// Record same status history twice within 5 seconds
	if err := service.RecordStatusHistory(project.ID, task.TaskID, "open", "in_progress", "testuser", nil); err != nil {
		t.Fatalf("RecordStatusHistory failed: %v", err)
	}
	if err := service.RecordStatusHistory(project.ID, task.TaskID, "open", "in_progress", "testuser", nil); err != nil {
		t.Fatalf("RecordStatusHistory failed: %v", err)
	}

	// Verify only one record exists (deduplication)
	history, err := service.GetStatusHistory(project.ID, task.TaskID)
	if err != nil {
		t.Fatalf("GetStatusHistory failed: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("Expected 1 history record (deduplicated), got %d", len(history))
	}
}

func TestGetStatusHistory(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewTaskService(db)

	// Create a project and task
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	reporter := "testuser"
	task, err := service.CreateTask(project.ID, "Task 1", "open", "medium", "task", nil, nil, nil, nil, &reporter, nil, nil, nil, 0, nil)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// Record multiple status changes
	time.Sleep(6 * time.Second) // Wait for deduplication window to pass
	if err := service.RecordStatusHistory(project.ID, task.TaskID, "open", "in_progress", "testuser", nil); err != nil {
		t.Fatalf("RecordStatusHistory failed: %v", err)
	}
	time.Sleep(6 * time.Second)
	if err := service.RecordStatusHistory(project.ID, task.TaskID, "in_progress", "done", "testuser", nil); err != nil {
		t.Fatalf("RecordStatusHistory failed: %v", err)
	}

	// Get history (should be newest first)
	history, err := service.GetStatusHistory(project.ID, task.TaskID)
	if err != nil {
		t.Fatalf("GetStatusHistory failed: %v", err)
	}

	if len(history) != 2 {
		t.Errorf("Expected 2 history records, got %d", len(history))
	}
	// Newest first
	if history[0].NewStatus != "done" {
		t.Errorf("Expected first record NewStatus 'done', got '%s'", history[0].NewStatus)
	}
	if history[1].NewStatus != "in_progress" {
		t.Errorf("Expected second record NewStatus 'in_progress', got '%s'", history[1].NewStatus)
	}
}

func TestGetSubtaskCounts(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewTaskService(db)

	// Create a project
	projectService := NewProjectService(db)
	project, err := projectService.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create parent task
	reporter := "testuser"
	parentTask, err := service.CreateTask(project.ID, "Parent Task", "open", "medium", "task", nil, nil, nil, nil, &reporter, nil, nil, nil, 0, nil)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// Create subtasks
	parentID := parentTask.TaskID
	_, err = service.CreateTask(project.ID, "Subtask 1", "open", "low", "subtask", nil, nil, nil, nil, &reporter, nil, nil, &parentID, 0, nil)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}
	_, err = service.CreateTask(project.ID, "Subtask 2", "done", "low", "subtask", nil, nil, nil, nil, &reporter, nil, nil, &parentID, 0, nil)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// Get subtask counts
	totalMap, completedMap, err := service.GetSubtaskCounts(project.ID, []int{parentTask.TaskID})
	if err != nil {
		t.Fatalf("GetSubtaskCounts failed: %v", err)
	}

	if totalMap[parentTask.TaskID] != 2 {
		t.Errorf("Expected total count 2, got %d", totalMap[parentTask.TaskID])
	}
	if completedMap[parentTask.TaskID] != 1 {
		t.Errorf("Expected completed count 1, got %d", completedMap[parentTask.TaskID])
	}
}

func TestGetSubtaskCounts_EmptyList(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewTaskService(db)

	// Get subtask counts with empty list
	totalMap, completedMap, err := service.GetSubtaskCounts(1, []int{})
	if err != nil {
		t.Fatalf("GetSubtaskCounts failed: %v", err)
	}

	if len(totalMap) != 0 {
		t.Errorf("Expected empty totalMap, got %d items", len(totalMap))
	}
	if len(completedMap) != 0 {
		t.Errorf("Expected empty completedMap, got %d items", len(completedMap))
	}
}
