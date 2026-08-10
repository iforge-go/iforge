package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config application configuration
type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Database   DatabaseConfig   `yaml:"database"`
	Log        LogConfig        `yaml:"log"`
	Repository RepositoryConfig `yaml:"repository"`
}

// ServerConfig server configuration
type ServerConfig struct {
	HTTPPort    int    `yaml:"http_port"`
	SSHPort     int    `yaml:"ssh_port"`
	SSHEnabled  bool   `yaml:"ssh_enabled"`
	ExternalURL string `yaml:"external_url"`
}

// DatabaseConfig database configuration
type DatabaseConfig struct {
	Driver                 string         `yaml:"driver"`
	SQLite                 SQLiteConfig   `yaml:"sqlite"`
	MySQL                  MySQLConfig    `yaml:"mysql"`
	Postgres               PostgresConfig `yaml:"postgres"`
	ConnMaxLifetimeSeconds int            `yaml:"conn_max_lifetime_seconds"`
	ConnMaxIdleTimeSeconds int            `yaml:"conn_max_idle_time_seconds"`
}

// SQLiteConfig SQLite configuration
type SQLiteConfig struct {
	Path string `yaml:"path"`
}

// MySQLConfig MySQL configuration
type MySQLConfig struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	User         string `yaml:"user"`
	Password     string `yaml:"password"`
	Database     string `yaml:"database"`
	Charset      string `yaml:"charset"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
	MaxOpenConns int    `yaml:"max_open_conns"`
}

// PostgresConfig PostgreSQL configuration
type PostgresConfig struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	User         string `yaml:"user"`
	Password     string `yaml:"password"`
	Database     string `yaml:"database"`
	SSLMode      string `yaml:"sslmode"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
	MaxOpenConns int    `yaml:"max_open_conns"`
}

// LogConfig log configuration
type LogConfig struct {
	Level string `yaml:"level"`
	Path  string `yaml:"path"`
}

// RepositoryConfig repository configuration
type RepositoryConfig struct {
	Path string `yaml:"path"`
}

// Load loads a configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	cfg.SetDefaults()

	return &cfg, nil
}

// SetDefaults sets default configuration values
func (c *Config) SetDefaults() {
	// Server defaults
	if c.Server.HTTPPort == 0 {
		c.Server.HTTPPort = 8081
	}
	if c.Server.SSHPort == 0 {
		c.Server.SSHPort = 2022
	}
	if c.Server.SSHEnabled == false {
		c.Server.SSHEnabled = true
	}
	if c.Server.ExternalURL == "" {
		c.Server.ExternalURL = fmt.Sprintf("http://localhost:%d", c.Server.HTTPPort)
	}

	// Database defaults
	if c.Database.Driver == "" {
		c.Database.Driver = "mysql"
	}
	if c.Database.SQLite.Path == "" {
		c.Database.SQLite.Path = "./data/iforge.db"
	}
	if c.Database.MySQL.Host == "" {
		c.Database.MySQL.Host = "localhost"
	}
	if c.Database.MySQL.Port == 0 {
		c.Database.MySQL.Port = 3306
	}
	if c.Database.MySQL.Database == "" {
		c.Database.MySQL.Database = "iforge"
	}
	if c.Database.MySQL.Charset == "" {
		c.Database.MySQL.Charset = "utf8mb4"
	}
	if c.Database.MySQL.MaxIdleConns == 0 {
		c.Database.MySQL.MaxIdleConns = 10
	}
	if c.Database.MySQL.MaxOpenConns == 0 {
		c.Database.MySQL.MaxOpenConns = 30
	}
	if c.Database.Postgres.Host == "" {
		c.Database.Postgres.Host = "localhost"
	}
	if c.Database.Postgres.Port == 0 {
		c.Database.Postgres.Port = 5432
	}
	if c.Database.Postgres.Database == "" {
		c.Database.Postgres.Database = "iforge"
	}
	if c.Database.Postgres.SSLMode == "" {
		c.Database.Postgres.SSLMode = "disable"
	}
	if c.Database.Postgres.MaxIdleConns == 0 {
		c.Database.Postgres.MaxIdleConns = 10
	}
	if c.Database.Postgres.MaxOpenConns == 0 {
		c.Database.Postgres.MaxOpenConns = 30
	}
	if c.Database.ConnMaxLifetimeSeconds == 0 {
		c.Database.ConnMaxLifetimeSeconds = 1800
	}
	if c.Database.ConnMaxIdleTimeSeconds == 0 {
		c.Database.ConnMaxIdleTimeSeconds = 300
	}

	if c.Log.Level == "" {
		c.Log.Level = "info"
	}
	if c.Log.Path == "" {
		c.Log.Path = "./logs"
	}

	// Repository defaults
	if c.Repository.Path == "" {
		c.Repository.Path = "./repositories"
	}
}

func (c *Config) LoadFromEnv() {
	if port := os.Getenv("IFORGE_HTTP_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			c.Server.HTTPPort = p
		}
	}
	if port := os.Getenv("IFORGE_SSH_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			c.Server.SSHPort = p
		}
	}
	if url := os.Getenv("IFORGE_EXTERNAL_URL"); url != "" {
		c.Server.ExternalURL = url
	}

	if driver := os.Getenv("IFORGE_DB_DRIVER"); driver != "" {
		c.Database.Driver = driver
	}

	if path := os.Getenv("IFORGE_SQLITE_PATH"); path != "" {
		c.Database.SQLite.Path = path
	}

	if host := os.Getenv("IFORGE_MYSQL_HOST"); host != "" {
		c.Database.MySQL.Host = host
	}
	if port := os.Getenv("IFORGE_MYSQL_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			c.Database.MySQL.Port = p
		}
	}
	if user := os.Getenv("IFORGE_MYSQL_USER"); user != "" {
		c.Database.MySQL.User = user
	}
	if password := os.Getenv("IFORGE_MYSQL_PASSWORD"); password != "" {
		c.Database.MySQL.Password = password
	}
	if database := os.Getenv("IFORGE_MYSQL_DATABASE"); database != "" {
		c.Database.MySQL.Database = database
	}

	// PostgreSQL configuration
	if host := os.Getenv("IFORGE_POSTGRES_HOST"); host != "" {
		c.Database.Postgres.Host = host
	}
	if port := os.Getenv("IFORGE_POSTGRES_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			c.Database.Postgres.Port = p
		}
	}
	if user := os.Getenv("IFORGE_POSTGRES_USER"); user != "" {
		c.Database.Postgres.User = user
	}
	if password := os.Getenv("IFORGE_POSTGRES_PASSWORD"); password != "" {
		c.Database.Postgres.Password = password
	}
	if database := os.Getenv("IFORGE_POSTGRES_DATABASE"); database != "" {
		c.Database.Postgres.Database = database
	}
	if value := os.Getenv("IFORGE_DB_MAX_IDLE_CONNS"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			switch c.Database.Driver {
			case "postgres":
				c.Database.Postgres.MaxIdleConns = parsed
			default:
				c.Database.MySQL.MaxIdleConns = parsed
			}
		}
	}
	if value := os.Getenv("IFORGE_DB_MAX_OPEN_CONNS"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			switch c.Database.Driver {
			case "postgres":
				c.Database.Postgres.MaxOpenConns = parsed
			default:
				c.Database.MySQL.MaxOpenConns = parsed
			}
		}
	}
	if value := os.Getenv("IFORGE_DB_CONN_MAX_LIFETIME_SECONDS"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			c.Database.ConnMaxLifetimeSeconds = parsed
		}
	}
	if value := os.Getenv("IFORGE_DB_CONN_MAX_IDLE_TIME_SECONDS"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			c.Database.ConnMaxIdleTimeSeconds = parsed
		}
	}
}

// MaxIdleConns returns the configured pool idle limit for the active driver.
func (c DatabaseConfig) MaxIdleConns() int {
	if c.Driver == "postgres" {
		return c.Postgres.MaxIdleConns
	}
	return c.MySQL.MaxIdleConns
}

// MaxOpenConns returns the configured pool connection limit for the active driver.
func (c DatabaseConfig) MaxOpenConns() int {
	if c.Driver == "postgres" {
		return c.Postgres.MaxOpenConns
	}
	return c.MySQL.MaxOpenConns
}

func (c *DatabaseConfig) GetDSN() (string, error) {
	switch c.Driver {
	case "sqlite":
		return c.SQLite.Path, nil
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
			c.MySQL.User, c.MySQL.Password, c.MySQL.Host, c.MySQL.Port,
			c.MySQL.Database, c.MySQL.Charset), nil
	case "postgres":
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			c.Postgres.Host, c.Postgres.Port, c.Postgres.User, c.Postgres.Password,
			c.Postgres.Database, c.Postgres.SSLMode), nil
	default:
		return "", fmt.Errorf("unsupported database driver: %s", c.Driver)
	}
}

func FindConfigFile() string {
	if path := os.Getenv("IFORGE_CONFIG"); path != "" {
		return path
	}

	candidates := []string{
		"config.yaml",
		"config.yml",
	}

	// Current directory
	for _, name := range candidates {
		if _, err := os.Stat(name); err == nil {
			absPath, _ := filepath.Abs(name)
			return absPath
		}
	}

	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		for _, name := range candidates {
			path := filepath.Join(execDir, name)
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}

	return ""
}
