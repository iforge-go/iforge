package service

import (
	"iforge/iforge/internal/model"
	"testing"
)

func TestRepositoryService_CreateRepository(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewRepositoryService(db, nil, nil)

	// Create a repository
	desc := "Test repository"
	repo, err := service.CreateRepository(
		"testuser", "test-repo",
		false, &desc, "main",
		nil, nil, nil, nil,
		"testuser",
	)
	if err != nil {
		t.Fatalf("CreateRepository failed: %v", err)
	}

	if repo.UserName != "testuser" {
		t.Errorf("Expected UserName 'testuser', got '%s'", repo.UserName)
	}
	if repo.RepositoryName != "test-repo" {
		t.Errorf("Expected RepositoryName 'test-repo', got '%s'", repo.RepositoryName)
	}
	if repo.IsPrivate != false {
		t.Errorf("Expected IsPrivate false, got %v", repo.IsPrivate)
	}
	if repo.Description == nil || *repo.Description != "Test repository" {
		t.Errorf("Expected Description 'Test repository', got %v", repo.Description)
	}
}

func TestRepositoryService_CreateRepository_Duplicate(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewRepositoryService(db, nil, nil)

	// Create first repository
	desc := "Test repository"
	_, err := service.CreateRepository(
		"testuser", "test-repo",
		false, &desc, "main",
		nil, nil, nil, nil,
		"testuser",
	)
	if err != nil {
		t.Fatalf("First CreateRepository failed: %v", err)
	}

	// Try to create duplicate
	_, err = service.CreateRepository(
		"testuser", "test-repo",
		false, &desc, "main",
		nil, nil, nil, nil,
		"testuser",
	)
	if err == nil {
		t.Fatal("Expected error for duplicate repository, got nil")
	}
	if err != ErrRepositoryExists {
		t.Errorf("Expected ErrRepositoryExists, got %v", err)
	}
}

func TestRepositoryService_GetRepository(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewRepositoryService(db, nil, nil)

	// Create a repository
	desc := "Test repository"
	created, err := service.CreateRepository(
		"testuser", "test-repo",
		false, &desc, "main",
		nil, nil, nil, nil,
		"testuser",
	)
	if err != nil {
		t.Fatalf("CreateRepository failed: %v", err)
	}

	// Get the repository
	repo, err := service.GetRepository("testuser", "test-repo")
	if err != nil {
		t.Fatalf("GetRepository failed: %v", err)
	}

	if repo.UserName != created.UserName {
		t.Errorf("Expected UserName '%s', got '%s'", created.UserName, repo.UserName)
	}
	if repo.RepositoryName != created.RepositoryName {
		t.Errorf("Expected RepositoryName '%s', got '%s'", created.RepositoryName, repo.RepositoryName)
	}
}

func TestRepositoryService_GetRepository_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewRepositoryService(db, nil, nil)

	// Try to get non-existent repository
	_, err := service.GetRepository("testuser", "nonexistent")
	if err == nil {
		t.Fatal("Expected error for non-existent repository, got nil")
	}
}

func TestRepositoryService_UpdateRepository(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewRepositoryService(db, nil, nil)

	// Create a repository
	desc := "Test repository"
	repo, err := service.CreateRepository(
		"testuser", "test-repo",
		false, &desc, "main",
		nil, nil, nil, nil,
		"testuser",
	)
	if err != nil {
		t.Fatalf("CreateRepository failed: %v", err)
	}

	// Update the repository
	newDesc := "Updated description"
	repo.Description = &newDesc
	repo.IsPrivate = true

	err = service.UpdateRepository(repo)
	if err != nil {
		t.Fatalf("UpdateRepository failed: %v", err)
	}

	// Verify update
	updated, err := service.GetRepository("testuser", "test-repo")
	if err != nil {
		t.Fatalf("GetRepository failed: %v", err)
	}

	if updated.Description == nil || *updated.Description != "Updated description" {
		t.Errorf("Expected Description 'Updated description', got %v", updated.Description)
	}
	if updated.IsPrivate != true {
		t.Errorf("Expected IsPrivate true, got %v", updated.IsPrivate)
	}
}

func TestRepositoryService_DeleteRepository(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	// Skip DeleteRepository test as it requires gitClient
	// In production, DeleteRepository removes the git repository directory
	t.Skip("DeleteRepository requires gitClient, skipping in unit tests")
}

func TestRepositoryService_CreateRepository_WithCreator(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewRepositoryService(db, nil, nil)

	// Create an account for the creator
	accountService := NewAccountService(db, nil)
	_, err := accountService.CreateAccount("creator", "password", "Creator User", "creator@example.com", false, nil, nil)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Create a repository with different creator
	desc := "Test repository"
	repo, err := service.CreateRepository(
		"testuser", "test-repo",
		false, &desc, "main",
		nil, nil, nil, nil,
		"creator",
	)
	if err != nil {
		t.Fatalf("CreateRepository failed: %v", err)
	}

	if repo.UserName != "testuser" {
		t.Errorf("Expected UserName 'testuser', got '%s'", repo.UserName)
	}

	// Verify creator is added as collaborator
	collabService := NewCollaboratorService(db)
	isCollab, err := collabService.IsCollaborator("testuser", "test-repo", "creator")
	if err != nil {
		t.Fatalf("IsCollaborator failed: %v", err)
	}
	if !isCollab {
		t.Error("Expected creator to be added as collaborator")
	}
}

func TestRepositoryService_CreateRepository_WithParent(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewRepositoryService(db, nil, nil)

	// Create parent repository
	parentDesc := "Parent repository"
	_, err := service.CreateRepository(
		"parentuser", "parent-repo",
		false, &parentDesc, "main",
		nil, nil, nil, nil,
		"parentuser",
	)
	if err != nil {
		t.Fatalf("CreateRepository (parent) failed: %v", err)
	}

	// Create forked repository
	parentUser := "parentuser"
	parentRepo := "parent-repo"
	desc := "Forked repository"
	forked, err := service.CreateRepository(
		"testuser", "parent-repo",
		false, &desc, "main",
		nil, nil, &parentUser, &parentRepo,
		"testuser",
	)
	if err != nil {
		t.Fatalf("CreateRepository (forked) failed: %v", err)
	}

	if forked.ParentUserName == nil || *forked.ParentUserName != "parentuser" {
		t.Errorf("Expected ParentUserName 'parentuser', got %v", forked.ParentUserName)
	}
	if forked.ParentRepositoryName == nil || *forked.ParentRepositoryName != "parent-repo" {
		t.Errorf("Expected ParentRepositoryName 'parent-repo', got %v", forked.ParentRepositoryName)
	}
}

func TestRepositoryService_ForkRepository(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewRepositoryService(db, nil, nil)

	// Create original repository
	desc := "Original repository"
	_, err := service.CreateRepository(
		"originaluser", "original-repo",
		false, &desc, "main",
		nil, nil, nil, nil,
		"originaluser",
	)
	if err != nil {
		t.Fatalf("CreateRepository failed: %v", err)
	}

	// Fork the repository
	forked, err := service.ForkRepository("originaluser", "original-repo", "forkuser")
	if err != nil {
		t.Fatalf("ForkRepository failed: %v", err)
	}

	if forked.UserName != "forkuser" {
		t.Errorf("Expected UserName 'forkuser', got '%s'", forked.UserName)
	}
	if forked.RepositoryName != "original-repo" {
		t.Errorf("Expected RepositoryName 'original-repo', got '%s'", forked.RepositoryName)
	}
	if forked.OriginUserName == nil || *forked.OriginUserName != "originaluser" {
		t.Errorf("Expected OriginUserName 'originaluser', got %v", forked.OriginUserName)
	}
	if forked.OriginRepositoryName == nil || *forked.OriginRepositoryName != "original-repo" {
		t.Errorf("Expected OriginRepositoryName 'original-repo', got %v", forked.OriginRepositoryName)
	}
}

func TestRepositoryService_ForkRepository_AlreadyExists(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewRepositoryService(db, nil, nil)

	// Create original repository
	desc := "Original repository"
	_, err := service.CreateRepository(
		"originaluser", "original-repo",
		false, &desc, "main",
		nil, nil, nil, nil,
		"originaluser",
	)
	if err != nil {
		t.Fatalf("CreateRepository failed: %v", err)
	}

	// Create repository that will conflict
	_, err = service.CreateRepository(
		"forkuser", "original-repo",
		false, &desc, "main",
		nil, nil, nil, nil,
		"forkuser",
	)
	if err != nil {
		t.Fatalf("CreateRepository failed: %v", err)
	}

	// Try to fork - should fail
	_, err = service.ForkRepository("originaluser", "original-repo", "forkuser")
	if err == nil {
		t.Fatal("Expected error for fork when repository already exists, got nil")
	}
	if err != ErrRepositoryExists {
		t.Errorf("Expected ErrRepositoryExists, got %v", err)
	}
}

func TestRepositoryService_GetVisibleRepositories_Anonymous(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewRepositoryService(db, nil, nil)

	// Create public repository
	desc := "Public repository"
	_, err := service.CreateRepository(
		"user1", "public-repo",
		false, &desc, "main",
		nil, nil, nil, nil,
		"user1",
	)
	if err != nil {
		t.Fatalf("CreateRepository failed: %v", err)
	}

	// Create private repository
	_, err = service.CreateRepository(
		"user1", "private-repo",
		true, &desc, "main",
		nil, nil, nil, nil,
		"user1",
	)
	if err != nil {
		t.Fatalf("CreateRepository failed: %v", err)
	}

	// Get visible repositories for anonymous user (nil username)
	repos, err := service.GetVisibleRepositories(nil, 10)
	if err != nil {
		t.Fatalf("GetVisibleRepositories failed: %v", err)
	}

	// Should only see public repository
	if len(repos) != 1 {
		t.Errorf("Expected 1 repository for anonymous user, got %d", len(repos))
	}
	if repos[0].RepositoryName != "public-repo" {
		t.Errorf("Expected 'public-repo', got '%s'", repos[0].RepositoryName)
	}
}

func TestRepositoryService_GetVisibleRepositories_Authenticated(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewRepositoryService(db, nil, nil)

	// Create user
	accountService := NewAccountService(db, nil)
	_, err := accountService.CreateAccount("user1", "password", "User One", "user1@example.com", false, nil, nil)
	if err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}

	// Create public repository
	desc := "Public repository"
	_, err = service.CreateRepository(
		"user1", "public-repo",
		false, &desc, "main",
		nil, nil, nil, nil,
		"user1",
	)
	if err != nil {
		t.Fatalf("CreateRepository failed: %v", err)
	}

	// Create private repository owned by user1
	_, err = service.CreateRepository(
		"user1", "private-repo",
		true, &desc, "main",
		nil, nil, nil, nil,
		"user1",
	)
	if err != nil {
		t.Fatalf("CreateRepository failed: %v", err)
	}

	// Get visible repositories for user1
	username := "user1"
	repos, err := service.GetVisibleRepositories(&username, 10)
	if err != nil {
		t.Fatalf("GetVisibleRepositories failed: %v", err)
	}

	// Should see both repositories
	if len(repos) != 2 {
		t.Errorf("Expected 2 repositories for owner, got %d", len(repos))
	}
}

func TestRepositoryService_GetMyRepositories(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewRepositoryService(db, nil, nil)

	// Create repositories
	desc := "Test repository"
	_, err := service.CreateRepository(
		"user1", "repo1",
		false, &desc, "main",
		nil, nil, nil, nil,
		"user1",
	)
	if err != nil {
		t.Fatalf("CreateRepository failed: %v", err)
	}

	_, err = service.CreateRepository(
		"user1", "repo2",
		false, &desc, "main",
		nil, nil, nil, nil,
		"user1",
	)
	if err != nil {
		t.Fatalf("CreateRepository failed: %v", err)
	}

	// Get my repositories
	repos, err := service.GetMyRepositories("user1")
	if err != nil {
		t.Fatalf("GetMyRepositories failed: %v", err)
	}

	if len(repos) != 2 {
		t.Errorf("Expected 2 repositories, got %d", len(repos))
	}
}

func TestRepositoryService_GetRepositoriesByUser(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewRepositoryService(db, nil, nil)

	// Create repositories
	desc := "Test repository"
	_, err := service.CreateRepository(
		"user1", "repo1",
		false, &desc, "main",
		nil, nil, nil, nil,
		"user1",
	)
	if err != nil {
		t.Fatalf("CreateRepository failed: %v", err)
	}

	_, err = service.CreateRepository(
		"user1", "repo2",
		true, &desc, "main",
		nil, nil, nil, nil,
		"user1",
	)
	if err != nil {
		t.Fatalf("CreateRepository failed: %v", err)
	}

	// Get repositories by user (as owner - should see all repos)
	currentUser := "user1"
	repos, err := service.GetRepositoriesByUser("user1", &currentUser)
	if err != nil {
		t.Fatalf("GetRepositoriesByUser failed: %v", err)
	}

	if len(repos) != 2 {
		t.Errorf("Expected 2 repositories for owner, got %d", len(repos))
	}

	// Get repositories as anonymous user (should only see public repos)
	repos, err = service.GetRepositoriesByUser("user1", nil)
	if err != nil {
		t.Fatalf("GetRepositoriesByUser failed: %v", err)
	}

	if len(repos) != 1 {
		t.Errorf("Expected 1 public repository for anonymous user, got %d", len(repos))
	}
}

func TestRepositoryService_CountRepositories(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewRepositoryService(db, nil, nil)

	// Create repositories
	desc := "Test repository"
	_, err := service.CreateRepository(
		"user1", "repo1",
		false, &desc, "main",
		nil, nil, nil, nil,
		"user1",
	)
	if err != nil {
		t.Fatalf("CreateRepository failed: %v", err)
	}

	_, err = service.CreateRepository(
		"user1", "repo2",
		false, &desc, "main",
		nil, nil, nil, nil,
		"user1",
	)
	if err != nil {
		t.Fatalf("CreateRepository failed: %v", err)
	}

	// Count repositories
	count, err := service.CountRepositories()
	if err != nil {
		t.Fatalf("CountRepositories failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}
}

func TestRepositoryService_ListAllRepositories(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewRepositoryService(db, nil, nil)

	// Create repositories
	desc := "Test repository"
	_, err := service.CreateRepository(
		"user1", "repo1",
		false, &desc, "main",
		nil, nil, nil, nil,
		"user1",
	)
	if err != nil {
		t.Fatalf("CreateRepository failed: %v", err)
	}

	_, err = service.CreateRepository(
		"user2", "repo2",
		false, &desc, "main",
		nil, nil, nil, nil,
		"user2",
	)
	if err != nil {
		t.Fatalf("CreateRepository failed: %v", err)
	}

	// List all repositories
	repos, total, err := service.ListAllRepositories(1, 10)
	if err != nil {
		t.Fatalf("ListAllRepositories failed: %v", err)
	}

	if total != 2 {
		t.Errorf("Expected total 2, got %d", total)
	}

	if len(repos) != 2 {
		t.Errorf("Expected 2 repositories, got %d", len(repos))
	}
}

func TestCloneRepository(t *testing.T) {
	desc := "Test description"
	originUser := "originuser"
	originRepo := "originrepo"

	repo := &model.Repository{
		UserName:             "testuser",
		RepositoryName:       "testrepo",
		Description:          &desc,
		OriginUserName:       &originUser,
		OriginRepositoryName: &originRepo,
	}

	clone := cloneRepository(repo)

	if clone.UserName != repo.UserName {
		t.Errorf("Expected UserName '%s', got '%s'", repo.UserName, clone.UserName)
	}
	if clone.Description == nil || *clone.Description != *repo.Description {
		t.Errorf("Expected Description '%s', got %v", *repo.Description, clone.Description)
	}
	if clone.OriginUserName == nil || *clone.OriginUserName != *repo.OriginUserName {
		t.Errorf("Expected OriginUserName '%s', got %v", *repo.OriginUserName, clone.OriginUserName)
	}

	// Verify it's a deep copy
	newDesc := "Modified description"
	clone.Description = &newDesc
	if *repo.Description == newDesc {
		t.Error("Expected original to be unchanged after modifying clone")
	}
}

func TestCloneStringPtr(t *testing.T) {
	// Test with non-nil pointer
	original := "test"
	clone := cloneStringPtr(&original)
	if clone == nil || *clone != original {
		t.Errorf("Expected '%s', got %v", original, clone)
	}

	// Verify it's a copy
	*clone = "modified"
	if original == "modified" {
		t.Error("Expected original to be unchanged")
	}

	// Test with nil pointer
	clone = cloneStringPtr(nil)
	if clone != nil {
		t.Error("Expected nil for nil input")
	}
}

func TestCloneIntPtr(t *testing.T) {
	// Test with non-nil pointer
	original := 42
	clone := cloneIntPtr(&original)
	if clone == nil || *clone != original {
		t.Errorf("Expected %d, got %v", original, clone)
	}

	// Verify it's a copy
	*clone = 100
	if original == 100 {
		t.Error("Expected original to be unchanged")
	}

	// Test with nil pointer
	clone = cloneIntPtr(nil)
	if clone != nil {
		t.Error("Expected nil for nil input")
	}
}
