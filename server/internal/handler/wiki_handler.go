package handler

import (
	"net/http"

	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/model"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type WikiHandler struct {
	wikiService *service.WikiService
	repoService *service.RepositoryService
}

func NewWikiHandler(wikiService *service.WikiService, repoService *service.RepositoryService) *WikiHandler {
	return &WikiHandler{
		wikiService: wikiService,
		repoService: repoService,
	}
}

func (h *WikiHandler) isWikiEditable(repo *model.Repository, user *model.Account) bool {
	wikiOption := repo.WikiOption
	if wikiOption == "" {
		wikiOption = "PUBLIC"
	}

	switch wikiOption {
	case "DISABLE":
		return false
	case "PRIVATE":
		return h.repoService.HasMemberRole(repo, user)
	case "PUBLIC":
		return h.repoService.HasViewerRole(repo, user)
	case "ALL":
		if repo.IsPrivate {
			return false
		}
		return user != nil
	default:
		return false
	}
}

func (h *WikiHandler) ListWikiPages(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	repository, err := h.repoService.GetRepository(owner, repo)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	user := contextutil.GetUserFromContext(c)
	if repository.IsPrivate {
		if user == nil || !h.repoService.HasViewerRole(repository, user) {
			respondError(c, http.StatusNotFound, "Repository not found")
			return nil
		}
	}

	pages, err := h.wikiService.ListWikiPages(owner, repo)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list wiki pages")
		return nil
	}

	c.Status(http.StatusOK).JSON(pages)
	return nil
}

func (h *WikiHandler) GetWikiPage(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	pageName := c.Params("page")

	repository, err := h.repoService.GetRepository(owner, repo)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	user := contextutil.GetUserFromContext(c)
	if repository.IsPrivate {
		if user == nil || !h.repoService.HasViewerRole(repository, user) {
			respondError(c, http.StatusNotFound, "Repository not found")
			return nil
		}
	}

	page, err := h.wikiService.GetWikiPage(owner, repo, pageName)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to get wiki page")
		return nil
	}

	if page == nil {
		respondError(c, http.StatusNotFound, "Wiki page not found")
		return nil
	}

	c.Status(http.StatusOK).JSON(page)
	return nil
}

func (h *WikiHandler) CreateWikiPage(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	repository, err := h.repoService.GetRepository(owner, repo)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	if !h.isWikiEditable(repository, user) {
		respondError(c, http.StatusForbidden, "Wiki editing is disabled or insufficient permissions")
		return nil
	}

	var req struct {
		PageName string `json:"pageName" validate:"required"`
		Title    string `json:"title" validate:"required"`
		Content  string `json:"content" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request")
		return nil
	}

	page, err := h.wikiService.CreateWikiPage(owner, repo, req.PageName, req.Title, req.Content, user.UserName)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to create wiki page")
		return nil
	}

	c.Status(http.StatusCreated).JSON(page)
	return nil
}

func (h *WikiHandler) UpdateWikiPage(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	pageName := c.Params("page")

	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	repository, err := h.repoService.GetRepository(owner, repo)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	// Check WikiOption permission (aligned with GitBucket)
	if !h.isWikiEditable(repository, user) {
		respondError(c, http.StatusForbidden, "Wiki editing is disabled or insufficient permissions")
		return nil
	}

	var req struct {
		Title   string `json:"title" validate:"required"`
		Content string `json:"content" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request")
		return nil
	}

	page, err := h.wikiService.UpdateWikiPage(owner, repo, pageName, req.Title, req.Content, user.UserName)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to update wiki page")
		return nil
	}

	c.Status(http.StatusOK).JSON(page)
	return nil
}

// DeleteWikiPage deletes a wiki page
func (h *WikiHandler) DeleteWikiPage(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	pageName := c.Params("page")

	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	repository, err := h.repoService.GetRepository(owner, repo)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	// Check WikiOption permission (aligned with GitBucket)
	if !h.isWikiEditable(repository, user) {
		respondError(c, http.StatusForbidden, "Wiki editing is disabled or insufficient permissions")
		return nil
	}

	if err := h.wikiService.DeleteWikiPage(owner, repo, pageName); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to delete wiki page")
		return nil
	}

	c.Status(http.StatusNoContent)
	return nil
}
