package handler

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type CICDHandler struct {
	pipelineService   *service.PipelineService
	runnerService     *service.RunnerService
	secretService     *service.SecretService
	deploymentService *service.DeploymentService
	repoService       *service.RepositoryService
	cronService       *service.CronService
}

func NewCICDHandler(
	pipelineService *service.PipelineService,
	runnerService *service.RunnerService,
	secretService *service.SecretService,
	deploymentService *service.DeploymentService,
	repoService *service.RepositoryService,
	cronService *service.CronService,
) *CICDHandler {
	return &CICDHandler{
		pipelineService:   pipelineService,
		runnerService:     runnerService,
		secretService:     secretService,
		deploymentService: deploymentService,
		repoService:       repoService,
		cronService:       cronService,
	}
}

func (h *CICDHandler) ListPipelines(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	repository, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}
	_ = repository

	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))

	pipelines, total, err := h.pipelineService.ListPipelines(owner, repoName, page, pageSize)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list pipelines")
		return nil
	}

	c.JSON(fiber.Map{
		"items":    pipelines,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
	return nil
}

func (h *CICDHandler) GetPipeline(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	_, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}

	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid pipeline ID")
		return nil
	}

	pipeline, err := h.pipelineService.GetPipeline(id)
	if err != nil {
		respondError(c, http.StatusNotFound, "Pipeline not found")
		return nil
	}

	if pipeline.UserName != owner || pipeline.RepositoryName != repoName {
		respondError(c, http.StatusNotFound, "Pipeline not found")
		return nil
	}

	c.JSON(pipeline)
	return nil
}

func (h *CICDHandler) ListJobs(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	_, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}

	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid pipeline ID")
		return nil
	}

	pipeline, err := h.pipelineService.GetPipeline(id)
	if err != nil {
		respondError(c, http.StatusNotFound, "Pipeline not found")
		return nil
	}
	if pipeline.UserName != owner || pipeline.RepositoryName != repoName {
		respondError(c, http.StatusNotFound, "Pipeline not found")
		return nil
	}

	jobs, err := h.pipelineService.ListJobs(id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list jobs")
		return nil
	}

	c.JSON(fiber.Map{"items": jobs})
	return nil
}

func (h *CICDHandler) ListArtifacts(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	_, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}

	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid pipeline ID")
		return nil
	}

	pipeline, err := h.pipelineService.GetPipeline(id)
	if err != nil {
		respondError(c, http.StatusNotFound, "Pipeline not found")
		return nil
	}
	if pipeline.UserName != owner || pipeline.RepositoryName != repoName {
		respondError(c, http.StatusNotFound, "Pipeline not found")
		return nil
	}

	artifacts, err := h.pipelineService.ListArtifacts(id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list artifacts")
		return nil
	}

	c.JSON(fiber.Map{"items": artifacts})
	return nil
}

func (h *CICDHandler) DownloadArtifact(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	_, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}

	artifactID, err := strconv.ParseInt(c.Params("artifactId"), 10, 64)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid artifact ID")
		return nil
	}

	artifact, err := h.pipelineService.GetArtifact(artifactID)
	if err != nil {
		respondError(c, http.StatusNotFound, "Artifact not found")
		return nil
	}

	job, err := h.pipelineService.GetJob(artifact.JobID)
	if err != nil {
		respondError(c, http.StatusNotFound, "Job not found")
		return nil
	}
	pipeline, err := h.pipelineService.GetPipeline(job.PipelineID)
	if err != nil {
		respondError(c, http.StatusNotFound, "Pipeline not found")
		return nil
	}
	if pipeline.UserName != owner || pipeline.RepositoryName != repoName {
		respondError(c, http.StatusNotFound, "Artifact not found")
		return nil
	}

	return c.Download(artifact.StoragePath, artifact.Name)
}

func (h *CICDHandler) GetJob(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	_, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}

	jobID, err := strconv.ParseInt(c.Params("jobId"), 10, 64)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid job ID")
		return nil
	}

	job, err := h.pipelineService.GetJob(jobID)
	if err != nil {
		respondError(c, http.StatusNotFound, "Job not found")
		return nil
	}

	pipeline, err := h.pipelineService.GetPipeline(job.PipelineID)
	if err != nil {
		respondError(c, http.StatusNotFound, "Pipeline not found")
		return nil
	}
	if pipeline.UserName != owner || pipeline.RepositoryName != repoName {
		respondError(c, http.StatusNotFound, "Job not found")
		return nil
	}

	c.JSON(job)
	return nil
}

func (h *CICDHandler) GetJobLogs(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	_, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}

	jobID, err := strconv.ParseInt(c.Params("jobId"), 10, 64)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid job ID")
		return nil
	}

	job, err := h.pipelineService.GetJob(jobID)
	if err != nil {
		respondError(c, http.StatusNotFound, "Job not found")
		return nil
	}
	pipeline, err := h.pipelineService.GetPipeline(job.PipelineID)
	if err != nil {
		respondError(c, http.StatusNotFound, "Pipeline not found")
		return nil
	}
	if pipeline.UserName != owner || pipeline.RepositoryName != repoName {
		respondError(c, http.StatusNotFound, "Job not found")
		return nil
	}

	fromLine, _ := strconv.Atoi(c.Query("from_line", "0"))
	limit, _ := strconv.Atoi(c.Query("limit", "5000"))

	logs, err := h.pipelineService.ListJobLogs(jobID, fromLine, limit)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list job logs")
		return nil
	}

	c.JSON(fiber.Map{
		"items":    logs,
		"fromLine": fromLine,
		"limit":    limit,
	})
	return nil
}

func (h *CICDHandler) StreamJobLogs(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	_, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}

	jobID, err := strconv.ParseInt(c.Params("jobId"), 10, 64)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid job ID")
		return nil
	}

	job, err := h.pipelineService.GetJob(jobID)
	if err != nil {
		respondError(c, http.StatusNotFound, "Job not found")
		return nil
	}
	pipeline, err := h.pipelineService.GetPipeline(job.PipelineID)
	if err != nil {
		respondError(c, http.StatusNotFound, "Pipeline not found")
		return nil
	}
	if pipeline.UserName != owner || pipeline.RepositoryName != repoName {
		respondError(c, http.StatusNotFound, "Job not found")
		return nil
	}

	fromLine, _ := strconv.Atoi(c.Query("from_line", "0"))

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		writeEvent := func(event string, data interface{}) bool {
			jsonBytes, _ := json.Marshal(data)
			if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, string(jsonBytes)); err != nil {
				return false
			}
			w.Flush()
			return true
		}

		lastLineNo := fromLine

		for {
			logs, err := h.pipelineService.ListJobLogs(jobID, lastLineNo, 5000)
			if err == nil && len(logs) > 0 {
				for _, log := range logs {
					if !writeEvent("log", log) {
						return
					}
					lastLineNo = log.LineNo + 1
				}
			}

			currentJob, err := h.pipelineService.GetJob(jobID)
			if err == nil && currentJob.Status != "running" && currentJob.Status != "pending" {
				writeEvent("done", map[string]interface{}{
					"status": currentJob.Status,
				})
				return
			}

			time.Sleep(200 * time.Millisecond)
		}
	})

	return nil
}

func (h *CICDHandler) CancelPipeline(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	repository, user, ok := getRepoAndCheckAccess(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}
	if !h.repoService.HasMemberRole(repository, user) {
		respondError(c, http.StatusForbidden, "Write access required")
		return nil
	}

	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid pipeline ID")
		return nil
	}

	if err := h.pipelineService.CancelPipeline(id); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to cancel pipeline")
		return nil
	}

	c.JSON(fiber.Map{"ok": true})
	return nil
}

func (h *CICDHandler) RetryPipeline(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	repository, user, ok := getRepoAndCheckAccess(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}
	if !h.repoService.HasMemberRole(repository, user) {
		respondError(c, http.StatusForbidden, "Write access required")
		return nil
	}

	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid pipeline ID")
		return nil
	}

	if err := h.pipelineService.RetryPipeline(id); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to retry pipeline")
		return nil
	}

	c.JSON(fiber.Map{"ok": true})
	return nil
}

type TriggerJobRequest struct {
	Variables map[string]string `json:"variables"`
}

func (h *CICDHandler) TriggerManualJob(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	repository, user, ok := getRepoAndCheckAccess(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}
	if !h.repoService.HasMemberRole(repository, user) {
		respondError(c, http.StatusForbidden, "Write access required")
		return nil
	}

	jobID, err := strconv.ParseInt(c.Params("jobId"), 10, 64)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid job ID")
		return nil
	}

	job, err := h.pipelineService.GetJob(jobID)
	if err != nil {
		respondError(c, http.StatusNotFound, "Job not found")
		return nil
	}
	if job.When != "manual" {
		respondError(c, http.StatusBadRequest, "Job is not manual-triggerable")
		return nil
	}

	if job.Status != "pending" && job.Status != "failed" && job.Status != "canceled" {
		respondError(c, http.StatusBadRequest, "Job cannot be triggered in current status")
		return nil
	}

	if err := h.pipelineService.UpdateJobStatus(jobID, "pending", nil, nil); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to trigger job")
		return nil
	}

	c.JSON(fiber.Map{"ok": true})
	return nil
}

func (h *CICDHandler) ListRunners(c *fiber.Ctx) error {
	runners, err := h.runnerService.ListRunners()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list runners")
		return nil
	}
	c.JSON(fiber.Map{"items": runners})
	return nil
}

func (h *CICDHandler) DeleteRunner(c *fiber.Ctx) error {
	runnerID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid runner ID")
	}
	if err := h.runnerService.DeleteRunner(runnerID); err != nil {
		if err == service.ErrRunnerNotFound {
			return respondError(c, http.StatusNotFound, "Runner not found")
		}
		if err.Error() == "cannot delete builtin runner" {
			return respondError(c, http.StatusBadRequest, "Cannot delete builtin runner")
		}
		return respondError(c, http.StatusInternalServerError, "Failed to delete runner: "+err.Error())
	}
	return c.JSON(fiber.Map{"ok": true})
}

type SetSecretRequest struct {
	Key   string `json:"key" validate:"required"`
	Value string `json:"value" validate:"required"`
}

func (h *CICDHandler) ListSecrets(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	repository, user, ok := getRepoAndCheckAccess(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}
	if !h.repoService.HasMemberRole(repository, user) {
		respondError(c, http.StatusForbidden, "Write access required")
		return nil
	}

	secrets, err := h.secretService.ListSecrets(owner, repoName)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list secrets")
		return nil
	}
	c.JSON(fiber.Map{"items": secrets})
	return nil
}

func (h *CICDHandler) SetSecret(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	repository, user, ok := getRepoAndCheckAccess(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}
	if !h.repoService.HasMemberRole(repository, user) {
		respondError(c, http.StatusForbidden, "Write access required")
		return nil
	}

	var req SetSecretRequest
	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body")
		return nil
	}
	if req.Key == "" || req.Value == "" {
		respondError(c, http.StatusBadRequest, "Key and value are required")
		return nil
	}

	creator := ""
	if u := contextutil.GetUserFromContext(c); u != nil {
		creator = u.UserName
	}

	if err := h.secretService.SetSecret(owner, repoName, req.Key, req.Value, creator); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to set secret")
		return nil
	}

	c.Status(http.StatusCreated).JSON(fiber.Map{"ok": true})
	return nil
}

func (h *CICDHandler) DeleteSecret(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	repository, user, ok := getRepoAndCheckAccess(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}
	if !h.repoService.HasMemberRole(repository, user) {
		respondError(c, http.StatusForbidden, "Write access required")
		return nil
	}

	key := c.Params("key")
	if err := h.secretService.DeleteSecret(owner, repoName, key); err != nil {
		if err == service.ErrSecretNotFound {
			respondError(c, http.StatusNotFound, "Secret not found")
			return nil
		}
		respondError(c, http.StatusInternalServerError, "Failed to delete secret")
		return nil
	}

	c.JSON(fiber.Map{"ok": true})
	return nil
}

func (h *CICDHandler) ListEnvironments(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	_, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}

	envs, err := h.deploymentService.ListEnvironments(owner, repoName)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list environments")
		return nil
	}
	c.JSON(fiber.Map{"items": envs})
	return nil
}

func (h *CICDHandler) ListDeployments(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	_, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}

	envName := c.Query("environment")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))

	deployments, total, err := h.deploymentService.ListDeployments(owner, repoName, envName, page, pageSize)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list deployments")
		return nil
	}

	c.JSON(fiber.Map{
		"items":    deployments,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
	return nil
}

// ListDeploymentsByCommit GET /repos/:owner/:repo/commits/:commit/deployments
// For end-to-end traceability (commit detail page shows deployment status)
func (h *CICDHandler) ListDeploymentsByCommit(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")
	commitSHA := c.Params("commit")

	_, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}

	deployments, err := h.deploymentService.ListDeploymentsByCommit(owner, repoName, commitSHA)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list deployments")
		return nil
	}
	c.JSON(fiber.Map{"items": deployments})
	return nil
}

// CreateCronRequest request body for creating a cron schedule
type CreateCronRequest struct {
	Name       string `json:"name"`
	Schedule   string `json:"schedule"`
	Branch     string `json:"branch"`
	YAMLConfig string `json:"yamlConfig"`
}

// UpdateCronRequest represents a cron update request
type UpdateCronRequest struct {
	Schedule   string `json:"schedule"`
	Branch     string `json:"branch"`
	YAMLConfig string `json:"yamlConfig"`
	Enabled    bool   `json:"enabled"`
}

// ListCrons GET /repos/:owner/:repo/crons
func (h *CICDHandler) ListCrons(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	if _, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repoName); !ok {
		return nil
	}

	crons, err := h.cronService.ListCrons(owner, repoName)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list crons")
		return nil
	}
	c.JSON(fiber.Map{"items": crons})
	return nil
}

// CreateCron POST /repos/:owner/:repo/crons
func (h *CICDHandler) CreateCron(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	repository, user, ok := getRepoAndCheckAccess(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}
	if !h.repoService.HasMemberRole(repository, user) {
		respondError(c, http.StatusForbidden, "Write access required")
		return nil
	}

	var req CreateCronRequest
	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body")
		return nil
	}
	if req.Name == "" || req.Schedule == "" || req.Branch == "" {
		respondError(c, http.StatusBadRequest, "name, schedule, branch are required")
		return nil
	}

	cron, err := h.cronService.CreateCron(owner, repoName, req.Name, req.Schedule, req.Branch, req.YAMLConfig, user.UserName)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCronExpr):
			respondError(c, http.StatusBadRequest, "Invalid cron expression: "+err.Error())
		case errors.Is(err, service.ErrCronNameExists):
			respondError(c, http.StatusConflict, "Cron name already exists in this repository")
		default:
			respondError(c, http.StatusInternalServerError, "Failed to create cron")
		}
		return nil
	}
	c.Status(http.StatusCreated).JSON(cron)
	return nil
}

// GetCron GET /repos/:owner/:repo/crons/:id
func (h *CICDHandler) GetCron(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	if _, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repoName); !ok {
		return nil
	}

	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid cron ID")
		return nil
	}

	cron, err := h.cronService.GetCron(id)
	if err != nil {
		if errors.Is(err, service.ErrCronNotFound) {
			respondError(c, http.StatusNotFound, "Cron not found")
			return nil
		}
		respondError(c, http.StatusInternalServerError, "Failed to get cron")
		return nil
	}
	c.JSON(cron)
	return nil
}

// UpdateCron PATCH /repos/:owner/:repo/crons/:id
func (h *CICDHandler) UpdateCron(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	repository, user, ok := getRepoAndCheckAccess(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}
	if !h.repoService.HasMemberRole(repository, user) {
		respondError(c, http.StatusForbidden, "Write access required")
		return nil
	}

	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid cron ID")
		return nil
	}

	var req UpdateCronRequest
	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body")
		return nil
	}
	if req.Schedule == "" || req.Branch == "" {
		respondError(c, http.StatusBadRequest, "schedule, branch are required")
		return nil
	}

	cron, err := h.cronService.UpdateCron(id, req.Schedule, req.Branch, req.YAMLConfig, req.Enabled)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCronExpr):
			respondError(c, http.StatusBadRequest, "Invalid cron expression: "+err.Error())
		case errors.Is(err, service.ErrCronNotFound):
			respondError(c, http.StatusNotFound, "Cron not found")
		default:
			respondError(c, http.StatusInternalServerError, "Failed to update cron")
		}
		return nil
	}
	c.JSON(cron)
	return nil
}

// DeleteCron DELETE /repos/:owner/:repo/crons/:id
func (h *CICDHandler) DeleteCron(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	repository, user, ok := getRepoAndCheckAccess(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}
	if !h.repoService.HasMemberRole(repository, user) {
		respondError(c, http.StatusForbidden, "Write access required")
		return nil
	}

	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid cron ID")
		return nil
	}

	if err := h.cronService.DeleteCron(id); err != nil {
		if errors.Is(err, service.ErrCronNotFound) {
			respondError(c, http.StatusNotFound, "Cron not found")
			return nil
		}
		respondError(c, http.StatusInternalServerError, "Failed to delete cron")
		return nil
	}
	c.Status(http.StatusNoContent)
	return nil
}
