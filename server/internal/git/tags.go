package git

import (
	"fmt"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// ListTags lists all tags in a repository
func (s *Client) ListTags(owner, repo string) ([]TagInfo, error) {
	repoPath := s.RepositoryPath(owner, repo)

	// Open repository
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	// Get tag iterator
	tagIter, err := r.Tags()
	if err != nil {
		return nil, fmt.Errorf("failed to get tags: %w", err)
	}

	tags := []TagInfo{}
	err = tagIter.ForEach(func(ref *plumbing.Reference) error {
		tagName := ref.Name().Short()
		tagInfo := TagInfo{
			Name:     tagName,
			CommitId: ref.Hash().String(),
		}

		// Try to get annotated tag object
		tagObj, err := r.TagObject(ref.Hash())
		if err == nil {
			// This is an annotated tag
			tagInfo.Message = tagObj.Message
			tagInfo.Tagger = tagObj.Tagger.Name
			tagInfo.TagDate = tagObj.Tagger.When
			tagInfo.IsAnnotated = true
			// Get the commit hash the tag points to
			commit, err := tagObj.Commit()
			if err == nil {
				tagInfo.CommitId = commit.Hash.String()
			}
		} else {
			// This is a lightweight tag
			tagInfo.IsAnnotated = false
		}

		tags = append(tags, tagInfo)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to iterate tags: %w", err)
	}

	return tags, nil
}

// CreateTag creates a new tag
func (s *Client) CreateTag(owner, repo, tagName, target, message, taggerName, taggerEmail string) error {
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

	// Get target commit
	var targetHash plumbing.Hash
	if target == "" || target == "HEAD" {
		ref, err := r.Head()
		if err != nil {
			// HEAD might be unborn, try to find any branch
			branchIter, bErr := r.Branches()
			if bErr != nil {
				return fmt.Errorf("failed to get HEAD: %w", err)
			}
			var fallbackHash plumbing.Hash
			var found bool
			_ = branchIter.ForEach(func(b *plumbing.Reference) error {
				if !found {
					fallbackHash = b.Hash()
					found = true
				}
				return nil
			})
			if !found {
				return fmt.Errorf("failed to get HEAD: %w", err)
			}
			targetHash = fallbackHash
		} else {
			targetHash = ref.Hash()
		}
	} else {
		// Try as branch
		ref, err := r.Reference(plumbing.NewBranchReferenceName(target), true)
		if err == nil {
			targetHash = ref.Hash()
		} else {
			// Try as commit hash
			targetHash = plumbing.NewHash(target)
		}
	}

	if message != "" {
		// Create annotated tag
		tag := &object.Tag{
			Name:   tagName,
			Target: targetHash,
			Tagger: object.Signature{
				Name:  taggerName,
				Email: taggerEmail,
				When:  time.Now(),
			},
			Message: message,
		}

		encoded := &plumbing.MemoryObject{}
		if err := tag.Encode(encoded); err != nil {
			return fmt.Errorf("failed to encode tag: %w", err)
		}

		tagHash, err := r.Storer.SetEncodedObject(encoded)
		if err != nil {
			return fmt.Errorf("failed to store tag: %w", err)
		}

		// Create reference
		ref := plumbing.NewHashReference(plumbing.NewTagReferenceName(tagName), tagHash)
		if err := r.Storer.SetReference(ref); err != nil {
			return fmt.Errorf("failed to create tag reference: %w", err)
		}
	} else {
		// Create lightweight tag
		ref := plumbing.NewHashReference(plumbing.NewTagReferenceName(tagName), targetHash)
		if err := r.Storer.SetReference(ref); err != nil {
			return fmt.Errorf("failed to create tag reference: %w", err)
		}
	}

	// Invalidate caches (tag list, commit list, etc.)
	InvalidateRepoCaches(owner, repo)
	return nil
}

// DeleteTag deletes a tag
func (s *Client) DeleteTag(owner, repo, tagName string) error {
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

	// Delete tag reference
	if err := r.Storer.RemoveReference(plumbing.NewTagReferenceName(tagName)); err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}

	// Invalidate caches (tag list, commit list, etc.)
	InvalidateRepoCaches(owner, repo)
	return nil
}
