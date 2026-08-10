package handler

import (
	"fmt"
	"net/http"

	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/hashid"
	"iforge/iforge/internal/model"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

// encodeID encodes a numeric ID into a short URL-safe string using Hashids.
// Thin wrapper forwarding to the shared hashid package to avoid circular imports.
func encodeID(id int) string {
	return hashid.Encode(id)
}

// decodeID decodes a Hashids string back to the original numeric ID.
func decodeID(s string) (int, error) {
	return hashid.Decode(s)
}

// Helper functions for common handler operations

// getUserAndRepo extracts the current user and repository from context/database.
// Returns the user, repository, and a boolean indicating if the operation should continue.
// If false, the handler has already written an error response.
func getUserAndRepo(c *fiber.Ctx, repoService *service.RepositoryService, owner, repoName string) (*model.Account, *model.Repository, bool) {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil, nil, false
	}

	repository, err := repoService.GetRepository(owner, repoName)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil, nil, false
	}

	return user, repository, true
}

// getRepoAndCheckAccess retrieves a repository and checks read access.
// Public repos: readable by anyone. Private repos: viewer role or above required.
// Returns 404 (not 403) to avoid leaking private repo existence.
func getRepoAndCheckAccess(c *fiber.Ctx, repoService *service.RepositoryService, owner, repoName string) (*model.Repository, *model.Account, bool) {
	repository, err := repoService.GetRepository(owner, repoName)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil, nil, false
	}
	user := contextutil.GetUserFromContext(c)
	if repository.IsPrivate && (user == nil || !repoService.HasViewerRole(repository, user)) {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil, nil, false
	}
	return repository, user, true
}

// getUserAndCheckPermission gets the current user and checks if they have admin-level permission
// (repository owner, ADMIN collaborator, or system admin).
// Aligned with GitBucket's ownerOnly authenticator.
// Returns the user, repository, and a boolean indicating if the operation should continue.
// If false, the handler has already written an error response.
func getUserAndCheckPermission(c *fiber.Ctx, repoService *service.RepositoryService, owner, repoName string) (*model.Account, *model.Repository, bool) {
	user, repository, ok := getUserAndRepo(c, repoService, owner, repoName)
	if !ok {
		return nil, nil, false
	}

	if !repoService.HasOwnerRole(repository, user) {
		respondError(c, http.StatusForbidden, "Forbidden: Owner access required")
		return nil, nil, false
	}

	return user, repository, true
}

// getUserAndCheckWritePermission checks if user has write-level access
// (repository owner, DEVELOPER/ADMIN collaborator, or system admin).
// Aligned with GitBucket's writableUsersOnly authenticator.
func getUserAndCheckWritePermission(c *fiber.Ctx, repoService *service.RepositoryService, owner, repoName string) (*model.Account, *model.Repository, bool) {
	user, repository, ok := getUserAndRepo(c, repoService, owner, repoName)
	if !ok {
		return nil, nil, false
	}

	if !repoService.HasMemberRole(repository, user) {
		respondError(c, http.StatusForbidden, "Forbidden: Developer access required")
		return nil, nil, false
	}

	return user, repository, true
}

// parsePaginationParams parses page and per_page query parameters
// Returns page number and offset for database queries
func parsePaginationParams(c *fiber.Ctx) (page, perPage, offset int) {
	page = 1
	if p := c.Query("page"); p != "" {
		if pInt, err := parseIntParam(p); err == nil {
			page = pInt
		}
	}

	perPage = 30
	if pp := c.Query("per_page"); pp != "" {
		if ppInt, err := parseIntParam(pp); err == nil {
			perPage = ppInt
		}
	}

	offset = (page - 1) * perPage
	return page, perPage, offset
}

// parseIntParam safely parses a string to int
func parseIntParam(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

// respondError sends an error response with the given status code and message.
// Returns the error from c.JSON() so it can be used directly in handler
// return statements, matching Fiber's idiom: `return respondError(c, ...)`.
func respondError(c *fiber.Ctx, statusCode int, message string) error {
	return c.Status(statusCode).JSON(fiber.Map{"error": message})
}

// respondErrorWithMessageKey sends an error response with an i18n message key
// and a human-readable fallback. Use only for user-facing localized errors,
// not for program logic identifiers (use "reason" field instead).
func respondErrorWithMessageKey(c *fiber.Ctx, statusCode int, messageKey, message string) error {
	return c.Status(statusCode).JSON(fiber.Map{"error": message, "messageKey": messageKey})
}

// respondSuccess sends a success response with the given data.
// Returns the error from c.JSON() so it can be used directly in handler
// return statements: `return respondSuccess(c, http.StatusOK, data)`.
func respondSuccess(c *fiber.Ctx, statusCode int, data interface{}) error {
	return c.Status(statusCode).JSON(data)
}

// getCurrentUser gets the current authenticated user from context
// Returns the user and a boolean indicating if the operation should continue
// If false, the handler has already written an error response
func getCurrentUser(c *fiber.Ctx) (*model.Account, bool) {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil, false
	}
	return user, true
}

// checkProjectReadAccess checks if the current user can read the project.
// Public projects: readable by anyone. Private projects: viewer role or above required.
// Returns 404 (not 403) to avoid leaking private project existence.
func checkProjectReadAccess(c *fiber.Ctx, projectService *service.ProjectService, projectID int) bool {
	project, err := projectService.GetProject(projectID)
	if err != nil {
		respondError(c, http.StatusNotFound, "Project not found")
		return false
	}
	if !project.IsPrivate {
		return true
	}
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusNotFound, "Project not found")
		return false
	}
	if _, err := projectService.RequireRole(project.ID, user.UserName, model.ProjectRoleViewer); err != nil {
		respondError(c, http.StatusNotFound, "Project not found")
		return false
	}
	return true
}

// checkProjectWriteAccess checks if the current user has write access (member role or above).
// Returns the user and true on success; on failure, error response is already written.
func checkProjectWriteAccess(c *fiber.Ctx, projectService *service.ProjectService, projectID int) (*model.Account, bool) {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil, false
	}
	project, err := projectService.GetProject(projectID)
	if err != nil {
		if err == service.ErrProjectNotFound {
			respondError(c, http.StatusNotFound, "Project not found")
			return nil, false
		}
		respondError(c, http.StatusInternalServerError, "Failed to get project")
		return nil, false
	}
	if _, err := projectService.RequireRole(project.ID, user.UserName, model.ProjectRoleMember); err != nil {
		respondError(c, http.StatusForbidden, "Only project members can perform this action")
		return nil, false
	}
	return user, true
}
