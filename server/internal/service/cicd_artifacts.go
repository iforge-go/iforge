package service

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"iforge/iforge/internal/git"
	"iforge/iforge/internal/model"
)

// ============================================================================
// Artifacts + Cache (Milestone 3)
//
// Artifacts: Job outputs shared within a pipeline (subsequent stages can download artifacts from previous stages)
// Cache: Dependency cache reused across pipelines (e.g., node_modules, .cache/go-build)
//
// Storage layout:
//   {dataDir}/cicd-artifacts/pipeline-{id}/job-{id}/...   (artifact files)
//   {dataDir}/cicd-cache/{owner}/{repo}/{cache-key}/...   (cache files)
// ============================================================================

// ArtifactsConfig is the parsed structure of job.Artifacts JSON
type ArtifactsConfig struct {
	Paths    []string `json:"paths"`
	ExpireIn string   `json:"expire_in"` // e.g., "1 week", "3 days", "1 hour"
}

// CacheConfig is the parsed structure of job.Cache JSON
type CacheConfig struct {
	Paths []string `json:"paths"`
	Key   string   `json:"key"` // Cache key; caches with the same key are reused across pipelines
}

// uploadArtifacts collects files matching artifacts.paths after job success,
// copies them to the storage directory, and creates Artifact records in the DB.
func (e *Executor) uploadArtifacts(jobID int64, workDir string, job model.Job) {
	if job.Artifacts == "" || job.Artifacts == "{}" {
		return
	}
	var cfg ArtifactsConfig
	if err := json.Unmarshal([]byte(job.Artifacts), &cfg); err != nil {
		e.appendLog(jobID, -1, fmt.Sprintf("WARNING: parse artifacts config failed: %v", err), "stderr")
		return
	}
	if len(cfg.Paths) == 0 {
		return
	}

	var pipeline model.Pipeline
	if err := e.db.First(&pipeline, job.PipelineID).Error; err != nil {
		e.appendLog(jobID, -1, fmt.Sprintf("WARNING: load pipeline for artifacts: %v", err), "stderr")
		return
	}

	storageBase := filepath.Join(e.artifactsDir(),
		fmt.Sprintf("pipeline-%d", pipeline.ID),
		fmt.Sprintf("job-%d", jobID))

	var expiresAt *time.Time
	if cfg.ExpireIn != "" {
		if d := parseDuration(cfg.ExpireIn); d > 0 {
			t := time.Now().Add(d)
			expiresAt = &t
		}
	}

	// Artifact size limits
	const (
		maxSingleFileSize = 100 * 1024 * 1024      // 100MB per file
		maxTotalSize      = 1 * 1024 * 1024 * 1024 // 1GB total
	)

	count := 0
	var totalSize int64
	for _, pattern := range cfg.Paths {
		srcPath := filepath.Join(workDir, pattern)
		matches, err := filepath.Glob(srcPath)
		if err != nil || len(matches) == 0 {
			e.appendLog(jobID, -1, fmt.Sprintf("WARNING: artifact pattern %q matched nothing", pattern), "stderr")
			continue
		}
		for _, src := range matches {
			relPath, err := filepath.Rel(workDir, src)
			if err != nil {
				continue
			}

			// Check individual file size
			if info, err := os.Stat(src); err == nil && !info.IsDir() {
				if info.Size() > maxSingleFileSize {
					e.appendLog(jobID, -1, fmt.Sprintf("WARNING: skip artifact %q (%s) exceeds max single file size (%s)",
						relPath, formatSize(info.Size()), formatSize(maxSingleFileSize)), "stderr")
					continue
				}
			}

			// Check total size limit
			if totalSize >= maxTotalSize {
				e.appendLog(jobID, -1, fmt.Sprintf("WARNING: skip artifact %q, total size exceeds limit (%s)",
					relPath, formatSize(maxTotalSize)), "stderr")
				break
			}

			dst := filepath.Join(storageBase, relPath)

			var size int64
			if info, err := os.Stat(src); err == nil && info.IsDir() {
				size, err = copyDir(src, dst)
			} else {
				size, err = copyFile(src, dst)
			}
			if err != nil {
				e.appendLog(jobID, -1, fmt.Sprintf("WARNING: collect artifact %q failed: %v", relPath, err), "stderr")
				continue
			}

			// Check total size again after copying
			if totalSize+size > maxTotalSize {
				e.appendLog(jobID, -1, fmt.Sprintf("WARNING: skip artifact %q (%s), would exceed total size limit (%s)",
					relPath, formatSize(size), formatSize(maxTotalSize)), "stderr")
				// Remove copied file
				_ = os.RemoveAll(dst)
				continue
			}

			artifact := &model.Artifact{
				JobID:       jobID,
				Name:        relPath,
				Path:        relPath,
				Size:        size,
				StoragePath: dst,
				ExpiresAt:   expiresAt,
				CreatedAt:   time.Now(),
			}
			e.db.Create(artifact)
			count++
			totalSize += size
		}
	}

	if count > 0 {
		e.appendLog(jobID, -1, fmt.Sprintf("Uploaded %d artifact(s) (%s)", count, formatSize(totalSize)), "stdout")
	}
}

// downloadArtifacts downloads artifacts from successful jobs in previous stages of the same pipeline before the job starts.
func (e *Executor) downloadArtifacts(jobID int64, workDir string, pipelineID int64, currentStage string) {
	// Find successful jobs in previous stages (ordered by stage name, before current stage)
	var prevJobs []model.Job
	if err := e.db.Where("pipeline_id = ? AND stage_name < ? AND status = ?",
		pipelineID, currentStage, model.JobStatusSuccess).
		Order("stage_name ASC, job_name ASC").Find(&prevJobs).Error; err != nil {
		return
	}

	count := 0
	for _, prevJob := range prevJobs {
		var artifacts []model.Artifact
		if err := e.db.Where("job_id = ?", prevJob.ID).Find(&artifacts).Error; err != nil {
			continue
		}
		for _, art := range artifacts {
			if _, err := os.Stat(art.StoragePath); os.IsNotExist(err) {
				continue // storage file expired or cleaned up
			}
			dst := filepath.Join(workDir, art.Path)
			if info, err := os.Stat(art.StoragePath); err == nil && info.IsDir() {
				if _, err := copyDir(art.StoragePath, dst); err == nil {
					count++
				}
			} else {
				if _, err := copyFile(art.StoragePath, dst); err == nil {
					count++
				}
			}
		}
	}

	if count > 0 {
		e.appendLog(jobID, -1, fmt.Sprintf("Restored %d artifact(s) from previous stages", count), "stdout")
	}
}

// uploadCache collects files matching cache.paths after job success and copies them to cache storage.
// Cache is stored by owner/repo/key and reused across pipelines.
func (e *Executor) uploadCache(jobID int64, workDir string, job model.Job, owner, repo string) {
	if job.Cache == "" || job.Cache == "{}" {
		return
	}
	var cfg CacheConfig
	if err := json.Unmarshal([]byte(job.Cache), &cfg); err != nil {
		e.appendLog(jobID, -1, fmt.Sprintf("WARNING: parse cache config failed: %v", err), "stderr")
		return
	}
	if len(cfg.Paths) == 0 {
		return
	}

	cacheKey := cfg.Key
	if cacheKey == "" {
		cacheKey = "default"
	}

	storageBase := filepath.Join(e.cacheDir(), owner, repo, cacheKey)

	// Clear old cache before writing (ensures consistency)
	_ = os.RemoveAll(storageBase)
	_ = os.MkdirAll(storageBase, 0755)

	count := 0
	var totalSize int64
	for _, pattern := range cfg.Paths {
		srcPath := filepath.Join(workDir, pattern)
		matches, err := filepath.Glob(srcPath)
		if err != nil || len(matches) == 0 {
			continue
		}
		for _, src := range matches {
			relPath, err := filepath.Rel(workDir, src)
			if err != nil {
				continue
			}
			dst := filepath.Join(storageBase, relPath)

			var size int64
			if info, err := os.Stat(src); err == nil && info.IsDir() {
				size, err = copyDir(src, dst)
			} else {
				size, err = copyFile(src, dst)
			}
			if err != nil {
				continue
			}
			count++
			totalSize += size
		}
	}

	if count > 0 {
		e.appendLog(jobID, -1, fmt.Sprintf("Cached %d path(s) (%s) with key %q", count, formatSize(totalSize), cacheKey), "stdout")
	}
}

// downloadCache restores cache.paths from cache storage to workDir before job starts.
func (e *Executor) downloadCache(jobID int64, workDir string, job model.Job, owner, repo string) {
	if job.Cache == "" || job.Cache == "{}" {
		return
	}
	var cfg CacheConfig
	if err := json.Unmarshal([]byte(job.Cache), &cfg); err != nil {
		return
	}
	if len(cfg.Paths) == 0 {
		return
	}

	cacheKey := cfg.Key
	if cacheKey == "" {
		cacheKey = "default"
	}

	storageBase := filepath.Join(e.cacheDir(), owner, repo, cacheKey)
	if _, err := os.Stat(storageBase); os.IsNotExist(err) {
		e.appendLog(jobID, -1, fmt.Sprintf("Cache miss (key=%q)", cacheKey), "stdout")
		return
	}

	// Restore cache files to workDir
	count := 0
	var totalSize int64
	err := filepath.Walk(storageBase, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(storageBase, path)
		if err != nil {
			return nil
		}
		dst := filepath.Join(workDir, relPath)
		size, err := copyFile(path, dst)
		if err != nil {
			return nil
		}
		count++
		totalSize += size
		return nil
	})
	if err == nil && count > 0 {
		e.appendLog(jobID, -1, fmt.Sprintf("Cache hit (key=%q): restored %d file(s) (%s)", cacheKey, count, formatSize(totalSize)), "stdout")
	}
}

// artifactsDir returns the artifact storage root directory
func (e *Executor) artifactsDir() string {
	return filepath.Join(e.dataDir, "cicd-artifacts")
}

// cacheDir returns the cache storage root directory
func (e *Executor) cacheDir() string {
	return filepath.Join(e.dataDir, "cicd-cache")
}

// SetDataDir injects the data directory (used for artifact/cache storage)
func (e *Executor) SetDataDir(dir string) {
	e.dataDir = dir
}

// ============================================================================
// Helper functions
// ============================================================================

// copyFile copies a single file and returns its size
func copyFile(src, dst string) (int64, error) {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return 0, err
	}
	srcFile, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer dstFile.Close()

	return io.Copy(dstFile, srcFile)
}

// copyDir recursively copies a directory and returns the total size of all files
func copyDir(src, dst string) (int64, error) {
	var totalSize int64
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)
		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}
		size, err := copyFile(path, dstPath)
		if err != nil {
			return err
		}
		totalSize += size
		return nil
	})
	return totalSize, err
}

// parseDuration parses time strings like "1 week", "3 days", "1 hour"
func parseDuration(s string) time.Duration {
	s = strings.TrimSpace(strings.ToLower(s))
	parts := strings.Fields(s)
	if len(parts) != 2 {
		return 0
	}
	n, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0
	}
	switch parts[1] {
	case "second", "seconds", "sec", "secs", "s":
		return time.Duration(n) * time.Second
	case "minute", "minutes", "min", "mins":
		return time.Duration(n) * time.Minute
	case "hour", "hours", "h":
		return time.Duration(n) * time.Hour
	case "day", "days", "d":
		return time.Duration(n) * 24 * time.Hour
	case "week", "weeks", "w":
		return time.Duration(n) * 7 * 24 * time.Hour
	case "month", "months":
		return time.Duration(n) * 30 * 24 * time.Hour
	}
	return 0
}

// formatSize formats bytes into human-readable string
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// Compile-time check to ensure git package is referenced (used by logger in appendLog)
var _ = git.GetLogger
