package handler

import (
	"net/http"

	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/model"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type MirrorHandler struct {
	mirrorService *service.MirrorService
	repoService   *service.RepositoryService
}

func NewMirrorHandler(mirrorService *service.MirrorService, repoService *service.RepositoryService) *MirrorHandler {
	return &MirrorHandler{
		mirrorService: mirrorService,
		repoService:   repoService,
	}
}

// GetMirror godoc
// @Summary Get repository mirror configuration
// @Description Get the mirror configuration for a repository
// @Tags mirrors
// @Accept json
// @Produce json
// @Param owner path string true "Repository owner"
// @Param repo path string true "Repository name"
// @Success 200 {object} model.RepositoryMirror
// @Failure 404 {object} map[string]string
// @Router /repos/{owner}/{repo}/mirror [get]
func (h *MirrorHandler) GetMirror(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	_, _, ok := getUserAndCheckPermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	mirror, err := h.mirrorService.GetMirror(owner, repo)
	if err != nil {
		if err == service.ErrMirrorNotFound {
			c.Status(http.StatusOK).JSON(nil)
			return nil
		}
		respondError(c, http.StatusInternalServerError, "Failed to get mirror configuration")
		return nil
	}

	c.Status(http.StatusOK).JSON(mirror)
	return nil
}

// CreateMirror godoc
// @Summary Create repository mirror configuration
// @Description Create a new mirror configuration for a repository
// @Tags mirrors
// @Accept json
// @Produce json
// @Param owner path string true "Repository owner"
// @Param repo path string true "Repository name"
// @Param body body model.RepositoryMirror true "Mirror configuration"
// @Success 201 {object} model.RepositoryMirror
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /repos/{owner}/{repo}/mirror [post]
func (h *MirrorHandler) CreateMirror(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	currentUser := contextutil.GetUserFromContext(c)
	if currentUser == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	// Check if user has admin access to the repository
	repoObj, err := h.repoService.GetRepository(owner, repo)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	if !h.repoService.HasOwnerRole(repoObj, currentUser) {
		respondError(c, http.StatusForbidden, "Forbidden")
		return nil
	}

	var mirror model.RepositoryMirror
	if err := c.BodyParser(&mirror); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body")
		return nil
	}

	mirror.UserName = owner
	mirror.RepositoryName = repo

	if err := h.mirrorService.CreateMirror(&mirror); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to create mirror configuration")
		return nil
	}

	c.Status(http.StatusCreated).JSON(mirror)
	return nil
}

// UpdateMirror godoc
// @Summary Update repository mirror configuration
// @Description Update the mirror configuration for a repository
// @Tags mirrors
// @Accept json
// @Produce json
// @Param owner path string true "Repository owner"
// @Param repo path string true "Repository name"
// @Param body body model.RepositoryMirror true "Mirror configuration"
// @Success 200 {object} model.RepositoryMirror
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /repos/{owner}/{repo}/mirror [put]
func (h *MirrorHandler) UpdateMirror(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	currentUser := contextutil.GetUserFromContext(c)
	if currentUser == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	// Check if user has admin access to the repository
	repoObj, err := h.repoService.GetRepository(owner, repo)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	if !h.repoService.HasOwnerRole(repoObj, currentUser) {
		respondError(c, http.StatusForbidden, "Forbidden")
		return nil
	}

	var mirror model.RepositoryMirror
	if err := c.BodyParser(&mirror); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body")
		return nil
	}

	mirror.UserName = owner
	mirror.RepositoryName = repo

	if err := h.mirrorService.UpdateMirror(&mirror); err != nil {
		if err == service.ErrMirrorNotFound {
			respondError(c, http.StatusNotFound, "Mirror configuration not found")
			return nil
		}
		respondError(c, http.StatusInternalServerError, "Failed to update mirror configuration")
		return nil
	}

	c.Status(http.StatusOK).JSON(mirror)
	return nil
}

// DeleteMirror godoc
// @Summary Delete repository mirror configuration
// @Description Delete the mirror configuration for a repository
// @Tags mirrors
// @Accept json
// @Produce json
// @Param owner path string true "Repository owner"
// @Param repo path string true "Repository name"
// @Success 204 "No Content"
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /repos/{owner}/{repo}/mirror [delete]
func (h *MirrorHandler) DeleteMirror(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	currentUser := contextutil.GetUserFromContext(c)
	if currentUser == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	// Check if user has admin access to the repository
	repoObj, err := h.repoService.GetRepository(owner, repo)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	if !h.repoService.HasOwnerRole(repoObj, currentUser) {
		respondError(c, http.StatusForbidden, "Forbidden")
		return nil
	}

	if err := h.mirrorService.DeleteMirror(owner, repo); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to delete mirror configuration")
		return nil
	}

	c.Status(http.StatusNoContent)
	return nil
}

// SyncMirror godoc
// @Summary Manually sync repository mirror
// @Description Manually trigger a sync for the repository mirror
// @Tags mirrors
// @Accept json
// @Produce json
// @Param owner path string true "Repository owner"
// @Param repo path string true "Repository name"
// @Success 200 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /repos/{owner}/{repo}/mirror/sync [post]
func (h *MirrorHandler) SyncMirror(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	currentUser := contextutil.GetUserFromContext(c)
	if currentUser == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	// Check if user has admin access to the repository
	repoObj, err := h.repoService.GetRepository(owner, repo)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	if !h.repoService.HasOwnerRole(repoObj, currentUser) {
		respondError(c, http.StatusForbidden, "Forbidden")
		return nil
	}

	if err := h.mirrorService.SyncMirror(owner, repo); err != nil {
		if err == service.ErrMirrorNotFound {
			respondError(c, http.StatusNotFound, "Mirror configuration not found")
			return nil
		}
		respondError(c, http.StatusInternalServerError, "Failed to sync mirror")
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Mirror sync initiated"})
	return nil
}

// RegisterRoutes registers mirror routes (deprecated - routes are now registered in router.go)
func (h *MirrorHandler) RegisterRoutes(rg interface{}) {
	// This method is no longer used in Fiber migration
	// Routes are registered directly in router.go
}
