package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// Plugin represents a plugin in the system
type Plugin struct {
	ID          int64     `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Name        string    `gorm:"size:100;uniqueIndex;column:name" json:"name"`
	Version     string    `gorm:"column:version" json:"version"`
	Description string    `gorm:"column:description" json:"description"`
	Author      string    `gorm:"column:author" json:"author"`
	URL         string    `gorm:"column:url" json:"url"`
	Endpoint    string    `gorm:"column:endpoint" json:"endpoint"`
	Enabled     bool      `gorm:"column:enabled" json:"enabled"`
	Config      string    `gorm:"column:config" json:"config"`
	Type        string    `gorm:"column:type" json:"type"` // "template" or "custom"
	TemplateID  string    `gorm:"column:template_id" json:"template_id"`
	CreatedAt   time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (Plugin) TableName() string { return "plugin" }

// PluginEvent represents an event that can trigger plugin actions
type PluginEvent struct {
	ID        int64     `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	PluginID  int64     `gorm:"column:plugin_id" json:"plugin_id"`
	EventType string    `gorm:"column:event_type" json:"event_type"`
	Handler   string    `gorm:"column:handler" json:"handler"`
	Enabled   bool      `gorm:"column:enabled" json:"enabled"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

func (PluginEvent) TableName() string { return "plugin_event" }

// PluginHook represents a webhook-like integration point
type PluginHook struct {
	ID        int64     `json:"id"`
	PluginID  int64     `json:"plugin_id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Events    string    `json:"events"` // JSON array of events
	Active    bool      `json:"active"`
	Secret    string    `json:"secret,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// PluginStatus represents the runtime status of a plugin
type PluginStatus struct {
	PluginID    int64     `json:"plugin_id"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Status      string    `json:"status"` // "running", "stopped", "error"
	LastPing    time.Time `json:"last_ping"`
	ErrorCount  int       `json:"error_count"`
	MemoryUsage int64     `json:"memory_usage"`
}

// Event types
const (
	EventTypePreReceive     = "pre-receive"
	EventTypePostReceive    = "post-receive"
	EventTypeIssueCreated   = "issue-created"
	EventTypeIssueUpdated   = "issue-updated"
	EventTypeIssueClosed    = "issue-closed"
	EventTypeMRCreated      = "mr-created"
	EventTypeMRUpdated      = "mr-updated"
	EventTypeMRMerged       = "mr-merged"
	EventTypePush           = "push"
	EventTypeFork           = "fork"
	EventTypeWikiCreated    = "wiki-created"
	EventTypeWikiUpdated    = "wiki-updated"
	EventTypeReleaseCreated = "release-created"
)

// PluginTemplate represents a plugin template
type PluginTemplate struct {
	ID           string             `gorm:"primaryKey;column:id" json:"id"`
	Name         string             `gorm:"column:name" json:"name"`
	Description  string             `gorm:"column:description" json:"description"`
	Version      string             `gorm:"column:version" json:"version"`
	Author       string             `gorm:"column:author" json:"author"`
	Category     string             `gorm:"column:category" json:"category"`
	Icon         string             `gorm:"column:icon" json:"icon"`
	Tags         StringArray        `gorm:"column:tags;type:text" json:"tags"`
	ConfigSchema FieldSchemaMap     `gorm:"column:config_schema;type:text" json:"config_schema"`
	Triggers     PluginTriggerArray `gorm:"column:triggers;type:text" json:"triggers"`
	CreatedAt    time.Time          `gorm:"column:created_at" json:"created_at"`
}

func (PluginTemplate) TableName() string { return "plugin_template" }

// FieldSchema defines the schema for a configuration field
type FieldSchema struct {
	Type        string        `json:"type"` // string, number, boolean, select, multi_select, textarea
	Label       string        `json:"label"`
	Description string        `json:"description"`
	Required    bool          `json:"required"`
	Default     interface{}   `json:"default"`
	Options     []FieldOption `json:"options,omitempty"`
	Placeholder string        `json:"placeholder,omitempty"`
}

// FieldOption represents an option for select/multi_select fields
type FieldOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// PluginTrigger represents a trigger configuration
type PluginTrigger struct {
	ID         int64             `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	PluginID   int64             `gorm:"column:plugin_id" json:"plugin_id"`
	Event      string            `gorm:"column:event" json:"event"`
	Conditions ConditionArray    `gorm:"column:conditions;type:text" json:"conditions"`
	Actions    PluginActionArray `gorm:"column:actions;type:text" json:"actions"`
}

func (PluginTrigger) TableName() string { return "plugin_trigger" }

// Condition represents a condition for filtering events
type Condition struct {
	Field    string      `json:"field"`    // e.g., "task.priority", "task.title"
	Operator string      `json:"operator"` // equals, contains, in, gt, lt, gte, lte, regex
	Value    interface{} `json:"value"`    // comparison value
}

// PluginAction represents an action to execute
type PluginAction struct {
	Type    string                 `json:"type"` // http_post, add_label, send_email, notify_user
	URL     string                 `json:"url,omitempty"`
	Headers map[string]string      `json:"headers,omitempty"`
	Body    string                 `json:"body,omitempty"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

// StringArray is a custom type for storing string arrays in database
type StringArray []string

// Value implements driver.Valuer interface for database storage
func (a StringArray) Value() (driver.Value, error) {
	if a == nil {
		return "[]", nil
	}
	bytes, err := json.Marshal(a)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal StringArray: %w", err)
	}
	return string(bytes), nil
}

// Scan implements sql.Scanner interface for database retrieval
func (a *StringArray) Scan(value interface{}) error {
	if value == nil {
		*a = []string{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return fmt.Errorf("unsupported type for StringArray: %T", value)
	}

	return json.Unmarshal(bytes, a)
}

// FieldSchemaMap is a custom type for storing map[string]FieldSchema in database
type FieldSchemaMap map[string]FieldSchema

// Value implements driver.Valuer interface for database storage
func (m FieldSchemaMap) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	bytes, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal FieldSchemaMap: %w", err)
	}
	return string(bytes), nil
}

// Scan implements sql.Scanner interface for database retrieval
func (m *FieldSchemaMap) Scan(value interface{}) error {
	if value == nil {
		*m = make(map[string]FieldSchema)
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return fmt.Errorf("unsupported type for FieldSchemaMap: %T", value)
	}

	return json.Unmarshal(bytes, m)
}

// ConditionArray is a custom type for storing []Condition in database
type ConditionArray []Condition

// Value implements driver.Valuer interface for database storage
func (a ConditionArray) Value() (driver.Value, error) {
	if a == nil {
		return "[]", nil
	}
	bytes, err := json.Marshal(a)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ConditionArray: %w", err)
	}
	return string(bytes), nil
}

// Scan implements sql.Scanner interface for database retrieval
func (a *ConditionArray) Scan(value interface{}) error {
	if value == nil {
		*a = []Condition{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return fmt.Errorf("unsupported type for ConditionArray: %T", value)
	}

	return json.Unmarshal(bytes, a)
}

// PluginActionArray is a custom type for storing []PluginAction in database
type PluginActionArray []PluginAction

// Value implements driver.Valuer interface for database storage
func (a PluginActionArray) Value() (driver.Value, error) {
	if a == nil {
		return "[]", nil
	}
	bytes, err := json.Marshal(a)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal PluginActionArray: %w", err)
	}
	return string(bytes), nil
}

// Scan implements sql.Scanner interface for database retrieval
func (a *PluginActionArray) Scan(value interface{}) error {
	if value == nil {
		*a = []PluginAction{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return fmt.Errorf("unsupported type for PluginActionArray: %T", value)
	}

	return json.Unmarshal(bytes, a)
}

// PluginTriggerArray is a custom type for storing []PluginTrigger in database
type PluginTriggerArray []PluginTrigger

// Value implements driver.Valuer interface for database storage
func (a PluginTriggerArray) Value() (driver.Value, error) {
	if a == nil {
		return "[]", nil
	}
	bytes, err := json.Marshal(a)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal PluginTriggerArray: %w", err)
	}
	return string(bytes), nil
}

// Scan implements sql.Scanner interface for database retrieval
func (a *PluginTriggerArray) Scan(value interface{}) error {
	if value == nil {
		*a = []PluginTrigger{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return fmt.Errorf("unsupported type for PluginTriggerArray: %T", value)
	}

	return json.Unmarshal(bytes, a)
}
