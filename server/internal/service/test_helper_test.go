package service

import (
	"fmt"
	"sync/atomic"
	"testing"

	"iforge/iforge/internal/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// testDBCounter generates unique database names
var testDBCounter int64

// setupTestDB creates an isolated in-memory SQLite database for testing (pure Go driver, no CGO)
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// Use unique DB name per test to avoid data sharing
	dbID := atomic.AddInt64(&testDBCounter, 1)
	dsn := fmt.Sprintf("file:testdb%d?mode=memory&cache=shared", dbID)

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Migrate all required tables
	if err := migrateTestTables(db); err != nil {
		t.Fatalf("Failed to migrate test tables: %v", err)
	}

	return db
}

// migrateTestTables migrates tables required for testing
func migrateTestTables(db *gorm.DB) error {
	// Use AutoMigrate to create table schemas
	// Migrate all tables needed for testing
	return db.AutoMigrate(
		// CI/CD tables
		&model.Pipeline{},
		&model.Job{},
		&model.JobLog{},
		&model.Artifact{},
		&model.Runner{},
		&model.Secret{},
		&model.Environment{},
		&model.Deployment{},
		&model.CronSchedule{},
		// Account tables
		&model.Account{},
		&model.OrganizationMember{},
		&model.AccountPreference{},
		&model.AccountExtraMailAddress{},
		&model.SSHKey{},
		&model.GPGKey{},
		&model.DeployKey{},
		&model.AccessToken{},
		&model.Collaborator{},
		// Repository tables
		&model.Repository{},
		&model.IssueIDCounter{}, // Issue tables
		&model.Issue{},
		&model.IssueComment{},
		&model.IssueLabel{},
		&model.IssueAssignment{},
		&model.Label{},
		&model.Milestone{},
		&model.Priority{},
		// Project tables
		&model.Project{},
		&model.ProjectMember{},
		&model.ProjectRepository{},
		&model.Sprint{},
		&model.UserStory{},
		&model.Task{},
		&model.TaskIDCounter{},
		&model.TaskComment{},
		&model.TaskStatus{},
		&model.TaskStatusHistory{},
		&model.TaskStatusTransition{},
		&model.TaskLabel{},
		&model.TaskLabelAssignment{},
		&model.TaskAssignment{},
		&model.TaskDependency{},
		// Epic tables
		&model.Epic{},
		// Star and Watch tables
		&model.RepositoryStar{},
		&model.RepositoryWatch{},
		// MergeRequest tables
		&model.MergeRequest{},
		// Webhook tables
		&model.Webhook{},
		&model.WebhookDelivery{},
		// Review tables
		&model.Review{},
		&model.ReviewComment{},
		// Release tables
		&model.ReleaseTag{},
		&model.ReleaseAsset{},
		// CommitStatus tables
		&model.CommitStatus{},
		// Activity tables
		&model.Activity{},
		// Audit tables
		&model.AuditLog{},
		// Notification tables
		&model.Notification{},
		// Wiki tables
		&WikiPage{},
		// SystemSettings tables
		&model.SystemSetting{},
		// PasswordReset tables
		&model.PasswordResetToken{},
		// CustomField tables
		&model.CustomField{},
		&model.IssueCustomField{},
		// StatsCache tables
		&model.RepositoryStatsCache{},
		// AIModelConfig tables
		&model.AIModelConfig{},
		// ScrumActivity tables
		&model.ScrumActivity{},
		// LFS tables
		&model.LFSObject{},
		// Mirror tables
		&model.RepositoryMirror{},
		// Plugin tables
		&model.Plugin{},
		&model.PluginEvent{},
	)
}

// cleanupTestDB cleans up the test database
func cleanupTestDB(db *gorm.DB) {
	// In-memory databases are automatically cleaned up on connection close
	// Additional cleanup logic can be added here
}
