package service

import (
	"encoding/json"
	"iforge/iforge/internal/model"
	"testing"
	"time"
)

// TestJobYAML_ShouldRun tests if job should run on specified ref
func TestJobYAML_ShouldRun(t *testing.T) {
	tests := []struct {
		name     string
		job      JobYAML
		ref      string
		expected bool
	}{
		{
			name: "no only/except - should run on any ref",
			job: JobYAML{
				Stage: "build",
			},
			ref:      "refs/heads/main",
			expected: true,
		},
		{
			name: "only main - should run on main",
			job: JobYAML{
				Stage: "build",
				Only:  []string{"main"},
			},
			ref:      "refs/heads/main",
			expected: true,
		},
		{
			name: "only main - should not run on develop",
			job: JobYAML{
				Stage: "build",
				Only:  []string{"main"},
			},
			ref:      "refs/heads/develop",
			expected: false,
		},
		{
			name: "except main - should not run on main",
			job: JobYAML{
				Stage:  "build",
				Except: []string{"main"},
			},
			ref:      "refs/heads/main",
			expected: false,
		},
		{
			name: "except main - should run on develop",
			job: JobYAML{
				Stage:  "build",
				Except: []string{"main"},
			},
			ref:      "refs/heads/develop",
			expected: true,
		},
		{
			name: "only tags - should run on tag",
			job: JobYAML{
				Stage: "deploy",
				Only:  []string{"tags"},
			},
			ref:      "refs/tags/v1.0.0",
			expected: true,
		},
		{
			name: "only tags - should not run on branch",
			job: JobYAML{
				Stage: "deploy",
				Only:  []string{"tags"},
			},
			ref:      "refs/heads/main",
			expected: false,
		},
		{
			name: "only branches - should run on branch",
			job: JobYAML{
				Stage: "test",
				Only:  []string{"branches"},
			},
			ref:      "refs/heads/main",
			expected: true,
		},
		{
			name: "only branches - should not run on tag",
			job: JobYAML{
				Stage: "test",
				Only:  []string{"branches"},
			},
			ref:      "refs/tags/v1.0.0",
			expected: false,
		},
		{
			name: "except overrides only",
			job: JobYAML{
				Stage:  "build",
				Only:   []string{"main", "develop"},
				Except: []string{"develop"},
			},
			ref:      "refs/heads/develop",
			expected: false,
		},
		{
			name: "multiple only patterns - match first",
			job: JobYAML{
				Stage: "build",
				Only:  []string{"main", "develop", "feature/*"},
			},
			ref:      "refs/heads/develop",
			expected: true,
		},
		{
			name: "multiple only patterns - match second",
			job: JobYAML{
				Stage: "build",
				Only:  []string{"main", "develop"},
			},
			ref:      "refs/heads/develop",
			expected: true,
		},
		{
			name: "multiple only patterns - no match",
			job: JobYAML{
				Stage: "build",
				Only:  []string{"main", "develop"},
			},
			ref:      "refs/heads/feature",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.job.ShouldRun(tt.ref)
			if result != tt.expected {
				t.Errorf("ShouldRun(%s) = %v, want %v", tt.ref, result, tt.expected)
			}
		})
	}
}

// TestNormalizeRef tests ref normalization
func TestNormalizeRef(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		expected string
	}{
		{
			name:     "refs/heads/main",
			ref:      "refs/heads/main",
			expected: "main",
		},
		{
			name:     "refs/heads/feature/branch",
			ref:      "refs/heads/feature/branch",
			expected: "feature/branch",
		},
		{
			name:     "refs/tags/v1.0.0",
			ref:      "refs/tags/v1.0.0",
			expected: "v1.0.0",
		},
		{
			name:     "already normalized - main",
			ref:      "main",
			expected: "main",
		},
		{
			name:     "already normalized - develop",
			ref:      "develop",
			expected: "develop",
		},
		{
			name:     "refs/heads/ with special chars",
			ref:      "refs/heads/feature-123",
			expected: "feature-123",
		},
		{
			name:     "refs/tags/ with special chars",
			ref:      "refs/tags/v1.0.0-rc1",
			expected: "v1.0.0-rc1",
		},
		{
			name:     "empty string",
			ref:      "",
			expected: "",
		},
		{
			name:     "partial refs/heads",
			ref:      "refs/heads",
			expected: "refs/heads",
		},
		{
			name:     "partial refs/tags",
			ref:      "refs/tags",
			expected: "refs/tags",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeRef(tt.ref)
			if result != tt.expected {
				t.Errorf("normalizeRef(%s) = %s, want %s", tt.ref, result, tt.expected)
			}
		})
	}
}

// TestMatchRef tests ref matching
func TestMatchRef(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		pattern  string
		isTag    bool
		expected bool
	}{
		{
			name:     "exact match",
			ref:      "main",
			pattern:  "main",
			isTag:    false,
			expected: true,
		},
		{
			name:     "exact match - develop",
			ref:      "develop",
			pattern:  "develop",
			isTag:    false,
			expected: true,
		},
		{
			name:     "no match",
			ref:      "main",
			pattern:  "develop",
			isTag:    false,
			expected: false,
		},
		{
			name:     "tags keyword - is tag",
			ref:      "v1.0.0",
			pattern:  "tags",
			isTag:    true,
			expected: true,
		},
		{
			name:     "tags keyword - not tag",
			ref:      "main",
			pattern:  "tags",
			isTag:    false,
			expected: false,
		},
		{
			name:     "branches keyword - is branch",
			ref:      "main",
			pattern:  "branches",
			isTag:    false,
			expected: true,
		},
		{
			name:     "branches keyword - is tag",
			ref:      "v1.0.0",
			pattern:  "branches",
			isTag:    true,
			expected: false,
		},
		{
			name:     "feature branch exact match",
			ref:      "feature/new-feature",
			pattern:  "feature/new-feature",
			isTag:    false,
			expected: true,
		},
		{
			name:     "tag version exact match",
			ref:      "v1.0.0",
			pattern:  "v1.0.0",
			isTag:    true,
			expected: true,
		},
		{
			name:     "tag version no match",
			ref:      "v1.0.0",
			pattern:  "v2.0.0",
			isTag:    true,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchRef(tt.ref, tt.pattern, tt.isTag)
			if result != tt.expected {
				t.Errorf("matchRef(%s, %s, %v) = %v, want %v", tt.ref, tt.pattern, tt.isTag, result, tt.expected)
			}
		})
	}
}

// TestCreatePipeline_ValidYAML tests normal Pipeline creation flow
func TestCreatePipeline_ValidYAML(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	service := NewPipelineService(db)

	yamlConfig := `
stages:
  - build
  - test

build:
  stage: build
  script:
    - echo "Building..."

test:
  stage: test
  script:
    - echo "Testing..."
`

	pipeline, err := service.CreatePipeline(
		"testuser",
		"testrepo",
		"push",
		"refs/heads/main",
		"abc123def456",
		"guanshiliang",
		"test commit",
		yamlConfig,
		nil,
	)

	if err != nil {
		t.Fatalf("CreatePipeline failed: %v", err)
	}

	if pipeline.ID == 0 {
		t.Error("Pipeline ID should not be 0")
	}

	if pipeline.Status != model.PipelineStatusPending {
		t.Errorf("Pipeline status = %s, want %s", pipeline.Status, model.PipelineStatusPending)
	}

	if pipeline.UserName != "testuser" {
		t.Errorf("Pipeline UserName = %s, want testuser", pipeline.UserName)
	}

	if pipeline.RepositoryName != "testrepo" {
		t.Errorf("Pipeline RepositoryName = %s, want testrepo", pipeline.RepositoryName)
	}

	// Verify Job record created
	var jobs []model.Job
	if err := db.Where("pipeline_id = ?", pipeline.ID).Find(&jobs).Error; err != nil {
		t.Fatalf("Failed to query jobs: %v", err)
	}

	if len(jobs) != 2 {
		t.Errorf("Expected 2 jobs, got %d", len(jobs))
	}

	// Verify job status
	for _, job := range jobs {
		if job.Status != model.JobStatusPending {
			t.Errorf("Job %s status = %s, want %s", job.JobName, job.Status, model.JobStatusPending)
		}
	}
}

// TestCreatePipeline_InvalidYAML tests invalid YAML config
func TestCreatePipeline_InvalidYAML(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	service := NewPipelineService(db)

	// Invalid YAML
	yamlConfig := `
invalid: yaml: content: [
`

	pipeline, err := service.CreatePipeline(
		"testuser",
		"testrepo",
		"push",
		"refs/heads/main",
		"abc123def456",
		"guanshiliang",
		"test commit",
		yamlConfig,
		nil,
	)

	if err != nil {
		t.Fatalf("CreatePipeline should not return error for invalid YAML: %v", err)
	}

	// Pipeline should be created with failed status
	if pipeline.Status != model.PipelineStatusFailed {
		t.Errorf("Pipeline status = %s, want %s", pipeline.Status, model.PipelineStatusFailed)
	}
}

// TestCreatePipeline_WithOnlyExcept tests only/except branch filtering
func TestCreatePipeline_WithOnlyExcept(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	service := NewPipelineService(db)

	yamlConfig := `
stages:
  - build
  - deploy

build:
  stage: build
  script:
    - echo "Building..."

deploy:
  stage: deploy
  script:
    - echo "Deploying..."
  only:
    - main
`

	pipeline, err := service.CreatePipeline(
		"testuser",
		"testrepo",
		"push",
		"refs/heads/develop", // develop branch
		"abc123def456",
		"guanshiliang",
		"test commit",
		yamlConfig,
		nil,
	)

	if err != nil {
		t.Fatalf("CreatePipeline failed: %v", err)
	}

	// Verify Job record
	var jobs []model.Job
	if err := db.Where("pipeline_id = ?", pipeline.ID).Find(&jobs).Error; err != nil {
		t.Fatalf("Failed to query jobs: %v", err)
	}

	if len(jobs) != 2 {
		t.Errorf("Expected 2 jobs, got %d", len(jobs))
	}

	// build job should be pending
	// deploy job should be skipped (only: main, but current is develop)
	for _, job := range jobs {
		if job.JobName == "build" {
			if job.Status != model.JobStatusPending {
				t.Errorf("build job status = %s, want %s", job.Status, model.JobStatusPending)
			}
		}
		if job.JobName == "deploy" {
			if job.Status != model.JobStatusSkipped {
				t.Errorf("deploy job status = %s, want %s", job.Status, model.JobStatusSkipped)
			}
		}
	}
}

// TestCreatePipeline_WithVariables tests global and job-level variables
func TestCreatePipeline_WithVariables(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	service := NewPipelineService(db)

	yamlConfig := `
stages:
  - build

variables:
  GO_VERSION: "1.21"
  NODE_ENV: "production"

build:
  stage: build
  script:
    - echo "Building..."
  variables:
    BUILD_TYPE: "release"
`

	pipeline, err := service.CreatePipeline(
		"testuser",
		"testrepo",
		"push",
		"refs/heads/main",
		"abc123def456",
		"guanshiliang",
		"test commit",
		yamlConfig,
		nil,
	)

	if err != nil {
		t.Fatalf("CreatePipeline failed: %v", err)
	}

	// Verify Job record
	var jobs []model.Job
	if err := db.Where("pipeline_id = ?", pipeline.ID).Find(&jobs).Error; err != nil {
		t.Fatalf("Failed to query jobs: %v", err)
	}

	if len(jobs) != 1 {
		t.Fatalf("Expected 1 job, got %d", len(jobs))
	}

	job := jobs[0]

	// Verify variables field contains global and job-level variables
	var vars map[string]string
	if err := json.Unmarshal([]byte(job.Vars), &vars); err != nil {
		t.Fatalf("Failed to unmarshal job variables: %v", err)
	}

	// Should include global variables
	if vars["GO_VERSION"] != "1.21" {
		t.Errorf("GO_VERSION = %s, want 1.21", vars["GO_VERSION"])
	}
	if vars["NODE_ENV"] != "production" {
		t.Errorf("NODE_ENV = %s, want production", vars["NODE_ENV"])
	}

	// Should include job-level variables
	if vars["BUILD_TYPE"] != "release" {
		t.Errorf("BUILD_TYPE = %s, want release", vars["BUILD_TYPE"])
	}
}

// TestCreatePipeline_WithEnvironment tests environment config
func TestCreatePipeline_WithEnvironment(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	service := NewPipelineService(db)

	yamlConfig := `
stages:
  - deploy

deploy:
  stage: deploy
  script:
    - echo "Deploying..."
  environment:
    name: production
`

	pipeline, err := service.CreatePipeline(
		"testuser",
		"testrepo",
		"push",
		"refs/heads/main",
		"abc123def456",
		"guanshiliang",
		"test commit",
		yamlConfig,
		nil,
	)

	if err != nil {
		t.Fatalf("CreatePipeline failed: %v", err)
	}

	// Verify Job record
	var jobs []model.Job
	if err := db.Where("pipeline_id = ?", pipeline.ID).Find(&jobs).Error; err != nil {
		t.Fatalf("Failed to query jobs: %v", err)
	}

	if len(jobs) != 1 {
		t.Fatalf("Expected 1 job, got %d", len(jobs))
	}

	job := jobs[0]

	// Verify environment field
	if job.Environment == nil || *job.Environment != "production" {
		t.Errorf("Job environment = %v, want production", job.Environment)
	}
}

// TestCreatePipeline_WithMergeRequest tests MR-triggered Pipeline
func TestCreatePipeline_WithMergeRequest(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	service := NewPipelineService(db)

	yamlConfig := `
stages:
  - test

test:
  stage: test
  script:
    - echo "Testing..."
`

	mrID := int64(123)
	pipeline, err := service.CreatePipeline(
		"testuser",
		"testrepo",
		"merge_request",
		"refs/heads/feature",
		"abc123def456",
		"guanshiliang",
		"test commit",
		yamlConfig,
		&mrID,
	)

	if err != nil {
		t.Fatalf("CreatePipeline failed: %v", err)
	}

	if pipeline.MergeRequestID == nil {
		t.Error("Pipeline MergeRequestID should not be nil")
	}

	if *pipeline.MergeRequestID != 123 {
		t.Errorf("Pipeline MergeRequestID = %d, want 123", *pipeline.MergeRequestID)
	}

	if pipeline.TriggerEvent != "merge_request" {
		t.Errorf("Pipeline TriggerEvent = %s, want merge_request", pipeline.TriggerEvent)
	}
}

// TestGetPipeline tests getting single Pipeline
func TestGetPipeline(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	service := NewPipelineService(db)

	// Create a Pipeline first
	yamlConfig := `
stages:
  - build

build:
  stage: build
  script:
    - echo "Building..."
`

	created, err := service.CreatePipeline(
		"testuser",
		"testrepo",
		"push",
		"refs/heads/main",
		"abc123def456",
		"guanshiliang",
		"test commit",
		yamlConfig,
		nil,
	)
	if err != nil {
		t.Fatalf("CreatePipeline failed: %v", err)
	}

	// Test getting existing Pipeline
	pipeline, err := service.GetPipeline(created.ID)
	if err != nil {
		t.Fatalf("GetPipeline failed: %v", err)
	}

	if pipeline.ID != created.ID {
		t.Errorf("Pipeline ID = %d, want %d", pipeline.ID, created.ID)
	}

	if pipeline.UserName != "testuser" {
		t.Errorf("Pipeline UserName = %s, want testuser", pipeline.UserName)
	}

	// Test getting non-existent Pipeline
	_, err = service.GetPipeline(99999)
	if err != ErrPipelineNotFound {
		t.Errorf("Expected ErrPipelineNotFound, got %v", err)
	}
}

// TestListPipelines tests listing Pipelines
func TestListPipelines(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	service := NewPipelineService(db)

	yamlConfig := `
stages:
  - build

build:
  stage: build
  script:
    - echo "Building..."
`

	// Create multiple Pipelines
	for i := 0; i < 5; i++ {
		_, err := service.CreatePipeline(
			"testuser",
			"testrepo",
			"push",
			"refs/heads/main",
			"abc123def456",
			"guanshiliang",
			"test commit",
			yamlConfig,
			nil,
		)
		if err != nil {
			t.Fatalf("CreatePipeline failed: %v", err)
		}
	}

	// Test pagination query
	pipelines, total, err := service.ListPipelines("testuser", "testrepo", 1, 10)
	if err != nil {
		t.Fatalf("ListPipelines failed: %v", err)
	}

	if total != 5 {
		t.Errorf("Total = %d, want 5", total)
	}

	if len(pipelines) != 5 {
		t.Errorf("Pipelines count = %d, want 5", len(pipelines))
	}

	// Test second page (should be empty)
	pipelines, total, err = service.ListPipelines("testuser", "testrepo", 2, 10)
	if err != nil {
		t.Fatalf("ListPipelines failed: %v", err)
	}

	if total != 5 {
		t.Errorf("Total = %d, want 5", total)
	}

	if len(pipelines) != 0 {
		t.Errorf("Pipelines count = %d, want 0", len(pipelines))
	}

	// Test 2 items per page
	pipelines, total, err = service.ListPipelines("testuser", "testrepo", 1, 2)
	if err != nil {
		t.Fatalf("ListPipelines failed: %v", err)
	}

	if total != 5 {
		t.Errorf("Total = %d, want 5", total)
	}

	if len(pipelines) != 2 {
		t.Errorf("Pipelines count = %d, want 2", len(pipelines))
	}
}

// TestListJobs tests listing Jobs
func TestListJobs(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	service := NewPipelineService(db)

	yamlConfig := `
stages:
  - build
  - test

build:
  stage: build
  script:
    - echo "Building..."

test:
  stage: test
  script:
    - echo "Testing..."
`

	pipeline, err := service.CreatePipeline(
		"testuser",
		"testrepo",
		"push",
		"refs/heads/main",
		"abc123def456",
		"guanshiliang",
		"test commit",
		yamlConfig,
		nil,
	)
	if err != nil {
		t.Fatalf("CreatePipeline failed: %v", err)
	}

	// Test listing Jobs
	jobs, err := service.ListJobs(pipeline.ID)
	if err != nil {
		t.Fatalf("ListJobs failed: %v", err)
	}

	if len(jobs) != 2 {
		t.Errorf("Jobs count = %d, want 2", len(jobs))
	}

	// Verify Job fields
	for _, job := range jobs {
		if job.PipelineID != pipeline.ID {
			t.Errorf("Job PipelineID = %d, want %d", job.PipelineID, pipeline.ID)
		}
		if job.StageName == "" {
			t.Error("Job StageName should not be empty")
		}
		if job.JobName == "" {
			t.Error("Job JobName should not be empty")
		}
	}
}

// TestGetJob tests getting single Job
func TestGetJob(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	service := NewPipelineService(db)

	yamlConfig := `
stages:
  - build

build:
  stage: build
  script:
    - echo "Building..."
`

	pipeline, err := service.CreatePipeline(
		"testuser",
		"testrepo",
		"push",
		"refs/heads/main",
		"abc123def456",
		"guanshiliang",
		"test commit",
		yamlConfig,
		nil,
	)
	if err != nil {
		t.Fatalf("CreatePipeline failed: %v", err)
	}

	// Get Job
	jobs, err := service.ListJobs(pipeline.ID)
	if err != nil {
		t.Fatalf("ListJobs failed: %v", err)
	}

	if len(jobs) == 0 {
		t.Fatal("No jobs found")
	}

	// Test getting existing Job
	job, err := service.GetJob(jobs[0].ID)
	if err != nil {
		t.Fatalf("GetJob failed: %v", err)
	}

	if job.ID != jobs[0].ID {
		t.Errorf("Job ID = %d, want %d", job.ID, jobs[0].ID)
	}

	// Test getting non-existent Job
	_, err = service.GetJob(99999)
	if err != ErrJobNotFound {
		t.Errorf("Expected ErrJobNotFound, got %v", err)
	}
}

// TestUpdateJobStatus tests updating Job status
func TestUpdateJobStatus(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	service := NewPipelineService(db)

	yamlConfig := `
stages:
  - build

build:
  stage: build
  script:
    - echo "Building..."
`

	pipeline, err := service.CreatePipeline(
		"testuser",
		"testrepo",
		"push",
		"refs/heads/main",
		"abc123def456",
		"guanshiliang",
		"test commit",
		yamlConfig,
		nil,
	)
	if err != nil {
		t.Fatalf("CreatePipeline failed: %v", err)
	}

	// Get Job
	jobs, err := service.ListJobs(pipeline.ID)
	if err != nil {
		t.Fatalf("ListJobs failed: %v", err)
	}

	if len(jobs) == 0 {
		t.Fatal("No jobs found")
	}

	jobID := jobs[0].ID

	// Update status to running
	exitCode := 0
	err = service.UpdateJobStatus(jobID, model.JobStatusRunning, nil, nil)
	if err != nil {
		t.Fatalf("UpdateJobStatus failed: %v", err)
	}

	// Verify status
	job, err := service.GetJob(jobID)
	if err != nil {
		t.Fatalf("GetJob failed: %v", err)
	}

	if job.Status != model.JobStatusRunning {
		t.Errorf("Job status = %s, want %s", job.Status, model.JobStatusRunning)
	}

	if job.StartedAt == nil {
		t.Error("Job StartedAt should not be nil")
	}

	// Update status to success
	err = service.UpdateJobStatus(jobID, model.JobStatusSuccess, &exitCode, nil)
	if err != nil {
		t.Fatalf("UpdateJobStatus failed: %v", err)
	}

	// Verify status
	job, err = service.GetJob(jobID)
	if err != nil {
		t.Fatalf("GetJob failed: %v", err)
	}

	if job.Status != model.JobStatusSuccess {
		t.Errorf("Job status = %s, want %s", job.Status, model.JobStatusSuccess)
	}

	if job.ExitCode == nil || *job.ExitCode != 0 {
		t.Errorf("Job ExitCode = %v, want 0", job.ExitCode)
	}

	if job.FinishedAt == nil {
		t.Error("Job FinishedAt should not be nil")
	}
}

// TestUpdateJobStatus_Failure tests updating Job status to failed
func TestUpdateJobStatus_Failure(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	service := NewPipelineService(db)

	yamlConfig := `
stages:
  - build

build:
  stage: build
  script:
    - echo "Building..."
`

	pipeline, err := service.CreatePipeline(
		"testuser",
		"testrepo",
		"push",
		"refs/heads/main",
		"abc123def456",
		"guanshiliang",
		"test commit",
		yamlConfig,
		nil,
	)
	if err != nil {
		t.Fatalf("CreatePipeline failed: %v", err)
	}

	// Get Job
	jobs, err := service.ListJobs(pipeline.ID)
	if err != nil {
		t.Fatalf("ListJobs failed: %v", err)
	}

	if len(jobs) == 0 {
		t.Fatal("No jobs found")
	}

	jobID := jobs[0].ID

	// Update status to failed
	exitCode := 1
	failureReason := "script_failure"
	err = service.UpdateJobStatus(jobID, model.JobStatusFailed, &exitCode, &failureReason)
	if err != nil {
		t.Fatalf("UpdateJobStatus failed: %v", err)
	}

	// Verify status
	job, err := service.GetJob(jobID)
	if err != nil {
		t.Fatalf("GetJob failed: %v", err)
	}

	if job.Status != model.JobStatusFailed {
		t.Errorf("Job status = %s, want %s", job.Status, model.JobStatusFailed)
	}

	if job.ExitCode == nil || *job.ExitCode != 1 {
		t.Errorf("Job ExitCode = %v, want 1", job.ExitCode)
	}

	if job.FailureReason == nil || *job.FailureReason != "script_failure" {
		t.Errorf("Job FailureReason = %v, want script_failure", job.FailureReason)
	}
}

// TestCancelPipeline tests canceling Pipeline
func TestCancelPipeline(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	service := NewPipelineService(db)

	yamlConfig := `
stages:
  - build
  - test

build:
  stage: build
  script:
    - echo "Building..."

test:
  stage: test
  script:
    - echo "Testing..."
`

	pipeline, err := service.CreatePipeline(
		"testuser",
		"testrepo",
		"push",
		"refs/heads/main",
		"abc123def456",
		"guanshiliang",
		"test commit",
		yamlConfig,
		nil,
	)
	if err != nil {
		t.Fatalf("CreatePipeline failed: %v", err)
	}

	// Cancel Pipeline
	err = service.CancelPipeline(pipeline.ID)
	if err != nil {
		t.Fatalf("CancelPipeline failed: %v", err)
	}

	// Verify Pipeline status
	p, err := service.GetPipeline(pipeline.ID)
	if err != nil {
		t.Fatalf("GetPipeline failed: %v", err)
	}

	if p.Status != model.PipelineStatusCanceled {
		t.Errorf("Pipeline status = %s, want %s", p.Status, model.PipelineStatusCanceled)
	}

	// Verify all Job statuses
	jobs, err := service.ListJobs(pipeline.ID)
	if err != nil {
		t.Fatalf("ListJobs failed: %v", err)
	}

	for _, job := range jobs {
		if job.Status != model.JobStatusCanceled {
			t.Errorf("Job %s status = %s, want %s", job.JobName, job.Status, model.JobStatusCanceled)
		}
		if job.FinishedAt == nil {
			t.Errorf("Job %s FinishedAt should not be nil", job.JobName)
		}
	}
}

// TestRetryPipeline tests retrying Pipeline
func TestRetryPipeline(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	service := NewPipelineService(db)

	yamlConfig := `
stages:
  - build

build:
  stage: build
  script:
    - echo "Building..."
`

	pipeline, err := service.CreatePipeline(
		"testuser",
		"testrepo",
		"push",
		"refs/heads/main",
		"abc123def456",
		"guanshiliang",
		"test commit",
		yamlConfig,
		nil,
	)
	if err != nil {
		t.Fatalf("CreatePipeline failed: %v", err)
	}

	// Get Job
	jobs, err := service.ListJobs(pipeline.ID)
	if err != nil {
		t.Fatalf("ListJobs failed: %v", err)
	}

	if len(jobs) == 0 {
		t.Fatal("No jobs found")
	}

	jobID := jobs[0].ID

	// Update Job status to failed
	exitCode := 1
	failureReason := "script_failure"
	err = service.UpdateJobStatus(jobID, model.JobStatusFailed, &exitCode, &failureReason)
	if err != nil {
		t.Fatalf("UpdateJobStatus failed: %v", err)
	}

	// Wait for async refreshPipelineStatus goroutine (spawned by UpdateJobStatus) to settle,
	// otherwise it races with the manual Pipeline status update and RetryPipeline transaction below,
	// causing SQLite "database table is locked" deadlocks.
	time.Sleep(200 * time.Millisecond)

	// Manually update Pipeline status to failed
	err = db.Model(&model.Pipeline{}).Where("id = ?", pipeline.ID).
		Update("status", model.PipelineStatusFailed).Error
	if err != nil {
		t.Fatalf("Failed to update pipeline status: %v", err)
	}

	// Retry Pipeline
	err = service.RetryPipeline(pipeline.ID)
	if err != nil {
		t.Fatalf("RetryPipeline failed: %v", err)
	}

	// Verify Job status (core logic: failed job reset to pending)
	// Note: do not check pipeline status, async scheduler may have updated it
	jobs, err = service.ListJobs(pipeline.ID)
	if err != nil {
		t.Fatalf("ListJobs failed: %v", err)
	}

	for _, job := range jobs {
		if job.Status != model.JobStatusPending {
			t.Errorf("Job %s status = %s, want %s", job.JobName, job.Status, model.JobStatusPending)
		}
		if job.StartedAt != nil {
			t.Errorf("Job %s StartedAt should be nil", job.JobName)
		}
		if job.FinishedAt != nil {
			t.Errorf("Job %s FinishedAt should be nil", job.JobName)
		}
		if job.ExitCode != nil {
			t.Errorf("Job %s ExitCode should be nil", job.JobName)
		}
		if job.FailureReason != nil {
			t.Errorf("Job %s FailureReason should be nil", job.JobName)
		}
	}
}

// TestParsePipelineYAML tests parsing Pipeline YAML
func TestParsePipelineYAML(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		expectError bool
		checkFunc   func(t *testing.T, p *PipelineYAML)
	}{
		{
			name: "valid YAML with stages and jobs",
			yaml: `
stages:
  - build
  - test
  - deploy

build:
  stage: build
  script:
    - echo "Building..."

test:
  stage: test
  script:
    - go test ./...

deploy:
  stage: deploy
  script:
    - ./deploy.sh
  only:
    - main
`,
			expectError: false,
			checkFunc: func(t *testing.T, p *PipelineYAML) {
				if len(p.Stages) != 3 {
					t.Errorf("Expected 3 stages, got %d", len(p.Stages))
				}
				if len(p.Jobs) != 3 {
					t.Errorf("Expected 3 jobs, got %d", len(p.Jobs))
				}
				if _, ok := p.Jobs["build"]; !ok {
					t.Error("Expected build job")
				}
				if _, ok := p.Jobs["test"]; !ok {
					t.Error("Expected test job")
				}
				if _, ok := p.Jobs["deploy"]; !ok {
					t.Error("Expected deploy job")
				}
			},
		},
		{
			name: "YAML with variables",
			yaml: `
variables:
  GO_VERSION: "1.21"
  NODE_ENV: "production"

build:
  stage: build
  script:
    - echo "Building..."
`,
			expectError: false,
			checkFunc: func(t *testing.T, p *PipelineYAML) {
				if len(p.Variables) != 2 {
					t.Errorf("Expected 2 variables, got %d", len(p.Variables))
				}
				if p.Variables["GO_VERSION"] != "1.21" {
					t.Errorf("GO_VERSION = %s, want 1.21", p.Variables["GO_VERSION"])
				}
				if p.Variables["NODE_ENV"] != "production" {
					t.Errorf("NODE_ENV = %s, want production", p.Variables["NODE_ENV"])
				}
			},
		},
		{
			name: "YAML with job-level variables",
			yaml: `
build:
  stage: build
  script:
    - echo "Building..."
  variables:
    BUILD_TYPE: "release"
`,
			expectError: false,
			checkFunc: func(t *testing.T, p *PipelineYAML) {
				job, ok := p.Jobs["build"]
				if !ok {
					t.Fatal("Expected build job")
				}
				if len(job.Variables) != 1 {
					t.Errorf("Expected 1 job variable, got %d", len(job.Variables))
				}
				if job.Variables["BUILD_TYPE"] != "release" {
					t.Errorf("BUILD_TYPE = %s, want release", job.Variables["BUILD_TYPE"])
				}
			},
		},
		{
			name: "YAML with environment",
			yaml: `
deploy:
  stage: deploy
  script:
    - ./deploy.sh
  environment:
    name: production
    url: https://example.com
`,
			expectError: false,
			checkFunc: func(t *testing.T, p *PipelineYAML) {
				job, ok := p.Jobs["deploy"]
				if !ok {
					t.Fatal("Expected deploy job")
				}
				if job.Environment == nil {
					t.Fatal("Expected environment")
				}
				if job.Environment.Name != "production" {
					t.Errorf("Environment name = %s, want production", job.Environment.Name)
				}
				if job.Environment.URL == nil || *job.Environment.URL != "https://example.com" {
					t.Errorf("Environment URL = %v, want https://example.com", job.Environment.URL)
				}
			},
		},
		{
			name: "YAML with artifacts",
			yaml: `
build:
  stage: build
  script:
    - go build -o bin/app
  artifacts:
    paths:
      - bin/
    expire_in: 1 week
`,
			expectError: false,
			checkFunc: func(t *testing.T, p *PipelineYAML) {
				job, ok := p.Jobs["build"]
				if !ok {
					t.Fatal("Expected build job")
				}
				if job.Artifacts == nil {
					t.Fatal("Expected artifacts")
				}
				if len(job.Artifacts.Paths) != 1 {
					t.Errorf("Expected 1 artifact path, got %d", len(job.Artifacts.Paths))
				}
				if job.Artifacts.ExpireIn != "1 week" {
					t.Errorf("ExpireIn = %s, want 1 week", job.Artifacts.ExpireIn)
				}
			},
		},
		{
			name: "YAML with cache",
			yaml: `
build:
  stage: build
  script:
    - go build
  cache:
    key: go-cache
    paths:
      - .cache/go-build
`,
			expectError: false,
			checkFunc: func(t *testing.T, p *PipelineYAML) {
				job, ok := p.Jobs["build"]
				if !ok {
					t.Fatal("Expected build job")
				}
				if job.Cache == nil {
					t.Fatal("Expected cache")
				}
				if job.Cache.Key != "go-cache" {
					t.Errorf("Cache key = %s, want go-cache", job.Cache.Key)
				}
				if len(job.Cache.Paths) != 1 {
					t.Errorf("Expected 1 cache path, got %d", len(job.Cache.Paths))
				}
			},
		},
		{
			name: "YAML with needs",
			yaml: `
build:
  stage: build
  script:
    - go build

test:
  stage: test
  script:
    - go test
  needs:
    - build
`,
			expectError: false,
			checkFunc: func(t *testing.T, p *PipelineYAML) {
				job, ok := p.Jobs["test"]
				if !ok {
					t.Fatal("Expected test job")
				}
				if len(job.Needs) != 1 {
					t.Errorf("Expected 1 need, got %d", len(job.Needs))
				}
				if job.Needs[0] != "build" {
					t.Errorf("Needs[0] = %s, want build", job.Needs[0])
				}
			},
		},
		{
			name: "YAML with when",
			yaml: `
deploy:
  stage: deploy
  script:
    - ./deploy.sh
  when: manual
`,
			expectError: false,
			checkFunc: func(t *testing.T, p *PipelineYAML) {
				job, ok := p.Jobs["deploy"]
				if !ok {
					t.Fatal("Expected deploy job")
				}
				if job.When != "manual" {
					t.Errorf("When = %s, want manual", job.When)
				}
			},
		},
		{
			name: "YAML with only/except",
			yaml: `
deploy:
  stage: deploy
  script:
    - ./deploy.sh
  only:
    - main
    - tags
  except:
    - develop
`,
			expectError: false,
			checkFunc: func(t *testing.T, p *PipelineYAML) {
				job, ok := p.Jobs["deploy"]
				if !ok {
					t.Fatal("Expected deploy job")
				}
				if len(job.Only) != 2 {
					t.Errorf("Expected 2 only, got %d", len(job.Only))
				}
				if len(job.Except) != 1 {
					t.Errorf("Expected 1 except, got %d", len(job.Except))
				}
			},
		},
		{
			name: "YAML without stages (auto-detect)",
			yaml: `
build:
  stage: build
  script:
    - echo "Building..."

test:
  stage: test
  script:
    - echo "Testing..."
`,
			expectError: false,
			checkFunc: func(t *testing.T, p *PipelineYAML) {
				if len(p.Stages) != 2 {
					t.Errorf("Expected 2 stages, got %d", len(p.Stages))
				}
			},
		},
		{
			name: "YAML with job without stage (default to test)",
			yaml: `
build:
  script:
    - echo "Building..."
`,
			expectError: false,
			checkFunc: func(t *testing.T, p *PipelineYAML) {
				job, ok := p.Jobs["build"]
				if !ok {
					t.Fatal("Expected build job")
				}
				if job.Stage != "test" {
					t.Errorf("Stage = %s, want test", job.Stage)
				}
			},
		},
		{
			name:        "invalid YAML",
			yaml:        `invalid: yaml: content: [`,
			expectError: true,
		},
		{
			name:        "empty YAML",
			yaml:        ``,
			expectError: false,
			checkFunc: func(t *testing.T, p *PipelineYAML) {
				if len(p.Jobs) != 0 {
					t.Errorf("Expected 0 jobs, got %d", len(p.Jobs))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := ParsePipelineYAML([]byte(tt.yaml))

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, p)
			}
		})
	}
}
