package service

import (
	"testing"
)

func TestCollaboratorService_AddCollaborator(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCollaboratorService(db)

	// Add a collaborator
	collab, err := service.AddCollaborator("owner", "repo", "collaborator", "write")
	if err != nil {
		t.Fatalf("AddCollaborator failed: %v", err)
	}

	if collab.UserName != "owner" {
		t.Errorf("Expected UserName 'owner', got '%s'", collab.UserName)
	}
	if collab.RepositoryName != "repo" {
		t.Errorf("Expected RepositoryName 'repo', got '%s'", collab.RepositoryName)
	}
	if collab.CollaboratorName != "collaborator" {
		t.Errorf("Expected CollaboratorName 'collaborator', got '%s'", collab.CollaboratorName)
	}
	if collab.Role != "write" {
		t.Errorf("Expected Role 'write', got '%s'", collab.Role)
	}
}

func TestCollaboratorService_AddCollaborator_Duplicate(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCollaboratorService(db)

	// Add first collaborator
	_, err := service.AddCollaborator("owner", "repo", "collaborator", "write")
	if err != nil {
		t.Fatalf("First AddCollaborator failed: %v", err)
	}

	// Try to add duplicate
	_, err = service.AddCollaborator("owner", "repo", "collaborator", "read")
	if err == nil {
		t.Fatal("Expected error for duplicate collaborator, got nil")
	}
	if err != ErrCollaboratorExists {
		t.Errorf("Expected ErrCollaboratorExists, got %v", err)
	}
}

func TestCollaboratorService_IsCollaborator(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCollaboratorService(db)

	// Add a collaborator
	_, err := service.AddCollaborator("owner", "repo", "collaborator", "write")
	if err != nil {
		t.Fatalf("AddCollaborator failed: %v", err)
	}

	// Check if collaborator exists
	isCollab, err := service.IsCollaborator("owner", "repo", "collaborator")
	if err != nil {
		t.Fatalf("IsCollaborator failed: %v", err)
	}
	if !isCollab {
		t.Error("Expected user to be a collaborator")
	}

	// Check if non-collaborator exists
	isCollab, err = service.IsCollaborator("owner", "repo", "noncollaborator")
	if err != nil {
		t.Fatalf("IsCollaborator failed: %v", err)
	}
	if isCollab {
		t.Error("Expected user to not be a collaborator")
	}
}

func TestCollaboratorService_GetCollaboratorRole(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCollaboratorService(db)

	// Add a collaborator
	_, err := service.AddCollaborator("owner", "repo", "collaborator", "write")
	if err != nil {
		t.Fatalf("AddCollaborator failed: %v", err)
	}

	// Get collaborator role
	role, err := service.GetCollaboratorRole("owner", "repo", "collaborator")
	if err != nil {
		t.Fatalf("GetCollaboratorRole failed: %v", err)
	}
	if role != "write" {
		t.Errorf("Expected role 'write', got '%s'", role)
	}

	// Get role for non-collaborator
	role, err = service.GetCollaboratorRole("owner", "repo", "noncollaborator")
	if err != nil {
		t.Fatalf("GetCollaboratorRole failed: %v", err)
	}
	if role != "" {
		t.Errorf("Expected empty role for non-collaborator, got '%s'", role)
	}
}

func TestCollaboratorService_UpdateCollaboratorRole(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCollaboratorService(db)

	// Add a collaborator
	_, err := service.AddCollaborator("owner", "repo", "collaborator", "write")
	if err != nil {
		t.Fatalf("AddCollaborator failed: %v", err)
	}

	// Update role
	err = service.UpdateCollaboratorRole("owner", "repo", "collaborator", "admin")
	if err != nil {
		t.Fatalf("UpdateCollaboratorRole failed: %v", err)
	}

	// Verify update
	role, err := service.GetCollaboratorRole("owner", "repo", "collaborator")
	if err != nil {
		t.Fatalf("GetCollaboratorRole failed: %v", err)
	}
	if role != "admin" {
		t.Errorf("Expected role 'admin', got '%s'", role)
	}
}

func TestCollaboratorService_UpdateCollaboratorRole_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCollaboratorService(db)

	// Try to update non-existent collaborator
	err := service.UpdateCollaboratorRole("owner", "repo", "noncollaborator", "admin")
	if err == nil {
		t.Fatal("Expected error for non-existent collaborator, got nil")
	}
	if err != ErrCollaboratorNotFound {
		t.Errorf("Expected ErrCollaboratorNotFound, got %v", err)
	}
}

func TestCollaboratorService_RemoveCollaborator(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCollaboratorService(db)

	// Add a collaborator
	_, err := service.AddCollaborator("owner", "repo", "collaborator", "write")
	if err != nil {
		t.Fatalf("AddCollaborator failed: %v", err)
	}

	// Remove collaborator
	err = service.RemoveCollaborator("owner", "repo", "collaborator")
	if err != nil {
		t.Fatalf("RemoveCollaborator failed: %v", err)
	}

	// Verify removal
	isCollab, err := service.IsCollaborator("owner", "repo", "collaborator")
	if err != nil {
		t.Fatalf("IsCollaborator failed: %v", err)
	}
	if isCollab {
		t.Error("Expected collaborator to be removed")
	}
}

func TestCollaboratorService_RemoveCollaborator_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCollaboratorService(db)

	// Try to remove non-existent collaborator
	err := service.RemoveCollaborator("owner", "repo", "noncollaborator")
	if err == nil {
		t.Fatal("Expected error for non-existent collaborator, got nil")
	}
	if err != ErrCollaboratorNotFound {
		t.Errorf("Expected ErrCollaboratorNotFound, got %v", err)
	}
}

func TestCollaboratorService_ListCollaborators(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewCollaboratorService(db)

	// Add collaborators
	_, err := service.AddCollaborator("owner", "repo", "collab1", "write")
	if err != nil {
		t.Fatalf("AddCollaborator failed: %v", err)
	}
	_, err = service.AddCollaborator("owner", "repo", "collab2", "read")
	if err != nil {
		t.Fatalf("AddCollaborator failed: %v", err)
	}

	// List collaborators
	collabs, err := service.ListCollaborators("owner", "repo")
	if err != nil {
		t.Fatalf("ListCollaborators failed: %v", err)
	}

	if len(collabs) != 2 {
		t.Errorf("Expected 2 collaborators, got %d", len(collabs))
	}
}
