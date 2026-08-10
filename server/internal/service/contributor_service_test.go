package service

import (
	"testing"
	"time"
)

func TestContributorStats_Fields(t *testing.T) {
	now := time.Now()
	stats := ContributorStats{
		Name:         "John Doe",
		Email:        "john@example.com",
		Commits:      10,
		Additions:    100,
		Deletions:    50,
		FirstCommit:  now.Add(-24 * time.Hour),
		LastCommit:   now,
		FilesChanged: 5,
	}

	if stats.Name != "John Doe" {
		t.Errorf("Expected Name 'John Doe', got %q", stats.Name)
	}
	if stats.Email != "john@example.com" {
		t.Errorf("Expected Email 'john@example.com', got %q", stats.Email)
	}
	if stats.Commits != 10 {
		t.Errorf("Expected Commits 10, got %d", stats.Commits)
	}
	if stats.Additions != 100 {
		t.Errorf("Expected Additions 100, got %d", stats.Additions)
	}
	if stats.Deletions != 50 {
		t.Errorf("Expected Deletions 50, got %d", stats.Deletions)
	}
	if stats.FilesChanged != 5 {
		t.Errorf("Expected FilesChanged 5, got %d", stats.FilesChanged)
	}
}

func TestNewContributorService(t *testing.T) {
	service := NewContributorService(nil)
	if service == nil {
		t.Fatal("Expected ContributorService to be created")
	}
	if service.gitClient != nil {
		t.Error("Expected gitClient to be nil")
	}
}
