package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

// PluginEventEngine handles event dispatching and action execution
type PluginEventEngine struct {
	db          *gorm.DB
	httpClient  *http.Client
	mailService *MailService
	mu          sync.RWMutex
	handlers    map[string][]*PluginEventHandler
}

// EventHandler represents a registered event handler
type PluginEventHandler struct {
	PluginID int64
	Trigger  *model.PluginTrigger
	Config   map[string]interface{}
}

// NewPluginEventEngine creates a new event engine
func NewPluginEventEngine(db *gorm.DB, mailService *MailService) *PluginEventEngine {
	return &PluginEventEngine{
		db: db,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		mailService: mailService,
		handlers:    make(map[string][]*PluginEventHandler),
	}
}

// LoadHandlers loads all enabled plugin handlers
func (e *PluginEventEngine) LoadHandlers() error {
	var plugins []*model.Plugin
	if err := e.db.Where("enabled = ?", true).Find(&plugins).Error; err != nil {
		return fmt.Errorf("failed to load plugins: %w", err)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	for _, plugin := range plugins {
		var triggers []model.PluginTrigger
		if err := e.db.Where("plugin_id = ?", plugin.ID).Find(&triggers).Error; err != nil {
			continue
		}

		var config map[string]interface{}
		if plugin.Config != "" {
			json.Unmarshal([]byte(plugin.Config), &config)
		}

		for i := range triggers {
			trigger := &triggers[i]
			events := e.parseEvents(trigger.Event)
			for _, event := range events {
				e.handlers[event] = append(e.handlers[event], &PluginEventHandler{
					PluginID: plugin.ID,
					Trigger:  trigger,
					Config:   config,
				})
			}
		}
	}

	return nil
}

// Emit dispatches an event to all registered handlers
func (e *PluginEventEngine) Emit(ctx context.Context, eventType string, data map[string]interface{}) error {
	e.mu.RLock()
	handlers := e.handlers[eventType]
	e.mu.RUnlock()

	if len(handlers) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(handlers))

	for _, handler := range handlers {
		wg.Add(1)
		go func(h *PluginEventHandler) {
			defer wg.Done()

			// Check conditions
			if !e.matchConditions(h.Trigger.Conditions, data) {
				return
			}

			// Execute actions
			for _, action := range h.Trigger.Actions {
				if err := e.executeAction(ctx, action, data, h.Config); err != nil {
					errCh <- fmt.Errorf("plugin %d action failed: %w", h.PluginID, err)
				}
			}
		}(handler)
	}

	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("event dispatch errors: %v", errs)
	}

	return nil
}

// parseEvents parses event string (may contain template variables)
func (e *PluginEventEngine) parseEvents(eventStr string) []string {
	// If it's a template variable, return as-is (will be resolved at runtime)
	if strings.Contains(eventStr, "{{") {
		return []string{eventStr}
	}

	// Otherwise, split by comma
	events := strings.Split(eventStr, ",")
	result := make([]string, 0, len(events))
	for _, event := range events {
		event = strings.TrimSpace(event)
		if event != "" {
			result = append(result, event)
		}
	}
	return result
}

// matchConditions checks if conditions are satisfied
func (e *PluginEventEngine) matchConditions(conditions []model.Condition, data map[string]interface{}) bool {
	for _, cond := range conditions {
		fieldValue := e.getFieldValue(data, cond.Field)

		switch cond.Operator {
		case "equals":
			if fmt.Sprint(fieldValue) != fmt.Sprint(cond.Value) {
				return false
			}
		case "contains":
			if !strings.Contains(fmt.Sprint(fieldValue), fmt.Sprint(cond.Value)) {
				return false
			}
		case "in":
			if !e.valueInList(fieldValue, cond.Value) {
				return false
			}
		case "gt":
			if !e.compareNumbers(fieldValue, cond.Value, ">") {
				return false
			}
		case "lt":
			if !e.compareNumbers(fieldValue, cond.Value, "<") {
				return false
			}
		case "gte":
			if !e.compareNumbers(fieldValue, cond.Value, ">=") {
				return false
			}
		case "lte":
			if !e.compareNumbers(fieldValue, cond.Value, "<=") {
				return false
			}
		case "regex":
			pattern, ok := cond.Value.(string)
			if !ok {
				return false
			}
			matched, err := regexp.MatchString(pattern, fmt.Sprint(fieldValue))
			if err != nil || !matched {
				return false
			}
		}
	}
	return true
}

// getFieldValue extracts field value from data using dot notation
func (e *PluginEventEngine) getFieldValue(data map[string]interface{}, field string) interface{} {
	parts := strings.Split(field, ".")
	var current interface{} = data

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			current = v[part]
		default:
			return nil
		}
	}

	return current
}

// valueInList checks if value is in list
func (e *PluginEventEngine) valueInList(value interface{}, list interface{}) bool {
	listStr := fmt.Sprint(list)
	valueStr := fmt.Sprint(value)

	var items []string
	if err := json.Unmarshal([]byte(listStr), &items); err != nil {
		return false
	}

	for _, item := range items {
		if item == valueStr {
			return true
		}
	}
	return false
}

// compareNumbers compares two numbers
func (e *PluginEventEngine) compareNumbers(a, b interface{}, op string) bool {
	// Simplified comparison - in production, use proper type conversion
	aStr := fmt.Sprint(a)
	bStr := fmt.Sprint(b)

	switch op {
	case ">":
		return aStr > bStr
	case "<":
		return aStr < bStr
	case ">=":
		return aStr >= bStr
	case "<=":
		return aStr <= bStr
	}
	return false
}

// executeAction executes a plugin action
func (e *PluginEventEngine) executeAction(ctx context.Context, action model.PluginAction, data, config map[string]interface{}) error {
	switch action.Type {
	case "http_post":
		return e.executeHTTPPost(ctx, action, data, config)
	case "add_label":
		return e.executeAddLabel(ctx, action, data, config)
	case "send_email":
		return e.executeSendEmail(ctx, action, data, config)
	case "notify_user":
		return e.executeNotifyUser(ctx, action, data, config)
	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}

// executeHTTPPost executes HTTP POST action
func (e *PluginEventEngine) executeHTTPPost(ctx context.Context, action model.PluginAction, data, config map[string]interface{}) error {
	url := e.renderTemplate(action.URL, data, config)
	body := e.renderTemplate(action.Body, data, config)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader([]byte(body)))
	if err != nil {
		return err
	}

	// Set headers
	for k, v := range action.Headers {
		req.Header.Set(k, e.renderTemplate(v, data, config))
	}
	req.Header.Set("Content-Type", "application/json")

	// Add DingTalk signature if secret is provided
	if secret, ok := config["secret"].(string); ok && secret != "" {
		timestamp := fmt.Sprint(time.Now().UnixMilli())
		sign := e.generateDingtalkSign(secret, timestamp)
		req.URL.RawQuery += fmt.Sprintf("&timestamp=%s&sign=%s", timestamp, sign)
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP request failed with status %d", resp.StatusCode)
	}

	return nil
}

// generateDingtalkSign generates DingTalk signature
func (e *PluginEventEngine) generateDingtalkSign(secret, timestamp string) string {
	stringToSign := timestamp + "\n" + secret
	hmac256 := hmac.New(sha256.New, []byte(secret))
	hmac256.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(hmac256.Sum(nil))
}

// executeAddLabel executes add label action
func (e *PluginEventEngine) executeAddLabel(ctx context.Context, action model.PluginAction, data, config map[string]interface{}) error {
	// Extract task ID and label from data
	taskID, ok := data["task.id"].(int64)
	if !ok {
		return fmt.Errorf("task.id not found in event data")
	}

	// Parse rules from config
	rulesStr := e.renderTemplate(fmt.Sprint(action.Params["rules"]), data, config)
	var rules []map[string]string
	if err := json.Unmarshal([]byte(rulesStr), &rules); err != nil {
		return err
	}

	// Check title against rules
	title := fmt.Sprint(data["task.title"])
	for _, rule := range rules {
		keyword := rule["keyword"]
		label := rule["label"]
		if strings.Contains(strings.ToLower(title), strings.ToLower(keyword)) {
			// Add label via database
			if err := e.db.Exec("INSERT INTO task_label (task_id, label) VALUES (?, ?)", taskID, label).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

// executeSendEmail executes send email action
func (e *PluginEventEngine) executeSendEmail(ctx context.Context, action model.PluginAction, data, config map[string]interface{}) error {
	to := e.renderTemplate(fmt.Sprint(action.Params["to"]), data, config)
	subject := e.renderTemplate(fmt.Sprint(action.Params["subject"]), data, config)
	body := e.renderTemplate(fmt.Sprint(action.Params["body"]), data, config)

	// Parse recipients (comma-separated)
	recipients := strings.Split(to, ",")
	for i := range recipients {
		recipients[i] = strings.TrimSpace(recipients[i])
	}

	// Check if body is HTML
	isHTML := strings.Contains(body, "<") && strings.Contains(body, ">")

	// Send email via mail service
	if e.mailService != nil {
		msg := &EmailMessage{
			To:      recipients,
			Subject: subject,
			Body:    body,
			IsHTML:  isHTML,
		}

		if err := e.mailService.SendMail(msg); err != nil {
			return fmt.Errorf("failed to send email: %w", err)
		}
	} else {
		// Fallback: just log if mail service not available
		fmt.Printf("Send email to: %s, subject: %s\n", to, subject)
	}

	return nil
}

// executeNotifyUser executes notify user action
func (e *PluginEventEngine) executeNotifyUser(ctx context.Context, action model.PluginAction, data, config map[string]interface{}) error {
	// TODO: Integrate with notification service
	fmt.Printf("Notify user: %v\n", action.Params)
	return nil
}

// renderTemplate renders template string with data and config
func (e *PluginEventEngine) renderTemplate(tmplStr string, data, config map[string]interface{}) string {
	// Simple template rendering - replace {{.field}} with actual values
	result := tmplStr

	// Replace config variables
	if config != nil {
		for k, v := range config {
			placeholder := fmt.Sprintf("{{.config.%s}}", k)
			result = strings.ReplaceAll(result, placeholder, fmt.Sprint(v))
		}
	}

	// Replace data variables
	if data != nil {
		for k, v := range data {
			placeholder := fmt.Sprintf("{{.%s}}", k)
			result = strings.ReplaceAll(result, placeholder, fmt.Sprint(v))
		}
	}

	return result
}
