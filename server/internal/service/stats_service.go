package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	gitsvc "iforge/iforge/internal/git"
	"iforge/iforge/internal/model"

	"gorm.io/gorm"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// statsCacheTTL controls cache validity for repository statistics. Stats data changes infrequently (only after push),
// so 5 minutes is sufficient to avoid redundant commit history calculations.
const statsCacheTTL = 5 * time.Minute

// Contributor represents a contributor with statistics
type Contributor struct {
	Name         string `json:"name"`
	Email        string `json:"email"`
	Commits      int    `json:"commits"`
	Additions    int    `json:"additions"`
	Deletions    int    `json:"deletions"`
	LastCommitAt string `json:"lastCommitAt"`
}

// CodeFrequency represents code change statistics
type CodeFrequency struct {
	Timestamp string `json:"timestamp"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

// RepositoryStats represents repository statistics
type RepositoryStats struct {
	Contributors  []Contributor   `json:"contributors"`
	CodeFrequency []CodeFrequency `json:"codeFrequency"`
	TotalCommits  int             `json:"totalCommits"`
	TotalFiles    int             `json:"totalFiles"`
	DefaultBranch string          `json:"defaultBranch"`
}

// StatsService handles repository statistics
type StatsService struct {
	gitClient *gitsvc.Client
	db        *gorm.DB
}

// NewStatsService creates a new StatsService
func NewStatsService(db *gorm.DB, gitClient *gitsvc.Client) *StatsService {
	return &StatsService{
		gitClient: gitClient,
		db:        db,
	}
}

// InvalidateStatsCache deletes statistics cache for a repository.
// Called after repository updates, deletions, or pushes to ensure fresh data on next query.
func (s *StatsService) InvalidateStatsCache(owner, repo string) {
	_ = s.db.Where("user_name = ? AND repository_name = ?", owner, repo).
		Delete(&model.RepositoryStatsCache{}).Error
}

// GetRepositoryStats retrieves statistics for a repository.
//
// Optimized: checks repository_stats_cache first, returns deserialized result if cache hits within statsCacheTTL (5 minutes).
// Otherwise falls back to git traversal and writes result to cache table after completion.
// Original implementation traversed entire commit history on each request, blocking for hundreds of ms to seconds on repos with >1000 commits.
func (s *StatsService) GetRepositoryStats(owner, repo string) (*RepositoryStats, error) {
	// 1. Try reading from cache
	var cached model.RepositoryStatsCache
	err := s.db.Where("user_name = ? AND repository_name = ?", owner, repo).
		First(&cached).Error
	if err == nil && time.Since(cached.UpdatedDate) < statsCacheTTL {
		// Cache hit and not expired
		var stats RepositoryStats
		if json.Unmarshal([]byte(cached.StatsJSON), &stats) == nil {
			return &stats, nil
		}
		// Deserialization failed: clear corrupt cache and recalculate
		_ = s.db.Delete(&cached).Error
	} else if err != nil && err != gorm.ErrRecordNotFound {
		// Cache query error (not not found): don't block main flow, continue with original calculation
	}

	// 2. Cache miss, call original calculation logic
	stats, err := s.computeRepositoryStats(owner, repo)
	if err != nil {
		return nil, err
	}

	// 3. Async write to cache (non-blocking; failures ignored, next query recalculates).
	// Uses context with timeout to prevent goroutine from accessing closed DB on shutdown.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		payload, mErr := json.Marshal(stats)
		if mErr != nil {
			return
		}
		cacheRow := model.RepositoryStatsCache{
			UserName:       owner,
			RepositoryName: repo,
			StatsJSON:      string(payload),
			UpdatedDate:    time.Now(),
		}
		// UPSERT: overwrite if exists (GORM Save upserts by primary key for composite-key models)
		_ = s.db.WithContext(ctx).Save(&cacheRow).Error
	}()

	return stats, nil
}

// computeRepositoryStats is the original GetRepositoryStats implementation, traversing git history.
func (s *StatsService) computeRepositoryStats(owner, repo string) (*RepositoryStats, error) {
	// Get repository info
	var repoModel model.Repository
	err := s.db.
		Where("user_name = ? AND repository_name = ?", owner, repo).
		First(&repoModel).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}
	defaultBranch := repoModel.DefaultBranch

	// Open git repository
	gitRepo, err := s.gitClient.OpenRepository(owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to open git repository: %w", err)
	}

	// Get HEAD reference
	ref, err := gitRepo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Get commit iterator
	commitIter, err := gitRepo.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return nil, fmt.Errorf("failed to get commit log: %w", err)
	}

	// Collect contributor statistics
	contributorMap := make(map[string]*Contributor)
	// Accumulate codeFrequency using map and sort at end to avoid O(N*W) linear search in original implementation
	codeFreqMap := make(map[string]*CodeFrequency)
	var weekFreqOrder []string
	totalCommits := 0

	err = commitIter.ForEach(func(c *object.Commit) error {
		totalCommits++

		// Contributor stats
		key := c.Author.Email
		if _, exists := contributorMap[key]; !exists {
			contributorMap[key] = &Contributor{
				Name:  c.Author.Name,
				Email: c.Author.Email,
			}
		}
		contributorMap[key].Commits++
		contributorMap[key].LastCommitAt = c.Author.When.Format(time.RFC3339)

		// Get file changes for this commit (calculate once)
		var commitAdditions, commitDeletions int
		if c.NumParents() > 0 {
			parent, err := c.Parent(0)
			if err == nil {
				parentTree, err := parent.Tree()
				if err == nil {
					currentTree, err := c.Tree()
					if err == nil {
						changes, err := object.DiffTree(parentTree, currentTree)
						if err == nil {
							for _, change := range changes {
								patch, err := change.Patch()
								if err == nil {
									for _, fileStat := range patch.Stats() {
										commitAdditions += fileStat.Addition
										commitDeletions += fileStat.Deletion
									}
								}
							}
						}
					}
				}
			}
		}

		// Update contributor stats
		contributorMap[key].Additions += commitAdditions
		contributorMap[key].Deletions += commitDeletions

		// Code frequency (group by week), using map instead of original linear search
		weekStart := c.Author.When.AddDate(0, 0, -int(c.Author.When.Weekday()))
		weekKey := weekStart.Format("2006-01-02")
		if entry, ok := codeFreqMap[weekKey]; ok {
			entry.Additions += commitAdditions
			entry.Deletions += commitDeletions
		} else {
			codeFreqMap[weekKey] = &CodeFrequency{
				Timestamp: weekKey,
				Additions: commitAdditions,
				Deletions: commitDeletions,
			}
			weekFreqOrder = append(weekFreqOrder, weekKey)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to iterate commits: %w", err)
	}

	// Convert map to slice
	contributors := make([]Contributor, 0, len(contributorMap))
	for _, c := range contributorMap {
		contributors = append(contributors, *c)
	}

	// Sort by commits descending
	sort.Slice(contributors, func(i, j int) bool {
		return contributors[i].Commits > contributors[j].Commits
	})

	// Sort code frequency by timestamp (output from map in order)
	codeFrequency := make([]CodeFrequency, 0, len(weekFreqOrder))
	for _, wk := range weekFreqOrder {
		if entry, ok := codeFreqMap[wk]; ok {
			codeFrequency = append(codeFrequency, *entry)
		}
	}
	sort.Slice(codeFrequency, func(i, j int) bool {
		return codeFrequency[i].Timestamp < codeFrequency[j].Timestamp
	})

	// Count total files
	totalFiles := 0
	headCommit, err := gitRepo.CommitObject(ref.Hash())
	if err == nil {
		tree, err := headCommit.Tree()
		if err == nil {
			files := tree.Files()
			files.ForEach(func(*object.File) error {
				totalFiles++
				return nil
			})
		}
	}

	return &RepositoryStats{
		Contributors:  contributors,
		CodeFrequency: codeFrequency,
		TotalCommits:  totalCommits,
		TotalFiles:    totalFiles,
		DefaultBranch: defaultBranch,
	}, nil
}
