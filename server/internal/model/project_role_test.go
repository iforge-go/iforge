package model

import "testing"

func TestProjectRoleLevel(t *testing.T) {
	tests := []struct {
		role     string
		expected int
	}{
		// Normal cases
		{"owner", 4},
		{"admin", 3},
		{"member", 2},
		{"viewer", 1},
		// Case insensitive
		{"OWNER", 4},
		{"Owner", 4},
		{"ADMIN", 3},
		{"Admin", 3},
		{"MEMBER", 2},
		{"Member", 2},
		{"VIEWER", 1},
		{"Viewer", 1},
		// Invalid roles
		{"unknown", 0},
		{"", 0},
		// Legacy values not recognized
		{"write", 0},
		{"read", 0},
		{"developer", 0},
		{"guest", 0},
		// Semantic difference from role.go: member/viewer/admin have different levels here (owner=4), ensure isolation
	}
	for _, tt := range tests {
		got := ProjectRoleLevel(tt.role)
		if got != tt.expected {
			t.Errorf("ProjectRoleLevel(%q) = %d, want %d", tt.role, got, tt.expected)
		}
	}
}

func TestValidProjectRole(t *testing.T) {
	tests := []struct {
		role     string
		expected bool
	}{
		// Valid four-level roles
		{"owner", true},
		{"admin", true},
		{"member", true},
		{"viewer", true},
		// Case insensitive
		{"OWNER", true},
		{"Admin", true},
		{"MEMBER", true},
		{"Viewer", true},
		// Invalid roles
		{"", false},
		{"unknown", false},
		{"write", false},
		{"read", false},
		{"developer", false},
		{"guest", false},
		{" root ", false},  // No trim
	}
	for _, tt := range tests {
		got := ValidProjectRole(tt.role)
		if got != tt.expected {
			t.Errorf("ValidProjectRole(%q) = %v, want %v", tt.role, got, tt.expected)
		}
	}
}

func TestHasProjectRoleAtLeast(t *testing.T) {
	tests := []struct {
		role     string
		required string
		expected bool
	}{
		// Level satisfied (owner is highest)
		{"owner", "owner", true},
		{"owner", "admin", true},
		{"owner", "member", true},
		{"owner", "viewer", true},
		{"admin", "admin", true},
		{"admin", "member", true},
		{"admin", "viewer", true},
		{"member", "member", true},
		{"member", "viewer", true},
		{"viewer", "viewer", true},
		// Level insufficient
		{"viewer", "member", false},
		{"viewer", "admin", false},
		{"viewer", "owner", false},
		{"member", "admin", false},
		{"member", "owner", false},
		{"admin", "owner", false},
		// Case insensitive
		{"OWNER", "admin", true},
		{"Admin", "Member", true},
		{"Member", "Viewer", true},
		// Invalid roles
		{"", "viewer", false},
		{"unknown", "viewer", false},
		// Invalid required returns 0, any valid role >= 0
		{"viewer", "unknown", true},
		{"viewer", "", true},
	}
	for _, tt := range tests {
		got := HasProjectRoleAtLeast(tt.role, tt.required)
		if got != tt.expected {
			t.Errorf("HasProjectRoleAtLeast(%q, %q) = %v, want %v", tt.role, tt.required, got, tt.expected)
		}
	}
}
