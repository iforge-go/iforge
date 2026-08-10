package git

import (
	"path/filepath"
	"testing"
)

func TestRepositoryPath(t *testing.T) {
	svc := NewClient("/tmp/repos")

	tests := []struct {
		name     string
		owner    string
		repo     string
		expected string
	}{
		{
			name:     "Simple path",
			owner:    "user",
			repo:     "repo",
			expected: filepath.FromSlash("/tmp/repos/user/repo.git"),
		},
		{
			name:     "Path with special characters",
			owner:    "user-name",
			repo:     "repo_name",
			expected: filepath.FromSlash("/tmp/repos/user-name/repo_name.git"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.RepositoryPath(tt.owner, tt.repo)
			if result != tt.expected {
				t.Errorf("RepositoryPath() = %q, want %q", result, tt.expected)
			}
		})
	}
}
