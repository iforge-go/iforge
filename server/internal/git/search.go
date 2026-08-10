package git

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// MaxSearchResults limits the maximum number of files returned per search to avoid
// oversized responses or long processing times in large repositories.
const MaxSearchResults = 200

// MaxSearchFiles limits the maximum number of files scanned per search to prevent
// large repositories from blocking for too long.
const MaxSearchFiles = 10000

// MaxFileSize limits individual file size (bytes); oversized files are skipped for performance.
const MaxFileSize = 1 * 1024 * 1024 // 1MB

// SearchWorkers is the number of concurrent search goroutines.
const SearchWorkers = 4

// errSearchLimitReached is an internal sentinel error used to terminate ForEach iteration early.
// Not exported; callers should check SearchCodeResponse.Truncated to determine if results were truncated.
var errSearchLimitReached = errors.New("search limit reached")

// SearchCodeResponse wraps search results with a truncation flag.
type SearchCodeResponse struct {
	Results   []SearchResult `json:"results"`
	Truncated bool           `json:"truncated"`
}

// skippedDirs contains directory names that should be skipped (dependency dirs, build artifacts, etc.)
var skippedDirs = map[string]bool{
	"node_modules":  true,
	"vendor":        true,
	".git":          true,
	".svn":          true,
	".hg":           true,
	"__pycache__":   true,
	".cache":        true,
	".npm":          true,
	".yarn":         true,
	"dist":          true,
	"build":         true,
	"target":        true,
	".next":         true,
	".nuxt":         true,
	".vuepress":     true,
	".docusaurus":   true,
	".turbo":        true,
	".parcel-cache": true,
}

// skippedExtensions contains file extensions that should be skipped (binaries, images, fonts, etc.)
var skippedExtensions = map[string]bool{
	// Binaries / executables
	".exe": true, ".dll": true, ".so": true, ".dylib": true, ".bin": true,
	".obj": true, ".o": true, ".a": true, ".lib": true,
	// Images and media
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".bmp": true,
	".ico": true, ".svg": true, ".webp": true, ".avif": true,
	".mp3": true, ".mp4": true, ".wav": true, ".avi": true, ".mov": true,
	".webm": true, ".mkv": true, ".flac": true, ".ogg": true,
	// Documents
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	".ppt": true, ".pptx": true,
	// Archives
	".zip": true, ".tar": true, ".gz": true, ".bz2": true, ".xz": true,
	".7z": true, ".rar": true, ".jar": true, ".war": true,
	// Databases
	".db": true, ".sqlite": true, ".sqlite3": true,
	// Fonts
	".ttf": true, ".otf": true, ".woff": true, ".woff2": true, ".eot": true,
	// Other binaries
	".class": true, ".pyc": true, ".pyo": true,
	".wasm": true,
}

// shouldSkipPath checks whether a file path should be skipped.
func shouldSkipPath(path string) bool {
	// Check each directory component in the path
	parts := strings.Split(filepath.ToSlash(path), "/")
	for _, part := range parts {
		if skippedDirs[part] {
			return true
		}
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(path))
	if skippedExtensions[ext] {
		return true
	}

	return false
}

// SearchCode searches for code in repository files.
//
// Behavior:
//   - Case-insensitive substring matching
//   - Skips binaries and dependency directories (node_modules, vendor, .git, etc.)
//   - Skips oversized files (>1MB)
//   - Up to 10 matching lines per file
//   - Returns at most MaxSearchResults files (200); Truncated=true when exceeded
//   - Context cancellation (including timeout) stops iteration immediately; partial results are returned with Truncated=true
//   - Uses concurrent worker goroutines to accelerate search
func (s *Client) SearchCode(ctx context.Context, owner, repo, ref, query string) (*SearchCodeResponse, error) {
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

	resp := &SearchCodeResponse{Results: []SearchResult{}}
	queryLower := strings.ToLower(query)
	filesScanned := 0

	// Collect files to search
	type fileToSearch struct {
		name    string
		content string
	}
	var files []fileToSearch

	// Phase 1: collect files (fast filter)
	err = tree.Files().ForEach(func(file *object.File) error {
		// 1. Check if ctx has been cancelled (including timeout)
		if err := ctx.Err(); err != nil {
			return err
		}
		// 2. Check if we've reached the scan file limit
		if filesScanned >= MaxSearchFiles {
			return errSearchLimitReached
		}

		filesScanned++

		// 3. Path filter: skip dependency directories and binary files
		if shouldSkipPath(file.Name) {
			return nil
		}

		// 4. File size filter: skip oversized files
		if file.Size > MaxFileSize {
			return nil
		}

		// 5. Skip binary files (go-git detection)
		isBinary, _ := file.IsBinary()
		if isBinary {
			return nil
		}

		// 6. Read file content
		content, err := file.Contents()
		if err != nil {
			return nil // Skip files that can't be read
		}

		files = append(files, fileToSearch{
			name:    file.Name,
			content: content,
		})

		return nil
	})

	if err != nil {
		// Scan limit reached: continue searching already-collected files
		if !errors.Is(err, errSearchLimitReached) {
			// ctx timeout or cancellation: mark truncated and return partial results
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				resp.Truncated = true
				return resp, nil
			}
			return nil, fmt.Errorf("failed to search files: %w", err)
		}
	}

	// Phase 2: concurrent search
	var mu sync.Mutex
	var wg sync.WaitGroup
	fileChan := make(chan fileToSearch, len(files))

	// Start worker goroutines
	for i := 0; i < SearchWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range fileChan {
				// Check if result limit has been reached
				mu.Lock()
				if len(resp.Results) >= MaxSearchResults {
					mu.Unlock()
					return
				}
				mu.Unlock()

				// Search file content
				lines := strings.Split(f.content, "\n")
				var matches []string

				for i, line := range lines {
					if strings.Contains(strings.ToLower(line), queryLower) {
						lineNum := i + 1
						matches = append(matches, fmt.Sprintf("L%d: %s", lineNum, strings.TrimSpace(line)))

						// Limit matches per file
						if len(matches) >= 10 {
							break
						}
					}
				}

				// Add to results
				if len(matches) > 0 {
					mu.Lock()
					if len(resp.Results) < MaxSearchResults {
						resp.Results = append(resp.Results, SearchResult{
							Path:    f.name,
							Matches: matches,
						})
					}
					mu.Unlock()
				}
			}
		}()
	}

	// Send files to worker goroutines
	for _, f := range files {
		select {
		case <-ctx.Done():
			close(fileChan)
			wg.Wait()
			resp.Truncated = true
			return resp, nil
		case fileChan <- f:
		}
	}
	close(fileChan)

	// Wait for all worker goroutines to finish
	wg.Wait()

	// Check if truncated due to result limit
	if len(resp.Results) >= MaxSearchResults {
		resp.Truncated = true
	}

	return resp, nil
}

// BlameFile returns blame information for a file
func (s *Client) BlameFile(owner, repo, ref, path string) (*BlameResult, error) {
	// Normalize path to use forward slashes for git operations
	path = strings.ReplaceAll(path, "\\", "/")

	repoPath := s.RepositoryPath(owner, repo)

	// Open repository
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	// Get commit
	var commit *object.Commit
	if ref == "" || ref == "HEAD" {
		headRef, err := r.Head()
		if err != nil {
			// HEAD may be unborn; try the first available branch
			branchIter, bErr := r.Branches()
			if bErr != nil {
				return nil, fmt.Errorf("failed to get HEAD: %w", err)
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
				return nil, fmt.Errorf("failed to get HEAD: %w", err)
			}
			commit, err = r.CommitObject(fallbackHash)
			if err != nil {
				return nil, fmt.Errorf("failed to get commit: %w", err)
			}
		} else {
			commit, err = r.CommitObject(headRef.Hash())
			if err != nil {
				return nil, fmt.Errorf("failed to get commit: %w", err)
			}
		}
	} else {
		// Try as branch
		branchRef, err := r.Reference(plumbing.NewBranchReferenceName(ref), true)
		if err == nil {
			commit, err = r.CommitObject(branchRef.Hash())
			if err != nil {
				return nil, fmt.Errorf("failed to get commit: %w", err)
			}
		} else {
			// Try as tag
			tagRef, err := r.Reference(plumbing.NewTagReferenceName(ref), true)
			if err == nil {
				commit, err = r.CommitObject(tagRef.Hash())
				if err != nil {
					return nil, fmt.Errorf("failed to get commit: %w", err)
				}
			} else {
				return nil, fmt.Errorf("reference not found: %s", ref)
			}
		}
	}

	// Get file content
	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tree: %w", err)
	}
	file, err := tree.File(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	content, err := file.Contents()
	if err != nil {
		return nil, fmt.Errorf("failed to get file content: %w", err)
	}

	lines := strings.Split(content, "\n")
	blameLines := make([]BlameLine, len(lines))

	// For each line, find the commit that last modified it
	for i, line := range lines {
		blameLines[i] = BlameLine{
			CommitID: commit.Hash.String(),
			Author:   commit.Author.Name,
			Email:    commit.Author.Email,
			Date:     commit.Author.When.Format("2006-01-02"),
			LineNum:  i + 1,
			Content:  line,
		}
	}

	return &BlameResult{Lines: blameLines}, nil
}
