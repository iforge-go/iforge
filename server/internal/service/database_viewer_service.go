package service

import (
	"fmt"
	"regexp"
	"strings"

	"gorm.io/gorm"
)

// DatabaseViewerService handles database viewing operations
type DatabaseViewerService struct {
	db *gorm.DB
}

// NewDatabaseViewerService creates a new DatabaseViewerService
func NewDatabaseViewerService(db *gorm.DB) *DatabaseViewerService {
	return &DatabaseViewerService{
		db: db,
	}
}

// TableInfo represents information about a database table
type TableInfo struct {
	Name       string `json:"name"`
	RowCount   int    `json:"rowCount"`
	SizeBytes  int64  `json:"sizeBytes"`
	Engine     string `json:"engine"`
	CreateTime string `json:"createTime"`
}

// ColumnInfo represents information about a table column
type ColumnInfo struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Nullable     bool   `json:"nullable"`
	Key          string `json:"key"`
	DefaultValue string `json:"defaultValue"`
	Extra        string `json:"extra"`
}

// QueryResult represents the result of a SQL query
type QueryResult struct {
	Columns []string        `json:"columns"`
	Rows    [][]interface{} `json:"rows"`
	Count   int             `json:"count"`
}

// ListTables lists all tables in the database
func (s *DatabaseViewerService) ListTables() ([]TableInfo, error) {
	var tableNames []string

	switch s.db.Dialector.Name() {
	case "sqlite":
		if err := s.db.Raw(`
			SELECT name
			FROM sqlite_master
			WHERE type='table' AND name NOT LIKE 'sqlite_%'
			ORDER BY name
		`).Scan(&tableNames).Error; err != nil {
			return nil, fmt.Errorf("failed to list tables: %w", err)
		}
	case "mysql":
		// MySQL: query information_schema.tables, limited to current database, excluding system views
		if err := s.db.Raw(`
			SELECT table_name
			FROM information_schema.tables
			WHERE table_schema = DATABASE() AND table_type = 'BASE TABLE'
			ORDER BY table_name
		`).Scan(&tableNames).Error; err != nil {
			return nil, fmt.Errorf("failed to list tables: %w", err)
		}
	case "postgres":
		// PostgreSQL: query all base tables under public schema
		if err := s.db.Raw(`
			SELECT table_name
			FROM information_schema.tables
			WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
			ORDER BY table_name
		`).Scan(&tableNames).Error; err != nil {
			return nil, fmt.Errorf("failed to list tables: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", s.db.Dialector.Name())
	}

	var tables []TableInfo
	for _, tableName := range tableNames {
		table := TableInfo{Name: tableName}

		// Get row count (tableName is safe as it comes from ListTables system query)
		var count int64
		if err := s.db.Table(tableName).Count(&count).Error; err != nil {
			table.RowCount = 0
		} else {
			table.RowCount = int(count)
		}

		tables = append(tables, table)
	}

	return tables, nil
}

// GetTableSchema gets the schema of a table
func (s *DatabaseViewerService) GetTableSchema(tableName string) ([]ColumnInfo, error) {
	if !isValidTableName(tableName) {
		return nil, fmt.Errorf("invalid table name")
	}

	var columns []ColumnInfo

	switch s.db.Dialector.Name() {
	case "sqlite":
		// SQLite: PRAGMA table_info returns (cid, name, type, notnull, dflt_value, pk)
		type pragmaResult struct {
			CID       int
			Name      string
			Type      string
			NotNull   int
			DfltValue *string
			PK        int
		}
		var results []pragmaResult
		if err := s.db.Raw(fmt.Sprintf("PRAGMA table_info(%s)", tableName)).Scan(&results).Error; err != nil {
			return nil, fmt.Errorf("failed to get table schema: %w", err)
		}
		for _, r := range results {
			col := ColumnInfo{
				Name:     r.Name,
				Type:     r.Type,
				Nullable: r.NotNull == 0,
				Key:      fmt.Sprintf("%d", r.PK),
			}
			if r.DfltValue != nil {
				col.DefaultValue = *r.DfltValue
			}
			columns = append(columns, col)
		}
	case "mysql":
		// MySQL: SHOW COLUMNS FROM returns (Field, Type, Null, Key, Default, Extra)
		type mysqlColumnResult struct {
			Field   string
			Type    string
			Null    string
			Key     string
			Default *string
			Extra   string
		}
		var results []mysqlColumnResult
		if err := s.db.Raw(fmt.Sprintf("SHOW COLUMNS FROM %s", tableName)).Scan(&results).Error; err != nil {
			return nil, fmt.Errorf("failed to get table schema: %w", err)
		}
		for _, r := range results {
			col := ColumnInfo{
				Name:     r.Field,
				Type:     r.Type,
				Nullable: r.Null == "YES",
				Key:      r.Key,
				Extra:    r.Extra,
			}
			if r.Default != nil {
				col.DefaultValue = *r.Default
			}
			columns = append(columns, col)
		}
	case "postgres":
		// PostgreSQL: query information_schema.columns
		type pgColumnResult struct {
			ColumnName    string
			DataType      string
			IsNullable    string
			ColumnDefault *string
		}
		var results []pgColumnResult
		if err := s.db.Raw(`
			SELECT column_name, data_type, is_nullable, column_default
			FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = ?
			ORDER BY ordinal_position
		`, tableName).Scan(&results).Error; err != nil {
			return nil, fmt.Errorf("failed to get table schema: %w", err)
		}
		for _, r := range results {
			col := ColumnInfo{
				Name:     r.ColumnName,
				Type:     r.DataType,
				Nullable: r.IsNullable == "YES",
			}
			if r.ColumnDefault != nil {
				col.DefaultValue = *r.ColumnDefault
			}
			columns = append(columns, col)
		}
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", s.db.Dialector.Name())
	}

	return columns, nil
}

// QueryTable queries data from a table
func (s *DatabaseViewerService) QueryTable(tableName string, limit, offset int) (*QueryResult, error) {
	// Validate table name to prevent SQL injection
	if !isValidTableName(tableName) {
		return nil, fmt.Errorf("invalid table name")
	}

	// Set default values
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	// Query data: tableName validated by isValidTableName (alphanumeric + underscore only),
	// using Table() + Limit/Offset with GORM query builder is safer than Raw string concatenation
	rows, err := s.db.Table(tableName).Limit(limit).Offset(offset).Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to query table: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// Scan rows
	var resultRows [][]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert byte slices to strings for JSON serialization
		row := make([]interface{}, len(columns))
		for i, v := range values {
			switch val := v.(type) {
			case []byte:
				row[i] = string(val)
			default:
				row[i] = val
			}
		}

		resultRows = append(resultRows, row)
	}

	return &QueryResult{
		Columns: columns,
		Rows:    resultRows,
		Count:   len(resultRows),
	}, nil
}

// ExecuteQuery executes a custom SQL query (read-only)
func (s *DatabaseViewerService) ExecuteQuery(query string) (*QueryResult, error) {
	// Only allow SELECT statements for safety
	trimmedQuery := strings.TrimSpace(query)
	upperQuery := strings.ToUpper(trimmedQuery)

	if !strings.HasPrefix(upperQuery, "SELECT") {
		return nil, fmt.Errorf("only SELECT queries are allowed")
	}

	// Prevent dangerous operations using word boundaries
	dangerous := []string{"INSERT", "UPDATE", "DELETE", "DROP", "ALTER", "CREATE", "TRUNCATE", "EXEC", "EXECUTE"}
	for _, keyword := range dangerous {
		// Check for keyword with word boundaries to avoid false positives
		pattern := `\b` + keyword + `\b`
		if matched, _ := regexp.MatchString(pattern, upperQuery); matched {
			return nil, fmt.Errorf("query contains forbidden keyword: %s", keyword)
		}
	}

	// Prevent subqueries that could bypass restrictions
	if strings.Count(upperQuery, "SELECT") > 1 {
		return nil, fmt.Errorf("subqueries are not allowed")
	}

	rows, err := s.db.Raw(query).Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// Scan rows
	var resultRows [][]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert byte slices to strings for JSON serialization
		row := make([]interface{}, len(columns))
		for i, v := range values {
			switch val := v.(type) {
			case []byte:
				row[i] = string(val)
			default:
				row[i] = val
			}
		}

		resultRows = append(resultRows, row)
	}

	return &QueryResult{
		Columns: columns,
		Rows:    resultRows,
		Count:   len(resultRows),
	}, nil
}

// isValidTableName checks if a table name is valid
func isValidTableName(name string) bool {
	if len(name) == 0 || len(name) > 100 {
		return false
	}
	// Only allow alphanumeric characters and underscores
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}
