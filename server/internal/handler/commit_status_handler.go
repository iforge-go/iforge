package handler

import (
	"net/http"

	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type CommitStatusHandler struct {
	commitStatusService *service.CommitStatusService
	repoService         *service.RepositoryService
}

func NewCommitStatusHandler(commitStatusService *service.CommitStatusService, repoService *service.RepositoryService) *CommitStatusHandler {
	return &CommitStatusHandler{
		commitStatusService: commitStatusService,
		repoService:         repoService,
	}
}

type CreateStatusRequest struct {
	State       string  `json:"state" validate:"required,oneof=pending success failure error"`
	Context     string  `json:"context"`
	TargetURL   *string `json:"target_url"`
	Description *string `json:"description"`
}

func (h *CommitStatusHandler) CreateStatus(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")
	commitID := c.Params("commit")

	user, _, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}

	var req CreateStatusRequest
	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	context := req.Context
	if context == "" {
		context = "default"
	}

	status, err := h.commitStatusService.CreateOrUpdateStatus(
		owner, repoName, commitID, context, req.State,
		req.TargetURL, req.Description, user.UserName,
	)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to create status")
		return nil
	}

	c.Status(http.StatusCreated).JSON(status)
	return nil
}

func (h *CommitStatusHandler) ListStatuses(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")
	commitID := c.Params("commit")

	repository, err := h.repoService.GetRepository(owner, repoName)
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

	statuses, err := h.commitStatusService.GetStatusesForCommit(owner, repoName, commitID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list statuses")
		return nil
	}

	c.Status(http.StatusOK).JSON(statuses)
	return nil
}

func (h *CommitStatusHandler) GetCombinedStatus(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")
	commitID := c.Params("commit")

	repository, err := h.repoService.GetRepository(owner, repoName)
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

	combinedStatus, err := h.commitStatusService.GetCombinedStatus(owner, repoName, commitID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to get combined status")
		return nil
	}

	c.Status(http.StatusOK).JSON(combinedStatus)
	return nil
}
