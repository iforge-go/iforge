package model

import "time"

// CI/CD data model.
//
// Design notes:
//   - 8 tables: Pipeline / Job / JobLog / Artifact / Runner / Secret / Environment / Deployment
//   - Repository-level association uses (user_name, repository_name), following iforge convention
//   - Pipeline/Job use auto-increment IDs (high volume, FK relations, no slug encoding needed)
//   - On Job completion a CommitStatus row is written (reusing commit_status table) so MR Checks tab becomes visible
//   - Deployment completes the full traceability chain: Story -> Task -> Branch -> Commit -> MR -> Pipeline -> Deployment

// Pipeline represents a single CI/CD run (one push/MR/manual trigger corresponds to one Pipeline)
type Pipeline struct {
	ID             int64  `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	UserName       string `gorm:"column:user_name;index:idx_pipeline_repo,priority:1;not null;size:100" json:"userName"`
	RepositoryName string `gorm:"column:repository_name;index:idx_pipeline_repo,priority:2;not null;size:100" json:"repositoryName"`
	TriggerEvent   string `gorm:"column:trigger_event;not null;size:50" json:"triggerEvent"`
	Ref            string `gorm:"column:ref;not null;size:255" json:"ref"`
	CommitSHA      string `gorm:"column:commit_sha;index:idx_pipeline_commit;not null;size:40" json:"commitSha"`
	Status         string `gorm:"column:status;index:idx_pipeline_status;not null;size:20" json:"status"`
	YAMLConfig     string `gorm:"column:yaml_config;type:text" json:"yamlConfig"`
	TriggeredBy    string `gorm:"column:triggered_by;not null;size:100" json:"triggeredBy"`
	MergeRequestID *int64 `gorm:"column:merge_request_id;index:idx_pipeline_mr" json:"mergeRequestId,omitempty"`
	CreatedAt      time.Time  `gorm:"column:created_at;index:idx_pipeline_created;not null" json:"createdAt"`
	StartedAt      *time.Time `gorm:"column:started_at" json:"startedAt,omitempty"`
	FinishedAt     *time.Time `gorm:"column:finished_at" json:"finishedAt,omitempty"`
	Duration       *int64     `gorm:"column:duration" json:"duration,omitempty"`
	Message        string     `gorm:"column:message;type:text" json:"message"`
}

func (Pipeline) TableName() string { return "cicd_pipeline" }

// Job represents a concrete task within a Pipeline (a stage may contain multiple parallel jobs)
type Job struct {
	ID         int64  `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	PipelineID int64  `gorm:"column:pipeline_id;index:idx_job_pipeline,priority:1;not null;foreignKey;constraint:OnDelete:CASCADE" json:"pipelineId"`
	StageName  string `gorm:"column:stage_name;index:idx_job_pipeline,priority:2;not null;size:100" json:"stageName"`
	JobName    string `gorm:"column:job_name;index:idx_job_pipeline,priority:3;not null;size:100" json:"jobName"`
	Status     string `gorm:"column:status;index:idx_job_status;not null;size:20" json:"status"`
	RunnerID   *int64 `gorm:"column:runner_id;index:idx_job_runner;foreignKey;constraint:OnDelete:SET NULL" json:"runnerId,omitempty"`
	Image      string `gorm:"column:image;size:255" json:"image"`
	Script     string `gorm:"column:script;type:text;not null" json:"script"`
	Needs      string `gorm:"column:needs;type:text" json:"needs"`
	Vars       string `gorm:"column:variables;type:text" json:"vars"`
	OnlyRefs   string `gorm:"column:only_refs;type:text" json:"onlyRefs"`
	ExceptRefs string `gorm:"column:except_refs;type:text" json:"exceptRefs"`
	When       string  `gorm:"column:when;not null;size:20" json:"when"`
	Environment *string `gorm:"column:environment;size:100" json:"environment,omitempty"`
	Artifacts   string  `gorm:"column:artifacts;type:text" json:"artifacts"`
	Cache       string  `gorm:"column:cache;type:text" json:"cache"`
	Release     string  `gorm:"column:release;type:text" json:"release"`
	StartedAt   *time.Time `gorm:"column:started_at" json:"startedAt,omitempty"`
	FinishedAt  *time.Time `gorm:"column:finished_at" json:"finishedAt,omitempty"`
	ExitCode    *int       `gorm:"column:exit_code" json:"exitCode,omitempty"`
	FailureReason *string `gorm:"column:failure_reason;type:text" json:"failureReason,omitempty"`
	// Timeout in seconds; 0 means no limit; default 3600 (1 hour)
	Timeout   int       `gorm:"column:timeout;not null;default:3600" json:"timeout"`
	CreatedAt time.Time `gorm:"column:created_at;not null" json:"createdAt"`
}

func (Job) TableName() string { return "cicd_job" }

// JobLog represents the log of a Job (stored per-line for streaming queries and pagination)
type JobLog struct {
	ID        int64     `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	JobID     int64     `gorm:"column:job_id;index:idx_joblog_job,priority:1" json:"jobId"`
	LineNo    int       `gorm:"column:line_no;index:idx_joblog_job,priority:2" json:"lineNo"`
	Content   string    `gorm:"column:content;type:text" json:"content"`
	Timestamp time.Time `gorm:"column:timestamp;index:idx_joblog_ts" json:"timestamp"`
	Stream    string    `gorm:"column:stream" json:"stream"`
}

func (JobLog) TableName() string { return "cicd_job_log" }

// Artifact represents a build artifact produced by a Job
type Artifact struct {
	ID          int64      `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	JobID       int64      `gorm:"column:job_id;index:idx_artifact_job" json:"jobId"`
	Name        string     `gorm:"column:name" json:"name"`
	Path        string     `gorm:"column:path" json:"path"`
	Size        int64      `gorm:"column:size" json:"size"`
	StoragePath string     `gorm:"column:storage_path" json:"storagePath"`
	ExpiresAt   *time.Time `gorm:"column:expires_at;index:idx_artifact_expires" json:"expiresAt,omitempty"`
	CreatedAt   time.Time  `gorm:"column:created_at" json:"createdAt"`
}

func (Artifact) TableName() string { return "cicd_artifact" }

// Runner represents a registered executor (built-in or external)
type Runner struct {
	ID        int64  `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Name      string `gorm:"column:name;not null;size:100" json:"name"`
	TokenHash string `gorm:"column:token_hash;uniqueIndex:idx_runner_token;not null;size:255" json:"-"`
	Status    string `gorm:"column:status;index:idx_runner_status;not null;size:20" json:"status"`
	Type      string `gorm:"column:type;not null;size:20" json:"type"`
	Tags      string `gorm:"column:tags;type:text" json:"tags"`
	LastHeartbeat *time.Time `gorm:"column:last_heartbeat" json:"lastHeartbeat,omitempty"`
	Version       string     `gorm:"column:version;size:50" json:"version"`
	IPAddress     string     `gorm:"column:ip_address;size:45" json:"ipAddress"`
	Description   string     `gorm:"column:description;size:500" json:"description"`
	// IsBuiltin marks a built-in runner (cannot be deleted)
	IsBuiltin bool      `gorm:"column:is_builtin;not null;default:false" json:"isBuiltin"`
	MaxJobs   int       `gorm:"column:max_jobs;not null;default:0" json:"maxJobs"`
	CreatedAt time.Time `gorm:"column:created_at;not null" json:"createdAt"`
	// Repo scope (user_name/repo_name); empty means globally available
	UserName       *string `gorm:"column:user_name;index:idx_runner_repo,priority:1;size:100" json:"userName,omitempty"`
	RepositoryName *string `gorm:"column:repository_name;index:idx_runner_repo,priority:2;size:100" json:"repositoryName,omitempty"`
}

func (Runner) TableName() string { return "cicd_runner" }

// Secret represents a repository-level encrypted variable (injected into the Job container as env vars)
type Secret struct {
	ID             int64     `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	UserName       string    `gorm:"column:user_name;index:idx_secret_repo,priority:1;uniqueIndex:idx_secret_repo_key,priority:1;not null;size:100" json:"userName"`
	RepositoryName string    `gorm:"column:repository_name;index:idx_secret_repo,priority:2;uniqueIndex:idx_secret_repo_key,priority:2;not null;size:100" json:"repositoryName"`
	Key            string    `gorm:"column:key;index:idx_secret_repo,priority:3;uniqueIndex:idx_secret_repo_key,priority:3;not null;size:100" json:"key"`
	EncryptedValue string    `gorm:"column:encrypted_value;type:text;not null" json:"-"`
	CreatedBy      string    `gorm:"column:created_by;not null;size:100" json:"createdBy"`
	CreatedAt      time.Time `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt      time.Time `gorm:"column:updated_at;not null" json:"updatedAt"`
}

func (Secret) TableName() string { return "cicd_secret" }

// Environment represents a deployment environment (dev/staging/prod, etc.)
type Environment struct {
	ID             int64   `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	UserName       string  `gorm:"column:user_name;index:idx_env_repo,priority:1;uniqueIndex:idx_env_repo_name,priority:1;not null;size:100" json:"userName"`
	RepositoryName string  `gorm:"column:repository_name;index:idx_env_repo,priority:2;uniqueIndex:idx_env_repo_name,priority:2;not null;size:100" json:"repositoryName"`
	Name           string  `gorm:"column:name;index:idx_env_repo,priority:3;uniqueIndex:idx_env_repo_name,priority:3;not null;size:100" json:"name"`
	URL            *string `gorm:"column:url;size:500" json:"url,omitempty"`
	DeploymentPolicy string `gorm:"column:deployment_policy;not null;size:20" json:"deploymentPolicy"`
	ProtectionRules  string `gorm:"column:protection_rules;type:text" json:"protectionRules"`
	CreatedAt        time.Time `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt        time.Time `gorm:"column:updated_at;not null" json:"updatedAt"`
}

func (Environment) TableName() string { return "cicd_environment" }

// Deployment represents a deployment record (the final piece of the full traceability chain)
type Deployment struct {
	ID             int64  `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	EnvironmentID  int64  `gorm:"column:environment_id;index:idx_deploy_env" json:"environmentId"`
	UserName       string `gorm:"column:user_name;index:idx_deploy_repo,priority:1" json:"userName"`
	RepositoryName string `gorm:"column:repository_name;index:idx_deploy_repo,priority:2" json:"repositoryName"`
	PipelineID     *int64 `gorm:"column:pipeline_id;index:idx_deploy_pipeline" json:"pipelineId,omitempty"`
	JobID          *int64 `gorm:"column:job_id" json:"jobId,omitempty"`
	CommitSHA      string `gorm:"column:commit_sha;index:idx_deploy_commit" json:"commitSha"`
	Status         string `gorm:"column:status;index:idx_deploy_status" json:"status"`
	DeployedBy     string `gorm:"column:deployed_by" json:"deployedBy"`
	DeployedAt     time.Time  `gorm:"column:deployed_at;index:idx_deploy_time" json:"deployedAt"`
	FinishedAt     *time.Time `gorm:"column:finished_at" json:"finishedAt,omitempty"`
	RollbackOfID   *int64  `gorm:"column:rollback_of_id" json:"rollbackOfId,omitempty"`
	Note           *string `gorm:"column:note;type:text" json:"note,omitempty"`
}

func (Deployment) TableName() string { return "cicd_deployment" }

// CronSchedule represents a scheduled CI/CD trigger configuration for a repository.
//
// Design notes:
//   - name is unique within a repo (easy to identify in the UI)
//   - schedule is a standard 5-field cron expression (minute hour day month weekday)
//   - branch specifies the target branch; the pipeline uses the latest commit on that branch
//   - yaml_config is optional; when empty, the repository's .iforge-ci.yml is used
//   - next_run_at is updated by the scheduler after each trigger; on startup it also serves as
//     a fallback query for next_run_at <= now
//   - triggered_by="system"; notification integration skips user notifications for this value
//     (to avoid sending notifications to a non-existent "system" user)
type CronSchedule struct {
	ID             int64  `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	UserName       string `gorm:"column:user_name;index:idx_cron_repo,priority:1;uniqueIndex:idx_cron_repo_name,priority:1" json:"userName"`
	RepositoryName string `gorm:"column:repository_name;index:idx_cron_repo,priority:2;uniqueIndex:idx_cron_repo_name,priority:2" json:"repositoryName"`
	Name           string `gorm:"column:name;type:varchar(255);uniqueIndex:idx_cron_repo_name,priority:3" json:"name"`
	Schedule       string `gorm:"column:schedule" json:"schedule"`
	Branch         string `gorm:"column:branch" json:"branch"`
	YAMLConfig     string `gorm:"column:yaml_config;type:text" json:"yamlConfig,omitempty"`
	Enabled        bool   `gorm:"column:enabled" json:"enabled"`
	Creator        string `gorm:"column:creator" json:"creator"`
	LastRunAt      *time.Time `gorm:"column:last_run_at" json:"lastRunAt,omitempty"`
	NextRunAt      *time.Time `gorm:"column:next_run_at;index:idx_cron_next" json:"nextRunAt,omitempty"`
	CreatedAt      time.Time  `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt      time.Time  `gorm:"column:updated_at" json:"updatedAt"`
}

func (CronSchedule) TableName() string { return "cicd_cron_schedule" }

// Pipeline status enum constants
const (
	PipelineStatusPending  = "pending"
	PipelineStatusRunning  = "running"
	PipelineStatusSuccess  = "success"
	PipelineStatusFailed   = "failed"
	PipelineStatusCanceled = "canceled"
)

// Job status enum constants
const (
	JobStatusPending  = "pending"
	JobStatusRunning  = "running"
	JobStatusSuccess  = "success"
	JobStatusFailed   = "failed"
	JobStatusCanceled = "canceled"
	JobStatusSkipped  = "skipped"
)

// Runner status enum constants
const (
	RunnerStatusOnline  = "online"
	RunnerStatusOffline = "offline"
	RunnerStatusBusy    = "busy"
)

// Runner type enum constants
const (
	RunnerTypeShell      = "shell"
	RunnerTypeDocker     = "docker"
	RunnerTypeKubernetes = "kubernetes"
)

// Trigger event enum constants
const (
	TriggerEventPush         = "push"
	TriggerEventMergeRequest = "merge_request"
	TriggerEventManual       = "manual"
	TriggerEventCron         = "cron"
	TriggerEventExternal     = "external"
)

// Deployment status enum constants
const (
	DeploymentStatusInProgress = "in_progress"
	DeploymentStatusSuccess    = "success"
	DeploymentStatusFailed     = "failed"
	DeploymentStatusCanceled   = "canceled"
)

// Job when enum constants
const (
	JobWhenOnSuccess = "on_success"
	JobWhenManual    = "manual"
	JobWhenAlways    = "always"
	JobWhenNever     = "never"
)
