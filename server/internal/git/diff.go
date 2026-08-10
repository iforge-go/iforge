package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

// ForkRepository forks a repository by copying all Git data
func (s *Client) ForkRepository(originalOwner, originalRepo, newOwner, newRepo string) error {
	originalPath := s.RepositoryPath(originalOwner, originalRepo)
	newPath := s.RepositoryPath(newOwner, newRepo)

	// Verify original repository exists
	if _, err := os.Stat(originalPath); os.IsNotExist(err) {
		return fmt.Errorf("original repository path does not exist: %s", originalPath)
	}

	// Create new repository directory
	if err := os.MkdirAll(filepath.Dir(newPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Clone as bare repository using git clone --bare
	// This is cross-platform and handles git internals properly
	cmd := exec.Command("git", "clone", "--bare", originalPath, newPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to fork repository: %w (output: %s)", err, string(output))
	}

	return nil
}

// GetForkStatus checks how many commits the fork's default branch is behind/ahead
// of the parent repository's default branch.
func (s *Client) GetForkStatus(forkOwner, forkRepo, forkDefaultBranch string, parentOwner, parentRepo, parentDefaultBranch string) (int, int, error) {
	// Fetch parent's default branch into fork repo as a temporary ref
	tmpRef := "refs/sync-fork/tmp"
	err := s.FetchBranch(forkOwner, forkRepo, parentOwner, parentRepo, parentDefaultBranch, tmpRef)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to fetch parent branch: %w", err)
	}

	// Compare fork's default branch with the fetched parent branch
	repoPath := s.RepositoryPath(forkOwner, forkRepo)
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to open fork repository: %w", err)
	}

	// Ensure the temporary ref is cleaned up on every return path, including
	// early returns below. A leftover tmpRef would cause the next fetch to be
	// rejected as non-fast-forward (FetchBranch now uses a forced refspec as
	// a belt-and-suspenders fix, but cleaning up is still the right thing).
	defer func() {
		_ = r.Storer.RemoveReference(plumbing.ReferenceName(tmpRef))
	}()

	forkHash, err := s.resolveRef(r, forkDefaultBranch)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to resolve fork branch: %w", err)
	}

	parentHash, err := s.resolveRef(r, tmpRef)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to resolve fetched parent ref: %w", err)
	}

	if forkHash == parentHash {
		return 0, 0, nil
	}

	mergeBase, err := s.findMergeBase(r, forkHash, parentHash)
	if err != nil {
		return 0, 0, nil
	}

	ahead, err := s.countCommitsFrom(r, mergeBase, forkHash)
	if err != nil {
		return 0, 0, err
	}

	behind, err := s.countCommitsFrom(r, mergeBase, parentHash)
	if err != nil {
		return 0, 0, err
	}

	return ahead, behind, nil
}

// SyncFork updates the fork's default branch to match the parent repository's default branch.
// It fetches the parent's default branch and updates the fork's default branch reference.
func (s *Client) SyncFork(forkOwner, forkRepo, forkDefaultBranch string, parentOwner, parentRepo, parentDefaultBranch string) error {
	// Fetch parent's default branch into fork repo
	tmpRef := "refs/sync-fork/tmp"
	err := s.FetchBranch(forkOwner, forkRepo, parentOwner, parentRepo, parentDefaultBranch, tmpRef)
	if err != nil {
		return fmt.Errorf("failed to fetch parent branch: %w", err)
	}

	// Open fork repo
	repoPath := s.RepositoryPath(forkOwner, forkRepo)
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open fork repository: %w", err)
	}

	// Ensure the temporary ref is cleaned up on every return path.
	defer func() {
		_ = r.Storer.RemoveReference(plumbing.ReferenceName(tmpRef))
	}()

	// Get the fetched parent commit hash
	parentHash, err := s.resolveRef(r, tmpRef)
	if err != nil {
		return fmt.Errorf("failed to resolve fetched parent ref: %w", err)
	}

	// Update fork's default branch to point to parent's latest commit
	// This is a fast-forward update - the fork's default branch is moved to match parent
	newRef := plumbing.NewHashReference(plumbing.NewBranchReferenceName(forkDefaultBranch), parentHash)
	if err := r.Storer.SetReference(newRef); err != nil {
		return fmt.Errorf("failed to update fork branch: %w", err)
	}

	// Invalidate all caches for the fork repository (per-repo, not global)
	InvalidateRepoCaches(forkOwner, forkRepo)

	return nil
}

// Compare compares base and head refs, returning commits and file changes between them.
// base and head can be commit hashes, branch names, or tag names.
func (s *Client) Compare(owner, repo, base, head string) (*CompareResult, error) {
	repoPath := s.RepositoryPath(owner, repo)

	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	// Resolve base ref to commit hash
	baseHash, err := s.resolveRef(r, base)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve base ref %q: %w", base, err)
	}

	// Resolve head ref to commit hash
	headHash, err := s.resolveRef(r, head)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve head ref %q: %w", head, err)
	}

	result := &CompareResult{
		BaseCommit: baseHash.String(),
		HeadCommit: headHash.String(),
	}

	// If base and head are the same, return empty result (mergeable)
	if baseHash == headHash {
		result.Mergeable = true
		return result, nil
	}

	baseCommit, err := r.CommitObject(baseHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get base commit: %w", err)
	}
	headCommit, err := r.CommitObject(headHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get head commit: %w", err)
	}

	// Get commits between base..head
	// Use Log to traverse from head to base, collecting commits
	iter, err := r.Log(&git.LogOptions{From: headHash})
	if err != nil {
		return nil, fmt.Errorf("failed to get log: %w", err)
	}

	commits := []CommitInfo{}
	err = iter.ForEach(func(c *object.Commit) error {
		// Stop when reaching base commit (excluding base)
		if c.Hash == baseHash {
			return storer.ErrStop
		}
		commits = append(commits, CommitInfo{
			ID:        c.Hash.String(),
			Message:   c.Message,
			Author:    c.Author.Name,
			Email:     c.Author.Email,
			Timestamp: c.Author.When,
		})
		return nil
	})
	if err != nil && err != storer.ErrStop {
		return nil, fmt.Errorf("failed to iterate commits: %w", err)
	}
	result.Commits = commits

	// Reuse getCommitFileChanges to get full file changes including Patch field
	// (getCommitFileChanges fills the Patch string via object.DiffTree + change.Patch())
	fileChanges, err := s.getCommitFileChanges(headCommit, baseCommit)
	if err != nil {
		return nil, fmt.Errorf("failed to get file changes: %w", err)
	}
	result.FileChanges = fileChanges

	// Check if auto-merge is possible
	result.Mergeable = s.checkMergeable(r, baseHash, headHash)

	return result, nil
}

// checkMergeable checks whether head can be auto-merged into base.
// Simple implementation: checks if base is an ancestor of head (fast-forward case).
func (s *Client) checkMergeable(r *git.Repository, baseHash, headHash plumbing.Hash) bool {
	// If base is an ancestor of head, fast-forward merge is possible
	iter, err := r.Log(&git.LogOptions{From: headHash})
	if err != nil {
		return false
	}

	var isAncestor bool
	_ = iter.ForEach(func(c *object.Commit) error {
		if c.Hash == baseHash {
			isAncestor = true
			return storer.ErrStop
		}
		return nil
	})

	return isAncestor
}

// GetAheadBehind calculates the ahead/behind commit count of branch relative to base.
func (s *Client) GetAheadBehind(owner, repo, branch, base string) (*AheadBehind, error) {
	repoPath := s.RepositoryPath(owner, repo)

	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	branchHash, err := s.resolveRef(r, branch)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve branch %q: %w", branch, err)
	}

	baseHash, err := s.resolveRef(r, base)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve base %q: %w", base, err)
	}

	if branchHash == baseHash {
		return &AheadBehind{}, nil
	}

	mergeBase, err := s.findMergeBase(r, baseHash, branchHash)
	if err != nil {
		// No common ancestor (unrelated histories) -- not an error, just nothing ahead/behind
		return &AheadBehind{Ahead: 0, Behind: 0}, nil
	}

	ahead, err := s.countCommitsFrom(r, mergeBase, branchHash)
	if err != nil {
		return nil, err
	}

	behind, err := s.countCommitsFrom(r, mergeBase, baseHash)
	if err != nil {
		return nil, err
	}

	return &AheadBehind{Ahead: ahead, Behind: behind}, nil
}

// findMergeBase finds the most recent common ancestor of two commits.
func (s *Client) findMergeBase(r *git.Repository, a, b plumbing.Hash) (plumbing.Hash, error) {
	aAncestors := make(map[plumbing.Hash]bool)

	aIter, err := r.Log(&git.LogOptions{From: a})
	if err != nil {
		return plumbing.ZeroHash, err
	}
	_ = aIter.ForEach(func(c *object.Commit) error {
		aAncestors[c.Hash] = true
		return nil
	})

	bIter, err := r.Log(&git.LogOptions{From: b})
	if err != nil {
		return plumbing.ZeroHash, err
	}

	var result plumbing.Hash
	_ = bIter.ForEach(func(c *object.Commit) error {
		if aAncestors[c.Hash] {
			result = c.Hash
			return storer.ErrStop
		}
		return nil
	})

	if result.IsZero() {
		return plumbing.ZeroHash, fmt.Errorf("no merge base found")
	}
	return result, nil
}

// countCommitsFrom counts the number of commits from base to head (excluding base)
func (s *Client) countCommitsFrom(r *git.Repository, base, head plumbing.Hash) (int, error) {
	iter, err := r.Log(&git.LogOptions{From: head})
	if err != nil {
		return 0, fmt.Errorf("failed to get log: %w", err)
	}

	count := 0
	err = iter.ForEach(func(c *object.Commit) error {
		if c.Hash == base {
			return storer.ErrStop
		}
		count++
		return nil
	})
	if err != nil && err != storer.ErrStop {
		return 0, fmt.Errorf("failed to iterate commits: %w", err)
	}

	return count, nil
}
