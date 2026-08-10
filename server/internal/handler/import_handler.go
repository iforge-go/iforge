package handler

import (
	"net/http"

	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type ImportHandler struct {
	importService *service.ImportService
	repoService   *service.RepositoryService
}

func NewImportHandler(importService *service.ImportService, repoService *service.RepositoryService) *ImportHandler {
	return &ImportHandler{
		importService: importService,
		repoService:   repoService,
	}
}

// ImportFromURL imports a repository from a URL
func (h *ImportHandler) ImportFromURL(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	var req struct {
		Owner       string `json:"owner" validate:"required"`
		RepoName    string `json:"repoName" validate:"required"`
		SourceURL   string `json:"sourceUrl" validate:"required"`
		IsPrivate   bool   `json:"isPrivate"`
		Description string `json:"description"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	// Verify user has permission to create repos under this owner
	if req.Owner != user.UserName && !user.IsAdmin {
		respondError(c, http.StatusForbidden, "Forbidden")
		return nil
	}

	// Import the repository
	if err := h.importService.ImportRepository(req.Owner, req.RepoName, req.SourceURL); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	// Create repository record in database
	desc := req.Description
	repo, err := h.repoService.CreateRepository(
		req.Owner,
		req.RepoName,
		req.IsPrivate,
		&desc,
		"main",
		nil, nil, nil, nil,
		user.UserName,
	)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusCreated).JSON(repo)
	return nil
}

// ParseURL parses a repository URL and returns owner and repo name
func (h *ImportHandler) ParseURL(c *fiber.Ctx) error {
	url := c.Query("url")
	if url == "" {
		respondError(c, http.StatusBadRequest, "url parameter is required")
		return nil
	}

	owner, repo, err := h.importService.ParseRepositoryURL(url)
	if err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{
		"owner": owner,
		"repo":  repo,
	})
	return nil
}
