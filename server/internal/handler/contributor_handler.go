package handler

import (
	"net/http"

	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type ContributorHandler struct {
	contributorService *service.ContributorService
	repoService        *service.RepositoryService
}

func NewContributorHandler(contributorService *service.ContributorService, repoService *service.RepositoryService) *ContributorHandler {
	return &ContributorHandler{
		contributorService: contributorService,
		repoService:        repoService,
	}
}

func (h *ContributorHandler) GetContributors(c *fiber.Ctx) error {
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

	contributors, err := h.contributorService.GetContributors(owner, repo)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to get contributors")
		return nil
	}

	c.Status(http.StatusOK).JSON(contributors)
	return nil
}
