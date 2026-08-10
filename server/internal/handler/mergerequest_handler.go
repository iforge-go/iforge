package handler

import (
	"fmt"
	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/event"
	"iforge/iforge/internal/git"
	"iforge/iforge/internal/service"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type MergeRequestHandler struct {
	mrService           *service.MergeRequestService
	issueService        *service.IssueService
	gitClient           *git.Client
	notificationService *service.NotificationService
	repoService         *service.RepositoryService
	eventBus            *event.Bus
	accountService      *service.AccountService
}

func NewMergeRequestHandler(
	mrService *service.MergeRequestService,
	issueService *service.IssueService,
	gitClient *git.Client,
	notificationService *service.NotificationService,
	repoService *service.RepositoryService,
	eventBus *event.Bus,
	accountService *service.AccountService,
) *MergeRequestHandler {
	return &MergeRequestHandler{
		mrService:           mrService,
		issueService:        issueService,
		gitClient:           gitClient,
		notificationService: notificationService,
		repoService:         repoService,
		eventBus:            eventBus,
		accountService:      accountService,
	}
}

func (h *MergeRequestHandler) MergeMergeRequest(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	issueIdStr := c.Params("id")

	issueId, err := strconv.Atoi(issueIdStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid issue ID")
		return nil
	}

	// Get current user
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	// Check repository access (Developer role or higher required to merge)
	repository, err := h.repoService.GetRepository(owner, repo)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	if !h.repoService.HasMemberRole(repository, user) {
		respondError(c, http.StatusForbidden, "Forbidden: Developer access required to merge")
		return nil
	}

	var req struct {
		Strategy string `json:"strategy"` // merge-commit, squash, rebase
		Message  string `json:"message"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	// Use user info for author
	authorName := user.UserName
	authorEmail := user.MailAddress

	strategy := service.MergeStrategy(req.Strategy)
	if strategy == "" {
		strategy = service.MergeStrategyMergeCommit
	}

	mergeCommitId, err := h.mrService.MergeMergeRequestWithStrategy(
		owner, repo, issueId, strategy, req.Message, authorName, authorEmail,
	)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	// Invalidate cache for the target repository (new merge commit changes branches, commits, files)
	git.InvalidateRepositoryCache(owner, repo)
	// For cross-repo MRs, also invalidate the source repository's cache
	if mr, err := h.mrService.GetMergeRequest(owner, repo, issueId); err == nil {
		if mr.RequestUserName != owner || mr.RequestRepositoryName != repo {
			git.InvalidateRepositoryCache(mr.RequestUserName, mr.RequestRepositoryName)
		}
	}

	// Publish webhook event for merge
	mr, err := h.mrService.GetMergeRequest(owner, repo, issueId)
	if err == nil {
		issue, err := h.issueService.GetIssue(owner, repo, issueId)
		if err == nil {
			h.eventBus.Publish(event.NewMergeRequestEvent(repository, user, "merged", issue, mr))
		}
	}

	c.Status(http.StatusOK).JSON(fiber.Map{
		"message":     fmt.Sprintf("merge request #%d merged successfully", issueId),
		"mergeCommit": mergeCommitId,
		"strategy":    string(strategy),
	})
	return nil
}

func (h *MergeRequestHandler) GetMergeRequest(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	issueIdStr := c.Params("id")

	issueId, err := strconv.Atoi(issueIdStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid issue ID")
		return nil
	}

	_, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	mr, err := h.mrService.GetMergeRequest(owner, repo, issueId)
	if err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(mr)
	return nil
}

func (h *MergeRequestHandler) ListMergeRequests(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	_, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	state := c.Query("state", "")
	limit, _ := strconv.Atoi(c.Query("limit", "25"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	mrs, err := h.mrService.ListMergeRequests(owner, repo, state, limit, offset)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	var userNames []string
	seen := make(map[string]bool)
	for _, mr := range mrs {
		for _, name := range []string{mr.UserName, mr.RequestUserName} {
			if name != "" && !seen[name] {
				seen[name] = true
				userNames = append(userNames, name)
			}
		}
	}
	avatarInfo, _ := h.accountService.GetUserAvatarInfo(userNames)
	participants := h.accountService.BuildParticipants(avatarInfo, userNames)

	c.Status(http.StatusOK).JSON(fiber.Map{
		"mergeRequests": mrs,
		"participants":  participants,
	})
	return nil
}

func (h *MergeRequestHandler) CreateMergeRequest(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	// Check repository access
	repository, err := h.repoService.GetRepository(owner, repo)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	// Check IssuesOption permission (aligned with GitBucket's isIssueEditable,
	// shared with issue creation since GitBucket treats PR as a special issue)
	if !h.repoService.IsIssueEditable(repository, user) {
		respondError(c, http.StatusForbidden, "Forbidden: You cannot create merge requests in this repository")
		return nil
	}

	var req struct {
		Title     string `json:"title" validate:"required"`
		Body      string `json:"body"`
		Head      string `json:"head" validate:"required"`
		Base      string `json:"base" validate:"required"`
		HeadUser  string `json:"headUser"`
		HeadRepo  string `json:"headRepo"`
		Assignee  string `json:"assignee"`
		Milestone int    `json:"milestone"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	// Determine source repository (cross-repo or same-repo)
	headUser := req.HeadUser
	headRepo := req.HeadRepo
	isCrossRepo := headUser != "" && headRepo != ""

	if isCrossRepo {
		// Validate fork repository exists and is a fork of target repository
		forkRepo, err := h.repoService.GetRepository(headUser, headRepo)
		if err != nil {
			respondError(c, http.StatusNotFound, "Source repository not found")
			return nil
		}

		// Check if it's actually a fork of the target repository
		if forkRepo.OriginUserName == nil || forkRepo.OriginRepositoryName == nil ||
			*forkRepo.OriginUserName != owner || *forkRepo.OriginRepositoryName != repo {
			respondError(c, http.StatusBadRequest, "Source repository is not a fork of target repository")
			return nil
		}

		// Check user has read access to fork repository
		if !h.repoService.HasViewerRole(forkRepo, user) {
			respondError(c, http.StatusForbidden, "No read access to source repository")
			return nil
		}
	} else {
		// Same-repo MR: default to target repository
		headUser = owner
		headRepo = repo
	}

	// Create MR record first (to get issueId)
	mr, err := h.mrService.CreateMergeRequest(
		owner, repo, headUser, headRepo, req.Base, req.Head,
		req.Title, req.Body, "", "", false, user.UserName,
	)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	// For cross-repo MR, fetch the fork branch to refs/mr/{issueId}/head
	if isCrossRepo {
		mrRef := fmt.Sprintf("refs/mr/%d/head", mr.IssueID)
		err = h.gitClient.FetchBranch(
			owner, repo, // target (base) repository
			headUser, headRepo, // source (head/fork) repository
			req.Head, // source branch
			mrRef,    // target ref
		)
		if err != nil {
			respondError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to fetch source branch: %v", err))
			return nil
		}

		// Compare base branch with the fetched ref
		compareResult, err := h.gitClient.Compare(owner, repo, req.Base, mrRef)
		if err != nil {
			_ = h.gitClient.CleanupMRRef(owner, repo, mr.IssueID)
			respondError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to compare branches: %v", err))
			return nil
		}

		// Update MR with commit range
		commitIdFrom := ""
		commitIdTo := ""
		if len(compareResult.Commits) > 0 {
			commitIdFrom = compareResult.Commits[0].ID
			commitIdTo = compareResult.Commits[len(compareResult.Commits)-1].ID
		}

		err = h.mrService.UpdateMergeRequest(owner, repo, mr.IssueID, map[string]interface{}{
			"commit_id_from": commitIdFrom,
			"commit_id_to":   commitIdTo,
		})
		if err != nil {
			_ = h.gitClient.CleanupMRRef(owner, repo, mr.IssueID)
			respondError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to update MR: %v", err))
			return nil
		}
	} else {
		// Same-repo MR: use existing logic
		compareResult, err := h.gitClient.Compare(owner, repo, req.Base, req.Head)
		if err != nil {
			respondError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to compare branches: %v", err))
			return nil
		}

		commitIdFrom := ""
		commitIdTo := ""
		if len(compareResult.Commits) > 0 {
			commitIdFrom = compareResult.Commits[0].ID
			commitIdTo = compareResult.Commits[len(compareResult.Commits)-1].ID
		}

		err = h.mrService.UpdateMergeRequest(owner, repo, mr.IssueID, map[string]interface{}{
			"commit_id_from": commitIdFrom,
			"commit_id_to":   commitIdTo,
		})
		if err != nil {
			respondError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to update MR: %v", err))
			return nil
		}
	}

	// Get the associated issue for webhook event
	issue, err := h.issueService.GetIssue(owner, repo, mr.IssueID)
	if err == nil {
		// Publish webhook event
		h.eventBus.Publish(event.NewMergeRequestEvent(repository, user, "opened", issue, mr))
	}

	c.Status(http.StatusCreated).JSON(mr)
	return nil
}

func (h *MergeRequestHandler) UpdateMergeRequest(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	issueIdStr := c.Params("id")

	issueId, err := strconv.Atoi(issueIdStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid issue ID")
		return nil
	}

	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	// Check repository access
	repository, err := h.repoService.GetRepository(owner, repo)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	var req struct {
		Title     *string `json:"title"`
		Body      *string `json:"body"`
		Assignee  *string `json:"assignee"`
		Milestone *int    `json:"milestone"`
		State     *string `json:"state"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	// Get the issue first
	issue, err := h.issueService.GetIssue(owner, repo, issueId)
	if err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return nil
	}

	// Check if user is the MR author or has developer access.
	// Aligned with GitBucket: isEditableContent = hasDeveloperRole || author.
	// Note: the author can update/close their own MR even without collaborator
	// role on the target repository (e.g., fork contributors who opened the MR).
	// Use OpenedUserName (actual author) instead of UserName (repo owner).
	if issue.OpenedUserName != user.UserName && !h.repoService.HasMemberRole(repository, user) {
		respondError(c, http.StatusForbidden, "Forbidden: Only the author or developers can update")
		return nil
	}

	// Save original state before applying any updates — needed by the
	// webhook event below to detect closed→open / open→closed transitions.
	originalClosed := issue.Closed

	// Update issue fields
	if req.Title != nil {
		issue.Title = *req.Title
	}
	if req.Body != nil {
		issue.Content = req.Body
	}
	if req.State != nil {
		issue.Closed = (*req.State == "closed")
	}
	if req.Milestone != nil {
		issue.MilestoneID = req.Milestone
	}

	if err := h.issueService.UpdateIssue(issue); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	mr, err := h.mrService.GetMergeRequest(owner, repo, issueId)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	// Cleanup temporary ref when MR is closed
	if issue.Closed && !originalClosed {
		// Check if it's a cross-repo MR
		isCrossRepo := mr.RequestUserName != owner || mr.RequestRepositoryName != repo
		if isCrossRepo {
			// Cleanup the temporary ref
			if err := h.gitClient.CleanupMRRef(owner, repo, issueId); err != nil {
				// Log error but don't fail the request
				fmt.Printf("Warning: failed to cleanup MR ref: %v\n", err)
			}
		}
	}

	// Publish webhook event for state changes
	if req.State != nil {
		action := "edited"
		if issue.Closed && !originalClosed {
			action = "closed"
		} else if !issue.Closed && originalClosed {
			action = "reopened"
		}

		h.eventBus.Publish(event.NewMergeRequestEvent(repository, user, action, issue, mr))
	}

	c.Status(http.StatusOK).JSON(mr)
	return nil
}

func (h *MergeRequestHandler) GetMergeRequestDiff(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	issueIdStr := c.Params("id")

	issueId, err := strconv.Atoi(issueIdStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid issue ID")
		return nil
	}

	_, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	mr, err := h.mrService.GetMergeRequest(owner, repo, issueId)
	if err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return nil
	}

	// TODO: Implement GetDiff method in Client
	// For now, return a placeholder message
	diff := fmt.Sprintf("Diff from %s to %s\n\nThis feature needs to be implemented.", mr.CommitIDFrom, mr.CommitIDTo)

	c.Set("Content-Type", "text/plain; charset=utf-8")
	c.Status(http.StatusOK).SendString(diff)
	return nil
}

func (h *MergeRequestHandler) Compare(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	_, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	basehead := c.Params("basehead")

	parts := splitBaseHead(basehead)
	if len(parts) != 2 {
		respondError(c, http.StatusBadRequest, "Invalid compare format. Expected: base...head")
		return nil
	}
	base, head := parts[0], parts[1]

	if base == "" || head == "" {
		respondError(c, http.StatusBadRequest, "Both base and head refs are required")
		return nil
	}

	headUser := c.Query("headUser")
	headRepo := c.Query("headRepo")
	isCrossRepo := headUser != "" && headRepo != ""

	if isCrossRepo {
		tempRef := "refs/compare/temp"

		err := h.gitClient.FetchBranch(
			owner, repo,
			headUser, headRepo,
			head,
			tempRef,
		)
		if err != nil {
			respondError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to fetch source branch: %v", err))
			return nil
		}
		defer h.gitClient.CleanupCompareRef(owner, repo)

		result, err := h.gitClient.Compare(owner, repo, base, tempRef)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return nil
		}

		c.Status(http.StatusOK).JSON(result)
		return nil
	}

	result, err := h.gitClient.Compare(owner, repo, base, head)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(result)
	return nil
}

// splitBaseHead splits "base...head" into [base, head]
func splitBaseHead(s string) []string {
	for i := 0; i <= len(s)-3; i++ {
		if s[i] == '.' && s[i+1] == '.' && s[i+2] == '.' {
			return []string{s[:i], s[i+3:]}
		}
	}
	return nil
}

// GetAheadBehind gets ahead/behind commit count relative to base branch
func (h *MergeRequestHandler) GetAheadBehind(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	_, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	branch := c.Params("branch")
	if decodedBranch, err := url.QueryUnescape(branch); err == nil {
		branch = decodedBranch
	}
	base := c.Query("base", "main")

	result, err := h.gitClient.GetAheadBehind(owner, repo, branch, base)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "reference not found") || strings.Contains(errMsg, "cannot resolve ref") {
			respondError(c, http.StatusNotFound, "Branch not found")
			return nil
		}
		git.GetLogger().Error("GetAheadBehind error", err, map[string]interface{}{
			"owner": owner, "repo": repo, "branch": branch, "base": base,
		})
		respondError(c, http.StatusInternalServerError, errMsg)
		return nil
	}

	c.Status(http.StatusOK).JSON(result)
	return nil
}
