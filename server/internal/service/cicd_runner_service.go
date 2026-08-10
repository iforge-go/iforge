package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"iforge/iforge/internal/git"
	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

// ErrNoJobAvailable indicates no claimable job is currently available.
var ErrNoJobAvailable = errors.New("no job available")

// RegisterExternalRunner registers an external runner and returns the plaintext token (returned only once).
func (s *RunnerService) RegisterExternalRunner(name, runnerType, description, ipAddress string) (*model.Runner, string, error) {
	token, hash, err := GenerateRunnerToken()
	if err != nil {
		return nil, "", err
	}

	runner := &model.Runner{
		Name:      name,
		TokenHash: hash,
		// Default to offline on registration; transitions to online after the first heartbeat.
		// Prevents runners that were registered but never started from appearing online in the UI.
		Status:      model.RunnerStatusOffline,
		Type:        runnerType,
		Description: description,
		IsBuiltin:   false,
		MaxJobs:     1,
		IPAddress:   ipAddress,
		Version:     "0.1.0",
		CreatedAt:   time.Now(),
	}
	if err := s.db.Create(runner).Error; err != nil {
		return nil, "", err
	}
	return runner, token, nil
}

// ClaimJobResponse is the return structure for ClaimJobForRunner,
// containing all information a runner needs to execute a job.
type ClaimJobResponse struct {
	Job      *model.Job        `json:"job"`
	Pipeline *model.Pipeline   `json:"pipeline"`
	Envs     map[string]string `json:"envs"`
	// CloneURL is the HTTP URL for the runner to clone the repo (e.g. http://localhost:8081/owner/repo.git).
	CloneURL string `json:"cloneUrl"`
}

// ClaimJobForRunner atomically assigns a pending job to the specified runner.
//
// Uses an atomic UPDATE to prevent multiple runners from claiming the same job:
// UPDATE cicd_job SET status='running', runner_id=? WHERE id=? AND status='pending'
// If RowsAffected=0, another runner claimed it first; continue to the next job.
//
// DAG dependency: only claims jobs whose needs are empty or all succeeded;
// jobs with unmet dependencies are skipped (left for subsequent polls or
// the built-in executor's scheduleReadyJobs to handle).
func (s *RunnerService) ClaimJobForRunner(runnerID int64, secretService *SecretService, iforgeURL string) (*ClaimJobResponse, error) {
	// Fetch runner info to check repository scope.
	var runner model.Runner
	if err := s.db.First(&runner, runnerID).Error; err != nil {
		return nil, fmt.Errorf("runner not found: %w", err)
	}

	// Build query conditions: only find jobs the runner is allowed to execute.
	// JOINs the Pipeline table to filter by repository scope.
	query := s.db.Table("cicd_job").
		Select("cicd_job.*").
		Joins("JOIN cicd_pipeline ON cicd_job.pipeline_id = cicd_pipeline.id").
		Where("cicd_job.status = ?", model.JobStatusPending)

	if runner.UserName != nil && runner.RepositoryName != nil {
		// Repository-level runner: can only execute jobs for this repository.
		query = query.Where("cicd_pipeline.user_name = ? AND cicd_pipeline.repository_name = ?", *runner.UserName, *runner.RepositoryName)
	} else if runner.UserName != nil {
		// User-level runner: can execute all jobs for this user.
		query = query.Where("cicd_pipeline.user_name = ?", *runner.UserName)
	}
	// Global runner (both userName and repositoryName are nil): can execute all jobs.

	// Find matching pending jobs (ordered by creation time, oldest first).
	var jobs []model.Job
	if err := query.Order("cicd_job.created_at ASC").Limit(20).Find(&jobs).Error; err != nil {
		return nil, err
	}

	for _, job := range jobs {
		// DAG dependency check: only claim jobs whose needs have all succeeded.
		needs := parseJobNeeds(job.Needs)
		if len(needs) > 0 {
			allSuccess, _ := checkJobNeedsStatus(s.db, job.PipelineID, needs)
			if !allSuccess {
				continue
			}
		}

		// Atomic claim: only succeeds while status is still pending.
		result := s.db.Model(&model.Job{}).
			Where("id = ? AND status = ?", job.ID, model.JobStatusPending).
			Updates(map[string]interface{}{
				"status":     model.JobStatusRunning,
				"runner_id":  runnerID,
				"started_at": time.Now(),
			})
		if result.Error != nil {
			continue
		}
		if result.RowsAffected == 0 {
			// Claimed by another runner; try the next one.
			continue
		}

		// Claim succeeded; load the pipeline.
		var pipeline model.Pipeline
		if err := s.db.First(&pipeline, job.PipelineID).Error; err != nil {
			return nil, err
		}

		// Build environment variables.
		envs := BuildJobEnvVars(secretService, pipeline, job)

		// Build clone URL.
		cloneURL := iforgeURL + "/" + pipeline.UserName + "/" + pipeline.RepositoryName + ".git"

		return &ClaimJobResponse{
			Job:      &job,
			Pipeline: &pipeline,
			Envs:     envs,
			CloneURL: cloneURL,
		}, nil
	}

	return nil, ErrNoJobAvailable
}

// UpdateJobFromRunner handles job status updates reported by the runner.
func (s *RunnerService) UpdateJobFromRunner(jobID, runnerID int64, status string, exitCode *int, failureReason *string) error {
	// Verify the job was claimed by this runner.
	var job model.Job
	if err := s.db.First(&job, jobID).Error; err != nil {
		return ErrJobNotFound
	}
	if job.RunnerID == nil || *job.RunnerID != runnerID {
		return errors.New("job not assigned to this runner")
	}

	// Verify runner's repository-level permissions.
	var runner model.Runner
	if err := s.db.First(&runner, runnerID).Error; err != nil {
		return ErrRunnerNotFound
	}

	// Fetch the job's pipeline to check repository permissions.
	var pipeline model.Pipeline
	if err := s.db.First(&pipeline, job.PipelineID).Error; err != nil {
		return errors.New("pipeline not found")
	}

	// Check runner's repository scope.
	if runner.UserName != nil && runner.RepositoryName != nil {
		// Repository-level runner: can only operate on this repository's jobs.
		if pipeline.UserName != *runner.UserName || pipeline.RepositoryName != *runner.RepositoryName {
			return errors.New("runner not authorized for this repository")
		}
	} else if runner.UserName != nil {
		// User-level runner: can only operate on this user's jobs.
		if pipeline.UserName != *runner.UserName {
			return errors.New("runner not authorized for this user")
		}
	}
	// Global runner (both userName and repositoryName are nil): can operate on all jobs.

	updates := map[string]interface{}{
		"status":      status,
		"finished_at": time.Now(),
	}
	if exitCode != nil {
		updates["exit_code"] = *exitCode
	}
	if failureReason != nil {
		updates["failure_reason"] = *failureReason
	}

	return s.db.Model(&model.Job{}).Where("id = ?", jobID).Updates(updates).Error
}

// JobLogBatch is a single log entry in a batch upload from the runner.
type JobLogBatch struct {
	LineNo  int    `json:"lineNo"`
	Content string `json:"content"`
	Stream  string `json:"stream"` // stdout | stderr
}

// AppendJobLogsFromRunner handles batch log uploads from the runner.
func (s *RunnerService) AppendJobLogsFromRunner(jobID, runnerID int64, logs []JobLogBatch) error {
	// Verify job ownership.
	var job model.Job
	if err := s.db.First(&job, jobID).Error; err != nil {
		return ErrJobNotFound
	}
	if job.RunnerID == nil || *job.RunnerID != runnerID {
		return errors.New("job not assigned to this runner")
	}

	now := time.Now()
	for _, l := range logs {
		logEntry := &model.JobLog{
			JobID:     jobID,
			LineNo:    l.LineNo,
			Content:   l.Content,
			Stream:    l.Stream,
			Timestamp: now,
		}
		s.db.Create(logEntry)
	}
	return nil
}

// UpdateRunnerStatus updates the runner status (busy/online/offline) and refreshes the heartbeat timestamp.
func (s *RunnerService) UpdateRunnerStatus(runnerID int64, status string) error {
	now := time.Now()
	return s.db.Model(&model.Runner{}).Where("id = ?", runnerID).
		Updates(map[string]interface{}{
			"status":         status,
			"last_heartbeat": now,
		}).Error
}

// CheckStaleRunners marks runners that have not sent a heartbeat within the timeout as offline.
// Timeout threshold: 90 seconds (3x the 30-second heartbeat interval, tolerates network jitter).
func (s *RunnerService) CheckStaleRunners() error {
	threshold := time.Now().Add(-90 * time.Second)
	return s.db.Model(&model.Runner{}).
		Where("status = ? AND is_builtin = ?", "online", false).
		Where("last_heartbeat IS NULL OR last_heartbeat < ?", threshold).
		Update("status", "offline").Error
}

// StartStaleChecker starts a periodic goroutine that scans every 30 seconds.
func (s *RunnerService) StartStaleChecker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if err := s.CheckStaleRunners(); err != nil {
			git.GetLogger().Error("Failed to check stale runners: "+err.Error(), nil)
		}
	}
}

// GetRunnerByID retrieves a runner by its ID.
func (s *RunnerService) GetRunnerByID(runnerID int64) (*model.Runner, error) {
	var runner model.Runner
	if err := s.db.First(&runner, runnerID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRunnerNotFound
		}
		return nil, err
	}
	return &runner, nil
}

// DeleteRunner deletes an external runner (built-in runners cannot be deleted).
func (s *RunnerService) DeleteRunner(runnerID int64) error {
	var runner model.Runner
	if err := s.db.First(&runner, runnerID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRunnerNotFound
		}
		return err
	}
	if runner.IsBuiltin {
		return errors.New("cannot delete builtin runner")
	}
	return s.db.Delete(&model.Runner{}, runnerID).Error
}

// BuildJobEnvVars builds environment variables for job execution (shared by built-in executor and external runners).
func BuildJobEnvVars(secretService *SecretService, pipeline model.Pipeline, job model.Job) map[string]string {
	envs := map[string]string{
		"CI":                "true",
		"IFORGE":            "true",
		"CI_PIPELINE_ID":    fmt.Sprintf("%d", pipeline.ID),
		"CI_JOB_ID":         fmt.Sprintf("%d", job.ID),
		"CI_JOB_NAME":       job.JobName,
		"CI_JOB_STAGE":      job.StageName,
		"CI_PIPELINE_REF":   pipeline.Ref,
		"CI_COMMIT_SHA":     pipeline.CommitSHA,
		"CI_COMMIT_MESSAGE": pipeline.Message,
		"CI_REPO_OWNER":     pipeline.UserName,
		"CI_REPO_NAME":      pipeline.RepositoryName,
		"CI_TRIGGER_EVENT":  pipeline.TriggerEvent,
		"CI_TRIGGERED_BY":   pipeline.TriggeredBy,
	}

	// Inject job-level vars (parsed from JSON string).
	if job.Vars != "" && job.Vars != "{}" {
		var jobVars map[string]string
		if err := json.Unmarshal([]byte(job.Vars), &jobVars); err == nil {
			for k, v := range jobVars {
				envs[k] = v
			}
		}
	}

	// Inject secrets (repo-level AES-256-GCM encrypted at rest, decrypted at runtime).
	// Secrets have lower priority than job-level vars.
	if secretService != nil {
		if secrets, err := secretService.DecryptSecrets(pipeline.UserName, pipeline.RepositoryName); err == nil {
			for k, v := range secrets {
				if _, exists := envs[k]; !exists {
					envs[k] = v
				}
			}
		}
	}

	return envs
}
