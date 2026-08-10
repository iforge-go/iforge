package service

import (
	"iforge/iforge/internal/model"
	"time"

	"gorm.io/gorm"
)

type StarService struct {
	db *gorm.DB
}

func NewStarService(db *gorm.DB) *StarService {
	return &StarService{db: db}
}

// StarRepository adds a star to a repository
func (s *StarService) StarRepository(userName, repoName, starrer string) error {
	// Check if already starred
	var count int64
	s.db.Model(&model.RepositoryStar{}).
		Where("user_name = ? AND repository_name = ? AND starrer = ?", userName, repoName, starrer).
		Count(&count)

	if count > 0 {
		return nil // Already starred
	}

	star := &model.RepositoryStar{
		UserName:       userName,
		RepositoryName: repoName,
		Starer:         starrer,
		RegisteredDate: time.Now(),
	}
	return s.db.Create(star).Error
}

// UnstarRepository removes a star from a repository
func (s *StarService) UnstarRepository(userName, repoName, starrer string) error {
	return s.db.
		Where("user_name = ? AND repository_name = ? AND starrer = ?", userName, repoName, starrer).
		Delete(&model.RepositoryStar{}).Error
}

// IsStarred checks if a user has starred a repository
func (s *StarService) IsStarred(userName, repoName, starrer string) (bool, error) {
	var count int64
	err := s.db.Model(&model.RepositoryStar{}).
		Where("user_name = ? AND repository_name = ? AND starrer = ?", userName, repoName, starrer).
		Count(&count).Error
	return count > 0, err
}

// GetStarCount returns the number of stars for a repository
func (s *StarService) GetStarCount(userName, repoName string) (int, error) {
	var count int64
	err := s.db.Model(&model.RepositoryStar{}).
		Where("user_name = ? AND repository_name = ?", userName, repoName).
		Count(&count).Error
	return int(count), err
}

// GetStargazers returns all users who starred a repository
func (s *StarService) GetStargazers(userName, repoName string) ([]string, error) {
	var stars []model.RepositoryStar
	err := s.db.
		Where("user_name = ? AND repository_name = ?", userName, repoName).
		Order("registered_date DESC").
		Find(&stars).Error
	if err != nil {
		return nil, err
	}

	stargazers := make([]string, len(stars))
	for i, star := range stars {
		stargazers[i] = star.Starer
	}
	return stargazers, nil
}

// GetStarredRepos returns all repositories that a user has starred.
func (s *StarService) GetStarredRepos(starrer string) ([]*model.Repository, error) {
	var repos []*model.Repository
	err := s.db.
		Joins("JOIN repository_star rs ON rs.user_name = repository.user_name AND rs.repository_name = repository.repository_name").
		Where("rs.starrer = ?", starrer).
		Order("rs.registered_date DESC").
		Find(&repos).Error
	return repos, err
}

type WatchService struct {
	db *gorm.DB
}

func NewWatchService(db *gorm.DB) *WatchService {
	return &WatchService{db: db}
}

// WatchRepository starts watching a repository
func (s *WatchService) WatchRepository(userName, repoName, watcher string, notification bool) error {
	// Check if already watching
	var count int64
	s.db.Model(&model.RepositoryWatch{}).
		Where("user_name = ? AND repository_name = ? AND watcher = ?", userName, repoName, watcher).
		Count(&count)

	if count > 0 {
		// Update notification setting
		return s.db.Model(&model.RepositoryWatch{}).
			Where("user_name = ? AND repository_name = ? AND watcher = ?", userName, repoName, watcher).
			Update("notification", notification).Error
	}

	watch := &model.RepositoryWatch{
		UserName:       userName,
		RepositoryName: repoName,
		Watcher:        watcher,
		Notification:   notification,
	}
	return s.db.Create(watch).Error
}

// UnwatchRepository stops watching a repository
func (s *WatchService) UnwatchRepository(userName, repoName, watcher string) error {
	return s.db.
		Where("user_name = ? AND repository_name = ? AND watcher = ?", userName, repoName, watcher).
		Delete(&model.RepositoryWatch{}).Error
}

// IsWatching checks if a user is watching a repository
func (s *WatchService) IsWatching(userName, repoName, watcher string) (bool, error) {
	var count int64
	err := s.db.Model(&model.RepositoryWatch{}).
		Where("user_name = ? AND repository_name = ? AND watcher = ?", userName, repoName, watcher).
		Count(&count).Error
	return count > 0, err
}

// GetWatchCount returns the number of watchers for a repository
func (s *WatchService) GetWatchCount(userName, repoName string) (int, error) {
	var count int64
	err := s.db.Model(&model.RepositoryWatch{}).
		Where("user_name = ? AND repository_name = ?", userName, repoName).
		Count(&count).Error
	return int(count), err
}

// GetWatchers returns all users watching a repository
func (s *WatchService) GetWatchers(userName, repoName string) ([]string, error) {
	var watches []model.RepositoryWatch
	err := s.db.
		Where("user_name = ? AND repository_name = ?", userName, repoName).
		Find(&watches).Error
	if err != nil {
		return nil, err
	}

	watchers := make([]string, len(watches))
	for i, watch := range watches {
		watchers[i] = watch.Watcher
	}
	return watchers, nil
}
