package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"iforge/iforge/internal/contextutil"
	gitsvc "iforge/iforge/internal/git"
	"iforge/iforge/internal/model"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type ActivityHandler struct {
	activityService *service.ActivityService
	repoService     *service.RepositoryService
	accountService  *service.AccountService
}

func NewActivityHandler(activityService *service.ActivityService, repoService *service.RepositoryService, accountService *service.AccountService) *ActivityHandler {
	return &ActivityHandler{
		activityService: activityService,
		repoService:     repoService,
		accountService:  accountService,
	}
}

func (h *ActivityHandler) ListActivities(c *fiber.Ctx) error {
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

	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	activities, err := h.activityService.GetActivities(owner, repo, limit, offset)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list activities")
		return nil
	}

	userNames := collectUserNames(activities)
	avatarInfo, _ := h.accountService.GetUserAvatarInfo(userNames)
	participants := h.accountService.BuildParticipants(avatarInfo, userNames)

	c.Status(http.StatusOK).JSON(fiber.Map{
		"activities":   activities,
		"participants": participants,
	})
	return nil
}

func (h *ActivityHandler) ListUserActivities(c *fiber.Ctx) error {
	username := c.Params("username")

	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	activities, err := h.activityService.GetUserActivities(username, limit, offset)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list user activities")
		return nil
	}

	userNames := collectUserNames(activities)
	avatarInfo, _ := h.accountService.GetUserAvatarInfo(userNames)
	participants := h.accountService.BuildParticipants(avatarInfo, userNames)

	c.Status(http.StatusOK).JSON(fiber.Map{
		"activities":   activities,
		"participants": participants,
	})
	return nil
}

func (h *ActivityHandler) ListRecentActivities(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	activities, err := h.activityService.GetRecentActivities(user.UserName, limit, offset)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list activities")
		return nil
	}

	if activities == nil {
		activities = []*model.Activity{}
	}

	userNames := collectUserNames(activities)
	avatarInfo, _ := h.accountService.GetUserAvatarInfo(userNames)
	participants := h.accountService.BuildParticipants(avatarInfo, userNames)

	c.Status(http.StatusOK).JSON(fiber.Map{
		"activities":   activities,
		"participants": participants,
	})
	return nil
}

func (h *ActivityHandler) GetUserContributions(c *fiber.Ctx) error {
	username := c.Params("username")

	contributions, err := h.activityService.GetUserContributionDays(username)
	if err != nil {
		gitsvc.GetLogger().Error("GetUserContributions error", err, map[string]interface{}{"username": username})
		respondError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to get contributions: %v", err))
		return nil
	}

	if contributions == nil {
		contributions = []*model.ContributionDay{}
	}

	c.Status(http.StatusOK).JSON(contributions)
	return nil
}

func collectUserNames(activities []*model.Activity) []string {
	var userNames []string
	seen := make(map[string]bool)
	for _, a := range activities {
		if !seen[a.ActivityUserName] {
			seen[a.ActivityUserName] = true
			userNames = append(userNames, a.ActivityUserName)
		}
	}
	return userNames
}
