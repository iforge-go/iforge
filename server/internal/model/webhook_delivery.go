package model

import "time"

// WebhookDelivery represents a webhook delivery attempt
type WebhookDelivery struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	WebhookID    uint      `gorm:"index" json:"webhookId"`
	Event        string    `gorm:"column:event" json:"event"` // push, issues, merge_request, release, ping
	Action       string    `gorm:"column:action" json:"action"` // opened, closed, merged, published, etc.
	StatusCode   int       `gorm:"column:status_code" json:"statusCode"`
	RequestBody  string    `gorm:"column:request_body;type:text" json:"requestBody"`
	ResponseBody string    `gorm:"column:response_body;type:text" json:"responseBody"`
	Duration     int       `gorm:"column:duration" json:"duration"` // milliseconds
	Success      bool      `gorm:"column:success" json:"success"`
	ErrorMessage string    `gorm:"column:error_message" json:"errorMessage"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"createdAt"`
}

func (WebhookDelivery) TableName() string { return "webhook_deliveries" }
