package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// ListBranches lists all branches in a repository.
// defaultBranch is the repository's configured default branch (e.g. from the
// database). If empty, the function falls back to HEAD.
func (s *Client) ListBranches(owner, repo, defaultBranch string) ([]BranchInfo, error) {
	// Check cache first
	cacheKey := BranchListCacheKey(owner, repo)
	if cached, ok := BranchListCache.Get(cacheKey); ok {
		return cached.([]BranchInfo), nil
	}

	repoPath := s.RepositoryPath(owner, repo)

	// Open repository
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	// Fall back to HEAD if default branch not provided
	if defaultBranch == "" {
		headRef, err := r.Head()
		if err == nil {
			defaultBranch = headRef.Name().Short()
			GetLogger().Debug("Using default branch from HEAD", map[string]interface{}{"branch": defaultBranch})
		} else {
			GetLogger().Warn("Failed to get HEAD", map[string]interface{}{"error": err})
		}
	}

	// Get branch iterator
	branchIter, err := r.Branches()
	if err != nil {
		return nil, fmt.Errorf("failed to get branches: %w", err)
	}

	branches := []BranchInfo{}
	err = branchIter.ForEach(func(ref *plumbing.Reference) error {
		branchName := ref.Name().Short()
		branches = append(branches, BranchInfo{
			Name:      branchName,
			CommitId:  ref.Hash().String(),
			IsDefault: branchName == defaultBranch,
		})
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to iterate branches: %w", err)
	}

	// If default branch doesn't exist in branch list, use the first branch as default
	if defaultBranch != "" {
		found := false
		for _, b := range branches {
			if b.Name == defaultBranch {
				found = true
				break
			}
		}
		if !found && len(branches) > 0 {
			branches[0].IsDefault = true
		}
	} else if len(branches) > 0 {
		branches[0].IsDefault = true
	}

	// Store in cache
	BranchListCache.Set(cacheKey, branches)
	return branches, nil
}

// CreateBranch creates a new branch
func (s *Client) CreateBranch(owner, repo, branchName, fromBranch string) error {
	// Validate branch name to prevent command injection
	if err := IsValidRefName(branchName); err != nil {
		return fmt.Errorf("invalid branch name: %w", err)
	}
	if fromBranch != "" && fromBranch != "HEAD" {
		if err := IsValidRefName(fromBranch); err != nil {
			return fmt.Errorf("invalid source branch name: %w", err)
		}
	}

	repoPath := s.RepositoryPath(owner, repo)

	// Acquire repository-level lock for write operations
	lock := GetRepoLock(repoPath)
	lock.Lock()
	defer lock.Unlock()

	// Open repository
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	// Get source reference
	var sourceRef *plumbing.Reference
	if fromBranch == "HEAD" || fromBranch == "" {
		sourceRef, err = r.Head()
	} else {
		sourceRef, err = r.Reference(plumbing.NewBranchReferenceName(fromBranch), true)
	}
	if err != nil {
		return fmt.Errorf("failed to get source branch: %w", err)
	}

	// Create new branch reference
	newRef := plumbing.NewHashReference(plumbing.NewBranchReferenceName(branchName), sourceRef.Hash())
	if err := r.Storer.SetReference(newRef); err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	// Invalidate all caches for this repository (branch list + any related caches)
	InvalidateRepoCaches(owner, repo)

	return nil
}

// DeleteBranchListCache deletes the branch list cache for a repository
func (s *Client) DeleteBranchListCache(owner, repo string) {
	InvalidateRepoCaches(owner, repo)
}

// UpdateDefaultBranch updates the default branch (HEAD reference) in the git repository
func (s *Client) UpdateDefaultBranch(owner, repo, branchName string) error {
	repoPath := s.RepositoryPath(owner, repo)

	// Acquire repository-level lock for write operations
	lock := GetRepoLock(repoPath)
	lock.Lock()
	defer lock.Unlock()

	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	// Check if branch exists
	if _, err := r.Reference(plumbing.NewBranchReferenceName(branchName), true); err != nil {
		return fmt.Errorf("branch %s does not exist: %w", branchName, err)
	}

	// Update HEAD to point to the new default branch
	newHead := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName(branchName))
	if err := r.Storer.SetReference(newHead); err != nil {
		return fmt.Errorf("failed to update HEAD: %w", err)
	}

	// Invalidate all caches for this repository
	InvalidateRepoCaches(owner, repo)

	return nil
}

// DeleteBranch deletes a branch
func (s *Client) DeleteBranch(owner, repo, branchName string) error {
	// Validate branch name to prevent command injection
	if err := IsValidRefName(branchName); err != nil {
		return fmt.Errorf("invalid branch name: %w", err)
	}

	repoPath := s.RepositoryPath(owner, repo)

	// Acquire repository-level lock for write operations
	lock := GetRepoLock(repoPath)
	lock.Lock()
	defer lock.Unlock()

	// Open repository
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	// Check if trying to delete the default branch
	headRef, err := r.Head()
	if err == nil {
		if headRef.Name().Short() == branchName {
			return fmt.Errorf("cannot delete the default branch")
		}
	} else {
		// HEAD might not exist, try to resolve it as a symbolic ref
		ref, refErr := r.Reference(plumbing.HEAD, true)
		if refErr == nil && ref.Name().Short() == branchName {
			return fmt.Errorf("cannot delete the default branch")
		}
	}

	// Delete branch reference
	if err := r.Storer.RemoveReference(plumbing.NewBranchReferenceName(branchName)); err != nil {
		return fmt.Errorf("failed to delete branch: %w", err)
	}

	// Invalidate all caches for this repository
	InvalidateRepoCaches(owner, repo)

	return nil
}

// RenameBranch renames a branch from oldName to newName
func (s *Client) RenameBranch(owner, repo, oldName, newName string) error {
	// Validate branch names to prevent command injection
	if err := IsValidRefName(oldName); err != nil {
		return fmt.Errorf("invalid old branch name: %w", err)
	}
	if err := IsValidRefName(newName); err != nil {
		return fmt.Errorf("invalid new branch name: %w", err)
	}

	repoPath := s.RepositoryPath(owner, repo)

	// Acquire repository-level lock for write operations
	lock := GetRepoLock(repoPath)
	lock.Lock()
	defer lock.Unlock()

	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	// Check if old branch exists
	oldRef, err := r.Reference(plumbing.NewBranchReferenceName(oldName), true)
	if err != nil {
		return fmt.Errorf("branch %q not found", oldName)
	}

	// Check if new branch name already exists
	_, err = r.Reference(plumbing.NewBranchReferenceName(newName), true)
	if err == nil {
		return fmt.Errorf("branch %q already exists", newName)
	}

	// Create new branch reference pointing to same commit
	newRef := plumbing.NewHashReference(plumbing.NewBranchReferenceName(newName), oldRef.Hash())
	if err := r.Storer.SetReference(newRef); err != nil {
		return fmt.Errorf("failed to create branch %q: %w", newName, err)
	}

	// Update HEAD if renaming the default branch
	headRef, err := r.Head()
	if err == nil && headRef.Name().Short() == oldName {
		symbolicRef := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName(newName))
		if err := r.Storer.SetReference(symbolicRef); err != nil {
			return fmt.Errorf("failed to update HEAD: %w", err)
		}
	}

	// Remove old branch reference
	if err := r.Storer.RemoveReference(plumbing.NewBranchReferenceName(oldName)); err != nil {
		return fmt.Errorf("failed to remove old branch %q: %w", oldName, err)
	}

	// Invalidate all caches for this repository
	InvalidateRepoCaches(owner, repo)
	return nil
}

// CloneToWorkDir clones the bare repository into workDir and checks out the
// given commit SHA. Used by the CI/CD executor to provide a working copy
// of the repository at the exact commit being built.
func (s *Client) CloneToWorkDir(owner, repo, commitSHA, workDir string) error {
	repoPath := s.RepositoryPath(owner, repo)
	r, err := git.PlainClone(workDir, false, &git.CloneOptions{
		URL: repoPath,
	})
	if err != nil {
		return fmt.Errorf("clone %s/%s: %w", owner, repo, err)
	}
	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("worktree: %w", err)
	}
	if err := w.Checkout(&git.CheckoutOptions{
		Hash: plumbing.NewHash(commitSHA),
	}); err != nil {
		return fmt.Errorf("checkout %s: %w", commitSHA, err)
	}
	return nil
}

// GetBranchHead returns the commit SHA that the given branch points to.
// Returns os.ErrNotExist if the branch does not exist.
func (s *Client) GetBranchHead(owner, repo, branch string) (string, error) {
	r, err := s.OpenRepository(owner, repo)
	if err != nil {
		return "", err
	}
	ref, err := r.Reference(plumbing.ReferenceName("refs/heads/"+branch), true)
	if err != nil {
		return "", err
	}
	return ref.Hash().String(), nil
}

// FetchBranch fetches a branch from a source repository to a target reference in the target repository.
// This enables cross-repository operations by making objects from the source repo available in the target repo.
// Uses native git command for reliable bare repository fetch with refspec.
func (s *Client) FetchBranch(
	targetOwner, targetRepo string,
	sourceOwner, sourceRepo string,
	sourceBranch string,
	targetRef string,
) error {
	// Validate branch names to prevent command injection
	if err := IsValidRefName(sourceBranch); err != nil {
		return fmt.Errorf("invalid source branch name: %w", err)
	}
	// targetRef can be a full ref path like "refs/merge-requests/1/head", validate it
	if !strings.HasPrefix(targetRef, "refs/") {
		if err := IsValidRefName(targetRef); err != nil {
			return fmt.Errorf("invalid target ref: %w", err)
		}
	}

	targetPath := s.RepositoryPath(targetOwner, targetRepo)
	sourcePath := s.RepositoryPath(sourceOwner, sourceRepo)

	// Acquire repository-level lock for write operations (target repo)
	lock := GetRepoLock(targetPath)
	lock.Lock()
	defer lock.Unlock()

	// Verify source repository exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("source repository not found: %s/%s", sourceOwner, sourceRepo)
	}

	// Verify target repository exists
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return fmt.Errorf("target repository not found: %s/%s", targetOwner, targetRepo)
	}

	// Convert source path to absolute path to avoid issues when git fetch runs from target directory
	absSourcePath, err := filepath.Abs(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for source repository: %w", err)
	}

	// Use native git fetch with refspec: sourceBranch:targetRef
	// The leading "+" forces non-fast-forward updates, so a stale targetRef
	// left over from a previous (possibly crashed) run does not cause the
	// fetch to be rejected with "non-fast-forward".
	refspec := fmt.Sprintf("+refs/heads/%s:%s", sourceBranch, targetRef)
	cmd := exec.Command("git", "fetch", absSourcePath, refspec)
	cmd.Dir = targetPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git fetch failed: %w, output: %s", err, string(output))
	}

	return nil
}
