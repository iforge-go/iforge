package service

import (
	"encoding/json"
	"testing"
	"time"

	"iforge/iforge/internal/model"
)

func TestStatsService_InvalidateStatsCache(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewStatsService(db, nil)

	// Create a cache entry
	cache := &model.RepositoryStatsCache{
		UserName:       "testuser",
		RepositoryName: "testrepo",
		StatsJSON:      `{"totalCommits":10}`,
		UpdatedDate:    time.Now(),
	}
	db.Create(cache)

	// Verify cache exists
	var count int64
	db.Model(&model.RepositoryStatsCache{}).
		Where("user_name = ? AND repository_name = ?", "testuser", "testrepo").
		Count(&count)
	if count != 1 {
		t.Errorf("Expected 1 cache entry, got %d", count)
	}

	// Invalidate cache
	service.InvalidateStatsCache("testuser", "testrepo")

	// Verify cache was deleted
	db.Model(&model.RepositoryStatsCache{}).
		Where("user_name = ? AND repository_name = ?", "testuser", "testrepo").
		Count(&count)
	if count != 0 {
		t.Errorf("Expected 0 cache entries after invalidation, got %d", count)
	}
}

func TestStatsService_InvalidateStatsCache_NonExistent(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewStatsService(db, nil)

	// Invalidate non-existent cache (should not error)
	service.InvalidateStatsCache("nonexistent", "repo")

	// Verify no error occurred
	var count int64
	db.Model(&model.RepositoryStatsCache{}).Count(&count)
	if count != 0 {
		t.Errorf("Expected 0 cache entries, got %d", count)
	}
}

func TestStatsService_GetRepositoryStats_CacheHit(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewStatsService(db, nil)

	// Create a valid cache entry
	stats := RepositoryStats{
		TotalCommits:  100,
		TotalFiles:    50,
		DefaultBranch: "main",
		Contributors: []Contributor{
			{Name: "Test User", Email: "test@example.com", Commits: 100},
		},
		CodeFrequency: []CodeFrequency{
			{Timestamp: "2024-01-01", Additions: 1000, Deletions: 500},
		},
	}
	statsJSON, _ := json.Marshal(stats)

	cache := &model.RepositoryStatsCache{
		UserName:       "testuser",
		RepositoryName: "testrepo",
		StatsJSON:      string(statsJSON),
		UpdatedDate:    time.Now(), // Fresh cache
	}
	db.Create(cache)

	// Get stats (should hit cache)
	retrieved, err := service.GetRepositoryStats("testuser", "testrepo")
	if err != nil {
		t.Fatalf("GetRepositoryStats failed: %v", err)
	}

	if retrieved.TotalCommits != 100 {
		t.Errorf("Expected TotalCommits 100, got %d", retrieved.TotalCommits)
	}
	if retrieved.TotalFiles != 50 {
		t.Errorf("Expected TotalFiles 50, got %d", retrieved.TotalFiles)
	}
	if retrieved.DefaultBranch != "main" {
		t.Errorf("Expected DefaultBranch 'main', got '%s'", retrieved.DefaultBranch)
	}
	if len(retrieved.Contributors) != 1 {
		t.Errorf("Expected 1 contributor, got %d", len(retrieved.Contributors))
	}
}

func TestStatsService_GetRepositoryStats_CacheExpired(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewStatsService(db, nil)

	// Create an expired cache entry
	stats := RepositoryStats{
		TotalCommits: 100,
	}
	statsJSON, _ := json.Marshal(stats)

	cache := &model.RepositoryStatsCache{
		UserName:       "testuser",
		RepositoryName: "testrepo",
		StatsJSON:      string(statsJSON),
		UpdatedDate:    time.Now().Add(-10 * time.Minute), // Expired (> 5 minutes old)
	}
	db.Create(cache)

	// Get stats (should miss cache and try to compute)
	// Since we don't have a git client, computeRepositoryStats will fail
	_, err := service.GetRepositoryStats("testuser", "testrepo")
	if err == nil {
		t.Error("Expected error when cache expired and no git client, got nil")
	}
}

func TestStatsService_GetRepositoryStats_CacheCorrupted(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewStatsService(db, nil)

	// Create a cache entry with corrupted JSON
	cache := &model.RepositoryStatsCache{
		UserName:       "testuser",
		RepositoryName: "testrepo",
		StatsJSON:      "invalid-json{",
		UpdatedDate:    time.Now(),
	}
	db.Create(cache)

	// Get stats (should detect corruption and try to recompute)
	_, err := service.GetRepositoryStats("testuser", "testrepo")
	if err == nil {
		t.Error("Expected error when cache corrupted and no git client, got nil")
	}

	// Verify corrupted cache was deleted
	var count int64
	db.Model(&model.RepositoryStatsCache{}).
		Where("user_name = ? AND repository_name = ?", "testuser", "testrepo").
		Count(&count)
	if count != 0 {
		t.Errorf("Expected corrupted cache to be deleted, but found %d", count)
	}
}

func TestStatsService_GetRepositoryStats_NoCache(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewStatsService(db, nil)

	// Get stats with no cache (should try to compute)
	_, err := service.GetRepositoryStats("testuser", "testrepo")
	if err == nil {
		t.Error("Expected error when no cache and no git client, got nil")
	}
}

func TestStatsService_CacheIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewStatsService(db, nil)

	// Create cache for different repos
	stats1 := RepositoryStats{TotalCommits: 100}
	stats2 := RepositoryStats{TotalCommits: 200}
	statsJSON1, _ := json.Marshal(stats1)
	statsJSON2, _ := json.Marshal(stats2)

	db.Create(&model.RepositoryStatsCache{
		UserName:       "user1",
		RepositoryName: "repo1",
		StatsJSON:      string(statsJSON1),
		UpdatedDate:    time.Now(),
	})
	db.Create(&model.RepositoryStatsCache{
		UserName:       "user2",
		RepositoryName: "repo2",
		StatsJSON:      string(statsJSON2),
		UpdatedDate:    time.Now(),
	})

	// Get stats for user1/repo1
	retrieved1, err := service.GetRepositoryStats("user1", "repo1")
	if err != nil {
		t.Fatalf("GetRepositoryStats for user1/repo1 failed: %v", err)
	}
	if retrieved1.TotalCommits != 100 {
		t.Errorf("Expected TotalCommits 100 for user1/repo1, got %d", retrieved1.TotalCommits)
	}

	// Get stats for user2/repo2
	retrieved2, err := service.GetRepositoryStats("user2", "repo2")
	if err != nil {
		t.Fatalf("GetRepositoryStats for user2/repo2 failed: %v", err)
	}
	if retrieved2.TotalCommits != 200 {
		t.Errorf("Expected TotalCommits 200 for user2/repo2, got %d", retrieved2.TotalCommits)
	}

	// Invalidate cache for user1/repo1
	service.InvalidateStatsCache("user1", "repo1")

	// Verify user2/repo2 cache still exists
	var count int64
	db.Model(&model.RepositoryStatsCache{}).
		Where("user_name = ? AND repository_name = ?", "user2", "repo2").
		Count(&count)
	if count != 1 {
		t.Errorf("Expected user2/repo2 cache to still exist, got %d", count)
	}
}
