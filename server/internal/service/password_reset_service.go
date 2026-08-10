package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

var (
	ErrTokenNotFound = errors.New("token not found")
	ErrTokenExpired  = errors.New("token expired")
)

// PasswordResetService handles password reset operations
type PasswordResetService struct {
	db *gorm.DB
}

// NewPasswordResetService creates a new PasswordResetService
func NewPasswordResetService(db *gorm.DB) *PasswordResetService {
	return &PasswordResetService{db: db}
}

// GenerateToken generates a random token for password reset
func (s *PasswordResetService) GenerateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CreateResetToken creates a password reset token for a user
func (s *PasswordResetService) CreateResetToken(username string) (string, error) {
	token, err := s.GenerateToken()
	if err != nil {
		return "", err
	}

	// Token expires in 24 hours
	expiresAt := time.Now().Add(24 * time.Hour)

	resetToken := &model.PasswordResetToken{
		Token:     token,
		UserName:  username,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

	if err := s.db.Create(resetToken).Error; err != nil {
		return "", err
	}

	return token, nil
}

// ValidateToken validates a password reset token
func (s *PasswordResetService) ValidateToken(token string) (string, error) {
	var resetToken model.PasswordResetToken
	err := s.db.Where("token = ?", token).First(&resetToken).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", ErrTokenNotFound
		}
		return "", err
	}

	if time.Now().After(resetToken.ExpiresAt) {
		return "", ErrTokenExpired
	}

	return resetToken.UserName, nil
}

// ResetPassword resets a user's password using a token
func (s *PasswordResetService) ResetPassword(token, newPassword string) error {
	username, err := s.ValidateToken(token)
	if err != nil {
		return err
	}

	// Hash the new password using bcrypt
	hashedPassword, err := hashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	if err := s.db.Model(&model.Account{}).
		Where("user_name = ?", username).
		Update("password", hashedPassword).Error; err != nil {
		return err
	}

	// Delete the token
	if err := s.db.Where("token = ?", token).
		Delete(&model.PasswordResetToken{}).Error; err != nil {
		return err
	}

	return nil
}

// DeleteToken deletes a password reset token
func (s *PasswordResetService) DeleteToken(token string) error {
	return s.db.Where("token = ?", token).
		Delete(&model.PasswordResetToken{}).Error
}

// CleanupExpiredTokens removes all expired tokens
func (s *PasswordResetService) CleanupExpiredTokens() error {
	return s.db.Where("expires_at < ?", time.Now()).
		Delete(&model.PasswordResetToken{}).Error
}
