package service

import (
	"sort"
	"time"

	gitsvc "iforge/iforge/internal/git"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// ContributorStats represents statistics for a contributor
type ContributorStats struct {
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Commits      int       `json:"commits"`
	Additions    int       `json:"additions"`
	Deletions    int       `json:"deletions"`
	FirstCommit  time.Time `json:"firstCommit"`
	LastCommit   time.Time `json:"lastCommit"`
	FilesChanged int       `json:"filesChanged"`
}

// ContributorService handles contributor statistics
type ContributorService struct {
	gitClient *gitsvc.Client
}

// NewContributorService creates a new ContributorService
func NewContributorService(gitClient *gitsvc.Client) *ContributorService {
	return &ContributorService{
		gitClient: gitClient,
	}
}

// GetContributors retrieves contributor statistics for a repository
func (s *ContributorService) GetContributors(owner, repo string) ([]ContributorStats, error) {
	// Open git repository
	gitRepo, err := s.gitClient.OpenRepository(owner, repo)
	if err != nil {
		return nil, err
	}

	// Get HEAD reference
	ref, err := gitRepo.Head()
	if err != nil {
		return nil, err
	}

	// Get commit iterator
	commitIter, err := gitRepo.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return nil, err
	}

	// Collect contributor statistics
	contributorMap := make(map[string]*ContributorStats)
	filesChangedMap := make(map[string]map[string]bool) // email -> set of files

	err = commitIter.ForEach(func(c *object.Commit) error {
		email := c.Author.Email
		name := c.Author.Name

		// Initialize contributor if not exists
		if _, exists := contributorMap[email]; !exists {
			contributorMap[email] = &ContributorStats{
				Name:        name,
				Email:       email,
				FirstCommit: c.Author.When,
				LastCommit:  c.Author.When,
			}
			filesChangedMap[email] = make(map[string]bool)
		}

		contributor := contributorMap[email]
		contributor.Commits++

		// Update first/last commit times
		if c.Author.When.Before(contributor.FirstCommit) {
			contributor.FirstCommit = c.Author.When
		}
		if c.Author.When.After(contributor.LastCommit) {
			contributor.LastCommit = c.Author.When
		}

		// Get file changes for this commit
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
								filesChangedMap[email][change.From.Name] = true

								patch, err := change.Patch()
								if err == nil {
									for _, fileStat := range patch.Stats() {
										contributor.Additions += fileStat.Addition
										contributor.Deletions += fileStat.Deletion
									}
								}
							}
						}
					}
				}
			}
		} else {
			// First commit - all files are additions
			tree, err := c.Tree()
			if err == nil {
				files := tree.Files()
				files.ForEach(func(f *object.File) error {
					binary, _ := f.IsBinary()
					if !binary {
						content, err := f.Contents()
						if err == nil {
							lines := len([]rune(content))
							contributor.Additions += lines
							filesChangedMap[email][f.Name] = true
						}
					}
					return nil
				})
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Convert map to slice and calculate files changed
	contributors := make([]ContributorStats, 0, len(contributorMap))
	for email, stats := range contributorMap {
		stats.FilesChanged = len(filesChangedMap[email])
		contributors = append(contributors, *stats)
	}

	// Sort by commits descending
	sort.Slice(contributors, func(i, j int) bool {
		return contributors[i].Commits > contributors[j].Commits
	})

	return contributors, nil
}
