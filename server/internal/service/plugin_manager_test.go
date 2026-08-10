package service

import (
	"testing"
	"time"

	"iforge/iforge/internal/model"
)

func TestNewPluginManager(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	pm := NewPluginManager(db)
	if pm == nil {
		t.Fatal("NewPluginManager returned nil")
	}

	if pm.db != db {
		t.Error("PluginManager.db not set correctly")
	}

	if pm.httpClient == nil {
		t.Error("PluginManager.httpClient is nil")
	}

	if pm.statuses == nil {
		t.Error("PluginManager.statuses is nil")
	}

	if pm.eventHandlers == nil {
		t.Error("PluginManager.eventHandlers is nil")
	}

	if pm.stopCh == nil {
		t.Error("PluginManager.stopCh is nil")
	}

	// Cleanup
	close(pm.stopCh)
}

func TestPluginManager_LoadPlugins(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	pm := NewPluginManager(db)
	defer close(pm.stopCh)

	// Create test plugin
	plugin := &model.Plugin{
		Name:        "test-plugin",
		Version:     "1.0.0",
		Description: "Test plugin",
		Endpoint:    "http://localhost:8080",
		Enabled:     true,
	}
	if err := db.Create(plugin).Error; err != nil {
		t.Fatalf("Failed to create plugin: %v", err)
	}

	// Load plugins
	if err := pm.LoadPlugins(); err != nil {
		t.Fatalf("LoadPlugins failed: %v", err)
	}

	// Verify status is initialized
	pm.mu.RLock()
	status, exists := pm.statuses[plugin.ID]
	pm.mu.RUnlock()

	if !exists {
		t.Fatal("Plugin status not initialized")
	}

	if status.Status != "unknown" {
		t.Errorf("Plugin status = %s, want unknown", status.Status)
	}
}

func TestPluginManager_LoadPlugins_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	pm := NewPluginManager(db)
	defer close(pm.stopCh)

	// Load plugins (empty database)
	if err := pm.LoadPlugins(); err != nil {
		t.Fatalf("LoadPlugins failed: %v", err)
	}

	// Verify no statuses
	pm.mu.RLock()
	count := len(pm.statuses)
	pm.mu.RUnlock()

	if count != 0 {
		t.Errorf("Status count = %d, want 0", count)
	}
}

func TestPluginManager_GetPlugin(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	pm := NewPluginManager(db)
	defer close(pm.stopCh)

	// Create test plugin
	plugin := &model.Plugin{
		Name:     "test-plugin",
		Version:  "1.0.0",
		Endpoint: "http://localhost:8080",
		Enabled:  true,
	}
	if err := db.Create(plugin).Error; err != nil {
		t.Fatalf("Failed to create plugin: %v", err)
	}

	// Get existing plugin
	retrieved, err := pm.GetPlugin(plugin.ID)
	if err != nil {
		t.Fatalf("GetPlugin failed: %v", err)
	}

	if retrieved.Name != "test-plugin" {
		t.Errorf("Plugin name = %s, want test-plugin", retrieved.Name)
	}

	// Get non-existent plugin
	_, err = pm.GetPlugin(99999)
	if err != ErrPluginNotFound {
		t.Errorf("GetPlugin error = %v, want ErrPluginNotFound", err)
	}
}

func TestPluginManager_GetPluginStatus(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	pm := NewPluginManager(db)
	defer close(pm.stopCh)

	// Create test plugin
	plugin := &model.Plugin{
		Name:     "test-plugin",
		Version:  "1.0.0",
		Endpoint: "http://localhost:8080",
		Enabled:  true,
	}
	if err := db.Create(plugin).Error; err != nil {
		t.Fatalf("Failed to create plugin: %v", err)
	}

	// Load plugins
	if err := pm.LoadPlugins(); err != nil {
		t.Fatalf("LoadPlugins failed: %v", err)
	}

	// Get existing plugin status
	status, err := pm.GetPluginStatus(plugin.ID)
	if err != nil {
		t.Fatalf("GetPluginStatus failed: %v", err)
	}

	if status.PluginID != plugin.ID {
		t.Errorf("PluginStatus.PluginID = %d, want %d", status.PluginID, plugin.ID)
	}

	if status.Status != "unknown" {
		t.Errorf("PluginStatus.Status = %s, want unknown", status.Status)
	}

	// Get non-existent plugin status
	_, err = pm.GetPluginStatus(99999)
	if err != ErrPluginNotFound {
		t.Errorf("GetPluginStatus error = %v, want ErrPluginNotFound", err)
	}
}

func TestPluginManager_ListPlugins(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	pm := NewPluginManager(db)
	defer close(pm.stopCh)

	// Create multiple test plugins
	plugins := []*model.Plugin{
		{Name: "plugin1", Version: "1.0.0", Endpoint: "http://localhost:8081", Enabled: true},
		{Name: "plugin2", Version: "2.0.0", Endpoint: "http://localhost:8082", Enabled: true},
		{Name: "plugin3", Version: "3.0.0", Endpoint: "http://localhost:8083", Enabled: false},
	}

	for _, p := range plugins {
		if err := db.Create(p).Error; err != nil {
			t.Fatalf("Failed to create plugin: %v", err)
		}
	}

	// List all plugins
	list := pm.ListPlugins()

	if len(list) != 3 {
		t.Errorf("Plugin count = %d, want 3", len(list))
	}
}

func TestPluginManager_ListPlugins_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	pm := NewPluginManager(db)
	defer close(pm.stopCh)

	// List all plugins (empty database)
	list := pm.ListPlugins()

	if len(list) != 0 {
		t.Errorf("Plugin count = %d, want 0", len(list))
	}
}

func TestPluginManager_RegisterPlugin(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	pm := NewPluginManager(db)
	defer close(pm.stopCh)

	// Register new plugin
	plugin := &model.Plugin{
		Name:     "test-plugin",
		Version:  "1.0.0",
		Endpoint: "http://localhost:8080",
		Enabled:  true,
	}

	err := pm.RegisterPlugin(plugin)
	if err != nil {
		t.Fatalf("RegisterPlugin failed: %v", err)
	}

	// Verify plugin exists in database
	var count int64
	db.Model(&model.Plugin{}).Where("name = ?", "test-plugin").Count(&count)
	if count != 1 {
		t.Errorf("Plugin count in DB = %d, want 1", count)
	}
}

func TestPluginManager_RegisterPlugin_Duplicate(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	pm := NewPluginManager(db)
	defer close(pm.stopCh)

	// Register first plugin
	plugin1 := &model.Plugin{
		Name:     "test-plugin",
		Version:  "1.0.0",
		Endpoint: "http://localhost:8080",
		Enabled:  true,
	}

	err := pm.RegisterPlugin(plugin1)
	if err != nil {
		t.Fatalf("RegisterPlugin failed: %v", err)
	}

	// Try registering plugin with same name
	plugin2 := &model.Plugin{
		Name:     "test-plugin",
		Version:  "2.0.0",
		Endpoint: "http://localhost:8081",
		Enabled:  true,
	}

	err = pm.RegisterPlugin(plugin2)
	if err != ErrPluginExists {
		t.Errorf("RegisterPlugin error = %v, want ErrPluginExists", err)
	}
}

func TestPluginManager_UnregisterPlugin(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	pm := NewPluginManager(db)
	defer close(pm.stopCh)

	// Register plugin
	plugin := &model.Plugin{
		Name:     "test-plugin",
		Version:  "1.0.0",
		Endpoint: "http://localhost:8080",
		Enabled:  true,
	}

	err := pm.RegisterPlugin(plugin)
	if err != nil {
		t.Fatalf("RegisterPlugin failed: %v", err)
	}

	// Unregister plugin
	err = pm.UnregisterPlugin(plugin.ID)
	if err != nil {
		t.Fatalf("UnregisterPlugin failed: %v", err)
	}

	// Verify plugin is removed from database
	var count int64
	db.Model(&model.Plugin{}).Where("id = ?", plugin.ID).Count(&count)
	if count != 0 {
		t.Error("Plugin should be removed from database")
	}
}

func TestPluginManager_UnregisterPlugin_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	pm := NewPluginManager(db)
	defer close(pm.stopCh)

	err := pm.UnregisterPlugin(99999)
	if err != ErrPluginNotFound {
		t.Errorf("UnregisterPlugin error = %v, want ErrPluginNotFound", err)
	}
}

func TestPluginManager_DisablePlugin(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	pm := NewPluginManager(db)
	defer close(pm.stopCh)

	// Register plugin
	plugin := &model.Plugin{
		Name:     "test-plugin",
		Version:  "1.0.0",
		Endpoint: "http://localhost:8080",
		Enabled:  true,
	}

	err := pm.RegisterPlugin(plugin)
	if err != nil {
		t.Fatalf("RegisterPlugin failed: %v", err)
	}

	// Disable plugin
	err = pm.DisablePlugin(plugin.ID)
	if err != nil {
		t.Fatalf("DisablePlugin failed: %v", err)
	}

	// Verify plugin is disabled in database
	var disabledPlugin model.Plugin
	db.First(&disabledPlugin, plugin.ID)
	if disabledPlugin.Enabled {
		t.Error("Plugin should be disabled")
	}
}

func TestPluginManager_DisablePlugin_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	pm := NewPluginManager(db)
	defer close(pm.stopCh)

	err := pm.DisablePlugin(99999)
	if err != ErrPluginNotFound {
		t.Errorf("DisablePlugin error = %v, want ErrPluginNotFound", err)
	}
}

func TestPluginManager_EnablePlugin(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	pm := NewPluginManager(db)
	defer close(pm.stopCh)

	// Register plugin (disabled)
	plugin := &model.Plugin{
		Name:     "test-plugin",
		Version:  "1.0.0",
		Endpoint: "http://localhost:8080",
		Enabled:  false,
	}

	err := pm.RegisterPlugin(plugin)
	if err != nil {
		t.Fatalf("RegisterPlugin failed: %v", err)
	}

	// Enable plugin
	err = pm.EnablePlugin(plugin.ID)
	if err != nil {
		t.Fatalf("EnablePlugin failed: %v", err)
	}

	// Verify plugin is enabled in database
	var enabledPlugin model.Plugin
	db.First(&enabledPlugin, plugin.ID)
	if !enabledPlugin.Enabled {
		t.Error("Plugin should be enabled")
	}
}

func TestPluginManager_EnablePlugin_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	pm := NewPluginManager(db)
	defer close(pm.stopCh)

	err := pm.EnablePlugin(99999)
	if err != ErrPluginNotFound {
		t.Errorf("EnablePlugin error = %v, want ErrPluginNotFound", err)
	}
}

func TestPluginManager_Stop(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	pm := NewPluginManager(db)

	// Stop plugin manager
	pm.Stop()

	// Verify stopCh is closed
	select {
	case <-pm.stopCh:
		// Normal shutdown
	case <-time.After(1 * time.Second):
		t.Error("PluginManager.Stop did not close stopCh")
	}
}
