package service

import (
	"encoding/json"
	"time"

	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

// AuditService records and queries security-relevant administrative actions.
// Log() is fire-and-forget (errors swallowed) so audit logging never blocks
// the business operation it is recording.
type AuditService struct {
	db *gorm.DB
}

func NewAuditService(db *gorm.DB) *AuditService {
	return &AuditService{db: db}
}

// Log records an audit entry. detail may be nil.
// Errors are logged via gorm's built-in logger but not returned: audit
// logging must never break the business operation it is recording.
func (s *AuditService) Log(actorUserName, actorIP, action, resourceType, resourceID string, detail map[string]interface{}, success bool) {
	var detailJSON string
	if detail != nil {
		if b, err := json.Marshal(detail); err == nil {
			detailJSON = string(b)
		}
	}
	entry := &model.AuditLog{
		ActorUserName: actorUserName,
		ActorIP:       actorIP,
		Action:        action,
		ResourceType:  resourceType,
		ResourceID:    resourceID,
		Detail:        detailJSON,
		Success:       success,
		CreatedAt:     time.Now(),
	}
	_ = s.db.Create(entry).Error
}

// AuditQuery filters for ListAuditLogs.
type AuditQuery struct {
	Actor        string
	Action       string
	ResourceType string
	StartTime    *time.Time
	EndTime      *time.Time
	Page         int
	PageSize     int
}

// ListAuditLogs returns paginated audit logs matching the query, newest first.
func (s *AuditService) ListAuditLogs(q AuditQuery) ([]model.AuditLog, int64, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 200 {
		q.PageSize = 50
	}

	tx := s.db.Model(&model.AuditLog{})
	if q.Actor != "" {
		tx = tx.Where("actor_user_name = ?", q.Actor)
	}
	if q.Action != "" {
		tx = tx.Where("action = ?", q.Action)
	}
	if q.ResourceType != "" {
		tx = tx.Where("resource_type = ?", q.ResourceType)
	}
	if q.StartTime != nil {
		tx = tx.Where("created_at >= ?", *q.StartTime)
	}
	if q.EndTime != nil {
		tx = tx.Where("created_at <= ?", *q.EndTime)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var logs []model.AuditLog
	if err := tx.Order("created_at DESC").
		Offset((q.Page - 1) * q.PageSize).
		Limit(q.PageSize).
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}
