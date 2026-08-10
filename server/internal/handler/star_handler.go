package handler

import (
	"net/http"

	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/model"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type StarHandler struct {
	starService *service.StarService
	repoService *service.RepositoryService
}

func NewStarHandler(starService *service.StarService, repoService *service.RepositoryService) *StarHandler {
	return &StarHandler{
		starService: starService,
		repoService: repoService,
	}
}

func (h *StarHandler) StarRepository(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	owner := c.Params("owner")
	repo := c.Params("repo")

	repository, err := h.repoService.GetRepository(owner, repo)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	if repository.IsPrivate {
		if !h.repoService.HasViewerRole(repository, user) {
			respondError(c, http.StatusNotFound, "Repository not found")
			return nil
		}
	}

	if err := h.starService.StarRepository(owner, repo, user.UserName); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to star repository")
		return nil
	}

	c.Status(http.StatusNoContent)
	return nil
}

func (h *StarHandler) UnstarRepository(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	owner := c.Params("owner")
	repo := c.Params("repo")

	repository, err := h.repoService.GetRepository(owner, repo)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	if repository.IsPrivate {
		if !h.repoService.HasViewerRole(repository, user) {
			respondError(c, http.StatusNotFound, "Repository not found")
			return nil
		}
	}

	if err := h.starService.UnstarRepository(owner, repo, user.UserName); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to unstar repository")
		return nil
	}

	c.Status(http.StatusNoContent)
	return nil
}

func (h *StarHandler) IsStarred(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		c.Status(http.StatusOK).JSON(fiber.Map{"starred": false})
		return nil
	}

	owner := c.Params("owner")
	repo := c.Params("repo")

	repository, err := h.repoService.GetRepository(owner, repo)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	if repository.IsPrivate {
		if !h.repoService.HasViewerRole(repository, user) {
			respondError(c, http.StatusNotFound, "Repository not found")
			return nil
		}
	}

	starred, err := h.starService.IsStarred(owner, repo, user.UserName)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to check star status")
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"starred": starred})
	return nil
}

func (h *StarHandler) GetStarCount(c *fiber.Ctx) error {
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

	count, err := h.starService.GetStarCount(owner, repo)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to get star count")
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"count": count})
	return nil
}

func (h *StarHandler) GetStargazers(c *fiber.Ctx) error {
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

	stargazers, err := h.starService.GetStargazers(owner, repo)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to get stargazers")
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"stargazers": stargazers})
	return nil
}

func (h *StarHandler) GetStarredRepos(c *fiber.Ctx) error {
	username := c.Params("username")

	repos, err := h.starService.GetStarredRepos(username)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to get starred repos")
		return nil
	}

	// Filter out private repos that the current user cannot access
	viewer := contextutil.GetUserFromContext(c)
	visible := make([]*model.Repository, 0, len(repos))
	for _, repo := range repos {
		if !repo.IsPrivate {
			visible = append(visible, repo)
		} else if viewer != nil && h.repoService.HasViewerRole(repo, viewer) {
			visible = append(visible, repo)
		}
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"repos": visible})
	return nil
}

type WatchHandler struct {
	watchService *service.WatchService
	repoService  *service.RepositoryService
}

func NewWatchHandler(watchService *service.WatchService, repoService *service.RepositoryService) *WatchHandler {
	return &WatchHandler{
		watchService: watchService,
		repoService:  repoService,
	}
}

// WatchRepository starts watching a repository
func (h *WatchHandler) WatchRepository(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	owner := c.Params("owner")
	repo := c.Params("repo")

	// Check repository access for private repos
	repository, err := h.repoService.GetRepository(owner, repo)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	if repository.IsPrivate {
		if !h.repoService.HasViewerRole(repository, user) {
			respondError(c, http.StatusNotFound, "Repository not found")
			return nil
		}
	}

	var req struct {
		Notification bool `json:"notification"`
	}
	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request")
		return nil
	}

	if err := h.watchService.WatchRepository(owner, repo, user.UserName, req.Notification); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to watch repository")
		return nil
	}

	c.Status(http.StatusNoContent)
	return nil
}

// UnwatchRepository stops watching a repository
func (h *WatchHandler) UnwatchRepository(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	owner := c.Params("owner")
	repo := c.Params("repo")

	// Check repository access for private repos
	repository, err := h.repoService.GetRepository(owner, repo)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	if repository.IsPrivate {
		if !h.repoService.HasViewerRole(repository, user) {
			respondError(c, http.StatusNotFound, "Repository not found")
			return nil
		}
	}

	if err := h.watchService.UnwatchRepository(owner, repo, user.UserName); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to unwatch repository")
		return nil
	}

	c.Status(http.StatusNoContent)
	return nil
}

// IsWatching checks if current user is watching a repository
func (h *WatchHandler) IsWatching(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		c.Status(http.StatusOK).JSON(fiber.Map{"watching": false})
		return nil
	}

	owner := c.Params("owner")
	repo := c.Params("repo")

	// Check repository access for private repos
	repository, err := h.repoService.GetRepository(owner, repo)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	if repository.IsPrivate {
		if !h.repoService.HasViewerRole(repository, user) {
			respondError(c, http.StatusNotFound, "Repository not found")
			return nil
		}
	}

	watching, err := h.watchService.IsWatching(owner, repo, user.UserName)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to check watch status")
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"watching": watching})
	return nil
}

// GetWatchCount returns the number of watchers for a repository
func (h *WatchHandler) GetWatchCount(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	// Check repository access for private repos
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

	count, err := h.watchService.GetWatchCount(owner, repo)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to get watch count")
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"count": count})
	return nil
}

// GetWatchers returns all users watching a repository
func (h *WatchHandler) GetWatchers(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	// Check repository access for private repos
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

	watchers, err := h.watchService.GetWatchers(owner, repo)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to get watchers")
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"watchers": watchers})
	return nil
}
