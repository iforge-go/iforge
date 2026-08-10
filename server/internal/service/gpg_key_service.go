package service

import (
	"errors"
	"time"

	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

var (
	ErrGPGKeyNotFound = errors.New("GPG key not found")
)

type GPGKeyService struct {
	db *gorm.DB
}

func NewGPGKeyService(db *gorm.DB) *GPGKeyService {
	return &GPGKeyService{db: db}
}

// ListGPGKeys lists all GPG keys for a user
func (s *GPGKeyService) ListGPGKeys(userName string) ([]*model.GPGKey, error) {
	var keys []*model.GPGKey
	err := s.db.
		Where("user_name = ?", userName).
		Order("registered_date DESC").
		Find(&keys).Error
	return keys, err
}

// GetGPGKey gets a specific GPG key
func (s *GPGKeyService) GetGPGKey(userName string, keyID int64) (*model.GPGKey, error) {
	key := &model.GPGKey{}
	err := s.db.
		Where("user_name = ? AND key_id = ?", userName, keyID).
		First(key).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrGPGKeyNotFound
		}
		return nil, err
	}
	return key, nil
}

// AddGPGKey adds a new GPG key
func (s *GPGKeyService) AddGPGKey(userName, title, publicKey string) (*model.GPGKey, error) {
	// TODO: Parse GPG public key to extract key ID
	// For now, generate a simple key ID based on timestamp
	keyID := time.Now().UnixNano()
	gpgKeyID := generateGPGKeyID(publicKey)

	key := &model.GPGKey{
		UserName:       userName,
		KeyID:          keyID,
		GpgKeyID:       gpgKeyID,
		Title:          title,
		PublicKey:      publicKey,
		RegisteredDate: time.Now(),
	}

	if err := s.db.Create(key).Error; err != nil {
		return nil, err
	}

	return key, nil
}

// DeleteGPGKey deletes a GPG key
func (s *GPGKeyService) DeleteGPGKey(userName string, keyID int64) error {
	return s.db.
		Where("user_name = ? AND key_id = ?", userName, keyID).
		Delete(&model.GPGKey{}).Error
}

// VerifyCommitSignature verifies a commit signature using GPG keys
func (s *GPGKeyService) VerifyCommitSignature(commitID string) (bool, string, error) {
	// TODO: Implement GPG signature verification
	// This would require parsing the commit signature and verifying it against stored keys
	return false, "", nil
}

// generateGPGKeyID generates a GPG key ID from a public key
func generateGPGKeyID(publicKey string) string {
	// TODO: Parse the actual GPG public key to extract the key ID
	// For now, return a placeholder
	if len(publicKey) >= 16 {
		return publicKey[:16]
	}
	return "0000000000000000"
}
