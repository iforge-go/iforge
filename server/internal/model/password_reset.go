package model

import "time"

// PasswordResetToken represents a password reset token
type PasswordResetToken struct {
	Token     string    `gorm:"primaryKey;column:token" json:"token"`
	UserName  string    `gorm:"column:user_name" json:"userName"`
	ExpiresAt time.Time `gorm:"column:expires_at" json:"expiresAt"`
	CreatedAt time.Time `gorm:"column:created_at" json:"createdAt"`
}

func (PasswordResetToken) TableName() string { return "password_reset_token" }
