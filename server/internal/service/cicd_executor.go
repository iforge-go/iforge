package service

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"iforge/iforge/internal/git"
	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

// ============================================================================
// Executor - Built-in Job Executor
// ============================================================================

// Executor is responsible for scheduling and executing pending jobs
//
// Current implementation: Built-in Shell Executor (forks subprocess in main process to run shell scripts)
// Future milestone: Add Docker Executor (container isolation)
//
// Scheduling strategy (DAG dependency scheduling):
//   - Worker pool for parallel execution (default 4 workers, configurable via IFORGE_CI_WORKERS env var)
//   - SchedulePipeline only schedules pending jobs with empty needs or all needs succeeded
//   - After each job completes, triggers scheduleReadyJobs to schedule the next batch (dependencies satisfied)
//   - Jobs with failed dependencies are marked as skipped (unless when:always)
//
// Note: Current implementation does not handle:
//   - Docker container isolation (implemented but optional)
type Executor struct {
	db            *gorm.DB
	secretService *SecretService
	gitClient     *git.Client
	queue         chan int64 // Job ID queue
	stopCh        chan struct{}
	wg            sync.WaitGroup
	workerCount   int // Worker pool size

	// Docker availability detection (lazy loaded, checked on first isDockerAvailable call)
	dockerChecked   bool
	dockerAvailable bool
	dockerMu        sync.Mutex

	// Log line number mutex (protects appendLog lineNo=-1 auto-increment during multi-worker parallel execution)
	logMu sync.Mutex

	// dataDir for artifact/cache storage (injected by Container)
	dataDir string

	// Notification services (optional, skipped if not injected; injected by Container)
	// When Executor's built-in job completes or pipeline reaches terminal state, triggers in-app notifications + email + Webhook
	notificationService *NotificationService
	mailService         *MailService
	webhookService      *WebhookService

	// Release services (optional, skipped if not injected; injected by Container)
	// When release job completes and pipeline.ref is a tag, automatically creates Release + uploads assets
	releaseService   *ReleaseService
	releaseUploadDir string // Release file upload root directory (e.g., "{dataDir}/_releases")
}

// NewExecutor creates an executor (not started; triggered by SchedulePipeline)
func NewExecutor(db *gorm.DB) *Executor {
	// Read worker pool size from environment variable, default 4
	workerCount := 4
	if envWorkers := os.Getenv("IFORGE_CI_WORKERS"); envWorkers != "" {
		if n, err := strconv.Atoi(envWorkers); err == nil && n > 0 {
			workerCount = n
		}
	}
	return &Executor{
		db:          db,
		queue:       make(chan int64, 100),
		stopCh:      make(chan struct{}),
		workerCount: workerCount,
	}
}

// SetSecretService injects SecretService (avoids circular dependency; called by Container after initialization)
func (e *Executor) SetSecretService(s *SecretService) {
	e.secretService = s
}

// SetGitClient injects GitClient (used to clone repository to working directory)
func (e *Executor) SetGitClient(c *git.Client) {
	e.gitClient = c
}

// SetNotificationService injects notification service (notifies triggerer when pipeline reaches terminal state)
func (e *Executor) SetNotificationService(n *NotificationService) { e.notificationService = n }

// SetMailService injects mail service (sends email to triggerer when pipeline reaches terminal state)
func (e *Executor) SetMailService(m *MailService) { e.mailService = m }

// SetWebhookService injects Webhook service (triggers repository webhook when pipeline reaches terminal state)
func (e *Executor) SetWebhookService(w *WebhookService) { e.webhookService = w }

// SetReleaseService injects Release service (auto-creates Release + uploads assets for release jobs)
// uploadDir is the release file upload root directory (same as ReleaseHandler, e.g., "{dataDir}/_releases")
func (e *Executor) SetReleaseService(s *ReleaseService, uploadDir string) {
	e.releaseService = s
	e.releaseUploadDir = uploadDir
}

// isDockerAvailable checks if Docker daemon is available (lazy loaded, checked only once)
//
// When job.Image is non-empty and Docker is available, use Docker Executor (container isolation);
// otherwise fall back to Shell Executor (run directly on host).
func (e *Executor) isDockerAvailable() bool {
	e.dockerMu.Lock()
	defer e.dockerMu.Unlock()
	if e.dockerChecked {
		return e.dockerAvailable
	}
	// Run `docker version` to check if Docker daemon is running
	cmd := exec.Command("docker", "version", "--format", "{{.Server.Version}}")
	if err := cmd.Run(); err == nil {
		e.dockerAvailable = true
		git.GetLogger().Info("CICD: Docker detected, jobs with image will run in Docker containers", nil)
	} else {
		e.dockerAvailable = false
		git.GetLogger().Info("CICD: Docker not available, all jobs will use shell executor", nil)
	}
	e.dockerChecked = true
	return e.dockerAvailable
}

// Start starts the worker pool (multiple worker goroutines process jobs in parallel)
func (e *Executor) Start() {
	git.GetLogger().Info(fmt.Sprintf("CICD: Starting executor with %d workers", e.workerCount), nil)
	for i := 0; i < e.workerCount; i++ {
		e.wg.Add(1)
		go e.worker()
	}
}

// Stop stops workers (called when Container shuts down)
func (e *Executor) Stop() {
	close(e.stopCh)
	e.wg.Wait()
}

// SchedulePipeline schedules all runnable pending jobs in a Pipeline
// Called asynchronously by CreatePipeline, and also after each job completes to trigger the next round
//
// DAG dependency scheduling: only schedules jobs with empty needs or all needs succeeded;
// jobs with failed dependencies are marked as skipped (unless when:always).
func (e *Executor) SchedulePipeline(pipelineID int64) {
	e.scheduleReadyJobs(pipelineID)
}

// ScheduleReadyJobs public method: triggers the next round of scheduling for a pipeline
// Called by PipelineService after external runner completes a job
func (e *Executor) ScheduleReadyJobs(pipelineID int64) {
	e.scheduleReadyJobs(pipelineID)
}

// scheduleReadyJobs schedules all pending jobs in a pipeline whose dependencies are satisfied
//
// For each pending job:
//   - needs is empty → enqueue directly
//   - all needs succeeded → enqueue
//   - needs has failed/canceled/skipped → mark job as skipped (except when:always)
//   - needs still has pending/running → skip, wait for next round
func (e *Executor) scheduleReadyJobs(pipelineID int64) {
	var jobs []model.Job
	if err := e.db.Where("pipeline_id = ? AND status = ?", pipelineID, model.JobStatusPending).
		Order("stage_name ASC, job_name ASC").Find(&jobs).Error; err != nil {
		git.GetLogger().Error(fmt.Sprintf("CICD: scheduleReadyJobs pipeline %d query failed: %v", pipelineID, err), nil)
		return
	}

	scheduled, skipped := 0, 0
	for _, job := range jobs {
		// Skip manual jobs (require manual trigger)
		if job.When == model.JobWhenManual {
			continue
		}

		needs := parseJobNeeds(job.Needs)
		if len(needs) == 0 {
			// No dependencies, schedule directly
			e.enqueueJob(job.ID, job.JobName)
			scheduled++
			continue
		}

		// Check dependency status
		allSuccess, hasFailed := e.checkNeedsStatus(pipelineID, needs)
		if allSuccess {
			e.enqueueJob(job.ID, job.JobName)
			scheduled++
		} else if hasFailed {
			// Dependency failed, mark as skipped (when:always still executes)
			if job.When == model.JobWhenAlways {
				e.enqueueJob(job.ID, job.JobName)
				scheduled++
			} else {
				e.markJobSkipped(job.ID, "dependency_failed")
				skipped++
			}
		}
		// Dependencies still running (allSuccess=false, hasFailed=false) → wait, don't schedule
	}
	if scheduled > 0 || skipped > 0 {
		git.GetLogger().Info(fmt.Sprintf("CICD: scheduleReadyJobs pipeline=%d scheduled=%d skipped=%d (of %d pending)",
			pipelineID, scheduled, skipped, len(jobs)), nil)
	}
}

// enqueueJob enqueues a job for execution (marks as failed if queue is full)
func (e *Executor) enqueueJob(jobID int64, jobName string) {
	select {
	case e.queue <- jobID:
	default:
		// Mark job as failed when queue is full instead of silently dropping
		git.GetLogger().Error(fmt.Sprintf("CICD: executor queue full, job %d (%s) marked as failed", jobID, jobName), nil)
		e.markJobFailed(jobID, "executor_queue_full", -1)
	}
}

// checkNeedsStatus checks the status of all jobs in needs
// Returns: (allSuccess: all dependencies succeeded, hasFailed: any dependency failed/canceled/skipped)
func (e *Executor) checkNeedsStatus(pipelineID int64, needs []string) (allSuccess bool, hasFailed bool) {
	return checkJobNeedsStatus(e.db, pipelineID, needs)
}

// checkJobNeedsStatus package-level function: checks the status of all jobs in needs
// Shared by Executor and RunnerService (ClaimJobForRunner) to ensure external runners also respect DAG dependencies
//
// Returns: (allSuccess: all dependencies succeeded, hasFailed: any dependency failed/canceled/skipped)
func checkJobNeedsStatus(db *gorm.DB, pipelineID int64, needs []string) (allSuccess bool, hasFailed bool) {
	if len(needs) == 0 {
		return true, false
	}
	var jobs []model.Job
	if err := db.Where("pipeline_id = ? AND job_name IN ?", pipelineID, needs).Find(&jobs).Error; err != nil {
		return false, false
	}
	if len(jobs) != len(needs) {
		// Some needs jobs don't exist (possibly YAML typo), treat as unsatisfied
		return false, false
	}
	allSuccess = true
	for _, j := range jobs {
		switch j.Status {
		case model.JobStatusSuccess:
			// ok
		case model.JobStatusFailed, model.JobStatusCanceled, model.JobStatusSkipped:
			hasFailed = true
			allSuccess = false
		default: // pending, running
			allSuccess = false
		}
	}
	return allSuccess, hasFailed
}

// markJobSkipped marks a job as skipped (when dependencies fail)
func (e *Executor) markJobSkipped(jobID int64, reason string) {
	e.db.Model(&model.Job{}).Where("id = ?", jobID).Updates(map[string]interface{}{
		"status":         model.JobStatusSkipped,
		"finished_at":    time.Now(),
		"failure_reason": reason,
	})
}

// parseJobNeeds parses job.Needs JSON field into a list of job names
// Empty JSON / null / "[]" all return nil
func parseJobNeeds(needsJSON string) []string {
	if needsJSON == "" || needsJSON == "null" {
		return nil
	}
	var needs []string
	if err := json.Unmarshal([]byte(needsJSON), &needs); err != nil {
		return nil
	}
	return needs
}

// ScheduleJob schedules a single job (used when manually triggering a manual job)
func (e *Executor) ScheduleJob(jobID int64) {
	select {
	case e.queue <- jobID:
	default:
	}
}

// worker main loop
func (e *Executor) worker() {
	defer e.wg.Done()
	for {
		select {
		case <-e.stopCh:
			return
		case jobID := <-e.queue:
			e.executeJob(jobID)
		}
	}
}

// executeJob executes a single job
func (e *Executor) executeJob(jobID int64) {
	git.GetLogger().Info(fmt.Sprintf("CICD: executeJob start jobID=%d", jobID), nil)
	// Load job
	var job model.Job
	if err := e.db.First(&job, jobID).Error; err != nil {
		git.GetLogger().Error(fmt.Sprintf("CICD: executeJob job %d not found: %v", jobID, err), nil)
		return
	}

	// Atomic claim: UPDATE cicd_job SET status='running' WHERE id=? AND status='pending'
	// Prevents built-in executor and external runner from claiming the same job simultaneously.
	// RowsAffected=0 means already claimed by external runner (or canceled), give up.
	claimResult := e.db.Model(&model.Job{}).
		Where("id = ? AND status = ?", jobID, model.JobStatusPending).
		Updates(map[string]interface{}{
			"status":     model.JobStatusRunning,
			"started_at": time.Now(),
		})
	if claimResult.Error != nil {
		git.GetLogger().Error(fmt.Sprintf("CICD: executeJob job %d claim failed: %v", jobID, claimResult.Error), nil)
		return
	}
	if claimResult.RowsAffected == 0 {
		git.GetLogger().Info(fmt.Sprintf("CICD: executeJob job %d (%s) already claimed by external runner, skip", jobID, job.JobName), nil)
		return
	}
	// Reload job to get latest state (including started_at, etc.)
	if err := e.db.First(&job, jobID).Error; err != nil {
		git.GetLogger().Error(fmt.Sprintf("CICD: executeJob job %d reload failed: %v", jobID, err), nil)
		return
	}
	git.GetLogger().Info(fmt.Sprintf("CICD: executeJob job %d (%s, stage=%s) marked running", jobID, job.JobName, job.StageName), nil)

	// Load pipeline (to get commit_sha and other context)
	var pipeline model.Pipeline
	if err := e.db.First(&pipeline, job.PipelineID).Error; err != nil {
		e.markJobFailed(jobID, "pipeline not found", -1)
		return
	}

	// Write log: start
	e.appendLog(jobID, 0, fmt.Sprintf("Starting job %q (stage=%s) on commit %s",
		job.JobName, job.StageName, truncateSHA(pipeline.CommitSHA)), "stdout")
	e.appendLog(jobID, 1, fmt.Sprintf("Image: %s", fallback(job.Image, "(shell executor, no image)")), "stdout")
	e.appendLog(jobID, 2, fmt.Sprintf("Triggered by: %s via %s", pipeline.TriggeredBy, pipeline.TriggerEvent), "stdout")
	e.appendLog(jobID, 3, "", "stdout")

	// If job has environment configured, create Deployment record in in_progress state when execution starts
	// (updated to success/failed based on exitCode when execution ends)
	deployment := e.createDeploymentInProgress(pipeline, job)

	// Prepare working directory
	workDir, cleanup, err := e.prepareWorkDir(pipeline, job)
	if err != nil {
		e.appendLog(jobID, -1, fmt.Sprintf("ERROR: failed to prepare work dir: %v", err), "stderr")
		e.markJobFailed(jobID, "workdir_preparation_failed", -1)
		// Also update deployment status on failure
		e.finalizeDeployment(deployment, model.DeploymentStatusFailed, ptrStr("workdir_preparation_failed"))
		git.GetLogger().Error(fmt.Sprintf("CICD: executeJob job %d (%s) prepareWorkDir failed: %v", jobID, job.JobName, err), nil)
		return
	}
	defer cleanup()

	e.appendLog(jobID, -1, fmt.Sprintf("Working directory: %s", workDir), "stdout")
	e.appendLog(jobID, -1, "", "stdout")

	// Restore cache (reused across pipelines, e.g., node_modules, go-build cache)
	if e.dataDir != "" {
		e.downloadCache(jobID, workDir, job, pipeline.UserName, pipeline.RepositoryName)
		// Download artifacts from previous stages (passed within the same pipeline)
		e.downloadArtifacts(jobID, workDir, pipeline.ID, job.StageName)
	}

	// Parse environment variables
	envVars := e.buildEnvVars(pipeline, job)
	envs := envMapToSlice(envVars)

	// Set timeout control (default 3600 seconds = 1 hour, 0 means no limit)
	timeout := job.Timeout
	if timeout <= 0 {
		timeout = 3600
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	// Execute script
	git.GetLogger().Info(fmt.Sprintf("CICD: executeJob job %d (%s) running script (%d lines, timeout=%ds)", jobID, job.JobName, len(strings.Split(job.Script, "\n")), timeout), nil)
	var exitCode int
	if job.Image != "" && e.isDockerAvailable() {
		e.appendLog(jobID, -1, fmt.Sprintf("Running in Docker container (image: %s, timeout: %ds)", job.Image, timeout), "stdout")
		exitCode = e.runScriptInDocker(ctx, jobID, workDir, job.Image, job.Script, envs)
	} else {
		if job.Image != "" {
			e.appendLog(jobID, -1, fmt.Sprintf("Docker not available, falling back to shell executor (image %s ignored)", job.Image), "stdout")
		}
		exitCode = e.runScriptInShell(ctx, jobID, workDir, job.Script, envs)
	}

	// Check if timed out
	if ctx.Err() == context.DeadlineExceeded {
		e.appendLog(jobID, -1, fmt.Sprintf("Job %q timed out after %d seconds", job.JobName, timeout), "stderr")
		e.markJobFailed(jobID, "timeout", -1)
		return
	}

	git.GetLogger().Info(fmt.Sprintf("CICD: executeJob job %d (%s) finished exitCode=%d", jobID, job.JobName, exitCode), nil)

	// Write completion log
	e.appendLog(jobID, -1, "", "stdout")
	if exitCode == 0 {
		e.appendLog(jobID, -1, fmt.Sprintf("Job %q completed successfully", job.JobName), "stdout")
	} else {
		e.appendLog(jobID, -1, fmt.Sprintf("Job %q failed with exit code %d", job.JobName, exitCode), "stderr")
	}

	// Update status
	if exitCode == 0 {
		e.db.Model(&job).Updates(map[string]interface{}{
			"status":      model.JobStatusSuccess,
			"exit_code":   exitCode,
			"finished_at": time.Now(),
		})
		// Upload artifacts (downloadable by subsequent stages in the same pipeline)
		// Upload cache (reused across pipelines)
		if e.dataDir != "" {
			e.uploadArtifacts(jobID, workDir, job)
			e.uploadCache(jobID, workDir, job, pipeline.UserName, pipeline.RepositoryName)
		}
		// If job has release configured and pipeline.ref is a tag, auto-create Release + upload assets
		// (order doesn't matter with uploadArtifacts, since release assets use files from workDir, not uploaded artifacts)
		e.publishReleaseIfNeeded(jobID, workDir, pipeline, job)
		// Write a commit_status (makes it visible in MR Checks tab)
		e.writeCommitStatus(pipeline, job, "success")
		// Update Deployment status to success
		e.finalizeDeployment(deployment, model.DeploymentStatusSuccess, nil)
	} else {
		reason := "script_failure"
		e.db.Model(&job).Updates(map[string]interface{}{
			"status":         model.JobStatusFailed,
			"exit_code":      exitCode,
			"finished_at":    time.Now(),
			"failure_reason": reason,
		})
		e.writeCommitStatus(pipeline, job, "failure")
		// Update Deployment status to failed
		e.finalizeDeployment(deployment, model.DeploymentStatusFailed, ptrStr(reason))
	}

	// Schedule next batch of jobs (pending jobs with satisfied dependencies, or mark jobs with failed dependencies as skipped)
	e.scheduleReadyJobs(job.PipelineID)
	// Asynchronously refresh pipeline status
	go e.refreshPipelineStatus(job.PipelineID)
}

// publishReleaseIfNeeded auto-creates Release + uploads assets after release job succeeds
//
// Trigger conditions (all must be met to execute):
//  1. Executor has releaseService injected (otherwise skip, e.g., external runner scenario)
//  2. job.Release configuration is non-empty (YAML has release: field)
//  3. pipeline.Ref starts with "refs/tags/" (only tag push triggered pipelines create releases)
//
// Process:
//  1. Parse job.Release (JSON) into ReleaseYAML
//  2. Variable substitution: $CI_COMMIT_TAG / $TAG → tag name
//  3. Call ReleaseService.CreateReleaseIfNotExists (idempotent)
//  4. Iterate ReleaseYAML.Assets globs, match files under workDir
//  5. Call ReleaseService.AttachAssetFromFile for each matched file
//  6. Write logs to job log throughout; failures don't block pipeline (only log errors)
func (e *Executor) publishReleaseIfNeeded(jobID int64, workDir string, pipeline model.Pipeline, job model.Job) {
	if e.releaseService == nil || job.Release == "" {
		return
	}

	// Only create releases on tag-triggered pipelines
	const tagsPrefix = "refs/tags/"
	if !strings.HasPrefix(pipeline.Ref, tagsPrefix) {
		e.appendLog(jobID, -1, "Release: skipped (pipeline is not triggered by a tag push)", "stdout")
		return
	}
	tagName := strings.TrimPrefix(pipeline.Ref, tagsPrefix)

	// Parse ReleaseYAML
	var cfg ReleaseYAML
	if err := json.Unmarshal([]byte(job.Release), &cfg); err != nil {
		e.appendLog(jobID, -1, fmt.Sprintf("Release: failed to parse release config: %v", err), "stderr")
		return
	}

	e.appendLog(jobID, -1, "", "stdout")
	e.appendLog(jobID, -1, fmt.Sprintf("Release: publishing release for tag %q", tagName), "stdout")

	// Variable substitution: $CI_COMMIT_TAG / $TAG in Release.Name / Description → tagName
	relName := replaceReleaseVars(cfg.Name, tagName)
	if relName == "" {
		relName = "Release " + tagName
	}
	relDesc := replaceReleaseVars(cfg.Description, tagName)

	// Create Release (idempotent)
	release, err := e.releaseService.CreateReleaseIfNotExists(
		pipeline.UserName, pipeline.RepositoryName,
		tagName, relName, relDesc, pipeline.TriggeredBy,
	)
	if err != nil {
		e.appendLog(jobID, -1, fmt.Sprintf("Release: failed to create release: %v", err), "stderr")
		return
	}
	e.appendLog(jobID, -1, fmt.Sprintf("Release: created release %q (tag=%s, author=%s)", release.Name, tagName, release.Author), "stdout")

	// Upload assets (iterate globs, match files under workDir)
	if len(cfg.Assets) == 0 {
		e.appendLog(jobID, -1, "Release: no assets configured, done", "stdout")
		return
	}

	uploaded, failed := 0, 0
	for _, pattern := range cfg.Assets {
		// Variable substitution: $CI_COMMIT_TAG / $TAG in pattern
		pattern = replaceReleaseVars(pattern, tagName)

		// Glob match files under workDir
		matches, err := filepath.Glob(filepath.Join(workDir, pattern))
		if err != nil {
			e.appendLog(jobID, -1, fmt.Sprintf("Release: invalid asset pattern %q: %v", pattern, err), "stderr")
			failed++
			continue
		}
		if len(matches) == 0 {
			e.appendLog(jobID, -1, fmt.Sprintf("Release: asset pattern %q matched no files", pattern), "stderr")
			failed++
			continue
		}

		for _, srcPath := range matches {
			// Skip directories
			info, err := os.Stat(srcPath)
			if err != nil || info.IsDir() {
				continue
			}

			fileName := filepath.Base(srcPath)
			asset, err := e.releaseService.AttachAssetFromFile(
				pipeline.UserName, pipeline.RepositoryName, tagName,
				e.releaseUploadDir, srcPath, fileName, pipeline.TriggeredBy,
			)
			if err != nil {
				e.appendLog(jobID, -1, fmt.Sprintf("Release: failed to attach asset %q: %v", fileName, err), "stderr")
				failed++
				continue
			}
			e.appendLog(jobID, -1, fmt.Sprintf("Release: attached asset %q (%d bytes)", fileName, asset.Size), "stdout")
			uploaded++
		}
	}

	if failed > 0 {
		e.appendLog(jobID, -1, fmt.Sprintf("Release: completed with %d asset(s) uploaded, %d failed", uploaded, failed), "stderr")
	} else {
		e.appendLog(jobID, -1, fmt.Sprintf("Release: completed, %d asset(s) uploaded", uploaded), "stdout")
	}
}

// replaceReleaseVars replaces variables in Release configuration
//
// Supports:
//   - $CI_COMMIT_TAG → tagName
//   - $TAG → tagName
func replaceReleaseVars(s, tagName string) string {
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "$CI_COMMIT_TAG", tagName)
	s = strings.ReplaceAll(s, "$TAG", tagName)
	return s
}

// prepareWorkDir prepares the working directory
//
// Creates .iforge-ci/<pipeline>/<job>/ subdirectory in system temp directory,
// then clones the repository and checks out to the pipeline's commit SHA.
// This allows job scripts to execute in repository context (e.g., cd api && go build).
func (e *Executor) prepareWorkDir(pipeline model.Pipeline, job model.Job) (string, func(), error) {
	base := filepath.Join(os.TempDir(), "iforge-ci", fmt.Sprintf("pipeline-%d", pipeline.ID), fmt.Sprintf("job-%d", job.ID))
	// Clean existing working directory (e.g., old directory may remain during retry)
	// Avoids CloneToWorkDir failure due to existing directory
	if err := os.RemoveAll(base); err != nil {
		return "", nil, fmt.Errorf("clean work dir: %w", err)
	}
	if err := os.MkdirAll(base, 0755); err != nil {
		return "", nil, err
	}
	// Clone repository to working directory and checkout to specified commit
	if e.gitClient != nil && pipeline.CommitSHA != "" {
		if err := e.gitClient.CloneToWorkDir(
			pipeline.UserName, pipeline.RepositoryName,
			pipeline.CommitSHA, base,
		); err != nil {
			return "", nil, fmt.Errorf("clone repo: %w", err)
		}
	}
	cleanup := func() {
		// Clean working directory by default; set IFORGE_CI_KEEP_WORKDIR=true to keep for debugging
		if os.Getenv("IFORGE_CI_KEEP_WORKDIR") != "true" {
			if err := os.RemoveAll(base); err != nil {
				git.GetLogger().Warn(fmt.Sprintf("Failed to cleanup work dir %s: %v", base, err), nil)
			}
		} else {
			git.GetLogger().Info(fmt.Sprintf("Keeping work dir for debugging: %s", base), nil)
		}
	}
	return base, cleanup, nil
}

// buildEnvVars builds environment variables for job execution
// Delegates to shared function BuildJobEnvVars (shared by built-in executor and external runner for consistency)
func (e *Executor) buildEnvVars(pipeline model.Pipeline, job model.Job) map[string]string {
	return BuildJobEnvVars(e.secretService, pipeline, job)
}

// runScriptInShell executes job's multi-line script directly on host (Shell Executor).
//
// Key: entire script executed as single process (not line-by-line), so cd/environment variables
// persist across lines. Windows uses command chaining; Unix uses bash -c "set -e\ncmd1\ncmd2".
// Stops on first line failure (set -e / && short-circuit).
//
// Security warning: Shell Executor executes user code directly on host, posing serious security risks.
// Production environments should disable it (set IFORGE_CI_ALLOW_SHELL_EXECUTOR=false) and use Docker Executor instead.
func (e *Executor) runScriptInShell(ctx context.Context, jobID int64, workDir, script string, envs []string) int {
	// Security check: Shell Executor disabled by default
	if os.Getenv("IFORGE_CI_ALLOW_SHELL_EXECUTOR") != "true" {
		e.appendLog(jobID, -1, "ERROR: Shell Executor is disabled for security reasons.", "stderr")
		e.appendLog(jobID, -1, "Set IFORGE_CI_ALLOW_SHELL_EXECUTOR=true to enable (NOT recommended for production).", "stderr")
		e.appendLog(jobID, -1, "Please use Docker Executor by specifying 'image:' in your pipeline YAML.", "stderr")
		return 1
	}

	if strings.TrimSpace(script) == "" {
		return 0
	}

	// Filter empty lines and comments, collect actual commands
	var commands []string
	for _, rawLine := range strings.Split(script, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		commands = append(commands, line)
		// Display each command (similar to terminal echo)
		e.appendLog(jobID, -1, "$ "+line, "stdout")
	}
	if len(commands) == 0 {
		return 0
	}

	// Build complete command for single execution (use passed ctx to control timeout)
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// Windows: chain commands with &&, stop on first line failure
		fullScript := strings.Join(commands, " && ")
		cmd = exec.CommandContext(ctx, "cmd", "/c", fullScript)
	} else {
		// Unix: set -e stops on first line failure, newline-separated
		fullScript := "set -e\n" + strings.Join(commands, "\n")
		cmd = exec.CommandContext(ctx, "bash", "-c", fullScript)
	}
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), envs...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		e.appendLog(jobID, -1, fmt.Sprintf("ERROR: %v", err), "stderr")
		return 1
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		e.appendLog(jobID, -1, fmt.Sprintf("ERROR: %v", err), "stderr")
		return 1
	}

	if err := cmd.Start(); err != nil {
		e.appendLog(jobID, -1, fmt.Sprintf("ERROR: %v", err), "stderr")
		return 1
	}

	// Read stdout/stderr in real-time (write to DB line by line)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		e.streamLogs(jobID, stdoutPipe, "stdout")
	}()
	go func() {
		defer wg.Done()
		e.streamLogs(jobID, stderrPipe, "stderr")
	}()

	err = cmd.Wait()
	wg.Wait()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			e.appendLog(jobID, -1, fmt.Sprintf("Command exited with code %d", exitErr.ExitCode()), "stderr")
			return exitErr.ExitCode()
		}
		e.appendLog(jobID, -1, fmt.Sprintf("ERROR: %v", err), "stderr")
		return 1
	}
	return 0
}

// runScriptInDocker executes job script inside Docker container (Docker Executor).
//
// Process:
//  1. Mount host workDir (with cloned repository) to container's /workspace
//  2. Pass environment variables via -e (CI_*, secrets, vars)
//  3. Execute in container with sh -c "set -e\n<commands>" (stops on first line failure)
//  4. --rm ensures container auto-cleanup after exit
//
// Logs streamed to DB (consistent with Shell Executor).
func (e *Executor) runScriptInDocker(ctx context.Context, jobID int64, workDir, image, script string, envs []string) int {
	if strings.TrimSpace(script) == "" {
		return 0
	}

	// Filter empty lines and comments, collect actual commands
	var commands []string
	for _, rawLine := range strings.Split(script, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		commands = append(commands, line)
		e.appendLog(jobID, -1, "$ "+line, "stdout")
	}
	if len(commands) == 0 {
		return 0
	}

	// Execute in container with sh -c, set -e stops on first line failure
	containerScript := "set -e\n" + strings.Join(commands, "\n")

	// Build docker run parameters
	// Windows paths need conversion to forward slashes (supported by Docker Desktop)
	dockerWorkDir := workDir
	if runtime.GOOS == "windows" {
		dockerWorkDir = strings.ReplaceAll(workDir, "\\", "/")
	}

	// Generate random suffix to avoid container name conflicts (multiple workers may create containers simultaneously during parallel execution)
	randomSuffix := fmt.Sprintf("%x", time.Now().UnixNano()%1000000)
	containerName := fmt.Sprintf("iforge-job-%d-%s", jobID, randomSuffix)

	args := []string{
		"run", "--rm",
		"--name", containerName,
		"-v", fmt.Sprintf("%s:/workspace", dockerWorkDir),
		"-w", "/workspace",
		// Resource limits: prevent single job from exhausting host resources
		"--memory=2g",      // Limit memory to 2GB
		"--memory-swap=2g", // Disable swap
		"--cpus=2",         // Limit to 2 CPUs
		"--pids-limit=256", // Limit process count
		// Network isolation: disable network by default to prevent internal network access
		"--network=none",
	}

	// Pass environment variables
	for _, env := range envs {
		args = append(args, "-e", env)
	}

	// Image + execution command
	args = append(args, image, "sh", "-c", containerScript)

	// Use passed ctx to control timeout
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Env = os.Environ() // docker CLI itself doesn't need job's envs (already passed to container via -e)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		e.appendLog(jobID, -1, fmt.Sprintf("ERROR: %v", err), "stderr")
		return 1
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		e.appendLog(jobID, -1, fmt.Sprintf("ERROR: %v", err), "stderr")
		return 1
	}

	if err := cmd.Start(); err != nil {
		e.appendLog(jobID, -1, fmt.Sprintf("ERROR: failed to start docker container: %v", err), "stderr")
		return 1
	}

	// Read stdout/stderr in real-time (write to DB line by line)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		e.streamLogs(jobID, stdoutPipe, "stdout")
	}()
	go func() {
		defer wg.Done()
		e.streamLogs(jobID, stderrPipe, "stderr")
	}()

	err = cmd.Wait()
	wg.Wait()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			e.appendLog(jobID, -1, fmt.Sprintf("Docker container exited with code %d", exitErr.ExitCode()), "stderr")
			return exitErr.ExitCode()
		}
		e.appendLog(jobID, -1, fmt.Sprintf("ERROR: %v", err), "stderr")
		return 1
	}
	return 0
}

// streamLogs reads pipe in real-time and writes to DB line by line
func (e *Executor) streamLogs(jobID int64, r interface{ Read([]byte) (int, error) }, stream string) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		e.appendLog(jobID, -1, scanner.Text(), stream)
	}
}

// appendLog appends a log line (auto-increments to max+1 when lineNo = -1)
// Uses mutex to protect auto-increment of line numbers, preventing conflicts during multi-worker parallel execution
func (e *Executor) appendLog(jobID int64, lineNo int, content, stream string) {
	if lineNo < 0 {
		// Lock to protect line number auto-increment
		e.logMu.Lock()
		var maxLine int
		e.db.Model(&model.JobLog{}).Where("job_id = ?", jobID).
			Select("COALESCE(MAX(line_no), -1)").Row().Scan(&maxLine)
		lineNo = maxLine + 1
		e.logMu.Unlock()
	}
	log := &model.JobLog{
		JobID:     jobID,
		LineNo:    lineNo,
		Content:   content,
		Stream:    stream,
		Timestamp: time.Now(),
	}
	e.db.Create(log)
}

// markJobFailed marks job as failed
func (e *Executor) markJobFailed(jobID int64, reason string, exitCode int) {
	e.db.Model(&model.Job{}).Where("id = ?", jobID).Updates(map[string]interface{}{
		"status":         model.JobStatusFailed,
		"finished_at":    time.Now(),
		"exit_code":      exitCode,
		"failure_reason": reason,
	})
	// Query job's pipelineID, trigger next round of scheduling + refresh pipeline status
	// (fixes original bug: mistakenly used jobID as pipelineID, causing incorrect pipeline status)
	var job model.Job
	if err := e.db.Select("pipeline_id").First(&job, jobID).Error; err == nil {
		e.scheduleReadyJobs(job.PipelineID)
		go e.refreshPipelineStatus(job.PipelineID)
	}
}

// refreshPipelineStatus recalculates Pipeline status (consistent with PipelineService logic, but avoids circular dependency)
//
// When pipeline transitions from non-terminal state (pending/running) to terminal state (success/failed),
// triggers notifyPipelineFinished to send in-app notifications + emails + Webhooks.
func (e *Executor) refreshPipelineStatus(pipelineID int64) {
	// Query pipeline old status (used to determine if first time entering terminal state)
	var pipeline model.Pipeline
	if err := e.db.First(&pipeline, pipelineID).Error; err != nil {
		return
	}
	oldStatus := pipeline.Status

	var jobs []model.Job
	if err := e.db.Where("pipeline_id = ?", pipelineID).Find(&jobs).Error; err != nil {
		return
	}
	hasFailed := false
	hasRunning := false
	allDone := true
	for _, j := range jobs {
		switch j.Status {
		case model.JobStatusFailed, model.JobStatusCanceled:
			hasFailed = true
		case model.JobStatusPending, model.JobStatusRunning:
			hasRunning = true
			allDone = false
		}
	}
	var newStatus string
	switch {
	case hasFailed:
		newStatus = model.PipelineStatusFailed
	case hasRunning:
		newStatus = model.PipelineStatusRunning
	case allDone:
		newStatus = model.PipelineStatusSuccess
	default:
		newStatus = model.PipelineStatusRunning
	}
	updates := map[string]interface{}{"status": newStatus}
	if newStatus == model.PipelineStatusFailed || newStatus == model.PipelineStatusSuccess {
		now := time.Now()
		updates["finished_at"] = now
		if pipeline.StartedAt != nil {
			updates["duration"] = now.Sub(*pipeline.StartedAt).Milliseconds()
		}
	}
	e.db.Model(&model.Pipeline{}).Where("id = ?", pipelineID).Updates(updates)

	// Trigger notification when status first enters terminal state (success/failed) from non-terminal state
	if newStatus == model.PipelineStatusSuccess || newStatus == model.PipelineStatusFailed {
		if oldStatus != model.PipelineStatusSuccess && oldStatus != model.PipelineStatusFailed {
			pipeline.Status = newStatus
			notifyPipelineFinished(e.db, e.notificationService, e.mailService, e.webhookService, pipeline, newStatus)
		}
	}
}

// writeCommitStatus writes a CommitStatus record (makes it visible in MR Checks tab)
func (e *Executor) writeCommitStatus(pipeline model.Pipeline, job model.Job, state string) {
	// Reuse commit_status table (context uses "ci/iforge:<job_name>")
	context := "ci/iforge:" + job.JobName
	targetURL := fmt.Sprintf("/%s/%s/pipelines/%d/jobs/%d",
		pipeline.UserName, pipeline.RepositoryName, pipeline.ID, job.ID)

	cs := &model.CommitStatus{
		UserName:       pipeline.UserName,
		RepositoryName: pipeline.RepositoryName,
		CommitID:       pipeline.CommitSHA,
		Context:        context,
		State:          state,
		TargetURL:      &targetURL,
		Description:    ptrStr(fmt.Sprintf("Job %s %s", job.JobName, state)),
		UpdatedDate:    time.Now(),
		Creator:        "iforge-ci",
	}
	e.db.Create(cs)
}

// writeDeployment writes a Deployment record (when job has environment configured)
// Deprecated: kept for backward compatibility; new code should use createDeploymentInProgress + finalizeDeployment
func (e *Executor) writeDeployment(pipeline model.Pipeline, job model.Job) {
	d := e.createDeploymentInProgress(pipeline, job)
	if d == nil {
		return
	}
	e.finalizeDeployment(d, model.DeploymentStatusSuccess, nil)
}

// createDeploymentInProgress creates Deployment record in in_progress state when job starts executing
// Returns created deployment (for subsequent finalizeDeployment use); returns nil if job has no environment configured
func (e *Executor) createDeploymentInProgress(pipeline model.Pipeline, job model.Job) *model.Deployment {
	if job.Environment == nil || *job.Environment == "" {
		return nil
	}

	// Find or create environment
	var env model.Environment
	err := e.db.Where("user_name = ? AND repository_name = ? AND name = ?",
		pipeline.UserName, pipeline.RepositoryName, *job.Environment).First(&env).Error
	if err != nil {
		env = model.Environment{
			UserName:         pipeline.UserName,
			RepositoryName:   pipeline.RepositoryName,
			Name:             *job.Environment,
			DeploymentPolicy: "manual",
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}
		e.db.Create(&env)
	}

	d := &model.Deployment{
		EnvironmentID:  env.ID,
		UserName:       pipeline.UserName,
		RepositoryName: pipeline.RepositoryName,
		PipelineID:     &pipeline.ID,
		JobID:          &job.ID,
		CommitSHA:      pipeline.CommitSHA,
		Status:         model.DeploymentStatusInProgress,
		DeployedBy:     pipeline.TriggeredBy,
		DeployedAt:     time.Now(),
	}
	if err := e.db.Create(d).Error; err != nil {
		git.GetLogger().Error(fmt.Sprintf("CICD: createDeploymentInProgress failed: %v", err), nil)
		return nil
	}
	return d
}

// finalizeDeployment updates Deployment status to success/failed when job completes
// deployment can be nil (skipped when environment not configured)
func (e *Executor) finalizeDeployment(deployment *model.Deployment, status string, note *string) {
	if deployment == nil {
		return
	}
	now := time.Now()
	updates := map[string]interface{}{
		"status":      status,
		"finished_at": now,
	}
	if note != nil {
		updates["note"] = *note
	}
	if err := e.db.Model(&model.Deployment{}).Where("id = ?", deployment.ID).Updates(updates).Error; err != nil {
		git.GetLogger().Error(fmt.Sprintf("CICD: finalizeDeployment failed for deployment %d: %v", deployment.ID, err), nil)
	}
}

// ============================================================================
// Platform-specific command building
// ============================================================================

// buildCommandWithContext builds command with context
//
// Windows: cmd /c "<line>"
// Others:  bash -c "<line>"
func buildCommandWithContext(ctx context.Context, line string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "cmd", "/c", line)
	}
	return exec.CommandContext(ctx, "bash", "-c", line)
}

// ============================================================================
// Helper functions
// ============================================================================

func truncateSHA(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}

func fallback(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func ptrStr(s string) *string {
	return &s
}

func envMapToSlice(m map[string]string) []string {
	r := make([]string, 0, len(m))
	for k, v := range m {
		r = append(r, k+"="+v)
	}
	return r
}
