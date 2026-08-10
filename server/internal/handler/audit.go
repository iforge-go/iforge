package handler

import (
	"strconv"
	"time"

	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

var defaultAuditService *service.AuditService

func SetAuditService(svc *service.AuditService) {
	defaultAuditService = svc
}

func LogAudit(c *fiber.Ctx, action, resourceType, resourceID string, detail map[string]interface{}, success bool) {
	if defaultAuditService == nil {
		return
	}
	actor := ""
	if user := contextutil.GetUserFromContext(c); user != nil {
		actor = user.UserName
	}
	defaultAuditService.Log(actor, c.IP(), action, resourceType, resourceID, detail, success)
}

type AuditHandler struct {
	auditService *service.AuditService
}

func NewAuditHandler(auditService *service.AuditService) *AuditHandler {
	return &AuditHandler{auditService: auditService}
}

func (h *AuditHandler) ListAuditLogs(c *fiber.Ctx) error {
	q := service.AuditQuery{
		Actor:        c.Query("actor"),
		Action:       c.Query("action"),
		ResourceType: c.Query("resourceType"),
		Page:         1,
		PageSize:     50,
	}
	if v := c.Query("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			q.Page = p
		}
	}
	if v := c.Query("pageSize"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 && p <= 200 {
			q.PageSize = p
		}
	}
	if v := c.Query("startTime"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			q.StartTime = &t
		}
	}
	if v := c.Query("endTime"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			q.EndTime = &t
		}
	}

	logs, total, err := h.auditService.ListAuditLogs(q)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"logs":     logs,
		"total":    total,
		"page":     q.Page,
		"pageSize": q.PageSize,
	})
}
