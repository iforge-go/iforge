package model

import "time"

// LFSObject represents a Git LFS object stored for a repository
type LFSObject struct {
	UserName       string    `gorm:"primaryKey;column:user_name" json:"userName"`
	RepositoryName string    `gorm:"primaryKey;column:repository_name" json:"repositoryName"`
	OID            string    `gorm:"primaryKey;column:oid" json:"oid"`
	Size           int64     `gorm:"column:size" json:"size"`
	CreatedAt      time.Time `gorm:"column:created_at" json:"createdAt"`
}

func (LFSObject) TableName() string { return "lfs_object" }
