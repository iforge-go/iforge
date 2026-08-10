package handler

import (
	"fmt"
	"net/http"

	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/event"
	"iforge/iforge/internal/git"
	"iforge/iforge/internal/model"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type RepositoryHandler struct {
	repoService    *service.RepositoryService
	accountService *service.AccountService
	gitClient      *git.Client
	eventBus       *event.Bus
}

func NewRepositoryHandler(repoService *service.RepositoryService, accountService *service.AccountService, gitClient *git.Client, eventBus *event.Bus) *RepositoryHandler {
	return &RepositoryHandler{
		repoService:    repoService,
		accountService: accountService,
		gitClient:      gitClient,
		eventBus:       eventBus,
	}
}

type CreateRepositoryRequest struct {
	Name        string  `json:"name" validate:"required"`
	Description *string `json:"description"`
	Private     bool    `json:"private"`
	AutoInit    bool    `json:"autoInit"`
	Owner       string  `json:"owner"` // Optional: specify owner (user or organization)
}

// @Router /api/v1/repos [get]
func (h *RepositoryHandler) ListRepositories(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	var username *string
	if user != nil {
		username = &user.UserName
	}

	owner := c.Query("owner")
	if owner != "" {
		repos, err := h.repoService.GetRepositoriesByUser(owner, username)
		if err != nil {
			respondError(c, http.StatusInternalServerError, "Failed to list repositories: "+err.Error())
			return nil
		}
		c.Status(http.StatusOK).JSON(repos)
		return nil
	}

	repos, err := h.repoService.GetVisibleRepositories(username, 100)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list repositories: "+err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(repos)
	return nil
}

func (h *RepositoryHandler) GetUserRepositories(c *fiber.Ctx) error {
	username := c.Params("username")
	currentUser := contextutil.GetUserFromContext(c)
	var currentUsername *string
	if currentUser != nil {
		currentUsername = &currentUser.UserName
	}

	repos, err := h.repoService.GetRepositoriesByUser(username, currentUsername)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list repositories")
		return nil
	}

	if repos == nil {
		repos = []*model.Repository{}
	}

	c.Status(http.StatusOK).JSON(repos)
	return nil
}

// GetMyRepositories lists repositories the current user owns or collaborates on
func (h *RepositoryHandler) GetMyRepositories(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	repos, err := h.repoService.GetMyRepositories(user.UserName)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list repositories")
		return nil
	}

	if repos == nil {
		repos = []*model.Repository{}
	}

	c.Status(http.StatusOK).JSON(repos)
	return nil
}

// @Router /api/v1/repos/{owner}/{repo} [get]
func (h *RepositoryHandler) GetRepository(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	repo, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}

	c.Status(http.StatusOK).JSON(repo)
	return nil
}

// @Router /api/v1/repos [post]
func (h *RepositoryHandler) CreateRepository(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	var req CreateRepositoryRequest
	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body")
		return nil
	}

	ownerName := user.UserName
	if req.Owner != "" && req.Owner != user.UserName {
		_, err := h.accountService.GetAccountByUsername(req.Owner)
		if err != nil {
			respondError(c, http.StatusBadRequest, "Owner does not exist")
			return nil
		}

		organizations, err := h.accountService.GetUserOrganizations(user.UserName)
		if err != nil {
			respondError(c, http.StatusInternalServerError, "Failed to check organization membership")
			return nil
		}

		isOrganizationMember := false
		for _, organization := range organizations {
			if organization.UserName == req.Owner {
				isOrganizationMember = true
				break
			}
		}

		if !isOrganizationMember && !user.IsAdmin {
			respondError(c, http.StatusForbidden, "You are not a member of this organization")
			return nil
		}
		ownerName = req.Owner
	}

	repo, err := h.repoService.CreateRepository(
		ownerName, req.Name, req.Private, req.Description, "main",
		nil, nil, nil, nil,
		user.UserName,
	)
	if err != nil {
		if err == service.ErrRepositoryExists {
			respondError(c, http.StatusConflict, "Repository already exists")
			return nil
		}
		respondError(c, http.StatusInternalServerError, "Failed to create repository")
		return nil
	}

	if err := h.gitClient.InitRepository(ownerName, req.Name); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to initialize Git repository")
		return nil
	}

	h.eventBus.Publish(event.NewRepositoryCreatedEvent(repo, user))

	LogAudit(c, "repository.create", "repository", ownerName+"/"+req.Name, nil, true)

	c.Status(http.StatusCreated).JSON(repo)
	return nil
}

func (h *RepositoryHandler) UpdateRepository(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	owner := c.Params("owner")
	repoName := c.Params("repo")

	repo, err := h.repoService.GetRepository(owner, repoName)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	// Check admin permission
	if !h.repoService.HasOwnerRole(repo, user) && !user.IsAdmin {
		respondError(c, http.StatusForbidden, "Forbidden")
		return nil
	}

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body")
		return nil
	}

	if desc, ok := updates["description"]; ok {
		if descStr, ok := desc.(string); ok {
			repo.Description = &descStr
		}
	}
	if priv, ok := updates["private"]; ok {
		if privBool, ok := priv.(bool); ok {
			repo.IsPrivate = privBool
		}
	}
	if mergeOpt, ok := updates["defaultMergeOption"]; ok {
		if mergeOptStr, ok := mergeOpt.(string); ok {
			repo.Options.DefaultMergeOption = mergeOptStr
		}
	}
	if defaultBranch, ok := updates["defaultBranch"]; ok {
		if defaultBranchStr, ok := defaultBranch.(string); ok {
			repo.DefaultBranch = defaultBranchStr
			if err := h.gitClient.UpdateDefaultBranch(owner, repoName, defaultBranchStr); err != nil {
				respondError(c, http.StatusInternalServerError, "Failed to update default branch: "+err.Error())
				return nil
			}
		}
	}
	if issuesOption, ok := updates["issuesOption"]; ok {
		if issuesOptionStr, ok := issuesOption.(string); ok {
			validOptions := map[string]bool{
				"ALL":     true,
				"PUBLIC":  true,
				"PRIVATE": true,
				"DISABLE": true,
			}
			if !validOptions[issuesOptionStr] {
				respondError(c, http.StatusBadRequest, "Invalid issuesOption value")
				return nil
			}
			repo.IssuesOption = issuesOptionStr
		}
	}

	if err := h.repoService.UpdateRepository(repo); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to update repository")
		return nil
	}

	if _, ok := updates["defaultBranch"]; ok {
		h.gitClient.DeleteBranchListCache(owner, repoName)
	}

	LogAudit(c, "repository.update", "repository", owner+"/"+repoName, nil, true)

	c.Status(http.StatusOK).JSON(repo)
	return nil
}

// DeleteRepository deletes a repository
func (h *RepositoryHandler) DeleteRepository(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	owner := c.Params("owner")
	repoName := c.Params("repo")

	repo, err := h.repoService.GetRepository(owner, repoName)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	// Check admin permission
	if !h.repoService.HasOwnerRole(repo, user) && !user.IsAdmin {
		respondError(c, http.StatusForbidden, "Forbidden")
		return nil
	}

	if err := h.repoService.DeleteRepository(owner, repoName); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to delete repository")
		return nil
	}

	// Publish event: ActivitySubscriber records the delete_repository activity.
	// repo was fetched above before deletion; reuse it so the event carries
	// owner/name context without an extra query.
	h.eventBus.Publish(event.NewRepositoryDeletedEvent(repo, user))

	LogAudit(c, "repository.delete", "repository", owner+"/"+repoName, nil, true)

	c.Status(http.StatusNoContent)
	return nil
}

// ForkRepository forks a repository
func (h *RepositoryHandler) ForkRepository(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	owner := c.Params("owner")
	repoName := c.Params("repo")

	// Get the original repository to verify it exists
	_, err := h.repoService.GetRepository(owner, repoName)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	// Fork the repository in the database
	forkedRepo, err := h.repoService.ForkRepository(owner, repoName, user.UserName)
	if err != nil {
		if err == service.ErrRepositoryExists {
			respondError(c, http.StatusConflict, "You already have a fork of this repository")
			return nil
		}
		respondError(c, http.StatusInternalServerError, "Failed to fork repository")
		return nil
	}

	// Copy the Git data
	if err := h.gitClient.ForkRepository(owner, repoName, user.UserName, forkedRepo.RepositoryName); err != nil {
		// Rollback: delete the database entry if git fork fails
		h.repoService.DeleteRepository(user.UserName, forkedRepo.RepositoryName)
		respondError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to fork repository data: %v", err))
		return nil
	}

	// Invalidate caches for the new forked repository
	git.InvalidateRepositoryCache(user.UserName, forkedRepo.RepositoryName)

	// Publish event: ActivitySubscriber records the fork_repository activity.
	h.eventBus.Publish(event.NewRepositoryForkedEvent(forkedRepo, user, owner, repoName))

	c.Status(http.StatusAccepted).JSON(forkedRepo)
	return nil
}

// GetForks lists all forks of a repository
func (h *RepositoryHandler) GetForks(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	user := contextutil.GetUserFromContext(c)
	var currentUser *string
	if user != nil {
		currentUser = &user.UserName
	}

	forks, err := h.repoService.GetForks(owner, repoName, currentUser)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to get forks")
		return nil
	}

	c.Status(http.StatusOK).JSON(forks)
	return nil
}

// GetForkCount returns the number of forks for a repository
func (h *RepositoryHandler) GetForkCount(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	count, err := h.repoService.GetForkCount(owner, repoName)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to get fork count")
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"count": count})
	return nil
}

// IsForked checks if the current user has forked a repository
func (h *RepositoryHandler) IsForked(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	owner := c.Params("owner")
	repoName := c.Params("repo")

	isForked, err := h.repoService.IsForked(owner, repoName, user.UserName)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to check fork status")
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"forked": isForked})
	return nil
}

// GetUserRole gets the current user's role in the repository
func (h *RepositoryHandler) GetUserRole(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		c.Status(http.StatusOK).JSON(fiber.Map{"role": nil, "canCreateIssue": false})
		return nil
	}

	owner := c.Params("owner")
	repoName := c.Params("repo")

	repo, err := h.repoService.GetRepository(owner, repoName)
	if err != nil {
		c.Status(http.StatusOK).JSON(fiber.Map{"role": nil, "canCreateIssue": false})
		return nil
	}

	var role string
	if h.repoService.HasOwnerRole(repo, user) || user.IsAdmin {
		role = "owner"
	} else if h.repoService.HasMemberRole(repo, user) {
		role = "member"
	} else if h.repoService.HasViewerRole(repo, user) {
		role = "viewer"
	} else {
		role = ""
	}

	// Aligned with GitBucket's isIssueEditable: issue and PR share the same
	// permission gate based on IssuesOption (ALL/PUBLIC/PRIVATE/DISABLE).
	canCreateIssue := h.repoService.IsIssueEditable(repo, user)

	c.Status(http.StatusOK).JSON(fiber.Map{"role": role, "canCreateIssue": canCreateIssue})
	return nil
}

// TransferRepository transfers a repository to a new owner
func (h *RepositoryHandler) TransferRepository(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	owner := c.Params("owner")
	repoName := c.Params("repo")

	repo, err := h.repoService.GetRepository(owner, repoName)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	// Check admin permission
	if !h.repoService.HasOwnerRole(repo, user) && !user.IsAdmin {
		respondError(c, http.StatusForbidden, "Forbidden")
		return nil
	}

	var req struct {
		NewOwner string `json:"newOwner" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body")
		return nil
	}

	// Transfer the repository
	if err := h.repoService.TransferRepository(owner, repoName, req.NewOwner); err != nil {
		if err == service.ErrRepositoryNotFound {
			respondError(c, http.StatusNotFound, "Repository not found")
			return nil
		}
		if err == service.ErrRepositoryExists {
			respondError(c, http.StatusConflict, "Repository already exists in target owner")
			return nil
		}
		respondError(c, http.StatusInternalServerError, "Failed to transfer repository")
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Repository transferred successfully"})
	return nil
}

// RenameRepository renames a repository
func (h *RepositoryHandler) RenameRepository(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	owner := c.Params("owner")
	repoName := c.Params("repo")

	repo, err := h.repoService.GetRepository(owner, repoName)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	// Check admin permission
	if !h.repoService.HasOwnerRole(repo, user) && !user.IsAdmin {
		respondError(c, http.StatusForbidden, "Forbidden")
		return nil
	}

	var req struct {
		NewName string `json:"newName" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body")
		return nil
	}

	// Rename the repository
	if err := h.repoService.RenameRepository(owner, repoName, req.NewName); err != nil {
		if err == service.ErrRepositoryNotFound {
			respondError(c, http.StatusNotFound, "Repository not found")
			return nil
		}
		if err == service.ErrRepositoryExists {
			respondError(c, http.StatusConflict, "Repository name already exists")
			return nil
		}
		respondError(c, http.StatusInternalServerError, "Failed to rename repository")
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Repository renamed successfully"})
	return nil
}

// ArchiveRepository archives or unarchives a repository
func (h *RepositoryHandler) ArchiveRepository(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	owner := c.Params("owner")
	repoName := c.Params("repo")

	repo, err := h.repoService.GetRepository(owner, repoName)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	// Check admin permission
	if !h.repoService.HasOwnerRole(repo, user) && !user.IsAdmin {
		respondError(c, http.StatusForbidden, "Forbidden")
		return nil
	}

	var req struct {
		IsArchived bool `json:"isArchived"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body")
		return nil
	}

	if err := h.repoService.ArchiveRepository(owner, repoName, req.IsArchived); err != nil {
		if err == service.ErrRepositoryNotFound {
			respondError(c, http.StatusNotFound, "Repository not found")
			return nil
		}
		respondError(c, http.StatusInternalServerError, "Failed to archive repository")
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Repository archive status updated"})
	return nil
}

// SetTemplateRepository sets or unsets a repository as a template
func (h *RepositoryHandler) SetTemplateRepository(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	owner := c.Params("owner")
	repoName := c.Params("repo")

	repo, err := h.repoService.GetRepository(owner, repoName)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	// Check admin permission
	if !h.repoService.HasOwnerRole(repo, user) && !user.IsAdmin {
		respondError(c, http.StatusForbidden, "Forbidden")
		return nil
	}

	var req struct {
		IsTemplate bool `json:"isTemplate"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body")
		return nil
	}

	if err := h.repoService.SetTemplateRepository(owner, repoName, req.IsTemplate); err != nil {
		if err == service.ErrRepositoryNotFound {
			respondError(c, http.StatusNotFound, "Repository not found")
			return nil
		}
		respondError(c, http.StatusInternalServerError, "Failed to set template repository")
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Repository template status updated"})
	return nil
}

// CreateFromTemplate creates a new repository from a template
func (h *RepositoryHandler) CreateFromTemplate(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	var req struct {
		TemplateOwner string `json:"templateOwner" validate:"required"`
		TemplateRepo  string `json:"templateRepo" validate:"required"`
		NewOwner      string `json:"newOwner" validate:"required"`
		NewRepo       string `json:"newRepo" validate:"required"`
		IsPrivate     bool   `json:"isPrivate"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	// Check if user has access to template repository
	templateRepo, err := h.repoService.GetRepository(req.TemplateOwner, req.TemplateRepo)
	if err != nil {
		respondError(c, http.StatusNotFound, "Template repository not found")
		return nil
	}

	// Check private repository access
	if templateRepo.IsPrivate {
		if !h.repoService.HasViewerRole(templateRepo, user) {
			respondError(c, http.StatusNotFound, "Template repository not found")
			return nil
		}
	}

	// Check if user has permission to create repository in target owner
	if req.NewOwner != user.UserName {
		// Check if the specified owner exists (could be a user or organization)
		_, err := h.accountService.GetAccountByUsername(req.NewOwner)
		if err != nil {
			respondError(c, http.StatusBadRequest, "Target owner does not exist")
			return nil
		}

		// Check if the specified owner is an organization the user belongs to
		organizations, err := h.accountService.GetUserOrganizations(user.UserName)
		if err != nil {
			respondError(c, http.StatusInternalServerError, "Failed to check organization membership")
			return nil
		}

		isOrganizationMember := false
		for _, organization := range organizations {
			if organization.UserName == req.NewOwner {
				isOrganizationMember = true
				break
			}
		}

		// Allow admins to create repositories for any user/organization
		if !isOrganizationMember && !user.IsAdmin {
			respondError(c, http.StatusForbidden, "You are not a member of this organization")
			return nil
		}
	}

	// Create repository from template
	repo, err := h.repoService.CreateFromTemplate(req.TemplateOwner, req.TemplateRepo, req.NewOwner, req.NewRepo, req.IsPrivate)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusCreated).JSON(repo)
	return nil
}

// GetForkStatus checks if the current repository is a fork and returns how far
// behind/ahead it is relative to its parent repository.
func (h *RepositoryHandler) GetForkStatus(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	repo, err := h.repoService.GetRepository(owner, repoName)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	if repo.ParentUserName == nil || repo.ParentRepositoryName == nil {
		c.Status(http.StatusOK).JSON(fiber.Map{"isFork": false})
		return nil
	}

	parentOwner := *repo.ParentUserName
	parentRepoName := *repo.ParentRepositoryName

	parentRepo, err := h.repoService.GetRepository(parentOwner, parentRepoName)
	if err != nil {
		c.Status(http.StatusOK).JSON(fiber.Map{"isFork": true, "parentAvailable": false})
		return nil
	}

	ahead, behind, err := h.gitClient.GetForkStatus(
		owner, repoName, repo.DefaultBranch,
		parentOwner, parentRepoName, parentRepo.DefaultBranch,
	)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to check fork status: "+err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{
		"isFork":          true,
		"parentAvailable": true,
		"parentOwner":     parentOwner,
		"parentRepo":      parentRepoName,
		"ahead":           ahead,
		"behind":          behind,
	})
	return nil
}

// SyncFork syncs the fork's default branch with its parent repository's default branch.
func (h *RepositoryHandler) SyncFork(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	owner := c.Params("owner")
	repoName := c.Params("repo")

	repo, err := h.repoService.GetRepository(owner, repoName)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	if repo.ParentUserName == nil || repo.ParentRepositoryName == nil {
		respondError(c, http.StatusBadRequest, "This repository is not a fork")
		return nil
	}

	// Check write permission
	if !h.repoService.HasOwnerRole(repo, user) && !user.IsAdmin {
		respondError(c, http.StatusForbidden, "Forbidden")
		return nil
	}

	parentOwner := *repo.ParentUserName
	parentRepoName := *repo.ParentRepositoryName

	parentRepo, err := h.repoService.GetRepository(parentOwner, parentRepoName)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Parent repository not found")
		return nil
	}

	err = h.gitClient.SyncFork(
		owner, repoName, repo.DefaultBranch,
		parentOwner, parentRepoName, parentRepo.DefaultBranch,
	)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to sync fork: "+err.Error())
		return nil
	}

	// Invalidate repository cache
	git.InvalidateRepositoryCache(owner, repoName)

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Fork synced successfully"})
	return nil
}
