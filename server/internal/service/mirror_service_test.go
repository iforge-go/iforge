package service

import (
	"testing"
	"time"

	"iforge/iforge/internal/model"
)

func TestMirrorService_NewMirrorService(t *testing.T) {
	db := setupTestDB(t)
	service := NewMirrorService(db)

	if service == nil {
		t.Fatal("Expected MirrorService to be created")
	}
	if service.db != db {
		t.Error("Expected db to be set")
	}
}

func TestMirrorService_CreateMirror(t *testing.T) {
	db := setupTestDB(t)
	service := NewMirrorService(db)

	mirror := &model.RepositoryMirror{
		UserName:       "testuser",
		RepositoryName: "testrepo",
		MirrorURL:      "https://github.com/example/repo.git",
		SyncInterval:   60,
		Enabled:        true,
		SyncOnPush:     false,
	}

	err := service.CreateMirror(mirror)
	if err != nil {
		t.Fatalf("CreateMirror failed: %v", err)
	}

	// Verify LastSyncDate and NextSyncDate are set
	if mirror.LastSyncDate.IsZero() {
		t.Error("Expected LastSyncDate to be set")
	}
	if mirror.NextSyncDate.IsZero() {
		t.Error("Expected NextSyncDate to be set")
	}

	// Verify NextSyncDate is approximately SyncInterval minutes after LastSyncDate
	expectedDiff := time.Duration(mirror.SyncInterval) * time.Minute
	actualDiff := mirror.NextSyncDate.Sub(mirror.LastSyncDate)
	if actualDiff < expectedDiff-time.Second || actualDiff > expectedDiff+time.Second {
		t.Errorf("Expected NextSyncDate to be ~%v after LastSyncDate, got %v", expectedDiff, actualDiff)
	}
}

func TestMirrorService_GetMirror(t *testing.T) {
	db := setupTestDB(t)
	service := NewMirrorService(db)

	// Create a mirror first
	mirror := &model.RepositoryMirror{
		UserName:       "testuser",
		RepositoryName: "testrepo",
		MirrorURL:      "https://github.com/example/repo.git",
		SyncInterval:   60,
		Enabled:        true,
	}
	service.CreateMirror(mirror)

	// Test getting existing mirror
	retrieved, err := service.GetMirror("testuser", "testrepo")
	if err != nil {
		t.Fatalf("GetMirror failed: %v", err)
	}
	if retrieved.MirrorURL != mirror.MirrorURL {
		t.Errorf("Expected MirrorURL %s, got %s", mirror.MirrorURL, retrieved.MirrorURL)
	}

	// Test getting non-existent mirror
	_, err = service.GetMirror("nonexistent", "repo")
	if err != ErrMirrorNotFound {
		t.Errorf("Expected ErrMirrorNotFound, got %v", err)
	}
}

func TestMirrorService_UpdateMirror(t *testing.T) {
	db := setupTestDB(t)
	service := NewMirrorService(db)

	// Create a mirror first
	mirror := &model.RepositoryMirror{
		UserName:       "testuser",
		RepositoryName: "testrepo",
		MirrorURL:      "https://github.com/example/repo.git",
		SyncInterval:   60,
		Enabled:        true,
	}
	service.CreateMirror(mirror)

	// Update the mirror
	mirror.MirrorURL = "https://gitlab.com/example/repo.git"
	mirror.SyncInterval = 120
	mirror.Enabled = false

	err := service.UpdateMirror(mirror)
	if err != nil {
		t.Fatalf("UpdateMirror failed: %v", err)
	}

	// Verify update
	updated, err := service.GetMirror("testuser", "testrepo")
	if err != nil {
		t.Fatalf("GetMirror after update failed: %v", err)
	}
	if updated.MirrorURL != "https://gitlab.com/example/repo.git" {
		t.Errorf("Expected updated MirrorURL, got %s", updated.MirrorURL)
	}
	if updated.SyncInterval != 120 {
		t.Errorf("Expected SyncInterval 120, got %d", updated.SyncInterval)
	}
	if updated.Enabled != false {
		t.Error("Expected Enabled to be false")
	}
}

func TestMirrorService_DeleteMirror(t *testing.T) {
	db := setupTestDB(t)
	service := NewMirrorService(db)

	// Create a mirror first
	mirror := &model.RepositoryMirror{
		UserName:       "testuser",
		RepositoryName: "testrepo",
		MirrorURL:      "https://github.com/example/repo.git",
		SyncInterval:   60,
		Enabled:        true,
	}
	service.CreateMirror(mirror)

	// Delete the mirror
	err := service.DeleteMirror("testuser", "testrepo")
	if err != nil {
		t.Fatalf("DeleteMirror failed: %v", err)
	}

	// Verify deletion
	_, err = service.GetMirror("testuser", "testrepo")
	if err != ErrMirrorNotFound {
		t.Errorf("Expected ErrMirrorNotFound after deletion, got %v", err)
	}
}

func TestMirrorService_SyncMirror(t *testing.T) {
	db := setupTestDB(t)
	service := NewMirrorService(db)

	// Create a mirror first
	mirror := &model.RepositoryMirror{
		UserName:       "testuser",
		RepositoryName: "testrepo",
		MirrorURL:      "https://github.com/example/repo.git",
		SyncInterval:   60,
		Enabled:        true,
	}
	service.CreateMirror(mirror)

	// Wait a bit to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Sync the mirror
	err := service.SyncMirror("testuser", "testrepo")
	if err != nil {
		t.Fatalf("SyncMirror failed: %v", err)
	}

	// Verify sync updated timestamps
	synced, err := service.GetMirror("testuser", "testrepo")
	if err != nil {
		t.Fatalf("GetMirror after sync failed: %v", err)
	}

	// LastSyncDate should be updated
	if !synced.LastSyncDate.After(mirror.LastSyncDate) {
		t.Error("Expected LastSyncDate to be updated after sync")
	}

	// NextSyncDate should be updated
	if !synced.NextSyncDate.After(mirror.NextSyncDate) {
		t.Error("Expected NextSyncDate to be updated after sync")
	}
}

func TestMirrorService_SyncMirror_NotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewMirrorService(db)

	err := service.SyncMirror("nonexistent", "repo")
	if err != ErrMirrorNotFound {
		t.Errorf("Expected ErrMirrorNotFound, got %v", err)
	}
}

func TestMirrorService_GetMirrorsToSync(t *testing.T) {
	db := setupTestDB(t)
	service := NewMirrorService(db)

	// Create mirrors with different sync times
	now := time.Now()

	// Mirror 1: needs sync (past due)
	mirror1 := &model.RepositoryMirror{
		UserName:       "user1",
		RepositoryName: "repo1",
		MirrorURL:      "https://github.com/example/repo1.git",
		SyncInterval:   60,
		Enabled:        true,
		NextSyncDate:   now.Add(-1 * time.Hour), // Past due
	}
	service.db.Create(mirror1)

	// Mirror 2: doesn't need sync (future)
	mirror2 := &model.RepositoryMirror{
		UserName:       "user2",
		RepositoryName: "repo2",
		MirrorURL:      "https://github.com/example/repo2.git",
		SyncInterval:   60,
		Enabled:        true,
		NextSyncDate:   now.Add(1 * time.Hour), // Future
	}
	service.db.Create(mirror2)

	// Mirror 3: needs sync but disabled
	mirror3 := &model.RepositoryMirror{
		UserName:       "user3",
		RepositoryName: "repo3",
		MirrorURL:      "https://github.com/example/repo3.git",
		SyncInterval:   60,
		Enabled:        false,
		NextSyncDate:   now.Add(-1 * time.Hour), // Past due but disabled
	}
	service.db.Create(mirror3)

	// Get mirrors to sync
	mirrors, err := service.GetMirrorsToSync()
	if err != nil {
		t.Fatalf("GetMirrorsToSync failed: %v", err)
	}

	// Should only return mirror1 (enabled and past due)
	if len(mirrors) != 1 {
		t.Errorf("Expected 1 mirror to sync, got %d", len(mirrors))
	}
	if len(mirrors) > 0 && mirrors[0].UserName != "user1" {
		t.Errorf("Expected user1's mirror, got %s", mirrors[0].UserName)
	}
}
