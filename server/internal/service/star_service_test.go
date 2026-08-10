package service

import (
	"testing"

	"iforge/iforge/internal/model"
)

func TestStarRepository(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewStarService(db)

	// Create a repository first
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Star the repository
	err := service.StarRepository("testuser", "test-repo", "starrer1")
	if err != nil {
		t.Fatalf("StarRepository failed: %v", err)
	}

	// Verify star was created
	isStarred, err := service.IsStarred("testuser", "test-repo", "starrer1")
	if err != nil {
		t.Fatalf("IsStarred failed: %v", err)
	}
	if !isStarred {
		t.Errorf("Expected repository to be starred, but it wasn't")
	}
}

func TestStarRepository_AlreadyStarred(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewStarService(db)

	// Create a repository first
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Star the repository twice
	err := service.StarRepository("testuser", "test-repo", "starrer1")
	if err != nil {
		t.Fatalf("First StarRepository failed: %v", err)
	}

	// Second star should not error (idempotent)
	err = service.StarRepository("testuser", "test-repo", "starrer1")
	if err != nil {
		t.Fatalf("Second StarRepository should not error: %v", err)
	}

	// Verify only one star exists
	count, err := service.GetStarCount("testuser", "test-repo")
	if err != nil {
		t.Fatalf("GetStarCount failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected star count 1, got %d", count)
	}
}

func TestUnstarRepository(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewStarService(db)

	// Create a repository and star it
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	err := service.StarRepository("testuser", "test-repo", "starrer1")
	if err != nil {
		t.Fatalf("StarRepository failed: %v", err)
	}

	// Unstar the repository
	err = service.UnstarRepository("testuser", "test-repo", "starrer1")
	if err != nil {
		t.Fatalf("UnstarRepository failed: %v", err)
	}

	// Verify star was removed
	isStarred, err := service.IsStarred("testuser", "test-repo", "starrer1")
	if err != nil {
		t.Fatalf("IsStarred failed: %v", err)
	}
	if isStarred {
		t.Errorf("Expected repository to be unstarred, but it was starred")
	}
}

func TestIsStarred(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewStarService(db)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Check before starring
	isStarred, err := service.IsStarred("testuser", "test-repo", "starrer1")
	if err != nil {
		t.Fatalf("IsStarred failed: %v", err)
	}
	if isStarred {
		t.Errorf("Expected repository to not be starred initially")
	}

	// Star and check again
	err = service.StarRepository("testuser", "test-repo", "starrer1")
	if err != nil {
		t.Fatalf("StarRepository failed: %v", err)
	}

	isStarred, err = service.IsStarred("testuser", "test-repo", "starrer1")
	if err != nil {
		t.Fatalf("IsStarred failed: %v", err)
	}
	if !isStarred {
		t.Errorf("Expected repository to be starred after starring")
	}
}

func TestGetStarCount(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewStarService(db)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Check initial count
	count, err := service.GetStarCount("testuser", "test-repo")
	if err != nil {
		t.Fatalf("GetStarCount failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected initial star count 0, got %d", count)
	}

	// Add multiple stars
	for i := 1; i <= 3; i++ {
		err := service.StarRepository("testuser", "test-repo", "starrer"+string(rune('0'+i)))
		if err != nil {
			t.Fatalf("StarRepository failed: %v", err)
		}
	}

	// Check count after starring
	count, err = service.GetStarCount("testuser", "test-repo")
	if err != nil {
		t.Fatalf("GetStarCount failed: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected star count 3, got %d", count)
	}
}

func TestGetStargazers(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewStarService(db)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Add multiple stars
	starrers := []string{"starrer1", "starrer2", "starrer3"}
	for _, starrer := range starrers {
		err := service.StarRepository("testuser", "test-repo", starrer)
		if err != nil {
			t.Fatalf("StarRepository failed: %v", err)
		}
	}

	// Get stargazers
	result, err := service.GetStargazers("testuser", "test-repo")
	if err != nil {
		t.Fatalf("GetStargazers failed: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 stargazers, got %d", len(result))
	}

	// Verify all starrers are present
	starrerSet := make(map[string]bool)
	for _, s := range result {
		starrerSet[s] = true
	}
	for _, expected := range starrers {
		if !starrerSet[expected] {
			t.Errorf("Expected stargazer %s not found", expected)
		}
	}
}

func TestGetStarredRepos(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewStarService(db)

	// Create multiple repositories
	repos := []*model.Repository{
		{UserName: "user1", RepositoryName: "repo1", IsPrivate: false},
		{UserName: "user2", RepositoryName: "repo2", IsPrivate: false},
		{UserName: "user3", RepositoryName: "repo3", IsPrivate: false},
	}
	for _, repo := range repos {
		if err := db.Create(repo).Error; err != nil {
			t.Fatalf("Failed to create repository: %v", err)
		}
	}

	// Star multiple repositories with the same user
	for _, repo := range repos {
		err := service.StarRepository(repo.UserName, repo.RepositoryName, "starrer1")
		if err != nil {
			t.Fatalf("StarRepository failed: %v", err)
		}
	}

	// Get starred repos for starrer1
	result, err := service.GetStarredRepos("starrer1")
	if err != nil {
		t.Fatalf("GetStarredRepos failed: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 starred repos, got %d", len(result))
	}
}

func TestWatchRepository(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewWatchService(db)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Watch the repository
	err := service.WatchRepository("testuser", "test-repo", "watcher1", true)
	if err != nil {
		t.Fatalf("WatchRepository failed: %v", err)
	}

	// Verify watch was created
	isWatching, err := service.IsWatching("testuser", "test-repo", "watcher1")
	if err != nil {
		t.Fatalf("IsWatching failed: %v", err)
	}
	if !isWatching {
		t.Errorf("Expected repository to be watched, but it wasn't")
	}
}

func TestWatchRepository_AlreadyWatching(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewWatchService(db)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Watch the repository twice with different notification settings
	err := service.WatchRepository("testuser", "test-repo", "watcher1", true)
	if err != nil {
		t.Fatalf("First WatchRepository failed: %v", err)
	}

	// Second watch should update notification setting
	err = service.WatchRepository("testuser", "test-repo", "watcher1", false)
	if err != nil {
		t.Fatalf("Second WatchRepository should not error: %v", err)
	}

	// Verify only one watch exists
	count, err := service.GetWatchCount("testuser", "test-repo")
	if err != nil {
		t.Fatalf("GetWatchCount failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected watch count 1, got %d", count)
	}
}

func TestUnwatchRepository(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewWatchService(db)

	// Create a repository and watch it
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	err := service.WatchRepository("testuser", "test-repo", "watcher1", true)
	if err != nil {
		t.Fatalf("WatchRepository failed: %v", err)
	}

	// Unwatch the repository
	err = service.UnwatchRepository("testuser", "test-repo", "watcher1")
	if err != nil {
		t.Fatalf("UnwatchRepository failed: %v", err)
	}

	// Verify watch was removed
	isWatching, err := service.IsWatching("testuser", "test-repo", "watcher1")
	if err != nil {
		t.Fatalf("IsWatching failed: %v", err)
	}
	if isWatching {
		t.Errorf("Expected repository to be unwatched, but it was watched")
	}
}

func TestIsWatching(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewWatchService(db)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Check before watching
	isWatching, err := service.IsWatching("testuser", "test-repo", "watcher1")
	if err != nil {
		t.Fatalf("IsWatching failed: %v", err)
	}
	if isWatching {
		t.Errorf("Expected repository to not be watched initially")
	}

	// Watch and check again
	err = service.WatchRepository("testuser", "test-repo", "watcher1", true)
	if err != nil {
		t.Fatalf("WatchRepository failed: %v", err)
	}

	isWatching, err = service.IsWatching("testuser", "test-repo", "watcher1")
	if err != nil {
		t.Fatalf("IsWatching failed: %v", err)
	}
	if !isWatching {
		t.Errorf("Expected repository to be watched after watching")
	}
}

func TestGetWatchCount(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewWatchService(db)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Check initial count
	count, err := service.GetWatchCount("testuser", "test-repo")
	if err != nil {
		t.Fatalf("GetWatchCount failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected initial watch count 0, got %d", count)
	}

	// Add multiple watches
	for i := 1; i <= 3; i++ {
		err := service.WatchRepository("testuser", "test-repo", "watcher"+string(rune('0'+i)), true)
		if err != nil {
			t.Fatalf("WatchRepository failed: %v", err)
		}
	}

	// Check count after watching
	count, err = service.GetWatchCount("testuser", "test-repo")
	if err != nil {
		t.Fatalf("GetWatchCount failed: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected watch count 3, got %d", count)
	}
}

func TestGetWatchers(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewWatchService(db)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Add multiple watches
	watchers := []string{"watcher1", "watcher2", "watcher3"}
	for _, watcher := range watchers {
		err := service.WatchRepository("testuser", "test-repo", watcher, true)
		if err != nil {
			t.Fatalf("WatchRepository failed: %v", err)
		}
	}

	// Get watchers
	result, err := service.GetWatchers("testuser", "test-repo")
	if err != nil {
		t.Fatalf("GetWatchers failed: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 watchers, got %d", len(result))
	}

	// Verify all watchers are present
	watcherSet := make(map[string]bool)
	for _, w := range result {
		watcherSet[w] = true
	}
	for _, expected := range watchers {
		if !watcherSet[expected] {
			t.Errorf("Expected watcher %s not found", expected)
		}
	}
}
