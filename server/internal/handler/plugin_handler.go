package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"iforge/iforge/internal/model"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type PluginHandler struct {
	pluginManager  *service.PluginManager
	templateEngine *service.TemplateEngine
	eventEngine    *service.PluginEventEngine
}

func NewPluginHandler(pm *service.PluginManager, te *service.TemplateEngine, ee *service.PluginEventEngine) *PluginHandler {
	return &PluginHandler{
		pluginManager:  pm,
		templateEngine: te,
		eventEngine:    ee,
	}
}

func (h *PluginHandler) ListPlugins(c *fiber.Ctx) error {
	plugins := h.pluginManager.ListPlugins()
	c.Status(http.StatusOK).JSON(plugins)
	return nil
}

func (h *PluginHandler) GetPlugin(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		respondError(c, http.StatusBadRequest, "id is required")
		return nil
	}

	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		respondError(c, http.StatusBadRequest, "invalid id")
		return nil
	}

	plugin, err := h.pluginManager.GetPlugin(id)
	if err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(plugin)
	return nil
}

func (h *PluginHandler) RegisterPlugin(c *fiber.Ctx) error {
	var req struct {
		Name        string `json:"name" validate:"required"`
		Version     string `json:"version" validate:"required"`
		Description string `json:"description"`
		Author      string `json:"author"`
		URL         string `json:"url"`
		Endpoint    string `json:"endpoint" validate:"required"`
		Enabled     bool   `json:"enabled"`
		Config      string `json:"config"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	plugin := &model.Plugin{
		Name:        req.Name,
		Version:     req.Version,
		Description: req.Description,
		Author:      req.Author,
		URL:         req.URL,
		Endpoint:    req.Endpoint,
		Enabled:     req.Enabled,
		Config:      req.Config,
	}

	if err := h.pluginManager.RegisterPlugin(plugin); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusCreated).JSON(plugin)
	return nil
}

func (h *PluginHandler) UnregisterPlugin(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		respondError(c, http.StatusBadRequest, "id is required")
		return nil
	}

	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		respondError(c, http.StatusBadRequest, "invalid id")
		return nil
	}

	if err := h.pluginManager.UnregisterPlugin(id); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Plugin unregistered successfully"})
	return nil
}

func (h *PluginHandler) EnablePlugin(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		respondError(c, http.StatusBadRequest, "id is required")
		return nil
	}

	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		respondError(c, http.StatusBadRequest, "invalid id")
		return nil
	}

	if err := h.pluginManager.EnablePlugin(id); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Plugin enabled successfully"})
	return nil
}

func (h *PluginHandler) DisablePlugin(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		respondError(c, http.StatusBadRequest, "id is required")
		return nil
	}

	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		respondError(c, http.StatusBadRequest, "invalid id")
		return nil
	}

	if err := h.pluginManager.DisablePlugin(id); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Plugin disabled successfully"})
	return nil
}

func (h *PluginHandler) GetPluginStatus(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		respondError(c, http.StatusBadRequest, "id is required")
		return nil
	}

	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		respondError(c, http.StatusBadRequest, "invalid id")
		return nil
	}

	status, err := h.pluginManager.GetPluginStatus(id)
	if err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(status)
	return nil
}

// GetAllPluginStatuses gets statuses of all plugins
func (h *PluginHandler) GetAllPluginStatuses(c *fiber.Ctx) error {
	statuses := h.pluginManager.GetAllPluginStatuses()
	c.Status(http.StatusOK).JSON(statuses)
	return nil
}

// ListTemplates returns all available plugin templates
func (h *PluginHandler) ListTemplates(c *fiber.Ctx) error {
	templates, err := h.templateEngine.ListTemplates()
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}
	c.Status(http.StatusOK).JSON(templates)
	return nil
}

// GetTemplate returns a specific template by ID
func (h *PluginHandler) GetTemplate(c *fiber.Ctx) error {
	templateID := c.Params("templateId")
	if templateID == "" {
		respondError(c, http.StatusBadRequest, "template_id is required")
		return nil
	}

	template, err := h.templateEngine.GetTemplate(templateID)
	if err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(template)
	return nil
}

// InstallFromTemplate installs a plugin from a template
func (h *PluginHandler) InstallFromTemplate(c *fiber.Ctx) error {
	var req struct {
		TemplateID string                 `json:"template_id" validate:"required"`
		Name       string                 `json:"name" validate:"required"`
		Config     map[string]interface{} `json:"config"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	plugin, err := h.templateEngine.Instantiate(req.TemplateID, req.Config, req.Name)
	if err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	// Load the new plugin into the event engine
	if err := h.eventEngine.LoadHandlers(); err != nil {
		respondError(c, http.StatusInternalServerError, fmt.Sprintf("plugin installed but failed to load handlers: %v", err))
		return nil
	}

	c.Status(http.StatusCreated).JSON(plugin)
	return nil
}

// UpdatePlugin updates plugin configuration
func (h *PluginHandler) UpdatePlugin(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		respondError(c, http.StatusBadRequest, "id is required")
		return nil
	}

	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		respondError(c, http.StatusBadRequest, "invalid id")
		return nil
	}

	var req struct {
		Config map[string]interface{} `json:"config" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid config format")
		return nil
	}

	if err := h.pluginManager.UpdatePlugin(id, string(configJSON)); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Plugin updated successfully"})
	return nil
}
