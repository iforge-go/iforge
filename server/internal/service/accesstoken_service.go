package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

type AccessTokenService struct {
	db *gorm.DB
}

func NewAccessTokenService(db *gorm.DB) *AccessTokenService {
	return &AccessTokenService{db: db}
}

func (s *AccessTokenService) ListAccessTokens(userName string) ([]*model.AccessToken, error) {
	var tokens []*model.AccessToken
	err := s.db.
		Where("user_name = ?", userName).
		Order("registered_date DESC").
		Find(&tokens).Error
	return tokens, err
}

func (s *AccessTokenService) GetAccessToken(userName string, tokenID int) (*model.AccessToken, error) {
	var token model.AccessToken
	err := s.db.
		Where("user_name = ? AND access_token_id = ?", userName, tokenID).
		First(&token).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("access token not found")
		}
		return nil, err
	}
	return &token, nil
}

func (s *AccessTokenService) CreateAccessToken(userName, note string) (*model.AccessToken, string, error) {
	// Generate random token
	tokenBytes := make([]byte, 32)
	_, err := rand.Read(tokenBytes)
	if err != nil {
		return nil, "", err
	}
	token := hex.EncodeToString(tokenBytes)

	now := time.Now()
	accessToken := &model.AccessToken{
		UserName:       userName,
		Token:          token,
		Note:           note,
		RegisteredDate: now,
	}

	if err := s.db.Create(accessToken).Error; err != nil {
		return nil, "", err
	}

	return accessToken, token, nil
}

func (s *AccessTokenService) DeleteAccessToken(id int, userName string) error {
	result := s.db.
		Where("access_token_id = ? AND user_name = ?", id, userName).
		Delete(&model.AccessToken{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("access token not found")
	}
	return nil
}

func (s *AccessTokenService) ValidateAccessToken(token string) (*model.Account, error) {
	var accessToken model.AccessToken
	err := s.db.Where("token = ?", token).First(&accessToken).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("invalid access token")
		}
		return nil, err
	}

	var account model.Account
	// Filter by removed = false: tokens of soft-deleted users must not access the API.
	err = s.db.Where("user_name = ? AND removed = ?", accessToken.UserName, false).First(&account).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}

	return &account, nil
}
