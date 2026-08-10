package service

import (
	"errors"

	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

var (
	ErrPreferenceNotFound = errors.New("preference not found")
)

type AccountPreferenceService struct {
	db *gorm.DB
}

func NewAccountPreferenceService(db *gorm.DB) *AccountPreferenceService {
	return &AccountPreferenceService{db: db}
}

// GetPreference gets account preferences for a user
func (s *AccountPreferenceService) GetPreference(userName string) (*model.AccountPreference, error) {
	pref := &model.AccountPreference{}
	err := s.db.Where("user_name = ?", userName).First(pref).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Return default preferences
			return &model.AccountPreference{
				UserName:         userName,
				HighlighterTheme: "github-v2",
				Notification:     true,
				Timezone:         "UTC",
			}, nil
		}
		return nil, err
	}
	return pref, nil
}

// UpdatePreference updates account preferences
func (s *AccountPreferenceService) UpdatePreference(pref *model.AccountPreference) error {
	var count int64
	s.db.Model(&model.AccountPreference{}).Where("user_name = ?", pref.UserName).Count(&count)

	if count > 0 {
		return s.db.Model(&model.AccountPreference{}).
			Where("user_name = ?", pref.UserName).
			Updates(map[string]interface{}{
				"highlighter_theme": pref.HighlighterTheme,
				"notification":      pref.Notification,
				"timezone":          pref.Timezone,
			}).Error
	}

	return s.db.Create(pref).Error
}
