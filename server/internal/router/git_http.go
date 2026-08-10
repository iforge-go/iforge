package router

import (
	"iforge/iforge/internal/container"
	"iforge/iforge/internal/handler"

	"github.com/gofiber/fiber/v2"
)

// GitHTTPHandler wraps the Git HTTP handler for the router
type GitHTTPHandler struct {
	container *container.Container
	handler   *handler.GitHTTPHandler
}

// NewGitHTTPHandler creates a new Git HTTP handler wrapper
func NewGitHTTPHandler(c *container.Container) *GitHTTPHandler {
	return &GitHTTPHandler{
		container: c,
		handler:   handler.NewGitHTTPHandler(c.GitClient(), c.RepoService(), c.AccountService(), c.ActivityService(), c.CollaboratorService(), c.AccessTokenService(), c.EventBus()),
	}
}

// ServeHTTP handles Git HTTP requests
func (h *GitHTTPHandler) ServeHTTP(ctx *fiber.Ctx) {
	h.handler.ServeHTTP(ctx)
}
