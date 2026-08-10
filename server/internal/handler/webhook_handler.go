package handler

import (
	"net/http"
	"strconv"

	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type WebhookHandler struct {
	webhookService *service.WebhookService
	repoService    *service.RepositoryService
}

func NewWebhookHandler(webhookService *service.WebhookService, repoService *service.RepositoryService) *WebhookHandler {
	return &WebhookHandler{
		webhookService: webhookService,
		repoService:    repoService,
	}
}

type CreateWebhookRequest struct {
	URL         string   `json:"url" validate:"required"`
	ContentType string   `json:"contentType" validate:"required"`
	Token       *string  `json:"token"`
	Events      []string `json:"events"`
	Active      bool     `json:"active"`
	InsecureSSL bool     `json:"insecureSSL"`
}

func (h *WebhookHandler) ListWebhooks(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	_, _, ok := getUserAndCheckPermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	webhooks, err := h.webhookService.ListWebhooks(owner, repo)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list webhooks")
		return nil
	}

	c.Status(http.StatusOK).JSON(webhooks)
	return nil
}

func (h *WebhookHandler) CreateWebhook(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	_, _, ok := getUserAndCheckPermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	var req CreateWebhookRequest
	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body")
		return nil
	}

	eventsJSON := "[]"
	if len(req.Events) > 0 {
		eventsJSON = "["
		for i, event := range req.Events {
			if i > 0 {
				eventsJSON += ","
			}
			eventsJSON += `"` + event + `"`
		}
		eventsJSON += "]"
	}

	webhook, err := h.webhookService.CreateWebhook(owner, repo, req.URL, req.ContentType, req.Token, eventsJSON, req.Active, req.InsecureSSL)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to create webhook")
		return nil
	}

	c.Status(http.StatusCreated).JSON(webhook)
	return nil
}

func (h *WebhookHandler) DeleteWebhook(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	webhookIDStr := c.Params("id")

	_, _, ok := getUserAndCheckPermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	webhookID, err := strconv.ParseUint(webhookIDStr, 10, 32)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid webhook ID")
		return nil
	}

	if err := h.webhookService.DeleteWebhook(uint(webhookID)); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to delete webhook")
		return nil
	}

	c.Status(http.StatusNoContent)
	return nil
}

func (h *WebhookHandler) TestWebhook(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	webhookIDStr := c.Params("id")

	user, repoObj, ok := getUserAndCheckPermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	webhookID, err := strconv.ParseUint(webhookIDStr, 10, 32)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid webhook ID")
		return nil
	}

	if err := h.webhookService.TestWebhook(uint(webhookID), repoObj, user); err != nil {
		if err == service.ErrWebhookNotFound {
			respondError(c, http.StatusNotFound, "Webhook not found")
			return nil
		}
		respondError(c, http.StatusInternalServerError, "Failed to test webhook")
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Webhook test sent successfully"})
	return nil
}

func (h *WebhookHandler) ListWebhookDeliveries(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	webhookIDStr := c.Params("id")

	_, _, ok := getUserAndCheckPermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	webhookID, err := strconv.ParseUint(webhookIDStr, 10, 32)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid webhook ID")
		return nil
	}

	deliveries, err := h.webhookService.ListWebhookDeliveries(uint(webhookID))
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list deliveries")
		return nil
	}

	c.Status(http.StatusOK).JSON(deliveries)
	return nil
}
