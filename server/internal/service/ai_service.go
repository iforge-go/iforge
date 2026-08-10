package service

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"iforge/iforge/internal/model"
)

var (
	ErrAIDisabled      = errors.New("AI is not enabled; configure it in admin settings")
	ErrAIMisconfigured = errors.New("AI is enabled but baseUrl/apiKey/model are missing")
	ErrAIEmptyResponse = errors.New("AI returned no tasks")
)

// aiSystemMessage is the system prompt sent to LLM, defining its role and output constraints
const aiSystemMessage = "You are a senior product manager who decomposes user stories into actionable engineering tasks. Always respond with valid JSON only, no markdown, no commentary."

// aiOptimizeSystemMessage is the system prompt for AI user story optimization
const aiOptimizeSystemMessage = "You are a senior product manager who helps refine user stories. You improve clarity, completeness, and actionability while preserving the author's original intent. Always respond with valid JSON only, no markdown, no commentary."

// aiOptimizeTaskSystemMessage is the system prompt for AI task draft optimization
const aiOptimizeTaskSystemMessage = "You are a senior engineer who helps refine engineering tasks. You improve clarity, completeness, and actionability while preserving the author's original intent. Always respond with valid JSON only, no markdown, no commentary."

// aiOptimizeSprintSystemMessage is the system prompt for AI sprint draft optimization
const aiOptimizeSprintSystemMessage = "You are an agile coach who helps refine sprint plans. You improve clarity of the sprint title, enrich the description, and make the goal specific and measurable while preserving the author's original intent. Always respond with valid JSON only, no markdown, no commentary."

// AIService handles AI-powered features (user story decomposition, etc.)
// Calls OpenAI-compatible Chat Completions API (supports DeepSeek/Qwen/Zhipu/Moonshot/OpenAI).
type AIService struct {
	aiModelConfigService *AIModelConfigService
}

// NewAIService creates a new AIService
func NewAIService(aiModelConfigService *AIModelConfigService) *AIService {
	return &AIService{aiModelConfigService: aiModelConfigService}
}

// resolveSettings loads the default AI configuration and validates required fields.
// Shared by all AI invocation methods, replacing the scattered GetAISettings + validation logic.
func (s *AIService) resolveSettings() (*model.AIModelConfig, error) {
	cfg, err := s.aiModelConfigService.GetDefault()
	if err != nil {
		if err == ErrAIModelConfigNoDefault {
			return nil, ErrAIDisabled
		}
		return nil, fmt.Errorf("failed to load default AI config: %w", err)
	}
	if !cfg.Enabled {
		return nil, ErrAIDisabled
	}
	if cfg.BaseURL == "" || cfg.APIKey == "" || cfg.Model == "" {
		return nil, ErrAIMisconfigured
	}
	return cfg, nil
}

// AITaskSuggestion is a single AI-suggested task (pre-creation; no IDs)
type AITaskSuggestion struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"` // low/medium/high/urgent
	TaskType    string `json:"taskType"` // task/feature/bug/improvement
	StoryPoints *int   `json:"storyPoints"`
}

// AIDecomposeResult is the response payload for AI decomposition
type AIDecomposeResult struct {
	Tasks []AITaskSuggestion `json:"tasks"`
	Model string             `json:"model"`
	Usage map[string]int     `json:"usage,omitempty"`
}

// AIOptimizeResult is the response payload for AI story optimization
type AIOptimizeResult struct {
	Title              string         `json:"title"`
	Description        string         `json:"description"`
	AcceptanceCriteria string         `json:"acceptanceCriteria"`
	Priority           string         `json:"priority"`        // low/medium/high/urgent
	StoryPoints        int            `json:"storyPoints"`     // Fibonacci 1,2,3,5,8,13
	Summary            string         `json:"summary"`         // Optimization notes: AI's brief explanation of changes
	Usage              map[string]int `json:"usage,omitempty"` // Token usage (prompt_tokens/completion_tokens/total_tokens)
}

// AIOptimizeTaskResult is the response payload for AI task optimization.
// Unlike AIOptimizeResult: tasks have taskType but no acceptanceCriteria.
type AIOptimizeTaskResult struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Priority    string         `json:"priority"`        // low/medium/high/urgent
	TaskType    string         `json:"taskType"`        // task/feature/bug/improvement
	StoryPoints int            `json:"storyPoints"`     // Fibonacci 1,2,3,5,8,13
	Summary     string         `json:"summary"`         // Optimization notes: AI's brief explanation of changes
	Usage       map[string]int `json:"usage,omitempty"` // Token usage (prompt_tokens/completion_tokens/total_tokens)
}

// AIOptimizeSprintResult is the response payload for AI sprint optimization.
// Sprints only have title/description/goal, no priority/storyPoints/taskType/acceptanceCriteria.
type AIOptimizeSprintResult struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Goal        string         `json:"goal"`            // Sprint goal (specific, measurable)
	Summary     string         `json:"summary"`         // Optimization notes: AI's brief explanation of changes
	Usage       map[string]int `json:"usage,omitempty"` // Token usage (prompt_tokens/completion_tokens/total_tokens)
}

// DecomposeUserStory calls the LLM to break a user story into 3-8 suggested tasks.
// language is a locale code (e.g. "zh", "en") that controls the output language of
// task titles and descriptions; JSON keys and enum values stay in English.
func (s *AIService) DecomposeUserStory(story *model.UserStory, language string) (*AIDecomposeResult, error) {
	settings, err := s.resolveSettings()
	if err != nil {
		return nil, err
	}

	prompt := buildDecomposePrompt(story, language)
	body, err := s.callChatCompletions(settings, prompt)
	if err != nil {
		return nil, err
	}

	tasks, err := parseTasksFromChatResponse(body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}
	if len(tasks) < 1 {
		return nil, ErrAIEmptyResponse
	}
	if len(tasks) > 8 {
		tasks = tasks[:8]
	}

	return &AIDecomposeResult{
		Tasks: tasks,
		Model: settings.Model,
		Usage: extractUsage(body),
	}, nil
}

// DecomposeUserStoryStream is like DecomposeUserStory but streams the process to the client.
// Three callbacks are provided for real-time display:
//   - onPrompt:   called once before the LLM request, with the full prompt (system + user) for transparency
//   - onThinking: called for each reasoning_content delta (DeepSeek o1-style thinking; empty for models that don't support it)
//   - onDelta:    called for each content delta (the actual JSON response being generated)
//
// After the stream completes, the accumulated content is parsed into structured tasks.
func (s *AIService) DecomposeUserStoryStream(
	story *model.UserStory,
	language string,
	onPrompt func(string),
	onThinking func(string),
	onDelta func(string),
) (*AIDecomposeResult, error) {
	settings, err := s.resolveSettings()
	if err != nil {
		return nil, err
	}

	prompt := buildDecomposePrompt(story, language)

	// Push the full prompt to the frontend for display before making the request
	if onPrompt != nil {
		displayPrompt := fmt.Sprintf("[System]\n%s\n\n[User]\n%s", aiSystemMessage, prompt)
		onPrompt(displayPrompt)
	}

	url := strings.TrimRight(settings.BaseURL, "/") + "/chat/completions"

	payload := map[string]interface{}{
		"model": settings.Model,
		"messages": []map[string]string{
			{"role": "system", "content": aiSystemMessage},
			{"role": "user", "content": prompt},
		},
		"temperature":     0.4,
		"max_tokens":      streamingMaxTokens(settings.MaxTokens),
		"response_format": map[string]string{"type": "json_object"},
		"stream":          true,
		"stream_options":  map[string]bool{"include_usage": true}, // include usage stats in the last chunk
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+settings.APIKey)
	req.Header.Set("Accept", "text/event-stream")

	// Streaming requires a longer timeout
	timeout := time.Duration(settings.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("LLM API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("LLM API returned status %d: %s", resp.StatusCode, string(b))
	}

	// Read the LLM SSE stream line by line, extracting reasoning_content and content
	var fullContent strings.Builder
	var usage map[string]int
	hasReasoning := false
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // allow larger lines
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		// Parse both content (final output) and reasoning_content (thinking process, supported by DeepSeek etc.)
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content          string `json:"content"`
					ReasoningContent string `json:"reasoning_content"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue // skip unparseable lines (e.g., comments, heartbeats)
		}
		// Capture token usage (last chunk includes usage when stream_options.include_usage is true)
		if chunk.Usage != nil {
			usage = map[string]int{
				"prompt_tokens":     chunk.Usage.PromptTokens,
				"completion_tokens": chunk.Usage.CompletionTokens,
				"total_tokens":      chunk.Usage.TotalTokens,
			}
		}
		if len(chunk.Choices) > 0 {
			// Reasoning process (reasoning_content): returned by DeepSeek-R1 and similar models
			if reasoning := chunk.Choices[0].Delta.ReasoningContent; reasoning != "" {
				hasReasoning = true
				if onThinking != nil {
					onThinking(reasoning)
				}
			}
			// Final output (content): the actual JSON response
			if delta := chunk.Choices[0].Delta.Content; delta != "" {
				fullContent.WriteString(delta)
				if onDelta != nil {
					onDelta(delta)
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading LLM stream: %w", err)
	}

	// Parse the accumulated content into structured task list
	content := fullContent.String()
	tasks, err := parseTasksFromRawContent(content)
	if err != nil {
		// If content is empty but hasReasoning is true, the reasoning model used all tokens for thinking
		if content == "" && hasReasoning {
			return nil, fmt.Errorf("AI used all tokens for reasoning and produced no output. Please increase max_tokens in AI settings or simplify the input")
		}
		return nil, err
	}
	if len(tasks) < 1 {
		return nil, ErrAIEmptyResponse
	}
	if len(tasks) > 8 {
		tasks = tasks[:8]
	}

	return &AIDecomposeResult{
		Tasks: tasks,
		Model: settings.Model,
		Usage: usage,
	}, nil
}

// OptimizeUserStoryStream calls the LLM to optimize a user story draft and streams the process.
// Three callbacks (onPrompt/onThinking/onDelta) enable real-time SSE display, mirroring DecomposeUserStoryStream.
// After the stream completes, the accumulated content is parsed into an AIOptimizeResult.
func (s *AIService) OptimizeUserStoryStream(
	title, description, acceptanceCriteria, language string,
	onPrompt func(string),
	onThinking func(string),
	onDelta func(string),
) (*AIOptimizeResult, error) {
	settings, err := s.resolveSettings()
	if err != nil {
		return nil, err
	}

	prompt := buildOptimizePrompt(title, description, acceptanceCriteria, language)

	// Push the full prompt to the frontend for display before making the request
	if onPrompt != nil {
		displayPrompt := fmt.Sprintf("[System]\n%s\n\n[User]\n%s", aiOptimizeSystemMessage, prompt)
		onPrompt(displayPrompt)
	}

	url := strings.TrimRight(settings.BaseURL, "/") + "/chat/completions"

	payload := map[string]interface{}{
		"model": settings.Model,
		"messages": []map[string]string{
			{"role": "system", "content": aiOptimizeSystemMessage},
			{"role": "user", "content": prompt},
		},
		"temperature":     0.5,
		"max_tokens":      streamingMaxTokens(settings.MaxTokens),
		"response_format": map[string]string{"type": "json_object"},
		"stream":          true,
		"stream_options":  map[string]bool{"include_usage": true}, // include usage stats in the last chunk
	}

	// For reasoning models (e.g., DeepSeek-R1), reasoning_content consumes many tokens,
	// which may cause the final content to be empty. Track reasoning output for error diagnosis.
	hasReasoning := false

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+settings.APIKey)
	req.Header.Set("Accept", "text/event-stream")

	timeout := time.Duration(settings.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("LLM API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("LLM API returned status %d: %s", resp.StatusCode, string(b))
	}

	// Read the LLM SSE stream line by line
	var fullContent strings.Builder
	var usage map[string]int
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content          string `json:"content"`
					ReasoningContent string `json:"reasoning_content"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		// Capture token usage (last chunk includes usage when stream_options.include_usage is true)
		if chunk.Usage != nil {
			usage = map[string]int{
				"prompt_tokens":     chunk.Usage.PromptTokens,
				"completion_tokens": chunk.Usage.CompletionTokens,
				"total_tokens":      chunk.Usage.TotalTokens,
			}
		}
		if len(chunk.Choices) > 0 {
			if reasoning := chunk.Choices[0].Delta.ReasoningContent; reasoning != "" {
				hasReasoning = true
				if onThinking != nil {
					onThinking(reasoning)
				}
			}
			if delta := chunk.Choices[0].Delta.Content; delta != "" {
				fullContent.WriteString(delta)
				if onDelta != nil {
					onDelta(delta)
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading LLM stream: %w", err)
	}

	// Parse accumulated content into AIOptimizeResult
	content := fullContent.String()
	result, err := parseOptimizeResult(content)
	if err != nil {
		// If content is empty but hasReasoning is true, the reasoning model used all tokens for thinking
		if content == "" && hasReasoning {
			return nil, fmt.Errorf("AI used all tokens for reasoning and produced no output. Please increase max_tokens in AI settings or simplify the input")
		}
		return nil, err
	}
	result.Usage = usage
	return result, nil
}

// OptimizeTaskStream calls the LLM to optimize a task draft and streams the process.
// Mirrors OptimizeUserStoryStream but returns task-specific fields (taskType instead of acceptanceCriteria).
// Three callbacks (onPrompt/onThinking/onDelta) enable real-time SSE display.
func (s *AIService) OptimizeTaskStream(
	title, description, language string,
	onPrompt func(string),
	onThinking func(string),
	onDelta func(string),
) (*AIOptimizeTaskResult, error) {
	settings, err := s.resolveSettings()
	if err != nil {
		return nil, err
	}

	prompt := buildOptimizeTaskPrompt(title, description, language)

	// Push the full prompt to the frontend for display before making the request
	if onPrompt != nil {
		displayPrompt := fmt.Sprintf("[System]\n%s\n\n[User]\n%s", aiOptimizeTaskSystemMessage, prompt)
		onPrompt(displayPrompt)
	}

	url := strings.TrimRight(settings.BaseURL, "/") + "/chat/completions"

	payload := map[string]interface{}{
		"model": settings.Model,
		"messages": []map[string]string{
			{"role": "system", "content": aiOptimizeTaskSystemMessage},
			{"role": "user", "content": prompt},
		},
		"temperature":     0.5,
		"max_tokens":      streamingMaxTokens(settings.MaxTokens),
		"response_format": map[string]string{"type": "json_object"},
		"stream":          true,
		"stream_options":  map[string]bool{"include_usage": true}, // include usage stats in the last chunk
	}

	// For reasoning models (e.g., DeepSeek-R1), reasoning_content consumes many tokens
	hasReasoning := false

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+settings.APIKey)
	req.Header.Set("Accept", "text/event-stream")

	timeout := time.Duration(settings.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("LLM API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("LLM API returned status %d: %s", resp.StatusCode, string(b))
	}

	// Read the LLM SSE stream line by line
	var fullContent strings.Builder
	var usage map[string]int
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content          string `json:"content"`
					ReasoningContent string `json:"reasoning_content"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		// Capture token usage (last chunk includes usage when stream_options.include_usage is true)
		if chunk.Usage != nil {
			usage = map[string]int{
				"prompt_tokens":     chunk.Usage.PromptTokens,
				"completion_tokens": chunk.Usage.CompletionTokens,
				"total_tokens":      chunk.Usage.TotalTokens,
			}
		}
		if len(chunk.Choices) > 0 {
			if reasoning := chunk.Choices[0].Delta.ReasoningContent; reasoning != "" {
				hasReasoning = true
				if onThinking != nil {
					onThinking(reasoning)
				}
			}
			if delta := chunk.Choices[0].Delta.Content; delta != "" {
				fullContent.WriteString(delta)
				if onDelta != nil {
					onDelta(delta)
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading LLM stream: %w", err)
	}

	// Parse accumulated content into AIOptimizeTaskResult
	content := fullContent.String()
	result, err := parseOptimizeTaskResult(content)
	if err != nil {
		if content == "" && hasReasoning {
			return nil, fmt.Errorf("AI used all tokens for reasoning and produced no output. Please increase max_tokens in AI settings or simplify the input")
		}
		return nil, err
	}
	result.Usage = usage
	return result, nil
}

// OptimizeSprintStream mirrors OptimizeTaskStream but targets sprint structure (has goal, no priority/storyPoints/taskType).
// Three callbacks (onPrompt/onThinking/onDelta) enable real-time SSE display.
func (s *AIService) OptimizeSprintStream(
	title, description, goal, language string,
	onPrompt func(string),
	onThinking func(string),
	onDelta func(string),
) (*AIOptimizeSprintResult, error) {
	settings, err := s.resolveSettings()
	if err != nil {
		return nil, err
	}

	prompt := buildOptimizeSprintPrompt(title, description, goal, language)

	// Push the full prompt to the frontend for display before making the request
	if onPrompt != nil {
		displayPrompt := fmt.Sprintf("[System]\n%s\n\n[User]\n%s", aiOptimizeSprintSystemMessage, prompt)
		onPrompt(displayPrompt)
	}

	url := strings.TrimRight(settings.BaseURL, "/") + "/chat/completions"

	payload := map[string]interface{}{
		"model": settings.Model,
		"messages": []map[string]string{
			{"role": "system", "content": aiOptimizeSprintSystemMessage},
			{"role": "user", "content": prompt},
		},
		"temperature":     0.5,
		"max_tokens":      streamingMaxTokens(settings.MaxTokens),
		"response_format": map[string]string{"type": "json_object"},
		"stream":          true,
		"stream_options":  map[string]bool{"include_usage": true}, // include usage stats in the last chunk
	}

	// For reasoning models (e.g., DeepSeek-R1), reasoning_content consumes many tokens
	hasReasoning := false

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+settings.APIKey)
	req.Header.Set("Accept", "text/event-stream")

	timeout := time.Duration(settings.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("LLM API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("LLM API returned status %d: %s", resp.StatusCode, string(b))
	}

	// Read the LLM SSE stream line by line
	var fullContent strings.Builder
	var usage map[string]int
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content          string `json:"content"`
					ReasoningContent string `json:"reasoning_content"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		// Capture token usage (last chunk includes usage when stream_options.include_usage is true)
		if chunk.Usage != nil {
			usage = map[string]int{
				"prompt_tokens":     chunk.Usage.PromptTokens,
				"completion_tokens": chunk.Usage.CompletionTokens,
				"total_tokens":      chunk.Usage.TotalTokens,
			}
		}
		if len(chunk.Choices) > 0 {
			if reasoning := chunk.Choices[0].Delta.ReasoningContent; reasoning != "" {
				hasReasoning = true
				if onThinking != nil {
					onThinking(reasoning)
				}
			}
			if delta := chunk.Choices[0].Delta.Content; delta != "" {
				fullContent.WriteString(delta)
				if onDelta != nil {
					onDelta(delta)
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading LLM stream: %w", err)
	}

	// Parse accumulated content into AIOptimizeSprintResult
	content := fullContent.String()
	result, err := parseOptimizeSprintResult(content)
	if err != nil {
		if content == "" && hasReasoning {
			return nil, fmt.Errorf("AI used all tokens for reasoning and produced no output. Please increase max_tokens in AI settings or simplify the input")
		}
		return nil, err
	}
	result.Usage = usage
	return result, nil
}

// TestConnection sends a minimal ping to verify the default LLM API config
func (s *AIService) TestConnection() error {
	cfg, err := s.resolveSettings()
	if err != nil {
		return err
	}
	return s.TestConnectionWithConfig(cfg)
}

// TestConnectionWithConfig tests connectivity of any AI config (used by the Test button in edit forms).
// Does not rely on the database default config; sends a ping request directly using the provided cfg.
func (s *AIService) TestConnectionWithConfig(cfg *model.AIModelConfig) error {
	if cfg.BaseURL == "" || cfg.APIKey == "" || cfg.Model == "" {
		return ErrAIMisconfigured
	}
	payload := map[string]interface{}{
		"model": cfg.Model,
		"messages": []map[string]string{
			{"role": "user", "content": "ping"},
		},
		"max_tokens": 5,
	}
	payloadBytes, _ := json.Marshal(payload)
	url := strings.TrimRight(cfg.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("LLM API returned status %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

// callChatCompletions POSTs to {baseURL}/chat/completions and returns the raw response body
func (s *AIService) callChatCompletions(settings *model.AIModelConfig, userPrompt string) (map[string]interface{}, error) {
	url := strings.TrimRight(settings.BaseURL, "/") + "/chat/completions"

	payload := map[string]interface{}{
		"model": settings.Model,
		"messages": []map[string]string{
			{"role": "system", "content": aiSystemMessage},
			{"role": "user", "content": userPrompt},
		},
		"temperature":     0.4,
		"max_tokens":      settings.MaxTokens,
		"response_format": map[string]string{"type": "json_object"},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+settings.APIKey)

	timeout := time.Duration(settings.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("LLM API request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("LLM API returned status %d: %s", resp.StatusCode, string(raw))
	}

	var body map[string]interface{}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, fmt.Errorf("invalid JSON from LLM API: %w", err)
	}
	return body, nil
}

// buildDecomposePrompt builds the user prompt with story title/description/acceptance criteria.
// language is a locale code (e.g. "zh", "en") controlling the output language of human-readable text.
func buildDecomposePrompt(story *model.UserStory, language string) string {
	langName := languageName(language)
	var sb strings.Builder
	sb.WriteString("Decompose the following user story into 3 to 8 engineering tasks.\n\n")
	sb.WriteString("User Story Title:\n")
	sb.WriteString(story.Title)
	sb.WriteString("\n\n")
	if story.Description != nil && *story.Description != "" {
		sb.WriteString("Description:\n")
		sb.WriteString(*story.Description)
		sb.WriteString("\n\n")
	}
	if story.AcceptanceCriteria != nil && *story.AcceptanceCriteria != "" {
		sb.WriteString("Acceptance Criteria:\n")
		sb.WriteString(*story.AcceptanceCriteria)
		sb.WriteString("\n\n")
	}
	sb.WriteString(`Respond with EXACTLY this JSON structure (no markdown fences, no extra text):
{
  "tasks": [
    {
      "title": "Short imperative task title (max 80 chars)",
      "description": "Detailed description with implementation notes and edge cases.",
      "priority": "low|medium|high|urgent",
      "taskType": "task|feature|bug|improvement",
      "storyPoints": 1
    }
  ]
}

Rules:
- Each task must be independently actionable and testable.
- Cover all acceptance criteria. Map each criterion to at least one task.
- Order tasks by logical execution sequence.
- Use realistic story points (1, 2, 3, 5, 8, 13).
- Keep titles concise; put detail in description.
- IMPORTANT: Write all "title" and "description" content in ` + langName + `. Keep JSON keys and the enum values (priority, taskType) in English.
- Output JSON only.`)
	return sb.String()
}

// languageName maps a locale code to a human-readable language name for the LLM prompt.
func languageName(locale string) string {
	switch strings.ToLower(locale) {
	case "zh", "zh-cn", "zh-hans", "zh-tw", "zh-hant", "chinese":
		return "Simplified Chinese (简体中文)"
	case "en", "en-us", "en-gb", "english":
		return "English"
	default:
		if locale == "" {
			return "English"
		}
		return locale
	}
}

// streamingMaxTokens returns the max_tokens to use for streaming requests.
// Reasoning models (like DeepSeek-R1) consume tokens for reasoning_content,
// so we enforce a minimum of 4096 to ensure there's room for the actual JSON output.
func streamingMaxTokens(configured int) int {
	const minTokens = 4096
	if configured < minTokens {
		return minTokens
	}
	return configured
}

// buildOptimizePrompt builds the user prompt for AI story optimization.
// The AI is asked to improve title/description/acceptanceCriteria and suggest priority/storyPoints.
func buildOptimizePrompt(title, description, acceptanceCriteria, language string) string {
	langName := languageName(language)
	var sb strings.Builder
	sb.WriteString("Optimize the following user story draft. Improve the title to be concise and actionable. Expand the description with relevant detail. Refine acceptance criteria to be specific, measurable, and testable. Assess the story's complexity and suggest an appropriate priority and story points.\n\n")
	sb.WriteString("Current Title:\n")
	sb.WriteString(title)
	sb.WriteString("\n\n")
	if description != "" {
		sb.WriteString("Current Description:\n")
		sb.WriteString(description)
		sb.WriteString("\n\n")
	} else {
		sb.WriteString("Current Description:\n(none)\n\n")
	}
	if acceptanceCriteria != "" {
		sb.WriteString("Current Acceptance Criteria:\n")
		sb.WriteString(acceptanceCriteria)
		sb.WriteString("\n\n")
	} else {
		sb.WriteString("Current Acceptance Criteria:\n(none)\n\n")
	}
	sb.WriteString(`Respond with EXACTLY this JSON structure (no markdown fences, no extra text):
{
  "title": "optimized concise title (max 80 chars, imperative mood)",
  "description": "detailed description with context, user value, and technical hints",
  "acceptanceCriteria": "testable acceptance criteria, one per line",
  "priority": "low|medium|high|urgent",
  "storyPoints": 1,
  "summary": "brief explanation of key changes made (1-2 sentences)"
}

Rules:
- Preserve the author's original intent and domain-specific terminology.
- Title: max 80 chars, imperative mood (e.g., "Implement user login").
- Description: include context, user value, and technical hints.
- Acceptance criteria: one per line, each independently testable.
- Priority: based on business impact, urgency, and complexity.
- Story points: use Fibonacci scale (1, 2, 3, 5, 8, 13) based on complexity.
- IMPORTANT: Write all content (title, description, acceptanceCriteria, summary) in ` + langName + `. Keep JSON keys and the enum values (priority) in English.
- Output JSON only.`)
	return sb.String()
}

// parseOptimizeResult strips markdown fences and parses the JSON content into AIOptimizeResult.
func parseOptimizeResult(content string) (*AIOptimizeResult, error) {
	content = strings.TrimSpace(content)

	// Empty content: reasoning model may have exhausted tokens, or API returned an error
	if content == "" {
		return nil, fmt.Errorf("AI returned empty content (the model may have used all tokens for reasoning; try increasing max_tokens)")
	}

	// Strip markdown code fences if present (```json ... ``` or ``` ... ```)
	content = stripMarkdownFences(content)

	// Try direct parsing
	var result AIOptimizeResult
	if err := json.Unmarshal([]byte(content), &result); err == nil && result.Title != "" {
		return normalizeOptimizeResult(&result), nil
	}

	// Direct parsing failed, try extracting JSON object from mixed text
	// (model may have added explanatory text before/after the JSON)
	jsonStr := extractJSONObject(content)
	if jsonStr != "" && jsonStr != content {
		if err := json.Unmarshal([]byte(jsonStr), &result); err == nil && result.Title != "" {
			return normalizeOptimizeResult(&result), nil
		}
	}

	// Try repairing truncated JSON (LLM output gets truncated when max_tokens is insufficient)
	repaired := repairTruncatedJSON(content)
	if repaired != content {
		if err := json.Unmarshal([]byte(repaired), &result); err == nil && result.Title != "" {
			return normalizeOptimizeResult(&result), nil
		}
	}

	// All attempts failed, return truncated error message (avoid overly long error messages)
	return nil, fmt.Errorf("could not parse optimize result (length=%d, possibly truncated): %s", len(content), truncateString(content, 500))
}

// normalizeOptimizeResult clamps priority and storyPoints to legal values
func normalizeOptimizeResult(result *AIOptimizeResult) *AIOptimizeResult {
	switch result.Priority {
	case "low", "medium", "high", "urgent":
		// ok
	default:
		result.Priority = "medium"
	}
	if result.StoryPoints < 0 {
		result.StoryPoints = 0
	}
	return result
}

// buildOptimizeTaskPrompt builds the user prompt for AI task optimization.
// Unlike buildOptimizePrompt: tasks have taskType but no acceptanceCriteria.
func buildOptimizeTaskPrompt(title, description, language string) string {
	langName := languageName(language)
	var sb strings.Builder
	sb.WriteString("Optimize the following engineering task draft. Improve the title to be concise and actionable. Expand the description with relevant technical detail, edge cases, and implementation hints. Assess the task's complexity and suggest an appropriate priority, task type, and story points.\n\n")
	sb.WriteString("Current Title:\n")
	sb.WriteString(title)
	sb.WriteString("\n\n")
	if description != "" {
		sb.WriteString("Current Description:\n")
		sb.WriteString(description)
		sb.WriteString("\n\n")
	} else {
		sb.WriteString("Current Description:\n(none)\n\n")
	}
	sb.WriteString(`Respond with EXACTLY this JSON structure (no markdown fences, no extra text):
{
  "title": "optimized concise title (max 80 chars, imperative mood)",
  "description": "detailed description with context, technical hints, and edge cases",
  "priority": "low|medium|high|urgent",
  "taskType": "task|feature|bug|improvement",
  "storyPoints": 1,
  "summary": "brief explanation of key changes made (1-2 sentences)"
}

Rules:
- Preserve the author's original intent and domain-specific terminology.
- Title: max 80 chars, imperative mood (e.g., "Implement user login").
- Description: include context, technical hints, and edge cases.
- Priority: based on business impact, urgency, and complexity.
- Task type: choose the most appropriate type (task/feature/bug/improvement).
- Story points: use Fibonacci scale (1, 2, 3, 5, 8, 13) based on complexity.
- IMPORTANT: Write all content (title, description, summary) in ` + langName + `. Keep JSON keys and the enum values (priority, taskType) in English.
- Output JSON only.`)
	return sb.String()
}

// parseOptimizeTaskResult strips markdown fences and parses the JSON content into AIOptimizeTaskResult.
func parseOptimizeTaskResult(content string) (*AIOptimizeTaskResult, error) {
	content = strings.TrimSpace(content)

	if content == "" {
		return nil, fmt.Errorf("AI returned empty content (the model may have used all tokens for reasoning; try increasing max_tokens)")
	}

	content = stripMarkdownFences(content)

	var result AIOptimizeTaskResult
	if err := json.Unmarshal([]byte(content), &result); err == nil && result.Title != "" {
		return normalizeOptimizeTaskResult(&result), nil
	}

	jsonStr := extractJSONObject(content)
	if jsonStr != "" && jsonStr != content {
		if err := json.Unmarshal([]byte(jsonStr), &result); err == nil && result.Title != "" {
			return normalizeOptimizeTaskResult(&result), nil
		}
	}

	repaired := repairTruncatedJSON(content)
	if repaired != content {
		if err := json.Unmarshal([]byte(repaired), &result); err == nil && result.Title != "" {
			return normalizeOptimizeTaskResult(&result), nil
		}
	}

	return nil, fmt.Errorf("could not parse optimize task result (length=%d, possibly truncated): %s", len(content), truncateString(content, 500))
}

// normalizeOptimizeTaskResult clamps priority and taskType to legal values
func normalizeOptimizeTaskResult(result *AIOptimizeTaskResult) *AIOptimizeTaskResult {
	switch result.Priority {
	case "low", "medium", "high", "urgent":
		// ok
	default:
		result.Priority = "medium"
	}
	switch result.TaskType {
	case "task", "feature", "bug", "improvement":
		// ok
	default:
		result.TaskType = "task"
	}
	if result.StoryPoints < 0 {
		result.StoryPoints = 0
	}
	return result
}

// buildOptimizeSprintPrompt builds the prompt for optimizing sprint drafts.
// Unlike buildOptimizeTaskPrompt: sprints have goal but no priority/taskType/storyPoints.
func buildOptimizeSprintPrompt(title, description, goal, language string) string {
	langName := languageName(language)
	var sb strings.Builder
	sb.WriteString("Optimize the following sprint plan. Improve the title to be concise and descriptive. Expand the description with relevant context, scope, and deliverables. Refine the goal to be specific, measurable, and achievable within a sprint timeframe.\n\n")
	sb.WriteString("Current Title:\n")
	sb.WriteString(title)
	sb.WriteString("\n\n")
	if description != "" {
		sb.WriteString("Current Description:\n")
		sb.WriteString(description)
		sb.WriteString("\n\n")
	} else {
		sb.WriteString("Current Description:\n(none)\n\n")
	}
	if goal != "" {
		sb.WriteString("Current Goal:\n")
		sb.WriteString(goal)
		sb.WriteString("\n\n")
	} else {
		sb.WriteString("Current Goal:\n(none)\n\n")
	}
	sb.WriteString(`Respond with EXACTLY this JSON structure (no markdown fences, no extra text):
{
  "title": "optimized concise sprint title (max 80 chars)",
  "description": "detailed description with scope, deliverables, and context",
  "goal": "specific, measurable sprint goal (e.g., 'Complete user authentication flow with 90% test coverage')",
  "summary": "brief explanation of key changes made (1-2 sentences)"
}

Rules:
- Preserve the author's original intent and domain-specific terminology.
- Title: max 80 chars, concise and descriptive.
- Description: include scope, key deliverables, and relevant context.
- Goal: use SMART criteria (Specific, Measurable, Achievable, Relevant, Time-bound).
- IMPORTANT: Write all content (title, description, goal, summary) in ` + langName + `. Keep JSON keys in English.
- Output JSON only.`)
	return sb.String()
}

// parseOptimizeSprintResult strips markdown fences and parses the JSON content into AIOptimizeSprintResult.
func parseOptimizeSprintResult(content string) (*AIOptimizeSprintResult, error) {
	content = strings.TrimSpace(content)

	if content == "" {
		return nil, fmt.Errorf("AI returned empty content (the model may have used all tokens for reasoning; try increasing max_tokens)")
	}

	content = stripMarkdownFences(content)

	var result AIOptimizeSprintResult
	if err := json.Unmarshal([]byte(content), &result); err == nil && result.Title != "" {
		return &result, nil
	}

	jsonStr := extractJSONObject(content)
	if jsonStr != "" && jsonStr != content {
		if err := json.Unmarshal([]byte(jsonStr), &result); err == nil && result.Title != "" {
			return &result, nil
		}
	}

	repaired := repairTruncatedJSON(content)
	if repaired != content {
		if err := json.Unmarshal([]byte(repaired), &result); err == nil && result.Title != "" {
			return &result, nil
		}
	}

	return nil, fmt.Errorf("could not parse optimize sprint result (length=%d, possibly truncated): %s", len(content), truncateString(content, 500))
}

// stripMarkdownFences removes ```...``` or ```json...``` fences from content
func stripMarkdownFences(content string) string {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "```") {
		return content
	}
	lines := strings.Split(content, "\n")
	if len(lines) < 2 {
		return content
	}
	start := 1
	end := len(lines) - 1
	for end > start && strings.TrimSpace(lines[end]) == "" {
		end--
	}
	if end > start && strings.HasPrefix(strings.TrimSpace(lines[end]), "```") {
		content = strings.Join(lines[start:end], "\n")
	} else {
		content = strings.Join(lines[start:], "\n")
	}
	content = strings.TrimRight(content, "`")
	return strings.TrimSpace(content)
}

// extractJSONObject finds the first balanced {...} block in content.
// Returns "" if no valid JSON object is found.
func extractJSONObject(content string) string {
	start := strings.Index(content, "{")
	if start == -1 {
		return ""
	}
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(content); i++ {
		ch := content[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if ch == '{' {
			depth++
		} else if ch == '}' {
			depth--
			if depth == 0 {
				return content[start : i+1]
			}
		}
	}
	return ""
}

// truncateString truncates s to at most maxLen characters, appending "..." if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "...(truncated)"
}

// repairTruncatedJSON attempts to fix JSON that was cut off due to max_tokens limit.
// Strategy: find the last complete closing brace/bracket, discard everything after it,
// then close any still-open structures (objects/arrays) in the correct order.
// This is useful when an LLM's response is truncated mid-way through a JSON array/object.
func repairTruncatedJSON(content string) string {
	content = strings.TrimSpace(content)

	// Find the last complete closing brace or bracket
	lastClose := -1
	for i := len(content) - 1; i >= 0; i-- {
		ch := content[i]
		if ch == '}' || ch == ']' {
			lastClose = i
			break
		}
	}
	if lastClose == -1 {
		return content // nothing to repair
	}

	// Truncate after the last complete close, discarding incomplete trailing content
	repaired := content[:lastClose+1]

	// Track open/closed structures using a stack (ignoring strings)
	var stack []byte
	inString := false
	escaped := false
	for i := 0; i < len(repaired); i++ {
		ch := repaired[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if ch == '{' || ch == '[' {
			stack = append(stack, ch)
		} else if ch == '}' {
			if len(stack) > 0 && stack[len(stack)-1] == '{' {
				stack = stack[:len(stack)-1]
			}
		} else if ch == ']' {
			if len(stack) > 0 && stack[len(stack)-1] == '[' {
				stack = stack[:len(stack)-1]
			}
		}
	}

	// Close remaining open structures in reverse order (LIFO)
	var suffix strings.Builder
	for i := len(stack) - 1; i >= 0; i-- {
		switch stack[i] {
		case '{':
			suffix.WriteByte('}')
		case '[':
			suffix.WriteByte(']')
		}
	}

	if suffix.Len() == 0 {
		return repaired // already balanced (shouldn't happen if parsing failed, but just in case)
	}
	return repaired + suffix.String()
}

// parseTasksFromChatResponse extracts the "tasks" array from the OpenAI chat completion response.
// Handles markdown code fences and both {"tasks":[]} and bare array forms.
func parseTasksFromChatResponse(body map[string]interface{}) ([]AITaskSuggestion, error) {
	choices, ok := body["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return nil, fmt.Errorf("no choices in LLM response")
	}
	first, ok := choices[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid choice format")
	}
	message, ok := first["message"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no message in choice")
	}
	content, _ := message["content"].(string)
	if content == "" {
		return nil, fmt.Errorf("empty content in LLM response")
	}
	return parseTasksFromRawContent(content)
}

// parseTasksFromRawContent strips markdown fences and parses the JSON content
// into a slice of AITaskSuggestion. Accepts both {"tasks":[...]} and bare array forms.
func parseTasksFromRawContent(content string) ([]AITaskSuggestion, error) {
	// Strip markdown code fences
	content = stripMarkdownFences(content)

	// Try object form {"tasks": [...]} first
	var wrapper struct {
		Tasks []AITaskSuggestion `json:"tasks"`
	}
	if err := json.Unmarshal([]byte(content), &wrapper); err == nil && wrapper.Tasks != nil {
		return normalizeTaskSuggestions(wrapper.Tasks), nil
	}

	// Try bare array form
	var arr []AITaskSuggestion
	if err := json.Unmarshal([]byte(content), &arr); err == nil {
		return normalizeTaskSuggestions(arr), nil
	}

	// Direct parsing failed, try repairing truncated JSON (LLM output gets truncated when max_tokens is insufficient)
	repaired := repairTruncatedJSON(content)
	if repaired != content {
		if err := json.Unmarshal([]byte(repaired), &wrapper); err == nil && wrapper.Tasks != nil {
			return normalizeTaskSuggestions(wrapper.Tasks), nil
		}
		if err := json.Unmarshal([]byte(repaired), &arr); err == nil {
			return normalizeTaskSuggestions(arr), nil
		}
	}

	return nil, fmt.Errorf("could not parse tasks from content (length=%d, possibly truncated): %s", len(content), truncateString(content, 500))
}

// normalizeTaskSuggestions clamps priority/taskType to legal values and fills defaults
func normalizeTaskSuggestions(tasks []AITaskSuggestion) []AITaskSuggestion {
	validPriority := map[string]bool{"low": true, "medium": true, "high": true, "urgent": true}
	validType := map[string]bool{"task": true, "feature": true, "bug": true, "improvement": true, "subtask": true}
	for i := range tasks {
		if !validPriority[tasks[i].Priority] {
			tasks[i].Priority = "medium"
		}
		if !validType[tasks[i].TaskType] {
			tasks[i].TaskType = "task"
		}
		if tasks[i].Title == "" {
			tasks[i].Title = "Untitled task"
		}
	}
	return tasks
}

// extractUsage pulls the usage object from the response (best-effort)
func extractUsage(body map[string]interface{}) map[string]int {
	usage, ok := body["usage"].(map[string]interface{})
	if !ok {
		return nil
	}
	out := make(map[string]int)
	for k, v := range usage {
		if f, ok := v.(float64); ok {
			out[k] = int(f)
		}
	}
	return out
}
