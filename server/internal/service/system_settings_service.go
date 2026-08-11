package service

import (
	"fmt"
	"sync/atomic"

	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

// SystemSettingsService handles system settings operations.
// Settings are cached in memory (atomic.Pointer) because they rarely change
// and are read on nearly every request (e.g. /settings/general).
// Cache is invalidated on any write (SetSetting/DeleteSetting).
type SystemSettingsService struct {
	db    *gorm.DB
	cache atomic.Pointer[map[string]string]
}

// NewSystemSettingsService creates a new SystemSettingsService
func NewSystemSettingsService(db *gorm.DB) *SystemSettingsService {
	return &SystemSettingsService{db: db}
}

// invalidateCache clears the in-memory cache. Called after any write operation.
func (s *SystemSettingsService) invalidateCache() {
	s.cache.Store(nil)
}

// GetSetting gets a system setting by key
func (s *SystemSettingsService) GetSetting(key string) (string, error) {
	var setting model.SystemSetting
	err := s.db.Where("`key` = ?", key).First(&setting).Error
	if err == gorm.ErrRecordNotFound {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	return setting.Value, nil
}

// SetSetting sets a system setting
func (s *SystemSettingsService) SetSetting(key, value string) error {
	// Try to find existing setting first
	var setting model.SystemSetting
	result := s.db.Where("`key` = ?", key).First(&setting)

	if result.Error == gorm.ErrRecordNotFound {
		// Record doesn't exist, try to create it
		setting = model.SystemSetting{Key: key, Value: value}
		err := s.db.Create(&setting).Error
		if err != nil {
			// If duplicate key error (race condition), try update instead
			if isDuplicateKeyError(err) {
				if e := s.db.Model(&model.SystemSetting{}).Where("`key` = ?", key).Update("value", value).Error; e != nil {
					return e
				}
				s.invalidateCache()
				return nil
			}
			return err
		}
		s.invalidateCache()
		return nil
	}

	if result.Error != nil {
		return result.Error
	}

	// Record exists, update it
	if err := s.db.Model(&setting).Update("value", value).Error; err != nil {
		return err
	}
	s.invalidateCache()
	return nil
}

// GetAllSettings gets all system settings, served from in-memory cache when available.
// Cache is populated on first access and invalidated on any write.
func (s *SystemSettingsService) GetAllSettings() (map[string]string, error) {
	if cached := s.cache.Load(); cached != nil {
		return *cached, nil
	}

	var settingsList []model.SystemSetting
	if err := s.db.Find(&settingsList).Error; err != nil {
		return nil, err
	}

	settings := make(map[string]string)
	for _, setting := range settingsList {
		settings[setting.Key] = setting.Value
	}

	s.cache.Store(&settings)
	return settings, nil
}

// DeleteSetting deletes a system setting
func (s *SystemSettingsService) DeleteSetting(key string) error {
	if err := s.db.Where("`key` = ?", key).Delete(&model.SystemSetting{}).Error; err != nil {
		return err
	}
	s.invalidateCache()
	return nil
}

// SMTP settings
type SMTPSettings struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	SSL      bool   `json:"ssl"`
}

// GetSMTPSettings gets SMTP settings
func (s *SystemSettingsService) GetSMTPSettings() (*SMTPSettings, error) {
	settings, err := s.GetAllSettings()
	if err != nil {
		return nil, err
	}

	return &SMTPSettings{
		Host:     settings["smtp.host"],
		Port:     parseInt(settings["smtp.port"], 587),
		Username: settings["smtp.username"],
		Password: settings["smtp.password"],
		From:     settings["smtp.from"],
		SSL:      settings["smtp.ssl"] == "true",
	}, nil
}

// SetSMTPSettings sets SMTP settings
func (s *SystemSettingsService) SetSMTPSettings(smtp *SMTPSettings) error {
	settings := map[string]string{
		"smtp.host":     smtp.Host,
		"smtp.port":     intToString(smtp.Port),
		"smtp.username": smtp.Username,
		"smtp.password": smtp.Password,
		"smtp.from":     smtp.From,
		"smtp.ssl":      boolToString(smtp.SSL),
	}

	for key, value := range settings {
		if err := s.SetSetting(key, value); err != nil {
			return err
		}
	}

	return nil
}

// LDAP settings
type LDAPSettings struct {
	Enabled      bool   `json:"enabled"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	BaseDN       string `json:"baseDN"`
	BindDN       string `json:"bindDN"`
	BindPassword string `json:"bindPassword"`
	UserFilter   string `json:"userFilter"`
	EmailAttr    string `json:"emailAttr"`
	NameAttr     string `json:"nameAttr"`
	TLS          bool   `json:"tls"`
}

// GetLDAPSettings gets LDAP settings
func (s *SystemSettingsService) GetLDAPSettings() (*LDAPSettings, error) {
	settings, err := s.GetAllSettings()
	if err != nil {
		return nil, err
	}

	return &LDAPSettings{
		Enabled:      settings["ldap.enabled"] == "true",
		Host:         settings["ldap.host"],
		Port:         parseInt(settings["ldap.port"], 389),
		BaseDN:       settings["ldap.baseDN"],
		BindDN:       settings["ldap.bindDN"],
		BindPassword: settings["ldap.bindPassword"],
		UserFilter:   settings["ldap.userFilter"],
		EmailAttr:    settings["ldap.emailAttr"],
		NameAttr:     settings["ldap.nameAttr"],
		TLS:          settings["ldap.tls"] == "true",
	}, nil
}

// SetLDAPSettings sets LDAP settings
func (s *SystemSettingsService) SetLDAPSettings(ldap *LDAPSettings) error {
	settings := map[string]string{
		"ldap.enabled":      boolToString(ldap.Enabled),
		"ldap.host":         ldap.Host,
		"ldap.port":         intToString(ldap.Port),
		"ldap.baseDN":       ldap.BaseDN,
		"ldap.bindDN":       ldap.BindDN,
		"ldap.bindPassword": ldap.BindPassword,
		"ldap.userFilter":   ldap.UserFilter,
		"ldap.emailAttr":    ldap.EmailAttr,
		"ldap.nameAttr":     ldap.NameAttr,
		"ldap.tls":          boolToString(ldap.TLS),
	}

	for key, value := range settings {
		if err := s.SetSetting(key, value); err != nil {
			return err
		}
	}

	return nil
}

// General settings
type GeneralSettings struct {
	SiteName          string `json:"siteName"`
	Description       string `json:"description"`
	AllowRegistration bool   `json:"allowRegistration"`
	AllowAnonymous    bool   `json:"allowAnonymous"`
	DefaultBranch     string `json:"defaultBranch"`
	Timezone          string `json:"timezone"`
}

// GetGeneralSettings gets general settings
func (s *SystemSettingsService) GetGeneralSettings() (*GeneralSettings, error) {
	settings, err := s.GetAllSettings()
	if err != nil {
		return nil, err
	}

	return &GeneralSettings{
		SiteName:          settings["general.siteName"],
		Description:       settings["general.description"],
		AllowRegistration: settings["general.allowRegistration"] != "false",
		AllowAnonymous:    settings["general.allowAnonymous"] == "true",
		DefaultBranch:     settings["general.defaultBranch"],
		Timezone:          settings["general.timezone"],
	}, nil
}

// SetGeneralSettings sets general settings
func (s *SystemSettingsService) SetGeneralSettings(general *GeneralSettings) error {
	settings := map[string]string{
		"general.siteName":          general.SiteName,
		"general.description":       general.Description,
		"general.allowRegistration": boolToString(general.AllowRegistration),
		"general.allowAnonymous":    boolToString(general.AllowAnonymous),
		"general.defaultBranch":     general.DefaultBranch,
		"general.timezone":          general.Timezone,
	}

	for key, value := range settings {
		if err := s.SetSetting(key, value); err != nil {
			return err
		}
	}

	return nil
}

// Helper functions
func parseInt(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	if err != nil {
		return defaultValue
	}
	return i
}

func intToString(i int) string {
	return fmt.Sprintf("%d", i)
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// SSH settings
type SSHSettings struct {
	Enabled          bool   `json:"enabled"`
	Host             string `json:"host"`
	Port             int    `json:"port"`
	HttpsUrlTemplate string `json:"httpsUrlTemplate"` // e.g. "https://{host}/{owner}/{repo}.git" or "http://{host}/{owner}/{repo}.git"
	SshUrlTemplate   string `json:"sshUrlTemplate"`   // e.g. "git@{host}:{owner}/{repo}.git" or "ssh://git@{host}:{port}/{owner}/{repo}.git"
}

// HasSSHSettings checks if SSH settings exist in the database
func (s *SystemSettingsService) HasSSHSettings() (bool, error) {
	var count int64
	err := s.db.Model(&model.SystemSetting{}).
		Where("`key` IN ?", []string{"ssh.enabled", "ssh.host", "ssh.port"}).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetSSHSettings gets SSH settings
func (s *SystemSettingsService) GetSSHSettings() (*SSHSettings, error) {
	settings, err := s.GetAllSettings()
	if err != nil {
		return nil, err
	}

	return &SSHSettings{
		Enabled:          settings["ssh.enabled"] == "true",
		Host:             settings["ssh.host"],
		Port:             parseInt(settings["ssh.port"], 2022),
		HttpsUrlTemplate: settings["ssh.httpsUrlTemplate"],
		SshUrlTemplate:   settings["ssh.sshUrlTemplate"],
	}, nil
}

// SetSSHSettings sets SSH settings
func (s *SystemSettingsService) SetSSHSettings(ssh *SSHSettings) error {
	settings := map[string]string{
		"ssh.enabled":          boolToString(ssh.Enabled),
		"ssh.host":             ssh.Host,
		"ssh.port":             intToString(ssh.Port),
		"ssh.httpsUrlTemplate": ssh.HttpsUrlTemplate,
		"ssh.sshUrlTemplate":   ssh.SshUrlTemplate,
	}

	for key, value := range settings {
		if err := s.SetSetting(key, value); err != nil {
			return err
		}
	}

	return nil
}

// Webhook settings
type WebhookSettings struct {
	BlockPrivateAddress bool   `json:"blockPrivateAddress"`
	Whitelist           string `json:"whitelist"`
}

// GetWebhookSettings gets webhook settings
func (s *SystemSettingsService) GetWebhookSettings() (*WebhookSettings, error) {
	settings, err := s.GetAllSettings()
	if err != nil {
		return nil, err
	}

	return &WebhookSettings{
		BlockPrivateAddress: settings["webhook.blockPrivateAddress"] != "false",
		Whitelist:           settings["webhook.whitelist"],
	}, nil
}

// SetWebhookSettings sets webhook settings
func (s *SystemSettingsService) SetWebhookSettings(webhook *WebhookSettings) error {
	settings := map[string]string{
		"webhook.blockPrivateAddress": boolToString(webhook.BlockPrivateAddress),
		"webhook.whitelist":           webhook.Whitelist,
	}

	for key, value := range settings {
		if err := s.SetSetting(key, value); err != nil {
			return err
		}
	}

	return nil
}

// Upload settings
type UploadSettings struct {
	MaxFileSize int `json:"maxFileSize"` // in MB
	Timeout     int `json:"timeout"`     // in seconds
}

// GetUploadSettings gets upload settings
func (s *SystemSettingsService) GetUploadSettings() (*UploadSettings, error) {
	settings, err := s.GetAllSettings()
	if err != nil {
		return nil, err
	}

	return &UploadSettings{
		MaxFileSize: parseInt(settings["upload.maxFileSize"], 10),
		Timeout:     parseInt(settings["upload.timeout"], 30),
	}, nil
}

// SetUploadSettings sets upload settings
func (s *SystemSettingsService) SetUploadSettings(upload *UploadSettings) error {
	settings := map[string]string{
		"upload.maxFileSize": intToString(upload.MaxFileSize),
		"upload.timeout":     intToString(upload.Timeout),
	}

	for key, value := range settings {
		if err := s.SetSetting(key, value); err != nil {
			return err
		}
	}

	return nil
}

// Repository settings
type RepositorySettings struct {
	MaxDiffFiles     int    `json:"maxDiffFiles"`
	MaxDiffLines     int    `json:"maxDiffLines"`
	HttpsUrlTemplate string `json:"httpsUrlTemplate"` // e.g. "https://{host}/{owner}/{repo}.git"
	SshUrlTemplate   string `json:"sshUrlTemplate"`   // e.g. "git@{host}:{owner}/{repo}.git"
}

// GetRepositorySettings gets repository settings
func (s *SystemSettingsService) GetRepositorySettings() (*RepositorySettings, error) {
	settings, err := s.GetAllSettings()
	if err != nil {
		return nil, err
	}

	return &RepositorySettings{
		MaxDiffFiles:     parseInt(settings["repository.maxDiffFiles"], 100),
		MaxDiffLines:     parseInt(settings["repository.maxDiffLines"], 1000),
		HttpsUrlTemplate: settings["repository.httpsUrlTemplate"],
		SshUrlTemplate:   settings["repository.sshUrlTemplate"],
	}, nil
}

// SetRepositorySettings sets repository settings
func (s *SystemSettingsService) SetRepositorySettings(repo *RepositorySettings) error {
	settings := map[string]string{
		"repository.maxDiffFiles":     intToString(repo.MaxDiffFiles),
		"repository.maxDiffLines":     intToString(repo.MaxDiffLines),
		"repository.httpsUrlTemplate": repo.HttpsUrlTemplate,
		"repository.sshUrlTemplate":   repo.SshUrlTemplate,
	}

	for key, value := range settings {
		if err := s.SetSetting(key, value); err != nil {
			return err
		}
	}

	return nil
}

// AI settings (OpenAI-compatible API for LLM features like story decomposition)
type AISettings struct {
	Enabled   bool   `json:"enabled"`
	Provider  string `json:"provider"` // openai|deepseek|qwen|zhipu|moonshot|custom
	BaseURL   string `json:"baseUrl"`  // e.g. https://api.deepseek.com/v1
	APIKey    string `json:"apiKey"`
	Model     string `json:"model"`     // e.g. deepseek-chat, gpt-4o-mini
	Timeout   int    `json:"timeout"`   // seconds, default 60
	MaxTokens int    `json:"maxTokens"` // default 2000
}

// GetAISettings gets AI settings
func (s *SystemSettingsService) GetAISettings() (*AISettings, error) {
	settings, err := s.GetAllSettings()
	if err != nil {
		return nil, err
	}
	return &AISettings{
		Enabled:   settings["ai.enabled"] == "true",
		Provider:  settings["ai.provider"],
		BaseURL:   settings["ai.baseUrl"],
		APIKey:    settings["ai.apiKey"],
		Model:     settings["ai.model"],
		Timeout:   parseInt(settings["ai.timeout"], 60),
		MaxTokens: parseInt(settings["ai.maxTokens"], 2000),
	}, nil
}

// SetAISettings sets AI settings
func (s *SystemSettingsService) SetAISettings(ai *AISettings) error {
	settings := map[string]string{
		"ai.enabled":   boolToString(ai.Enabled),
		"ai.provider":  ai.Provider,
		"ai.baseUrl":   ai.BaseURL,
		"ai.apiKey":    ai.APIKey,
		"ai.model":     ai.Model,
		"ai.timeout":   intToString(ai.Timeout),
		"ai.maxTokens": intToString(ai.MaxTokens),
	}
	for key, value := range settings {
		if err := s.SetSetting(key, value); err != nil {
			return err
		}
	}
	return nil
}
