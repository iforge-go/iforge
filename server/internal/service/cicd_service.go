package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"iforge/iforge/internal/model"

	"gopkg.in/yaml.v2"
	"gorm.io/gorm"
)

// Error definitions
var (
	ErrPipelineNotFound    = errors.New("pipeline not found")
	ErrJobNotFound         = errors.New("job not found")
	ErrRunnerNotFound      = errors.New("runner not found")
	ErrSecretNotFound      = errors.New("secret not found")
	ErrEnvironmentNotFound = errors.New("environment not found")
	ErrDeploymentNotFound  = errors.New("deployment not found")
	ErrInvalidYAML         = errors.New("invalid CI/CD YAML configuration")
	ErrNoRunnerAvailable   = errors.New("no runner available")
	ErrSecretKeyExists     = errors.New("secret key already exists")
	ErrEnvironmentExists   = errors.New("environment name already exists")
)

// YAML configuration parsing

// PipelineYAML represents the parsed structure of .iforge-ci.yml (standard CI YAML syntax)
//
// Example:
//
//	stages:
//	  - build
//	  - test
//	  - deploy
//
//	variables:
//	  GO_VERSION: "1.21"
//
//	build:
//	  stage: build
//	  image: golang:1.21
//	  script:
//	    - go build -o bin/iforge ./cmd/server
//	  artifacts:
//	    paths:
//	      - bin/
//	    expire_in: 1 week
//	  cache:
//	    paths:
//	      - .cache/go-build
//
//	test:
//	  stage: test
//	  script:
//	    - go test ./...
//
//	deploy:staging:
//	  stage: deploy
//	  environment:
//	    name: staging
//	  script:
//	    - ./deploy.sh staging
//	  only:
//	    - main
//
//	deploy:prod:
//	  stage: deploy
//	  environment:
//	    name: production
//	  script:
//	    - ./deploy.sh prod
//	  when: manual
//	  only:
//	    - tags
type PipelineYAML struct {
	Stages    []string               `yaml:"stages"`
	Variables map[string]string      `yaml:"variables"`
	Jobs      map[string]JobYAML     `yaml:"-"` // Populated after parsing: job_name -> config
	Raw       map[string]interface{} `yaml:"-"` // Raw map
}

// JobYAML represents the YAML configuration for a single job
type JobYAML struct {
	Stage       string            `yaml:"stage"`
	Image       string            `yaml:"image"`
	Script      []string          `yaml:"script"`
	Variables   map[string]string `yaml:"variables"`
	Only        []string          `yaml:"only"`
	Except      []string          `yaml:"except"`
	When        string            `yaml:"when"` // on_success | manual | always | never
	Environment *EnvYAML          `yaml:"environment"`
	Artifacts   *ArtifactsYAML    `yaml:"artifacts"`
	Cache       *CacheYAML        `yaml:"cache"`
	Needs       []string          `yaml:"needs"`
	Release     *ReleaseYAML      `yaml:"release"` // Only effective when triggered by tag: auto-creates Release and uploads assets after job success
}

// EnvYAML represents environment configuration
type EnvYAML struct {
	Name string  `yaml:"name"`
	URL  *string `yaml:"url"`
}

// ArtifactsYAML represents artifacts configuration
type ArtifactsYAML struct {
	Paths    []string `yaml:"paths"`
	ExpireIn string   `yaml:"expire_in"` // e.g. "1 week", "3 days"
}

// CacheYAML represents cache configuration
type CacheYAML struct {
	Paths []string `yaml:"paths"`
	Key   string   `yaml:"key"`
}

// ReleaseYAML represents the release configuration for a job (standard CI style)
//
// Only effective when the pipeline is triggered by a tag push (pipeline.ref = "refs/tags/<tag>").
// After the job succeeds, the executor will:
//  1. Call ReleaseService.CreateReleaseIfNotExists to create a Release (idempotent)
//  2. Iterate Assets globs, match files under workDir, and copy them to the release upload directory
//  3. Call ReleaseService.AttachAssetFromFile to write DB records
//
// Supports variable substitution: $CI_COMMIT_TAG / $TAG -> tag name
//
// Example:
//
//	release_job:
//	  stage: release
//	  script:
//	    - echo "build done"
//	  release:
//	    name: Release $CI_COMMIT_TAG
//	    description: "Auto-generated release"
//	    assets:
//	      - build/*.tar.gz
//	  only:
//	    - tags
type ReleaseYAML struct {
	Name        string   `yaml:"name"`        // Optional, defaults to "Release <tag>"
	Description string   `yaml:"description"` // Optional, release content
	Assets      []string `yaml:"assets"`      // Glob matches files under workDir (relative to workDir)
}

// reservedTopLevelKeys are reserved top-level keys in PipelineYAML; all others are treated as jobs
var reservedTopLevelKeys = map[string]bool{
	"stages":        true,
	"variables":     true,
	"include":       true, // Reserved: future support for include
	"workflow":      true, // Reserved
	"default":       true, // Reserved
	"image":         true, // Reserved: global default image
	"before_script": true,
	"after_script":  true,
}

// ParsePipelineYAML parses .iforge-ci.yml content
func ParsePipelineYAML(content []byte) (*PipelineYAML, error) {
	// First unmarshal into raw map, separating reserved keys from jobs
	var raw map[string]interface{}
	if err := yaml.Unmarshal(content, &raw); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidYAML, err)
	}

	result := &PipelineYAML{
		Raw:  raw,
		Jobs: make(map[string]JobYAML),
	}

	// Parse stages
	if stagesVal, ok := raw["stages"]; ok {
		if stagesArr, ok := stagesVal.([]interface{}); ok {
			for _, s := range stagesArr {
				if str, ok := s.(string); ok {
					result.Stages = append(result.Stages, str)
				}
			}
		}
	}

	// Parse top-level variables
	if varsVal, ok := raw["variables"]; ok {
		if varsMap, ok := varsVal.(map[interface{}]interface{}); ok {
			result.Variables = make(map[string]string)
			for k, v := range varsMap {
				result.Variables[fmt.Sprintf("%v", k)] = fmt.Sprintf("%v", v)
			}
		}
	}

	// Parse all non-reserved keys as jobs
	for k, v := range raw {
		if reservedTopLevelKeys[k] {
			continue
		}
		jobMap, ok := v.(map[interface{}]interface{})
		if !ok {
			continue // Skip non-map entries
		}

		// Serialize map[interface{}]interface{} to YAML, then deserialize to JobYAML
		// This correctly handles nested structures
		jobBytes, err := yaml.Marshal(jobMap)
		if err != nil {
			continue
		}
		var job JobYAML
		if err := yaml.Unmarshal(jobBytes, &job); err != nil {
			return nil, fmt.Errorf("%w: job %q: %v", ErrInvalidYAML, k, err)
		}

		// Default to "test" stage if not specified
		if job.Stage == "" {
			job.Stage = "test"
		}

		// Default when
		if job.When == "" {
			job.When = model.JobWhenOnSuccess
		}

		result.Jobs[k] = job
	}

	// If stages not specified, collect from jobs in order of appearance
	if len(result.Stages) == 0 {
		seen := make(map[string]bool)
		for _, job := range result.Jobs {
			if !seen[job.Stage] {
				result.Stages = append(result.Stages, job.Stage)
				seen[job.Stage] = true
			}
		}
	}

	return result, nil
}

// ShouldRun determines if a job should run on the current ref (based on only/except)
func (j *JobYAML) ShouldRun(ref string) bool {
	// ref is like "refs/heads/main" or "refs/tags/v1.0" or "main"
	// Simplified to branch/tag name matching
	isTag := strings.HasPrefix(ref, "refs/tags/")
	name := normalizeRef(ref)

	// except takes priority
	for _, e := range j.Except {
		if matchRef(name, e, isTag) {
			return false
		}
	}

	// Run by default if no only specified
	if len(j.Only) == 0 {
		return true
	}

	for _, o := range j.Only {
		if matchRef(name, o, isTag) {
			return true
		}
	}
	return false
}

// matchRef simplified glob matching (supports tags/branches keywords + * wildcard)
//
// Parameters:
//   - name: normalized ref name (branch name or tag name)
//   - pattern: only/except config item
//   - isTag: whether current ref is a tag (affects "tags"/"branches" keyword matching)
func matchRef(name, pattern string, isTag bool) bool {
	// Exact match
	if pattern == name {
		return true
	}
	// Keyword match
	if pattern == "tags" {
		return isTag
	}
	if pattern == "branches" {
		return !isTag
	}
	// Simplified: no other glob support, exact match only
	return false
}

// normalizeRef normalizes "refs/heads/main" to "main"
func normalizeRef(ref string) string {
	const headsPrefix = "refs/heads/"
	const tagsPrefix = "refs/tags/"
	if len(ref) > len(headsPrefix) && ref[:len(headsPrefix)] == headsPrefix {
		return ref[len(headsPrefix):]
	}
	if len(ref) > len(tagsPrefix) && ref[:len(tagsPrefix)] == tagsPrefix {
		return ref[len(tagsPrefix):]
	}
	return ref
}

// PipelineService handles Pipeline/Job business logic
type PipelineService struct {
	db       *gorm.DB
	executor *Executor
	// Notification services (optional; channels are skipped if not injected; injected by Container after initialization)
	notificationService *NotificationService
	mailService         *MailService
	webhookService      *WebhookService
}

// NewPipelineService creates a PipelineService
func NewPipelineService(db *gorm.DB) *PipelineService {
	return &PipelineService{
		db:       db,
		executor: NewExecutor(db),
	}
}

// Executor exposes the executor reference for the Container to start workers
func (s *PipelineService) Executor() *Executor {
	return s.executor
}

// SetNotificationService injects the in-app notification service (notifies triggerer when pipeline reaches terminal state)
func (s *PipelineService) SetNotificationService(n *NotificationService) { s.notificationService = n }

// SetMailService injects the mail service (emails triggerer when pipeline reaches terminal state)
func (s *PipelineService) SetMailService(m *MailService) { s.mailService = m }

// SetWebhookService injects the Webhook service (triggers repository webhooks when pipeline reaches terminal state)
func (s *PipelineService) SetWebhookService(w *WebhookService) { s.webhookService = w }

// CreatePipeline creates a Pipeline (called by push/MR/manual triggers)
// The yamlConfig parameter is the raw content of .iforge-ci.yml
func (s *PipelineService) CreatePipeline(
	userName, repoName, triggerEvent, ref, commitSHA, triggeredBy, message, yamlConfig string,
	mrID *int64,
) (*model.Pipeline, error) {
	// Parse YAML
	parsed, err := ParsePipelineYAML([]byte(yamlConfig))
	if err != nil {
		// Create Pipeline even if YAML parsing fails, but set status to failed
		pipeline := &model.Pipeline{
			UserName:       userName,
			RepositoryName: repoName,
			TriggerEvent:   triggerEvent,
			Ref:            ref,
			CommitSHA:      commitSHA,
			Status:         model.PipelineStatusFailed,
			YAMLConfig:     yamlConfig,
			TriggeredBy:    triggeredBy,
			MergeRequestID: mrID,
			Message:        message,
			CreatedAt:      time.Now(),
		}
		if err := s.db.Create(pipeline).Error; err != nil {
			return nil, err
		}
		return pipeline, nil
	}

	// Create Pipeline + all Jobs (in a single transaction)
	var pipeline *model.Pipeline
	err = s.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		pipeline = &model.Pipeline{
			UserName:       userName,
			RepositoryName: repoName,
			TriggerEvent:   triggerEvent,
			Ref:            ref,
			CommitSHA:      commitSHA,
			Status:         model.PipelineStatusPending,
			YAMLConfig:     yamlConfig,
			TriggeredBy:    triggeredBy,
			MergeRequestID: mrID,
			Message:        message,
			CreatedAt:      now,
		}
		if err := tx.Create(pipeline).Error; err != nil {
			return err
		}

		// Serialize global variables
		globalVars := "{}"
		if len(parsed.Variables) > 0 {
			if b, err := json.Marshal(parsed.Variables); err == nil {
				globalVars = string(b)
			}
		}

		// Create records for each job
		for jobName, jobCfg := range parsed.Jobs {
			// Check if job should run based on only/except
			if !jobCfg.ShouldRun(ref) {
				// Create record for skipped job with status skipped
				job := &model.Job{
					PipelineID: pipeline.ID,
					StageName:  jobCfg.Stage,
					JobName:    jobName,
					Status:     model.JobStatusSkipped,
					Image:      jobCfg.Image,
					Script:     joinScript(jobCfg.Script),
					Needs:      toJSON(jobCfg.Needs),
					Vars:       mergeVars(globalVars, jobCfg.Variables),
					OnlyRefs:   toJSON(jobCfg.Only),
					ExceptRefs: toJSON(jobCfg.Except),
					When:       jobCfg.When,
					CreatedAt:  now,
				}
				if jobCfg.Environment != nil {
					env := jobCfg.Environment.Name
					job.Environment = &env
				}
				if err := tx.Create(job).Error; err != nil {
					return err
				}
				continue
			}

			// when=never skip directly
			if jobCfg.When == model.JobWhenNever {
				job := &model.Job{
					PipelineID: pipeline.ID,
					StageName:  jobCfg.Stage,
					JobName:    jobName,
					Status:     model.JobStatusSkipped,
					Image:      jobCfg.Image,
					Script:     joinScript(jobCfg.Script),
					When:       jobCfg.When,
					CreatedAt:  now,
				}
				if err := tx.Create(job).Error; err != nil {
					return err
				}
				continue
			}

			// job to be executed
			job := &model.Job{
				PipelineID: pipeline.ID,
				StageName:  jobCfg.Stage,
				JobName:    jobName,
				Status:     model.JobStatusPending,
				Image:      jobCfg.Image,
				Script:     joinScript(jobCfg.Script),
				Needs:      toJSON(jobCfg.Needs),
				Vars:       mergeVars(globalVars, jobCfg.Variables),
				OnlyRefs:   toJSON(jobCfg.Only),
				ExceptRefs: toJSON(jobCfg.Except),
				When:       jobCfg.When,
				CreatedAt:  now,
			}
			if jobCfg.Environment != nil {
				env := jobCfg.Environment.Name
				job.Environment = &env
			}
			if jobCfg.Artifacts != nil {
				job.Artifacts = toJSON(jobCfg.Artifacts)
			}
			if jobCfg.Cache != nil {
				job.Cache = toJSON(jobCfg.Cache)
			}
			if jobCfg.Release != nil {
				job.Release = toJSON(jobCfg.Release)
			}
			if err := tx.Create(job).Error; err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// async scheduling (non-blocking transaction)
	go s.executor.SchedulePipeline(pipeline.ID)

	return pipeline, nil
}

// GetPipeline retrieves a single Pipeline
func (s *PipelineService) GetPipeline(id int64) (*model.Pipeline, error) {
	var p model.Pipeline
	if err := s.db.First(&p, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPipelineNotFound
		}
		return nil, err
	}
	return &p, nil
}

// ListPipelines lists repository Pipelines (paginated)
func (s *PipelineService) ListPipelines(userName, repoName string, page, pageSize int) ([]model.Pipeline, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	var pipelines []model.Pipeline
	var total int64
	q := s.db.Where("user_name = ? AND repository_name = ?", userName, repoName)
	if err := q.Model(&model.Pipeline{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("created_at DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&pipelines).Error; err != nil {
		return nil, 0, err
	}
	return pipelines, total, nil
}

// ListJobs lists all Jobs of a Pipeline
func (s *PipelineService) ListJobs(pipelineID int64) ([]model.Job, error) {
	var jobs []model.Job
	if err := s.db.Where("pipeline_id = ?", pipelineID).
		Order("stage_name ASC, job_name ASC").
		Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

// ListArtifacts lists all artifacts of jobs in a pipeline
func (s *PipelineService) ListArtifacts(pipelineID int64) ([]model.Artifact, error) {
	// first query all job IDs under the pipeline
	var jobIDs []int64
	if err := s.db.Model(&model.Job{}).Where("pipeline_id = ?", pipelineID).Pluck("id", &jobIDs).Error; err != nil {
		return nil, err
	}
	if len(jobIDs) == 0 {
		return []model.Artifact{}, nil
	}
	var artifacts []model.Artifact
	if err := s.db.Where("job_id IN ?", jobIDs).Order("created_at DESC").Find(&artifacts).Error; err != nil {
		return nil, err
	}
	return artifacts, nil
}

// GetArtifact retrieves a single artifact by ID
func (s *PipelineService) GetArtifact(artifactID int64) (*model.Artifact, error) {
	var artifact model.Artifact
	if err := s.db.First(&artifact, artifactID).Error; err != nil {
		return nil, err
	}
	return &artifact, nil
}

// GetJob retrieves a single Job
func (s *PipelineService) GetJob(id int64) (*model.Job, error) {
	var j model.Job
	if err := s.db.First(&j, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJobNotFound
		}
		return nil, err
	}
	return &j, nil
}

// ListJobLogs lists Job logs (ordered by line number)
func (s *PipelineService) ListJobLogs(jobID int64, fromLine, limit int) ([]model.JobLog, error) {
	q := s.db.Where("job_id = ?", jobID)
	if fromLine > 0 {
		q = q.Where("line_no >= ?", fromLine)
	}
	if limit <= 0 || limit > 5000 {
		limit = 5000
	}
	var logs []model.JobLog
	if err := q.Order("line_no ASC").Limit(limit).Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

// AppendJobLog appends log (called by Executor)
func (s *PipelineService) AppendJobLog(jobID int64, lineNo int, content, stream string) error {
	log := &model.JobLog{
		JobID:     jobID,
		LineNo:    lineNo,
		Content:   content,
		Stream:    stream,
		Timestamp: time.Now(),
	}
	return s.db.Create(log).Error
}

// UpdateJobStatus updates Job status (called by Executor)
func (s *PipelineService) UpdateJobStatus(jobID int64, status string, exitCode *int, failureReason *string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if status == model.JobStatusRunning {
		now := time.Now()
		updates["started_at"] = now
	}
	if status == model.JobStatusSuccess || status == model.JobStatusFailed || status == model.JobStatusCanceled {
		now := time.Now()
		updates["finished_at"] = now
	}
	if exitCode != nil {
		updates["exit_code"] = *exitCode
	}
	if failureReason != nil {
		updates["failure_reason"] = *failureReason
	}
	if err := s.db.Model(&model.Job{}).Where("id = ?", jobID).Updates(updates).Error; err != nil {
		return err
	}

	// async update Pipeline status after Job completion
	if status == model.JobStatusSuccess || status == model.JobStatusFailed || status == model.JobStatusCanceled {
		go s.refreshPipelineStatus(jobID)
	}
	return nil
}

// refreshPipelineStatus recalculates Pipeline status
//
// When pipeline transitions from non-terminal (pending/running) to terminal (success/failed),
// triggers notifyPipelineFinished to send in-app notification + email + Webhook.
func (s *PipelineService) refreshPipelineStatus(jobID int64) {
	var job model.Job
	if err := s.db.First(&job, jobID).Error; err != nil {
		return
	}

	// query pipeline (get old status to check first terminal entry + get StartedAt for duration calculation)
	var pipeline model.Pipeline
	if err := s.db.First(&pipeline, job.PipelineID).Error; err != nil {
		return
	}
	oldStatus := pipeline.Status

	var jobs []model.Job
	if err := s.db.Where("pipeline_id = ?", job.PipelineID).Find(&jobs).Error; err != nil {
		return
	}

	// any job failed -> pipeline failed
	// all jobs success/skipped -> pipeline success
	// otherwise running
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
			dur := now.Sub(*pipeline.StartedAt).Milliseconds()
			updates["duration"] = dur
		}
	}

	s.db.Model(&model.Pipeline{}).Where("id = ?", job.PipelineID).Updates(updates)

	// trigger notification when status first enters terminal state (success/failed)
	// (avoid duplicate triggers: already-terminal pipelines are not notified)
	if newStatus == model.PipelineStatusSuccess || newStatus == model.PipelineStatusFailed {
		if oldStatus != model.PipelineStatusSuccess && oldStatus != model.PipelineStatusFailed {
			// write back new status to pipeline instance for notifyPipelineFinished use
			pipeline.Status = newStatus
			notifyPipelineFinished(s.db, s.notificationService, s.mailService, s.webhookService, pipeline, newStatus)
		}
	}
}

// notifyPipelineFinished triggers in-app notification + email + Webhook when pipeline enters terminal state
//
// notification recipients: pipeline.TriggeredBy (triggerer); system-triggered (e.g. cron) skips user notifications
// notification channels: in-app (NotificationService), email (MailService), Webhook (WebhookService)
// all three services are optional; channels are skipped if not injected.
//
// package-level function: shared by PipelineService.refreshPipelineStatus and Executor.refreshPipelineStatus,
// avoids duplicate notification logic implementation.
func notifyPipelineFinished(
	db *gorm.DB,
	notificationService *NotificationService,
	mailService *MailService,
	webhookService *WebhookService,
	pipeline model.Pipeline,
	finalStatus string,
) {
	branch := normalizeRef(pipeline.Ref)
	commitShort := pipeline.CommitSHA
	if len(commitShort) > 8 {
		commitShort = commitShort[:8]
	}
	statusText := "succeeded"
	if finalStatus == model.PipelineStatusFailed {
		statusText = "failed"
	}
	message := fmt.Sprintf("Pipeline #%d on branch %s commit %s %s",
		pipeline.ID, branch, commitShort, statusText)

	// system-triggered (e.g. cron) does not send user notifications, only triggers webhook
	userTriggered := pipeline.TriggeredBy != "" && pipeline.TriggeredBy != "system"

	// 1. In-app notification
	if notificationService != nil && userTriggered {
		notificationType := "pipeline_success"
		if finalStatus == model.PipelineStatusFailed {
			notificationType = "pipeline_failed"
		}
		if err := notificationService.CreateNotification(
			pipeline.TriggeredBy,
			pipeline.UserName,
			pipeline.RepositoryName,
			notificationType,
			nil, nil, // issueID, commentID not applicable
			pipeline.TriggeredBy,
			message,
		); err != nil {
			fmt.Printf("[warn] CICD: notifyPipelineFinished in-app notification failed: %v\n", err)
		}
	}

	// 2. Email notification
	if mailService != nil && userTriggered {
		var account struct {
			MailAddress string `gorm:"column:mail_address"`
		}
		if err := db.Table("account").Select("mail_address").
			Where("user_name = ?", pipeline.TriggeredBy).First(&account).Error; err == nil && account.MailAddress != "" {
			mailService.SendPipelineNotification(
				[]string{account.MailAddress},
				pipeline.UserName, pipeline.RepositoryName,
				int(pipeline.ID), branch, commitShort, finalStatus, pipeline.Message,
			)
		}
	}

	// 3. Webhook notification
	if webhookService != nil {
		action := "completed"
		payload := map[string]interface{}{
			"action":      action,
			"pipeline":    pipeline,
			"finalStatus": finalStatus,
			"repository": map[string]interface{}{
				"name":      pipeline.RepositoryName,
				"full_name": fmt.Sprintf("%s/%s", pipeline.UserName, pipeline.RepositoryName),
			},
			"sender": map[string]interface{}{
				"login": pipeline.TriggeredBy,
			},
		}
		webhookService.TriggerRepositoryWebhooks(
			pipeline.UserName, pipeline.RepositoryName,
			"pipeline", action, payload,
		)
	}
}

// RefreshPipelineStatusByJob public interface for external runner handler to call,
// triggers pipeline status recalculation after job status change
func (s *PipelineService) RefreshPipelineStatusByJob(jobID int64) {
	s.refreshPipelineStatus(jobID)
}

// ScheduleNextJobsByJobID triggers next DAG scheduling round after external runner completes job
// queries job pipelineID, then schedules pending jobs with satisfied dependencies
func (s *PipelineService) ScheduleNextJobsByJobID(jobID int64) {
	if s.executor == nil {
		return
	}
	var job model.Job
	if err := s.db.Select("pipeline_id").First(&job, jobID).Error; err != nil {
		return
	}
	s.executor.ScheduleReadyJobs(job.PipelineID)
}

// CancelPipeline cancels entire Pipeline (all running/pending jobs marked as canceled)
func (s *PipelineService) CancelPipeline(pipelineID int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// cancel all unfinished jobs
		if err := tx.Model(&model.Job{}).
			Where("pipeline_id = ? AND status IN ?", pipelineID, []string{model.JobStatusPending, model.JobStatusRunning}).
			Updates(map[string]interface{}{
				"status":      model.JobStatusCanceled,
				"finished_at": time.Now(),
			}).Error; err != nil {
			return err
		}
		// mark pipeline as canceled
		return tx.Model(&model.Pipeline{}).Where("id = ?", pipelineID).
			Updates(map[string]interface{}{
				"status":      model.PipelineStatusCanceled,
				"finished_at": time.Now(),
			}).Error
	})
}

// RetryPipeline retries failed jobs (creates new job records)
func (s *PipelineService) RetryPipeline(pipelineID int64) error {
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. check pipeline current status, only allow retry for terminal state (failed/canceled/success)
		//    running/pending pipelines cannot retry (in progress)
		var pipeline model.Pipeline
		if err := tx.First(&pipeline, pipelineID).Error; err != nil {
			return err
		}
		if pipeline.Status != model.PipelineStatusFailed &&
			pipeline.Status != model.PipelineStatusCanceled &&
			pipeline.Status != model.PipelineStatusSuccess {
			return fmt.Errorf("pipeline is not retryable (current status: %s, must be failed, canceled or success)", pipeline.Status)
		}

		// 2. reset failed/canceled jobs to pending
		result := tx.Model(&model.Job{}).
			Where("pipeline_id = ? AND status IN ?", pipelineID, []string{model.JobStatusFailed, model.JobStatusCanceled}).
			Updates(map[string]interface{}{
				"status":         model.JobStatusPending,
				"started_at":     nil,
				"finished_at":    nil,
				"exit_code":      nil,
				"failure_reason": nil,
			})
		if result.Error != nil {
			return result.Error
		}

		// 3. if no failed/canceled jobs (e.g. success state pipeline or canceled pipeline),
		//    reset all jobs to pending (full rerun), avoid pipeline stuck in running
		if result.RowsAffected == 0 {
			if err := tx.Model(&model.Job{}).
				Where("pipeline_id = ?", pipelineID).
				Updates(map[string]interface{}{
					"status":         model.JobStatusPending,
					"started_at":     nil,
					"finished_at":    nil,
					"exit_code":      nil,
					"failure_reason": nil,
				}).Error; err != nil {
				return err
			}
		}

		// 4. delete old logs for all jobs (old logs should not remain during retry, otherwise SSE returns them all at once)
		//    new log lineNo restarts counting from 0
		if err := tx.Where("job_id IN (?)",
			tx.Model(&model.Job{}).Select("id").Where("pipeline_id = ?", pipelineID),
		).Delete(&model.JobLog{}).Error; err != nil {
			return err
		}

		// 5. Pipeline returns to running
		return tx.Model(&model.Pipeline{}).Where("id = ?", pipelineID).
			Updates(map[string]interface{}{
				"status":      model.PipelineStatusRunning,
				"finished_at": nil,
				"duration":    nil,
			}).Error
	}); err != nil {
		return err
	}

	// 5. trigger scheduling after transaction commit (async, avoid blocking API response)
	//    without this step, jobs would stuck in pending after retry, pipeline would run forever when external runner is offline
	if s.executor != nil {
		go s.executor.SchedulePipeline(pipelineID)
	}
	return nil
}

// ============================================================================
// Runner Service
// ============================================================================

// RunnerService handles Runner registration, heartbeat, allocation
type RunnerService struct {
	db *gorm.DB
}

// NewRunnerService creates RunnerService
func NewRunnerService(db *gorm.DB) *RunnerService {
	return &RunnerService{db: db}
}

// EnsureBuiltinRunner ensures builtin runner exists (called at system startup)
func (s *RunnerService) EnsureBuiltinRunner(name, runnerType string) (*model.Runner, error) {
	var runner model.Runner
	err := s.db.Where("is_builtin = ? AND name = ?", true, name).First(&runner).Error
	if err == nil {
		return &runner, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// generate token
	token, hash, err := GenerateRunnerToken()
	if err != nil {
		return nil, err
	}

	runner = model.Runner{
		Name:        name,
		TokenHash:   hash,
		Status:      model.RunnerStatusOnline,
		Type:        runnerType,
		Description: "Built-in " + runnerType + " runner",
		IsBuiltin:   true,
		MaxJobs:     2, // default 2 concurrent jobs
		CreatedAt:   time.Now(),
		Version:     "0.1.0",
	}
	if err := s.db.Create(&runner).Error; err != nil {
		return nil, err
	}

	// Note: token is only returned once at creation, caller should log it
	_ = token
	return &runner, nil
}

// ListRunners lists all runners
func (s *RunnerService) ListRunners() ([]model.Runner, error) {
	var runners []model.Runner
	if err := s.db.Order("is_builtin DESC, created_at DESC").Find(&runners).Error; err != nil {
		return nil, err
	}
	return runners, nil
}

// GetRunner retrieves runner (with token verification)
func (s *RunnerService) GetRunnerByToken(token string) (*model.Runner, error) {
	hash := hashToken(token)
	var runner model.Runner
	if err := s.db.Where("token_hash = ?", hash).First(&runner).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRunnerNotFound
		}
		return nil, err
	}
	return &runner, nil
}

// UpdateHeartbeat updates runner heartbeat
func (s *RunnerService) UpdateHeartbeat(runnerID int64) error {
	now := time.Now()
	return s.db.Model(&model.Runner{}).Where("id = ?", runnerID).
		Updates(map[string]interface{}{
			"last_heartbeat": now,
			"status":         model.RunnerStatusOnline,
		}).Error
}

// AcquireAvailableRunner finds an available runner (called when scheduling pending jobs)
func (s *RunnerService) AcquireAvailableRunner(jobType string) (*model.Runner, error) {
	// simplified: find online and non-busy builtin runner
	var runner model.Runner
	err := s.db.Where("status = ? AND is_builtin = ?", model.RunnerStatusOnline, true).
		First(&runner).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNoRunnerAvailable
		}
		return nil, err
	}
	return &runner, nil
}

// GenerateRunnerToken generates runner token (plaintext + hash)
func GenerateRunnerToken() (string, string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}
	token := "grt_" + hex.EncodeToString(bytes) // iforge runner token
	return token, hashToken(token), nil
}

// hashToken calculates token hash (simplified: hex-encoded SHA256)
func hashToken(token string) string {
	// use SHA256
	return sha256Hex(token)
}

// sha256Hex calculates SHA256 hex (avoids new dependencies, uses standard library)
func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// ============================================================================
// Secret Service
// ============================================================================

// SecretService handles encrypted variables
type SecretService struct {
	db *gorm.DB
	// encryptionKey 32 bytes, loaded from system_setting (randomly generated on first use)
}

// NewSecretService creates SecretService
func NewSecretService(db *gorm.DB) *SecretService {
	return &SecretService{db: db}
}

// ListSecrets lists repository secrets (does not return encrypted values)
func (s *SecretService) ListSecrets(userName, repoName string) ([]model.Secret, error) {
	var secrets []model.Secret
	if err := s.db.Where("user_name = ? AND repository_name = ?", userName, repoName).
		Order("key ASC").Find(&secrets).Error; err != nil {
		return nil, err
	}
	return secrets, nil
}

// SetSecret creates or updates secret
func (s *SecretService) SetSecret(userName, repoName, key, plaintext, creator string) error {
	// encrypt
	encKey, err := s.getEncryptionKey()
	if err != nil {
		return err
	}
	encrypted, err := encryptAESGCM(encKey, plaintext)
	if err != nil {
		return err
	}

	// query if already exists
	var existing model.Secret
	err = s.db.Where("user_name = ? AND repository_name = ? AND `key` = ?", userName, repoName, key).
		First(&existing).Error

	if err == nil {
		// update
		return s.db.Model(&existing).Updates(map[string]interface{}{
			"encrypted_value": encrypted,
			"updated_at":      time.Now(),
		}).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// create
	secret := &model.Secret{
		UserName:       userName,
		RepositoryName: repoName,
		Key:            key,
		EncryptedValue: encrypted,
		CreatedBy:      creator,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	return s.db.Create(secret).Error
}

// DeleteSecret deletes secret
func (s *SecretService) DeleteSecret(userName, repoName, key string) error {
	result := s.db.Where("user_name = ? AND repository_name = ? AND `key` = ?", userName, repoName, key).
		Delete(&model.Secret{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrSecretNotFound
	}
	return nil
}

// DecryptSecrets decrypts all repository secrets (called by Executor when injecting environment variables)
func (s *SecretService) DecryptSecrets(userName, repoName string) (map[string]string, error) {
	var secrets []model.Secret
	if err := s.db.Where("user_name = ? AND repository_name = ?", userName, repoName).
		Find(&secrets).Error; err != nil {
		return nil, err
	}
	encKey, err := s.getEncryptionKey()
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(secrets))
	for _, sec := range secrets {
		plain, err := decryptAESGCM(encKey, sec.EncryptedValue)
		if err != nil {
			return nil, fmt.Errorf("decrypt secret %q: %w", sec.Key, err)
		}
		result[sec.Key] = plain
	}
	return result, nil
}

// getEncryptionKey loads secret_key from system_setting, generates if not found
func (s *SecretService) getEncryptionKey() ([]byte, error) {
	var setting model.SystemSetting
	err := s.db.Where("`key` = ?", "cicd_secret_key").First(&setting).Error
	if err == nil && setting.Value != "" {
		key, err := hex.DecodeString(setting.Value)
		if err == nil && len(key) == 32 {
			return key, nil
		}
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// use transaction + row-level lock to prevent race conditions
	var key []byte
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// check again (may have been created by another transaction while waiting for lock)
		var existingSetting model.SystemSetting
		err := tx.Set("gorm:query_option", "FOR UPDATE").
			Where("`key` = ?", "cicd_secret_key").First(&existingSetting).Error

		if err == nil && existingSetting.Value != "" {
			// already exists, use existing
			decodedKey, err := hex.DecodeString(existingSetting.Value)
			if err == nil && len(decodedKey) == 32 {
				key = decodedKey
				return nil
			}
		}

		// does not exist or invalid, generate new
		newKey := make([]byte, 32)
		if _, err := rand.Read(newKey); err != nil {
			return err
		}

		// if exists but invalid, update; otherwise create
		if err == nil {
			existingSetting.Value = hex.EncodeToString(newKey)
			if err := tx.Save(&existingSetting).Error; err != nil {
				return err
			}
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			newSetting := model.SystemSetting{
				Key:   "cicd_secret_key",
				Value: hex.EncodeToString(newKey),
			}
			if err := tx.Create(&newSetting).Error; err != nil {
				return err
			}
		} else {
			return err
		}

		key = newKey
		return nil
	})

	if err != nil {
		return nil, err
	}
	return key, nil
}

// encryptAESGCM encrypts with AES-256-GCM, returns hex-encoded string
func encryptAESGCM(key []byte, plaintext string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	encrypted := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(encrypted), nil
}

// decryptAESGCM decrypts with AES-256-GCM
func decryptAESGCM(key []byte, ciphertextHex string) (string, error) {
	ciphertext, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// ============================================================================
// Environment & Deployment Service
// ============================================================================

// DeploymentService handles deployment environments and deployment records
type DeploymentService struct {
	db *gorm.DB
}

// NewDeploymentService creates DeploymentService
func NewDeploymentService(db *gorm.DB) *DeploymentService {
	return &DeploymentService{db: db}
}

// ListEnvironments lists repository environments
func (s *DeploymentService) ListEnvironments(userName, repoName string) ([]model.Environment, error) {
	var envs []model.Environment
	if err := s.db.Where("user_name = ? AND repository_name = ?", userName, repoName).
		Order("name ASC").Find(&envs).Error; err != nil {
		return nil, err
	}
	return envs, nil
}

// GetOrCreateEnvironment queries or creates environment (auto-created when job configures environment)
func (s *DeploymentService) GetOrCreateEnvironment(userName, repoName, name string, url *string) (*model.Environment, error) {
	var env model.Environment
	err := s.db.Where("user_name = ? AND repository_name = ? AND name = ?", userName, repoName, name).
		First(&env).Error
	if err == nil {
		if url != nil && env.URL == nil {
			s.db.Model(&env).Update("url", url)
		}
		return &env, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	env = model.Environment{
		UserName:         userName,
		RepositoryName:   repoName,
		Name:             name,
		URL:              url,
		DeploymentPolicy: "manual",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	if err := s.db.Create(&env).Error; err != nil {
		return nil, err
	}
	return &env, nil
}

// CreateDeployment creates deployment record (called by executor when job completes)
func (s *DeploymentService) CreateDeployment(
	envID int64, userName, repoName, commitSHA, deployedBy string,
	pipelineID, jobID *int64,
) (*model.Deployment, error) {
	d := &model.Deployment{
		EnvironmentID:  envID,
		UserName:       userName,
		RepositoryName: repoName,
		PipelineID:     pipelineID,
		JobID:          jobID,
		CommitSHA:      commitSHA,
		Status:         model.DeploymentStatusInProgress,
		DeployedBy:     deployedBy,
		DeployedAt:     time.Now(),
	}
	if err := s.db.Create(d).Error; err != nil {
		return nil, err
	}
	return d, nil
}

// UpdateDeploymentStatus updates deployment status
func (s *DeploymentService) UpdateDeploymentStatus(id int64, status string, note *string) error {
	updates := map[string]interface{}{"status": status}
	if status == model.DeploymentStatusSuccess || status == model.DeploymentStatusFailed || status == model.DeploymentStatusCanceled {
		now := time.Now()
		updates["finished_at"] = now
	}
	if note != nil {
		updates["note"] = *note
	}
	return s.db.Model(&model.Deployment{}).Where("id = ?", id).Updates(updates).Error
}

// ListDeployments lists repository deployment records
func (s *DeploymentService) ListDeployments(userName, repoName string, envName string, page, pageSize int) ([]model.Deployment, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	q := s.db.Where("user_name = ? AND repository_name = ?", userName, repoName)
	if envName != "" {
		q = q.Joins("JOIN cicd_environment ON cicd_environment.id = cicd_deployment.environment_id").
			Where("cicd_environment.name = ?", envName)
	}
	var total int64
	if err := q.Model(&model.Deployment{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var deployments []model.Deployment
	if err := q.Order("deployed_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&deployments).Error; err != nil {
		return nil, 0, err
	}
	return deployments, total, nil
}

// ListDeploymentsByCommit lists all deployments for a commit (full traceability)
func (s *DeploymentService) ListDeploymentsByCommit(userName, repoName, commitSHA string) ([]model.Deployment, error) {
	var deployments []model.Deployment
	if err := s.db.Where("user_name = ? AND repository_name = ? AND commit_sha = ?", userName, repoName, commitSHA).
		Order("deployed_at DESC").Find(&deployments).Error; err != nil {
		return nil, err
	}
	return deployments, nil
}

// ============================================================================
// helper functions
// ============================================================================

// joinScript merges script array into single string (one command per line)
func joinScript(lines []string) string {
	result := ""
	for i, line := range lines {
		if i > 0 {
			result += "\n"
		}
		result += line
	}
	return result
}

// toJSON serializes to JSON string (returns "[]" or "{}" on failure)
func toJSON(v interface{}) string {
	if v == nil {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

// mergeVars merges global vars and job-level vars (job-level overrides global)
func mergeVars(globalVarsJSON string, jobVars map[string]string) string {
	global := map[string]string{}
	if globalVarsJSON != "" && globalVarsJSON != "{}" {
		_ = json.Unmarshal([]byte(globalVarsJSON), &global)
	}
	for k, v := range jobVars {
		global[k] = v
	}
	return toJSON(global)
}
