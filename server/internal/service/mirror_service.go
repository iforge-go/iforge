package service

import (
	"errors"
	"time"

	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

var (
	ErrMirrorNotFound = errors.New("mirror not found")
)

// MirrorService handles mirror operations
type MirrorService struct {
	db *gorm.DB
}

// NewMirrorService creates a new MirrorService
func NewMirrorService(db *gorm.DB) *MirrorService {
	return &MirrorService{db: db}
}

// GetMirror retrieves a mirror configuration
func (s *MirrorService) GetMirror(owner, repo string) (*model.RepositoryMirror, error) {
	mirror := &model.RepositoryMirror{}
	err := s.db.
		Where("user_name = ? AND repository_name = ?", owner, repo).
		First(mirror).Error

	if err == gorm.ErrRecordNotFound {
		return nil, ErrMirrorNotFound
	}
	if err != nil {
		return nil, err
	}

	return mirror, nil
}

// CreateMirror creates a new mirror configuration
func (s *MirrorService) CreateMirror(mirror *model.RepositoryMirror) error {
	now := time.Now()
	mirror.LastSyncDate = now
	mirror.NextSyncDate = now.Add(time.Duration(mirror.SyncInterval) * time.Minute)

	return s.db.Create(mirror).Error
}

// UpdateMirror updates a mirror configuration
func (s *MirrorService) UpdateMirror(mirror *model.RepositoryMirror) error {
	nextSync := time.Now().Add(time.Duration(mirror.SyncInterval) * time.Minute)

	return s.db.
		Model(&model.RepositoryMirror{}).
		Where("user_name = ? AND repository_name = ?", mirror.UserName, mirror.RepositoryName).
		Updates(map[string]interface{}{
			"mirror_url":     mirror.MirrorURL,
			"sync_interval":  mirror.SyncInterval,
			"next_sync_date": nextSync,
			"enabled":        mirror.Enabled,
			"sync_on_push":   mirror.SyncOnPush,
			"authentication": mirror.Authentication,
			"username":       mirror.Username,
			"password":       mirror.Password,
			"ssh_key":        mirror.SSHKey,
		}).Error
}

// DeleteMirror deletes a mirror configuration
func (s *MirrorService) DeleteMirror(owner, repo string) error {
	return s.db.
		Where("user_name = ? AND repository_name = ?", owner, repo).
		Delete(&model.RepositoryMirror{}).Error
}

// SyncMirror performs a manual sync of the mirror
func (s *MirrorService) SyncMirror(owner, repo string) error {
	mirror := &model.RepositoryMirror{}
	err := s.db.
		Where("user_name = ? AND repository_name = ?", owner, repo).
		First(mirror).Error

	if err == gorm.ErrRecordNotFound {
		return ErrMirrorNotFound
	}
	if err != nil {
		return err
	}

	now := time.Now()
	nextSync := now.Add(time.Duration(mirror.SyncInterval) * time.Minute)

	return s.db.
		Model(&model.RepositoryMirror{}).
		Where("user_name = ? AND repository_name = ?", owner, repo).
		Updates(map[string]interface{}{
			"last_sync_date": now,
			"next_sync_date": nextSync,
		}).Error
}

// GetMirrorsToSync retrieves mirrors that need to be synced
func (s *MirrorService) GetMirrorsToSync() ([]*model.RepositoryMirror, error) {
	now := time.Now()
	var mirrors []*model.RepositoryMirror

	err := s.db.
		Where("enabled = TRUE AND next_sync_date <= ?", now).
		Order("next_sync_date ASC").
		Find(&mirrors).Error

	return mirrors, err
}
