package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// Client handles Git operations. It is a pure Git layer with no
// database access; branch-protection persistence lives in RepositoryService.
type Client struct {
	reposPath string
}

// NewClient creates a new Client.
func NewClient(reposPath string) *Client {
	return &Client{
		reposPath: reposPath,
	}
}

// RepositoryPath returns the path to a repository.
func (s *Client) RepositoryPath(owner, repo string) string {
	return filepath.Join(s.reposPath, owner, repo+".git")
}

// OpenRepository opens an existing Git repository
func (s *Client) OpenRepository(owner, repo string) (*git.Repository, error) {
	return git.PlainOpen(s.RepositoryPath(owner, repo))
}

// CloneBare clones a repository as a bare repository.
func (s *Client) CloneBare(ctx context.Context, owner, repo, url string) (*git.Repository, error) {
	repoPath := s.RepositoryPath(owner, repo)
	if err := os.MkdirAll(filepath.Dir(repoPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}
	return git.PlainCloneContext(ctx, repoPath, true, &git.CloneOptions{
		URL: url,
	})
}

// InitRepository initializes a new Git repository
func (s *Client) InitRepository(owner, repo string) error {
	repoPath := s.RepositoryPath(owner, repo)

	// Create directory
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		return err
	}

	// Initialize bare repository
	r, err := git.PlainInit(repoPath, true)
	if err != nil {
		return err
	}

	// Set default branch to "main" (go-git defaults to "master")
	if err := r.Storer.SetReference(plumbing.NewSymbolicReference(plumbing.HEAD, "refs/heads/main")); err != nil {
		return fmt.Errorf("failed to set default branch: %w", err)
	}

	return nil
}

// FileEntry represents a file or directory in the repository
type FileEntry struct {
	Name       string      `json:"name"`
	Path       string      `json:"path"`
	Type       string      `json:"type"` // "file" or "dir"
	Size       int64       `json:"size"`
	Mode       string      `json:"mode"`
	LastCommit *CommitInfo `json:"lastCommit,omitempty"`
}

// CommitInfo represents commit information
type CommitInfo struct {
	ID        string       `json:"id"`
	Message   string       `json:"message"`
	Author    string       `json:"author"`
	Email     string       `json:"email"`
	Timestamp time.Time    `json:"timestamp"`
	Files     []FileChange `json:"files,omitempty"`
}

// PushCommitInfo is a lightweight commit info without file changes
type PushCommitInfo struct {
	ID      string    `json:"id"`
	Message string    `json:"message"`
	Author  string    `json:"author"`
	Email   string    `json:"email"`
	Time    time.Time `json:"time"`
}

// RefUpdate records a single ref change during push
type RefUpdate struct {
	RefName     string           `json:"refName"`
	BranchName  string           `json:"branchName"`
	OldSHA      string           `json:"oldSha"`
	NewSHA      string           `json:"newSha"`
	Commits     []PushCommitInfo `json:"commits"`
	IsNewBranch bool             `json:"isNewBranch"`
	IsDeleted   bool             `json:"isDeleted"`
}

// PushDetail is stored in Activity.AdditionalInfo as JSON
type PushDetail struct {
	RefUpdates []RefUpdate `json:"refUpdates"`
}

// FileContent represents file content
type FileContent struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Content  string `json:"content"`
	Size     int64  `json:"size"`
	IsBinary bool   `json:"isBinary"`
}

// FileChange represents a file change in a commit
type FileChange struct {
	Filename  string `json:"filename"`
	Status    string `json:"status"` // added, modified, deleted
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Patch     string `json:"patch,omitempty"`
}

// ProtectedBranchInfo represents protected branch information
type ProtectedBranchInfo struct {
	BranchName string `json:"branchName"`
	Status     string `json:"status"`
}

// BranchInfo represents branch information
type BranchInfo struct {
	Name      string `json:"name"`
	CommitId  string `json:"commitId"`
	IsDefault bool   `json:"isDefault"`
}

// AheadBehind represents the ahead/behind commit count between two branches.
type AheadBehind struct {
	Ahead  int `json:"ahead"`
	Behind int `json:"behind"`
}

// SearchResult represents a search result
type SearchResult struct {
	Path    string   `json:"path"`
	Matches []string `json:"matches"`
}

// CompareResult represents the comparison result between two refs
type CompareResult struct {
	Commits     []CommitInfo `json:"commits"`
	FileChanges []FileChange `json:"fileChanges"`
	BaseCommit  string       `json:"baseCommit"`
	HeadCommit  string       `json:"headCommit"`
	Mergeable   bool         `json:"mergeable"`
}

// IssueTemplate represents an issue template
type IssueTemplate struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// TagInfo represents tag information
type TagInfo struct {
	Name        string    `json:"name"`
	CommitId    string    `json:"commitId"`
	Message     string    `json:"message,omitempty"`
	Tagger      string    `json:"tagger,omitempty"`
	TagDate     time.Time `json:"tagDate,omitempty"`
	IsAnnotated bool      `json:"isAnnotated"`
}

// BlameLine represents a single line in blame output
type BlameLine struct {
	CommitID string `json:"commitId"`
	Author   string `json:"author"`
	Email    string `json:"email"`
	Date     string `json:"date"`
	LineNum  int    `json:"lineNum"`
	Content  string `json:"content"`
}

// BlameResult represents the complete blame output for a file
type BlameResult struct {
	Lines []BlameLine `json:"lines"`
}
