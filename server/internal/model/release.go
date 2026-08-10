package model

import "time"

// ReleaseTag represents a release
type ReleaseTag struct {
	UserName       string    `gorm:"primaryKey;column:user_name" json:"userName"`
	RepositoryName string    `gorm:"primaryKey;column:repository_name" json:"repositoryName"`
	Tag            string    `gorm:"primaryKey;column:tag" json:"tag"`
	Name           string    `gorm:"column:name" json:"name"`
	Content        *string   `gorm:"column:content" json:"content"`
	RegisteredDate time.Time `gorm:"column:registered_date" json:"registeredDate"`
	Author         string    `gorm:"column:author" json:"author"`
}

func (ReleaseTag) TableName() string { return "release_tag" }

// ReleaseAsset represents a release asset
type ReleaseAsset struct {
	// Composite index (user_name, repository_name, tag) covers per-release asset queries
	UserName       string    `gorm:"column:user_name;index:idx_asset_repo_tag,priority:1" json:"userName"`
	RepositoryName string    `gorm:"column:repository_name;index:idx_asset_repo_tag,priority:2" json:"repositoryName"`
	Tag            string    `gorm:"column:tag;index:idx_asset_repo_tag,priority:3" json:"tag"`
	AssetID        int       `gorm:"primaryKey;autoIncrement;column:asset_id" json:"assetId"`
	FileName       string    `gorm:"column:file_name" json:"fileName"`
	Label          *string   `gorm:"column:label" json:"label"`
	Size           int64     `gorm:"column:size" json:"size"`
	Uploader       string    `gorm:"column:uploader" json:"uploader"`
	RegisteredDate time.Time `gorm:"column:registered_date" json:"registeredDate"`
}

func (ReleaseAsset) TableName() string { return "release_asset" }
