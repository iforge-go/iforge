package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/event"
	"iforge/iforge/internal/git"
	"iforge/iforge/internal/model"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type IssueHandler struct {
	issueService     *service.IssueService
	repoService      *service.RepositoryService
	gitClient        *git.Client
	labelService     *service.LabelService
	milestoneService *service.MilestoneService
	eventBus         *event.Bus
	accountService   *service.AccountService
}

func NewIssueHandler(issueService *service.IssueService, repoService *service.RepositoryService, gitClient *git.Client, labelService *service.LabelService, milestoneService *service.MilestoneService, eventBus *event.Bus, accountService *service.AccountService) *IssueHandler {
	return &IssueHandler{
		issueService:     issueService,
		repoService:      repoService,
		gitClient:        gitClient,
		labelService:     labelService,
		milestoneService: milestoneService,
		eventBus:         eventBus,
		accountService:   accountService,
	}
}

type CreateIssueRequest struct {
	Title       string   `json:"title" validate:"required"`
	Body        *string  `json:"body"`
	Assignees   []string `json:"assignees"`
	MilestoneID *int     `json:"milestone_id"`
	Labels      []int    `json:"labels"`
}

// @Router /api/v1/repos/{owner}/{repo}/issues [get]
func (h *IssueHandler) ListIssues(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	_, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}

	state := c.Query("state", "open")
	closed := state == "closed"

	_, perPage, offset := parsePaginationParams(c)

	issues, err := h.issueService.ListIssuesWithLabels(owner, repoName, closed, perPage, offset)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list issues")
		return nil
	}

	// Collect openedUserName for all issues
	var userNames []string
	seen := make(map[string]bool)
	for _, issue := range issues {
		if !seen[issue.OpenedUserName] {
			seen[issue.OpenedUserName] = true
			userNames = append(userNames, issue.OpenedUserName)
		}
	}
	participants := h.accountService.BuildParticipantsForUsers(userNames)

	c.Status(http.StatusOK).JSON(fiber.Map{
		"issues":       issues,
		"participants": participants,
	})
	return nil
}

// ListIssueAuthors returns distinct users who have opened issues in a repo.
// @Summary List issue authors
// @Description Get distinct users who have authored issues in a repository
// @Tags issues
// @Produce json
// @Param owner path string true "Repository owner"
// @Param repo path string true "Repository name"
// @Success 200 {array} model.Account "Distinct issue authors"
// @Failure 404 {object} map[string]string "Repository not found"
// @Router /api/v1/repos/{owner}/{repo}/issues/authors [get]
func (h *IssueHandler) ListIssueAuthors(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	_, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}

	authors, err := h.issueService.GetIssueAuthors(owner, repoName, c.Query("type") == "mr")
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list issue authors")
		return nil
	}

	c.Status(http.StatusOK).JSON(authors)
	return nil
}

// @Router /api/v1/repos/{owner}/{repo}/issues/assignees [get]
func (h *IssueHandler) ListIssueAssignees(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	_, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}

	assignees, err := h.issueService.GetIssueAssignees(owner, repoName, c.Query("type") == "mr")
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list issue assignees")
		return nil
	}

	c.Status(http.StatusOK).JSON(assignees)
	return nil
}

// @Router /api/v1/repos/{owner}/{repo}/issues/{id} [get]
func (h *IssueHandler) GetIssue(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")
	issueID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid issue number")
		return nil
	}

	_, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}

	issue, err := h.issueService.GetIssue(owner, repoName, issueID)
	if err != nil {
		respondError(c, http.StatusNotFound, "Issue not found")
		return nil
	}

	labels, _ := h.labelService.GetIssueLabels(owner, repoName, issueID)
	if labels == nil {
		labels = []*model.Label{}
	}

	assignees, _ := h.issueService.GetAssignees(owner, repoName, issueID)
	if assignees == nil {
		assignees = []*model.Account{}
	}

	avatarInfo, _ := h.accountService.GetUserAvatarInfo([]string{issue.OpenedUserName})
	info := avatarInfo[issue.OpenedUserName]

	c.Status(http.StatusOK).JSON(fiber.Map{
		"userName":           issue.UserName,
		"repositoryName":     issue.RepositoryName,
		"issueId":            issue.IssueID,
		"openedUserName":     issue.OpenedUserName,
		"openedUserImage":    info.Image,
		"openedUserFullName": info.FullName,
		"milestoneId":        issue.MilestoneID,
		"priorityId":         issue.PriorityID,
		"title":              issue.Title,
		"content":            issue.Content,
		"closed":             issue.Closed,
		"closeReason":        issue.CloseReason,
		"locked":             issue.Locked,
		"registeredDate":     issue.RegisteredDate,
		"updatedDate":        issue.UpdatedDate,
		"isMergeRequest":     issue.IsMergeRequest,
		"labels":             labels,
		"assignees":          assignees,
	})
	return nil
}

// @Router /api/v1/repos/{owner}/{repo}/issues [post]
func (h *IssueHandler) CreateIssue(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	owner := c.Params("owner")
	repoName := c.Params("repo")

	repository, err := h.repoService.GetRepository(owner, repoName)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	if !h.repoService.IsIssueEditable(repository, user) {
		respondError(c, http.StatusForbidden, "Forbidden: You cannot create issues in this repository")
		return nil
	}

	var req CreateIssueRequest
	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body")
		return nil
	}

	issue, err := h.issueService.CreateIssue(
		owner, repoName, user.UserName, req.Title, req.Body,
		req.MilestoneID, nil, false,
	)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to create issue")
		return nil
	}

	for _, labelID := range req.Labels {
		h.issueService.AddLabel(owner, repoName, issue.IssueID, labelID)
	}

	if len(req.Assignees) > 0 {
		h.issueService.SetAssignees(owner, repoName, issue.IssueID, req.Assignees)
	}

	h.eventBus.Publish(event.NewIssueEvent(repository, user, "opened", issue))

	c.Status(http.StatusCreated).JSON(issue)
	return nil
}

func (h *IssueHandler) UpdateIssue(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")
	issueID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid issue number")
		return nil
	}

	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	repository, err := h.repoService.GetRepository(owner, repoName)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	issue, err := h.issueService.GetIssue(owner, repoName, issueID)
	if err != nil {
		respondError(c, http.StatusNotFound, "Issue not found")
		return nil
	}

	// Check permission based on what's being updated
	// Issue author can edit title/body, Developer+ can close/reopen/assign milestones/labels
	isIssueAuthor := issue.OpenedUserName == user.UserName
	hasDeveloperRole := h.repoService.HasMemberRole(repository, user)

	if !isIssueAuthor && !hasDeveloperRole {
		respondError(c, http.StatusForbidden, "Forbidden: Developer access required")
		return nil
	}

	// Save original state before update for webhook event
	originalClosed := issue.Closed

	bodyBytes := c.Body()

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(bodyBytes, &raw); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body")
		return nil
	}

	// Title and body can be edited by issue author or Developer+
	if v, ok := raw["title"]; ok {
		if !isIssueAuthor && !hasDeveloperRole {
			respondError(c, http.StatusForbidden, "Forbidden: Cannot edit title")
			return nil
		}
		var title string
		if json.Unmarshal(v, &title) == nil {
			issue.Title = title
		}
	}
	if v, ok := raw["body"]; ok {
		if !isIssueAuthor && !hasDeveloperRole {
			respondError(c, http.StatusForbidden, "Forbidden: Cannot edit body")
			return nil
		}
		var body string
		if json.Unmarshal(v, &body) == nil {
			issue.Content = &body
		}
	}

	// State changes (close/reopen) require Developer role
	oldMilestoneID := issue.MilestoneID
	if v, ok := raw["milestoneId"]; ok {
		if !hasDeveloperRole {
			respondError(c, http.StatusForbidden, "Forbidden: Developer access required to assign milestones")
			return nil
		}
		if string(v) == "null" {
			issue.MilestoneID = nil
		} else {
			var mid int
			if json.Unmarshal(v, &mid) == nil {
				issue.MilestoneID = &mid
			}
		}
	}
	var req struct {
		State       string  `json:"state"`
		CloseReason *string `json:"close_reason"`
	}
	if json.Unmarshal(bodyBytes, &req) == nil {
		if state := req.State; state != "" {
			if !hasDeveloperRole {
				respondError(c, http.StatusForbidden, "Forbidden: Developer access required to change state")
				return nil
			}
			issue.Closed = state == "closed"

			// Handle close reason when closing an issue
			if issue.Closed {
				if req.CloseReason != nil && *req.CloseReason != "" {
					issue.CloseReason = req.CloseReason
				} else {
					// Default to "completed" if no reason provided
					defaultReason := "completed"
					issue.CloseReason = &defaultReason
				}
			} else {
				// Clear close reason when reopening
				issue.CloseReason = nil
			}

			// State change notification + activity are handled by subscribers
			// via the TypeIssues event published below (action=closed/reopened).
		}
	}

	// Record milestone change activity
	milestoneChanged := (oldMilestoneID == nil) != (issue.MilestoneID == nil) ||
		(oldMilestoneID != nil && issue.MilestoneID != nil && *oldMilestoneID != *issue.MilestoneID)
	if milestoneChanged {
		var action, content string
		if issue.MilestoneID == nil {
			action = "remove_milestone"
			// Get old milestone title
			if oldMilestoneID != nil {
				if ms, err := h.milestoneService.GetMilestone(owner, repoName, *oldMilestoneID); err == nil {
					content = ms.Title
				}
			}
		} else {
			action = "add_milestone"
			// Get new milestone title
			if ms, err := h.milestoneService.GetMilestone(owner, repoName, *issue.MilestoneID); err == nil {
				content = ms.Title
			}
		}
		h.issueService.AddComment(owner, repoName, issueID, user.UserName, action, content)
	}

	if err := h.issueService.UpdateIssue(issue); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to update issue")
		return nil
	}

	// Publish webhook event for state changes
	if req.State != "" {
		action := "edited"
		if issue.Closed && !originalClosed {
			action = "closed"
		} else if !issue.Closed && originalClosed {
			action = "reopened"
		}

		repo, err := h.repoService.GetRepository(owner, repoName)
		if err == nil {
			h.eventBus.Publish(event.NewIssueEvent(repo, user, action, issue))
		}
	}

	c.Status(http.StatusOK).JSON(issue)
	return nil
}

// DeleteIssue deletes an issue
// Aligned with GitBucket: requires Owner role (HasOwnerRole)
func (h *IssueHandler) DeleteIssue(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")
	issueID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid issue ID")
		return nil
	}

	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	repository, err := h.repoService.GetRepository(owner, repoName)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	// Check if user has owner role (aligned with GitBucket)
	if !h.repoService.HasOwnerRole(repository, user) {
		respondError(c, http.StatusForbidden, "Forbidden: Owner access required to delete issues")
		return nil
	}

	if err := h.issueService.DeleteIssue(owner, repoName, issueID); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to delete issue")
		return nil
	}

	c.Status(http.StatusNoContent).SendString("")
	return nil
}

// DeleteComment deletes an issue comment
// Aligned with GitBucket: requires Owner role OR comment author
func (h *IssueHandler) DeleteComment(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")
	commentID, err := strconv.Atoi(c.Params("commentId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid comment ID")
		return nil
	}

	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	repository, err := h.repoService.GetRepository(owner, repoName)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	// Get the comment
	comment, err := h.issueService.GetComment(owner, repoName, commentID)
	if err != nil {
		respondError(c, http.StatusNotFound, "Comment not found")
		return nil
	}

	// Check if user is comment author OR has owner role (aligned with GitBucket)
	isCommentAuthor := comment.CommentedUserName == user.UserName
	hasOwnerRole := h.repoService.HasOwnerRole(repository, user)

	if !isCommentAuthor && !hasOwnerRole {
		respondError(c, http.StatusForbidden, "Forbidden: Only comment author or repository owner can delete comments")
		return nil
	}

	if err := h.issueService.DeleteComment(owner, repoName, commentID); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to delete comment")
		return nil
	}

	c.Status(http.StatusNoContent).SendString("")
	return nil
}

// UpdateComment updates an issue comment
// Aligned with GitBucket: requires Developer role OR comment author
func (h *IssueHandler) UpdateComment(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")
	commentID, err := strconv.Atoi(c.Params("commentId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid comment ID")
		return nil
	}

	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	repository, err := h.repoService.GetRepository(owner, repoName)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	// Get the comment
	comment, err := h.issueService.GetComment(owner, repoName, commentID)
	if err != nil {
		respondError(c, http.StatusNotFound, "Comment not found")
		return nil
	}

	// Check if user is comment author OR has developer role (aligned with GitBucket)
	isCommentAuthor := comment.CommentedUserName == user.UserName
	hasDeveloperRole := h.repoService.HasMemberRole(repository, user)

	if !isCommentAuthor && !hasDeveloperRole {
		respondError(c, http.StatusForbidden, "Forbidden: Only comment author or developer can edit comments")
		return nil
	}

	// Parse request body
	var req struct {
		Content string `json:"content"`
	}
	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body")
		return nil
	}

	// Update comment
	comment.Content = req.Content
	comment.UpdatedDate = time.Now()

	if err := h.issueService.UpdateComment(comment); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to update comment")
		return nil
	}

	c.Status(http.StatusOK).JSON(comment)
	return nil
}

// AddAssignee assigns a user to an issue
// @Summary Add assignee
// @Description Assign a user to an issue
// @Tags issues
// @Produce json
// @Param owner path string true "Repository owner"
// @Param repo path string true "Repository name"
// @Param id path int true "Issue ID"
// @Param username path string true "Username to assign"
// @Success 200 {object} map[string]interface{} "Updated assignees list"
// @Failure 400 {object} map[string]string "Invalid issue number or username"
// @Failure 403 {object} map[string]string "Write permission required"
// @Failure 404 {object} map[string]string "Issue not found"
// @Security BearerAuth
// @Router /api/v1/repos/{owner}/{repo}/issues/{id}/assignees/{username} [post]
func (h *IssueHandler) AddAssignee(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")
	issueID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid issue number")
		return nil
	}
	username := c.Params("username")
	if username == "" {
		respondError(c, http.StatusBadRequest, "username is required")
		return nil
	}

	if _, _, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repoName); !ok {
		return nil
	}

	if _, err := h.issueService.GetIssue(owner, repoName, issueID); err != nil {
		respondError(c, http.StatusNotFound, "Issue not found")
		return nil
	}

	if err := h.issueService.AddAssignee(owner, repoName, issueID, username); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to add assignee")
		return nil
	}

	assignees, _ := h.issueService.GetAssignees(owner, repoName, issueID)
	c.Status(http.StatusOK).JSON(fiber.Map{"assignees": assignees})
	return nil
}

// RemoveAssignee removes an assignee from an issue
// @Summary Remove assignee
// @Description Remove an assignee from an issue
// @Tags issues
// @Produce json
// @Param owner path string true "Repository owner"
// @Param repo path string true "Repository name"
// @Param id path int true "Issue ID"
// @Param username path string true "Username to remove"
// @Success 200 {object} map[string]interface{} "Updated assignees list"
// @Failure 400 {object} map[string]string "Invalid issue number or username"
// @Failure 403 {object} map[string]string "Write permission required"
// @Failure 404 {object} map[string]string "Issue not found"
// @Security BearerAuth
// @Router /api/v1/repos/{owner}/{repo}/issues/{id}/assignees/{username} [delete]
func (h *IssueHandler) RemoveAssignee(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")
	issueID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid issue number")
		return nil
	}
	username := c.Params("username")
	if username == "" {
		respondError(c, http.StatusBadRequest, "username is required")
		return nil
	}

	if _, _, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repoName); !ok {
		return nil
	}

	if _, err := h.issueService.GetIssue(owner, repoName, issueID); err != nil {
		respondError(c, http.StatusNotFound, "Issue not found")
		return nil
	}

	if err := h.issueService.RemoveAssignee(owner, repoName, issueID, username); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to remove assignee")
		return nil
	}

	assignees, _ := h.issueService.GetAssignees(owner, repoName, issueID)
	c.Status(http.StatusOK).JSON(fiber.Map{"assignees": assignees})
	return nil
}

// ListAssignees lists all assignees for an issue
// @Summary List assignees
// @Description List all assignees for an issue
// @Tags issues
// @Produce json
// @Param owner path string true "Repository owner"
// @Param repo path string true "Repository name"
// @Param id path int true "Issue ID"
// @Success 200 {object} map[string]interface{} "Assignees list"
// @Failure 400 {object} map[string]string "Invalid issue number"
// @Failure 404 {object} map[string]string "Repository not found"
// @Security BearerAuth
// @Router /api/v1/repos/{owner}/{repo}/issues/{id}/assignees [get]
func (h *IssueHandler) ListAssignees(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")
	issueID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid issue number")
		return nil
	}

	_, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}

	assignees, err := h.issueService.GetAssignees(owner, repoName, issueID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list assignees")
		return nil
	}
	if assignees == nil {
		assignees = []*model.Account{}
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"assignees": assignees})
	return nil
}

// LockIssue locks an issue to prevent further comments
func (h *IssueHandler) LockIssue(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")
	issueID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid issue number")
		return nil
	}

	user, repo, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}

	issue, err := h.issueService.GetIssue(owner, repoName, issueID)
	if err != nil {
		respondError(c, http.StatusNotFound, "Issue not found")
		return nil
	}

	isOwner := h.repoService.HasOwnerRole(repo, user)
	isAuthor := issue.OpenedUserName == user.UserName

	if !isOwner && !isAuthor {
		respondError(c, http.StatusForbidden, "Insufficient permissions")
		return nil
	}

	err = h.issueService.LockIssue(owner, repoName, issueID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to lock issue")
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Issue locked"})
	return nil
}

// UnlockIssue unlocks an issue to allow comments
func (h *IssueHandler) UnlockIssue(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")
	issueID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid issue number")
		return nil
	}

	user, repo, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}

	issue, err := h.issueService.GetIssue(owner, repoName, issueID)
	if err != nil {
		respondError(c, http.StatusNotFound, "Issue not found")
		return nil
	}

	isOwner := h.repoService.HasOwnerRole(repo, user)
	isAuthor := issue.OpenedUserName == user.UserName

	if !isOwner && !isAuthor {
		respondError(c, http.StatusForbidden, "Insufficient permissions")
		return nil
	}

	err = h.issueService.UnlockIssue(owner, repoName, issueID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to unlock issue")
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Issue unlocked"})
	return nil
}

// CreateCommentRequest represents a create comment request
type CreateCommentRequest struct {
	Body string `json:"body" validate:"required"`
}

// CreateComment creates a comment on an issue
func (h *IssueHandler) CreateComment(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	owner := c.Params("owner")
	repoName := c.Params("repo")
	issueID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid issue number")
		return nil
	}

	// Check repository access
	repository, err := h.repoService.GetRepository(owner, repoName)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	if !h.repoService.HasViewerRole(repository, user) {
		respondError(c, http.StatusForbidden, "Forbidden: No repository access")
		return nil
	}

	var req CreateCommentRequest
	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body")
		return nil
	}

	issue, err := h.issueService.GetIssue(owner, repoName, issueID)
	if err != nil {
		respondError(c, http.StatusNotFound, "Issue not found")
		return nil
	}

	if issue.Locked && !h.repoService.HasMemberRole(repository, user) {
		respondError(c, http.StatusForbidden, "Issue is locked")
		return nil
	}

	comment, err := h.issueService.AddComment(owner, repoName, issueID, user.UserName, "comment", req.Body)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to create comment")
		return nil
	}

	// Publish event: NotificationSubscriber fans out to author + past commenters,
	// ActivitySubscriber records issue_comment activity,
	// WebhookSubscriber delivers external webhook.
	h.eventBus.Publish(event.NewIssueCommentEvent(repository, user, "created", issue, comment))

	c.Status(http.StatusCreated).JSON(comment)
	return nil
}

// ListComments lists comments on an issue
func (h *IssueHandler) ListComments(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")
	issueID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid issue number")
		return nil
	}

	_, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}

	comments, err := h.issueService.GetComments(owner, repoName, issueID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list comments")
		return nil
	}

	// Sideload: batch fetch unique commenter avatar info
	seen := make(map[string]bool)
	var userNames []string
	for _, c := range comments {
		if !seen[c.CommentedUserName] {
			seen[c.CommentedUserName] = true
			userNames = append(userNames, c.CommentedUserName)
		}
	}
	participants := h.accountService.BuildParticipantsForUsers(userNames)

	c.Status(http.StatusOK).JSON(fiber.Map{
		"comments":     comments,
		"participants": participants,
	})
	return nil
}

// GetIssueTemplates retrieves issue templates from the repository
func (h *IssueHandler) GetIssueTemplates(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	_, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}

	templates, err := h.gitClient.GetIssueTemplates(owner, repoName)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to get issue templates")
		return nil
	}

	c.Status(http.StatusOK).JSON(templates)
	return nil
}

// BatchUpdateIssues batch updates multiple issues
// POST /repos/:owner/:repo/issues/batch
func (h *IssueHandler) BatchUpdateIssues(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	// Check write permission
	_, _, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}

	var req struct {
		IssueIDs       []int `json:"issueIds" validate:"required"`
		Closed         *bool `json:"closed"`
		MilestoneID    *int  `json:"milestoneId"`
		PriorityID     *int  `json:"priorityId"`
		AddLabelIDs    []int `json:"addLabelIds"`
		RemoveLabelIDs []int `json:"removeLabelIds"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	if len(req.IssueIDs) == 0 {
		respondError(c, http.StatusBadRequest, "No issues selected")
		return nil
	}

	if err := h.issueService.BatchUpdateIssues(
		owner, repoName,
		req.IssueIDs,
		req.Closed,
		req.MilestoneID,
		req.PriorityID,
		req.AddLabelIDs,
		req.RemoveLabelIDs,
	); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{
		"message": "Batch update completed",
		"updated": len(req.IssueIDs),
	})
	return nil
}
