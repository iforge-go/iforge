package database

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"iforge/iforge/internal/model"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var gormDB *gorm.DB

// PoolConfig controls the underlying database/sql connection pool. Values less
// than or equal to zero leave the corresponding database/sql default intact.
// SQLite is deliberately exempt: it always uses one connection because writes
// to a SQLite database are serialized.
type PoolConfig struct {
	MaxIdleConns int
	MaxOpenConns int
	MaxLifetime  time.Duration
	MaxIdleTime  time.Duration
}

// InitGORMDB initializes the GORM database connection
func InitGORMDB(driver, dsn string, poolConfig PoolConfig) error {
	var dialector gorm.Dialector
	switch driver {
	case "sqlite":
		dialector = sqlite.Open(dsn)
	case "mysql":
		dialector = mysql.Open(dsn)
	case "postgres":
		dialector = postgres.Open(dsn)
	default:
		return fmt.Errorf("unsupported driver for GORM: %s", driver)
	}

	var err error
	gormDB, err = gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect GORM database: %w", err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return fmt.Errorf("failed to access SQL connection pool: %w", err)
	}

	// SQLite performance optimization: enable WAL mode + busy_timeout to avoid full-table locks on writes.
	// The default DELETE journal mode locks the entire database for any write, causing concurrent
	// pushes / issue creation to serialize and hit "database is locked" errors.
	// WAL mode allows concurrent reads; busy_timeout auto-retries lock waits instead of failing immediately.
	if driver == "sqlite" {
		pragmas := []string{
			"PRAGMA journal_mode=WAL;",
			"PRAGMA busy_timeout=5000;",  // 5 second lock wait
			"PRAGMA synchronous=NORMAL;", // NORMAL is safe and faster under WAL mode
			"PRAGMA foreign_keys=ON;",
		}
		for _, p := range pragmas {
			if execErr := gormDB.Exec(p).Error; execErr != nil {
				log.Printf("warn: failed to set SQLite pragma %q: %v", p, execErr)
			}
		}
		// Connection pool: SQLite writes are serial, so a single connection suffices; reads can be concurrent.
		// MaxOpenConns=1 avoids SQLITE_BUSY from multi-connection lock contention,
		// serializing all DB operations through GORM's connection pool.
		// Deferred reads still benefit from WAL's concurrent read capability, much faster than DELETE mode.
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
	} else {
		configureConnectionPool(sqlDB, poolConfig)
	}

	log.Printf("Connected to database with GORM: %s", driver)
	return nil
}

// configureConnectionPool applies production-safe pool limits for networked
// databases. Keeping this in one place prevents configuration values from
// silently being ignored by a future database driver.
func configureConnectionPool(sqlDB interface {
	SetMaxIdleConns(int)
	SetMaxOpenConns(int)
	SetConnMaxLifetime(time.Duration)
	SetConnMaxIdleTime(time.Duration)
}, config PoolConfig) {
	if config.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	}
	if config.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	}
	if config.MaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(config.MaxLifetime)
	}
	if config.MaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(config.MaxIdleTime)
	}
}

// GetGORMDB returns the GORM database connection
func GetGORMDB() *gorm.DB {
	return gormDB
}

// CloseGORMDB closes the GORM database connection
func CloseGORMDB() error {
	if gormDB != nil {
		sqlDB, err := gormDB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// AutoMigrate runs database migrations using GORM
func AutoMigrate() error {
	err := gormDB.AutoMigrate(
		&model.Account{},
		&model.AccountExtraMailAddress{},
		&model.AccountPreference{},
		&model.Repository{},
		&model.Collaborator{},
		&model.RepositoryStar{},
		&model.RepositoryWatch{},
		&model.RepositoryMirror{},
		&model.ProtectedBranch{},
		&model.DeployKey{},
		&model.Issue{},
		&model.IssueIDCounter{},
		&model.IssueComment{},
		&model.CommitComment{},
		&model.Label{},
		&model.IssueLabel{},
		&model.IssueAssignment{},
		&model.LFSObject{},
		&model.Milestone{},
		&model.Priority{},
		&model.MergeRequest{},
		&model.Webhook{},
		&model.AccessToken{},
		&model.Activity{},
		&model.ReleaseTag{},
		&model.ReleaseAsset{},
		&model.RepositoryStatsCache{}, // Repository stats cache (avoids traversing git history on every request)
		&model.SSHKey{},
		&model.GPGKey{},
		&model.CommitStatus{},
		&model.OrganizationMember{},
		&model.WikiPage{},
		&model.Notification{},
		&model.Review{},
		&model.ReviewComment{},
		&model.CustomField{},
		&model.IssueCustomField{},
		&model.PasswordResetToken{},
		&model.SystemSetting{},
		&model.AIModelConfig{}, // Multi AI model configuration (replaces ai.* keys in system_settings)
		&model.Plugin{},
		&model.PluginEvent{},
		&model.PluginTemplate{},
		&model.PluginTrigger{},
		&model.WebhookDelivery{},
		// Project management models
		&model.Project{},
		&model.ProjectMember{},
		&model.ProjectRepository{},
		&model.Sprint{},
		&model.Epic{},
		&model.UserStory{},
		&model.Task{},
		&model.TaskIDCounter{},
		&model.TaskComment{},
		&model.TaskLabel{},
		&model.TaskLabelAssignment{},
		&model.TaskAssignment{},
		&model.TaskStatus{},
		&model.TaskStatusTransition{},
		&model.TaskStatusHistory{},
		&model.TaskBranch{},
		&model.TaskDependency{},
		&model.TaskWorkLog{},
		&model.ScrumActivity{},
		// Security & compliance
		&model.AuditLog{},
		// CI/CD (Pipeline/Job/Runner/Deployment/Cron full pipeline)
		&model.Pipeline{},
		&model.Job{},
		&model.JobLog{},
		&model.Artifact{},
		&model.Runner{},
		&model.Secret{},
		&model.Environment{},
		&model.Deployment{},
		&model.CronSchedule{},
	)
	if err != nil {
		return fmt.Errorf("failed to auto migrate: %w", err)
	}

	// Under MySQL, TEXT columns are capped at 64KB, but webhook delivery HTTP bodies
	// (large push event payloads can reach several MB), stats_json cache, and webhook events
	// configuration may exceed this. SQLite and PostgreSQL text types have no such limit.
	// After AutoMigrate, upgrade large MySQL columns to LONGTEXT (4GB).
	// MODIFY COLUMN is idempotent: it succeeds even if the column is already LONGTEXT.
	if gormDB.Dialector.Name() == "mysql" {
		largeTextColumns := []struct {
			Table  string
			Column string
		}{
			{"webhook_deliveries", "request_body"},
			{"webhook_deliveries", "response_body"},
			{"webhooks", "events"},
			{"repository_stats_cache", "stats_json"},
		}
		for _, c := range largeTextColumns {
			if err := gormDB.Exec(fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s LONGTEXT", c.Table, c.Column)).Error; err != nil {
				log.Printf("warn: failed to upgrade %s.%s to LONGTEXT: %v", c.Table, c.Column, err)
			}
		}
	}

	// Backfill migration: populate user_story.sprint_id from Tasks that are already assigned to a Sprint and have a parent Story.
	// (After Story-level Sprint planning ships, existing Sprint plans become immediately visible in the Backlog.) Idempotent.
	if err := MigrateStorySprintID(gormDB); err != nil {
		log.Printf("warn: failed to migrate story sprint_id: %v", err)
	}

	// Backfill migration: derive initial three-state category for task_status (aligned with Jira status categories).
	// AutoMigrate fills new category column with default 'todo'; this corrects rows to done/in_progress based on IsClosed and name keywords.
	// Idempotent: only updates rows still marked 'todo', never overwrites user-configured values.
	if err := MigrateStatusCategory(gormDB); err != nil {
		log.Printf("warn: failed to migrate task_status category: %v", err)
	}

	log.Println("Database migrations completed successfully")
	return nil
}

// MigrateStorySprintID backfills user_story.sprint_id from Tasks that are assigned to a Sprint and have a parent Story.
// Makes existing Sprint plans immediately visible in the Backlog.
// Idempotent: only updates Stories where sprint_id IS NULL, safe to run repeatedly.
// Edge case (a Story's Tasks spread across multiple Sprints): uses MIN(sprint_id) (the Sprint with the smallest ID).
func MigrateStorySprintID(db *gorm.DB) error {
	type storySprint struct {
		UserStoryID int `gorm:"column:user_story_id"`
		SprintID    int `gorm:"column:sprint_id"`
	}
	var rows []storySprint
	// For each task with user_story_id and non-null sprint_id, group by story and take MIN(sprint_id)
	if err := db.Raw(`SELECT user_story_id, MIN(sprint_id) AS sprint_id FROM task WHERE user_story_id IS NOT NULL AND sprint_id IS NOT NULL GROUP BY user_story_id`).Scan(&rows).Error; err != nil {
		return err
	}
	for _, r := range rows {
		// Only update Stories where sprint_id IS NULL (idempotent)
		if err := db.Model(&model.UserStory{}).Where("id = ? AND sprint_id IS NULL", r.UserStoryID).Update("sprint_id", r.SprintID).Error; err != nil {
			log.Printf("warn: failed to migrate story %d sprint_id: %v", r.UserStoryID, err)
		}
	}
	if len(rows) > 0 {
		log.Printf("Migrated %d user stories' sprint_id from existing task assignments", len(rows))
	}
	return nil
}

// MigrateStatusCategory backfills initial three-state category for task_status
// (aligned with Jira status categories: todo / in_progress / done).
//
// Background: When AutoMigrate adds the category column, it fills existing rows with default 'todo',
// causing statuses that should be done/in_progress (e.g., "completed", "in progress") to be incorrectly marked as todo,
// affecting burndown/velocity charts.
//
// Derivation rules (only affects rows where category='todo', idempotent):
//   - IsClosed=true -> done
//   - name matches in_progress keywords (and not closed) -> in_progress
//   - others remain todo
//
// Idempotency: After migration corrects mislabeled todo rows, they no longer satisfy category='todo',
// so subsequent restarts won't modify them again; user-configured values from the settings page are never overwritten.
func MigrateStatusCategory(db *gorm.DB) error {
	// todo with IsClosed=true -> done
	res := db.Model(&model.TaskStatus{}).Where("category = ? AND is_closed = ?", "todo", true).Update("category", "done")
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected > 0 {
		log.Printf("Migrated %d task_status rows to category=done (is_closed=true)", res.RowsAffected)
	}

	// todo with name matching in_progress keywords (and not closed) -> in_progress
	// English keywords use LOWER() for case-insensitivity; Chinese keywords use byte-level LIKE matching (LOWER has no effect on Chinese).
	keywords := []string{
		"%progress%", "%doing%", "%active%", "%review%", "%test%", "%develop%",
		"%进行中%", "%评审%", "%测试%", "%开发%",
	}
	var totalAffected int64
	for _, kw := range keywords {
		res := db.Model(&model.TaskStatus{}).
			Where("category = ? AND is_closed = ? AND LOWER(name) LIKE ?", "todo", false, kw).
			Update("category", "in_progress")
		if res.Error != nil {
			return res.Error
		}
		totalAffected += res.RowsAffected
	}
	if totalAffected > 0 {
		log.Printf("Migrated %d task_status rows to category=in_progress (name keyword match)", totalAffected)
	}
	return nil
}

// EnsureInitialized checks if the system has been initialized.
// If existing admin accounts exist (backward compatibility), marks as initialized.
// Otherwise, the system waits for the setup wizard.
func EnsureInitialized() error {
	var setting model.SystemSetting
	err := gormDB.Where("`key` = ?", "initialized").First(&setting).Error
	if err == nil && setting.Value == "true" {
		log.Println("System is already initialized")
		return nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to check initialization status: %w", err)
	}

	// Backward compatibility: if admin accounts already exist, auto-mark as initialized
	var adminCount int64
	gormDB.Model(&model.Account{}).Where("administrator = ? AND removed = ?", true, false).Count(&adminCount)
	if adminCount > 0 {
		log.Println("Existing admin accounts found, marking system as initialized")
		setting := &model.SystemSetting{Key: "initialized", Value: "true"}
		if err := gormDB.Create(setting).Error; err != nil {
			return fmt.Errorf("failed to mark system as initialized: %w", err)
		}
		return nil
	}

	log.Println("System is not initialized. Setup wizard required at /setup")
	return nil
}

// LoadOrGenerateSessionSecret loads the session secret from the database,
// or generates a new 32-byte random secret if none exists.
func LoadOrGenerateSessionSecret() (string, error) {
	var setting model.SystemSetting
	err := gormDB.Where("`key` = ?", "session_secret").First(&setting).Error
	if err == nil && setting.Value != "" {
		return setting.Value, nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return "", err
	}

	// Generate 32-byte random secret (64 hex chars)
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	secret := hex.EncodeToString(bytes)

	setting = model.SystemSetting{
		Key:   "session_secret",
		Value: secret,
	}
	if err := gormDB.Create(&setting).Error; err != nil {
		return "", err
	}

	return secret, nil
}
