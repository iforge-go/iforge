package handler

import (
	"net/http"

	gitsvc "iforge/iforge/internal/git"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type MonitorHandler struct {
	logger  *gitsvc.Logger
	metrics *service.MetricsCollector
}

func NewMonitorHandler(logger *gitsvc.Logger, metrics *service.MetricsCollector) *MonitorHandler {
	return &MonitorHandler{
		logger:  logger,
		metrics: metrics,
	}
}

func (h *MonitorHandler) GetMetrics(c *fiber.Ctx) error {
	metrics := h.metrics.GetAll()
	c.Status(http.StatusOK).JSON(fiber.Map{
		"metrics": metrics,
	})
	return nil
}

func (h *MonitorHandler) GetLogs(c *fiber.Ctx) error {
	c.Status(http.StatusOK).JSON(fiber.Map{
		"message": "Log retrieval not yet implemented",
	})
	return nil
}

func (h *MonitorHandler) GetSystemInfo(c *fiber.Ctx) error {
	info := fiber.Map{
		"version":    "1.0.0",
		"go_version": "1.21",
		"uptime":     "TODO",
		"memory": fiber.Map{
			"total":     "TODO",
			"used":      "TODO",
			"available": "TODO",
		},
		"cpu": fiber.Map{
			"cores": "TODO",
			"usage": "TODO",
		},
	}

	c.Status(http.StatusOK).JSON(info)
	return nil
}

func (h *MonitorHandler) GetPrometheusMetrics(c *fiber.Ctx) error {
	metrics := h.metrics.ExportPrometheus()
	c.Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	return c.SendString(metrics)
}
