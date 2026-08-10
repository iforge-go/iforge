package handler

import (
	"net/http"
	"strconv"

	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type DeployKeyHandler struct {
	deployKeyService *service.DeployKeyService
	repoService      *service.RepositoryService
}

func NewDeployKeyHandler(deployKeyService *service.DeployKeyService, repoService *service.RepositoryService) *DeployKeyHandler {
	return &DeployKeyHandler{
		deployKeyService: deployKeyService,
		repoService:      repoService,
	}
}

func (h *DeployKeyHandler) ListDeployKeys(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	_, _, ok := getUserAndCheckPermission(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}

	deployKeys, err := h.deployKeyService.ListDeployKeys(owner, repoName)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list deploy keys")
		return nil
	}

	c.Status(http.StatusOK).JSON(deployKeys)
	return nil
}

type CreateDeployKeyRequest struct {
	Title      string `json:"title" validate:"required"`
	PublicKey  string `json:"public_key" validate:"required"`
	AllowWrite bool   `json:"allow_write"`
}

func (h *DeployKeyHandler) CreateDeployKey(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	_, _, ok := getUserAndCheckPermission(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}

	var req CreateDeployKeyRequest
	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	deployKey, err := h.deployKeyService.CreateDeployKey(owner, repoName, req.Title, req.PublicKey, req.AllowWrite)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to create deploy key")
		return nil
	}

	c.Status(http.StatusCreated).JSON(deployKey)
	return nil
}

func (h *DeployKeyHandler) GetDeployKey(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")
	deployKeyID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid deploy key ID")
		return nil
	}

	_, _, ok := getUserAndCheckPermission(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}

	deployKey, err := h.deployKeyService.GetDeployKey(owner, repoName, deployKeyID)
	if err != nil {
		respondError(c, http.StatusNotFound, "Deploy key not found")
		return nil
	}

	c.Status(http.StatusOK).JSON(deployKey)
	return nil
}

func (h *DeployKeyHandler) DeleteDeployKey(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")
	deployKeyID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid deploy key ID")
		return nil
	}

	_, _, ok := getUserAndCheckPermission(c, h.repoService, owner, repoName)
	if !ok {
		return nil
	}

	if err := h.deployKeyService.DeleteDeployKey(owner, repoName, deployKeyID); err != nil {
		respondError(c, http.StatusNotFound, "Deploy key not found")
		return nil
	}

	c.Status(http.StatusNoContent).JSON(nil)
	return nil
}
