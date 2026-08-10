package handler

import (
	"net/http"

	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type DatabaseViewerHandler struct {
	databaseViewerService *service.DatabaseViewerService
}

func NewDatabaseViewerHandler(databaseViewerService *service.DatabaseViewerService) *DatabaseViewerHandler {
	return &DatabaseViewerHandler{
		databaseViewerService: databaseViewerService,
	}
}

func (h *DatabaseViewerHandler) ListTables(c *fiber.Ctx) error {
	tables, err := h.databaseViewerService.ListTables()
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(tables)
	return nil
}

func (h *DatabaseViewerHandler) GetTableSchema(c *fiber.Ctx) error {
	tableName := c.Params("table")
	if tableName == "" {
		respondError(c, http.StatusBadRequest, "table name is required")
		return nil
	}

	schema, err := h.databaseViewerService.GetTableSchema(tableName)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(schema)
	return nil
}

func (h *DatabaseViewerHandler) QueryTable(c *fiber.Ctx) error {
	tableName := c.Params("table")
	if tableName == "" {
		respondError(c, http.StatusBadRequest, "table name is required")
		return nil
	}

	var req struct {
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	}

	if err := c.BodyParser(&req); err != nil {
		req.Limit = 100
		req.Offset = 0
	}

	result, err := h.databaseViewerService.QueryTable(tableName, req.Limit, req.Offset)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(result)
	return nil
}

// ExecuteQuery is removed for security. Use QueryTable instead.
