package service

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"iforge/iforge/internal/git"
)

func TestLFSService_SaveMeta(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	// Create temp dir as repo path
	tempDir := t.TempDir()
	gitClient := git.NewClient(tempDir)
	service := NewLFSService(db, gitClient)

	owner := "testuser"
	repo := "testrepo"
	oid := "abc123def456"
	size := int64(1024)

	// Save metadata
	err := service.SaveMeta(owner, repo, oid, size)
	if err != nil {
		t.Fatalf("SaveMeta failed: %v", err)
	}

	// Verify metadata saved
	obj, err := service.GetMeta(owner, repo, oid)
	if err != nil {
		t.Fatalf("GetMeta failed: %v", err)
	}

	if obj.OID != oid {
		t.Errorf("Expected OID '%s', got '%s'", oid, obj.OID)
	}
	if obj.Size != size {
		t.Errorf("Expected Size %d, got %d", size, obj.Size)
	}
	if obj.UserName != owner {
		t.Errorf("Expected UserName '%s', got '%s'", owner, obj.UserName)
	}
	if obj.RepositoryName != repo {
		t.Errorf("Expected RepositoryName '%s', got '%s'", repo, obj.RepositoryName)
	}
}

func TestLFSService_SaveMeta_Duplicate(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	tempDir := t.TempDir()
	gitClient := git.NewClient(tempDir)
	service := NewLFSService(db, gitClient)

	owner := "testuser"
	repo := "testrepo"
	oid := "abc123def456"
	size := int64(1024)

	// Save metadata twice
	service.SaveMeta(owner, repo, oid, size)
	err := service.SaveMeta(owner, repo, oid, size)
	if err != nil {
		t.Errorf("SaveMeta should be idempotent, got error: %v", err)
	}

	// Verify only one record
	if !service.MetaExists(owner, repo, oid) {
		t.Error("Expected metadata to exist")
	}
}

func TestLFSService_GetMeta_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	tempDir := t.TempDir()
	gitClient := git.NewClient(tempDir)
	service := NewLFSService(db, gitClient)

	// Get non-existent metadata
	_, err := service.GetMeta("testuser", "testrepo", "nonexistent")
	if err != ErrLFSObjectNotFound {
		t.Errorf("Expected ErrLFSObjectNotFound, got %v", err)
	}
}

func TestLFSService_MetaExists(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	tempDir := t.TempDir()
	gitClient := git.NewClient(tempDir)
	service := NewLFSService(db, gitClient)

	owner := "testuser"
	repo := "testrepo"
	oid := "abc123def456"

	// Initially not exists
	if service.MetaExists(owner, repo, oid) {
		t.Error("Expected MetaExists to return false initially")
	}

	// Exists after save
	service.SaveMeta(owner, repo, oid, 1024)
	if !service.MetaExists(owner, repo, oid) {
		t.Error("Expected MetaExists to return true after SaveMeta")
	}
}

func TestLFSService_ListObjects(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	tempDir := t.TempDir()
	gitClient := git.NewClient(tempDir)
	service := NewLFSService(db, gitClient)

	owner := "testuser"
	repo := "testrepo"

	// Create multiple LFS objects
	for i := 0; i < 5; i++ {
		oid := strings.Repeat(string(rune('a'+i)), 12)
		service.SaveMeta(owner, repo, oid, int64(1024*(i+1)))
	}

	// List all objects
	objects, total, err := service.ListObjects(owner, repo, 1, 10)
	if err != nil {
		t.Fatalf("ListObjects failed: %v", err)
	}

	if total != 5 {
		t.Errorf("Expected total 5, got %d", total)
	}
	if len(objects) != 5 {
		t.Errorf("Expected 5 objects, got %d", len(objects))
	}
}

func TestLFSService_ListObjects_Pagination(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	tempDir := t.TempDir()
	gitClient := git.NewClient(tempDir)
	service := NewLFSService(db, gitClient)

	owner := "testuser"
	repo := "testrepo"

	// Create 10 objects
	for i := 0; i < 10; i++ {
		oid := strings.Repeat(string(rune('a'+i)), 12)
		service.SaveMeta(owner, repo, oid, int64(1024))
	}

	// First page
	objects1, total1, err := service.ListObjects(owner, repo, 1, 3)
	if err != nil {
		t.Fatalf("ListObjects page 1 failed: %v", err)
	}
	if total1 != 10 {
		t.Errorf("Expected total 10, got %d", total1)
	}
	if len(objects1) != 3 {
		t.Errorf("Expected 3 objects on page 1, got %d", len(objects1))
	}

	// Second page
	objects2, _, err := service.ListObjects(owner, repo, 2, 3)
	if err != nil {
		t.Fatalf("ListObjects page 2 failed: %v", err)
	}
	if len(objects2) != 3 {
		t.Errorf("Expected 3 objects on page 2, got %d", len(objects2))
	}

	// Verify pagination no overlap
	if objects1[0].OID == objects2[0].OID {
		t.Error("Pagination should not return overlapping results")
	}
}

func TestLFSService_ListObjects_InvalidParams(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	tempDir := t.TempDir()
	gitClient := git.NewClient(tempDir)
	service := NewLFSService(db, gitClient)

	owner := "testuser"
	repo := "testrepo"

	// Create test data
	service.SaveMeta(owner, repo, "abc123def456", 1024)

	// Test invalid page param (default to 1)
	objects, _, err := service.ListObjects(owner, repo, 0, 10)
	if err != nil {
		t.Fatalf("ListObjects with page=0 failed: %v", err)
	}
	if len(objects) != 1 {
		t.Errorf("Expected 1 object with page=0, got %d", len(objects))
	}

	// Test invalid limit param (default to 50)
	objects, _, err = service.ListObjects(owner, repo, 1, 0)
	if err != nil {
		t.Fatalf("ListObjects with limit=0 failed: %v", err)
	}
	if len(objects) != 1 {
		t.Errorf("Expected 1 object with limit=0, got %d", len(objects))
	}

	// Test limit > 200 (default to 50)
	objects, _, err = service.ListObjects(owner, repo, 1, 300)
	if err != nil {
		t.Fatalf("ListObjects with limit=300 failed: %v", err)
	}
	if len(objects) != 1 {
		t.Errorf("Expected 1 object with limit=300, got %d", len(objects))
	}
}

func TestLFSService_DeleteObject(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	tempDir := t.TempDir()
	gitClient := git.NewClient(tempDir)
	service := NewLFSService(db, gitClient)

	owner := "testuser"
	repo := "testrepo"
	oid := "abc123def456"

	// Save metadata and write file
	service.SaveMeta(owner, repo, oid, 1024)
	content := strings.NewReader("test content")
	service.WriteObjectContent(owner, repo, oid, content)

	// Delete object
	err := service.DeleteObject(owner, repo, oid)
	if err != nil {
		t.Fatalf("DeleteObject failed: %v", err)
	}

	// Verify metadata deleted
	if service.MetaExists(owner, repo, oid) {
		t.Error("Expected metadata to be deleted")
	}

	// Verify file deleted
	objPath := service.ObjectPath(owner, repo, oid)
	if _, err := os.Stat(objPath); !os.IsNotExist(err) {
		t.Error("Expected object file to be deleted")
	}
}

func TestLFSService_DeleteObject_NonExistent(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	tempDir := t.TempDir()
	gitClient := git.NewClient(tempDir)
	service := NewLFSService(db, gitClient)

	// Delete non-existent object (os.Remove ignores not-exist)
	err := service.DeleteObject("testuser", "testrepo", "nonexistent")
	if err != nil {
		t.Errorf("DeleteObject should succeed for non-existent object, got: %v", err)
	}
}

func TestLFSService_WriteObjectContent(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	tempDir := t.TempDir()
	gitClient := git.NewClient(tempDir)
	service := NewLFSService(db, gitClient)

	owner := "testuser"
	repo := "testrepo"
	oid := "abc123def456"
	content := "test content for LFS object"

	// Write object content
	reader := strings.NewReader(content)
	err := service.WriteObjectContent(owner, repo, oid, reader)
	if err != nil {
		t.Fatalf("WriteObjectContent failed: %v", err)
	}

	// Verify file created
	objPath := service.ObjectPath(owner, repo, oid)
	if _, err := os.Stat(objPath); os.IsNotExist(err) {
		t.Error("Expected object file to be created")
	}

	// Read and verify content
	data, err := os.ReadFile(objPath)
	if err != nil {
		t.Fatalf("Failed to read object file: %v", err)
	}
	if string(data) != content {
		t.Errorf("Expected content '%s', got '%s'", content, string(data))
	}
}

func TestLFSService_ReadObjectContent(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	tempDir := t.TempDir()
	gitClient := git.NewClient(tempDir)
	service := NewLFSService(db, gitClient)

	owner := "testuser"
	repo := "testrepo"
	oid := "abc123def456"
	content := "test content for LFS object"

	// Write content first
	reader := strings.NewReader(content)
	service.WriteObjectContent(owner, repo, oid, reader)

	// Read content
	readCloser, size, err := service.ReadObjectContent(owner, repo, oid)
	if err != nil {
		t.Fatalf("ReadObjectContent failed: %v", err)
	}
	defer readCloser.Close()

	// Verify size
	expectedSize := int64(len(content))
	if size != expectedSize {
		t.Errorf("Expected size %d, got %d", expectedSize, size)
	}

	// Read and verify content
	data, err := io.ReadAll(readCloser)
	if err != nil {
		t.Fatalf("Failed to read content: %v", err)
	}
	if string(data) != content {
		t.Errorf("Expected content '%s', got '%s'", content, string(data))
	}
}

func TestLFSService_ReadObjectContent_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	tempDir := t.TempDir()
	gitClient := git.NewClient(tempDir)
	service := NewLFSService(db, gitClient)

	// Read non-existent object
	_, _, err := service.ReadObjectContent("testuser", "testrepo", "nonexistent")
	if err != ErrLFSObjectNotFound {
		t.Errorf("Expected ErrLFSObjectNotFound, got %v", err)
	}
}

func TestLFSService_ObjectPath(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	tempDir := t.TempDir()
	gitClient := git.NewClient(tempDir)
	service := NewLFSService(db, gitClient)

	owner := "testuser"
	repo := "testrepo"

	tests := []struct {
		name     string
		oid      string
		contains string
	}{
		{
			name:     "normal OID with sharding",
			oid:      "abc123def456",
			contains: filepath.Join("ab", "c1", "abc123def456"),
		},
		{
			name:     "short OID without sharding",
			oid:      "abc",
			contains: "abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := service.ObjectPath(owner, repo, tt.oid)
			if !strings.Contains(path, tt.contains) {
				t.Errorf("Expected path to contain '%s', got '%s'", tt.contains, path)
			}
		})
	}
}

func TestLFSService_Integration(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	tempDir := t.TempDir()
	gitClient := git.NewClient(tempDir)
	service := NewLFSService(db, gitClient)

	owner := "testuser"
	repo := "testrepo"
	oid := "abc123def456"
	content := "integration test content"

	// 1. Save metadata
	err := service.SaveMeta(owner, repo, oid, int64(len(content)))
	if err != nil {
		t.Fatalf("SaveMeta failed: %v", err)
	}

	// 2. Write object content
	reader := strings.NewReader(content)
	err = service.WriteObjectContent(owner, repo, oid, reader)
	if err != nil {
		t.Fatalf("WriteObjectContent failed: %v", err)
	}

	// 3. Verify metadata exists
	if !service.MetaExists(owner, repo, oid) {
		t.Error("Expected metadata to exist")
	}

	// 4. Read object content
	readCloser, size, err := service.ReadObjectContent(owner, repo, oid)
	if err != nil {
		t.Fatalf("ReadObjectContent failed: %v", err)
	}

	data, err := io.ReadAll(readCloser)
	readCloser.Close() // Close handle to avoid Windows file-in-use error
	if err != nil {
		t.Fatalf("Failed to read content: %v", err)
	}
	if string(data) != content {
		t.Errorf("Expected content '%s', got '%s'", content, string(data))
	}
	if size != int64(len(content)) {
		t.Errorf("Expected size %d, got %d", len(content), size)
	}

	// 5. List objects
	objects, total, err := service.ListObjects(owner, repo, 1, 10)
	if err != nil {
		t.Fatalf("ListObjects failed: %v", err)
	}
	if total != 1 {
		t.Errorf("Expected total 1, got %d", total)
	}
	if len(objects) != 1 {
		t.Errorf("Expected 1 object, got %d", len(objects))
	}

	// 6. Delete object
	err = service.DeleteObject(owner, repo, oid)
	if err != nil {
		t.Fatalf("DeleteObject failed: %v", err)
	}

	// 7. Verify deleted
	if service.MetaExists(owner, repo, oid) {
		t.Error("Expected metadata to be deleted")
	}
	_, _, err = service.ReadObjectContent(owner, repo, oid)
	if err != ErrLFSObjectNotFound {
		t.Error("Expected ErrLFSObjectNotFound after deletion")
	}
}
