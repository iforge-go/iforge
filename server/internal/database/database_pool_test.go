package database

import (
	"path/filepath"
	"testing"
	"time"
)

func TestInitGORMDBAppliesConfiguredPoolToSQLite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "iforge.db")
	t.Cleanup(func() {
		if err := CloseGORMDB(); err != nil {
			t.Errorf("CloseGORMDB() error = %v", err)
		}
	})

	err := InitGORMDB("sqlite", path, PoolConfig{
		MaxIdleConns: 5,
		MaxOpenConns: 9,
		MaxLifetime:  time.Hour,
		MaxIdleTime:  time.Minute,
	})
	if err != nil {
		t.Fatalf("InitGORMDB() error = %v", err)
	}

	sqlDB, err := GetGORMDB().DB()
	if err != nil {
		t.Fatalf("GetGORMDB().DB() error = %v", err)
	}
	stats := sqlDB.Stats()
	if stats.MaxOpenConnections != 1 {
		t.Errorf("SQLite MaxOpenConnections = %d, want 1", stats.MaxOpenConnections)
	}
}

func TestConfigureConnectionPool(t *testing.T) {
	pool := &recordingPool{}
	config := PoolConfig{
		MaxIdleConns: 4,
		MaxOpenConns: 12,
		MaxLifetime:  30 * time.Minute,
		MaxIdleTime:  5 * time.Minute,
	}

	configureConnectionPool(pool, config)

	if pool.maxIdle != config.MaxIdleConns || pool.maxOpen != config.MaxOpenConns {
		t.Fatalf("pool limits = idle:%d open:%d, want idle:%d open:%d", pool.maxIdle, pool.maxOpen, config.MaxIdleConns, config.MaxOpenConns)
	}
	if pool.maxLifetime != config.MaxLifetime || pool.maxIdleTime != config.MaxIdleTime {
		t.Fatalf("pool durations = lifetime:%s idle:%s, want lifetime:%s idle:%s", pool.maxLifetime, pool.maxIdleTime, config.MaxLifetime, config.MaxIdleTime)
	}
}

type recordingPool struct {
	maxIdle     int
	maxOpen     int
	maxLifetime time.Duration
	maxIdleTime time.Duration
}

func (p *recordingPool) SetMaxIdleConns(value int)              { p.maxIdle = value }
func (p *recordingPool) SetMaxOpenConns(value int)              { p.maxOpen = value }
func (p *recordingPool) SetConnMaxLifetime(value time.Duration) { p.maxLifetime = value }
func (p *recordingPool) SetConnMaxIdleTime(value time.Duration) { p.maxIdleTime = value }
