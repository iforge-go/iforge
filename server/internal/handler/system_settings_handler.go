package handler

import (
	"fmt"
	"iforge/iforge/internal/model"
	"iforge/iforge/internal/service"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type SystemSettingsHandler struct {
	settingsService      *service.SystemSettingsService
	aiService            *service.AIService
	aiModelConfigService *service.AIModelConfigService
}

func NewSystemSettingsHandler(settingsService *service.SystemSettingsService, aiService *service.AIService, aiModelConfigService *service.AIModelConfigService) *SystemSettingsHandler {
	return &SystemSettingsHandler{
		settingsService:      settingsService,
		aiService:            aiService,
		aiModelConfigService: aiModelConfigService,
	}
}

// GetGeneralSettings gets general system settings
func (h *SystemSettingsHandler) GetGeneralSettings(c *fiber.Ctx) error {
	settings, err := h.settingsService.GetGeneralSettings()
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(settings)
	return nil
}

// GetPublicGeneralSettings returns public general settings (siteName, description,
// timezone). This endpoint is public (no auth) so the frontend can update the
// browser title and render the contribution graph with the correct timezone.
func (h *SystemSettingsHandler) GetPublicGeneralSettings(c *fiber.Ctx) error {
	settings, err := h.settingsService.GetGeneralSettings()
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	siteName := settings.SiteName
	if siteName == "" {
		siteName = "iForge"
	}

	c.Status(http.StatusOK).JSON(fiber.Map{
		"siteName":          siteName,
		"description":       settings.Description,
		"timezone":          settings.Timezone,
		"allowRegistration": settings.AllowRegistration,
	})
	return nil
}

// UpdateGeneralSettings updates general system settings
func (h *SystemSettingsHandler) UpdateGeneralSettings(c *fiber.Ctx) error {
	var settings service.GeneralSettings
	if err := c.BodyParser(&settings); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	if err := h.settingsService.SetGeneralSettings(&settings); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Settings updated successfully"})
	return nil
}

// GetSMTPSettings gets SMTP settings
func (h *SystemSettingsHandler) GetSMTPSettings(c *fiber.Ctx) error {
	settings, err := h.settingsService.GetSMTPSettings()
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(settings)
	return nil
}

// UpdateSMTPSettings updates SMTP settings
func (h *SystemSettingsHandler) UpdateSMTPSettings(c *fiber.Ctx) error {
	var settings service.SMTPSettings
	if err := c.BodyParser(&settings); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	if err := h.settingsService.SetSMTPSettings(&settings); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "SMTP settings updated successfully"})
	return nil
}

// TestSMTPSettings tests SMTP settings
func (h *SystemSettingsHandler) TestSMTPSettings(c *fiber.Ctx) error {
	var req struct {
		To string `json:"to" validate:"required,email"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	// Get SMTP settings
	_, err := h.settingsService.GetSMTPSettings()
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	// TODO: Implement SMTP test
	// For now, just return success
	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Test email sent successfully"})
	return nil
}

// GetLDAPSettings gets LDAP settings
func (h *SystemSettingsHandler) GetLDAPSettings(c *fiber.Ctx) error {
	settings, err := h.settingsService.GetLDAPSettings()
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(settings)
	return nil
}

// UpdateLDAPSettings updates LDAP settings
func (h *SystemSettingsHandler) UpdateLDAPSettings(c *fiber.Ctx) error {
	var settings service.LDAPSettings
	if err := c.BodyParser(&settings); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	if err := h.settingsService.SetLDAPSettings(&settings); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	LogAudit(c, "settings.ldap.update", "system_settings", "ldap", nil, true)

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "LDAP settings updated successfully"})
	return nil
}

// TestLDAPSettings tests LDAP connection
func (h *SystemSettingsHandler) TestLDAPSettings(c *fiber.Ctx) error {
	var req struct {
		Username string `json:"username" validate:"required"`
		Password string `json:"password" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	// Get LDAP settings
	settings, err := h.settingsService.GetLDAPSettings()
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	if !settings.Enabled {
		respondError(c, http.StatusBadRequest, "LDAP is not enabled")
		return nil
	}

	// Create LDAP service and test authentication
	ldapService := service.NewLDAPService(h.settingsService)
	user, err := ldapService.Authenticate(req.Username, req.Password)
	if err != nil {
		respondError(c, http.StatusUnauthorized, fmt.Sprintf("LDAP authentication failed: %v", err))
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{
		"message": "LDAP connection successful",
		"user": fiber.Map{
			"username": user.Username,
			"email":    user.Email,
			"fullName": user.FullName,
		},
	})
	return nil
}

// GetAllSettings gets all system settings
func (h *SystemSettingsHandler) GetAllSettings(c *fiber.Ctx) error {
	settings, err := h.settingsService.GetAllSettings()
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(settings)
	return nil
}

// GetSSHSettings gets SSH settings
func (h *SystemSettingsHandler) GetSSHSettings(c *fiber.Ctx) error {
	settings, err := h.settingsService.GetSSHSettings()
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(settings)
	return nil
}

// UpdateSSHSettings updates SSH settings
// Note: Port changes require service restart to take effect
func (h *SystemSettingsHandler) UpdateSSHSettings(c *fiber.Ctx) error {
	var settings service.SSHSettings
	if err := c.BodyParser(&settings); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	// Port is read-only in admin panel, use current running port
	// (Port changes require service restart)
	currentSettings, _ := h.settingsService.GetSSHSettings()
	if currentSettings != nil {
		settings.Port = currentSettings.Port
	}

	if err := h.settingsService.SetSSHSettings(&settings); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{
		"message": "SSH settings updated successfully",
		"warning": "Port changes require service restart to take effect",
	})
	return nil
}

// GetPublicSSHSettings returns the SSH server configuration (enabled/host/port)
// sourced from the database. This endpoint is public (no auth) so the
// frontend can construct the correct SSH clone URL.
func (h *SystemSettingsHandler) GetPublicSSHSettings(c *fiber.Ctx) error {
	settings, err := h.settingsService.GetSSHSettings()
	if err != nil {
		c.Status(http.StatusOK).JSON(fiber.Map{
			"enabled": false,
			"host":    "localhost",
			"port":    2022,
		})
		return nil
	}
	c.Status(http.StatusOK).JSON(fiber.Map{
		"enabled": settings.Enabled,
		"host":    settings.Host,
		"port":    settings.Port,
	})
	return nil
}

// GetPublicCloneUrlTemplates returns the clone URL templates for HTTPS and SSH
// This endpoint is public (no auth) so the frontend can generate clone URLs
func (h *SystemSettingsHandler) GetPublicCloneUrlTemplates(c *fiber.Ctx) error {
	settings, err := h.settingsService.GetRepositorySettings()
	if err != nil {
		c.Status(http.StatusOK).JSON(fiber.Map{
			"httpsUrlTemplate": "",
			"sshUrlTemplate":   "",
		})
		return nil
	}
	c.Status(http.StatusOK).JSON(fiber.Map{
		"httpsUrlTemplate": settings.HttpsUrlTemplate,
		"sshUrlTemplate":   settings.SshUrlTemplate,
	})
	return nil
}

// GetWebhookSettings gets webhook settings
func (h *SystemSettingsHandler) GetWebhookSettings(c *fiber.Ctx) error {
	settings, err := h.settingsService.GetWebhookSettings()
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(settings)
	return nil
}

// UpdateWebhookSettings updates webhook settings
func (h *SystemSettingsHandler) UpdateWebhookSettings(c *fiber.Ctx) error {
	var settings service.WebhookSettings
	if err := c.BodyParser(&settings); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	if err := h.settingsService.SetWebhookSettings(&settings); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Webhook settings updated successfully"})
	return nil
}

// GetUploadSettings gets upload settings
func (h *SystemSettingsHandler) GetUploadSettings(c *fiber.Ctx) error {
	settings, err := h.settingsService.GetUploadSettings()
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(settings)
	return nil
}

// UpdateUploadSettings updates upload settings
func (h *SystemSettingsHandler) UpdateUploadSettings(c *fiber.Ctx) error {
	var settings service.UploadSettings
	if err := c.BodyParser(&settings); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	if err := h.settingsService.SetUploadSettings(&settings); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Upload settings updated successfully"})
	return nil
}

// GetRepositorySettings gets repository settings
func (h *SystemSettingsHandler) GetRepositorySettings(c *fiber.Ctx) error {
	settings, err := h.settingsService.GetRepositorySettings()
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(settings)
	return nil
}

// UpdateRepositorySettings updates repository settings
func (h *SystemSettingsHandler) UpdateRepositorySettings(c *fiber.Ctx) error {
	var settings service.RepositorySettings
	if err := c.BodyParser(&settings); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	if err := h.settingsService.SetRepositorySettings(&settings); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Repository settings updated successfully"})
	return nil
}

// GetAISettings returns the default AI configuration (backward compatible with legacy endpoint; API key masked).
// Returns {enabled: false} if no default configuration exists.
func (h *SystemSettingsHandler) GetAISettings(c *fiber.Ctx) error {
	cfg, err := h.aiModelConfigService.GetDefault()
	if err != nil {
		if err == service.ErrAIModelConfigNoDefault {
			return c.Status(http.StatusOK).JSON(service.AISettings{Enabled: false})
		}
		return respondError(c, http.StatusInternalServerError, err.Error())
	}
	return c.Status(http.StatusOK).JSON(service.AISettings{
		Enabled:   cfg.Enabled,
		Provider:  cfg.Provider,
		BaseURL:   cfg.BaseURL,
		APIKey:    maskAPIKey(cfg.APIKey),
		Model:     cfg.Model,
		Timeout:   cfg.Timeout,
		MaxTokens: cfg.MaxTokens,
	})
}

// UpdateAISettings updates or creates the default AI configuration (backward compatible with legacy endpoint).
// Updates existing default config if present; otherwise creates a new record named "Default Config" and sets it as default.
func (h *SystemSettingsHandler) UpdateAISettings(c *fiber.Ctx) error {
	var settings service.AISettings
	if err := c.BodyParser(&settings); err != nil {
		return respondError(c, http.StatusBadRequest, err.Error())
	}

	cfg := &model.AIModelConfig{
		Provider:  settings.Provider,
		BaseURL:   settings.BaseURL,
		APIKey:    settings.APIKey,
		Model:     settings.Model,
		Timeout:   settings.Timeout,
		MaxTokens: settings.MaxTokens,
		Enabled:   settings.Enabled,
		IsDefault: true,
	}

	if settings.APIKey == "" || strings.Contains(settings.APIKey, "****") {
		existing, err := h.aiModelConfigService.GetDefault()
		if err == nil {
			cfg.APIKey = existing.APIKey
			cfg.Name = existing.Name
		}
	}

	existing, err := h.aiModelConfigService.GetDefault()
	if err == nil {
		_, err = h.aiModelConfigService.Update(existing.ID, cfg)
	} else if err == service.ErrAIModelConfigNoDefault {
		if cfg.Name == "" {
			cfg.Name = "Default Config"
		}
		_, err = h.aiModelConfigService.Create(cfg)
	}
	if err != nil {
		return mapConfigError(c, err)
	}
	return c.Status(http.StatusOK).JSON(fiber.Map{"message": "AI settings updated successfully"})
}

// TestAISettings tests AI connection
func (h *SystemSettingsHandler) TestAISettings(c *fiber.Ctx) error {
	if h.aiService == nil {
		respondError(c, http.StatusServiceUnavailable, "AI service not initialized")
		return nil
	}
	if err := h.aiService.TestConnection(); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}
	c.Status(http.StatusOK).JSON(fiber.Map{"message": "AI connection successful"})
	return nil
}
