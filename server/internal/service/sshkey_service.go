package service

import (
	"fmt"
	"strings"
	"time"

	"iforge/iforge/internal/model"

	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

// normalizePublicKey parses an SSH public key (authorized_keys format, possibly
// with comment) and re-serializes it to a canonical "<type> <base64>" string
// without trailing comment or newline. This ensures database-stored keys and
// keys presented during SSH auth compare equal.
func normalizePublicKey(publicKey string) (string, error) {
	key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(publicKey))
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(ssh.MarshalAuthorizedKey(key)), "\n"), nil
}

// SSHKeyService handles SSH key operations
type SSHKeyService struct {
	db *gorm.DB
}

// NewSSHKeyService creates a new SSHKeyService
func NewSSHKeyService(db *gorm.DB) *SSHKeyService {
	return &SSHKeyService{db: db}
}

// ListSSHKeys lists all SSH keys for a user
func (s *SSHKeyService) ListSSHKeys(userName string) ([]*model.SSHKey, error) {
	var keys []*model.SSHKey
	err := s.db.
		Where("user_name = ?", userName).
		Order("registered_date DESC").
		Find(&keys).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query SSH keys: %w", err)
	}
	return keys, nil
}

// GetSSHKey gets an SSH key by ID
func (s *SSHKeyService) GetSSHKey(userName string, sshKeyID int) (*model.SSHKey, error) {
	var key model.SSHKey
	err := s.db.
		Where("user_name = ? AND ssh_key_id = ?", userName, sshKeyID).
		First(&key).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("SSH key not found")
		}
		return nil, fmt.Errorf("failed to get SSH key: %w", err)
	}
	return &key, nil
}

// CreateSSHKey creates a new SSH key
func (s *SSHKeyService) CreateSSHKey(userName, title, publicKey string) (*model.SSHKey, error) {
	// Normalize public key to canonical format (strips comment/whitespace)
	normalized, err := normalizePublicKey(publicKey)
	if err != nil {
		return nil, fmt.Errorf("invalid SSH public key: %w", err)
	}

	// Check if public key already exists
	var count int64
	s.db.Model(&model.SSHKey{}).Where("public_key = ?", normalized).Count(&count)
	if count > 0 {
		return nil, fmt.Errorf("this SSH key already exists")
	}

	now := time.Now()
	key := &model.SSHKey{
		UserName:       userName,
		Title:          title,
		PublicKey:      normalized,
		RegisteredDate: now,
	}

	if err := s.db.Create(key).Error; err != nil {
		return nil, fmt.Errorf("failed to create SSH key: %w", err)
	}

	return key, nil
}

// GetAccountByPublicKey looks up the account that owns a given public key.
// The publicKey argument should be in authorized_keys format; it is normalized
// before querying so callers may pass raw key material with or without comment.
func (s *SSHKeyService) GetAccountByPublicKey(publicKey string) (*model.Account, error) {
	normalized, err := normalizePublicKey(publicKey)
	if err != nil {
		return nil, fmt.Errorf("invalid public key: %w", err)
	}

	var sshKey model.SSHKey
	if err := s.db.Where("public_key = ?", normalized).First(&sshKey).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("SSH key not found")
		}
		return nil, err
	}

	account := &model.Account{}
	if err := s.db.Where("user_name = ? AND removed = ?", sshKey.UserName, false).First(account).Error; err != nil {
		return nil, err
	}
	return account, nil
}

// DeleteSSHKey deletes an SSH key
func (s *SSHKeyService) DeleteSSHKey(userName string, sshKeyID int) error {
	result := s.db.
		Where("user_name = ? AND ssh_key_id = ?", userName, sshKeyID).
		Delete(&model.SSHKey{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete SSH key: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("SSH key not found")
	}
	return nil
}
