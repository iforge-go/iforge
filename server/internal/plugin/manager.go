package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Manager manages plugin lifecycle
type Manager struct {
	mu       sync.RWMutex
	plugins  map[string]Plugin
	enabled  map[string]bool
	configs  map[string]map[string]interface{}
	homeDir  string
}

// NewManager creates a new plugin manager
func NewManager(homeDir string) *Manager {
	return &Manager{
		plugins: make(map[string]Plugin),
		enabled: make(map[string]bool),
		configs: make(map[string]map[string]interface{}),
		homeDir: homeDir,
	}
}

// LoadPlugins loads all plugins from the plugins directory
func (m *Manager) LoadPlugins() error {
	pluginsDir := filepath.Join(m.homeDir, "plugins")
	
	// Create plugins directory if it doesn't exist
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugins directory: %w", err)
	}
	
	// Load plugin configurations
	configFile := filepath.Join(pluginsDir, "plugins.json")
	if _, err := os.Stat(configFile); err == nil {
		data, err := os.ReadFile(configFile)
		if err != nil {
			return fmt.Errorf("failed to read plugin config: %w", err)
		}
		
		var configs []PluginConfig
		if err := json.Unmarshal(data, &configs); err != nil {
			return fmt.Errorf("failed to parse plugin config: %w", err)
		}
		
		m.mu.Lock()
		defer m.mu.Unlock()
		
		for _, cfg := range configs {
			m.enabled[cfg.Name] = cfg.Enabled
			m.configs[cfg.Name] = cfg.Config
		}
	}
	
	return nil
}

// PluginConfig represents plugin configuration stored in plugins.json
type PluginConfig struct {
	Name    string                 `json:"name"`
	Enabled bool                   `json:"enabled"`
	Config  map[string]interface{} `json:"config"`
}

// Register registers a plugin
func (m *Manager) Register(plugin Plugin) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	name := plugin.Name()
	
	if _, exists := m.plugins[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}
	
	// Load config if exists
	config := m.configs[name]
	if config == nil {
		config = make(map[string]interface{})
	}
	
	// Initialize plugin
	if err := plugin.Init(config); err != nil {
		return fmt.Errorf("failed to initialize plugin %s: %w", name, err)
	}
	
	m.plugins[name] = plugin
	
	// Start if enabled
	if m.enabled[name] {
		if err := plugin.Start(); err != nil {
			return fmt.Errorf("failed to start plugin %s: %w", name, err)
		}
	}
	
	return nil
}

// Unregister unregisters a plugin
func (m *Manager) Unregister(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	plugin, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}
	
	// Stop plugin if running
	if m.enabled[name] {
		if err := plugin.Stop(); err != nil {
			return fmt.Errorf("failed to stop plugin %s: %w", name, err)
		}
	}
	
	delete(m.plugins, name)
	delete(m.enabled, name)
	delete(m.configs, name)
	
	return nil
}

// Enable enables a plugin
func (m *Manager) Enable(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	plugin, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}
	
	if m.enabled[name] {
		return nil // Already enabled
	}
	
	if err := plugin.Start(); err != nil {
		return fmt.Errorf("failed to start plugin %s: %w", name, err)
	}
	
	m.enabled[name] = true
	
	return m.saveConfig()
}

// Disable disables a plugin
func (m *Manager) Disable(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	plugin, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}
	
	if !m.enabled[name] {
		return nil // Already disabled
	}
	
	if err := plugin.Stop(); err != nil {
		return fmt.Errorf("failed to stop plugin %s: %w", name, err)
	}
	
	m.enabled[name] = false
	
	return m.saveConfig()
}

// UpdateConfig updates plugin configuration
func (m *Manager) UpdateConfig(name string, config map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	plugin, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}
	
	// Reinitialize plugin with new config
	if err := plugin.Init(config); err != nil {
		return fmt.Errorf("failed to reinitialize plugin %s: %w", name, err)
	}
	
	m.configs[name] = config
	
	return m.saveConfig()
}

// GetPlugin returns a plugin by name
func (m *Manager) GetPlugin(name string) (Plugin, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	plugin, exists := m.plugins[name]
	return plugin, exists
}

// ListPlugins returns all registered plugins
func (m *Manager) ListPlugins() []PluginInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	plugins := make([]PluginInfo, 0, len(m.plugins))
	for name, plugin := range m.plugins {
		plugins = append(plugins, PluginInfo{
			Name:        name,
			Version:     plugin.Version(),
			Description: plugin.Description(),
			Enabled:     m.enabled[name],
			Config:      m.configs[name],
		})
	}
	
	return plugins
}

// IsEnabled returns whether a plugin is enabled
func (m *Manager) IsEnabled(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.enabled[name]
}

// ExecuteHooks executes hooks for a specific hook point
func (m *Manager) ExecuteHooks(hook HookPoint, context *HookContext) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	context.Hook = hook
	
	for name, plugin := range m.plugins {
		if !m.enabled[name] {
			continue
		}
		
		if hookable, ok := plugin.(Hookable); ok {
			handlers := hookable.GetHooks()
			if handler, exists := handlers[hook]; exists {
				if err := handler(context); err != nil {
					return fmt.Errorf("plugin %s hook %s failed: %w", name, hook, err)
				}
			}
		}
	}
	
	return nil
}

// saveConfig saves plugin configuration to disk
func (m *Manager) saveConfig() error {
	pluginsDir := filepath.Join(m.homeDir, "plugins")
	configFile := filepath.Join(pluginsDir, "plugins.json")
	
	configs := make([]PluginConfig, 0, len(m.plugins))
	for name := range m.plugins {
		configs = append(configs, PluginConfig{
			Name:    name,
			Enabled: m.enabled[name],
			Config:  m.configs[name],
		})
	}
	
	data, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal plugin config: %w", err)
	}
	
	if err := os.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write plugin config: %w", err)
	}
	
	return nil
}

// StopAll stops all enabled plugins
func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	for name, plugin := range m.plugins {
		if m.enabled[name] {
			plugin.Stop()
		}
	}
}
