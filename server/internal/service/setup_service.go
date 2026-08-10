package service

import (
	"sync/atomic"

	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

// SetupService manages the system initialization state.
// It caches the initialized flag in memory for fast middleware checks.
type SetupService struct {
	db          *gorm.DB
	initialized atomic.Bool
}

// NewSetupService creates a new SetupService and reads the initial state from DB.
func NewSetupService(db *gorm.DB) *SetupService {
	s := &SetupService{db: db}
	s.RefreshInitialized()
	return s
}

// IsInitialized returns the cached initialization status.
func (s *SetupService) IsInitialized() bool {
	return s.initialized.Load()
}

// RefreshInitialized reads the initialization status from the database
// and updates the in-memory cache.
func (s *SetupService) RefreshInitialized() {
	var setting model.SystemSetting
	err := s.db.Where("`key` = ?", "initialized").First(&setting).Error
	if err == nil && setting.Value == "true" {
		s.initialized.Store(true)
	} else {
		s.initialized.Store(false)
	}
}
