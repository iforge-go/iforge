package model

import "strings"

// Project member role constants (four levels).
//
//	owner  = project creator; unique; non-transferable (transfer not implemented in this phase)
//	admin  = member management + repository association + all Scrum writes
//	member = all Scrum writes
//	viewer = read-only
const (
	ProjectRoleOwner  = "owner"
	ProjectRoleAdmin  = "admin"
	ProjectRoleMember = "member"
	ProjectRoleViewer = "viewer"
)

// ProjectRoleLevel returns the numeric level of a role (higher = more permissions).
// owner=4, admin=3, member=2, viewer=1; unknown role or empty string returns 0.
func ProjectRoleLevel(role string) int {
	switch strings.ToLower(role) {
	case ProjectRoleOwner:
		return 4
	case ProjectRoleAdmin:
		return 3
	case ProjectRoleMember:
		return 2
	case ProjectRoleViewer:
		return 1
	default:
		return 0
	}
}

// ValidProjectRole checks whether the role is one of the four legal levels.
// Note: owner can only be produced by project creation in business logic;
// callers of AddMember/UpdateMemberRole should reject owner before invoking this function.
func ValidProjectRole(role string) bool {
	switch strings.ToLower(role) {
	case ProjectRoleOwner, ProjectRoleAdmin, ProjectRoleMember, ProjectRoleViewer:
		return true
	}
	return false
}

// HasProjectRoleAtLeast reports whether the role level is >= required.
// Used by RequireRole gating: the user's role level must meet or exceed the required role.
func HasProjectRoleAtLeast(role, required string) bool {
	return ProjectRoleLevel(role) >= ProjectRoleLevel(required)
}
