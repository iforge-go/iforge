package service

import (
	"bytes"
	"context"
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
	ErrPluginNotFound    = errors.New("plugin not found")
	ErrPluginExists      = errors.New("plugin already exists")
	ErrPluginDisabled    = errors.New("plugin is disabled")
	ErrPluginUnreachable = errors.New("plugin is unreachable")
)

// PluginManager manages plugin lifecycle and event dispatching
type PluginManager struct {
	mu            sync.RWMutex
	httpClient    *http.Client
	statuses      map[int64]*model.PluginStatus
	eventHandlers map[string][]*LegacyEventHandler
	stopCh        chan struct{}
	db            *gorm.DB
}

// LegacyEventHandler represents a registered event handler (legacy webhook-based plugins)
type LegacyEventHandler struct {
	PluginID int64
	Endpoint string
	Priority int
}

// NewPluginManager creates a new plugin manager
func NewPluginManager(db *gorm.DB) *PluginManager {
	pm := &PluginManager{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		statuses:      make(map[int64]*model.PluginStatus),
		eventHandlers: make(map[string][]*LegacyEventHandler),
		stopCh:        make(chan struct{}),
		db:            db,
	}

	go pm.healthCheckLoop()

	return pm
}

// LoadPlugins loads all plugins from database
func (pm *PluginManager) LoadPlugins() error {
	var plugins []*model.Plugin
	if err := pm.db.Find(&plugins).Error; err != nil {
		return fmt.Errorf("failed to query plugins: %w", err)
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	for _, plugin := range plugins {
		pm.statuses[plugin.ID] = &model.PluginStatus{
			PluginID: plugin.ID,
			Name:     plugin.Name,
			Version:  plugin.Version,
			Status:   "unknown",
		}

		if plugin.Enabled {
			pm.loadPluginEventHandlers(plugin)
		}
	}

	return nil
}

// loadPluginEventHandlers loads event handlers for a specific plugin
func (pm *PluginManager) loadPluginEventHandlers(plugin *model.Plugin) {
	var events []*model.PluginEvent
	err := pm.db.Where("plugin_id = ? AND enabled = ?", plugin.ID, true).Find(&events).Error
	if err != nil {
		logger := gitsvc.GetLogger()
		logger.Error("Failed to load event handlers for plugin", err, map[string]interface{}{"plugin_id": plugin.ID})
		return
	}

	for _, event := range events {
		pm.eventHandlers[event.EventType] = append(pm.eventHandlers[event.EventType], &LegacyEventHandler{
			PluginID: plugin.ID,
			Endpoint: plugin.Endpoint + event.Handler,
			Priority: 0,
		})
	}
}

// RegisterPlugin registers a new plugin
func (pm *PluginManager) RegisterPlugin(plugin *model.Plugin) error {
	// Check if plugin with same name already exists
	var existing model.Plugin
	if err := pm.db.Where("name = ?", plugin.Name).First(&existing).Error; err == nil {
		return ErrPluginExists
	}

	plugin.CreatedAt = time.Now()
	plugin.UpdatedAt = time.Now()

	if err := pm.db.Create(plugin).Error; err != nil {
		return fmt.Errorf("failed to insert plugin: %w", err)
	}

	pm.mu.Lock()
	pm.statuses[plugin.ID] = &model.PluginStatus{
		PluginID: plugin.ID,
		Name:     plugin.Name,
		Version:  plugin.Version,
		Status:   "unknown",
	}
	if plugin.Enabled {
		pm.loadPluginEventHandlers(plugin)
	}
	pm.mu.Unlock()

	return nil
}

// UnregisterPlugin removes a plugin
func (pm *PluginManager) UnregisterPlugin(pluginID int64) error {
	var plugin model.Plugin
	if err := pm.db.First(&plugin, pluginID).Error; err != nil {
		return ErrPluginNotFound
	}

	// Delete plugin triggers (template-based plugins)
	pm.db.Where("plugin_id = ?", pluginID).Delete(&model.PluginTrigger{})

	// Delete event handlers (legacy plugins)
	pm.db.Where("plugin_id = ?", pluginID).Delete(&model.PluginEvent{})

	// Delete plugin
	if err := pm.db.Delete(&model.Plugin{}, pluginID).Error; err != nil {
		return fmt.Errorf("failed to delete plugin: %w", err)
	}

	pm.mu.Lock()
	delete(pm.statuses, pluginID)
	for eventType, handlers := range pm.eventHandlers {
		var newHandlers []*LegacyEventHandler
		for _, h := range handlers {
			if h.PluginID != pluginID {
				newHandlers = append(newHandlers, h)
			}
		}
		pm.eventHandlers[eventType] = newHandlers
	}
	pm.mu.Unlock()

	logger := gitsvc.GetLogger()
	logger.Info("Unregistered plugin", map[string]interface{}{"plugin_name": plugin.Name})
	return nil
}

// EnablePlugin enables a plugin
func (pm *PluginManager) EnablePlugin(pluginID int64) error {
	var plugin model.Plugin
	if err := pm.db.First(&plugin, pluginID).Error; err != nil {
		return ErrPluginNotFound
	}

	if err := pm.db.Model(&model.Plugin{}).
		Where("id = ?", pluginID).
		Updates(map[string]interface{}{
			"enabled":    true,
			"updated_at": time.Now(),
		}).Error; err != nil {
		return fmt.Errorf("failed to enable plugin: %w", err)
	}

	pm.mu.Lock()
	pm.loadPluginEventHandlers(&plugin)
	pm.mu.Unlock()

	logger := gitsvc.GetLogger()
	logger.Info("Enabled plugin", map[string]interface{}{"plugin_name": plugin.Name})
	return nil
}

// DisablePlugin disables a plugin
func (pm *PluginManager) DisablePlugin(pluginID int64) error {
	var plugin model.Plugin
	if err := pm.db.First(&plugin, pluginID).Error; err != nil {
		return ErrPluginNotFound
	}

	if err := pm.db.Model(&model.Plugin{}).
		Where("id = ?", pluginID).
		Updates(map[string]interface{}{
			"enabled":    false,
			"updated_at": time.Now(),
		}).Error; err != nil {
		return fmt.Errorf("failed to disable plugin: %w", err)
	}

	pm.mu.Lock()
	for eventType, handlers := range pm.eventHandlers {
		var newHandlers []*LegacyEventHandler
		for _, h := range handlers {
			if h.PluginID != pluginID {
				newHandlers = append(newHandlers, h)
			}
		}
		pm.eventHandlers[eventType] = newHandlers
	}
	pm.mu.Unlock()

	logger := gitsvc.GetLogger()
	logger.Info("Disabled plugin", map[string]interface{}{"plugin_name": plugin.Name})
	return nil
}

// DispatchEvent dispatches an event to all registered handlers
func (pm *PluginManager) DispatchEvent(ctx context.Context, eventType string, payload interface{}) error {
	pm.mu.RLock()
	handlers := pm.eventHandlers[eventType]
	pm.mu.RUnlock()

	if len(handlers) == 0 {
		return nil
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(handlers))

	for _, handler := range handlers {
		wg.Add(1)
		go func(h *LegacyEventHandler) {
			defer wg.Done()

			var plugin model.Plugin
			if err := pm.db.First(&plugin, h.PluginID).Error; err != nil {
				return
			}
			if !plugin.Enabled {
				return
			}

			err := pm.callPlugin(ctx, h.Endpoint, payloadBytes)
			if err != nil {
				logger := gitsvc.GetLogger()
				logger.Error("Failed to call plugin", err, map[string]interface{}{
					"plugin_name": plugin.Name,
					"endpoint":    h.Endpoint,
				})
				errCh <- err

				pm.mu.Lock()
				if status, ok := pm.statuses[h.PluginID]; ok {
					status.ErrorCount++
				}
				pm.mu.Unlock()
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
		return fmt.Errorf("plugin dispatch errors: %v", errs)
	}

	return nil
}

// callPlugin calls a plugin endpoint
func (pm *PluginManager) callPlugin(ctx context.Context, endpoint string, payload []byte) error {
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "iForge-Plugin-Client/1.0")

	resp, err := pm.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("plugin returned error status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// healthCheckLoop periodically checks plugin health
func (pm *PluginManager) healthCheckLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pm.checkPluginHealth()
		case <-pm.stopCh:
			return
		}
	}
}

// checkPluginHealth checks health of all enabled plugins
func (pm *PluginManager) checkPluginHealth() {
	var plugins []*model.Plugin
	pm.db.Where("enabled = ?", true).Find(&plugins)

	for _, plugin := range plugins {
		if plugin.Endpoint == "" {
			continue
		}
		go func(p *model.Plugin) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			healthEndpoint := p.Endpoint + "/health"
			req, err := http.NewRequestWithContext(ctx, "GET", healthEndpoint, nil)
			if err != nil {
				pm.updatePluginStatus(p.ID, "error")
				return
			}

			resp, err := pm.httpClient.Do(req)
			if err != nil {
				pm.updatePluginStatus(p.ID, "error")
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == 200 {
				pm.updatePluginStatus(p.ID, "running")
			} else {
				pm.updatePluginStatus(p.ID, "error")
			}
		}(plugin)
	}
}

// updatePluginStatus updates the status of a plugin
func (pm *PluginManager) updatePluginStatus(pluginID int64, status string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if s, ok := pm.statuses[pluginID]; ok {
		s.Status = status
		s.LastPing = time.Now()
	}
}

// GetPluginStatus returns the status of a plugin
func (pm *PluginManager) GetPluginStatus(pluginID int64) (*model.PluginStatus, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	status, exists := pm.statuses[pluginID]
	if !exists {
		return nil, ErrPluginNotFound
	}

	return status, nil
}

// GetAllPluginStatuses returns statuses of all plugins
func (pm *PluginManager) GetAllPluginStatuses() []*model.PluginStatus {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	statuses := make([]*model.PluginStatus, 0, len(pm.statuses))
	for _, s := range pm.statuses {
		statuses = append(statuses, s)
	}

	return statuses
}

// Stop stops the plugin manager
func (pm *PluginManager) Stop() {
	close(pm.stopCh)
}

// GetPlugin returns a plugin by ID from database
func (pm *PluginManager) GetPlugin(pluginID int64) (*model.Plugin, error) {
	var plugin model.Plugin
	if err := pm.db.First(&plugin, pluginID).Error; err != nil {
		return nil, ErrPluginNotFound
	}
	return &plugin, nil
}

// ListPlugins returns all plugins from database
func (pm *PluginManager) ListPlugins() []*model.Plugin {
	var plugins []*model.Plugin
	if err := pm.db.Find(&plugins).Error; err != nil {
		return []*model.Plugin{}
	}
	return plugins
}

// UpdatePlugin updates plugin configuration
func (pm *PluginManager) UpdatePlugin(pluginID int64, config string) error {
	var plugin model.Plugin
	if err := pm.db.First(&plugin, pluginID).Error; err != nil {
		return ErrPluginNotFound
	}

	if err := pm.db.Model(&model.Plugin{}).
		Where("id = ?", pluginID).
		Updates(map[string]interface{}{
			"config":     config,
			"updated_at": time.Now(),
		}).Error; err != nil {
		return fmt.Errorf("failed to update plugin config: %w", err)
	}

	logger := gitsvc.GetLogger()
	logger.Info("Updated plugin config", map[string]interface{}{"plugin_name": plugin.Name})
	return nil
}
