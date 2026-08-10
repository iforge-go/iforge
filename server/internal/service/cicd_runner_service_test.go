package service

import (
	"testing"
	"time"

	"iforge/iforge/internal/model"
)

func TestRunnerService_RegisterExternalRunner(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewRunnerService(db)

	runner, token, err := service.RegisterExternalRunner(
		"external-runner-1",
		"shell",
		"Test external runner",
		"192.168.1.100",
	)

	if err != nil {
		t.Fatalf("RegisterExternalRunner failed: %v", err)
	}

	if runner.ID == 0 {
		t.Error("Expected runner ID to be set")
	}
	if runner.Name != "external-runner-1" {
		t.Errorf("Expected Name 'external-runner-1', got '%s'", runner.Name)
	}
	if token == "" {
		t.Error("Expected token to be generated")
	}
	if runner.Status != model.RunnerStatusOffline {
		t.Errorf("Expected status offline, got '%s'", runner.Status)
	}
	if runner.IsBuiltin {
		t.Error("Expected IsBuiltin false for external runner")
	}
	if runner.IPAddress != "192.168.1.100" {
		t.Errorf("Expected IPAddress '192.168.1.100', got '%s'", runner.IPAddress)
	}
}

func TestRunnerService_GetRunnerByID(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewRunnerService(db)

	// Register runner
	created, _, _ := service.RegisterExternalRunner("test-runner", "shell", "Test", "127.0.0.1")

	// Get runner
	runner, err := service.GetRunnerByID(created.ID)
	if err != nil {
		t.Fatalf("GetRunnerByID failed: %v", err)
	}

	if runner.Name != "test-runner" {
		t.Errorf("Expected Name 'test-runner', got '%s'", runner.Name)
	}
}

func TestRunnerService_GetRunnerByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewRunnerService(db)

	_, err := service.GetRunnerByID(999)
	if err != ErrRunnerNotFound {
		t.Errorf("Expected ErrRunnerNotFound, got %v", err)
	}
}

func TestRunnerService_DeleteRunner(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewRunnerService(db)

	// Register external runner
	created, _, _ := service.RegisterExternalRunner("test-runner", "shell", "Test", "127.0.0.1")

	// Delete runner
	err := service.DeleteRunner(created.ID)
	if err != nil {
		t.Fatalf("DeleteRunner failed: %v", err)
	}

	// Verify deleted
	_, err = service.GetRunnerByID(created.ID)
	if err != ErrRunnerNotFound {
		t.Error("Expected runner to be deleted")
	}
}

func TestRunnerService_DeleteRunner_Builtin(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewRunnerService(db)

	// Create built-in runner
	builtinRunner := &model.Runner{
		Name:      "builtin-runner",
		TokenHash: "hash",
		Status:    model.RunnerStatusOnline,
		Type:      "shell",
		IsBuiltin: true,
		MaxJobs:   1,
		CreatedAt: time.Now(),
	}
	db.Create(builtinRunner)

	// Try deleting built-in runner
	err := service.DeleteRunner(builtinRunner.ID)
	if err == nil {
		t.Error("Expected error when deleting builtin runner")
	}
}

func TestRunnerService_DeleteRunner_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewRunnerService(db)

	err := service.DeleteRunner(999)
	if err != ErrRunnerNotFound {
		t.Errorf("Expected ErrRunnerNotFound, got %v", err)
	}
}

func TestRunnerService_UpdateRunnerStatus(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewRunnerService(db)

	// Register runner
	created, _, _ := service.RegisterExternalRunner("test-runner", "shell", "Test", "127.0.0.1")

	// Update status to online
	err := service.UpdateRunnerStatus(created.ID, "online")
	if err != nil {
		t.Fatalf("UpdateRunnerStatus failed: %v", err)
	}

	// Verify status updated
	runner, _ := service.GetRunnerByID(created.ID)
	if runner.Status != "online" {
		t.Errorf("Expected status 'online', got '%s'", runner.Status)
	}
	if runner.LastHeartbeat == nil {
		t.Error("Expected LastHeartbeat to be set")
	}
}

func TestRunnerService_CheckStaleRunners(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewRunnerService(db)

	// Create online but stale runner
	oldTime := time.Now().Add(-120 * time.Second)
	staleRunner := &model.Runner{
		Name:          "stale-runner",
		TokenHash:     "hash",
		Status:        "online",
		Type:          "shell",
		IsBuiltin:     false,
		MaxJobs:       1,
		LastHeartbeat: &oldTime,
		CreatedAt:     time.Now(),
	}
	db.Create(staleRunner)

	// Create online and active runner
	activeRunner := &model.Runner{
		Name:          "active-runner",
		TokenHash:     "hash2",
		Status:        "online",
		Type:          "shell",
		IsBuiltin:     false,
		MaxJobs:       1,
		LastHeartbeat: func() *time.Time { t := time.Now(); return &t }(),
		CreatedAt:     time.Now(),
	}
	db.Create(activeRunner)

	// Check stale runner
	err := service.CheckStaleRunners()
	if err != nil {
		t.Fatalf("CheckStaleRunners failed: %v", err)
	}

	// Verify stale runner marked as offline
	stale, _ := service.GetRunnerByID(staleRunner.ID)
	if stale.Status != "offline" {
		t.Errorf("Expected stale runner to be offline, got '%s'", stale.Status)
	}

	// Verify active runner still online
	active, _ := service.GetRunnerByID(activeRunner.ID)
	if active.Status != "online" {
		t.Errorf("Expected active runner to be online, got '%s'", active.Status)
	}
}

func TestRunnerService_ClaimJobForRunner_NoJobs(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewRunnerService(db)

	// Register runner
	runner, _, _ := service.RegisterExternalRunner("test-runner", "shell", "Test", "127.0.0.1")

	// Try claiming job (no pending job)
	_, err := service.ClaimJobForRunner(runner.ID, nil, "http://localhost:8081")
	if err != ErrNoJobAvailable {
		t.Errorf("Expected ErrNoJobAvailable, got %v", err)
	}
}

func TestRunnerService_ClaimJobForRunner_WithJob(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewRunnerService(db)

	// Create pipeline
	pipeline := &model.Pipeline{
		UserName:       "testuser",
		RepositoryName: "testrepo",
		Ref:            "main",
		CommitSHA:      "abc123",
		Message:        "test commit",
		TriggerEvent:   "push",
		TriggeredBy:    "testuser",
		Status:         "running",
		CreatedAt:      time.Now(),
	}
	db.Create(pipeline)

	// Create pending job
	job := &model.Job{
		PipelineID: pipeline.ID,
		JobName:    "build",
		StageName:  "build",
		Status:     model.JobStatusPending,
		CreatedAt:  time.Now(),
	}
	db.Create(job)

	// Register runner
	runner, _, _ := service.RegisterExternalRunner("test-runner", "shell", "Test", "127.0.0.1")

	// Claim job
	response, err := service.ClaimJobForRunner(runner.ID, nil, "http://localhost:8081")
	if err != nil {
		t.Fatalf("ClaimJobForRunner failed: %v", err)
	}

	if response.Job.ID != job.ID {
		t.Errorf("Expected job ID %d, got %d", job.ID, response.Job.ID)
	}
	if response.Pipeline.ID != pipeline.ID {
		t.Errorf("Expected pipeline ID %d, got %d", pipeline.ID, response.Pipeline.ID)
	}
	if response.CloneURL != "http://localhost:8081/testuser/testrepo.git" {
		t.Errorf("Expected CloneURL 'http://localhost:8081/testuser/testrepo.git', got '%s'", response.CloneURL)
	}

	// Verify job status updated to running
	var updatedJob model.Job
	db.First(&updatedJob, job.ID)
	if updatedJob.Status != model.JobStatusRunning {
		t.Errorf("Expected job status running, got '%s'", updatedJob.Status)
	}
	if updatedJob.RunnerID == nil || *updatedJob.RunnerID != runner.ID {
		t.Error("Expected job to be assigned to runner")
	}
}

func TestRunnerService_UpdateJobFromRunner(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewRunnerService(db)

	// Create pipeline and job
	pipeline := &model.Pipeline{
		UserName:       "testuser",
		RepositoryName: "testrepo",
		Ref:            "main",
		CommitSHA:      "abc123",
		Message:        "test",
		TriggerEvent:   "push",
		TriggeredBy:    "testuser",
		Status:         "running",
		CreatedAt:      time.Now(),
	}
	db.Create(pipeline)

	runnerID := int64(1)
	job := &model.Job{
		PipelineID: pipeline.ID,
		JobName:    "build",
		StageName:  "build",
		Status:     model.JobStatusRunning,
		RunnerID:   &runnerID,
		CreatedAt:  time.Now(),
	}
	db.Create(job)

	// Update job status to success
	exitCode := 0
	err := service.UpdateJobFromRunner(job.ID, runnerID, model.JobStatusSuccess, &exitCode, nil)
	if err != nil {
		t.Fatalf("UpdateJobFromRunner failed: %v", err)
	}

	// Verify status updated
	var updatedJob model.Job
	db.First(&updatedJob, job.ID)
	if updatedJob.Status != model.JobStatusSuccess {
		t.Errorf("Expected status success, got '%s'", updatedJob.Status)
	}
	if updatedJob.ExitCode == nil || *updatedJob.ExitCode != 0 {
		t.Error("Expected exit code 0")
	}
}

func TestRunnerService_UpdateJobFromRunner_WrongRunner(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewRunnerService(db)

	// Create job assigned to runner 1
	runnerID := int64(1)
	pipeline := &model.Pipeline{
		UserName:       "testuser",
		RepositoryName: "testrepo",
		Ref:            "main",
		CommitSHA:      "abc123",
		Message:        "test",
		TriggerEvent:   "push",
		TriggeredBy:    "testuser",
		Status:         "running",
		CreatedAt:      time.Now(),
	}
	db.Create(pipeline)

	job := &model.Job{
		PipelineID: pipeline.ID,
		JobName:    "build",
		StageName:  "build",
		Status:     model.JobStatusRunning,
		RunnerID:   &runnerID,
		CreatedAt:  time.Now(),
	}
	db.Create(job)

	// Try updating with runner 2
	wrongRunnerID := int64(2)
	err := service.UpdateJobFromRunner(job.ID, wrongRunnerID, model.JobStatusSuccess, nil, nil)
	if err == nil {
		t.Error("Expected error when wrong runner tries to update job")
	}
}

func TestRunnerService_AppendJobLogsFromRunner(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewRunnerService(db)

	// Create job
	runnerID := int64(1)
	pipeline := &model.Pipeline{
		UserName:       "testuser",
		RepositoryName: "testrepo",
		Ref:            "main",
		CommitSHA:      "abc123",
		Message:        "test",
		TriggerEvent:   "push",
		TriggeredBy:    "testuser",
		Status:         "running",
		CreatedAt:      time.Now(),
	}
	db.Create(pipeline)

	job := &model.Job{
		PipelineID: pipeline.ID,
		JobName:    "build",
		StageName:  "build",
		Status:     model.JobStatusRunning,
		RunnerID:   &runnerID,
		CreatedAt:  time.Now(),
	}
	db.Create(job)

	// Batch upload logs
	logs := []JobLogBatch{
		{LineNo: 1, Content: "Building...", Stream: "stdout"},
		{LineNo: 2, Content: "Step 1/3", Stream: "stdout"},
		{LineNo: 3, Content: "Warning: deprecated", Stream: "stderr"},
	}

	err := service.AppendJobLogsFromRunner(job.ID, runnerID, logs)
	if err != nil {
		t.Fatalf("AppendJobLogsFromRunner failed: %v", err)
	}

	// Verify logs saved
	var logCount int64
	db.Model(&model.JobLog{}).Where("job_id = ?", job.ID).Count(&logCount)
	if logCount != 3 {
		t.Errorf("Expected 3 log entries, got %d", logCount)
	}
}

func TestRunnerService_AppendJobLogsFromRunner_WrongRunner(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewRunnerService(db)

	// Create job assigned to runner 1
	runnerID := int64(1)
	pipeline := &model.Pipeline{
		UserName:       "testuser",
		RepositoryName: "testrepo",
		Ref:            "main",
		CommitSHA:      "abc123",
		Message:        "test",
		TriggerEvent:   "push",
		TriggeredBy:    "testuser",
		Status:         "running",
		CreatedAt:      time.Now(),
	}
	db.Create(pipeline)

	job := &model.Job{
		PipelineID: pipeline.ID,
		JobName:    "build",
		StageName:  "build",
		Status:     model.JobStatusRunning,
		RunnerID:   &runnerID,
		CreatedAt:  time.Now(),
	}
	db.Create(job)

	// Try uploading logs with runner 2
	wrongRunnerID := int64(2)
	logs := []JobLogBatch{{LineNo: 1, Content: "test", Stream: "stdout"}}
	err := service.AppendJobLogsFromRunner(job.ID, wrongRunnerID, logs)
	if err == nil {
		t.Error("Expected error when wrong runner tries to append logs")
	}
}

func TestBuildJobEnvVars(t *testing.T) {
	pipeline := model.Pipeline{
		ID:             1,
		UserName:       "testuser",
		RepositoryName: "testrepo",
		Ref:            "main",
		CommitSHA:      "abc123",
		Message:        "test commit",
		TriggerEvent:   "push",
		TriggeredBy:    "testuser",
	}

	job := model.Job{
		ID:        10,
		JobName:   "build",
		StageName: "build",
		Vars:      `{"CUSTOM_VAR":"custom_value"}`,
	}

	envs := BuildJobEnvVars(nil, pipeline, job)

	// Verify basic environment variables
	if envs["CI"] != "true" {
		t.Error("Expected CI=true")
	}
	if envs["IFORGE"] != "true" {
		t.Error("Expected IFORGE=true")
	}
	if envs["CI_PIPELINE_ID"] != "1" {
		t.Errorf("Expected CI_PIPELINE_ID='1', got '%s'", envs["CI_PIPELINE_ID"])
	}
	if envs["CI_JOB_ID"] != "10" {
		t.Errorf("Expected CI_JOB_ID='10', got '%s'", envs["CI_JOB_ID"])
	}
	if envs["CI_JOB_NAME"] != "build" {
		t.Errorf("Expected CI_JOB_NAME='build', got '%s'", envs["CI_JOB_NAME"])
	}
	if envs["CI_REPO_OWNER"] != "testuser" {
		t.Errorf("Expected CI_REPO_OWNER='testuser', got '%s'", envs["CI_REPO_OWNER"])
	}
	if envs["CI_REPO_NAME"] != "testrepo" {
		t.Errorf("Expected CI_REPO_NAME='testrepo', got '%s'", envs["CI_REPO_NAME"])
	}

	// Verify custom variables
	if envs["CUSTOM_VAR"] != "custom_value" {
		t.Errorf("Expected CUSTOM_VAR='custom_value', got '%s'", envs["CUSTOM_VAR"])
	}
}

func TestBuildJobEnvVars_EmptyVars(t *testing.T) {
	pipeline := model.Pipeline{
		ID:             1,
		UserName:       "testuser",
		RepositoryName: "testrepo",
		Ref:            "main",
		CommitSHA:      "abc123",
		Message:        "test",
		TriggerEvent:   "push",
		TriggeredBy:    "testuser",
	}

	job := model.Job{
		ID:        10,
		JobName:   "build",
		StageName: "build",
		Vars:      "",
	}

	envs := BuildJobEnvVars(nil, pipeline, job)

	// Should execute normally, should not fail due to empty vars
	if envs["CI"] != "true" {
		t.Error("Expected CI=true")
	}
}

func TestBuildJobEnvVars_InvalidJSON(t *testing.T) {
	pipeline := model.Pipeline{
		ID:             1,
		UserName:       "testuser",
		RepositoryName: "testrepo",
		Ref:            "main",
		CommitSHA:      "abc123",
		Message:        "test",
		TriggerEvent:   "push",
		TriggeredBy:    "testuser",
	}

	job := model.Job{
		ID:        10,
		JobName:   "build",
		StageName: "build",
		Vars:      "invalid json",
	}

	envs := BuildJobEnvVars(nil, pipeline, job)

	// Should ignore invalid JSON and continue returning basic environment variables
	if envs["CI"] != "true" {
		t.Error("Expected CI=true")
	}
}

