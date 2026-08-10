package git

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

// MergeBase returns the best common ancestor commit hash of the two given commits.
// Mimics `git merge-base <sha1> <sha2>`. Used to find the divergence point
// between a task branch and the default branch, so we can list only the
// commits unique to the task branch.
func (s *Client) MergeBase(owner, repo, sha1, sha2 string) (string, error) {
	repoPath := s.RepositoryPath(owner, repo)
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	c1, err := r.CommitObject(plumbing.NewHash(sha1))
	if err != nil {
		return "", fmt.Errorf("failed to get commit %s: %w", sha1, err)
	}
	c2, err := r.CommitObject(plumbing.NewHash(sha2))
	if err != nil {
		return "", fmt.Errorf("failed to get commit %s: %w", sha2, err)
	}

	bases, err := c1.MergeBase(c2)
	if err != nil {
		return "", err
	}
	if len(bases) == 0 {
		return "", fmt.Errorf("no merge base found between %s and %s", sha1, sha2)
	}
	return bases[0].Hash.String(), nil
}

// ListCommitsBetween lists commits between oldSHA and newSHA (no file changes)
func (s *Client) ListCommitsBetween(owner, repo, oldSHA, newSHA string, limit int) ([]PushCommitInfo, error) {
	if limit <= 0 {
		limit = 50
	}

	repoPath := s.RepositoryPath(owner, repo)
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	newHash := plumbing.NewHash(newSHA)
	oldHash := plumbing.NewHash(oldSHA)
	isNewBranch := oldSHA == "0000000000000000000000000000000000000000"
	isDeleted := newSHA == "0000000000000000000000000000000000000000"

	if isDeleted {
		return []PushCommitInfo{}, nil
	}

	commits := []PushCommitInfo{}
	iter, err := r.Log(&git.LogOptions{From: newHash})
	if err != nil {
		return nil, fmt.Errorf("failed to get log: %w", err)
	}

	count := 0
	err = iter.ForEach(func(c *object.Commit) error {
		if count >= limit {
			return storer.ErrStop
		}
		if !isNewBranch && c.Hash == oldHash {
			return storer.ErrStop
		}
		commits = append(commits, PushCommitInfo{
			ID:      c.Hash.String(),
			Message: c.Message,
			Author:  c.Author.Name,
			Email:   c.Author.Email,
			Time:    c.Author.When,
		})
		count++
		return nil
	})
	if err != nil && err != storer.ErrStop {
		return nil, fmt.Errorf("failed to iterate commits: %w", err)
	}

	return commits, nil
}

// ListCommits lists commits in the repository
func (s *Client) ListCommits(owner, repo, ref string, limit int) ([]CommitInfo, error) {
	// Check cache first
	cacheKey := CommitListCacheKey(owner, repo, ref, limit)
	if cached, ok := CommitListCache.Get(cacheKey); ok {
		return cached.([]CommitInfo), nil
	}

	repoPath := s.RepositoryPath(owner, repo)

	// Open repository
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	// Get reference
	hash, err := s.resolveReference(r, ref)
	if err != nil {
		return nil, err
	}

	// Get commit iterator
	commitIter, err := r.Log(&git.LogOptions{From: hash})
	if err != nil {
		return nil, fmt.Errorf("failed to get log: %w", err)
	}

	commits := []CommitInfo{}
	count := 0

	err = commitIter.ForEach(func(c *object.Commit) error {
		if limit > 0 && count >= limit {
			return nil
		}

		commits = append(commits, CommitInfo{
			ID:        c.Hash.String(),
			Message:   c.Message,
			Author:    c.Author.Name,
			Email:     c.Author.Email,
			Timestamp: c.Author.When,
		})

		count++
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to iterate commits: %w", err)
	}

	// Sort by author timestamp descending (newest first)
	sort.Slice(commits, func(i, j int) bool {
		return commits[i].Timestamp.After(commits[j].Timestamp)
	})

	// Cache the result
	CommitListCache.Set(cacheKey, commits)

	return commits, nil
}

// GetCommit gets a specific commit by hash
func (s *Client) GetCommit(owner, repo, commitHash string) (*CommitInfo, error) {
	repoPath := s.RepositoryPath(owner, repo)

	// Open repository
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	// Get commit
	hash := plumbing.NewHash(commitHash)
	commit, err := r.CommitObject(hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	// Get parent commit for diff
	var parentCommit *object.Commit
	if commit.NumParents() > 0 {
		parentCommit, err = commit.Parent(0)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent commit: %w", err)
		}
	}

	// Get file changes
	files, err := s.getCommitFileChanges(commit, parentCommit)
	if err != nil {
		return nil, fmt.Errorf("failed to get file changes: %w", err)
	}

	return &CommitInfo{
		ID:        commit.Hash.String(),
		Message:   commit.Message,
		Author:    commit.Author.Name,
		Email:     commit.Author.Email,
		Timestamp: commit.Author.When,
		Files:     files,
	}, nil
}

// getCommitFileChanges gets the file changes for a commit
func (s *Client) getCommitFileChanges(commit, parentCommit *object.Commit) ([]FileChange, error) {
	files := []FileChange{}

	// Get commit tree
	commitTree, err := commit.Tree()
	if err != nil {
		return nil, err
	}

	// Get parent tree if exists
	var parentTree *object.Tree
	if parentCommit != nil {
		parentTree, err = parentCommit.Tree()
		if err != nil {
			return nil, err
		}
	}

	// Get changes between trees
	changes, err := object.DiffTree(parentTree, commitTree)
	if err != nil {
		return nil, err
	}

	// Process each change
	for _, change := range changes {
		fileChange := FileChange{
			Filename: change.To.Name,
		}

		// Determine status
		if change.To.Name == "" {
			fileChange.Status = "deleted"
			fileChange.Filename = change.From.Name
		} else if change.From.Name == "" {
			fileChange.Status = "added"
		} else {
			fileChange.Status = "modified"
		}

		// Get patch content
		patch, err := change.Patch()
		if err == nil && patch != nil {
			fileChange.Patch = patch.String()

			// Count additions and deletions
			for _, fileStat := range patch.FilePatches() {
				for _, chunk := range fileStat.Chunks() {
					content := chunk.Content()
					lines := len(strings.Split(content, "\n")) - 1
					if lines < 0 {
						lines = 0
					}

					switch chunk.Type() {
					case 1: // Add
						fileChange.Additions += lines
					case 2: // Delete
						fileChange.Deletions += lines
					}
				}
			}
		}

		files = append(files, fileChange)
	}

	return files, nil
}

// GetCommitDiff gets the diff between two commits
func (s *Client) GetCommitDiff(owner, repo, fromCommit, toCommit string) ([]FileChange, error) {
	repoPath := s.RepositoryPath(owner, repo)

	// Open repository
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	// Get from commit
	fromHash := plumbing.NewHash(fromCommit)
	fromCommitObj, err := r.CommitObject(fromHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get from commit: %w", err)
	}

	// Get to commit
	toHash := plumbing.NewHash(toCommit)
	toCommitObj, err := r.CommitObject(toHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get to commit: %w", err)
	}

	// Get file changes
	return s.getCommitFileChanges(toCommitObj, fromCommitObj)
}

// GeneratePatch generates a patch file for a commit
func (s *Client) GeneratePatch(owner, repo, commitID string) ([]byte, error) {
	repoPath := s.RepositoryPath(owner, repo)

	// Open repository
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	// Get commit
	hash := plumbing.NewHash(commitID)
	commit, err := r.CommitObject(hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	// Get parent commit
	var parentCommit *object.Commit
	if commit.NumParents() > 0 {
		parentCommit, err = commit.Parent(0)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent commit: %w", err)
		}
	}

	// Generate patch
	var patch bytes.Buffer

	// Write commit header
	patch.WriteString(fmt.Sprintf("From %s Mon Sep 17 00:00:00 2001\n", hash.String()))
	patch.WriteString(fmt.Sprintf("From: %s <%s>\n", commit.Author.Name, commit.Author.Email))
	patch.WriteString(fmt.Sprintf("Date: %s\n", commit.Author.When.Format("Mon, 2 Jan 2006 15:04:05 -0700")))
	patch.WriteString(fmt.Sprintf("Subject: [PATCH] %s\n", strings.Split(commit.Message, "\n")[0]))
	patch.WriteString("\n")
	patch.WriteString("---\n")

	// Get file changes
	if parentCommit != nil {
		parentTree, err := parentCommit.Tree()
		if err != nil {
			return nil, fmt.Errorf("failed to get parent tree: %w", err)
		}

		commitTree, err := commit.Tree()
		if err != nil {
			return nil, fmt.Errorf("failed to get commit tree: %w", err)
		}

		changes, err := object.DiffTree(parentTree, commitTree)
		if err != nil {
			return nil, fmt.Errorf("failed to diff trees: %w", err)
		}

		for _, change := range changes {
			action, err := change.Action()
			if err != nil {
				return nil, fmt.Errorf("failed to get action: %w", err)
			}
			actionStr := action.String()
			patch.WriteString(fmt.Sprintf("\ndiff --git %s %s\n", change.From.Name, change.To.Name))
			patch.WriteString(fmt.Sprintf("%s %s\n", actionStr, change.To.Name))

			// Get file content
			if change.To.Name != "" {
				toFile, err := commitTree.File(change.To.Name)
				if err == nil {
					toContent, err := toFile.Contents()
					if err == nil {
						patch.WriteString(fmt.Sprintf("--- /dev/null\n+++ b/%s\n", change.To.Name))
						// Write unified diff format
						lines := strings.Split(toContent, "\n")
						patch.WriteString(fmt.Sprintf("@@ -0,0 +1,%d @@\n", len(lines)))
						for _, line := range lines {
							patch.WriteString(fmt.Sprintf("+%s\n", line))
						}
					}
				}
			}
		}
	}

	patch.WriteString("--\n")
	patch.WriteString("iForge\n")

	return patch.Bytes(), nil
}
