package service

import (
	"testing"

	"iforge/iforge/internal/model"
)

func TestLanguageName(t *testing.T) {
	tests := []struct {
		locale   string
		expected string
	}{
		{"zh", "Simplified Chinese (简体中文)"},
		{"zh-cn", "Simplified Chinese (简体中文)"},
		{"zh-hans", "Simplified Chinese (简体中文)"},
		{"en", "English"},
		{"en-us", "English"},
		{"en-gb", "English"},
		{"", "English"},
		{"fr", "fr"},
		{"de", "de"},
	}

	for _, tt := range tests {
		result := languageName(tt.locale)
		if result != tt.expected {
			t.Errorf("languageName(%q) = %q; want %q", tt.locale, result, tt.expected)
		}
	}
}

func TestStreamingMaxTokens(t *testing.T) {
	tests := []struct {
		configured int
		expected   int
	}{
		{0, 4096},
		{1000, 4096},
		{4095, 4096},
		{4096, 4096},
		{8192, 8192},
		{16384, 16384},
	}

	for _, tt := range tests {
		result := streamingMaxTokens(tt.configured)
		if result != tt.expected {
			t.Errorf("streamingMaxTokens(%d) = %d; want %d", tt.configured, result, tt.expected)
		}
	}
}

func TestStripMarkdownFences(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no fences",
			input:    "plain text",
			expected: "plain text",
		},
		{
			name:     "with code fences",
			input:    "```\nplain text\n```",
			expected: "plain text",
		},
		{
			name:     "with json code fences",
			input:    "```json\n{\"key\": \"value\"}\n```",
			expected: "{\"key\": \"value\"}",
		},
		{
			name:     "with trailing newlines",
			input:    "```\nplain text\n```\n\n",
			expected: "plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripMarkdownFences(tt.input)
			if result != tt.expected {
				t.Errorf("stripMarkdownFences(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractJSONObject(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple object",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "object with text before",
			input:    `Some text before {"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "object with text after",
			input:    `{"key": "value"} some text after`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "nested object",
			input:    `{"outer": {"inner": "value"}}`,
			expected: `{"outer": {"inner": "value"}}`,
		},
		{
			name:     "no object",
			input:    `no json here`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSONObject(tt.input)
			if result != tt.expected {
				t.Errorf("extractJSONObject(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a long string", 10, "this is a ...(truncated)"},
		{"", 5, ""},
	}

	for _, tt := range tests {
		result := truncateString(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncateString(%q, %d) = %q; want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestRepairTruncatedJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		hasClose bool
	}{
		{
			name:     "complete object",
			input:    `{"key": "value"}`,
			hasClose: true,
		},
		{
			name:     "truncated object",
			input:    `{"key": "value", "arr": [1, 2`,
			hasClose: true,
		},
		{
			name:     "no closing brace",
			input:    `{"key": "value"`,
			hasClose: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := repairTruncatedJSON(tt.input)
			// Just verify it doesn't panic and returns something
			if result == "" && tt.input != "" {
				t.Errorf("repairTruncatedJSON(%q) returned empty string", tt.input)
			}
		})
	}
}

func TestNormalizeTaskSuggestions(t *testing.T) {
	tasks := []AITaskSuggestion{
		{
			Title:       "Task 1",
			Priority:    "invalid",
			TaskType:    "invalid",
			StoryPoints: intPtr(3),
		},
		{
			Title:       "",
			Priority:    "high",
			TaskType:    "feature",
			StoryPoints: intPtr(5),
		},
		{
			Title:       "Task 3",
			Priority:    "low",
			TaskType:    "bug",
			StoryPoints: intPtr(1),
		},
	}

	normalized := normalizeTaskSuggestions(tasks)

	if len(normalized) != 3 {
		t.Fatalf("Expected 3 tasks, got %d", len(normalized))
	}

	// First task should have defaults applied
	if normalized[0].Priority != "medium" {
		t.Errorf("Expected priority 'medium', got %q", normalized[0].Priority)
	}
	if normalized[0].TaskType != "task" {
		t.Errorf("Expected taskType 'task', got %q", normalized[0].TaskType)
	}

	// Second task should have default title
	if normalized[1].Title != "Untitled task" {
		t.Errorf("Expected title 'Untitled task', got %q", normalized[1].Title)
	}

	// Third task should keep valid values
	if normalized[2].Priority != "low" {
		t.Errorf("Expected priority 'low', got %q", normalized[2].Priority)
	}
	if normalized[2].TaskType != "bug" {
		t.Errorf("Expected taskType 'bug', got %q", normalized[2].TaskType)
	}
}

func TestNormalizeOptimizeResult(t *testing.T) {
	tests := []struct {
		name     string
		input    *AIOptimizeResult
		expected *AIOptimizeResult
	}{
		{
			name: "valid values",
			input: &AIOptimizeResult{
				Priority:    "high",
				StoryPoints: 5,
			},
			expected: &AIOptimizeResult{
				Priority:    "high",
				StoryPoints: 5,
			},
		},
		{
			name: "invalid priority",
			input: &AIOptimizeResult{
				Priority:    "invalid",
				StoryPoints: 3,
			},
			expected: &AIOptimizeResult{
				Priority:    "medium",
				StoryPoints: 3,
			},
		},
		{
			name: "negative story points",
			input: &AIOptimizeResult{
				Priority:    "low",
				StoryPoints: -1,
			},
			expected: &AIOptimizeResult{
				Priority:    "low",
				StoryPoints: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeOptimizeResult(tt.input)
			if result.Priority != tt.expected.Priority {
				t.Errorf("Priority = %q; want %q", result.Priority, tt.expected.Priority)
			}
			if result.StoryPoints != tt.expected.StoryPoints {
				t.Errorf("StoryPoints = %d; want %d", result.StoryPoints, tt.expected.StoryPoints)
			}
		})
	}
}

func TestNormalizeOptimizeTaskResult(t *testing.T) {
	tests := []struct {
		name     string
		input    *AIOptimizeTaskResult
		expected *AIOptimizeTaskResult
	}{
		{
			name: "valid values",
			input: &AIOptimizeTaskResult{
				Priority:    "urgent",
				TaskType:    "feature",
				StoryPoints: 8,
			},
			expected: &AIOptimizeTaskResult{
				Priority:    "urgent",
				TaskType:    "feature",
				StoryPoints: 8,
			},
		},
		{
			name: "invalid priority and task type",
			input: &AIOptimizeTaskResult{
				Priority:    "invalid",
				TaskType:    "invalid",
				StoryPoints: 3,
			},
			expected: &AIOptimizeTaskResult{
				Priority:    "medium",
				TaskType:    "task",
				StoryPoints: 3,
			},
		},
		{
			name: "negative story points",
			input: &AIOptimizeTaskResult{
				Priority:    "low",
				TaskType:    "bug",
				StoryPoints: -5,
			},
			expected: &AIOptimizeTaskResult{
				Priority:    "low",
				TaskType:    "bug",
				StoryPoints: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeOptimizeTaskResult(tt.input)
			if result.Priority != tt.expected.Priority {
				t.Errorf("Priority = %q; want %q", result.Priority, tt.expected.Priority)
			}
			if result.TaskType != tt.expected.TaskType {
				t.Errorf("TaskType = %q; want %q", result.TaskType, tt.expected.TaskType)
			}
			if result.StoryPoints != tt.expected.StoryPoints {
				t.Errorf("StoryPoints = %d; want %d", result.StoryPoints, tt.expected.StoryPoints)
			}
		})
	}
}

func TestExtractUsage(t *testing.T) {
	tests := []struct {
		name     string
		body     map[string]interface{}
		expected map[string]int
	}{
		{
			name: "valid usage",
			body: map[string]interface{}{
				"usage": map[string]interface{}{
					"prompt_tokens":     float64(100),
					"completion_tokens": float64(50),
					"total_tokens":      float64(150),
				},
			},
			expected: map[string]int{
				"prompt_tokens":     100,
				"completion_tokens": 50,
				"total_tokens":      150,
			},
		},
		{
			name:     "no usage",
			body:     map[string]interface{}{},
			expected: nil,
		},
		{
			name: "invalid usage type",
			body: map[string]interface{}{
				"usage": "not a map",
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractUsage(tt.body)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %v", result)
				}
				return
			}
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("usage[%q] = %d; want %d", k, result[k], v)
				}
			}
		})
	}
}

func TestParseTasksFromRawContent(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantErr   bool
		taskCount int
	}{
		{
			name: "valid wrapper format",
			content: `{
				"tasks": [
					{"title": "Task 1", "description": "Desc 1", "priority": "high", "taskType": "task", "storyPoints": 3}
				]
			}`,
			wantErr:   false,
			taskCount: 1,
		},
		{
			name: "valid array format",
			content: `[
				{"title": "Task 1", "description": "Desc 1", "priority": "medium", "taskType": "feature", "storyPoints": 5}
			]`,
			wantErr:   false,
			taskCount: 1,
		},
		{
			name:      "invalid json",
			content:   "not json at all",
			wantErr:   true,
			taskCount: 0,
		},
		{
			name:      "empty content",
			content:   "",
			wantErr:   true,
			taskCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks, err := parseTasksFromRawContent(tt.content)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if len(tasks) != tt.taskCount {
				t.Errorf("Expected %d tasks, got %d", tt.taskCount, len(tasks))
			}
		})
	}
}

func TestParseOptimizeResult(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name: "valid result",
			content: `{
				"title": "Optimized Title",
				"description": "Optimized description",
				"acceptanceCriteria": "Criteria",
				"priority": "high",
				"storyPoints": 5,
				"summary": "Summary"
			}`,
			wantErr: false,
		},
		{
			name:    "empty content",
			content: "",
			wantErr: true,
		},
		{
			name:    "invalid json",
			content: "not json",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseOptimizeResult(tt.content)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if result == nil {
				t.Error("Expected result, got nil")
			}
		})
	}
}

func TestParseOptimizeTaskResult(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name: "valid result",
			content: `{
				"title": "Optimized Task",
				"description": "Description",
				"priority": "medium",
				"taskType": "feature",
				"storyPoints": 3,
				"summary": "Summary"
			}`,
			wantErr: false,
		},
		{
			name:    "empty content",
			content: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseOptimizeTaskResult(tt.content)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if result == nil {
				t.Error("Expected result, got nil")
			}
		})
	}
}

func TestParseOptimizeSprintResult(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name: "valid result",
			content: `{
				"title": "Sprint Title",
				"description": "Description",
				"goal": "Sprint goal",
				"summary": "Summary"
			}`,
			wantErr: false,
		},
		{
			name:    "empty content",
			content: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseOptimizeSprintResult(tt.content)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if result == nil {
				t.Error("Expected result, got nil")
			}
		})
	}
}

func TestBuildDecomposePrompt(t *testing.T) {
	story := &model.UserStory{
		Title:              "Test Story",
		Description:        stringPtr("Test description"),
		AcceptanceCriteria: stringPtr("Test criteria"),
	}

	prompt := buildDecomposePrompt(story, "en")

	if prompt == "" {
		t.Error("Expected non-empty prompt")
	}

	// Check that prompt contains key elements
	if !containsString(prompt, "Test Story") {
		t.Error("Prompt should contain story title")
	}
	if !containsString(prompt, "Test description") {
		t.Error("Prompt should contain story description")
	}
	if !containsString(prompt, "Test criteria") {
		t.Error("Prompt should contain acceptance criteria")
	}
	if !containsString(prompt, "English") {
		t.Error("Prompt should contain language name")
	}
}

func TestBuildOptimizePrompt(t *testing.T) {
	prompt := buildOptimizePrompt("Title", "Description", "Criteria", "zh")

	if prompt == "" {
		t.Error("Expected non-empty prompt")
	}

	if !containsString(prompt, "Title") {
		t.Error("Prompt should contain title")
	}
	if !containsString(prompt, "Description") {
		t.Error("Prompt should contain description")
	}
	if !containsString(prompt, "Criteria") {
		t.Error("Prompt should contain criteria")
	}
	if !containsString(prompt, "Simplified Chinese") {
		t.Error("Prompt should contain Chinese language name")
	}
}

func TestBuildOptimizeTaskPrompt(t *testing.T) {
	prompt := buildOptimizeTaskPrompt("Task Title", "Task Description", "en")

	if prompt == "" {
		t.Error("Expected non-empty prompt")
	}

	if !containsString(prompt, "Task Title") {
		t.Error("Prompt should contain task title")
	}
	if !containsString(prompt, "Task Description") {
		t.Error("Prompt should contain task description")
	}
}

func TestBuildOptimizeSprintPrompt(t *testing.T) {
	prompt := buildOptimizeSprintPrompt("Sprint Title", "Description", "Goal", "en")

	if prompt == "" {
		t.Error("Expected non-empty prompt")
	}

	if !containsString(prompt, "Sprint Title") {
		t.Error("Prompt should contain sprint title")
	}
	if !containsString(prompt, "Description") {
		t.Error("Prompt should contain description")
	}
	if !containsString(prompt, "Goal") {
		t.Error("Prompt should contain goal")
	}
}

func TestAIService_TestConnectionWithConfig(t *testing.T) {
	service := &AIService{}

	tests := []struct {
		name    string
		config  *model.AIModelConfig
		wantErr bool
	}{
		{
			name: "missing base url",
			config: &model.AIModelConfig{
				APIKey: "key",
				Model:  "model",
			},
			wantErr: true,
		},
		{
			name: "missing api key",
			config: &model.AIModelConfig{
				BaseURL: "http://example.com",
				Model:   "model",
			},
			wantErr: true,
		},
		{
			name: "missing model",
			config: &model.AIModelConfig{
				BaseURL: "http://example.com",
				APIKey:  "key",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.TestConnectionWithConfig(tt.config)
			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func stringPtr(s string) *string {
	return &s
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
