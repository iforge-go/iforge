package service

import (
	"testing"
)

func TestWikiService_CreateWikiPage(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewWikiService(db)

	// Create a wiki page
	page, err := service.CreateWikiPage("testuser", "test-repo", "home", "Home Page", "# Welcome", "testuser")
	if err != nil {
		t.Fatalf("CreateWikiPage failed: %v", err)
	}

	if page.UserName != "testuser" {
		t.Errorf("Expected UserName 'testuser', got '%s'", page.UserName)
	}
	if page.RepositoryName != "test-repo" {
		t.Errorf("Expected RepositoryName 'test-repo', got '%s'", page.RepositoryName)
	}
	if page.PageName != "home" {
		t.Errorf("Expected PageName 'home', got '%s'", page.PageName)
	}
	if page.Title != "Home Page" {
		t.Errorf("Expected Title 'Home Page', got '%s'", page.Title)
	}
	if page.Content != "# Welcome" {
		t.Errorf("Expected Content '# Welcome', got '%s'", page.Content)
	}
	if page.Author != "testuser" {
		t.Errorf("Expected Author 'testuser', got '%s'", page.Author)
	}
}

func TestWikiService_GetWikiPage(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewWikiService(db)

	// Create a wiki page
	_, err := service.CreateWikiPage("testuser", "test-repo", "home", "Home Page", "# Welcome", "testuser")
	if err != nil {
		t.Fatalf("CreateWikiPage failed: %v", err)
	}

	// Get the wiki page
	page, err := service.GetWikiPage("testuser", "test-repo", "home")
	if err != nil {
		t.Fatalf("GetWikiPage failed: %v", err)
	}
	if page == nil {
		t.Fatal("Expected page to be non-nil")
	}

	if page.PageName != "home" {
		t.Errorf("Expected PageName 'home', got '%s'", page.PageName)
	}
	if page.Title != "Home Page" {
		t.Errorf("Expected Title 'Home Page', got '%s'", page.Title)
	}
}

func TestWikiService_GetWikiPage_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewWikiService(db)

	// Get non-existent page
	page, err := service.GetWikiPage("testuser", "test-repo", "nonexistent")
	if err != nil {
		t.Fatalf("GetWikiPage failed: %v", err)
	}
	if page != nil {
		t.Errorf("Expected nil for non-existent page, got %v", page)
	}
}

func TestWikiService_ListWikiPages(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewWikiService(db)

	// Create multiple wiki pages
	pages := []struct {
		pageName string
		title    string
	}{
		{"home", "Home"},
		{"about", "About"},
		{"contact", "Contact"},
	}

	for _, p := range pages {
		_, err := service.CreateWikiPage("testuser", "test-repo", p.pageName, p.title, "Content", "testuser")
		if err != nil {
			t.Fatalf("CreateWikiPage(%s) failed: %v", p.pageName, err)
		}
	}

	// List wiki pages
	result, err := service.ListWikiPages("testuser", "test-repo")
	if err != nil {
		t.Fatalf("ListWikiPages failed: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("Expected 3 pages, got %d", len(result))
	}

	// Verify ordering (should be by page_name)
	if result[0].PageName != "about" {
		t.Errorf("Expected first page 'about', got '%s'", result[0].PageName)
	}
	if result[1].PageName != "contact" {
		t.Errorf("Expected second page 'contact', got '%s'", result[1].PageName)
	}
	if result[2].PageName != "home" {
		t.Errorf("Expected third page 'home', got '%s'", result[2].PageName)
	}
}

func TestWikiService_ListWikiPages_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewWikiService(db)

	// List pages for repo with no pages
	pages, err := service.ListWikiPages("testuser", "test-repo")
	if err != nil {
		t.Fatalf("ListWikiPages failed: %v", err)
	}
	if len(pages) != 0 {
		t.Errorf("Expected 0 pages, got %d", len(pages))
	}
}

func TestWikiService_UpdateWikiPage(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewWikiService(db)

	// Create a wiki page
	_, err := service.CreateWikiPage("testuser", "test-repo", "home", "Home Page", "# Welcome", "testuser")
	if err != nil {
		t.Fatalf("CreateWikiPage failed: %v", err)
	}

	// Update the wiki page
	updated, err := service.UpdateWikiPage("testuser", "test-repo", "home", "Updated Home", "# Updated Content", "editor")
	if err != nil {
		t.Fatalf("UpdateWikiPage failed: %v", err)
	}

	if updated.Title != "Updated Home" {
		t.Errorf("Expected Title 'Updated Home', got '%s'", updated.Title)
	}
	if updated.Content != "# Updated Content" {
		t.Errorf("Expected Content '# Updated Content', got '%s'", updated.Content)
	}
	if updated.Author != "editor" {
		t.Errorf("Expected Author 'editor', got '%s'", updated.Author)
	}

	// Verify the update persisted
	page, err := service.GetWikiPage("testuser", "test-repo", "home")
	if err != nil {
		t.Fatalf("GetWikiPage failed: %v", err)
	}
	if page.Title != "Updated Home" {
		t.Errorf("Expected persisted Title 'Updated Home', got '%s'", page.Title)
	}
}

func TestWikiService_DeleteWikiPage(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewWikiService(db)

	// Create a wiki page
	_, err := service.CreateWikiPage("testuser", "test-repo", "home", "Home Page", "# Welcome", "testuser")
	if err != nil {
		t.Fatalf("CreateWikiPage failed: %v", err)
	}

	// Delete the wiki page
	if err := service.DeleteWikiPage("testuser", "test-repo", "home"); err != nil {
		t.Fatalf("DeleteWikiPage failed: %v", err)
	}

	// Verify deletion
	page, err := service.GetWikiPage("testuser", "test-repo", "home")
	if err != nil {
		t.Fatalf("GetWikiPage failed: %v", err)
	}
	if page != nil {
		t.Error("Expected nil after deletion")
	}
}

func TestWikiService_DeleteWikiPage_NonExistent(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewWikiService(db)

	// Delete non-existent page (should not error)
	if err := service.DeleteWikiPage("testuser", "test-repo", "nonexistent"); err != nil {
		t.Fatalf("DeleteWikiPage for non-existent page failed: %v", err)
	}
}

func TestWikiService_MultipleRepos(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewWikiService(db)

	// Create pages in different repos
	_, err := service.CreateWikiPage("user1", "repo1", "home", "Repo1 Home", "Content", "user1")
	if err != nil {
		t.Fatalf("CreateWikiPage for repo1 failed: %v", err)
	}

	_, err = service.CreateWikiPage("user2", "repo2", "home", "Repo2 Home", "Content", "user2")
	if err != nil {
		t.Fatalf("CreateWikiPage for repo2 failed: %v", err)
	}

	// List pages for repo1
	pages1, err := service.ListWikiPages("user1", "repo1")
	if err != nil {
		t.Fatalf("ListWikiPages for repo1 failed: %v", err)
	}
	if len(pages1) != 1 {
		t.Errorf("Expected 1 page for repo1, got %d", len(pages1))
	}

	// List pages for repo2
	pages2, err := service.ListWikiPages("user2", "repo2")
	if err != nil {
		t.Fatalf("ListWikiPages for repo2 failed: %v", err)
	}
	if len(pages2) != 1 {
		t.Errorf("Expected 1 page for repo2, got %d", len(pages2))
	}
}
