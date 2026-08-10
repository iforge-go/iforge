package service

import (
	"testing"

	"iforge/iforge/internal/model"
)

func TestWebhookService_CreateWebhook(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewWebhookService(db)
	defer service.Shutdown(1000)

	// Create a repository first
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create a webhook
	token := "test-token"
	webhook, err := service.CreateWebhook(
		"testuser", "test-repo",
		"https://example.com/webhook",
		"application/json",
		&token,
		`["push","issues"]`,
		true,
		false,
	)
	if err != nil {
		t.Fatalf("CreateWebhook failed: %v", err)
	}

	if webhook.URL != "https://example.com/webhook" {
		t.Errorf("Expected URL 'https://example.com/webhook', got '%s'", webhook.URL)
	}
	if webhook.ContentType != "application/json" {
		t.Errorf("Expected ContentType 'application/json', got '%s'", webhook.ContentType)
	}
	if webhook.Token == nil || *webhook.Token != "test-token" {
		t.Errorf("Expected Token 'test-token', got %v", webhook.Token)
	}
	if webhook.Active != true {
		t.Errorf("Expected Active true, got %v", webhook.Active)
	}
	if webhook.Events != `["push","issues"]` {
		t.Errorf("Expected Events '[\"push\",\"issues\"]', got '%s'", webhook.Events)
	}
}

func TestWebhookService_GetWebhook(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewWebhookService(db)
	defer service.Shutdown(1000)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create a webhook
	token := "test-token"
	created, err := service.CreateWebhook(
		"testuser", "test-repo",
		"https://example.com/webhook",
		"application/json",
		&token,
		`["push"]`,
		true,
		false,
	)
	if err != nil {
		t.Fatalf("CreateWebhook failed: %v", err)
	}

	// Get the webhook
	webhook, err := service.GetWebhook(created.ID)
	if err != nil {
		t.Fatalf("GetWebhook failed: %v", err)
	}

	if webhook.ID != created.ID {
		t.Errorf("Expected ID %d, got %d", created.ID, webhook.ID)
	}
	if webhook.URL != "https://example.com/webhook" {
		t.Errorf("Expected URL 'https://example.com/webhook', got '%s'", webhook.URL)
	}
}

func TestWebhookService_GetWebhook_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewWebhookService(db)
	defer service.Shutdown(1000)

	// Try to get non-existent webhook
	_, err := service.GetWebhook(999)
	if err == nil {
		t.Fatal("Expected error for non-existent webhook, got nil")
	}
	if err != ErrWebhookNotFound {
		t.Errorf("Expected ErrWebhookNotFound, got %v", err)
	}
}

func TestWebhookService_ListWebhooks(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewWebhookService(db)
	defer service.Shutdown(1000)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create multiple webhooks
	token := "test-token"
	for i := 1; i <= 3; i++ {
		url := "https://example.com/webhook" + string(rune('0'+i))
		_, err := service.CreateWebhook(
			"testuser", "test-repo",
			url,
			"application/json",
			&token,
			`["push"]`,
			true,
			false,
		)
		if err != nil {
			t.Fatalf("CreateWebhook failed: %v", err)
		}
	}

	// List webhooks
	webhooks, err := service.ListWebhooks("testuser", "test-repo")
	if err != nil {
		t.Fatalf("ListWebhooks failed: %v", err)
	}

	if len(webhooks) != 3 {
		t.Errorf("Expected 3 webhooks, got %d", len(webhooks))
	}
}

func TestWebhookService_UpdateWebhook(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewWebhookService(db)
	defer service.Shutdown(1000)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create a webhook
	token := "test-token"
	created, err := service.CreateWebhook(
		"testuser", "test-repo",
		"https://example.com/webhook",
		"application/json",
		&token,
		`["push"]`,
		true,
		false,
	)
	if err != nil {
		t.Fatalf("CreateWebhook failed: %v", err)
	}

	// Update the webhook
	newToken := "new-token"
	updated, err := service.UpdateWebhook(
		created.ID,
		"https://example.com/new-webhook",
		"text/plain",
		&newToken,
		`["issues"]`,
		false,
		true,
	)
	if err != nil {
		t.Fatalf("UpdateWebhook failed: %v", err)
	}

	if updated.URL != "https://example.com/new-webhook" {
		t.Errorf("Expected URL 'https://example.com/new-webhook', got '%s'", updated.URL)
	}
	if updated.ContentType != "text/plain" {
		t.Errorf("Expected ContentType 'text/plain', got '%s'", updated.ContentType)
	}
	if updated.Token == nil || *updated.Token != "new-token" {
		t.Errorf("Expected Token 'new-token', got %v", updated.Token)
	}
	if updated.Active != false {
		t.Errorf("Expected Active false, got %v", updated.Active)
	}
	if updated.Events != `["issues"]` {
		t.Errorf("Expected Events '[\"issues\"]', got '%s'", updated.Events)
	}
}

func TestWebhookService_DeleteWebhook(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewWebhookService(db)
	defer service.Shutdown(1000)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create a webhook
	token := "test-token"
	created, err := service.CreateWebhook(
		"testuser", "test-repo",
		"https://example.com/webhook",
		"application/json",
		&token,
		`["push"]`,
		true,
		false,
	)
	if err != nil {
		t.Fatalf("CreateWebhook failed: %v", err)
	}

	// Delete the webhook
	err = service.DeleteWebhook(created.ID)
	if err != nil {
		t.Fatalf("DeleteWebhook failed: %v", err)
	}

	// Verify deletion
	_, err = service.GetWebhook(created.ID)
	if err == nil {
		t.Fatal("Expected error after deletion, got nil")
	}
	if err != ErrWebhookNotFound {
		t.Errorf("Expected ErrWebhookNotFound, got %v", err)
	}
}

func TestWebhookService_ListWebhookDeliveries(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewWebhookService(db)
	defer service.Shutdown(1000)

	// Create a repository
	repo := &model.Repository{
		UserName:       "testuser",
		RepositoryName: "test-repo",
		IsPrivate:      false,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create a webhook
	token := "test-token"
	webhook, err := service.CreateWebhook(
		"testuser", "test-repo",
		"https://example.com/webhook",
		"application/json",
		&token,
		`["push"]`,
		true,
		false,
	)
	if err != nil {
		t.Fatalf("CreateWebhook failed: %v", err)
	}

	// Manually create some delivery records
	for i := 1; i <= 3; i++ {
		delivery := &model.WebhookDelivery{
			WebhookID:    webhook.ID,
			Event:        "push",
			Action:       "created",
			RequestBody:  `{"test":"data"}`,
			ResponseBody: `{"status":"ok"}`,
			Duration:     100,
			Success:      true,
		}
		if err := db.Create(delivery).Error; err != nil {
			t.Fatalf("Failed to create delivery: %v", err)
		}
	}

	// List deliveries
	deliveries, err := service.ListWebhookDeliveries(webhook.ID)
	if err != nil {
		t.Fatalf("ListWebhookDeliveries failed: %v", err)
	}

	if len(deliveries) != 3 {
		t.Errorf("Expected 3 deliveries, got %d", len(deliveries))
	}
}
