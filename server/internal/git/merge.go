package git

import (
	"fmt"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// MergeBranch merges a branch into another branch (for bare repositories)
func (s *Client) MergeBranch(owner, repo, baseBranch, headBranch, commitMessage, authorName, authorEmail string) (string, error) {
	repoPath := s.RepositoryPath(owner, repo)

	// Open repository
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	// Get base branch commit
	baseRef, err := r.Reference(plumbing.NewBranchReferenceName(baseBranch), true)
	if err != nil {
		return "", fmt.Errorf("failed to get base branch: %w", err)
	}

	baseCommit, err := r.CommitObject(baseRef.Hash())
	if err != nil {
		return "", fmt.Errorf("failed to get base commit: %w", err)
	}

	// Get head branch commit
	headHash, err := s.resolveRef(r, headBranch)
	if err != nil {
		return "", fmt.Errorf("failed to get head branch: %w", err)
	}

	headCommit, err := r.CommitObject(headHash)
	if err != nil {
		return "", fmt.Errorf("failed to get head commit: %w", err)
	}

	// Check if merge is needed (fast-forward check)
	baseTree, err := baseCommit.Tree()
	if err != nil {
		return "", fmt.Errorf("failed to get base tree: %w", err)
	}

	headTree, err := headCommit.Tree()
	if err != nil {
		return "", fmt.Errorf("failed to get head tree: %w", err)
	}

	// If trees are the same, no merge needed
	if baseTree.Hash == headTree.Hash {
		return headCommit.Hash.String(), nil
	}

	// Create merge commit with two parents
	mergeCommit := &object.Commit{
		TreeHash: headTree.Hash,
		Author: object.Signature{
			Name:  authorName,
			Email: authorEmail,
			When:  time.Now(),
		},
		Committer: object.Signature{
			Name:  authorName,
			Email: authorEmail,
			When:  time.Now(),
		},
		Message:      commitMessage,
		ParentHashes: []plumbing.Hash{baseCommit.Hash, headCommit.Hash},
	}

	// Encode and store the merge commit
	encoded := &plumbing.MemoryObject{}
	if err := mergeCommit.Encode(encoded); err != nil {
		return "", fmt.Errorf("failed to encode merge commit: %w", err)
	}

	commitHash, err := r.Storer.SetEncodedObject(encoded)
	if err != nil {
		return "", fmt.Errorf("failed to store merge commit: %w", err)
	}

	// Update base branch reference
	newRef := plumbing.NewHashReference(plumbing.NewBranchReferenceName(baseBranch), commitHash)
	if err := r.Storer.SetReference(newRef); err != nil {
		return "", fmt.Errorf("failed to update branch reference: %w", err)
	}

	return commitHash.String(), nil
}

// SquashMerge performs a squash merge of headBranch into baseBranch
// All commits from headBranch are squashed into a single commit on baseBranch
func (s *Client) SquashMerge(owner, repo, baseBranch, headBranch, commitMessage, authorName, authorEmail string) (string, error) {
	repoPath := s.RepositoryPath(owner, repo)

	// Open repository
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	// Get base branch commit
	baseRef, err := r.Reference(plumbing.NewBranchReferenceName(baseBranch), true)
	if err != nil {
		return "", fmt.Errorf("failed to get base branch: %w", err)
	}

	baseCommit, err := r.CommitObject(baseRef.Hash())
	if err != nil {
		return "", fmt.Errorf("failed to get base commit: %w", err)
	}

	// Get head branch commit
	headHash, err := s.resolveRef(r, headBranch)
	if err != nil {
		return "", fmt.Errorf("failed to get head branch: %w", err)
	}

	headCommit, err := r.CommitObject(headHash)
	if err != nil {
		return "", fmt.Errorf("failed to get head commit: %w", err)
	}

	// Get head tree (this becomes the new tree)
	headTree, err := headCommit.Tree()
	if err != nil {
		return "", fmt.Errorf("failed to get head tree: %w", err)
	}

	// Create a single squash commit with only the base commit as parent
	squashCommit := &object.Commit{
		TreeHash: headTree.Hash,
		Author: object.Signature{
			Name:  authorName,
			Email: authorEmail,
			When:  time.Now(),
		},
		Committer: object.Signature{
			Name:  authorName,
			Email: authorEmail,
			When:  time.Now(),
		},
		Message:      commitMessage,
		ParentHashes: []plumbing.Hash{baseCommit.Hash},
	}

	// Encode and store the squash commit
	encoded := &plumbing.MemoryObject{}
	if err := squashCommit.Encode(encoded); err != nil {
		return "", fmt.Errorf("failed to encode squash commit: %w", err)
	}

	commitHash, err := r.Storer.SetEncodedObject(encoded)
	if err != nil {
		return "", fmt.Errorf("failed to store squash commit: %w", err)
	}

	// Update base branch reference
	newRef := plumbing.NewHashReference(plumbing.NewBranchReferenceName(baseBranch), commitHash)
	if err := r.Storer.SetReference(newRef); err != nil {
		return "", fmt.Errorf("failed to update branch reference: %w", err)
	}

	return commitHash.String(), nil
}

// RebaseMerge rebases commits from headBranch onto baseBranch
// This replays each commit from headBranch on top of baseBranch
func (s *Client) RebaseMerge(owner, repo, baseBranch, headBranch, authorName, authorEmail string) (string, error) {
	repoPath := s.RepositoryPath(owner, repo)

	// Open repository
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	// Get base branch commit
	baseRef, err := r.Reference(plumbing.NewBranchReferenceName(baseBranch), true)
	if err != nil {
		return "", fmt.Errorf("failed to get base branch: %w", err)
	}

	baseCommit, err := r.CommitObject(baseRef.Hash())
	if err != nil {
		return "", fmt.Errorf("failed to get base commit: %w", err)
	}

	// Get head branch commit
	headHash, err := s.resolveRef(r, headBranch)
	if err != nil {
		return "", fmt.Errorf("failed to get head branch: %w", err)
	}

	headCommit, err := r.CommitObject(headHash)
	if err != nil {
		return "", fmt.Errorf("failed to get head commit: %w", err)
	}

	// For a simplified rebase, we take the head tree and create a new commit
	// with the base commit as parent
	// In a full implementation, we would replay each commit individually
	headTree, err := headCommit.Tree()
	if err != nil {
		return "", fmt.Errorf("failed to get head tree: %w", err)
	}

	// Create rebased commit
	rebasedCommit := &object.Commit{
		TreeHash: headTree.Hash,
		Author: object.Signature{
			Name:  authorName,
			Email: authorEmail,
			When:  time.Now(),
		},
		Committer: object.Signature{
			Name:  authorName,
			Email: authorEmail,
			When:  time.Now(),
		},
		Message:      headCommit.Message,
		ParentHashes: []plumbing.Hash{baseCommit.Hash},
	}

	// Encode and store the rebased commit
	encoded := &plumbing.MemoryObject{}
	if err := rebasedCommit.Encode(encoded); err != nil {
		return "", fmt.Errorf("failed to encode rebased commit: %w", err)
	}

	commitHash, err := r.Storer.SetEncodedObject(encoded)
	if err != nil {
		return "", fmt.Errorf("failed to store rebased commit: %w", err)
	}

	// Update base branch reference
	newRef := plumbing.NewHashReference(plumbing.NewBranchReferenceName(baseBranch), commitHash)
	if err := r.Storer.SetReference(newRef); err != nil {
		return "", fmt.Errorf("failed to update branch reference: %w", err)
	}

	return commitHash.String(), nil
}
