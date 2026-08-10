package model

import "testing"

func TestRoleHierarchy(t *testing.T) {
	tests := []struct {
		name     string
		role     string
		expected int
	}{
		{"admin role", "admin", 3},
		{"member role", "member", 2},
		{"viewer role", "viewer", 1},
		{"unknown role", "unknown", 0},
		{"empty role", "", 0},
		{"case insensitive admin", "ADMIN", 3},
		{"case insensitive member", "Member", 2},
		{"case insensitive viewer", "VIEWER", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RoleHierarchy(tt.role)
			if result != tt.expected {
				t.Errorf("RoleHierarchy(%q) = %d, want %d", tt.role, result, tt.expected)
			}
		})
	}
}

func TestHasRoleAtLeast(t *testing.T) {
	tests := []struct {
		name     string
		role     string
		required string
		expected bool
	}{
		{"admin has admin", "admin", "admin", true},
		{"admin has member", "admin", "member", true},
		{"admin has viewer", "admin", "viewer", true},
		{"member has member", "member", "member", true},
		{"member has viewer", "member", "viewer", true},
		{"member not admin", "member", "admin", false},
		{"viewer has viewer", "viewer", "viewer", true},
		{"viewer not member", "viewer", "member", false},
		{"viewer not admin", "viewer", "admin", false},
		{"unknown not viewer", "unknown", "viewer", false},
		{"empty not viewer", "", "viewer", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasRoleAtLeast(tt.role, tt.required)
			if result != tt.expected {
				t.Errorf("HasRoleAtLeast(%q, %q) = %v, want %v", tt.role, tt.required, result, tt.expected)
			}
		})
	}
}
