package git

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// ArchiveRepository creates an archive of the repository and writes it to the provided writer.
// This is a streaming implementation to avoid loading entire archives into memory.
func (s *Client) ArchiveRepository(w io.Writer, owner, repo, ref, format string) (string, error) {
	repoPath := s.RepositoryPath(owner, repo)

	// Open repository
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	// Get commit
	var commit *object.Commit
	if ref == "" || ref == "HEAD" {
		headRef, err := r.Head()
		if err != nil {
		// HEAD might be unborn, try to find any branch
		branchIter, bErr := r.Branches()
			if bErr != nil {
				return "", fmt.Errorf("failed to get HEAD: %w", err)
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
				return "", fmt.Errorf("failed to get HEAD: %w", err)
			}
			commit, err = r.CommitObject(fallbackHash)
			if err != nil {
				return "", fmt.Errorf("failed to get commit: %w", err)
			}
		} else {
			commit, err = r.CommitObject(headRef.Hash())
			if err != nil {
				return "", fmt.Errorf("failed to get commit: %w", err)
			}
		}
	} else {
		// Try as branch
		branchRef, err := r.Reference(plumbing.NewBranchReferenceName(ref), true)
		if err == nil {
			commit, err = r.CommitObject(branchRef.Hash())
			if err != nil {
				return "", fmt.Errorf("failed to get commit: %w", err)
			}
		} else {
			// Try as tag
			tagRef, err := r.Reference(plumbing.NewTagReferenceName(ref), true)
			if err == nil {
				commit, err = r.CommitObject(tagRef.Hash())
				if err != nil {
					return "", fmt.Errorf("failed to get commit: %w", err)
				}
			} else {
				return "", fmt.Errorf("reference not found: %s", ref)
			}
		}
	}

	// Get tree
	tree, err := commit.Tree()
	if err != nil {
		return "", fmt.Errorf("failed to get tree: %w", err)
	}

	// Create archive based on format
	switch format {
	case "zip":
		return s.createZipArchive(w, tree, repo)
	case "tar.gz", "tgz":
		return s.createTarGzArchive(w, tree, repo)
	default:
		return "", fmt.Errorf("unsupported archive format: %s", format)
	}
}

// createZipArchive creates a ZIP archive of the tree and writes it to the provided writer
func (s *Client) createZipArchive(w io.Writer, tree *object.Tree, repoName string) (string, error) {
	zipWriter := zip.NewWriter(w)

	// Walk through all files in the tree
	err := tree.Files().ForEach(func(f *object.File) error {
		// Get file content
		content, err := f.Contents()
		if err != nil {
			return err
		}

		// Create zip file header
		header := &zip.FileHeader{
			Name:   f.Name,
			Method: zip.Deflate,
		}
		header.SetModTime(time.Now())

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		_, err = writer.Write([]byte(content))
		return err
	})

	if err != nil {
		return "", fmt.Errorf("failed to create zip archive: %w", err)
	}

	if err := zipWriter.Close(); err != nil {
		return "", fmt.Errorf("failed to close zip writer: %w", err)
	}

	filename := fmt.Sprintf("%s.zip", repoName)
	return filename, nil
}

// createTarGzArchive creates a TAR.GZ archive of the tree and writes it to the provided writer
func (s *Client) createTarGzArchive(w io.Writer, tree *object.Tree, repoName string) (string, error) {
	gzWriter := gzip.NewWriter(w)
	tarWriter := tar.NewWriter(gzWriter)

	// Walk through all files in the tree
	err := tree.Files().ForEach(func(f *object.File) error {
		// Get file content
		content, err := f.Contents()
		if err != nil {
			return err
		}

		contentBytes := []byte(content)

		// Create tar header
		header := &tar.Header{
			Name:    f.Name,
			Size:    int64(len(contentBytes)),
			Mode:    int64(f.Mode),
			ModTime: time.Now(),
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		_, err = tarWriter.Write(contentBytes)
		return err
	})

	if err != nil {
		return "", fmt.Errorf("failed to create tar.gz archive: %w", err)
	}

	if err := tarWriter.Close(); err != nil {
		return "", fmt.Errorf("failed to close tar writer: %w", err)
	}

	if err := gzWriter.Close(); err != nil {
		return "", fmt.Errorf("failed to close gzip writer: %w", err)
	}

	filename := fmt.Sprintf("%s.tar.gz", repoName)
	return filename, nil
}
