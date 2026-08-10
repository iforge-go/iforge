package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	gitsvc "iforge/iforge/internal/git"
	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

var (
	ErrWebhookNotFound = errors.New("webhook not found")
)

// WebhookService handles webhook operations
type WebhookService struct {
	db *gorm.DB

	// P2-3: Webhook delivery worker pool for async delivery with backpressure
	deliveryQueue chan deliveryTask
	workerCount   int
	wg            sync.WaitGroup
	stopCh        chan struct{}
}

// deliveryTask represents a webhook delivery task
type deliveryTask struct {
	webhook *model.Webhook
	event   string
	action  string
	payload interface{}
}

// NewWebhookService creates a new WebhookService
func NewWebhookService(db *gorm.DB) *WebhookService {
	s := &WebhookService{
		db:            db,
		deliveryQueue: make(chan deliveryTask, 1000), // Buffer for backpressure
		workerCount:   5,                             // Fixed number of delivery workers
		stopCh:        make(chan struct{}),
	}

	// Start delivery workers
	s.startDeliveryWorkers()

	return s
}

// startDeliveryWorkers starts the webhook delivery worker pool
func (s *WebhookService) startDeliveryWorkers() {
	for i := 0; i < s.workerCount; i++ {
		s.wg.Add(1)
		go s.deliveryWorker(i)
	}
	gitsvc.GetLogger().Info("Webhook delivery workers started", map[string]interface{}{"count": s.workerCount})
}

// deliveryWorker is a worker that processes webhook delivery tasks
func (s *WebhookService) deliveryWorker(id int) {
	defer s.wg.Done()

	for {
		select {
		case <-s.stopCh:
			return
		case task, ok := <-s.deliveryQueue:
			if !ok {
				return
			}
			if err := s.deliverAndRecord(task.webhook, task.event, task.action, task.payload); err != nil {
				gitsvc.GetLogger().Error("Webhook delivery failed", err, map[string]interface{}{
					"worker_id":  id,
					"webhook_id": task.webhook.ID,
				})
			}
		}
	}
}

// Shutdown gracefully shuts down the webhook delivery workers
func (s *WebhookService) Shutdown(timeout time.Duration) error {
	close(s.stopCh)
	close(s.deliveryQueue)

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return errors.New("webhook service shutdown timeout")
	}
}

// CreateWebhook creates a new webhook
func (s *WebhookService) CreateWebhook(owner, repo, url, contentType string, token *string, events string, active bool, insecureSSL bool) (*model.Webhook, error) {
	webhook := &model.Webhook{
		UserName:       owner,
		RepositoryName: repo,
		URL:            url,
		Token:          token,
		ContentType:    contentType,
		Active:         active,
		Events:         events,
		SSLVerify:      !insecureSSL,
		InsecureSSL:    insecureSSL,
	}

	if err := s.db.Create(webhook).Error; err != nil {
		return nil, err
	}

	return webhook, nil
}

// GetWebhook gets a webhook by ID
func (s *WebhookService) GetWebhook(id uint) (*model.Webhook, error) {
	webhook := &model.Webhook{}
	if err := s.db.First(webhook, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrWebhookNotFound
		}
		return nil, err
	}
	return webhook, nil
}

// ListWebhooks lists all webhooks for a repository
func (s *WebhookService) ListWebhooks(owner, repo string) ([]*model.Webhook, error) {
	var webhooks []*model.Webhook
	err := s.db.
		Where("user_name = ? AND repository_name = ?", owner, repo).
		Order("created_at DESC").
		Find(&webhooks).Error
	return webhooks, err
}

// UpdateWebhook updates a webhook
func (s *WebhookService) UpdateWebhook(id uint, url, contentType string, token *string, events string, active bool, insecureSSL bool) (*model.Webhook, error) {
	webhook, err := s.GetWebhook(id)
	if err != nil {
		return nil, err
	}

	webhook.URL = url
	webhook.ContentType = contentType
	webhook.Token = token
	webhook.Events = events
	webhook.Active = active
	webhook.InsecureSSL = insecureSSL
	webhook.SSLVerify = !insecureSSL

	if err := s.db.Save(webhook).Error; err != nil {
		return nil, err
	}

	return webhook, nil
}

// DeleteWebhook deletes a webhook
func (s *WebhookService) DeleteWebhook(id uint) error {
	return s.db.Delete(&model.Webhook{}, id).Error
}

// ListWebhookDeliveries lists all deliveries for a webhook
func (s *WebhookService) ListWebhookDeliveries(webhookID uint) ([]*model.WebhookDelivery, error) {
	var deliveries []*model.WebhookDelivery
	err := s.db.
		Where("webhook_id = ?", webhookID).
		Order("created_at DESC").
		Limit(50).
		Find(&deliveries).Error
	return deliveries, err
}

// RedeliverWebhook redelivers a failed webhook delivery
func (s *WebhookService) RedeliverWebhook(deliveryID uint) error {
	delivery := &model.WebhookDelivery{}
	if err := s.db.First(delivery, deliveryID).Error; err != nil {
		return err
	}

	webhook, err := s.GetWebhook(delivery.WebhookID)
	if err != nil {
		return err
	}

	// Resend the webhook
	return s.deliverWebhook(webhook, delivery.Event, delivery.Action, []byte(delivery.RequestBody))
}

// TestWebhook sends a test ping event to the webhook
func (s *WebhookService) TestWebhook(id uint, repo *model.Repository, sender *model.Account) error {
	webhook, err := s.GetWebhook(id)
	if err != nil {
		return err
	}

	// Send a test ping event
	payload := map[string]interface{}{
		"zen":       "Half measures are as bad as nothing at all.",
		"hook_id":   webhook.ID,
		"hook_type": "Repository",
		"repository": map[string]interface{}{
			"name":      repo.RepositoryName,
			"full_name": fmt.Sprintf("%s/%s", repo.UserName, repo.RepositoryName),
			"private":   repo.IsPrivate,
			"html_url":  fmt.Sprintf("/%s/%s", repo.UserName, repo.RepositoryName),
		},
		"sender": map[string]interface{}{
			"login":      sender.UserName,
			"avatar_url": fmt.Sprintf("/avatars/%s", sender.UserName),
		},
	}

	return s.deliverAndRecord(webhook, "ping", "", payload)
}

// TriggerWebhook triggers a webhook with the given payload (legacy method, kept for compatibility)
func (s *WebhookService) TriggerWebhook(webhook *model.Webhook, event string, payload interface{}) error {
	return s.deliverAndRecord(webhook, event, "", payload)
}

// deliverAndRecord delivers a webhook and records the delivery
func (s *WebhookService) deliverAndRecord(webhook *model.Webhook, event, action string, payload interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return s.deliverWebhook(webhook, event, action, jsonData)
}

// deliverWebhook actually sends the webhook HTTP request
func (s *WebhookService) deliverWebhook(webhook *model.Webhook, event, action string, jsonData []byte) error {
	startTime := time.Now()

	req, err := http.NewRequest("POST", webhook.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		s.recordDelivery(webhook.ID, event, action, string(jsonData), "", 0, false, err.Error(), startTime)
		return err
	}

	req.Header.Set("Content-Type", webhook.ContentType)
	req.Header.Set("X-iForge-Event", event)
	req.Header.Set("X-iForge-Action", action)

	// Add SHA-256 signature if token is provided
	if webhook.Token != nil && *webhook.Token != "" {
		mac := hmac.New(sha256.New, []byte(*webhook.Token))
		mac.Write(jsonData)
		signature := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-iForge-Signature", "sha256="+signature)
	}

	// Configure HTTP client with SSL verification option
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: webhook.InsecureSSL,
		},
	}
	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
	}

	resp, err := client.Do(req)
	duration := time.Since(startTime).Milliseconds()

	if err != nil {
		s.recordDelivery(webhook.ID, event, action, string(jsonData), "", int(duration), false, err.Error(), startTime)
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	success := resp.StatusCode >= 200 && resp.StatusCode < 300

	s.recordDelivery(webhook.ID, event, action, string(jsonData), string(body), int(duration), success, "", startTime)

	if !success {
		return fmt.Errorf("webhook delivery failed with status %d", resp.StatusCode)
	}

	return nil
}

// recordDelivery records a webhook delivery attempt
func (s *WebhookService) recordDelivery(webhookID uint, event, action, requestBody, responseBody string, duration int, success bool, errorMessage string, createdAt time.Time) {
	delivery := &model.WebhookDelivery{
		WebhookID:    webhookID,
		Event:        event,
		Action:       action,
		RequestBody:  requestBody,
		ResponseBody: responseBody,
		Duration:     duration,
		Success:      success,
		ErrorMessage: errorMessage,
		CreatedAt:    createdAt,
	}

	if err := s.db.Create(delivery).Error; err != nil {
		gitsvc.GetLogger().Error("Failed to record webhook delivery", err, nil)
	}
}

// TriggerRepositoryWebhooks triggers all active webhooks for a repository that match the event type.
// Webhook deliveries are queued to the delivery worker pool for async processing with backpressure.
func (s *WebhookService) TriggerRepositoryWebhooks(owner, repo, event, action string, payload interface{}) {
	webhooks, err := s.ListWebhooks(owner, repo)
	if err != nil {
		gitsvc.GetLogger().Error("Failed to list webhooks", err, map[string]interface{}{"owner": owner, "repo": repo})
		return
	}

	for _, webhook := range webhooks {
		// Skip inactive webhooks
		if !webhook.Active {
			continue
		}

		// Check if webhook subscribes to this event
		if !s.webhookSubscribesToEvent(webhook, event) {
			continue
		}

		// Queue delivery task to worker pool (non-blocking with backpressure)
		task := deliveryTask{
			webhook: webhook,
			event:   event,
			action:  action,
			payload: payload,
		}

		select {
		case s.deliveryQueue <- task:
			// Successfully queued
		default:
			// Queue full, log warning and drop (backpressure)
			gitsvc.GetLogger().Warn("Webhook delivery queue full, dropping task", map[string]interface{}{
				"webhook_id": webhook.ID,
				"event":      event,
			})
		}
	}
}

// webhookSubscribesToEvent checks if a webhook subscribes to a specific event
func (s *WebhookService) webhookSubscribesToEvent(webhook *model.Webhook, event string) bool {
	if webhook.Events == "" || webhook.Events == "[]" {
		return true
	}

	var events []string
	if err := json.Unmarshal([]byte(webhook.Events), &events); err != nil {
		return false
	}

	for _, e := range events {
		if e == event || e == "*" {
			return true
		}
	}

	return false
}
