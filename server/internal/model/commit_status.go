package model

import "time"

// CommitStatus represents a commit status
type CommitStatus struct {
	UserName       string    `gorm:"primaryKey;column:user_name" json:"userName"`
	RepositoryName string    `gorm:"primaryKey;column:repository_name" json:"repositoryName"`
	CommitID       string    `gorm:"primaryKey;column:commit_id" json:"commitId"`
	Context        string    `gorm:"primaryKey;column:context" json:"context"`
	State          string    `gorm:"column:state" json:"state"`
	TargetURL      *string   `gorm:"column:target_url" json:"targetUrl"`
	Description    *string   `gorm:"column:description" json:"description"`
	UpdatedDate    time.Time `gorm:"column:updated_date" json:"updatedDate"`
	Creator        string    `gorm:"column:creator" json:"creator"`
}

func (CommitStatus) TableName() string { return "commit_status" }

// CombinedStatus represents the combined status for a commit
type CombinedStatus struct {
	UserName       string          `json:"userName"`
	RepositoryName string          `json:"repositoryName"`
	CommitID       string          `json:"commitId"`
	State          string          `json:"state"`
	Statuses       []*CommitStatus `json:"statuses"`
}
