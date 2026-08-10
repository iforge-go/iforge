package router

import (
	"iforge/iforge/internal/container"
	"iforge/iforge/internal/handler"
	"iforge/iforge/internal/middleware"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// Setup creates and configures the Fiber router
func Setup(c *container.Container) *fiber.App {
	app := fiber.New(fiber.Config{
		BodyLimit: 512 * 1024 * 1024, // 512MB for large git push payloads
	})

	// Git HTTP handler for /:owner/:repo.git/*path pattern
	gitHTTPHandler := NewGitHTTPHandler(c)
	// LFS handler for /:owner/:repo(.git)?/info/lfs/* paths
	lfsHTTPHandler := c.LFSHandler()

	// Global middleware
	app.Use(middleware.CORSMiddleware())
	app.Use(middleware.MetricsMiddleware())

	// Middleware to handle /:owner/:repo.git/*pattern and /:owner/:repo/*git-paths
	// This must be registered before routes to avoid conflicts.
	// LFS paths (info/lfs/*) must be intercepted before git smart HTTP.
	//
	// These paths cannot be expressed via Fiber route patterns: Fiber's
	// `:repo` parameter does not match the `.git` suffix, and the git
	// smart-HTTP sub-paths (info/refs, HEAD, git-upload-pack, ...) need to
	// be funneled to a single handler rather than matched 1:1.
	app.Use(gitPathMiddleware(gitHTTPHandler, lfsHTTPHandler))

	// Setup guard - blocks non-setup API routes when system is not initialized
	app.Use(middleware.SetupMiddleware(c.SetupService()))

	// WebSocket upgrade middleware (required by fiber/websocket/v2).
	// Without this middleware, websocket.New() returns 404 on upgrade requests.
	app.Use("/api/v1/ws", func(c *fiber.Ctx) error {
		if strings.EqualFold(c.Get("Upgrade"), "websocket") {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	// Public routes (no auth required)
	registerPublicRoutes(app, c)

	// Protected routes
	api := app.Group("/api/v1")
	api.Use(middleware.AuthMiddleware(c.AccountService()))
	// General rate limit: 100 req/min per user (or per IP if unauthenticated).
	// Mounted after AuthMiddleware so the user context is available for the key.
	api.Use(middleware.GeneralRateLimit())

	// External Runner API must be registered before the authRequired group.
	// Runners use Bearer token authentication (not session). If placed inside the authRequired subgroup,
	// Fiber v2's Group("") would apply RequireAuth to these routes, causing 401 when user==nil.
	// The register endpoint still requires admin session (enforced by RequireAdmin), other endpoints use runnerAuth.
	registerCICDRunnerRoutes(api, c)

	// Forced authentication subgroup: private data routes (user profile, SSH keys, tokens, notifications, etc.)
	// Unauthenticated access to these routes returns 401, avoiding the need for each handler to check user==nil.
	// Public access routes (browsing public repos, viewing public user info, etc.) are still registered on the api group.
	authRequired := api.Group("", middleware.RequireAuth())

	// User & account (mix of public/private, internally split)
	registerUserRoutes(api, authRequired, c)
	registerSSHKeyRoutes(authRequired, c)
	registerGPGKeyRoutes(authRequired, c)
	registerAccessTokenRoutes(authRequired, c)
	registerExtraMailRoutes(authRequired, c)
	registerPreferenceRoutes(authRequired, c)

	// Repository
	registerRepoRoutes(api, c)
	registerGitRoutes(api, c)
	registerCollaboratorRoutes(api, c)
	registerDeployKeyRoutes(api, c)
	registerMirrorRoutes(api, c)
	registerWebhookRoutes(api, c)
	registerCommitStatusRoutes(api, c)
	registerStarRoutes(api, c)
	registerWatchRoutes(api, c)
	registerReleaseRoutes(api, c)
	registerStatsRoutes(api, c)
	registerLFSRoutes(api, c)

	// Issue & merge request
	registerIssueRoutes(api, c)
	registerMergeRequestRoutes(api, c)
	registerLabelRoutes(api, c)
	registerMilestoneRoutes(api, c)
	registerCustomFieldRoutes(api, c)
	registerPriorityRoutes(api, c)
	registerReviewRoutes(api, c)

	// CI/CD (Path C: built-in Pipeline/Job/Runner/Deployment)
	registerCICDRoutes(api, c)

	// Wiki, notification, activity
	registerWikiRoutes(api, c)
	registerNotificationRoutes(authRequired, c)
	registerWebSocketRoutes(api, c)
	registerActivityRoutes(api, c)

	// Project management (Scrum)
	registerProjectRoutes(api, c)

	// Admin
	registerSystemSettingsRoutes(api, c)
	registerOIDCRoutes(api, c)
	registerDatabaseViewerRoutes(api, c)
	registerPluginRoutes(api, c)
	registerAdminRoutes(api, c)

	// Health check
	app.Get("/health", func(ctx *fiber.Ctx) error {
		return ctx.SendString("OK")
	})

	return app
}

// gitPathMiddleware routes git smart-HTTP and LFS requests to their dedicated
// handlers. It must run before the regular API routes so that paths like
// /:owner/:repo.git/info/refs are not swallowed by /:owner/:repo/* routes.
//
// Recognized path patterns (in evaluation order):
//
//  1. /:owner/:repo.git/info/lfs/*            → LFS handler
//  2. /:owner/:repo.git/*                     → Git HTTP handler
//  3. /:owner/:repo/info/lfs/*                → LFS handler
//  4. /:owner/:repo/<git sub-path>[/...]      → Git HTTP handler
//
// where <git sub-path> is one of: info, HEAD, objects, refs,
// git-upload-pack, git-receive-pack (see isGitSubPath).
//
// Everything else falls through to the next middleware/route.
//
// Note: pattern 4 also covers `/info/refs` etc. (subPath == "info") when the
// LFS check (parts[3] == "lfs") does not match, so plain git smart-HTTP
// discovery under /info/* still reaches the git handler.
func gitPathMiddleware(gitHTTPHandler *GitHTTPHandler, lfsHTTPHandler *handler.LFSHandler) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		path := ctx.Path()
		// Root path "/" has nothing to route; skip straight to next handler.
		if len(path) <= 1 {
			return ctx.Next()
		}

		// `parts` is the URL path split on "/"; parts[0] is the owner,
		// parts[1] is the repo (optionally suffixed with ".git").
		parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
		// Need at least /:owner/:repo (2 segments) to even consider routing.
		if len(parts) < 2 {
			return ctx.Next()
		}

		// Patterns 1 & 2: /:owner/:repo.git/... (repo name carries the .git suffix)
		if strings.HasSuffix(parts[1], ".git") {
			if isLFSPath(parts) {
				_ = lfsHTTPHandler.ServeHTTP(ctx)
				return nil
			}
			gitHTTPHandler.ServeHTTP(ctx)
			return nil
		}

		// Patterns 3 & 4: /:owner/:repo/... (repo name without .git suffix)
		if len(parts) >= 3 {
			// Pattern 3: LFS without .git suffix. isLFSPath also requires
			// parts[2] == "info", so non-"info" sub-paths skip this branch.
			if isLFSPath(parts) {
				_ = lfsHTTPHandler.ServeHTTP(ctx)
				return nil
			}
			// Pattern 4: git smart-HTTP endpoints (info/refs, HEAD, ...).
			if isGitSubPath(parts[2]) {
				gitHTTPHandler.ServeHTTP(ctx)
				return nil
			}
		}

		return ctx.Next()
	}
}

// isLFSPath reports whether the path segments describe an LFS endpoint,
// i.e. /:owner/:repo[.git]/info/lfs/... — `parts` is the URL path split on "/".
// Requires at least 4 segments with parts[2] == "info" and parts[3] == "lfs".
func isLFSPath(parts []string) bool {
	return len(parts) >= 4 && parts[2] == "info" && parts[3] == "lfs"
}

// isGitSubPath reports whether the 3rd path segment (after /:owner/:repo) is a
// recognized git smart-HTTP endpoint marker. See gitPathMiddleware docs.
func isGitSubPath(s string) bool {
	switch s {
	case "info", "HEAD", "objects", "refs", "git-upload-pack", "git-receive-pack":
		return true
	}
	return false
}
