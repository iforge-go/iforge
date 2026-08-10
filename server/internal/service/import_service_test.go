package service

import (
	"testing"
)

func TestParseRepositoryURL(t *testing.T) {
	// ParseRepositoryURL does not depend on gitClient, nil can be passed
	service := NewImportService(nil)

	tests := []struct {
		name          string
		url           string
		expectedOwner string
		expectedRepo  string
		expectError   bool
	}{
		{
			name:          "GitHub HTTPS URL",
			url:           "https://github.com/octocat/Hello-World",
			expectedOwner: "octocat",
			expectedRepo:  "Hello-World",
			expectError:   false,
		},
		{
			name:          "GitHub HTTPS URL with .git",
			url:           "https://github.com/octocat/Hello-World.git",
			expectedOwner: "octocat",
			expectedRepo:  "Hello-World",
			expectError:   false,
		},
		{
			name:          "GitLab HTTPS URL",
			url:           "https://gitlab.com/user/project",
			expectedOwner: "user",
			expectedRepo:  "project",
			expectError:   false,
		},
		{
			name:          "Bitbucket HTTPS URL",
			url:           "https://bitbucket.org/team/repo",
			expectedOwner: "team",
			expectedRepo:  "repo",
			expectError:   false,
		},
		{
			name:          "Git SSH URL",
			url:           "git@github.com:octocat/Hello-World.git",
			expectedOwner: "octocat",
			expectedRepo:  "Hello-World",
			expectError:   false,
		},
		{
			name:          "Git SSH URL without .git",
			url:           "git@github.com:octocat/Hello-World",
			expectedOwner: "octocat",
			expectedRepo:  "Hello-World",
			expectError:   false,
		},
		{
			name:        "Invalid URL - single segment",
			url:         "invalid",
			expectError: true,
		},
		{
			name:        "Invalid URL - empty",
			url:         "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := service.ParseRepositoryURL(tt.url)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if owner != tt.expectedOwner {
				t.Errorf("Owner mismatch: got %q, want %q", owner, tt.expectedOwner)
			}

			if repo != tt.expectedRepo {
				t.Errorf("Repo mismatch: got %q, want %q", repo, tt.expectedRepo)
			}
		})
	}
}
