package service

import (
	"testing"
)

func TestReleaseService_CreateRelease(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewReleaseService(db)

	// Create a release
	release, err := service.CreateRelease("testuser", "test-repo", "v1.0.0", "First Release", "Release notes", "testuser")
	if err != nil {
		t.Fatalf("CreateRelease failed: %v", err)
	}

	if release.Tag != "v1.0.0" {
		t.Errorf("Expected Tag 'v1.0.0', got '%s'", release.Tag)
	}
	if release.Name != "First Release" {
		t.Errorf("Expected Name 'First Release', got '%s'", release.Name)
	}
	if release.Content == nil || *release.Content != "Release notes" {
		t.Errorf("Expected Content 'Release notes', got %v", release.Content)
	}
	if release.Author != "testuser" {
		t.Errorf("Expected Author 'testuser', got '%s'", release.Author)
	}
}

func TestReleaseService_GetRelease(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewReleaseService(db)

	// Create a release
	_, err := service.CreateRelease("testuser", "test-repo", "v1.0.0", "First Release", "Release notes", "testuser")
	if err != nil {
		t.Fatalf("CreateRelease failed: %v", err)
	}

	// Get the release
	release, err := service.GetRelease("testuser", "test-repo", "v1.0.0")
	if err != nil {
		t.Fatalf("GetRelease failed: %v", err)
	}

	if release.Tag != "v1.0.0" {
		t.Errorf("Expected Tag 'v1.0.0', got '%s'", release.Tag)
	}
	if release.Name != "First Release" {
		t.Errorf("Expected Name 'First Release', got '%s'", release.Name)
	}
}

func TestReleaseService_GetRelease_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewReleaseService(db)

	_, err := service.GetRelease("testuser", "test-repo", "v999.0.0")
	if err == nil {
		t.Fatal("Expected error for non-existent release, got nil")
	}
}

func TestReleaseService_ListReleases(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewReleaseService(db)

	// Create multiple releases
	for _, tag := range []string{"v1.0.0", "v1.1.0", "v2.0.0"} {
		_, err := service.CreateRelease("testuser", "test-repo", tag, tag, "Notes", "testuser")
		if err != nil {
			t.Fatalf("CreateRelease(%s) failed: %v", tag, err)
		}
	}

	// List releases
	releases, err := service.ListReleases("testuser", "test-repo")
	if err != nil {
		t.Fatalf("ListReleases failed: %v", err)
	}
	if len(releases) != 3 {
		t.Errorf("Expected 3 releases, got %d", len(releases))
	}
}

func TestReleaseService_UpdateRelease(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewReleaseService(db)

	// Create a release
	_, err := service.CreateRelease("testuser", "test-repo", "v1.0.0", "First Release", "Original notes", "testuser")
	if err != nil {
		t.Fatalf("CreateRelease failed: %v", err)
	}

	// Update the release
	if err := service.UpdateRelease("testuser", "test-repo", "v1.0.0", "Updated Release", "Updated notes"); err != nil {
		t.Fatalf("UpdateRelease failed: %v", err)
	}

	// Verify update
	updated, err := service.GetRelease("testuser", "test-repo", "v1.0.0")
	if err != nil {
		t.Fatalf("GetRelease failed: %v", err)
	}
	if updated.Name != "Updated Release" {
		t.Errorf("Expected Name 'Updated Release', got '%s'", updated.Name)
	}
	if updated.Content == nil || *updated.Content != "Updated notes" {
		t.Errorf("Expected Content 'Updated notes', got %v", updated.Content)
	}
}

func TestReleaseService_DeleteRelease(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewReleaseService(db)

	// Create a release
	_, err := service.CreateRelease("testuser", "test-repo", "v1.0.0", "First Release", "Notes", "testuser")
	if err != nil {
		t.Fatalf("CreateRelease failed: %v", err)
	}

	// Delete the release
	if err := service.DeleteRelease("testuser", "test-repo", "v1.0.0"); err != nil {
		t.Fatalf("DeleteRelease failed: %v", err)
	}

	// Verify deletion
	_, err = service.GetRelease("testuser", "test-repo", "v1.0.0")
	if err == nil {
		t.Fatal("Expected error after deletion, got nil")
	}
}

func TestReleaseService_CreateReleaseIfNotExists(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewReleaseService(db)

	// Create first release
	release1, err := service.CreateReleaseIfNotExists("testuser", "test-repo", "v1.0.0", "First", "Notes", "testuser")
	if err != nil {
		t.Fatalf("CreateReleaseIfNotExists (first) failed: %v", err)
	}

	// Try to create same release again (should return existing)
	release2, err := service.CreateReleaseIfNotExists("testuser", "test-repo", "v1.0.0", "Second", "Different notes", "testuser")
	if err != nil {
		t.Fatalf("CreateReleaseIfNotExists (second) failed: %v", err)
	}

	// Should return the first release
	if release2.Name != "First" {
		t.Errorf("Expected Name 'First' (existing), got '%s'", release2.Name)
	}
	if release2.Tag != release1.Tag {
		t.Errorf("Expected Tag '%s', got '%s'", release1.Tag, release2.Tag)
	}
}

func TestReleaseService_AddReleaseAsset(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewReleaseService(db)

	// Create a release first
	_, err := service.CreateRelease("testuser", "test-repo", "v1.0.0", "First Release", "Notes", "testuser")
	if err != nil {
		t.Fatalf("CreateRelease failed: %v", err)
	}

	// Add an asset
	label := "Binary"
	asset, err := service.AddReleaseAsset("testuser", "test-repo", "v1.0.0", "app.exe", &label, 1024, "testuser")
	if err != nil {
		t.Fatalf("AddReleaseAsset failed: %v", err)
	}

	if asset.FileName != "app.exe" {
		t.Errorf("Expected FileName 'app.exe', got '%s'", asset.FileName)
	}
	if asset.Size != 1024 {
		t.Errorf("Expected Size 1024, got %d", asset.Size)
	}
	if asset.Uploader != "testuser" {
		t.Errorf("Expected Uploader 'testuser', got '%s'", asset.Uploader)
	}
}

func TestReleaseService_GetAsset(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewReleaseService(db)

	// Create a release
	_, err := service.CreateRelease("testuser", "test-repo", "v1.0.0", "First Release", "Notes", "testuser")
	if err != nil {
		t.Fatalf("CreateRelease failed: %v", err)
	}

	// Add an asset
	created, err := service.AddReleaseAsset("testuser", "test-repo", "v1.0.0", "app.exe", nil, 1024, "testuser")
	if err != nil {
		t.Fatalf("AddReleaseAsset failed: %v", err)
	}

	// Get the asset
	asset, err := service.GetAsset("testuser", "test-repo", "v1.0.0", created.AssetID)
	if err != nil {
		t.Fatalf("GetAsset failed: %v", err)
	}

	if asset.AssetID != created.AssetID {
		t.Errorf("Expected AssetID %d, got %d", created.AssetID, asset.AssetID)
	}
	if asset.FileName != "app.exe" {
		t.Errorf("Expected FileName 'app.exe', got '%s'", asset.FileName)
	}
}

func TestReleaseService_GetAsset_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewReleaseService(db)

	_, err := service.GetAsset("testuser", "test-repo", "v1.0.0", 999)
	if err == nil {
		t.Fatal("Expected error for non-existent asset, got nil")
	}
}

func TestReleaseService_ListReleaseAssets(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewReleaseService(db)

	// Create a release
	_, err := service.CreateRelease("testuser", "test-repo", "v1.0.0", "First Release", "Notes", "testuser")
	if err != nil {
		t.Fatalf("CreateRelease failed: %v", err)
	}

	// Add multiple assets
	for _, name := range []string{"app.exe", "readme.txt", "config.json"} {
		_, err := service.AddReleaseAsset("testuser", "test-repo", "v1.0.0", name, nil, 100, "testuser")
		if err != nil {
			t.Fatalf("AddReleaseAsset(%s) failed: %v", name, err)
		}
	}

	// List assets
	assets, err := service.ListReleaseAssets("testuser", "test-repo", "v1.0.0")
	if err != nil {
		t.Fatalf("ListReleaseAssets failed: %v", err)
	}
	if len(assets) != 3 {
		t.Errorf("Expected 3 assets, got %d", len(assets))
	}
}

func TestReleaseService_DeleteReleaseAsset(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewReleaseService(db)

	// Create a release
	_, err := service.CreateRelease("testuser", "test-repo", "v1.0.0", "First Release", "Notes", "testuser")
	if err != nil {
		t.Fatalf("CreateRelease failed: %v", err)
	}

	// Add an asset
	asset, err := service.AddReleaseAsset("testuser", "test-repo", "v1.0.0", "app.exe", nil, 1024, "testuser")
	if err != nil {
		t.Fatalf("AddReleaseAsset failed: %v", err)
	}

	// Delete the asset
	if err := service.DeleteReleaseAsset("testuser", "test-repo", "v1.0.0", asset.AssetID); err != nil {
		t.Fatalf("DeleteReleaseAsset failed: %v", err)
	}

	// Verify deletion
	_, err = service.GetAsset("testuser", "test-repo", "v1.0.0", asset.AssetID)
	if err == nil {
		t.Fatal("Expected error after deletion, got nil")
	}
}
