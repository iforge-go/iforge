package handler

import (
	"net/http"
	"strconv"
	"strings"

	"iforge/iforge/internal/model"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type AIModelConfigHandler struct {
	svc       *service.AIModelConfigService
	aiService *service.AIService
}

func NewAIModelConfigHandler(svc *service.AIModelConfigService, aiService *service.AIService) *AIModelConfigHandler {
	return &AIModelConfigHandler{svc: svc, aiService: aiService}
}

func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 4 {
		return "****"
	}
	return strings.Repeat("*", len(key)-4) + key[len(key)-4:]
}

func (h *AIModelConfigHandler) List(c *fiber.Ctx) error {
	cfgs, err := h.svc.List()
	if err != nil {
		return respondError(c, http.StatusInternalServerError, err.Error())
	}
	for _, cfg := range cfgs {
		cfg.APIKey = maskAPIKey(cfg.APIKey)
	}
	return c.JSON(cfgs)
}

func (h *AIModelConfigHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return respondError(c, http.StatusBadRequest, "invalid id")
	}
	cfg, err := h.svc.Get(uint(id))
	if err != nil {
		return mapConfigError(c, err)
	}
	cfg.APIKey = maskAPIKey(cfg.APIKey)
	return c.JSON(cfg)
}

func (h *AIModelConfigHandler) Create(c *fiber.Ctx) error {
	var cfg model.AIModelConfig
	if err := c.BodyParser(&cfg); err != nil {
		return respondError(c, http.StatusBadRequest, err.Error())
	}
	created, err := h.svc.Create(&cfg)
	if err != nil {
		return mapConfigError(c, err)
	}
	created.APIKey = maskAPIKey(created.APIKey)
	return c.Status(http.StatusCreated).JSON(created)
}

func (h *AIModelConfigHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return respondError(c, http.StatusBadRequest, "invalid id")
	}
	var cfg model.AIModelConfig
	if err := c.BodyParser(&cfg); err != nil {
		return respondError(c, http.StatusBadRequest, err.Error())
	}
	updated, err := h.svc.Update(uint(id), &cfg)
	if err != nil {
		return mapConfigError(c, err)
	}
	updated.APIKey = maskAPIKey(updated.APIKey)
	return c.JSON(updated)
}

func (h *AIModelConfigHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return respondError(c, http.StatusBadRequest, "invalid id")
	}
	if err := h.svc.Delete(uint(id)); err != nil {
		return mapConfigError(c, err)
	}
	return c.JSON(fiber.Map{"message": "deleted"})
}

func (h *AIModelConfigHandler) SetDefault(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return respondError(c, http.StatusBadRequest, "invalid id")
	}
	cfg, err := h.svc.SetDefault(uint(id))
	if err != nil {
		return mapConfigError(c, err)
	}
	cfg.APIKey = maskAPIKey(cfg.APIKey)
	return c.JSON(cfg)
}

func (h *AIModelConfigHandler) Test(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return respondError(c, http.StatusBadRequest, "invalid id")
	}

	var body model.AIModelConfig
	hasBody := c.BodyParser(&body) == nil && body.BaseURL != ""

	var cfg *model.AIModelConfig
	if hasBody {
		if body.APIKey == "" || strings.Contains(body.APIKey, "****") {
			existing, err := h.svc.Get(uint(id))
			if err == nil {
				body.APIKey = existing.APIKey
			}
		}
		cfg = &body
	} else {
		cfg, err = h.svc.Get(uint(id))
		if err != nil {
			return mapConfigError(c, err)
		}
	}

	if err := h.aiService.TestConnectionWithConfig(cfg); err != nil {
		_ = h.svc.UpdateTestResult(uint(id), false)
		return respondError(c, http.StatusBadRequest, err.Error())
	}
	_ = h.svc.UpdateTestResult(uint(id), true)
	return c.JSON(fiber.Map{"message": "AI connection successful"})
}

func mapConfigError(c *fiber.Ctx, err error) error {
	switch err {
	case service.ErrAIModelConfigNotFound:
		return respondError(c, http.StatusNotFound, err.Error())
	case service.ErrAIModelConfigNameConflict:
		return respondError(c, http.StatusConflict, err.Error())
	case service.ErrAIModelConfigLastDefault:
		return respondError(c, http.StatusBadRequest, err.Error())
	default:
		return respondError(c, http.StatusInternalServerError, err.Error())
	}
}
