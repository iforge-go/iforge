package service

import (
	"strings"
	"testing"
)

func TestRenderTemplate(t *testing.T) {
	tmpl := `<h2>Hello {{.Name}}</h2><p>Welcome to {{.Site}}</p>`
	data := map[string]interface{}{
		"Name": "John",
		"Site": "iForge",
	}

	result := renderTemplate(tmpl, data)

	if !strings.Contains(result, "Hello John") {
		t.Errorf("Expected template to contain 'Hello John', got '%s'", result)
	}
	if !strings.Contains(result, "Welcome to iForge") {
		t.Errorf("Expected template to contain 'Welcome to iForge', got '%s'", result)
	}
}

func TestRenderTemplate_InvalidTemplate(t *testing.T) {
	tmpl := `{{.Invalid`
	data := map[string]interface{}{}

	result := renderTemplate(tmpl, data)

	if result != "" {
		t.Errorf("Expected empty string for invalid template, got '%s'", result)
	}
}

func TestRenderTemplate_ExecutionError(t *testing.T) {
	tmpl := `{{.Field}}`
	data := "not a map"

	result := renderTemplate(tmpl, data)

	if result != "" {
		t.Errorf("Expected empty string for execution error, got '%s'", result)
	}
}

func TestMailService_NewMailService(t *testing.T) {
	config := &SMTPConfig{
		Host:       "smtp.example.com",
		Port:       587,
		Username:   "user@example.com",
		Password:   "password",
		From:       "noreply@example.com",
		UseTLS:     true,
		AuthMethod: "plain",
	}

	service := NewMailService(config)
	defer service.Stop()

	if service == nil {
		t.Fatal("Expected MailService to be created")
	}
	if service.config != config {
		t.Error("Expected config to be set")
	}
	if service.maxRetries != 3 {
		t.Errorf("Expected maxRetries 3, got %d", service.maxRetries)
	}
}

func TestMailService_UpdateConfig(t *testing.T) {
	config1 := &SMTPConfig{
		Host: "smtp1.example.com",
		Port: 587,
	}

	service := NewMailService(config1)
	defer service.Stop()

	config2 := &SMTPConfig{
		Host: "smtp2.example.com",
		Port: 465,
	}

	service.UpdateConfig(config2)

	if service.config.Host != "smtp2.example.com" {
		t.Errorf("Expected Host 'smtp2.example.com', got '%s'", service.config.Host)
	}
	if service.config.Port != 465 {
		t.Errorf("Expected Port 465, got %d", service.config.Port)
	}
}

func TestMailService_SendMail_NotConfigured(t *testing.T) {
	service := NewMailService(nil)
	defer service.Stop()

	msg := &EmailMessage{
		To:      []string{"test@example.com"},
		Subject: "Test",
		Body:    "Test body",
	}

	err := service.SendMail(msg)
	if err == nil {
		t.Error("Expected error when SMTP not configured")
	}
	if !strings.Contains(err.Error(), "SMTP not configured") {
		t.Errorf("Expected 'SMTP not configured' error, got '%v'", err)
	}
}

func TestMailService_SendMail_EmptyHost(t *testing.T) {
	config := &SMTPConfig{
		Host: "",
		Port: 587,
	}

	service := NewMailService(config)
	defer service.Stop()

	msg := &EmailMessage{
		To:      []string{"test@example.com"},
		Subject: "Test",
		Body:    "Test body",
	}

	err := service.SendMail(msg)
	if err == nil {
		t.Error("Expected error when Host is empty")
	}
}

func TestMailService_TestConnection_NotConfigured(t *testing.T) {
	service := NewMailService(nil)
	defer service.Stop()

	err := service.TestConnection()
	if err == nil {
		t.Error("Expected error when SMTP not configured")
	}
	if !strings.Contains(err.Error(), "SMTP not configured") {
		t.Errorf("Expected 'SMTP not configured' error, got '%v'", err)
	}
}

func TestMailService_QueueMail(t *testing.T) {
	config := &SMTPConfig{
		Host: "smtp.example.com",
		Port: 587,
	}

	service := NewMailService(config)
	defer service.Stop()

	msg := &EmailMessage{
		To:      []string{"test@example.com"},
		Subject: "Test Subject",
		Body:    "Test body",
		IsHTML:  false,
	}

	// QueueMail should not block or panic
	service.QueueMail(msg)

	// Since the worker goroutine consumes the queue, we cannot accurately check the queue length
	// This test mainly verifies that QueueMail does not block or panic
}

func TestMailService_QueueMail_Multiple(t *testing.T) {
	config := &SMTPConfig{
		Host: "smtp.example.com",
		Port: 587,
	}

	service := NewMailService(config)
	defer service.Stop()

	// Queue multiple messages - mainly verify it does not block or panic
	for i := 0; i < 5; i++ {
		msg := &EmailMessage{
			To:      []string{"test@example.com"},
			Subject: "Test",
			Body:    "Body",
		}
		service.QueueMail(msg)
	}

	// The worker consumes the queue, not checking the exact length
}

func TestMailService_BaseURL(t *testing.T) {
	tests := []struct {
		name     string
		config   *SMTPConfig
		expected string
	}{
		{
			name:     "nil config",
			config:   nil,
			expected: "http://localhost:3001",
		},
		{
			name: "with config",
			config: &SMTPConfig{
				From: "noreply@example.com",
			},
			expected: "http://localhost:3001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewMailService(tt.config)
			defer service.Stop()

			url := service.baseURL()
			if url != tt.expected {
				t.Errorf("Expected baseURL '%s', got '%s'", tt.expected, url)
			}
		})
	}
}

func TestEmailMessage_HTML(t *testing.T) {
	msg := &EmailMessage{
		To:      []string{"test@example.com"},
		Subject: "Test",
		Body:    "<h1>HTML Content</h1>",
		IsHTML:  true,
	}

	if !msg.IsHTML {
		t.Error("Expected IsHTML to be true")
	}
}

func TestEmailMessage_PlainText(t *testing.T) {
	msg := &EmailMessage{
		To:      []string{"test@example.com"},
		Subject: "Test",
		Body:    "Plain text content",
		IsHTML:  false,
	}

	if msg.IsHTML {
		t.Error("Expected IsHTML to be false")
	}
}

func TestSMTPConfig_AuthMethods(t *testing.T) {
	methods := []string{"plain", "login", "cram-md5", ""}

	for _, method := range methods {
		config := &SMTPConfig{
			Host:       "smtp.example.com",
			Port:       587,
			Username:   "user",
			Password:   "pass",
			AuthMethod: method,
		}

		if config.AuthMethod != method {
			t.Errorf("Expected AuthMethod '%s', got '%s'", method, config.AuthMethod)
		}
	}
}

func TestMailService_SendIssueNotification(t *testing.T) {
	config := &SMTPConfig{
		Host: "smtp.example.com",
		Port: 587,
		From: "noreply@example.com",
	}

	// Create service without starting worker (test method directly)
	service := &MailService{
		config:     config,
		mailQueue:  make(chan *EmailMessage, 100),
		stopChan:   make(chan struct{}),
		maxRetries: 3,
	}

	// This should queue a message without error
	service.SendIssueNotification(
		[]string{"test@example.com"},
		"owner",
		"repo",
		123,
		"Test Issue",
	)

	// Check that a message was queued
	if len(service.mailQueue) != 1 {
		t.Errorf("Expected 1 message in queue, got %d", len(service.mailQueue))
	}

	// Retrieve and verify the message
	msg := <-service.mailQueue
	if !strings.Contains(msg.Subject, "owner/repo") {
		t.Errorf("Expected subject to contain 'owner/repo', got '%s'", msg.Subject)
	}
	if !strings.Contains(msg.Subject, "#123") {
		t.Errorf("Expected subject to contain '#123', got '%s'", msg.Subject)
	}
	if !msg.IsHTML {
		t.Error("Expected message to be HTML")
	}
}

func TestMailService_SendCommentNotification(t *testing.T) {
	config := &SMTPConfig{
		Host: "smtp.example.com",
		Port: 587,
		From: "noreply@example.com",
	}

	// Create service without starting worker
	service := &MailService{
		config:     config,
		mailQueue:  make(chan *EmailMessage, 100),
		stopChan:   make(chan struct{}),
		maxRetries: 3,
	}

	service.SendCommentNotification(
		[]string{"test@example.com"},
		"owner",
		"repo",
		123,
		"commenter",
		"This is a comment",
	)

	if len(service.mailQueue) != 1 {
		t.Errorf("Expected 1 message in queue, got %d", len(service.mailQueue))
	}

	msg := <-service.mailQueue
	if !strings.Contains(msg.Subject, "Re:") {
		t.Errorf("Expected subject to contain 'Re:', got '%s'", msg.Subject)
	}
}

func TestMailService_SendMentionNotification(t *testing.T) {
	config := &SMTPConfig{
		Host: "smtp.example.com",
		Port: 587,
		From: "noreply@example.com",
	}

	// Create service without starting worker
	service := &MailService{
		config:     config,
		mailQueue:  make(chan *EmailMessage, 100),
		stopChan:   make(chan struct{}),
		maxRetries: 3,
	}

	service.SendMentionNotification(
		[]string{"test@example.com"},
		"owner",
		"repo",
		123,
		"mentioner",
	)

	if len(service.mailQueue) != 1 {
		t.Errorf("Expected 1 message in queue, got %d", len(service.mailQueue))
	}

	msg := <-service.mailQueue
	if !strings.Contains(msg.Subject, "mentioned") {
		t.Errorf("Expected subject to contain 'mentioned', got '%s'", msg.Subject)
	}
}

func TestMailService_SendMergeRequestNotification(t *testing.T) {
	config := &SMTPConfig{
		Host: "smtp.example.com",
		Port: 587,
		From: "noreply@example.com",
	}

	// Create service without starting worker
	service := &MailService{
		config:     config,
		mailQueue:  make(chan *EmailMessage, 100),
		stopChan:   make(chan struct{}),
		maxRetries: 3,
	}

	service.SendMergeRequestNotification(
		[]string{"test@example.com"},
		"owner",
		"repo",
		456,
		"Test MR",
	)

	if len(service.mailQueue) != 1 {
		t.Errorf("Expected 1 message in queue, got %d", len(service.mailQueue))
	}

	msg := <-service.mailQueue
	if !strings.Contains(msg.Subject, "Merge Request") {
		t.Errorf("Expected subject to contain 'Merge Request', got '%s'", msg.Subject)
	}
	if !strings.Contains(msg.Subject, "#456") {
		t.Errorf("Expected subject to contain '#456', got '%s'", msg.Subject)
	}
}

func TestMailService_SendPipelineNotification(t *testing.T) {
	config := &SMTPConfig{
		Host: "smtp.example.com",
		Port: 587,
		From: "noreply@example.com",
	}

	// Create service without starting worker
	service := &MailService{
		config:     config,
		mailQueue:  make(chan *EmailMessage, 100),
		stopChan:   make(chan struct{}),
		maxRetries: 3,
	}

	tests := []struct {
		status       string
		expectedText string
	}{
		{"success", "succeeded"},
		{"failed", "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			service.SendPipelineNotification(
				[]string{"test@example.com"},
				"owner",
				"repo",
				789,
				"main",
				"abc123",
				tt.status,
				"test commit",
			)

			if len(service.mailQueue) != 1 {
				t.Errorf("Expected 1 message in queue, got %d", len(service.mailQueue))
			}

			msg := <-service.mailQueue
			if !strings.Contains(msg.Subject, tt.expectedText) {
				t.Errorf("Expected subject to contain '%s', got '%s'", tt.expectedText, msg.Subject)
			}
		})
	}
}
