package model

import "time"

// WikiPage represents a wiki page
type WikiPage struct {
	UserName       string    `gorm:"primaryKey;column:user_name" json:"userName"`
	RepositoryName string    `gorm:"primaryKey;column:repository_name" json:"repositoryName"`
	PageName       string    `gorm:"primaryKey;column:page_name" json:"pageName"`
	Title          string    `gorm:"column:title" json:"title"`
	Content        string    `gorm:"column:content" json:"content"`
	Author         string    `gorm:"column:author" json:"author"`
	RegisteredDate time.Time `gorm:"column:registered_date" json:"registeredDate"`
	UpdatedDate    time.Time `gorm:"column:updated_date" json:"updatedDate"`
}

func (WikiPage) TableName() string { return "wiki_page" }
