package service

import (
	"testing"

	"iforge/iforge/internal/model"
)

func TestDatabaseViewerService_ListTables(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewDatabaseViewerService(db)

	// List all tables
	tables, err := service.ListTables()
	if err != nil {
		t.Fatalf("ListTables failed: %v", err)
	}

	// Should have at least the tables we migrated
	if len(tables) == 0 {
		t.Error("Expected at least one table, got 0")
	}

	// Check if account table exists
	found := false
	for _, table := range tables {
		if table.Name == "account" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find 'account' table in list")
	}
}

func TestDatabaseViewerService_GetTableSchema(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewDatabaseViewerService(db)

	// Get schema for account table
	columns, err := service.GetTableSchema("account")
	if err != nil {
		t.Fatalf("GetTableSchema failed: %v", err)
	}

	if len(columns) == 0 {
		t.Error("Expected at least one column, got 0")
	}

	// Check if user_name column exists
	found := false
	for _, col := range columns {
		if col.Name == "user_name" {
			found = true
			if col.Type == "" {
				t.Error("Expected column type to be set")
			}
			break
		}
	}
	if !found {
		t.Error("Expected to find 'user_name' column in schema")
	}
}

func TestDatabaseViewerService_GetTableSchema_InvalidTable(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewDatabaseViewerService(db)

	// Try to get schema for invalid table name
	_, err := service.GetTableSchema("invalid-table-name")
	if err == nil {
		t.Error("Expected error for invalid table name, got nil")
	}
}

func TestDatabaseViewerService_GetTableSchema_NonExistent(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewDatabaseViewerService(db)

	// SQLite PRAGMA table_info returns empty result for non-existent tables (no error)
	columns, err := service.GetTableSchema("non_existent_table")
	if err != nil {
		t.Fatalf("GetTableSchema failed: %v", err)
	}
	// Should return empty columns for non-existent table
	if len(columns) != 0 {
		t.Errorf("Expected 0 columns for non-existent table, got %d", len(columns))
	}
}

func TestDatabaseViewerService_QueryTable(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewDatabaseViewerService(db)

	// Create some test data
	account := &model.Account{
		UserName:       "testuser",
		FullName:       "Test User",
		MailAddress:    "test@example.com",
		Password:       "hashed_password",
		IsAdmin:        false,
		IsOrganization: false,
		IsRemoved:      false,
	}
	db.Create(account)

	// Query account table
	result, err := service.QueryTable("account", 10, 0)
	if err != nil {
		t.Fatalf("QueryTable failed: %v", err)
	}

	if result.Count == 0 {
		t.Error("Expected at least one row, got 0")
	}

	if len(result.Columns) == 0 {
		t.Error("Expected at least one column, got 0")
	}

	// Check if user_name column is in results
	foundCol := false
	for _, col := range result.Columns {
		if col == "user_name" {
			foundCol = true
			break
		}
	}
	if !foundCol {
		t.Error("Expected to find 'user_name' column in query results")
	}
}

func TestDatabaseViewerService_QueryTable_InvalidTable(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewDatabaseViewerService(db)

	// Try to query invalid table name
	_, err := service.QueryTable("invalid-table-name", 10, 0)
	if err == nil {
		t.Error("Expected error for invalid table name, got nil")
	}
}

func TestDatabaseViewerService_QueryTable_DefaultLimit(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewDatabaseViewerService(db)

	// Query with limit=0 (should use default)
	result, err := service.QueryTable("account", 0, 0)
	if err != nil {
		t.Fatalf("QueryTable failed: %v", err)
	}

	// Should not error, just use default limit
	if result == nil {
		t.Error("Expected result to be non-nil")
	}
}

func TestDatabaseViewerService_QueryTable_NegativeOffset(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewDatabaseViewerService(db)

	// Query with negative offset (should use 0)
	result, err := service.QueryTable("account", 10, -5)
	if err != nil {
		t.Fatalf("QueryTable failed: %v", err)
	}

	// Should not error, just use offset=0
	if result == nil {
		t.Error("Expected result to be non-nil")
	}
}

func TestDatabaseViewerService_ExecuteQuery(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewDatabaseViewerService(db)

	// Create test data
	account := &model.Account{
		UserName:       "queryuser",
		FullName:       "Query User",
		MailAddress:    "query@example.com",
		Password:       "hashed_password",
		IsAdmin:        false,
		IsOrganization: false,
		IsRemoved:      false,
	}
	db.Create(account)

	// Execute SELECT query
	result, err := service.ExecuteQuery("SELECT user_name, full_name FROM account WHERE user_name = 'queryuser'")
	if err != nil {
		t.Fatalf("ExecuteQuery failed: %v", err)
	}

	if result.Count == 0 {
		t.Error("Expected at least one row, got 0")
	}

	if len(result.Columns) != 2 {
		t.Errorf("Expected 2 columns, got %d", len(result.Columns))
	}
}

func TestDatabaseViewerService_ExecuteQuery_ForbiddenKeywords(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewDatabaseViewerService(db)

	// Try forbidden keywords
	forbiddenQueries := []string{
		"INSERT INTO account VALUES ('test')",
		"UPDATE account SET user_name = 'test'",
		"DELETE FROM account",
		"DROP TABLE account",
		"ALTER TABLE account ADD COLUMN test",
		"CREATE TABLE test (id INT)",
		"TRUNCATE TABLE account",
	}

	for _, query := range forbiddenQueries {
		_, err := service.ExecuteQuery(query)
		if err == nil {
			t.Errorf("Expected error for forbidden query '%s', got nil", query)
		}
	}
}

func TestDatabaseViewerService_ExecuteQuery_NonSelect(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewDatabaseViewerService(db)

	// Try non-SELECT query
	_, err := service.ExecuteQuery("SHOW TABLES")
	if err == nil {
		t.Error("Expected error for non-SELECT query, got nil")
	}
}

func TestDatabaseViewerService_ExecuteQuery_Subquery(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewDatabaseViewerService(db)

	// Try subquery (should be forbidden)
	_, err := service.ExecuteQuery("SELECT * FROM account WHERE user_name IN (SELECT user_name FROM account)")
	if err == nil {
		t.Error("Expected error for subquery, got nil")
	}
}

func TestIsValidTableName(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"account", true},
		{"user_profile", true},
		{"table123", true},
		{"TABLE_NAME", true},
		{"", false},
		{"invalid-table", false},
		{"invalid.table", false},
		{"invalid table", false},
		{"invalid;table", false},
		{"invalid'table", false},
		{"invalid\"table", false},
		{"very_long_table_name_that_exceeds_one_hundred_characters_limit_and_should_be_rejected_by_the_validation_function_because_it_is_too_long", false},
	}

	for _, tt := range tests {
		result := isValidTableName(tt.name)
		if result != tt.expected {
			t.Errorf("isValidTableName(%q) = %v, expected %v", tt.name, result, tt.expected)
		}
	}
}

func TestDatabaseViewerService_TableRowCount(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewDatabaseViewerService(db)

	// Create multiple accounts
	for i := 0; i < 5; i++ {
		account := &model.Account{
			UserName:       "user" + string(rune('0'+i)),
			FullName:       "User " + string(rune('0'+i)),
			MailAddress:    "user" + string(rune('0'+i)) + "@example.com",
			Password:       "hashed_password",
			IsAdmin:        false,
			IsOrganization: false,
			IsRemoved:      false,
		}
		db.Create(account)
	}

	// List tables and check row count
	tables, err := service.ListTables()
	if err != nil {
		t.Fatalf("ListTables failed: %v", err)
	}

	// Find account table
	for _, table := range tables {
		if table.Name == "account" {
			if table.RowCount < 5 {
				t.Errorf("Expected at least 5 rows in account table, got %d", table.RowCount)
			}
			break
		}
	}
}
