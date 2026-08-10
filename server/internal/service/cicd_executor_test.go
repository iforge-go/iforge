package service

import (
	"os"
	"path/filepath"
	"testing"

	gitsvc "iforge/iforge/internal/git"
)

func TestNewExecutor(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	executor := NewExecutor(db)
	if executor == nil {
		t.Fatal("NewExecutor returned nil")
	}

	if executor.db != db {
		t.Error("Executor.db not set correctly")
	}

	if executor.queue == nil {
		t.Error("Executor.queue is nil")
	}

	if executor.stopCh == nil {
		t.Error("Executor.stopCh is nil")
	}

	if executor.workerCount != 4 {
		t.Errorf("Executor.workerCount = %d, want 4", executor.workerCount)
	}
}

func TestNewExecutor_WithEnvWorkers(t *testing.T) {
	// Set environment variable
	os.Setenv("IFORGE_CI_WORKERS", "8")
	defer os.Unsetenv("IFORGE_CI_WORKERS")

	db := setupTestDB(t)
	defer cleanupTestDB(db)

	executor := NewExecutor(db)
	if executor.workerCount != 8 {
		t.Errorf("Executor.workerCount = %d, want 8", executor.workerCount)
	}
}

func TestNewExecutor_InvalidEnvWorkers(t *testing.T) {
	// Set invalid environment variable
	os.Setenv("IFORGE_CI_WORKERS", "invalid")
	defer os.Unsetenv("IFORGE_CI_WORKERS")

	db := setupTestDB(t)
	defer cleanupTestDB(db)

	executor := NewExecutor(db)
	// Should use default value 4
	if executor.workerCount != 4 {
		t.Errorf("Executor.workerCount = %d, want 4 (default)", executor.workerCount)
	}
}

func TestExecutor_SetSecretService(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	executor := NewExecutor(db)
	secretService := &SecretService{db: db}

	executor.SetSecretService(secretService)

	if executor.secretService != secretService {
		t.Error("Executor.secretService not set correctly")
	}
}

func TestExecutor_SetGitClient(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	executor := NewExecutor(db)
	gitClient := &gitsvc.Client{}

	executor.SetGitClient(gitClient)

	if executor.gitClient != gitClient {
		t.Error("Executor.gitClient not set correctly")
	}
}

func TestExecutor_SetNotificationService(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	executor := NewExecutor(db)
	notificationService := &NotificationService{db: db}

	executor.SetNotificationService(notificationService)

	if executor.notificationService != notificationService {
		t.Error("Executor.notificationService not set correctly")
	}
}

func TestExecutor_SetMailService(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	executor := NewExecutor(db)
	mailService := &MailService{}

	executor.SetMailService(mailService)

	if executor.mailService != mailService {
		t.Error("Executor.mailService not set correctly")
	}
}

func TestExecutor_SetWebhookService(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	executor := NewExecutor(db)
	webhookService := &WebhookService{db: db}

	executor.SetWebhookService(webhookService)

	if executor.webhookService != webhookService {
		t.Error("Executor.webhookService not set correctly")
	}
}

func TestExecutor_SetReleaseService(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	executor := NewExecutor(db)
	releaseService := &ReleaseService{db: db}

	executor.SetReleaseService(releaseService, "/tmp/releases")

	if executor.releaseService != releaseService {
		t.Error("Executor.releaseService not set correctly")
	}

	if executor.releaseUploadDir != "/tmp/releases" {
		t.Errorf("Executor.releaseUploadDir = %s, want /tmp/releases", executor.releaseUploadDir)
	}
}

func TestExecutor_SetDataDir(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	executor := NewExecutor(db)
	executor.SetDataDir("/tmp/data")

	if executor.dataDir != "/tmp/data" {
		t.Errorf("Executor.dataDir = %s, want /tmp/data", executor.dataDir)
	}
}

func TestExecutor_ArtifactsDir(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	executor := NewExecutor(db)
	executor.SetDataDir("/tmp/data")

	artifactsDir := executor.artifactsDir()
	expected := filepath.Join("/tmp/data", "cicd-artifacts")

	if artifactsDir != expected {
		t.Errorf("Executor.artifactsDir() = %s, want %s", artifactsDir, expected)
	}
}

func TestExecutor_CacheDir(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	executor := NewExecutor(db)
	executor.SetDataDir("/tmp/data")

	cacheDir := executor.cacheDir()
	expected := filepath.Join("/tmp/data", "cicd-cache")

	if cacheDir != expected {
		t.Errorf("Executor.cacheDir() = %s, want %s", cacheDir, expected)
	}
}
