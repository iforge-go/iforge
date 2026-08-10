package model

import "strings"

// Repository collaborator role constants.
// Aligned with PMS module role naming: admin / member / viewer
const (
	RoleAdmin  = "admin"
	RoleMember = "member"
	RoleViewer = "viewer"
)

// RoleHierarchy returns the numeric level of a role (higher = more permissions)
func RoleHierarchy(role string) int {
	switch strings.ToLower(role) {
	case RoleAdmin:
		return 3
	case RoleMember:
		return 2
	case RoleViewer:
		return 1
	default:
		return 0
	}
}

// HasRoleAtLeast checks if a role meets or exceeds the required role level
func HasRoleAtLeast(role string, required string) bool {
	return RoleHierarchy(role) >= RoleHierarchy(required)
}
