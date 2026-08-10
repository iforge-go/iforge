package service

import (
	"errors"
	"strings"
	"time"

	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

var (
	ErrAIModelConfigNotFound     = errors.New("AI model config not found")
	ErrAIModelConfigNameConflict = errors.New("AI model config name already exists")
	ErrAIModelConfigLastDefault  = errors.New("cannot delete the default AI model config; set another as default first")
	ErrAIModelConfigNoDefault    = errors.New("no default AI model config; set one as default in admin settings")
)

type AIModelConfigService struct {
	db *gorm.DB
}

// NewAIModelConfigService creates a new AIModelConfigService
func NewAIModelConfigService(db *gorm.DB) *AIModelConfigService {
	return &AIModelConfigService{db: db}
}

// List returns all configurations, sorted by IsDefault DESC, UpdatedAt DESC (default config first).
func (s *AIModelConfigService) List() ([]*model.AIModelConfig, error) {
	var cfgs []*model.AIModelConfig
	err := s.db.Order("is_default DESC, updated_at DESC").Find(&cfgs).Error
	return cfgs, err
}

func (s *AIModelConfigService) Get(id uint) (*model.AIModelConfig, error) {
	var cfg model.AIModelConfig
	err := s.db.Where("id = ?", id).First(&cfg).Error
	if err == gorm.ErrRecordNotFound {
		return nil, ErrAIModelConfigNotFound
	}
	return &cfg, err
}

func (s *AIModelConfigService) GetDefault() (*model.AIModelConfig, error) {
	var cfg model.AIModelConfig
	err := s.db.Where("is_default = ?", true).First(&cfg).Error
	if err == gorm.ErrRecordNotFound {
		return nil, ErrAIModelConfigNoDefault
	}
	return &cfg, err
}

func (s *AIModelConfigService) Create(cfg *model.AIModelConfig) (*model.AIModelConfig, error) {
	cfg.ID = 0
	cfg.CreatedAt = time.Now()
	cfg.UpdatedAt = time.Now()

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if cfg.IsDefault {
			if e := tx.Model(&model.AIModelConfig{}).Where("is_default = ?", true).Update("is_default", false).Error; e != nil {
				return e
			}
		}
		return tx.Create(cfg).Error
	})
	if err != nil {
		if isDuplicateKeyError(err) {
			return nil, ErrAIModelConfigNameConflict
		}
		return nil, err
	}
	return cfg, nil
}

func (s *AIModelConfigService) Update(id uint, cfg *model.AIModelConfig) (*model.AIModelConfig, error) {
	if cfg.APIKey == "" || strings.Contains(cfg.APIKey, "****") {
		var existing model.AIModelConfig
		if err := s.db.Select("api_key").Where("id = ?", id).First(&existing).Error; err == nil {
			cfg.APIKey = existing.APIKey
		}
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// If setting as default, clear default flag from other configs first
		if cfg.IsDefault {
			if e := tx.Model(&model.AIModelConfig{}).Where("is_default = ? AND id != ?", true, id).Update("is_default", false).Error; e != nil {
				return e
			}
		}
		result := tx.Model(&model.AIModelConfig{}).Where("id = ?", id).Updates(map[string]interface{}{
			"name":       cfg.Name,
			"provider":   cfg.Provider,
			"base_url":   cfg.BaseURL,
			"api_key":    cfg.APIKey,
			"model":      cfg.Model,
			"timeout":    cfg.Timeout,
			"max_tokens": cfg.MaxTokens,
			"enabled":    cfg.Enabled,
			"is_default": cfg.IsDefault,
			"updated_at": time.Now(),
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrAIModelConfigNotFound
		}
		return nil
	})
	if err != nil {
		if isDuplicateKeyError(err) {
			return nil, ErrAIModelConfigNameConflict
		}
		if err == ErrAIModelConfigNotFound {
			return nil, err
		}
		return nil, err
	}

	return s.Get(id)
}

func (s *AIModelConfigService) Delete(id uint) error {
	var cfg model.AIModelConfig
	if err := s.db.Where("id = ?", id).First(&cfg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrAIModelConfigNotFound
		}
		return err
	}

	if cfg.IsDefault {
		var count int64
		if err := s.db.Model(&model.AIModelConfig{}).Where("id != ?", id).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrAIModelConfigLastDefault
		}
	}

	return s.db.Where("id = ?", id).Delete(&model.AIModelConfig{}).Error
}

func (s *AIModelConfigService) SetDefault(id uint) (*model.AIModelConfig, error) {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if e := tx.Model(&model.AIModelConfig{}).Where("1 = 1").Update("is_default", false).Error; e != nil {
			return e
		}
		result := tx.Model(&model.AIModelConfig{}).Where("id = ?", id).Updates(map[string]interface{}{
			"is_default": true,
			"enabled":    true,
			"updated_at": time.Now(),
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrAIModelConfigNotFound
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Get(id)
}

// UpdateTestResult updates the test status and test time for the specified configuration.
// success=true sets test_status="success"; success=false sets test_status="failed".
func (s *AIModelConfigService) UpdateTestResult(id uint, success bool) error {
	status := "success"
	if !success {
		status = "failed"
	}
	now := time.Now()
	return s.db.Model(&model.AIModelConfig{}).Where("id = ?", id).Updates(map[string]interface{}{
		"test_status": status,
		"tested_at":   &now,
	}).Error
}

func isDuplicateKeyError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "Duplicate entry") ||
		strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "duplicate key value")
}
