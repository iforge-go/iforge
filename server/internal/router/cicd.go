package router

import (
	"iforge/iforge/internal/container"
	"iforge/iforge/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

// registerCICDRoutes registers CI/CD routes
//
// Route organization:
//   - Pipeline:  /repos/:owner/:repo/pipelines/*
//   - Job:       /repos/:owner/:repo/jobs/*  + /pipelines/:id/jobs
//   - Secret:    /repos/:owner/:repo/secrets/*
//   - Env:       /repos/:owner/:repo/environments
//   - Deploy:    /repos/:owner/:repo/deployments + /commits/:commit/deployments
//   - Runner:    /admin/runners (admin view)
//
// Note: External Runner API (/runners/*) is registered separately by registerCICDRunnerRoutes,
// and must be registered before the authRequired group in router.go to avoid being intercepted
// by the global RequireAuth middleware (runners use Bearer token auth, not session).
func registerCICDRoutes(api fiber.Router, c *container.Container) {
	h := c.CICDHandler()

	// Pipelines
	api.Get("/repos/:owner/:repo/pipelines", h.ListPipelines)
	api.Get("/repos/:owner/:repo/pipelines/:id", h.GetPipeline)
	api.Post("/repos/:owner/:repo/pipelines/:id/cancel", h.CancelPipeline)
	api.Post("/repos/:owner/:repo/pipelines/:id/retry", h.RetryPipeline)
	api.Get("/repos/:owner/:repo/pipelines/:id/jobs", h.ListJobs)
	api.Get("/repos/:owner/:repo/pipelines/:id/artifacts", h.ListArtifacts)

	// Jobs (independent paths for direct access by job ID)
	api.Get("/repos/:owner/:repo/jobs/:jobId", h.GetJob)
	api.Get("/repos/:owner/:repo/jobs/:jobId/logs", h.GetJobLogs)
	api.Get("/repos/:owner/:repo/jobs/:jobId/logs/stream", h.StreamJobLogs)
	api.Post("/repos/:owner/:repo/jobs/:jobId/trigger", h.TriggerManualJob)

	// Artifacts (download)
	api.Get("/repos/:owner/:repo/artifacts/:artifactId/download", h.DownloadArtifact)

	// Secrets
	api.Get("/repos/:owner/:repo/secrets", h.ListSecrets)
	api.Post("/repos/:owner/:repo/secrets", h.SetSecret)
	api.Delete("/repos/:owner/:repo/secrets/:key", h.DeleteSecret)

	// Environments & Deployments
	api.Get("/repos/:owner/:repo/environments", h.ListEnvironments)
	api.Get("/repos/:owner/:repo/deployments", h.ListDeployments)
	api.Get("/repos/:owner/:repo/commits/:commit/deployments", h.ListDeploymentsByCommit)

	// Cron Schedules (scheduled CI/CD pipeline triggers)
	api.Get("/repos/:owner/:repo/crons", h.ListCrons)
	api.Post("/repos/:owner/:repo/crons", h.CreateCron)
	api.Get("/repos/:owner/:repo/crons/:id", h.GetCron)
	api.Patch("/repos/:owner/:repo/crons/:id", h.UpdateCron)
	api.Delete("/repos/:owner/:repo/crons/:id", h.DeleteCron)

	// Runners (admin view, requires admin session)
	api.Get("/admin/runners", h.ListRunners)
	api.Delete("/admin/runners/:id", h.DeleteRunner)
}

// registerCICDRunnerRoutes registers external Runner API routes
//
// Must be called before `authRequired := api.Group("", middleware.RequireAuth())` in router.go,
// otherwise runner Bearer token requests will be intercepted by the global RequireAuth middleware (user==nil).
//
// Endpoints:
//
//	POST /runners/register            - Register runner (requires admin session)
//	POST /runners/heartbeat           - Heartbeat (Bearer token)
//	POST /runners/jobs/claim          - Claim pending job (Bearer token)
//	POST /runners/jobs/:id/status     - Report job status (Bearer token)
//	POST /runners/jobs/:id/logs       - Batch upload logs (Bearer token)
func registerCICDRunnerRoutes(api fiber.Router, c *container.Container) {
	rh := c.CICDRunnerHandler()
	runnerAuth := middleware.RunnerAuthMiddleware(c.RunnerService())

	// register requires admin session (AuthMiddleware parses session + RequireAdmin enforces admin)
	api.Post("/runners/register", middleware.RequireAdmin(), rh.RegisterRunner)

	// The following endpoints use runner Bearer token authentication (not session)
	api.Post("/runners/heartbeat", runnerAuth, rh.Heartbeat)
	api.Post("/runners/jobs/claim", runnerAuth, rh.ClaimJob)
	api.Post("/runners/jobs/:id/status", runnerAuth, rh.UpdateJobStatus)
	api.Post("/runners/jobs/:id/logs", runnerAuth, rh.AppendJobLogs)
}
