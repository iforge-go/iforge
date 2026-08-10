package git

import (
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// resolveReference resolves a reference name to a commit hash
// Supports: HEAD, branch names, and commit hashes
func (s *Client) resolveReference(r *git.Repository, ref string) (plumbing.Hash, error) {
	if ref == "" || ref == "HEAD" {
		refObj, err := r.Head()
		if err != nil {
			// HEAD doesn't exist, try to find any branch
			branches, err := r.Branches()
			if err != nil {
				return plumbing.Hash{}, fmt.Errorf("failed to get HEAD: %w", err)
			}

			var firstBranch *plumbing.Reference
			err = branches.ForEach(func(branch *plumbing.Reference) error {
				if firstBranch == nil {
					firstBranch = branch
				}
				return nil
			})

			if err != nil || firstBranch == nil {
				return plumbing.Hash{}, fmt.Errorf("failed to get HEAD: %w", err)
			}

			return firstBranch.Hash(), nil
		}
		return refObj.Hash(), nil
	}

	// Try as branch reference first
	branchRef := plumbing.NewBranchReferenceName(ref)
	refObj, err := r.Reference(branchRef, true)
	if err == nil {
		return refObj.Hash(), nil
	}

	// Try as tag reference
	tagRef := plumbing.NewTagReferenceName(ref)
	refObj, err = r.Reference(tagRef, true)
	if err == nil {
		return refObj.Hash(), nil
	}

	// Try as commit hash
	hash := plumbing.NewHash(ref)
	if hash.String() != "0000000000000000000000000000000000000000" {
		return hash, nil
	}

	return plumbing.Hash{}, fmt.Errorf("reference not found: %s", ref)
}

// resolveRef resolves a ref (commit hash, branch name, or tag name) to a plumbing.Hash.
func (s *Client) resolveRef(r *git.Repository, ref string) (plumbing.Hash, error) {
	// Try as commit hash first
	if len(ref) >= 7 {
		hash := plumbing.NewHash(ref)
		if !hash.IsZero() {
			if _, err := r.CommitObject(hash); err == nil {
				return hash, nil
			}
		}
	}

	// Try as branch name
	refName := plumbing.NewBranchReferenceName(ref)
	if hash, err := r.ResolveRevision(plumbing.Revision(refName)); err == nil {
		return *hash, nil
	}

	// Try as tag name
	tagName := plumbing.NewTagReferenceName(ref)
	if hash, err := r.ResolveRevision(plumbing.Revision(tagName)); err == nil {
		return *hash, nil
	}

	// Try as a direct revision
	if hash, err := r.ResolveRevision(plumbing.Revision(ref)); err == nil {
		return *hash, nil
	}

	return plumbing.ZeroHash, fmt.Errorf("cannot resolve ref: %s", ref)
}

// ResolveRef resolves a reference (branch, tag, commit hash, or arbitrary ref path) to a commit hash.
// This is a public wrapper around resolveRef for external use.
func (s *Client) ResolveRef(owner, repo, ref string) (string, error) {
	repoPath := s.RepositoryPath(owner, repo)

	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	hash, err := s.resolveRef(r, ref)
	if err != nil {
		return "", err
	}

	return hash.String(), nil
}

// CleanupMRRef removes the temporary reference created for a merge request.
func (s *Client) CleanupMRRef(owner, repo string, mrID int) error {
	repoPath := s.RepositoryPath(owner, repo)

	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	refName := plumbing.ReferenceName(fmt.Sprintf("refs/mr/%d/head", mrID))

	// Check if reference exists
	ref, err := r.Reference(refName, true)
	if err != nil {
		// Reference doesn't exist, nothing to clean up
		return nil
	}

	// Remove the reference
	if err := r.Storer.RemoveReference(ref.Name()); err != nil {
		return fmt.Errorf("failed to remove reference %s: %w", refName, err)
	}

	return nil
}

// CleanupCompareRef removes the temporary reference used for cross-repo compare operations.
func (s *Client) CleanupCompareRef(owner, repo string) error {
	repoPath := s.RepositoryPath(owner, repo)

	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	refName := plumbing.ReferenceName("refs/compare/temp")

	// Check if reference exists
	ref, err := r.Reference(refName, true)
	if err != nil {
		// Reference doesn't exist, nothing to clean up
		return nil
	}

	// Remove the reference
	if err := r.Storer.RemoveReference(ref.Name()); err != nil {
		return fmt.Errorf("failed to remove reference %s: %w", refName, err)
	}

	return nil
}
