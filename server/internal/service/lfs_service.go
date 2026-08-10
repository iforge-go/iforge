package service

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"

	"iforge/iforge/internal/git"
	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

// ErrLFSObjectNotFound is returned when an LFS object cannot be located
var ErrLFSObjectNotFound = errors.New("lfs object not found")

// LFSService handles Git LFS object storage and metadata
type LFSService struct {
	db        *gorm.DB
	gitClient *git.Client
}

// NewLFSService creates a new LFSService
func NewLFSService(db *gorm.DB, gitClient *git.Client) *LFSService {
	return &LFSService{db: db, gitClient: gitClient}
}

// ObjectPath returns the on-disk path for a sharded LFS object file
func (s *LFSService) ObjectPath(owner, repo, oid string) string {
	if len(oid) < 4 {
		return filepath.Join(s.gitClient.RepositoryPath(owner, repo), "lfs", "objects", oid)
	}
	return filepath.Join(
		s.gitClient.RepositoryPath(owner, repo),
		"lfs", "objects",
		oid[:2], oid[2:4], oid,
	)
}

// SaveMeta persists LFS object metadata. If a record already exists, it is a no-op.
func (s *LFSService) SaveMeta(owner, repo, oid string, size int64) error {
	existing, err := s.GetMeta(owner, repo, oid)
	if err == nil && existing != nil {
		return nil
	}
	obj := &model.LFSObject{
		UserName:       owner,
		RepositoryName: repo,
		OID:            oid,
		Size:           size,
		CreatedAt:      time.Now(),
	}
	return s.db.Create(obj).Error
}

// GetMeta retrieves LFS object metadata by oid
func (s *LFSService) GetMeta(owner, repo, oid string) (*model.LFSObject, error) {
	var obj model.LFSObject
	err := s.db.Where("user_name = ? AND repository_name = ? AND oid = ?", owner, repo, oid).
		First(&obj).Error
	if err == gorm.ErrRecordNotFound {
		return nil, ErrLFSObjectNotFound
	}
	if err != nil {
		return nil, err
	}
	return &obj, nil
}

// MetaExists reports whether metadata for the given oid exists
func (s *LFSService) MetaExists(owner, repo, oid string) bool {
	var count int64
	s.db.Model(&model.LFSObject{}).
		Where("user_name = ? AND repository_name = ? AND oid = ?", owner, repo, oid).
		Count(&count)
	return count > 0
}

// ListObjects returns a paginated list of LFS objects for a repository
func (s *LFSService) ListObjects(owner, repo string, page, limit int) ([]*model.LFSObject, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset := (page - 1) * limit

	var total int64
	if err := s.db.Model(&model.LFSObject{}).
		Where("user_name = ? AND repository_name = ?", owner, repo).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var objects []*model.LFSObject
	err := s.db.Where("user_name = ? AND repository_name = ?", owner, repo).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&objects).Error
	return objects, total, err
}

// DeleteObject removes both the metadata record and the on-disk file
func (s *LFSService) DeleteObject(owner, repo, oid string) error {
	if err := s.db.Where("user_name = ? AND repository_name = ? AND oid = ?", owner, repo, oid).
		Delete(&model.LFSObject{}).Error; err != nil {
		return err
	}
	objPath := s.ObjectPath(owner, repo, oid)
	if err := os.Remove(objPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// WriteObjectContent stores the raw object content on disk
func (s *LFSService) WriteObjectContent(owner, repo, oid string, reader io.Reader) error {
	objPath := s.ObjectPath(owner, repo, oid)
	if err := os.MkdirAll(filepath.Dir(objPath), 0o755); err != nil {
		return err
	}
	f, err := os.Create(objPath)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, reader); err != nil {
		return err
	}
	return nil
}

// ReadObjectContent opens the object file for streaming
func (s *LFSService) ReadObjectContent(owner, repo, oid string) (io.ReadCloser, int64, error) {
	objPath := s.ObjectPath(owner, repo, oid)
	info, err := os.Stat(objPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, ErrLFSObjectNotFound
		}
		return nil, 0, err
	}
	f, err := os.Open(objPath)
	if err != nil {
		return nil, 0, err
	}
	return f, info.Size(), nil
}
