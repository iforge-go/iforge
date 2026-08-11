package git

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// validRefNameRegex validates Git ref names (branches, tags).
// Allowed: letters, digits, -, _, /, .
var validRefNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-/\.]+$`)

// validCommitHashRegex validates commit hashes (7 or 40 hex characters).
var validCommitHashRegex = regexp.MustCompile(`^[a-fA-F0-9]+$`)

// IsValidRefName validates a Git ref name (branch, tag) according to official naming rules.
// See: https://git-scm.com/docs/git-check-ref-format
func IsValidRefName(name string) error {
	if name == "" {
		return fmt.Errorf("ref name cannot be empty")
	}

	// Must not start with - or .
	if strings.HasPrefix(name, "-") || strings.HasPrefix(name, ".") {
		return fmt.Errorf("ref name cannot start with - or .")
	}

	// Must not contain ..
	if strings.Contains(name, "..") {
		return fmt.Errorf("ref name cannot contain ..")
	}

	// Must not contain forbidden characters
	invalidChars := []string{"~", "^", ":", "?", "*", "[", "\\", " ", "@{", "\x00"}
	for _, char := range invalidChars {
		if strings.Contains(name, char) {
			return fmt.Errorf("ref name contains invalid character: %q", char)
		}
	}

	// Must match the regex
	if !validRefNameRegex.MatchString(name) {
		return fmt.Errorf("ref name contains invalid characters")
	}

	// Must not end with .lock
	if strings.HasSuffix(name, ".lock") {
		return fmt.Errorf("ref name cannot end with .lock")
	}

	// Must not start or end with /
	if strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/") {
		return fmt.Errorf("ref name cannot start or end with /")
	}

	// Must not contain consecutive slashes
	if strings.Contains(name, "//") {
		return fmt.Errorf("ref name cannot contain consecutive slashes")
	}

	return nil
}

// ValidateRepoPath validates a file path inside a repository to prevent path traversal attacks.
func ValidateRepoPath(path string) error {
	if path == "" {
		return nil // empty path means repository root
	}

	// Normalize path
	path = filepath.Clean(path)

	// Must not start with / or \ (checked before IsAbs for consistent error message across platforms)
	if strings.HasPrefix(path, "/") || strings.HasPrefix(path, "\\") {
		return fmt.Errorf("path cannot start with /")
	}

	// Must not be absolute (Unix filepath.IsAbs, or Windows drive path like C:\)
	if filepath.IsAbs(path) || isWindowsDrivePath(path) {
		return fmt.Errorf("path must be relative")
	}

	// Must not contain path traversal (after normalization, should not contain ..)
	if strings.Contains(path, "..") {
		return fmt.Errorf("path traversal not allowed")
	}

	// Must not contain Windows special characters
	if strings.ContainsAny(path, "<>:\"|?*\x00") {
		return fmt.Errorf("path contains invalid characters")
	}

	return nil
}

// isWindowsDrivePath checks for Windows drive-letter paths like C:\ or C:/ (works on all platforms).
func isWindowsDrivePath(path string) bool {
	return len(path) >= 3 && path[1] == ':' && (path[2] == '\\' || path[2] == '/') && isAlpha(path[0])
}

func isAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// IsValidCommitHash validates a commit hash format (7 or 40 hex characters).
func IsValidCommitHash(hash string) error {
	if len(hash) != 40 && len(hash) != 7 {
		return fmt.Errorf("invalid commit hash length: %d (expected 7 or 40)", len(hash))
	}

	if !validCommitHashRegex.MatchString(hash) {
		return fmt.Errorf("commit hash contains invalid characters")
	}

	return nil
}
