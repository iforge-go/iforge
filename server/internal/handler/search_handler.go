package handler

import (
	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/service"
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type SearchHandler struct {
	searchService  *service.SearchService
	repoService    *service.RepositoryService
	accountService *service.AccountService
}

func NewSearchHandler(searchService *service.SearchService, repoService *service.RepositoryService, accountService *service.AccountService) *SearchHandler {
	return &SearchHandler{
		searchService:  searchService,
		repoService:    repoService,
		accountService: accountService,
	}
}

func (h *SearchHandler) SearchIssues(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	query := c.Query("q")
	if query == "" {
		query = "is:open"
	}

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

	searchQuery := h.searchService.ParseSearchQuery(query)
	searchQuery.Owner = owner
	searchQuery.Repo = repo

	limit, _ := strconv.Atoi(c.Query("limit", "25"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	searchQuery.Limit = limit
	searchQuery.Offset = offset

	result, err := h.searchService.SearchIssues(searchQuery)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	var userNames []string
	seen := make(map[string]bool)
	for _, issue := range result.Issues {
		for _, name := range issue.ParticipantUserNames {
			if name != "" && !seen[name] {
				seen[name] = true
				userNames = append(userNames, name)
			}
		}
	}
	avatarInfo, _ := h.accountService.GetUserAvatarInfo(userNames)
	participants := h.accountService.BuildParticipants(avatarInfo, userNames)

	c.Status(http.StatusOK).JSON(fiber.Map{
		"issues":       result.Issues,
		"total":        result.Total,
		"limit":        limit,
		"offset":       offset,
		"participants": participants,
	})
	return nil
}

func (h *SearchHandler) SearchAllIssues(c *fiber.Ctx) error {
	query := c.Query("q")

	if query == "" {
		respondError(c, http.StatusBadRequest, "Query parameter 'q' is required")
		return nil
	}

	searchQuery := h.searchService.ParseSearchQuery(query)

	limit, _ := strconv.Atoi(c.Query("limit", "25"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	searchQuery.Limit = limit
	searchQuery.Offset = offset

	result, err := h.searchService.SearchIssues(searchQuery)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	var userNames []string
	seen := make(map[string]bool)
	for _, issue := range result.Issues {
		for _, name := range issue.ParticipantUserNames {
			if name != "" && !seen[name] {
				seen[name] = true
				userNames = append(userNames, name)
			}
		}
	}
	avatarInfo, _ := h.accountService.GetUserAvatarInfo(userNames)
	participants := h.accountService.BuildParticipants(avatarInfo, userNames)

	c.Status(http.StatusOK).JSON(fiber.Map{
		"issues":       result.Issues,
		"total":        result.Total,
		"limit":        limit,
		"offset":       offset,
		"participants": participants,
	})
	return nil
}
