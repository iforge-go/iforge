package service

import (
	"time"

	"gorm.io/gorm"
)

type WikiPage struct {
	UserName       string    `json:"userName" gorm:"primaryKey;column:user_name"`
	RepositoryName string    `json:"repositoryName" gorm:"primaryKey;column:repository_name"`
	PageName       string    `json:"pageName" gorm:"primaryKey;column:page_name"`
	Title          string    `json:"title" gorm:"column:title"`
	Content        string    `json:"content" gorm:"column:content"`
	Author         string    `json:"author" gorm:"column:author"`
	RegisteredDate time.Time `json:"registeredDate" gorm:"column:registered_date"`
	UpdatedDate    time.Time `json:"updatedDate" gorm:"column:updated_date"`
}

func (WikiPage) TableName() string { return "wiki_page" }

type WikiService struct {
	db *gorm.DB
}

func NewWikiService(db *gorm.DB) *WikiService {
	return &WikiService{db: db}
}

// ListWikiPages returns all wiki pages for a repository
func (s *WikiService) ListWikiPages(userName, repositoryName string) ([]*WikiPage, error) {
	var pages []*WikiPage
	err := s.db.
		Where("user_name = ? AND repository_name = ?", userName, repositoryName).
		Order("page_name").
		Find(&pages).Error
	return pages, err
}

// GetWikiPage returns a specific wiki page
func (s *WikiService) GetWikiPage(userName, repositoryName, pageName string) (*WikiPage, error) {
	page := &WikiPage{}
	err := s.db.
		Where("user_name = ? AND repository_name = ? AND page_name = ?", userName, repositoryName, pageName).
		First(page).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return page, nil
}

// CreateWikiPage creates a new wiki page
func (s *WikiService) CreateWikiPage(userName, repositoryName, pageName, title, content, author string) (*WikiPage, error) {
	now := time.Now()
	page := &WikiPage{
		UserName:       userName,
		RepositoryName: repositoryName,
		PageName:       pageName,
		Title:          title,
		Content:        content,
		Author:         author,
		RegisteredDate: now,
		UpdatedDate:    now,
	}

	if err := s.db.Create(page).Error; err != nil {
		return nil, err
	}

	return page, nil
}

// UpdateWikiPage updates an existing wiki page
func (s *WikiService) UpdateWikiPage(userName, repositoryName, pageName, title, content, author string) (*WikiPage, error) {
	now := time.Now()
	err := s.db.
		Model(&WikiPage{}).
		Where("user_name = ? AND repository_name = ? AND page_name = ?", userName, repositoryName, pageName).
		Updates(map[string]interface{}{
			"title":        title,
			"content":      content,
			"author":       author,
			"updated_date": now,
		}).Error
	if err != nil {
		return nil, err
	}

	return &WikiPage{
		UserName:       userName,
		RepositoryName: repositoryName,
		PageName:       pageName,
		Title:          title,
		Content:        content,
		Author:         author,
		UpdatedDate:    now,
	}, nil
}

// DeleteWikiPage deletes a wiki page
func (s *WikiService) DeleteWikiPage(userName, repositoryName, pageName string) error {
	return s.db.
		Where("user_name = ? AND repository_name = ? AND page_name = ?", userName, repositoryName, pageName).
		Delete(&WikiPage{}).Error
}
