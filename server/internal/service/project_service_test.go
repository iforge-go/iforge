package service

import (
	"errors"
	"testing"
	"time"

	"iforge/iforge/internal/model"
)

func TestCreateProject(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewProjectService(db)

	desc := "Test project description"
	project, err := service.CreateProject("testuser", "test-project", &desc, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	if project.OwnerName != "testuser" {
		t.Errorf("Expected owner 'testuser', got '%s'", project.OwnerName)
	}
	if project.Name != "test-project" {
		t.Errorf("Expected name 'test-project', got '%s'", project.Name)
	}
	if project.Description == nil || *project.Description != desc {
		t.Errorf("Expected description '%s', got %v", desc, project.Description)
	}
	if project.IsPrivate {
		t.Errorf("Expected IsPrivate false, got true")
	}
	if project.ID == 0 {
		t.Errorf("Expected project ID to be set, got 0")
	}
}

func TestCreateProject_Private(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewProjectService(db)

	project, err := service.CreateProject("testuser", "private-project", nil, true)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	if !project.IsPrivate {
		t.Errorf("Expected IsPrivate true, got false")
	}
	if project.Description != nil {
		t.Errorf("Expected nil description, got %v", project.Description)
	}
}

func TestGetProject(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewProjectService(db)

	// Create a project first
	project, err := service.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Get the project
	retrieved, err := service.GetProject(project.ID)
	if err != nil {
		t.Fatalf("GetProject failed: %v", err)
	}

	if retrieved.ID != project.ID {
		t.Errorf("Expected ID %d, got %d", project.ID, retrieved.ID)
	}
	if retrieved.OwnerName != project.OwnerName {
		t.Errorf("Expected owner '%s', got '%s'", project.OwnerName, retrieved.OwnerName)
	}
	if retrieved.Name != project.Name {
		t.Errorf("Expected name '%s', got '%s'", project.Name, retrieved.Name)
	}
}

func TestGetProject_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewProjectService(db)

	_, err := service.GetProject(999)
	if err == nil {
		t.Fatal("Expected error for non-existent project, got nil")
	}
	if !errors.Is(err, ErrProjectNotFound) {
		t.Errorf("Expected ErrProjectNotFound, got %v", err)
	}
}

func TestUpdateProject(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewProjectService(db)

	// Create a project
	project, err := service.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Update the project
	newDesc := "Updated description"
	project.Description = &newDesc
	project.Name = "updated-project"

	if err := service.UpdateProject(project); err != nil {
		t.Fatalf("UpdateProject failed: %v", err)
	}

	// Retrieve and verify
	updated, err := service.GetProject(project.ID)
	if err != nil {
		t.Fatalf("GetProject failed: %v", err)
	}

	if updated.Name != "updated-project" {
		t.Errorf("Expected name 'updated-project', got '%s'", updated.Name)
	}
	if updated.Description == nil || *updated.Description != newDesc {
		t.Errorf("Expected description '%s', got %v", newDesc, updated.Description)
	}
}

func TestListProjects(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewProjectService(db)

	// Create multiple projects for the same owner
	for i := 0; i < 3; i++ {
		_, err := service.CreateProject("testuser", "project-"+string(rune('a'+i)), nil, false)
		if err != nil {
			t.Fatalf("CreateProject failed: %v", err)
		}
	}

	// List projects
	projects, err := service.ListProjects("testuser", 0, 0)
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}

	if len(projects) != 3 {
		t.Errorf("Expected 3 projects, got %d", len(projects))
	}
}

func TestListProjects_WithLimit(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewProjectService(db)

	// Create multiple projects
	for i := 0; i < 5; i++ {
		_, err := service.CreateProject("testuser", "project-"+string(rune('a'+i)), nil, false)
		if err != nil {
			t.Fatalf("CreateProject failed: %v", err)
		}
	}

	// List with limit
	projects, err := service.ListProjects("testuser", 2, 0)
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}

	if len(projects) != 2 {
		t.Errorf("Expected 2 projects with limit, got %d", len(projects))
	}
}

func TestListProjects_AsMember(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewProjectService(db)

	// Create a project owned by user1
	project, err := service.CreateProject("user1", "user1-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Add user2 as a member
	if err := service.AddMember(project.ID, "user2", "member"); err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}

	// user2 should see the project in their list
	projects, err := service.ListProjects("user2", 0, 0)
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}

	if len(projects) != 1 {
		t.Errorf("Expected 1 project for user2, got %d", len(projects))
	}
	if projects[0].ID != project.ID {
		t.Errorf("Expected project ID %d, got %d", project.ID, projects[0].ID)
	}
}

func TestCountProjects(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewProjectService(db)

	// Create multiple projects
	for i := 0; i < 4; i++ {
		_, err := service.CreateProject("testuser", "project-"+string(rune('a'+i)), nil, false)
		if err != nil {
			t.Fatalf("CreateProject failed: %v", err)
		}
	}

	count, err := service.CountProjects()
	if err != nil {
		t.Fatalf("CountProjects failed: %v", err)
	}

	if count != 4 {
		t.Errorf("Expected count 4, got %d", count)
	}
}

func TestDeleteProject(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewProjectService(db)

	// Create a project
	project, err := service.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Add a member
	if err := service.AddMember(project.ID, "member1", "member"); err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}

	// Add a repository
	if err := service.AddRepository(project.ID, "testuser", "test-repo"); err != nil {
		t.Fatalf("AddRepository failed: %v", err)
	}

	// Delete the project
	if err := service.DeleteProject(project.ID); err != nil {
		t.Fatalf("DeleteProject failed: %v", err)
	}

	// Verify project is deleted
	_, err = service.GetProject(project.ID)
	if !errors.Is(err, ErrProjectNotFound) {
		t.Errorf("Expected ErrProjectNotFound after deletion, got %v", err)
	}

	// Verify members are deleted
	members, err := service.GetMembers(project.ID)
	if err != nil {
		t.Fatalf("GetMembers failed: %v", err)
	}
	if len(members) != 0 {
		t.Errorf("Expected 0 members after deletion, got %d", len(members))
	}

	// Verify repositories are deleted
	repos, err := service.GetRepositories(project.ID)
	if err != nil {
		t.Fatalf("GetRepositories failed: %v", err)
	}
	if len(repos) != 0 {
		t.Errorf("Expected 0 repositories after deletion, got %d", len(repos))
	}
}

func TestAddMember(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewProjectService(db)

	// Create a project
	project, err := service.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Add a member
	if err := service.AddMember(project.ID, "member1", "member"); err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}

	// Verify member is added
	members, err := service.GetMembers(project.ID)
	if err != nil {
		t.Fatalf("GetMembers failed: %v", err)
	}

	if len(members) != 1 {
		t.Errorf("Expected 1 member, got %d", len(members))
	}
	if members[0].UserName != "member1" {
		t.Errorf("Expected member 'member1', got '%s'", members[0].UserName)
	}
	if members[0].Role != "member" {
		t.Errorf("Expected role 'member', got '%s'", members[0].Role)
	}
}

func TestAddMember_InvalidRole(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewProjectService(db)

	// Create a project
	project, err := service.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Try to add member with invalid role
	err = service.AddMember(project.ID, "member1", "invalid_role")
	if err == nil {
		t.Fatal("Expected error for invalid role, got nil")
	}
	if !errors.Is(err, ErrInvalidRole) {
		t.Errorf("Expected ErrInvalidRole, got %v", err)
	}
}

func TestEnsureMember_AlreadyMember(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewProjectService(db)

	// Create a project
	project, err := service.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Add a member
	if err := service.AddMember(project.ID, "member1", "member"); err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}

	// EnsureMember should not change role if already a member
	if err := service.EnsureMember(project.ID, "member1", "admin"); err != nil {
		t.Fatalf("EnsureMember failed: %v", err)
	}

	// Verify role is still 'member' (not changed to 'admin')
	role := service.GetMemberRole(project.ID, "member1")
	if role != "member" {
		t.Errorf("Expected role 'member' (unchanged), got '%s'", role)
	}
}

func TestEnsureMember_NewMember(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewProjectService(db)

	// Create a project
	project, err := service.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// EnsureMember should add new member
	if err := service.EnsureMember(project.ID, "newmember", "member"); err != nil {
		t.Fatalf("EnsureMember failed: %v", err)
	}

	// Verify member is added
	role := service.GetMemberRole(project.ID, "newmember")
	if role != "member" {
		t.Errorf("Expected role 'member', got '%s'", role)
	}
}

func TestEnsureMember_EmptyUsername(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewProjectService(db)

	// Create a project
	project, err := service.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// EnsureMember with empty username should return nil
	if err := service.EnsureMember(project.ID, "", "member"); err != nil {
		t.Fatalf("EnsureMember with empty username should return nil, got %v", err)
	}
}

func TestRemoveMember(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewProjectService(db)

	// Create a project
	project, err := service.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Add a member
	if err := service.AddMember(project.ID, "member1", "member"); err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}

	// Remove the member
	if err := service.RemoveMember(project.ID, "member1"); err != nil {
		t.Fatalf("RemoveMember failed: %v", err)
	}

	// Verify member is removed
	members, err := service.GetMembers(project.ID)
	if err != nil {
		t.Fatalf("GetMembers failed: %v", err)
	}
	if len(members) != 0 {
		t.Errorf("Expected 0 members after removal, got %d", len(members))
	}
}

func TestUpdateMemberRole(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewProjectService(db)

	// Create a project
	project, err := service.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Add a member
	if err := service.AddMember(project.ID, "member1", "member"); err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}

	// Update member role to admin
	if err := service.UpdateMemberRole(project.ID, "member1", "admin"); err != nil {
		t.Fatalf("UpdateMemberRole failed: %v", err)
	}

	// Verify role is updated
	role := service.GetMemberRole(project.ID, "member1")
	if role != "admin" {
		t.Errorf("Expected role 'admin', got '%s'", role)
	}
}

func TestUpdateMemberRole_InvalidRole(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewProjectService(db)

	// Create a project
	project, err := service.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Add a member
	if err := service.AddMember(project.ID, "member1", "member"); err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}

	// Try to update to invalid role
	err = service.UpdateMemberRole(project.ID, "member1", "invalid_role")
	if !errors.Is(err, ErrInvalidRole) {
		t.Errorf("Expected ErrInvalidRole, got %v", err)
	}
}

func TestUpdateMemberRole_OwnerCannotBeModified(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewProjectService(db)

	// Create a project (testuser is the owner)
	project, err := service.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Try to update owner's role (should fail)
	err = service.UpdateMemberRole(project.ID, "testuser", "admin")
	if !errors.Is(err, ErrProjectForbidden) {
		t.Errorf("Expected ErrProjectForbidden when modifying owner, got %v", err)
	}
}

func TestUpdateMemberRole_MemberNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewProjectService(db)

	// Create a project
	project, err := service.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Try to update non-existent member
	err = service.UpdateMemberRole(project.ID, "nonexistent", "admin")
	if !errors.Is(err, ErrProjectMemberNotFound) {
		t.Errorf("Expected ErrProjectMemberNotFound, got %v", err)
	}
}

func TestGetMemberRole_Owner(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewProjectService(db)

	// Create a project
	project, err := service.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Owner should have role 'owner'
	role := service.GetMemberRole(project.ID, "testuser")
	if role != "owner" {
		t.Errorf("Expected role 'owner', got '%s'", role)
	}
}

func TestGetMemberRole_Member(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewProjectService(db)

	// Create a project
	project, err := service.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Add a member
	if err := service.AddMember(project.ID, "member1", "admin"); err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}

	// Member should have role 'admin'
	role := service.GetMemberRole(project.ID, "member1")
	if role != "admin" {
		t.Errorf("Expected role 'admin', got '%s'", role)
	}
}

func TestGetMemberRole_NonMember(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewProjectService(db)

	// Create a project
	project, err := service.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Non-member should have empty role
	role := service.GetMemberRole(project.ID, "nonmember")
	if role != "" {
		t.Errorf("Expected empty role for non-member, got '%s'", role)
	}
}

func TestGetMemberRole_EmptyUsername(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewProjectService(db)

	// Create a project
	project, err := service.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Empty username should return empty role
	role := service.GetMemberRole(project.ID, "")
	if role != "" {
		t.Errorf("Expected empty role for empty username, got '%s'", role)
	}
}

func TestRequireRole_SufficientRole(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewProjectService(db)

	// Create a project
	project, err := service.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Add a member with admin role
	if err := service.AddMember(project.ID, "member1", "admin"); err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}

	// Require member role (admin >= member)
	role, err := service.RequireRole(project.ID, "member1", "member")
	if err != nil {
		t.Fatalf("RequireRole failed: %v", err)
	}
	if role != "admin" {
		t.Errorf("Expected role 'admin', got '%s'", role)
	}
}

func TestRequireRole_InsufficientRole(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewProjectService(db)

	// Create a project
	project, err := service.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Add a member with viewer role
	if err := service.AddMember(project.ID, "member1", "viewer"); err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}

	// Require admin role (viewer < admin)
	_, err = service.RequireRole(project.ID, "member1", "admin")
	if !errors.Is(err, ErrProjectForbidden) {
		t.Errorf("Expected ErrProjectForbidden for insufficient role, got %v", err)
	}
}

func TestRequireRole_NonMember(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewProjectService(db)

	// Create a project
	project, err := service.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Require role for non-member
	_, err = service.RequireRole(project.ID, "nonmember", "member")
	if !errors.Is(err, ErrProjectForbidden) {
		t.Errorf("Expected ErrProjectForbidden for non-member, got %v", err)
	}
}

func TestAddRepository(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewProjectService(db)

	// Create a project
	project, err := service.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Add a repository
	if err := service.AddRepository(project.ID, "testuser", "test-repo"); err != nil {
		t.Fatalf("AddRepository failed: %v", err)
	}

	// Verify repository is added
	repos, err := service.GetRepositories(project.ID)
	if err != nil {
		t.Fatalf("GetRepositories failed: %v", err)
	}

	if len(repos) != 1 {
		t.Errorf("Expected 1 repository, got %d", len(repos))
	}
	if repos[0].UserName != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", repos[0].UserName)
	}
	if repos[0].RepositoryName != "test-repo" {
		t.Errorf("Expected repo name 'test-repo', got '%s'", repos[0].RepositoryName)
	}
}

func TestRemoveRepository(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewProjectService(db)

	// Create a project
	project, err := service.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Add a repository
	if err := service.AddRepository(project.ID, "testuser", "test-repo"); err != nil {
		t.Fatalf("AddRepository failed: %v", err)
	}

	// Remove the repository
	if err := service.RemoveRepository(project.ID, "testuser", "test-repo"); err != nil {
		t.Fatalf("RemoveRepository failed: %v", err)
	}

	// Verify repository is removed
	repos, err := service.GetRepositories(project.ID)
	if err != nil {
		t.Fatalf("GetRepositories failed: %v", err)
	}
	if len(repos) != 0 {
		t.Errorf("Expected 0 repositories after removal, got %d", len(repos))
	}
}

func TestFillProjectCounts(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewProjectService(db)

	// Create a project
	project, err := service.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Add members
	if err := service.AddMember(project.ID, "member1", "member"); err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}
	if err := service.AddMember(project.ID, "member2", "member"); err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}

	// Add sprints
	now := time.Now()
	sprint1 := &model.Sprint{
		ProjectID: project.ID,
		Title:     "Sprint 1",
		Status:    "active",
		CreatedBy: "testuser",
		CreatedAt: now,
		UpdatedAt: now,
	}
	sprint2 := &model.Sprint{
		ProjectID: project.ID,
		Title:     "Sprint 2",
		Status:    "open",
		CreatedBy: "testuser",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := db.Create(sprint1).Error; err != nil {
		t.Fatalf("Failed to create sprint: %v", err)
	}
	if err := db.Create(sprint2).Error; err != nil {
		t.Fatalf("Failed to create sprint: %v", err)
	}

	// Add user stories
	story1 := &model.UserStory{
		ProjectID:    project.ID,
		Title:        "Story 1",
		Status:       "open",
		Priority:     "medium",
		ReporterName: "testuser",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	story2 := &model.UserStory{
		ProjectID:    project.ID,
		Title:        "Story 2",
		Status:       "open",
		Priority:     "high",
		ReporterName: "testuser",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	story3 := &model.UserStory{
		ProjectID:    project.ID,
		Title:        "Story 3",
		Status:       "done",
		Priority:     "low",
		ReporterName: "testuser",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := db.Create(story1).Error; err != nil {
		t.Fatalf("Failed to create story: %v", err)
	}
	if err := db.Create(story2).Error; err != nil {
		t.Fatalf("Failed to create story: %v", err)
	}
	if err := db.Create(story3).Error; err != nil {
		t.Fatalf("Failed to create story: %v", err)
	}

	// Fill project counts
	projects := []*model.Project{project}
	if err := service.FillProjectCounts(projects); err != nil {
		t.Fatalf("FillProjectCounts failed: %v", err)
	}

	// Verify counts
	if project.MemberCount != 2 {
		t.Errorf("Expected MemberCount 2, got %d", project.MemberCount)
	}
	if project.SprintCount != 2 {
		t.Errorf("Expected SprintCount 2, got %d", project.SprintCount)
	}
	if project.UserStoryCount != 3 {
		t.Errorf("Expected UserStoryCount 3, got %d", project.UserStoryCount)
	}
}

func TestFillProjectCounts_EmptyList(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewProjectService(db)

	// FillProjectCounts with empty list should return nil
	if err := service.FillProjectCounts([]*model.Project{}); err != nil {
		t.Fatalf("FillProjectCounts with empty list should return nil, got %v", err)
	}
}

func TestGetMembers(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewProjectService(db)

	// Create a project
	project, err := service.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Add multiple members
	if err := service.AddMember(project.ID, "member1", "member"); err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}
	if err := service.AddMember(project.ID, "member2", "admin"); err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}

	// Get members
	members, err := service.GetMembers(project.ID)
	if err != nil {
		t.Fatalf("GetMembers failed: %v", err)
	}

	if len(members) != 2 {
		t.Errorf("Expected 2 members, got %d", len(members))
	}
}

func TestGetRepositories(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewProjectService(db)

	// Create a project
	project, err := service.CreateProject("testuser", "test-project", nil, false)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Add multiple repositories
	if err := service.AddRepository(project.ID, "testuser", "repo1"); err != nil {
		t.Fatalf("AddRepository failed: %v", err)
	}
	if err := service.AddRepository(project.ID, "testuser", "repo2"); err != nil {
		t.Fatalf("AddRepository failed: %v", err)
	}

	// Get repositories
	repos, err := service.GetRepositories(project.ID)
	if err != nil {
		t.Fatalf("GetRepositories failed: %v", err)
	}

	if len(repos) != 2 {
		t.Errorf("Expected 2 repositories, got %d", len(repos))
	}
}
