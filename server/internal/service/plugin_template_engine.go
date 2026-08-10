package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"
	"time"

	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

// TemplateEngine manages plugin templates and instantiation
type TemplateEngine struct {
	db        *gorm.DB
	templates map[string]*model.PluginTemplate
}

// NewTemplateEngine creates a new template engine
func NewTemplateEngine(db *gorm.DB) *TemplateEngine {
	return &TemplateEngine{
		db:        db,
		templates: make(map[string]*model.PluginTemplate),
	}
}

// LoadBuiltinTemplates loads all builtin templates
func (e *TemplateEngine) LoadBuiltinTemplates() error {
	builtinTemplates := getBuiltinTemplates()

	for i := range builtinTemplates {
		tmpl := &builtinTemplates[i]
		e.templates[tmpl.ID] = tmpl

		// Save to database if not exists
		var count int64
		e.db.Model(&model.PluginTemplate{}).Where("id = ?", tmpl.ID).Count(&count)
		if count == 0 {
			if err := e.db.Create(tmpl).Error; err != nil {
				return fmt.Errorf("failed to save template %s: %w", tmpl.ID, err)
			}
		}
	}

	return nil
}

// GetTemplate returns a template by ID
func (e *TemplateEngine) GetTemplate(templateID string) (*model.PluginTemplate, error) {
	if tmpl, ok := e.templates[templateID]; ok {
		return tmpl, nil
	}

	var tmpl model.PluginTemplate
	if err := e.db.Where("template_id = ?", templateID).First(&tmpl).Error; err != nil {
		return nil, fmt.Errorf("template not found: %s", templateID)
	}

	return &tmpl, nil
}

// ListTemplates returns all available templates
func (e *TemplateEngine) ListTemplates() ([]*model.PluginTemplate, error) {
	var templates []*model.PluginTemplate
	if err := e.db.Find(&templates).Error; err != nil {
		return nil, err
	}
	return templates, nil
}

// Instantiate creates a plugin instance from a template
func (e *TemplateEngine) Instantiate(templateID string, userConfig map[string]interface{}, name string) (*model.Plugin, error) {
	tmpl, err := e.GetTemplate(templateID)
	if err != nil {
		return nil, err
	}

	// Validate config
	if err := e.validateConfig(tmpl, userConfig); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Check name uniqueness
	var count int64
	e.db.Model(&model.Plugin{}).Where("name = ?", name).Count(&count)
	if count > 0 {
		return nil, fmt.Errorf("plugin name %q already exists", name)
	}

	// Render triggers with user config
	triggers, err := e.renderTriggers(tmpl.Triggers, userConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to render triggers: %w", err)
	}

	// Create plugin instance
	plugin := &model.Plugin{
		Name:        name,
		Version:     tmpl.Version,
		Description: tmpl.Description,
		Author:      tmpl.Author,
		Enabled:     true,
		Config:      mustJSON(userConfig),
		Type:        "template",
		TemplateID:  templateID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save plugin
	if err := e.db.Create(plugin).Error; err != nil {
		return nil, fmt.Errorf("failed to create plugin: %w", err)
	}

	// Save triggers
	for _, trigger := range triggers {
		trigger.PluginID = plugin.ID
		if err := e.db.Create(&trigger).Error; err != nil {
			return nil, fmt.Errorf("failed to create trigger: %w", err)
		}
	}

	return plugin, nil
}

// validateConfig validates user config against template schema
func (e *TemplateEngine) validateConfig(tmpl *model.PluginTemplate, config map[string]interface{}) error {
	for name, field := range tmpl.ConfigSchema {
		value, exists := config[name]

		// Check required fields
		if field.Required && (!exists || value == nil || value == "") {
			return fmt.Errorf("field %s is required", name)
		}

		// Check enum values
		if len(field.Options) > 0 && exists && value != nil {
			valid := false
			for _, opt := range field.Options {
				if fmt.Sprint(opt.Value) == fmt.Sprint(value) {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("field %s has invalid value: %v", name, value)
			}
		}
	}

	return nil
}

// renderTriggers renders template variables in triggers
func (e *TemplateEngine) renderTriggers(triggers []model.PluginTrigger, config map[string]interface{}) ([]model.PluginTrigger, error) {
	ctx := map[string]interface{}{
		"config": config,
	}

	rendered := make([]model.PluginTrigger, len(triggers))
	for i, trigger := range triggers {
		// Render event
		event, err := e.renderString(trigger.Event, ctx)
		if err != nil {
			return nil, err
		}
		trigger.Event = event

		// Render conditions
		for j, cond := range trigger.Conditions {
			if strVal, ok := cond.Value.(string); ok {
				renderedVal, err := e.renderString(strVal, ctx)
				if err != nil {
					return nil, err
				}
				cond.Value = renderedVal
			}
			trigger.Conditions[j] = cond
		}

		// Render actions
		for j, action := range trigger.Actions {
			// Render URL
			if action.URL != "" {
				url, err := e.renderString(action.URL, ctx)
				if err != nil {
					return nil, err
				}
				action.URL = url
			}

			// Render body
			if action.Body != "" {
				body, err := e.renderString(action.Body, ctx)
				if err != nil {
					return nil, err
				}
				action.Body = body
			}

			// Render params
			for k, v := range action.Params {
				if strVal, ok := v.(string); ok {
					renderedVal, err := e.renderString(strVal, ctx)
					if err != nil {
						return nil, err
					}
					action.Params[k] = renderedVal
				}
			}

			trigger.Actions[j] = action
		}

		rendered[i] = trigger
	}

	return rendered, nil
}

// renderString renders a template string with context
func (e *TemplateEngine) renderString(tmplStr string, ctx map[string]interface{}) (string, error) {
	tmpl, err := template.New("render").Parse(tmplStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// mustJSON converts value to JSON string
func mustJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// getBuiltinTemplates returns all builtin templates
func getBuiltinTemplates() []model.PluginTemplate {
	return []model.PluginTemplate{
		dingtalkTaskNotifyTemplate(),
		dingtalkMRNotifyTemplate(),
		feishuTaskNotifyTemplate(),
		autoLabelerTemplate(),
		webhookGenericTemplate(),
		emailNotifyTemplate(),
	}
}

// dingtalkTaskNotifyTemplate returns DingTalk task notification template
func dingtalkTaskNotifyTemplate() model.PluginTemplate {
	return model.PluginTemplate{
		ID:          "dingtalk-task-notify",
		Name:        "DingTalk Task Notification",
		Description: "Send DingTalk notification when task status changes",
		Version:     "1.0.0",
		Author:      "iForge",
		Category:    "notification",
		Icon:        "dingtalk",
		Tags:        []string{"dingtalk", "notification", "task"},
		ConfigSchema: map[string]model.FieldSchema{
			"webhook_url": {
				Type:        "string",
				Label:       "Webhook URL",
				Description: "DingTalk robot webhook URL",
				Required:    true,
				Placeholder: "https://oapi.dingtalk.com/robot/send?access_token=xxx",
			},
			"secret": {
				Type:        "string",
				Label:       "Secret Key",
				Description: "DingTalk robot secret key (optional)",
				Required:    false,
			},
			"trigger_events": {
				Type:  "multi_select",
				Label: "Trigger Events",
				Options: []model.FieldOption{
					{Value: "task.created", Label: "Task Created"},
					{Value: "task.updated", Label: "Task Updated"},
					{Value: "task.status_changed", Label: "Status Changed"},
					{Value: "task.assigned", Label: "Task Assigned"},
				},
				Default: []string{"task.status_changed"},
			},
			"filter_priority": {
				Type:  "select",
				Label: "Priority Filter",
				Options: []model.FieldOption{
					{Value: "all", Label: "All"},
					{Value: "urgent", Label: "Urgent Only"},
					{Value: "high", Label: "High Priority and Above"},
				},
				Default: "all",
			},
			"message_template": {
				Type:        "textarea",
				Label:       "Message Template",
				Description: "Supports variables: {{.task.title}}, {{.task.status}}, {{.user.name}}",
				Default:     "📋 Task Update\nTitle: {{.task.title}}\nStatus: {{.task.old_status}} → {{.task.new_status}}\nUpdated by: {{.user.name}}",
			},
		},
		Triggers: []model.PluginTrigger{
			{
				Event: "{{.config.trigger_events}}",
				Actions: []model.PluginAction{
					{
						Type: "http_post",
						URL:  "{{.config.webhook_url}}",
						Headers: map[string]string{
							"Content-Type": "application/json",
						},
						Body: `{
							"msgtype": "markdown",
							"markdown": {
								"title": "Task Notification",
								"text": "{{.config.message_template}}"
							}
						}`,
					},
				},
			},
		},
	}
}

// dingtalkMRNotifyTemplate returns DingTalk MR notification template
func dingtalkMRNotifyTemplate() model.PluginTemplate {
	return model.PluginTemplate{
		ID:          "dingtalk-mr-notify",
		Name:        "DingTalk MR Notification",
		Description: "Send DingTalk notification when merge request changes",
		Version:     "1.0.0",
		Author:      "iForge",
		Category:    "notification",
		Icon:        "dingtalk",
		Tags:        []string{"dingtalk", "notification", "merge-request"},
		ConfigSchema: map[string]model.FieldSchema{
			"webhook_url": {
				Type:        "string",
				Label:       "Webhook URL",
				Description: "DingTalk robot webhook URL",
				Required:    true,
			},
			"trigger_events": {
				Type:  "multi_select",
				Label: "Trigger Events",
				Options: []model.FieldOption{
					{Value: "mr.created", Label: "MR Created"},
					{Value: "mr.updated", Label: "MR Updated"},
					{Value: "mr.merged", Label: "MR Merged"},
					{Value: "mr.closed", Label: "MR Closed"},
				},
				Default: []string{"mr.created", "mr.merged"},
			},
		},
		Triggers: []model.PluginTrigger{
			{
				Event: "{{.config.trigger_events}}",
				Actions: []model.PluginAction{
					{
						Type: "http_post",
						URL:  "{{.config.webhook_url}}",
						Body: `{
							"msgtype": "markdown",
							"markdown": {
								"title": "MR Notification",
								"text": "🔀 MR: {{.mr.title}}\nAuthor: {{.user.name}}\nStatus: {{.mr.status}}"
							}
						}`,
					},
				},
			},
		},
	}
}

// feishuTaskNotifyTemplate returns Feishu task notification template
func feishuTaskNotifyTemplate() model.PluginTemplate {
	return model.PluginTemplate{
		ID:          "feishu-task-notify",
		Name:        "Feishu Task Notification",
		Description: "Send Feishu notification when task changes",
		Version:     "1.0.0",
		Author:      "iForge",
		Category:    "notification",
		Icon:        "feishu",
		Tags:        []string{"feishu", "notification", "task"},
		ConfigSchema: map[string]model.FieldSchema{
			"webhook_url": {
				Type:        "string",
				Label:       "Webhook URL",
				Description: "Feishu robot webhook URL",
				Required:    true,
			},
			"trigger_events": {
				Type:  "multi_select",
				Label: "Trigger Events",
				Options: []model.FieldOption{
					{Value: "task.created", Label: "Task Created"},
					{Value: "task.updated", Label: "Task Updated"},
					{Value: "task.status_changed", Label: "Status Changed"},
				},
				Default: []string{"task.status_changed"},
			},
		},
		Triggers: []model.PluginTrigger{
			{
				Event: "{{.config.trigger_events}}",
				Actions: []model.PluginAction{
					{
						Type: "http_post",
						URL:  "{{.config.webhook_url}}",
						Body: `{
							"msg_type": "interactive",
							"card": {
								"elements": [{
									"tag": "markdown",
									"content": "**Task Update**\nTitle: {{.task.title}}\nStatus: {{.task.status}}"
								}]
							}
						}`,
					},
				},
			},
		},
	}
}

// autoLabelerTemplate returns auto labeler template
func autoLabelerTemplate() model.PluginTemplate {
	return model.PluginTemplate{
		ID:          "auto-labeler",
		Name:        "Auto Labeler",
		Description: "Automatically add labels based on title keywords",
		Version:     "1.0.0",
		Author:      "iForge",
		Category:    "automation",
		Icon:        "tag",
		Tags:        []string{"automation", "label", "task"},
		ConfigSchema: map[string]model.FieldSchema{
			"rules": {
				Type:        "textarea",
				Label:       "Labeling Rules",
				Description: "JSON format: [{\"keyword\": \"bug\", \"label\": \"bug\"}]",
				Required:    true,
				Default:     `[{"keyword": "bug", "label": "bug"}, {"keyword": "feature", "label": "enhancement"}]`,
			},
		},
		Triggers: []model.PluginTrigger{
			{
				Event: "task.created",
				Actions: []model.PluginAction{
					{
						Type: "add_label",
						Params: map[string]interface{}{
							"rules": "{{.config.rules}}",
						},
					},
				},
			},
		},
	}
}

// webhookGenericTemplate returns generic webhook template
func webhookGenericTemplate() model.PluginTemplate {
	return model.PluginTemplate{
		ID:          "webhook-generic",
		Name:        "Generic Webhook",
		Description: "Send custom webhook on events",
		Version:     "1.0.0",
		Author:      "iForge",
		Category:    "integration",
		Icon:        "webhook",
		Tags:        []string{"webhook", "integration"},
		ConfigSchema: map[string]model.FieldSchema{
			"webhook_url": {
				Type:        "string",
				Label:       "Webhook URL",
				Description: "Target webhook URL",
				Required:    true,
			},
			"trigger_events": {
				Type:  "multi_select",
				Label: "Trigger Events",
				Options: []model.FieldOption{
					{Value: "task.created", Label: "Task Created"},
					{Value: "task.updated", Label: "Task Updated"},
					{Value: "mr.created", Label: "MR Created"},
					{Value: "mr.merged", Label: "MR Merged"},
				},
				Required: true,
			},
			"custom_headers": {
				Type:        "textarea",
				Label:       "Custom Headers",
				Description: "JSON format: {\"Authorization\": \"Bearer xxx\"}",
				Required:    false,
			},
		},
		Triggers: []model.PluginTrigger{
			{
				Event: "{{.config.trigger_events}}",
				Actions: []model.PluginAction{
					{
						Type: "http_post",
						URL:  "{{.config.webhook_url}}",
						Body: `{{.event_data}}`,
					},
				},
			},
		},
	}
}

// emailNotifyTemplate returns email notification template
func emailNotifyTemplate() model.PluginTemplate {
	return model.PluginTemplate{
		ID:          "email-notify",
		Name:        "Email Notification",
		Description: "Send email notification on events",
		Version:     "1.0.0",
		Author:      "iForge",
		Category:    "notification",
		Icon:        "email",
		Tags:        []string{"email", "notification"},
		ConfigSchema: map[string]model.FieldSchema{
			"recipients": {
				Type:        "string",
				Label:       "Recipients",
				Description: "Comma-separated email addresses",
				Required:    true,
				Placeholder: "user1@example.com,user2@example.com",
			},
			"trigger_events": {
				Type:  "multi_select",
				Label: "Trigger Events",
				Options: []model.FieldOption{
					{Value: "task.created", Label: "Task Created"},
					{Value: "task.assigned", Label: "Task Assigned"},
					{Value: "mr.created", Label: "MR Created"},
				},
				Required: true,
			},
			"subject_template": {
				Type:    "string",
				Label:   "Subject Template",
				Default: "[iForge] {{.event_type}}: {{.task.title}}",
			},
			"body_template": {
				Type:        "textarea",
				Label:       "Body Template",
				Description: "Supports HTML",
				Default:     "<h2>Task Update</h2><p>Title: {{.task.title}}</p><p>Status: {{.task.status}}</p>",
			},
		},
		Triggers: []model.PluginTrigger{
			{
				Event: "{{.config.trigger_events}}",
				Actions: []model.PluginAction{
					{
						Type: "send_email",
						Params: map[string]interface{}{
							"to":      "{{.config.recipients}}",
							"subject": "{{.config.subject_template}}",
							"body":    "{{.config.body_template}}",
						},
					},
				},
			},
		},
	}
}
