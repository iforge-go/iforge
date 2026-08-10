package handler

import (
	"net/http"
	"strconv"

	"iforge/iforge/internal/middleware"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

// External Runner API Handler
//
// Endpoints for external Runner Agent:
//   POST /api/v1/runners/register           - Register runner (admin session)
//   POST /api/v1/runners/heartbeat          - Heartbeat (Bearer token)
//   POST /api/v1/runners/jobs/claim         - Claim pending job (Bearer token)
//   POST /api/v1/runners/jobs/:id/status    - Report job status (Bearer token)
//   POST /api/v1/runners/jobs/:id/logs      - Batch upload logs (Bearer token)

// CICDRunnerHandler handles external Runner API requests
type CICDRunnerHandler struct {
	runnerService   *service.RunnerService
	secretService   *service.SecretService
	pipelineService *service.PipelineService
	externalURL     string // External access URL for reverse proxy scenarios
}

func NewCICDRunnerHandler(
	runnerService *service.RunnerService,
	secretService *service.SecretService,
	pipelineService *service.PipelineService,
	externalURL string,
) *CICDRunnerHandler {
	return &CICDRunnerHandler{
		runnerService:   runnerService,
		secretService:   secretService,
		pipelineService: pipelineService,
		externalURL:     externalURL,
	}
}

// RegisterRunnerRequest is the request body for registering an external runner
type RegisterRunnerRequest struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // shell | docker
	Description string `json:"description"`
	IPAddress   string `json:"ipAddress"`
}

// RegisterRunnerResponse contains the plaintext token, returned only once
type RegisterRunnerResponse struct {
	RunnerID int64  `json:"runnerId"`
	Token    string `json:"token"`
}

// HeartbeatRequest is the heartbeat payload from a runner
type HeartbeatRequest struct {
	Status  string `json:"status"` // online | busy | offline
	Version string `json:"version"`
}

// UpdateJobStatusRequest is the job status update from a runner
type UpdateJobStatusRequest struct {
	Status        string  `json:"status"` // success | failed | running
	ExitCode      *int    `json:"exitCode"`
	FailureReason *string `json:"failureReason"`
}

// AppendJobLogsRequest is the batch log upload from a runner
type AppendJobLogsRequest struct {
	Logs []service.JobLogBatch `json:"logs"`
}

// RegisterRunner registers an external runner and returns a plaintext token.
// Requires admin session authentication.
func (h *CICDRunnerHandler) RegisterRunner(c *fiber.Ctx) error {
	var req RegisterRunnerRequest
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid request body")
	}
	if req.Name == "" {
		return respondError(c, http.StatusBadRequest, "Runner name is required")
	}
	if req.Type == "" {
		req.Type = "shell"
	}

	runner, token, err := h.runnerService.RegisterExternalRunner(req.Name, req.Type, req.Description, req.IPAddress)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to register runner: "+err.Error())
	}

	return c.Status(http.StatusCreated).JSON(RegisterRunnerResponse{
		RunnerID: runner.ID,
		Token:    token,
	})
}

// Heartbeat processes a runner heartbeat. Requires Bearer token authentication.
func (h *CICDRunnerHandler) Heartbeat(c *fiber.Ctx) error {
	runner := middleware.GetRunnerFromContext(c)
	if runner == nil {
		return respondError(c, http.StatusUnauthorized, "Runner not authenticated")
	}

	var req HeartbeatRequest
	_ = c.BodyParser(&req) // allow empty body

	status := req.Status
	if status == "" {
		status = "online"
	}
	if err := h.runnerService.UpdateRunnerStatus(runner.ID, status); err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to update runner status")
	}

	return c.JSON(fiber.Map{"ok": true})
}

// ClaimJob lets a runner claim a pending job. Requires Bearer token authentication.
func (h *CICDRunnerHandler) ClaimJob(c *fiber.Ctx) error {
	runner := middleware.GetRunnerFromContext(c)
	if runner == nil {
		return respondError(c, http.StatusUnauthorized, "Runner not authenticated")
	}

	// Use configured externalURL; fall back to request base URL
	iforgeURL := h.externalURL
	if iforgeURL == "" {
		iforgeURL = c.BaseURL()
	}

	resp, err := h.runnerService.ClaimJobForRunner(runner.ID, h.secretService, iforgeURL)
	if err != nil {
		if err == service.ErrNoJobAvailable {
			return c.Status(http.StatusNoContent).JSON(fiber.Map{"message": "no job available"})
		}
		return respondError(c, http.StatusInternalServerError, "Failed to claim job: "+err.Error())
	}

	return c.JSON(resp)
}

// UpdateJobStatus processes a job status update from a runner.
// Requires Bearer token authentication.
func (h *CICDRunnerHandler) UpdateJobStatus(c *fiber.Ctx) error {
	runner := middleware.GetRunnerFromContext(c)
	if runner == nil {
		return respondError(c, http.StatusUnauthorized, "Runner not authenticated")
	}

	jobID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid job ID")
	}

	var req UpdateJobStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid request body")
	}
	if req.Status == "" {
		return respondError(c, http.StatusBadRequest, "Status is required")
	}

	if err := h.runnerService.UpdateJobFromRunner(jobID, runner.ID, req.Status, req.ExitCode, req.FailureReason); err != nil {
		if err == service.ErrJobNotFound {
			return respondError(c, http.StatusNotFound, "Job not found")
		}
		if err.Error() == "job not assigned to this runner" {
			return respondError(c, http.StatusForbidden, "Job not assigned to this runner")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to update job status: "+err.Error())
	}

	// On terminal status, trigger next DAG scheduling and async pipeline refresh
	if req.Status == "success" || req.Status == "failed" {
		h.pipelineService.ScheduleNextJobsByJobID(jobID)
		go h.pipelineService.RefreshPipelineStatusByJob(jobID)
	}

	return c.JSON(fiber.Map{"ok": true})
}

// AppendJobLogs processes batch log uploads from a runner.
// Requires Bearer token authentication.
func (h *CICDRunnerHandler) AppendJobLogs(c *fiber.Ctx) error {
	runner := middleware.GetRunnerFromContext(c)
	if runner == nil {
		return respondError(c, http.StatusUnauthorized, "Runner not authenticated")
	}

	jobID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid job ID")
	}

	var req AppendJobLogsRequest
	if err := c.BodyParser(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid request body")
	}

	if err := h.runnerService.AppendJobLogsFromRunner(jobID, runner.ID, req.Logs); err != nil {
		if err == service.ErrJobNotFound {
			return respondError(c, http.StatusNotFound, "Job not found")
		}
		if err.Error() == "job not assigned to this runner" {
			return respondError(c, http.StatusForbidden, "Job not assigned to this runner")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to append job logs: "+err.Error())
	}

	return c.JSON(fiber.Map{"ok": true})
}
