package service

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

// ReleaseService handles release operations
type ReleaseService struct {
	db *gorm.DB
}

// NewReleaseService creates a new ReleaseService
func NewReleaseService(db *gorm.DB) *ReleaseService {
	return &ReleaseService{
		db: db,
	}
}

// ListReleases lists all releases for a repository
func (s *ReleaseService) ListReleases(userName, repositoryName string) ([]*model.ReleaseTag, error) {
	var releases []*model.ReleaseTag
	err := s.db.
		Where("user_name = ? AND repository_name = ?", userName, repositoryName).
		Order("registered_date DESC").
		Find(&releases).Error

	if err != nil {
		return nil, fmt.Errorf("failed to query releases: %w", err)
	}

	return releases, nil
}

// GetRelease gets a specific release
func (s *ReleaseService) GetRelease(userName, repositoryName, tag string) (*model.ReleaseTag, error) {
	release := &model.ReleaseTag{}
	err := s.db.
		Where("user_name = ? AND repository_name = ? AND tag = ?", userName, repositoryName, tag).
		First(release).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("release not found")
		}
		return nil, fmt.Errorf("failed to get release: %w", err)
	}

	return release, nil
}

// CreateRelease creates a new release
func (s *ReleaseService) CreateRelease(userName, repositoryName, tag, name, content, author string) (*model.ReleaseTag, error) {
	registeredDate := time.Now()
	release := &model.ReleaseTag{
		UserName:       userName,
		RepositoryName: repositoryName,
		Tag:            tag,
		Name:           name,
		Content:        &content,
		RegisteredDate: registeredDate,
		Author:         author,
	}

	if err := s.db.Create(release).Error; err != nil {
		return nil, fmt.Errorf("failed to create release: %w", err)
	}

	return release, nil
}

// CreateReleaseIfNotExists idempotently creates a release.
// Returns existing release if (userName, repositoryName, tag) already exists.
// Used by CICD executor to auto-publish after release job completion.
func (s *ReleaseService) CreateReleaseIfNotExists(userName, repositoryName, tag, name, content, author string) (*model.ReleaseTag, error) {
	// Check if already exists (same tag may be referenced by multiple release jobs or retries)
	existing, err := s.GetRelease(userName, repositoryName, tag)
	if err == nil && existing != nil {
		return existing, nil
	}

	return s.CreateRelease(userName, repositoryName, tag, name, content, author)
}

// AttachAssetFromFile copies a file to release upload directory and creates DB record.
// Used by CICD executor to upload build artifacts as release assets.
//   - Copies sourcePath to {uploadDir}/{owner}/{repo}/{tag}/{fileName}
//   - Calls AddReleaseAsset to write DB record
//
// Parameters:
//   - uploadDir: release upload root directory
//   - sourcePath: absolute path of source file
//   - fileName: uploaded file name (defaults to basename of sourcePath if empty)
func (s *ReleaseService) AttachAssetFromFile(
	userName, repositoryName, tag, uploadDir, sourcePath, fileName, uploader string,
) (*model.ReleaseAsset, error) {
	// Default file name
	if fileName == "" {
		fileName = filepath.Base(sourcePath)
	}
	// Security check: file name must be basename to prevent path traversal
	fileName = filepath.Base(fileName)
	if fileName == "." || fileName == "/" || fileName == "" {
		return nil, fmt.Errorf("invalid file name: %q", fileName)
	}

	// Get file size
	info, err := os.Stat(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("source file not found: %w", err)
	}
	size := info.Size()

	// Create target directory
	assetDir := filepath.Join(uploadDir, userName, repositoryName, tag)
	if err := os.MkdirAll(assetDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create asset dir: %w", err)
	}

	// Copy file
	dstPath := filepath.Join(assetDir, fileName)
	srcFile, err := os.Open(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create dst file: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		os.Remove(dstPath) // Clean up failed file
		return nil, fmt.Errorf("failed to copy file: %w", err)
	}

	// Write DB record
	asset, err := s.AddReleaseAsset(userName, repositoryName, tag, fileName, nil, size, uploader)
	if err != nil {
		os.Remove(dstPath) // Clean up file on DB failure
		return nil, err
	}
	return asset, nil
}

// UpdateRelease updates a release
func (s *ReleaseService) UpdateRelease(userName, repositoryName, tag, name, content string) error {
	err := s.db.
		Model(&model.ReleaseTag{}).
		Where("user_name = ? AND repository_name = ? AND tag = ?", userName, repositoryName, tag).
		Updates(map[string]interface{}{
			"name":    name,
			"content": content,
		}).Error

	if err != nil {
		return fmt.Errorf("failed to update release: %w", err)
	}

	return nil
}

// DeleteRelease deletes a release
func (s *ReleaseService) DeleteRelease(userName, repositoryName, tag string) error {
	// First delete all assets
	if err := s.db.
		Where("user_name = ? AND repository_name = ? AND tag = ?", userName, repositoryName, tag).
		Delete(&model.ReleaseAsset{}).Error; err != nil {
		return fmt.Errorf("failed to delete release assets: %w", err)
	}

	// Then delete the release
	if err := s.db.
		Where("user_name = ? AND repository_name = ? AND tag = ?", userName, repositoryName, tag).
		Delete(&model.ReleaseTag{}).Error; err != nil {
		return fmt.Errorf("failed to delete release: %w", err)
	}

	return nil
}

// ListReleaseAssets lists all assets for a release
func (s *ReleaseService) ListReleaseAssets(userName, repositoryName, tag string) ([]*model.ReleaseAsset, error) {
	var assets []*model.ReleaseAsset
	err := s.db.
		Where("user_name = ? AND repository_name = ? AND tag = ?", userName, repositoryName, tag).
		Order("registered_date DESC").
		Find(&assets).Error

	if err != nil {
		return nil, fmt.Errorf("failed to query release assets: %w", err)
	}

	return assets, nil
}

// AddReleaseAsset adds an asset to a release
func (s *ReleaseService) AddReleaseAsset(userName, repositoryName, tag, fileName string, label *string, size int64, uploader string) (*model.ReleaseAsset, error) {
	registeredDate := time.Now()
	asset := &model.ReleaseAsset{
		UserName:       userName,
		RepositoryName: repositoryName,
		Tag:            tag,
		FileName:       fileName,
		Label:          label,
		Size:           size,
		Uploader:       uploader,
		RegisteredDate: registeredDate,
	}

	if err := s.db.Create(asset).Error; err != nil {
		return nil, fmt.Errorf("failed to add release asset: %w", err)
	}

	return asset, nil
}

// GetAsset gets a specific release asset by ID
func (s *ReleaseService) GetAsset(userName, repositoryName, tag string, assetID int) (*model.ReleaseAsset, error) {
	asset := &model.ReleaseAsset{}
	err := s.db.
		Where("user_name = ? AND repository_name = ? AND tag = ? AND asset_id = ?", userName, repositoryName, tag, assetID).
		First(asset).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("asset not found")
		}
		return nil, fmt.Errorf("failed to get asset: %w", err)
	}

	return asset, nil
}

// DeleteReleaseAsset deletes a release asset
func (s *ReleaseService) DeleteReleaseAsset(userName, repositoryName, tag string, assetID int) error {
	err := s.db.
		Where("user_name = ? AND repository_name = ? AND tag = ? AND asset_id = ?", userName, repositoryName, tag, assetID).
		Delete(&model.ReleaseAsset{}).Error

	if err != nil {
		return fmt.Errorf("failed to delete release asset: %w", err)
	}

	return nil
}
