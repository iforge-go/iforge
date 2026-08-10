package git

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// ListFiles lists files in a directory
func (s *Client) ListFiles(owner, repo, ref, path string) ([]FileEntry, error) {
	// Validate path to prevent path traversal attacks
	if err := ValidateRepoPath(path); err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Normalize path to use forward slashes for git operations
	path = strings.ReplaceAll(path, "\\", "/")

	cacheKey := FileListCacheKey(owner, repo, ref, path)
	if cached, ok := FileListCache.Get(cacheKey); ok {
		return cached.([]FileEntry), nil
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

	// Get commit
	commit, err := r.CommitObject(hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	// Get tree
	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tree: %w", err)
	}

	// Navigate to path if specified
	if path != "" && path != "/" {
		tree, err = tree.Tree(path)
		if err != nil {
			return nil, fmt.Errorf("failed to get tree for path %s: %w", path, err)
		}
	}

	// List entries
	entries := []FileEntry{}
	paths := []string{}
	for _, entry := range tree.Entries {
		entryType := "file"
		if entry.Mode == 040000 { // Directory mode
			entryType = "dir"
		}

		fullPath := strings.ReplaceAll(filepath.Join(path, entry.Name), "\\", "/")
		paths = append(paths, fullPath)

		fileEntry := FileEntry{
			Name: entry.Name,
			Path: fullPath,
			Type: entryType,
			Mode: fmt.Sprintf("%06o", entry.Mode),
		}
		entries = append(entries, fileEntry)
	}

	// Batch fetch last commits for all paths in a single BFS traversal
	lastCommits := s.getLastCommitsForPaths(r, commit, paths)
	for i := range entries {
		if lastCommit, ok := lastCommits[entries[i].Path]; ok {
			entries[i].LastCommit = &CommitInfo{
				ID:        lastCommit.Hash.String(),
				Message:   lastCommit.Message,
				Author:    lastCommit.Author.Name,
				Email:     lastCommit.Author.Email,
				Timestamp: lastCommit.Author.When,
			}
		}
	}

	FileListCache.Set(cacheKey, entries)
	return entries, nil
}

// ListAllFiles recursively returns all file paths in the repository tree.
// Only files (not directories) are returned. Used for the "Go to file" feature.
func (s *Client) ListAllFiles(owner, repo, ref string) ([]FileEntry, error) {
	cacheKey := FileListCacheKey(owner, repo, ref, "__all__")
	if cached, ok := FileListCache.Get(cacheKey); ok {
		return cached.([]FileEntry), nil
	}

	repoPath := s.RepositoryPath(owner, repo)

	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	hash, err := s.resolveReference(r, ref)
	if err != nil {
		return nil, err
	}

	commit, err := r.CommitObject(hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tree: %w", err)
	}

	entries := []FileEntry{}
	err = tree.Files().ForEach(func(f *object.File) error {
		entries = append(entries, FileEntry{
			Name: f.Name,
			Path: f.Name,
			Type: "file",
			Size: f.Size,
			Mode: fmt.Sprintf("%06o", f.Mode),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk tree: %w", err)
	}

	FileListCache.Set(cacheKey, entries)
	return entries, nil
}

// findTreeEntryRecursive finds a tree entry by path using go-git's built-in methods
func findTreeEntryRecursive(tree *object.Tree, path string) (*object.TreeEntry, error) {
	if path == "" || path == "/" {
		return nil, fmt.Errorf("empty path")
	}

	// Try as file first
	file, err := tree.File(path)
	if err == nil {
		return &object.TreeEntry{
			Name: file.Name,
			Mode: file.Mode,
			Hash: file.Hash,
		}, nil
	}

	// Try as directory - use tree.Tree which handles nested paths
	subTree, err := tree.Tree(path)
	if err == nil {
		parts := strings.Split(path, "/")
		name := parts[len(parts)-1]
		return &object.TreeEntry{
			Name: name,
			Mode: 040000,
			Hash: subTree.Hash,
		}, nil
	}

	return nil, fmt.Errorf("path not found: %s", path)
}

// getLastCommitsForPaths gets the last commit for multiple paths in a single traversal.
// Walks the commit graph via parent links with depth and time limits, sorts by time,
// then finds the last modification for each path by comparing file hashes with parent commits.
// This is much more efficient than calling getLastCommitForPath for each path individually.
func (s *Client) getLastCommitsForPaths(r *git.Repository, startCommit *object.Commit, paths []string) map[string]*object.Commit {
	result := make(map[string]*object.Commit)
	remaining := make(map[string]bool)
	for _, p := range paths {
		remaining[p] = true
	}

	// Performance limits for large repositories
	const (
		maxCommits  = 1000            // Maximum number of commits to traverse
		maxDuration = 5 * time.Second // Maximum traversal time
	)

	startTime := time.Now()

	// BFS walk of commit graph via parent links with limits
	var commits []*object.Commit
	visited := make(map[plumbing.Hash]bool)
	queue := []*object.Commit{startCommit}

	for len(queue) > 0 && len(commits) < maxCommits && time.Since(startTime) < maxDuration {
		c := queue[0]
		queue = queue[1:]
		if visited[c.Hash] {
			continue
		}
		visited[c.Hash] = true
		commits = append(commits, c)

		parents := c.Parents()
		for {
			p, err := parents.Next()
			if err != nil {
				break
			}
			queue = append(queue, p)
		}
	}

	// Sort by commit time (newest first)
	sort.Slice(commits, func(i, j int) bool {
		return commits[i].Author.When.After(commits[j].Author.When)
	})

	// Walk from newest to oldest, find the first commit that modified each path.
	// A commit "modified" a file if the file's hash differs from its parent's hash.
	for _, commit := range commits {
		if len(remaining) == 0 {
			break
		}

		tree, err := commit.Tree()
		if err != nil {
			continue
		}

		for path := range remaining {
			entry, err := findTreeEntryRecursive(tree, path)
			if err != nil {
				continue // File doesn't exist in this commit
			}

			// Check if any parent has a different hash for this path
			parents := commit.Parents()
			modified := true
			for {
				parent, err := parents.Next()
				if err != nil {
					break
				}
				parentTree, err := parent.Tree()
				if err != nil {
					continue
				}
				parentEntry, err := findTreeEntryRecursive(parentTree, path)
				if err != nil {
					// File doesn't exist in parent → this commit created it
					modified = true
					break
				}
				if parentEntry.Hash != entry.Hash {
					// Hash changed → this commit modified it
					modified = true
					break
				}
				// Same hash → this commit didn't modify it
				modified = false
			}

			if modified {
				result[path] = commit
				delete(remaining, path)
			}
		}
	}

	return result
}

// findTreeEntry finds a tree entry by name in the given tree
func findTreeEntry(tree *object.Tree, name string) *object.TreeEntry {
	for i := range tree.Entries {
		if tree.Entries[i].Name == name {
			return &tree.Entries[i]
		}
	}
	return nil
}

// GetFileContent gets the content of a file
func (s *Client) GetFileContent(owner, repo, ref, path string) (*FileContent, error) {
	// Validate path to prevent path traversal attacks
	if err := ValidateRepoPath(path); err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Normalize path to use forward slashes for git operations
	path = strings.ReplaceAll(path, "\\", "/")

	// Check cache first
	cacheKey := FileContentCacheKey(owner, repo, ref, path)
	if cached, ok := FileContentCache.Get(cacheKey); ok {
		return cached.(*FileContent), nil
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

	// Get commit
	commit, err := r.CommitObject(hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	// Get tree
	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tree: %w", err)
	}

	// Get file
	file, err := tree.File(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	// Get content
	content, err := file.Contents()
	if err != nil {
		return nil, fmt.Errorf("failed to get file content: %w", err)
	}

	// Check if binary
	isBinary := isBinaryContent(content)

	result := &FileContent{
		Name:     file.Name,
		Path:     path,
		Content:  content,
		Size:     file.Size,
		IsBinary: isBinary,
	}

	// Store in cache
	FileContentCache.Set(cacheKey, result)
	return result, nil
}

// GetRawFileContent gets raw file content as bytes
func (s *Client) GetRawFileContent(owner, repo, ref, path string) ([]byte, error) {
	// Validate path to prevent path traversal attacks
	if err := ValidateRepoPath(path); err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Normalize path to use forward slashes for git operations
	path = strings.ReplaceAll(path, "\\", "/")

	GetLogger().Debug("GetRawFileContent", map[string]interface{}{
		"owner": owner, "repo": repo, "ref": ref, "path": path,
	})

	repoPath := s.RepositoryPath(owner, repo)
	GetLogger().Debug("Repository path", map[string]interface{}{"path": repoPath})

	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	hash, err := s.resolveReference(r, ref)
	if err != nil {
		return nil, err
	}

	commit, err := r.CommitObject(hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tree: %w", err)
	}

	file, err := tree.File(path)
	if err != nil {
		GetLogger().Error("Failed to find file in tree", err, map[string]interface{}{"path": path})
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	reader, err := file.Reader()
	if err != nil {
		return nil, fmt.Errorf("failed to get file reader: %w", err)
	}
	defer reader.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	return buf.Bytes(), nil
}

// isBinaryContent checks if content is binary
func isBinaryContent(content string) bool {
	// Check first 8000 bytes for null bytes
	checkLen := 8000
	if len(content) < checkLen {
		checkLen = len(content)
	}

	for i := 0; i < checkLen; i++ {
		if content[i] == 0 {
			return true
		}
	}
	return false
}

// CreateFile creates a new file in the repository
func (s *Client) CreateFile(owner, repo, branch, path, content, message, authorName, authorEmail string) error {
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

	// Get branch reference
	ref, err := r.Reference(plumbing.NewBranchReferenceName(branch), true)
	if err != nil {
		// Branch doesn't exist - check if this is a truly empty repository
		headRef, headErr := r.Head()
		if headErr != nil {
			// Truly empty repository (no HEAD), create initial commit
			GetLogger().Debug("CreateFile: empty repository, creating initial commit", map[string]interface{}{
				"owner": owner, "repo": repo, "branch": branch, "path": path,
			})
			
			emptyTree := &object.Tree{}
			emptyTreeObject := &plumbing.MemoryObject{}
			if err := emptyTree.Encode(emptyTreeObject); err != nil {
				return fmt.Errorf("failed to encode empty tree: %w", err)
			}
			_, err = r.Storer.SetEncodedObject(emptyTreeObject)
			if err != nil {
				return fmt.Errorf("failed to store empty tree: %w", err)
			}

			// Create blob for the file
			blobObject := &plumbing.MemoryObject{}
			blobObject.SetType(plumbing.BlobObject)
			writer, err := blobObject.Writer()
			if err != nil {
				return fmt.Errorf("failed to create blob writer: %w", err)
			}
			if _, err := writer.Write([]byte(content)); err != nil {
				writer.Close()
				return fmt.Errorf("failed to write blob content: %w", err)
			}
			if err := writer.Close(); err != nil {
				return fmt.Errorf("failed to close blob writer: %w", err)
			}
			blobHash, err := r.Storer.SetEncodedObject(blobObject)
			if err != nil {
				return fmt.Errorf("failed to store blob: %w", err)
			}

			// Build tree with the file
			newTreeHash, err := s.addFileToTree(r, emptyTree, path, blobHash)
			if err != nil {
				return fmt.Errorf("failed to build tree: %w", err)
			}

			// Create initial commit (no parents)
			newCommit := &object.Commit{
				TreeHash: newTreeHash,
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
				Message:      message,
				ParentHashes: nil,
			}
			commitObject := &plumbing.MemoryObject{}
			if err := newCommit.Encode(commitObject); err != nil {
				return fmt.Errorf("failed to encode commit: %w", err)
			}
			commitHash, err := r.Storer.SetEncodedObject(commitObject)
			if err != nil {
				return fmt.Errorf("failed to store commit: %w", err)
			}

			// Create branch reference
			newRef := plumbing.NewHashReference(plumbing.NewBranchReferenceName(branch), commitHash)
			if err := r.Storer.SetReference(newRef); err != nil {
				return fmt.Errorf("failed to create branch: %w", err)
			}

			// Set HEAD to this branch
			if err := r.Storer.SetReference(plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName(branch))); err != nil {
				return fmt.Errorf("failed to set HEAD: %w", err)
			}

			InvalidateRepoCaches(owner, repo)
			return nil
		}
		
		// Repository has commits, but branch doesn't exist yet
		// Use HEAD commit as the base for the new branch
		GetLogger().Debug("CreateFile: branch doesn't exist, creating from HEAD", map[string]interface{}{
			"owner": owner, "repo": repo, "branch": branch, "path": path, "head": headRef.Hash().String(),
		})
		
		commit, err := r.CommitObject(headRef.Hash())
		if err != nil {
			return fmt.Errorf("failed to get HEAD commit: %w", err)
		}
		
		tree, err := commit.Tree()
		if err != nil {
			return fmt.Errorf("failed to get HEAD tree: %w", err)
		}
		
		// Create blob for the new file
		blobObject := &plumbing.MemoryObject{}
		blobObject.SetType(plumbing.BlobObject)
		writer, err := blobObject.Writer()
		if err != nil {
			return fmt.Errorf("failed to create blob writer: %w", err)
		}
		if _, err := writer.Write([]byte(content)); err != nil {
			writer.Close()
			return fmt.Errorf("failed to write blob content: %w", err)
		}
		if err := writer.Close(); err != nil {
			return fmt.Errorf("failed to close blob writer: %w", err)
		}
		blobHash, err := r.Storer.SetEncodedObject(blobObject)
		if err != nil {
			return fmt.Errorf("failed to store blob: %w", err)
		}
		
		// Build new tree with the file
		newTree, err := s.addFileToTree(r, tree, path, blobHash)
		if err != nil {
			return fmt.Errorf("failed to add file to tree: %w", err)
		}
		
		// Create new commit
		newCommit := &object.Commit{
			TreeHash: newTree,
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
			Message:      message,
			ParentHashes: []plumbing.Hash{commit.Hash},
		}
		
		commitObject := &plumbing.MemoryObject{}
		if err := newCommit.Encode(commitObject); err != nil {
			return fmt.Errorf("failed to encode commit: %w", err)
		}
		commitHash, err := r.Storer.SetEncodedObject(commitObject)
		if err != nil {
			return fmt.Errorf("failed to store commit: %w", err)
		}
		
		// Create branch reference pointing to the new commit
		newRef := plumbing.NewHashReference(plumbing.NewBranchReferenceName(branch), commitHash)
		if err := r.Storer.SetReference(newRef); err != nil {
			return fmt.Errorf("failed to create branch: %w", err)
		}
		
		InvalidateRepoCaches(owner, repo)
		return nil
	}

	// Get current commit
	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		return fmt.Errorf("failed to get commit: %w", err)
	}

	// Get current tree
	tree, err := commit.Tree()
	if err != nil {
		return fmt.Errorf("failed to get tree: %w", err)
	}

	// Create blob object for new file
	blobObject := &plumbing.MemoryObject{}
	blobObject.SetType(plumbing.BlobObject)
	writer, err := blobObject.Writer()
	if err != nil {
		return fmt.Errorf("failed to create blob writer: %w", err)
	}
	if _, err := writer.Write([]byte(content)); err != nil {
		writer.Close()
		return fmt.Errorf("failed to write blob content: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close blob writer: %w", err)
	}

	blobHash, err := r.Storer.SetEncodedObject(blobObject)
	if err != nil {
		return fmt.Errorf("failed to store blob: %w", err)
	}

	// Build new tree with the file
	newTree, err := s.addFileToTree(r, tree, path, blobHash)
	if err != nil {
		return fmt.Errorf("failed to add file to tree: %w", err)
	}

	// Check if tree actually changed — prevent empty commits
	if newTree == tree.Hash {
		return fmt.Errorf("file content unchanged, no commit created")
	}

	// Create new commit
	newCommit := &object.Commit{
		TreeHash: newTree,
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
		Message:      message,
		ParentHashes: []plumbing.Hash{commit.Hash},
	}

	// Encode commit
	commitObject := &plumbing.MemoryObject{}
	if err := newCommit.Encode(commitObject); err != nil {
		return fmt.Errorf("failed to encode commit: %w", err)
	}

	commitHash, err := r.Storer.SetEncodedObject(commitObject)
	if err != nil {
		return fmt.Errorf("failed to store commit: %w", err)
	}

	// Update branch reference
	newRef := plumbing.NewHashReference(plumbing.NewBranchReferenceName(branch), commitHash)
	if err := r.Storer.SetReference(newRef); err != nil {
		return fmt.Errorf("failed to update branch: %w", err)
	}

	// Invalidate all caches for this repository (per-repo, not global)
	InvalidateRepoCaches(owner, repo)

	return nil
}

// UpdateFile updates an existing file in the repository
func (s *Client) UpdateFile(owner, repo, branch, path, content, message, authorName, authorEmail string) error {
	// Same as CreateFile but replaces existing file
	return s.CreateFile(owner, repo, branch, path, content, message, authorName, authorEmail)
}

// DeleteFile deletes a file from the repository
func (s *Client) DeleteFile(owner, repo, branch, path, message, authorName, authorEmail string) error {
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

	// Get branch reference
	ref, err := r.Reference(plumbing.NewBranchReferenceName(branch), true)
	if err != nil {
		return fmt.Errorf("failed to get branch: %w", err)
	}

	// Get current commit
	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		return fmt.Errorf("failed to get commit: %w", err)
	}

	// Get current tree
	tree, err := commit.Tree()
	if err != nil {
		return fmt.Errorf("failed to get tree: %w", err)
	}

	// Build new tree without the file
	newTree, err := s.removeFileFromTree(r, tree, path)
	if err != nil {
		return fmt.Errorf("failed to remove file from tree: %w", err)
	}

	// Create new commit
	newCommit := &object.Commit{
		TreeHash: newTree,
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
		Message:      message,
		ParentHashes: []plumbing.Hash{commit.Hash},
	}

	// Encode commit
	commitObject := &plumbing.MemoryObject{}
	if err := newCommit.Encode(commitObject); err != nil {
		return fmt.Errorf("failed to encode commit: %w", err)
	}

	commitHash, err := r.Storer.SetEncodedObject(commitObject)
	if err != nil {
		return fmt.Errorf("failed to store commit: %w", err)
	}

	// Update branch reference
	newRef := plumbing.NewHashReference(plumbing.NewBranchReferenceName(branch), commitHash)
	if err := r.Storer.SetReference(newRef); err != nil {
		return fmt.Errorf("failed to update branch: %w", err)
	}

	// Invalidate all caches for this repository (per-repo, not global)
	InvalidateRepoCaches(owner, repo)

	return nil
}

// addFileToTree adds a file to a tree, creating parent directories as needed
func (s *Client) addFileToTree(r *git.Repository, tree *object.Tree, path string, blobHash plumbing.Hash) (plumbing.Hash, error) {
	// Split path into components
	parts := strings.Split(path, "/")

	// Build tree recursively from the deepest directory
	return s.buildTreeRecursive(r, tree, parts, 0, blobHash)
}

// buildTreeRecursive recursively builds tree structure
func (s *Client) buildTreeRecursive(r *git.Repository, currentTree *object.Tree, parts []string, index int, blobHash plumbing.Hash) (plumbing.Hash, error) {
	if index >= len(parts) {
		return currentTree.Hash, nil
	}

	currentName := parts[index]
	isLast := index == len(parts)-1

	// Find or create the entry
	var entries []object.TreeEntry
	found := false

	for _, entry := range currentTree.Entries {
		if entry.Name == currentName {
			found = true
			if isLast {
				// Replace file entry
				entries = append(entries, object.TreeEntry{
					Name: currentName,
					Mode: 0100644,
					Hash: blobHash,
				})
			} else {
				// Recurse into directory
				subTree, err := r.TreeObject(entry.Hash)
				if err != nil {
					return plumbing.Hash{}, err
				}
				newSubTreeHash, err := s.buildTreeRecursive(r, subTree, parts, index+1, blobHash)
				if err != nil {
					return plumbing.Hash{}, err
				}
				entries = append(entries, object.TreeEntry{
					Name: currentName,
					Mode: 040000,
					Hash: newSubTreeHash,
				})
			}
		} else {
			entries = append(entries, entry)
		}
	}

	if !found {
		if isLast {
			// Add new file entry
			entries = append(entries, object.TreeEntry{
				Name: currentName,
				Mode: 0100644,
				Hash: blobHash,
			})
		} else {
			// Create new directory
			emptyTree := &object.Tree{}
			newSubTreeHash, err := s.buildTreeRecursive(r, emptyTree, parts, index+1, blobHash)
			if err != nil {
				return plumbing.Hash{}, err
			}
			entries = append(entries, object.TreeEntry{
				Name: currentName,
				Mode: 040000,
				Hash: newSubTreeHash,
			})
		}
	}

	// Create new tree
	sortTreeEntries(entries)
	newTree := &object.Tree{
		Entries: entries,
	}

	treeObject := &plumbing.MemoryObject{}
	if err := newTree.Encode(treeObject); err != nil {
		return plumbing.Hash{}, err
	}

	return r.Storer.SetEncodedObject(treeObject)
}

// sortTreeEntries sorts tree entries by name, with directories (mode 040000)
// sorted as if they had a trailing "/" per git tree encoding requirements.
func sortTreeEntries(entries []object.TreeEntry) {
	sort.Slice(entries, func(i, j int) bool {
		a := entries[i].Name
		if entries[i].Mode == 040000 {
			a += "/"
		}
		b := entries[j].Name
		if entries[j].Mode == 040000 {
			b += "/"
		}
		return a < b
	})
}

// removeFileFromTree removes a file from a tree
func (s *Client) removeFileFromTree(r *git.Repository, tree *object.Tree, path string) (plumbing.Hash, error) {
	parts := strings.Split(path, "/")
	return s.removeFileRecursive(r, tree, parts, 0)
}

// removeFileRecursive recursively removes file from tree
func (s *Client) removeFileRecursive(r *git.Repository, currentTree *object.Tree, parts []string, index int) (plumbing.Hash, error) {
	if index >= len(parts) {
		return currentTree.Hash, nil
	}

	currentName := parts[index]
	isLast := index == len(parts)-1

	var entries []object.TreeEntry

	for _, entry := range currentTree.Entries {
		if entry.Name == currentName {
			if isLast {
				// Skip this entry (remove it)
				continue
			} else {
				// Recurse into directory
				subTree, err := r.TreeObject(entry.Hash)
				if err != nil {
					return plumbing.Hash{}, err
				}
				newSubTreeHash, err := s.removeFileRecursive(r, subTree, parts, index+1)
				if err != nil {
					return plumbing.Hash{}, err
				}
				// Only add if directory is not empty
				if newSubTreeHash != plumbing.ZeroHash {
					entries = append(entries, object.TreeEntry{
						Name: currentName,
						Mode: 040000,
						Hash: newSubTreeHash,
					})
				}
			}
		} else {
			entries = append(entries, entry)
		}
	}

	// If no entries left, return zero hash
	if len(entries) == 0 {
		return plumbing.ZeroHash, nil
	}

	// Create new tree
	sortTreeEntries(entries)
	newTree := &object.Tree{
		Entries: entries,
	}

	treeObject := &plumbing.MemoryObject{}
	if err := newTree.Encode(treeObject); err != nil {
		return plumbing.Hash{}, err
	}

	return r.Storer.SetEncodedObject(treeObject)
}

// GetIssueTemplates retrieves issue templates from the repository
// Templates are stored in .gitea/ISSUE_TEMPLATE/ or .github/ISSUE_TEMPLATE/
func (s *Client) GetIssueTemplates(owner, repo string) ([]IssueTemplate, error) {
	repoPath := s.RepositoryPath(owner, repo)

	// Open repository
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	// Get HEAD reference
	headRef, err := r.Head()
	if err != nil {
		// Repository might be empty
		return []IssueTemplate{}, nil
	}

	// Get HEAD commit
	commit, err := r.CommitObject(headRef.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD commit: %w", err)
	}

	// Get tree
	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tree: %w", err)
	}

	var templates []IssueTemplate

	// Check for templates in .gitea/ISSUE_TEMPLATE/ and .github/ISSUE_TEMPLATE/
	templateDirs := []string{".gitea/ISSUE_TEMPLATE", ".github/ISSUE_TEMPLATE"}

	for _, templateDir := range templateDirs {
		dirTree, err := tree.Tree(templateDir)
		if err != nil {
			continue // Directory doesn't exist
		}

		// Iterate through files in the template directory
		files := dirTree.Files()
		for {
			file, err := files.Next()
			if err != nil {
				break
			}

			// Only process markdown files
			if !strings.HasSuffix(strings.ToLower(file.Name), ".md") && !strings.HasSuffix(strings.ToLower(file.Name), ".yml") {
				continue
			}

			// Read file content
			content, err := file.Contents()
			if err != nil {
				continue
			}

			// Extract template name from filename (without extension)
			name := strings.TrimSuffix(file.Name, ".md")
			name = strings.TrimSuffix(name, ".yml")

			templates = append(templates, IssueTemplate{
				Name:    name,
				Content: content,
			})
		}
	}

	return templates, nil
}
