package config

import "testing"

func TestDatabasePoolDefaults(t *testing.T) {
	cfg := &Config{}
	cfg.SetDefaults()

	if got := cfg.Database.MaxOpenConns(); got != 30 {
		t.Errorf("MySQL MaxOpenConns() = %d, want 30", got)
	}
	if got := cfg.Database.MaxIdleConns(); got != 10 {
		t.Errorf("MySQL MaxIdleConns() = %d, want 10", got)
	}
	if cfg.Database.ConnMaxLifetimeSeconds != 1800 {
		t.Errorf("ConnMaxLifetimeSeconds = %d, want 1800", cfg.Database.ConnMaxLifetimeSeconds)
	}
	if cfg.Database.ConnMaxIdleTimeSeconds != 300 {
		t.Errorf("ConnMaxIdleTimeSeconds = %d, want 300", cfg.Database.ConnMaxIdleTimeSeconds)
	}
}

func TestDatabasePoolUsesActiveDriver(t *testing.T) {
	cfg := DatabaseConfig{
		Driver: "postgres",
		MySQL: MySQLConfig{
			MaxIdleConns: 1,
			MaxOpenConns: 2,
		},
		Postgres: PostgresConfig{
			MaxIdleConns: 3,
			MaxOpenConns: 4,
		},
	}

	if got := cfg.MaxIdleConns(); got != 3 {
		t.Errorf("MaxIdleConns() = %d, want 3", got)
	}
	if got := cfg.MaxOpenConns(); got != 4 {
		t.Errorf("MaxOpenConns() = %d, want 4", got)
	}
}
