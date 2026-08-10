package handler

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/model"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type ProjectHandler struct {
	projectService    *service.ProjectService
	accountService    *service.AccountService
	repositoryService *service.RepositoryService
}

func NewProjectHandler(projectService *service.ProjectService, accountService *service.AccountService, repositoryService *service.RepositoryService) *ProjectHandler {
	return &ProjectHandler{
		projectService:    projectService,
		accountService:    accountService,
		repositoryService: repositoryService,
	}
}

type CreateProjectRequest struct {
	Name        string  `json:"name" validate:"required"`
	Description *string `json:"description"`
	IsPrivate   bool    `json:"isPrivate"`
}

type UpdateProjectRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	IsPrivate   *bool   `json:"isPrivate"`
}

func (h *ProjectHandler) CreateProject(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	var req CreateProjectRequest
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid request body")
	}

	if req.Name == "" {
		return respondError(c, http.StatusBadRequest, "Name is required")
	}

	project, err := h.projectService.CreateProject(user.UserName, req.Name, req.Description, req.IsPrivate)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to create project")
	}

	// Add creator as owner member
	if err := h.projectService.AddMember(project.ID, user.UserName, "owner"); err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to add project member")
	}

	project.Slug = encodeID(project.ID)
	return c.Status(http.StatusCreated).JSON(project)
}

// GetProject retrieves a project by ID
func (h *ProjectHandler) GetProject(c *fiber.Ctx) error {
	id, err := decodeID(c.Params("projectSlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid project ID")
	}

	project, err := h.projectService.GetProject(id)
	if err != nil {
		if err == service.ErrProjectNotFound {
			return respondError(c, http.StatusNotFound, "Project not found")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to get project")
	}

	user := contextutil.GetUserFromContext(c)
	// Private projects require viewer+ role; public projects are open to all, including unauthenticated users.
	if project.IsPrivate {
		if user == nil {
			return respondError(c, http.StatusNotFound, "Project not found")
		}
		if _, err := h.projectService.RequireRole(project.ID, user.UserName, model.ProjectRoleViewer); err != nil {
			// Return 404 instead of 403 to avoid leaking private project existence
			return respondError(c, http.StatusNotFound, "Project not found")
		}
	}

	project.Slug = encodeID(project.ID)
	return c.Status(http.StatusOK).JSON(project)
}

// UpdateProject updates a project
func (h *ProjectHandler) UpdateProject(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	id, err := decodeID(c.Params("projectSlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid project ID")
	}

	project, err := h.projectService.GetProject(id)
	if err != nil {
		if err == service.ErrProjectNotFound {
			return respondError(c, http.StatusNotFound, "Project not found")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to get project")
	}

	// Check if user is owner
	if project.OwnerName != user.UserName {
		return respondError(c, http.StatusForbidden, "Only project owner can update")
	}

	var req UpdateProjectRequest
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid request body")
	}

	if req.Name != nil {
		project.Name = *req.Name
	}
	if req.Description != nil {
		project.Description = req.Description
	}
	if req.IsPrivate != nil {
		project.IsPrivate = *req.IsPrivate
	}

	if err := h.projectService.UpdateProject(project); err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to update project")
	}

	project.Slug = encodeID(project.ID)
	return c.Status(http.StatusOK).JSON(project)
}

// DeleteProject deletes a project
func (h *ProjectHandler) DeleteProject(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	id, err := decodeID(c.Params("projectSlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid project ID")
	}

	project, err := h.projectService.GetProject(id)
	if err != nil {
		if err == service.ErrProjectNotFound {
			return respondError(c, http.StatusNotFound, "Project not found")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to get project")
	}

	// Check if user is owner
	if project.OwnerName != user.UserName {
		return respondError(c, http.StatusForbidden, "Only project owner can delete")
	}

	if err := h.projectService.DeleteProject(id); err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to delete project")
	}

	return c.SendStatus(http.StatusNoContent)
}

// ListProjects lists projects for the current user
func (h *ProjectHandler) ListProjects(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	limit, _ := strconv.Atoi(c.Query("limit", "30"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	projects, err := h.projectService.ListProjects(user.UserName, limit, offset)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to list projects")
	}

	for i := range projects {
		projects[i].Slug = encodeID(projects[i].ID)
		// Populate current user's role in this project for frontend display
		projects[i].MyRole = h.projectService.GetMemberRole(projects[i].ID, user.UserName)
	}
	// Populate member, iteration, and user story counts; log but do not fail on error
	if err := h.projectService.FillProjectCounts(projects); err != nil {
		log.Printf("warn: FillProjectCounts failed for project list: %v", err)
	}
	return c.Status(http.StatusOK).JSON(projects)
}

// AddMember adds a member to a project
func (h *ProjectHandler) AddMember(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	id, err := decodeID(c.Params("projectSlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid project ID")
	}

	project, err := h.projectService.GetProject(id)
	if err != nil {
		if err == service.ErrProjectNotFound {
			return respondError(c, http.StatusNotFound, "Project not found")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to get project")
	}

	if _, err := h.projectService.RequireRole(project.ID, user.UserName, model.ProjectRoleAdmin); err != nil {
		return respondError(c, http.StatusForbidden, "Admin access required to manage members")
	}

	var req struct {
		UserName string `json:"userName" validate:"required"`
		Role     string `json:"role" validate:"required"`
	}
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid request body")
	}
	if strings.ToLower(req.Role) == model.ProjectRoleOwner {
		return respondError(c, http.StatusBadRequest, "Cannot assign owner role")
	}

	if err := h.projectService.AddMember(id, req.UserName, req.Role); err != nil {
		if err == service.ErrInvalidRole {
			return respondError(c, http.StatusBadRequest, "Invalid role")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to add member")
	}

	LogAudit(c, "project.member.add", "project", c.Params("projectSlug"),
		map[string]interface{}{"member": req.UserName, "role": req.Role}, true)

	return c.Status(http.StatusCreated).JSON(fiber.Map{"userName": req.UserName, "role": req.Role})
}

// RemoveMember removes a member from a project
func (h *ProjectHandler) RemoveMember(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	id, err := decodeID(c.Params("projectSlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid project ID")
	}

	project, err := h.projectService.GetProject(id)
	if err != nil {
		if err == service.ErrProjectNotFound {
			return respondError(c, http.StatusNotFound, "Project not found")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to get project")
	}

	if _, err := h.projectService.RequireRole(project.ID, user.UserName, model.ProjectRoleAdmin); err != nil {
		return respondError(c, http.StatusForbidden, "Admin access required to manage members")
	}

	userName := c.Params("userName")
	if userName == project.OwnerName {
		return respondError(c, http.StatusBadRequest, "Cannot remove project owner")
	}
	if err := h.projectService.RemoveMember(id, userName); err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to remove member")
	}

	return c.SendStatus(http.StatusNoContent)
}

// UpdateMemberRole updates the role of a project member
func (h *ProjectHandler) UpdateMemberRole(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	id, err := decodeID(c.Params("projectSlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid project ID")
	}

	project, err := h.projectService.GetProject(id)
	if err != nil {
		if err == service.ErrProjectNotFound {
			return respondError(c, http.StatusNotFound, "Project not found")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to get project")
	}

	if _, err := h.projectService.RequireRole(project.ID, user.UserName, model.ProjectRoleAdmin); err != nil {
		return respondError(c, http.StatusForbidden, "Admin access required to manage member roles")
	}

	userName := c.Params("userName")
	if userName == "" {
		return respondError(c, http.StatusBadRequest, "User name is required")
	}

	// Cannot change owner's role
	if userName == project.OwnerName {
		return respondError(c, http.StatusBadRequest, "Cannot change owner role")
	}

	var req struct {
		Role string `json:"role" validate:"required"`
	}
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid request body")
	}
	if req.Role == "" {
		return respondError(c, http.StatusBadRequest, "Role is required")
	}
	// Cannot promote member to owner
	if strings.ToLower(req.Role) == model.ProjectRoleOwner {
		return respondError(c, http.StatusBadRequest, "Cannot assign owner role")
	}

	if err := h.projectService.UpdateMemberRole(id, userName, req.Role); err != nil {
		switch err {
		case service.ErrProjectMemberNotFound:
			return respondError(c, http.StatusNotFound, "Member not found")
		case service.ErrInvalidRole:
			return respondError(c, http.StatusBadRequest, "Invalid role")
		case service.ErrProjectForbidden:
			return respondError(c, http.StatusBadRequest, "Cannot change owner role")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to update member role")
	}

	LogAudit(c, "project.member.role_update", "project", c.Params("projectSlug"),
		map[string]interface{}{"member": userName, "role": req.Role}, true)

	return c.SendStatus(http.StatusNoContent)
}

// GetMembers retrieves all members of a project
func (h *ProjectHandler) GetMembers(c *fiber.Ctx) error {
	id, err := decodeID(c.Params("projectSlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid project ID")
	}

	members, err := h.projectService.GetMembers(id)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to get members")
	}

	// Fill in user info (FullName, Image)
	userNames := make([]string, len(members))
	for i, m := range members {
		userNames[i] = m.UserName
	}
	avatarInfo, _ := h.accountService.GetUserAvatarInfo(userNames)
	for i := range members {
		if info, ok := avatarInfo[members[i].UserName]; ok {
			members[i].FullName = info.FullName
			members[i].Image = info.Image
		}
		members[i].ProjectSlug = encodeID(members[i].ProjectID)
	}
	return c.Status(http.StatusOK).JSON(members)
}

// AddRepositoryRequest represents an add repository request
type AddRepositoryRequest struct {
	UserName       string `json:"userName" validate:"required"`
	RepositoryName string `json:"repositoryName" validate:"required"`
}

// ListRepositories lists all repositories linked to a project
func (h *ProjectHandler) ListRepositories(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	id, err := decodeID(c.Params("projectSlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid project ID")
	}

	project, err := h.projectService.GetProject(id)
	if err != nil {
		if err == service.ErrProjectNotFound {
			return respondError(c, http.StatusNotFound, "Project not found")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to get project")
	}

	// Private projects require viewer+ role
	if project.IsPrivate {
		if _, err := h.projectService.RequireRole(project.ID, user.UserName, model.ProjectRoleViewer); err != nil {
			return respondError(c, http.StatusNotFound, "Project not found")
		}
	}

	repos, err := h.projectService.GetRepositories(id)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to get repositories")
	}

	// B1: Hide repositories the current user has no access to.
	// Project members should only see repositories they can actually open.
	visibleRepos := make([]*model.ProjectRepository, 0, len(repos))
	for _, r := range repos {
		repo, err := h.repositoryService.GetRepository(r.UserName, r.RepositoryName)
		if err != nil {
			continue
		}
		if h.repositoryService.HasViewerRole(repo, user) {
			r.ProjectSlug = encodeID(r.ProjectID)
			visibleRepos = append(visibleRepos, r)
		}
	}
	return c.Status(http.StatusOK).JSON(visibleRepos)
}

// AddRepository links a repository to a project
func (h *ProjectHandler) AddRepository(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	id, err := decodeID(c.Params("projectSlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid project ID")
	}

	// Verify project exists (404 before permission check)
	if _, err := h.projectService.GetProject(id); err != nil {
		if err == service.ErrProjectNotFound {
			return respondError(c, http.StatusNotFound, "Project not found")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to get project")
	}

	// Linking repositories requires admin+ role (owner/admin)
	if _, err := h.projectService.RequireRole(id, user.UserName, model.ProjectRoleAdmin); err != nil {
		return respondError(c, http.StatusForbidden, "Admin access required to link repositories")
	}

	var req AddRepositoryRequest
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid request body")
	}
	if req.UserName == "" || req.RepositoryName == "" {
		return respondError(c, http.StatusBadRequest, "userName and repositoryName are required")
	}

	// Verify the user has access to the repository being linked
	repo, err := h.repositoryService.GetRepository(req.UserName, req.RepositoryName)
	if err != nil {
		return respondError(c, http.StatusNotFound, "Repository not found")
	}
	if !h.repositoryService.HasViewerRole(repo, user) {
		return respondError(c, http.StatusForbidden, "You do not have access to this repository")
	}

	// Check for duplicate link
	existing, _ := h.projectService.GetRepositories(id)
	for _, r := range existing {
		if r.UserName == req.UserName && r.RepositoryName == req.RepositoryName {
			return respondError(c, http.StatusConflict, "Repository already linked")
		}
	}

	if err := h.projectService.AddRepository(id, req.UserName, req.RepositoryName); err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to link repository")
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"userName":       req.UserName,
		"repositoryName": req.RepositoryName,
		"projectSlug":    encodeID(id),
	})
}

// RemoveRepository unlinks a repository from a project
func (h *ProjectHandler) RemoveRepository(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	id, err := decodeID(c.Params("projectSlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid project ID")
	}

	// Verify project exists (404 before permission check)
	if _, err := h.projectService.GetProject(id); err != nil {
		if err == service.ErrProjectNotFound {
			return respondError(c, http.StatusNotFound, "Project not found")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to get project")
	}

	// Unlinking repositories requires admin+ role (owner/admin)
	if _, err := h.projectService.RequireRole(id, user.UserName, model.ProjectRoleAdmin); err != nil {
		return respondError(c, http.StatusForbidden, "Admin access required to unlink repositories")
	}

	userName := c.Params("userName")
	repoName := c.Params("repoName")
	if userName == "" || repoName == "" {
		return respondError(c, http.StatusBadRequest, "userName and repoName are required")
	}

	if err := h.projectService.RemoveRepository(id, userName, repoName); err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to unlink repository")
	}

	return c.SendStatus(http.StatusNoContent)
}
