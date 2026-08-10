package model

import "time"

// AIModelConfig represents an AI model configuration (OpenAI-compatible Chat Completions).
// Multiple configs may exist; only the one with IsDefault=true is used by AIService at any time.
type AIModelConfig struct {
	ID         uint      `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Name       string    `gorm:"column:name;size:100;not null;uniqueIndex" json:"name"`
	Provider   string    `gorm:"column:provider;size:32" json:"provider"`
	BaseURL    string    `gorm:"column:base_url;size:512" json:"baseUrl"`
	APIKey     string    `gorm:"column:api_key;size:512" json:"apiKey"`
	Model      string    `gorm:"column:model;size:128" json:"model"`
	Timeout    int       `gorm:"column:timeout" json:"timeout"`
	MaxTokens  int       `gorm:"column:max_tokens" json:"maxTokens"`
	Enabled    bool      `gorm:"column:enabled" json:"enabled"`
	IsDefault  bool      `gorm:"column:is_default;index" json:"isDefault"`
	TestStatus string    `gorm:"column:test_status;size:16" json:"testStatus"`
	TestedAt   *time.Time `gorm:"column:tested_at" json:"testedAt"`
	CreatedAt  time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt  time.Time `gorm:"column:updated_at" json:"updatedAt"`
}

func (AIModelConfig) TableName() string { return "ai_model_config" }
