package git

import (
	"strings"
	"testing"
)

func TestIsValidRefName(t *testing.T) {
	tests := []struct {
		name    string
		refName string
		wantErr bool
		errMsg  string
	}{
		// Valid cases
		{"valid simple branch", "main", false, ""},
		{"valid feature branch", "feature/new-feature", false, ""},
		{"valid branch with numbers", "release/v1.2.3", false, ""},
		{"valid branch with underscore", "bug_fix_123", false, ""},
		{"valid branch with dash", "hotfix-critical-bug", false, ""},
		{"valid nested branch", "feature/user/auth/login", false, ""},

		// Invalid cases - empty value
		{"empty string", "", true, "ref name cannot be empty"},

		// Invalid cases - prefix
		{"starts with dash", "-invalid", true, "ref name cannot start with - or ."},
		{"starts with dot", ".invalid", true, "ref name cannot start with - or ."},

		// Invalid cases - special characters
		{"contains tilde", "branch~1", true, "ref name contains invalid character"},
		{"contains caret", "branch^1", true, "ref name contains invalid character"},
		{"contains colon", "branch:name", true, "ref name contains invalid character"},
		{"contains question", "branch?name", true, "ref name contains invalid character"},
		{"contains asterisk", "branch*name", true, "ref name contains invalid character"},
		{"contains bracket", "branch[name]", true, "ref name contains invalid character"},
		{"contains backslash", "branch\\name", true, "ref name contains invalid character"},
		{"contains space", "branch name", true, "ref name contains invalid character"},
		{"contains @{", "branch@{name}", true, "ref name contains invalid character"},
		{"contains null", "branch\x00name", true, "ref name contains invalid character"},

		// Invalid cases - suffix
		{"ends with .lock", "branch.lock", true, "ref name cannot end with .lock"},

		// Invalid cases - slashes
		{"starts with slash", "/branch", true, "ref name cannot start or end with /"},
		{"ends with slash", "branch/", true, "ref name cannot start or end with /"},
		{"consecutive slashes", "branch//name", true, "ref name cannot contain consecutive slashes"},

		// Invalid cases - others
		{"contains ..", "branch..name", true, "ref name cannot contain .."},
		{"contains invalid regex chars", "branch$name", true, "ref name contains invalid characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IsValidRefName(tt.refName)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsValidRefName(%q) error = %v, wantErr %v", tt.refName, err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("IsValidRefName(%q) error = %v, want error containing %v", tt.refName, err.Error(), tt.errMsg)
			}
		})
	}
}

func TestValidateRepoPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		// Valid cases
		{"empty path (root)", "", false, ""},
		{"simple file", "README.md", false, ""},
		{"nested file", "src/main.go", false, ""},
		{"deeply nested file", "src/pkg/utils/helper.go", false, ""},
		{"file with spaces", "my file.txt", false, ""},
		{"file with underscore", "my_file.txt", false, ""},
		{"file with dash", "my-file.txt", false, ""},

		// Invalid cases - path traversal attack
		{"parent directory", "../etc/passwd", true, "path traversal not allowed"},
		{"nested parent", "src/../../../etc/passwd", true, "path traversal not allowed"},
		// filepath.Clean("src/../main.go") = "main.go", safe after normalization
		{"parent in middle", "src/../../etc/passwd", true, "path traversal not allowed"},

		// Invalid cases - absolute path (starts with /, blocked cross-platform)
		{"absolute unix path", "/etc/passwd", true, "path cannot start with /"},
		{"absolute windows path", "C:\\Windows\\System32", true, "path must be relative"},

		// Invalid cases - special characters
		{"contains null byte", "file\x00name.txt", true, "path contains invalid characters"},
		{"contains pipe", "file|name.txt", true, "path contains invalid characters"},
		{"contains question", "file?name.txt", true, "path contains invalid characters"},
		{"contains asterisk", "file*name.txt", true, "path contains invalid characters"},
		{"contains quote", "file\"name.txt", true, "path contains invalid characters"},
		{"contains angle bracket", "file<name>.txt", true, "path contains invalid characters"},

		// Invalid cases - prefix
		{"starts with slash", "/etc/passwd", true, "path cannot start with /"},
		{"starts with backslash", "\\Windows\\System32", true, "path cannot start with /"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRepoPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRepoPath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidateRepoPath(%q) error = %v, want error containing %v", tt.path, err.Error(), tt.errMsg)
			}
		})
	}
}

func TestIsValidCommitHash(t *testing.T) {
	tests := []struct {
		name    string
		hash    string
		wantErr bool
		errMsg  string
	}{
		// Valid cases
		{"valid 7-char hash", "abc1234", false, ""},
		{"valid 40-char hash", "abc1234567890abcdef1234567890abcdef12345", false, ""},
		{"valid uppercase", "ABC1234", false, ""},
		{"valid mixed case", "AbC1234", false, ""},
		{"all zeros", "0000000", false, ""},
		{"all f's", "fffffff", false, ""},

		// Invalid cases - length
		{"too short", "abc123", true, "invalid commit hash length"},
		{"too long", "abc12345", true, "invalid commit hash length"},
		{"empty string", "", true, "invalid commit hash length"},
		{"41 chars", "abc1234567890abcdef1234567890abcdef1234567", true, "invalid commit hash length"},

		// Invalid cases - characters
		{"contains g", "abc123g", true, "commit hash contains invalid characters"},
		{"contains space", "abc 123", true, "commit hash contains invalid characters"},
		{"contains special char", "abc123!", true, "commit hash contains invalid characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IsValidCommitHash(tt.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsValidCommitHash(%q) error = %v, wantErr %v", tt.hash, err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("IsValidCommitHash(%q) error = %v, want error containing %v", tt.hash, err.Error(), tt.errMsg)
			}
		})
	}
}

// BenchmarkIsValidRefName performance test
func BenchmarkIsValidRefName(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = IsValidRefName("feature/new-feature")
	}
}

// BenchmarkValidateRepoPath performance test
func BenchmarkValidateRepoPath(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = ValidateRepoPath("src/main.go")
	}
}

// BenchmarkIsValidCommitHash performance test
func BenchmarkIsValidCommitHash(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = IsValidCommitHash("abc1234567890abcdef1234567890abcdef123456")
	}
}
