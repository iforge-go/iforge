package database

import (
	"embed"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// schemaMigration tracks which migration versions have been applied.
// Stored in the `schema_migrations` table.
type schemaMigration struct {
	Version   string    `gorm:"primaryKey"`
	AppliedAt time.Time `gorm:"autoCreateTime"`
}

func (schemaMigration) TableName() string { return "schema_migrations" }

// RunMigrations applies all pending .up.sql migrations in lexical order.
// Idempotent: migrations already recorded in schema_migrations are skipped.
//
// Must be called AFTER InitGORMDB and AFTER AutoMigrate: data-backfill
// migrations (e.g. 0002) reference tables that AutoMigrate creates. On a
// fresh database, running RunMigrations first would fail with "no such table".
// The schema_migrations table itself is created here via a localized AutoMigrate
// call so the runner is self-contained.
//
// Coexists with GORM AutoMigrate: AutoMigrate handles additive model changes
// (new tables/columns), .up.sql handles destructive/transformative changes
// (column drops, type changes, complex indexes, data backfills).
func RunMigrations(db *gorm.DB) error {
	if err := db.AutoMigrate(&schemaMigration{}); err != nil {
		return fmt.Errorf("failed to ensure schema_migrations table: %w", err)
	}

	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("failed to read embedded migrations: %w", err)
	}

	var upFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			upFiles = append(upFiles, e.Name())
		}
	}
	sort.Strings(upFiles)

	if len(upFiles) == 0 {
		return nil
	}

	for _, name := range upFiles {
		version := strings.TrimSuffix(name, ".up.sql")

		var count int64
		if err := db.Model(&schemaMigration{}).Where("version = ?", version).Count(&count).Error; err != nil {
			return fmt.Errorf("failed to check migration %s: %w", name, err)
		}
		if count > 0 {
			continue
		}

		content, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", name, err)
		}

		// Split SQL by semicolon and execute statement by statement, compatible with MySQL's default lack of multi-statement support.
		// Skip comment lines and empty lines, only execute valid UPDATE/ALTER/CREATE statements.
		stmts := splitSQLStatements(string(content))
		for _, stmt := range stmts {
			if err := db.Exec(stmt).Error; err != nil {
				return fmt.Errorf("failed to apply migration %s: %w", name, err)
			}
		}

		if err := db.Create(&schemaMigration{Version: version}).Error; err != nil {
			return fmt.Errorf("failed to record migration %s: %w", name, err)
		}
		log.Printf("Applied migration: %s", version)
	}

	return nil
}

// splitSQLStatements splits SQL file content into individual statements by semicolon.
// Skips comment lines (starting with --) and empty lines, returning only valid SQL statements.
func splitSQLStatements(content string) []string {
	var stmts []string
	var current strings.Builder

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip empty lines and comment lines
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		current.WriteString(line)
		current.WriteString(" ")

		// If line ends with semicolon, the statement is complete
		if strings.HasSuffix(trimmed, ";") {
			stmt := strings.TrimSpace(current.String())
			if stmt != "" {
				stmts = append(stmts, stmt)
			}
			current.Reset()
		}
	}

	// Handle any remaining statement without a trailing semicolon
	if remaining := strings.TrimSpace(current.String()); remaining != "" {
		stmts = append(stmts, remaining)
	}

	return stmts
}
