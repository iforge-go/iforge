package model

import "time"

// Webhook represents a repository webhook
type Webhook struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserName       string    `gorm:"index;column:user_name" json:"userName"`
	RepositoryName string    `gorm:"index;column:repository_name" json:"repositoryName"`
	URL            string    `gorm:"column:url" json:"url"`
	Token          *string   `gorm:"column:token" json:"token"`
	ContentType    string    `gorm:"column:content_type" json:"contentType"`
	Active         bool      `gorm:"column:active;default:true" json:"active"`
	Events         string    `gorm:"column:events;type:text" json:"events"` // JSON array of event types
	SSLVerify      bool      `gorm:"column:ssl_verify;default:true" json:"sslVerify"`
	InsecureSSL    bool      `gorm:"column:insecure_ssl;default:false" json:"insecureSsl"`
	CreatedAt      time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt      time.Time `gorm:"column:updated_at" json:"updatedAt"`
}

func (Webhook) TableName() string { return "webhooks" }
