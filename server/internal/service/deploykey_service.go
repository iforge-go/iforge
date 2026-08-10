package service

import (
	"errors"
	"time"

	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

var (
	ErrDeployKeyNotFound = errors.New("deploy key not found")
)

type DeployKeyService struct {
	db *gorm.DB
}

func NewDeployKeyService(db *gorm.DB) *DeployKeyService {
	return &DeployKeyService{db: db}
}

// ListDeployKeys retrieves all deploy keys for a repository
func (s *DeployKeyService) ListDeployKeys(userName, repoName string) ([]*model.DeployKey, error) {
	var deployKeys []*model.DeployKey
	err := s.db.
		Where("user_name = ? AND repository_name = ?", userName, repoName).
		Order("registered_date DESC").
		Find(&deployKeys).Error
	return deployKeys, err
}

// GetDeployKey retrieves a deploy key by ID
func (s *DeployKeyService) GetDeployKey(userName, repoName string, deployKeyID int) (*model.DeployKey, error) {
	dk := &model.DeployKey{}
	err := s.db.
		Where("user_name = ? AND repository_name = ? AND deploy_key_id = ?", userName, repoName, deployKeyID).
		First(dk).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrDeployKeyNotFound
		}
		return nil, err
	}
	return dk, nil
}

// CreateDeployKey creates a new deploy key
func (s *DeployKeyService) CreateDeployKey(userName, repoName, title, publicKey string, allowWrite bool) (*model.DeployKey, error) {
	dk := &model.DeployKey{
		UserName:       userName,
		RepositoryName: repoName,
		Title:          title,
		PublicKey:      publicKey,
		AllowWrite:     allowWrite,
		RegisteredDate: time.Now(),
	}

	if err := s.db.Create(dk).Error; err != nil {
		return nil, err
	}

	return dk, nil
}

// DeleteDeployKey deletes a deploy key
func (s *DeployKeyService) DeleteDeployKey(userName, repoName string, deployKeyID int) error {
	result := s.db.
		Where("user_name = ? AND repository_name = ? AND deploy_key_id = ?", userName, repoName, deployKeyID).
		Delete(&model.DeployKey{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrDeployKeyNotFound
	}
	return nil
}

// GetDeployKeyByPublicKey finds a deploy key by its public key
func (s *DeployKeyService) GetDeployKeyByPublicKey(publicKey string) (*model.DeployKey, error) {
	dk := &model.DeployKey{}
	err := s.db.Where("public_key = ?", publicKey).First(dk).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDeployKeyNotFound
		}
		return nil, err
	}
	return dk, nil
}
