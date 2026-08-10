package router

import (
	"net/http"
	"strings"

	"iforge/iforge/internal/container"
	"iforge/iforge/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

// registerPublicRoutes registers public routes (no authentication required)
func registerPublicRoutes(app *fiber.App, c *container.Container) {
	pub := app.Group("/api/v1")

	// Authentication (login and register have rate limiting to prevent brute force and malicious registration)
	pub.Post("/login", middleware.StrictRateLimit(), c.AuthHandler().Login)
	pub.Post("/register", middleware.StrictRateLimit(), c.AuthHandler().Register)
	pub.Get("/register/check-username", c.AuthHandler().CheckUsername)

	// Public settings
	pub.Get("/settings/ssh", c.SystemSettingsHandler().GetPublicSSHSettings)
	pub.Get("/settings/general", c.SystemSettingsHandler().GetPublicGeneralSettings)
	pub.Get("/settings/clone-url-templates", c.SystemSettingsHandler().GetPublicCloneUrlTemplates)
	pub.Get("/settings/oidc/status", c.OIDCHandler().GetOIDCStatus)

	// Atom feeds
	app.Get("/feeds/users/:username.atom", c.AtomHandler().GetUserAtomFeed)
	app.Get("/feeds/repos/:owner/:repo.atom", c.AtomHandler().GetRepositoryAtomFeed)

	// SSH public keys
	app.Get("/keys/:username.keys", func(ctx *fiber.Ctx) error {
		username := ctx.Params("username")
		keys, err := c.SSHKeyService().ListSSHKeys(username)
		if err != nil {
			ctx.Status(http.StatusNotFound).SendString("")
			return nil
		}
		var keyTexts []string
		for _, key := range keys {
			keyTexts = append(keyTexts, key.PublicKey)
		}
		ctx.Set("Content-Type", "text/plain; charset=utf-8")
		ctx.Status(http.StatusOK).SendString(strings.Join(keyTexts, "\n"))
		return nil
	})

	// Archive downloads
	app.Get("/:owner/:repo/archive/*filepath", c.ArchiveHandler().DownloadArchive)

	// Raw file downloads (GitHub-style: /:owner/:repo/raw/:ref/*)
	app.Get("/:owner/:repo/raw/:ref/*", c.GitHandler().DownloadRawFile)

	// Password reset (strict rate limit: 5 req/min by IP)
	pub.Post("/password/reset/request", middleware.StrictRateLimit(), c.PasswordResetHandler().RequestReset)
	pub.Get("/password/reset/validate", c.PasswordResetHandler().ValidateResetToken)
	pub.Post("/password/reset", middleware.StrictRateLimit(), c.PasswordResetHandler().ResetPassword)

	// Setup wizard (only functional when system is not initialized)
	pub.Get("/setup/status", c.SetupHandler().GetSetupStatus)
	pub.Post("/setup", c.SetupHandler().Setup)
}
