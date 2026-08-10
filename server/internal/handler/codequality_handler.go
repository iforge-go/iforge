package handler

import (
	"net/http"

	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type CodeQualityHandler struct {
	codeQualityService *service.CodeQualityService
	repoService        *service.RepositoryService
}

func NewCodeQualityHandler(codeQualityService *service.CodeQualityService, repoService *service.RepositoryService) *CodeQualityHandler {
	return &CodeQualityHandler{
		codeQualityService: codeQualityService,
		repoService:        repoService,
	}
}

func (h *CodeQualityHandler) AnalyzeCodeQuality(c *fiber.Ctx) error {
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

	report, err := h.codeQualityService.AnalyzeCodeQuality(owner, repo)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to analyze code quality")
		return nil
	}

	c.Status(http.StatusOK).JSON(report)
	return nil
}
