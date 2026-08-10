package service

import (
	"testing"
	"time"
)

func TestAuditService_Log(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAuditService(db)

	// Log an audit entry
	service.Log("testuser", "127.0.0.1", "create", "repository", "1", map[string]interface{}{"name": "test-repo"}, true)

	// Verify the log was created
	logs, total, err := service.ListAuditLogs(AuditQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListAuditLogs failed: %v", err)
	}
	if total != 1 {
		t.Errorf("Expected 1 log, got %d", total)
	}
	if len(logs) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(logs))
	}

	log := logs[0]
	if log.ActorUserName != "testuser" {
		t.Errorf("Expected ActorUserName 'testuser', got '%s'", log.ActorUserName)
	}
	if log.ActorIP != "127.0.0.1" {
		t.Errorf("Expected ActorIP '127.0.0.1', got '%s'", log.ActorIP)
	}
	if log.Action != "create" {
		t.Errorf("Expected Action 'create', got '%s'", log.Action)
	}
	if log.ResourceType != "repository" {
		t.Errorf("Expected ResourceType 'repository', got '%s'", log.ResourceType)
	}
	if log.ResourceID != "1" {
		t.Errorf("Expected ResourceID '1', got '%s'", log.ResourceID)
	}
	if !log.Success {
		t.Error("Expected Success true, got false")
	}
}

func TestAuditService_Log_NilDetail(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAuditService(db)

	// Log with nil detail
	service.Log("testuser", "127.0.0.1", "delete", "repository", "2", nil, true)

	// Verify the log was created
	logs, _, err := service.ListAuditLogs(AuditQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListAuditLogs failed: %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(logs))
	}
	if logs[0].Detail != "" {
		t.Errorf("Expected empty Detail, got '%s'", logs[0].Detail)
	}
}

func TestAuditService_ListAuditLogs_FilterByActor(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAuditService(db)

	// Create logs from different actors
	service.Log("user1", "127.0.0.1", "create", "repository", "1", nil, true)
	service.Log("user2", "127.0.0.1", "create", "repository", "2", nil, true)
	service.Log("user1", "127.0.0.1", "delete", "repository", "3", nil, true)

	// Filter by actor
	logs, total, err := service.ListAuditLogs(AuditQuery{Actor: "user1", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListAuditLogs failed: %v", err)
	}
	if total != 2 {
		t.Errorf("Expected 2 logs for user1, got %d", total)
	}
	if len(logs) != 2 {
		t.Errorf("Expected 2 log entries, got %d", len(logs))
	}
}

func TestAuditService_ListAuditLogs_FilterByAction(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAuditService(db)

	// Create logs with different actions
	service.Log("testuser", "127.0.0.1", "create", "repository", "1", nil, true)
	service.Log("testuser", "127.0.0.1", "delete", "repository", "2", nil, true)
	service.Log("testuser", "127.0.0.1", "create", "repository", "3", nil, true)

	// Filter by action
	logs, total, err := service.ListAuditLogs(AuditQuery{Action: "create", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListAuditLogs failed: %v", err)
	}
	if total != 2 {
		t.Errorf("Expected 2 logs with action 'create', got %d", total)
	}
	if len(logs) != 2 {
		t.Errorf("Expected 2 log entries, got %d", len(logs))
	}
}

func TestAuditService_ListAuditLogs_FilterByResourceType(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAuditService(db)

	// Create logs with different resource types
	service.Log("testuser", "127.0.0.1", "create", "repository", "1", nil, true)
	service.Log("testuser", "127.0.0.1", "create", "user", "2", nil, true)
	service.Log("testuser", "127.0.0.1", "create", "repository", "3", nil, true)

	// Filter by resource type
	logs, total, err := service.ListAuditLogs(AuditQuery{ResourceType: "repository", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListAuditLogs failed: %v", err)
	}
	if total != 2 {
		t.Errorf("Expected 2 logs with resource type 'repository', got %d", total)
	}
	if len(logs) != 2 {
		t.Errorf("Expected 2 log entries, got %d", len(logs))
	}
}

func TestAuditService_ListAuditLogs_FilterByTime(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAuditService(db)

	// Create logs
	service.Log("testuser", "127.0.0.1", "create", "repository", "1", nil, true)
	time.Sleep(10 * time.Millisecond) // Ensure time difference
	middleTime := time.Now()
	time.Sleep(10 * time.Millisecond)
	service.Log("testuser", "127.0.0.1", "create", "repository", "2", nil, true)

	// Filter by start time
	logs, total, err := service.ListAuditLogs(AuditQuery{StartTime: &middleTime, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListAuditLogs failed: %v", err)
	}
	if total != 1 {
		t.Errorf("Expected 1 log after middle time, got %d", total)
	}
	if len(logs) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(logs))
	}
}

func TestAuditService_ListAuditLogs_Pagination(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAuditService(db)

	// Create 5 logs
	for i := 1; i <= 5; i++ {
		service.Log("testuser", "127.0.0.1", "create", "repository", string(rune('0'+i)), nil, true)
	}

	// Get page 1
	logs1, total, err := service.ListAuditLogs(AuditQuery{Page: 1, PageSize: 2})
	if err != nil {
		t.Fatalf("ListAuditLogs page 1 failed: %v", err)
	}
	if total != 5 {
		t.Errorf("Expected total 5, got %d", total)
	}
	if len(logs1) != 2 {
		t.Errorf("Expected 2 logs on page 1, got %d", len(logs1))
	}

	// Get page 2
	logs2, _, err := service.ListAuditLogs(AuditQuery{Page: 2, PageSize: 2})
	if err != nil {
		t.Fatalf("ListAuditLogs page 2 failed: %v", err)
	}
	if len(logs2) != 2 {
		t.Errorf("Expected 2 logs on page 2, got %d", len(logs2))
	}

	// Get page 3
	logs3, _, err := service.ListAuditLogs(AuditQuery{Page: 3, PageSize: 2})
	if err != nil {
		t.Fatalf("ListAuditLogs page 3 failed: %v", err)
	}
	if len(logs3) != 1 {
		t.Errorf("Expected 1 log on page 3, got %d", len(logs3))
	}
}

func TestAuditService_ListAuditLogs_DefaultPagination(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAuditService(db)

	// Create 60 logs
	for i := 1; i <= 60; i++ {
		service.Log("testuser", "127.0.0.1", "create", "repository", string(rune('0'+i%10)), nil, true)
	}

	// Get with invalid page/pageSize (should use defaults)
	logs, total, err := service.ListAuditLogs(AuditQuery{Page: 0, PageSize: 300})
	if err != nil {
		t.Fatalf("ListAuditLogs failed: %v", err)
	}
	if total != 60 {
		t.Errorf("Expected total 60, got %d", total)
	}
	// Default page size is 50
	if len(logs) != 50 {
		t.Errorf("Expected 50 logs with default page size, got %d", len(logs))
	}
}

func TestAuditService_ListAuditLogs_Ordering(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewAuditService(db)

	// Create logs with time gaps
	service.Log("testuser", "127.0.0.1", "create", "repository", "1", nil, true)
	time.Sleep(5 * time.Millisecond)
	service.Log("testuser", "127.0.0.1", "create", "repository", "2", nil, true)
	time.Sleep(5 * time.Millisecond)
	service.Log("testuser", "127.0.0.1", "create", "repository", "3", nil, true)

	// Get logs (should be newest first)
	logs, _, err := service.ListAuditLogs(AuditQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListAuditLogs failed: %v", err)
	}
	if len(logs) != 3 {
		t.Errorf("Expected 3 logs, got %d", len(logs))
	}

	// Verify ordering (newest first)
	if logs[0].ResourceID != "3" {
		t.Errorf("Expected first log ResourceID '3', got '%s'", logs[0].ResourceID)
	}
	if logs[2].ResourceID != "1" {
		t.Errorf("Expected last log ResourceID '1', got '%s'", logs[2].ResourceID)
	}
}
