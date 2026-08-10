package service

import (
	"context"
	"fmt"
	"os"
	"strings"

	gitsvc "iforge/iforge/internal/git"
)

// ImportService handles repository import operations
type ImportService struct {
	gitClient *gitsvc.Client
}

// NewImportService creates a new ImportService
func NewImportService(gitClient *gitsvc.Client) *ImportService {
	return &ImportService{gitClient: gitClient}
}

// ImportRepository imports a repository from a URL
func (s *ImportService) ImportRepository(owner, repoName, sourceURL string) error {
	// Check if repository already exists
	repoPath := s.gitClient.RepositoryPath(owner, repoName)
	if _, err := os.Stat(repoPath); err == nil {
		return fmt.Errorf("repository already exists")
	}

	// Clone the repository as a bare repository
	ctx := context.Background()
	_, err := s.gitClient.CloneBare(ctx, owner, repoName, sourceURL)
	if err != nil {
		// Clean up on failure
		os.RemoveAll(repoPath)
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	return nil
}

// ParseRepositoryURL parses a repository URL and returns owner and repo name
func (s *ImportService) ParseRepositoryURL(url string) (owner, repo string, err error) {
	// Remove .git suffix if present
	url = strings.TrimSuffix(url, ".git")

	// Handle different URL formats
	// https://github.com/owner/repo
	// git@github.com:owner/repo
	// https://gitlab.com/owner/repo

	// Check for SSH format first (git@host:owner/repo)
	if strings.Contains(url, ":") && strings.Contains(url, "@") {
		parts := strings.Split(url, ":")
		if len(parts) == 2 {
			ownerRepo := strings.Split(parts[1], "/")
			if len(ownerRepo) >= 2 {
				return ownerRepo[len(ownerRepo)-2], ownerRepo[len(ownerRepo)-1], nil
			}
		}
		return "", "", fmt.Errorf("invalid SSH repository URL")
	}

	// Handle HTTPS format
	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid repository URL")
	}

	return parts[len(parts)-2], parts[len(parts)-1], nil
}
