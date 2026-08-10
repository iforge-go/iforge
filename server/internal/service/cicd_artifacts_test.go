package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"1 second", time.Second},
		{"5 seconds", 5 * time.Second},
		{"1 sec", time.Second},
		{"10 secs", 10 * time.Second},
		{"30 s", 30 * time.Second},
		{"1 minute", time.Minute},
		{"5 minutes", 5 * time.Minute},
		{"1 min", time.Minute},
		{"10 mins", 10 * time.Minute},
		{"1 hour", time.Hour},
		{"3 hours", 3 * time.Hour},
		{"2 h", 2 * time.Hour},
		{"1 day", 24 * time.Hour},
		{"7 days", 7 * 24 * time.Hour},
		{"1 d", 24 * time.Hour},
		{"1 week", 7 * 24 * time.Hour},
		{"2 weeks", 14 * 24 * time.Hour},
		{"1 w", 7 * 24 * time.Hour},
		{"1 month", 30 * 24 * time.Hour},
		{"3 months", 90 * 24 * time.Hour},
		// Invalid cases
		{"", 0},
		{"invalid", 0},
		{"1", 0},
		{"abc days", 0},
		{"1 year", 0},
		{"1 2 3", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseDuration(tt.input)
			if result != tt.expected {
				t.Errorf("parseDuration(%q) = %v; want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{100, "100 B"},
		{1023, "1023 B"},
		{1024, "1.0 KiB"},
		{1024 * 1024, "1.0 MiB"},
		{1024 * 1024 * 1024, "1.0 GiB"},
		{1536, "1.5 KiB"},
		{1048576, "1.0 MiB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatSize(tt.input)
			if result != tt.expected {
				t.Errorf("formatSize(%d) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create source file
	srcPath := filepath.Join(tmpDir, "src.txt")
	content := "hello world"
	if err := os.WriteFile(srcPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Copy file
	dstPath := filepath.Join(tmpDir, "dst.txt")
	size, err := copyFile(srcPath, dstPath)
	if err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	if size != int64(len(content)) {
		t.Errorf("Expected size %d, got %d", len(content), size)
	}

	// Verify destination content
	data, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != content {
		t.Errorf("Expected content %q, got %q", content, string(data))
	}
}

func TestCopyFile_CreatesSubdirs(t *testing.T) {
	tmpDir := t.TempDir()

	srcPath := filepath.Join(tmpDir, "src.txt")
	if err := os.WriteFile(srcPath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	dstPath := filepath.Join(tmpDir, "a", "b", "c", "dst.txt")
	_, err := copyFile(srcPath, dstPath)
	if err != nil {
		t.Fatalf("copyFile should create subdirectories: %v", err)
	}

	if _, err := os.Stat(dstPath); os.IsNotExist(err) {
		t.Error("Expected destination file to exist")
	}
}

func TestCopyDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source directory structure
	srcDir := filepath.Join(tmpDir, "src")
	os.MkdirAll(filepath.Join(srcDir, "sub"), 0755)
	os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("content1"), 0644)
	os.WriteFile(filepath.Join(srcDir, "sub", "file2.txt"), []byte("content2"), 0644)

	dstDir := filepath.Join(tmpDir, "dst")
	size, err := copyDir(srcDir, dstDir)
	if err != nil {
		t.Fatalf("copyDir failed: %v", err)
	}

	if size == 0 {
		t.Error("Expected non-zero size")
	}

	// Verify files exist
	if _, err := os.Stat(filepath.Join(dstDir, "file1.txt")); os.IsNotExist(err) {
		t.Error("Expected file1.txt to exist")
	}
	if _, err := os.Stat(filepath.Join(dstDir, "sub", "file2.txt")); os.IsNotExist(err) {
		t.Error("Expected sub/file2.txt to exist")
	}
}

func TestArtifactsConfig_Parse(t *testing.T) {
	jsonStr := `{"paths": ["dist/**", "build/**"], "expire_in": "1 week"}`
	var cfg ArtifactsConfig
	if err := json.Unmarshal([]byte(jsonStr), &cfg); err != nil {
		t.Fatalf("Failed to parse artifacts config: %v", err)
	}

	if len(cfg.Paths) != 2 {
		t.Errorf("Expected 2 paths, got %d", len(cfg.Paths))
	}
	if cfg.ExpireIn != "1 week" {
		t.Errorf("Expected expire_in '1 week', got %q", cfg.ExpireIn)
	}
}

func TestCacheConfig_Parse(t *testing.T) {
	jsonStr := `{"paths": ["node_modules", ".cache"], "key": "npm-cache"}`
	var cfg CacheConfig
	if err := json.Unmarshal([]byte(jsonStr), &cfg); err != nil {
		t.Fatalf("Failed to parse cache config: %v", err)
	}

	if len(cfg.Paths) != 2 {
		t.Errorf("Expected 2 paths, got %d", len(cfg.Paths))
	}
	if cfg.Key != "npm-cache" {
		t.Errorf("Expected key 'npm-cache', got %q", cfg.Key)
	}
}
