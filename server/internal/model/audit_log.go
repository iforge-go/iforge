package model

import "time"

// AuditLog records security-relevant administrative actions for compliance.
// Unlike Activity (public user-facing feed), AuditLog is admin-only and
// captures destructive/privileged operations: deletes, role changes, system
// settings changes, member role updates, etc.
//
// Stored in the `audit_logs` table.
type AuditLog struct {
	AuditLogID    uint      `gorm:"primaryKey" json:"id"`
	ActorUserName string    `gorm:"index;size:255" json:"actorUserName"`
	ActorIP       string    `gorm:"size:64" json:"actorIP"`
	Action        string    `gorm:"index;size:100" json:"action"`       // e.g., "repository.delete"
	ResourceType  string    `gorm:"index;size:50" json:"resourceType"` // e.g., "repository"
	ResourceID    string    `gorm:"size:255" json:"resourceId"`        // e.g., "owner/repo"
	Detail        string    `gorm:"type:text" json:"detail"`           // JSON string
	Success       bool      `json:"success"`
	CreatedAt     time.Time `gorm:"index" json:"createdAt"`
}

func (AuditLog) TableName() string { return "audit_logs" }
