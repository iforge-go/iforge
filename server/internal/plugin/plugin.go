package plugin

// Plugin defines the interface that all plugins must implement
type Plugin interface {
	// Name returns the unique name of the plugin
	Name() string
	
	// Version returns the plugin version
	Version() string
	
	// Description returns a brief description of the plugin
	Description() string
	
	// Init initializes the plugin
	Init(config map[string]interface{}) error
	
	// Start starts the plugin
	Start() error
	
	// Stop stops the plugin
	Stop() error
	
	// GetConfig returns the plugin's configuration schema
	GetConfig() []ConfigField
}

// ConfigField defines a configuration field for a plugin
type ConfigField struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"` // string, number, boolean, select
	Label       string      `json:"label"`
	Description string      `json:"description"`
	Default     interface{} `json:"default"`
	Required    bool        `json:"required"`
	Options     []string    `json:"options,omitempty"` // for select type
}

// PluginInfo contains metadata about a plugin
type PluginInfo struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Author      string                 `json:"author"`
	Enabled     bool                   `json:"enabled"`
	Config      map[string]interface{} `json:"config"`
}

// HookPoint defines where plugins can hook into the system
type HookPoint string

const (
	HookPreReceive  HookPoint = "pre-receive"
	HookPostReceive HookPoint = "post-receive"
	HookPreMerge    HookPoint = "pre-merge"
	HookPostMerge   HookPoint = "post-merge"
	HookIssueCreate HookPoint = "issue-create"
	HookIssueUpdate HookPoint = "issue-update"
	HookMRCreate    HookPoint = "mr-create"
	HookMRUpdate    HookPoint = "mr-update"
)

// HookHandler is a function that handles a hook event
type HookHandler func(context *HookContext) error

// HookContext provides context for hook execution
type HookContext struct {
	Hook     HookPoint
	User     string
	Repo     string
	Data     map[string]interface{}
	Metadata map[string]string
}

// Hookable is an optional interface for plugins that want to handle hooks
type Hookable interface {
	// GetHooks returns the hooks this plugin wants to handle
	GetHooks() map[HookPoint]HookHandler
}
