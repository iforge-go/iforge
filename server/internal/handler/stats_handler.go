package handler

import (
	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/service"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

type StatsHandler struct {
	statsService *service.StatsService
	repoService  *service.RepositoryService
}

func NewStatsHandler(statsService *service.StatsService, repoService *service.RepositoryService) *StatsHandler {
	return &StatsHandler{
		statsService: statsService,
		repoService:  repoService,
	}
}

func (h *StatsHandler) GetRepositoryStats(c *fiber.Ctx) error {
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

	stats, err := h.statsService.GetRepositoryStats(owner, repo)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(stats)
	return nil
}
