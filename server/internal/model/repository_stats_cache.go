package model

import "time"

// RepositoryStatsCache caches the result of GetRepositoryStats.
//
// Background: the original implementation walked the entire git commit history on every
// request; once a repository grew beyond ~1000 commits this caused hundreds of milliseconds
// to seconds of CPU blocking. Stats data changes very infrequently (only after a push),
// so it is a good fit for on-disk caching.
//
// Invalidation strategy (dual safeguard):
//  1. TTL: if updated_date is older than statsCacheTTL (5 minutes), the row is considered stale
//  2. Active invalidation: on push/update/delete of a repository the service layer deletes the row
//
// Primary key is (user_name, repository_name); each repository has at most one cached row.
type RepositoryStatsCache struct {
	UserName       string    `gorm:"primaryKey;column:user_name" json:"userName"`
	RepositoryName string    `gorm:"primaryKey;column:repository_name" json:"repositoryName"`
	// StatsJSON is the JSON serialization of RepositoryStats
	StatsJSON   string    `gorm:"column:stats_json;type:text" json:"statsJson"`
	UpdatedDate time.Time `gorm:"column:updated_date;index:idx_stats_cache_updated" json:"updatedDate"`
}

func (RepositoryStatsCache) TableName() string { return "repository_stats_cache" }
