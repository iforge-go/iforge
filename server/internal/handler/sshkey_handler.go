package handler

import (
	"net/http"
	"strconv"

	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type SSHKeyHandler struct {
	sshKeyService *service.SSHKeyService
}

func NewSSHKeyHandler(sshKeyService *service.SSHKeyService) *SSHKeyHandler {
	return &SSHKeyHandler{
		sshKeyService: sshKeyService,
	}
}

// ListSSHKeys lists all SSH keys for the current user
func (h *SSHKeyHandler) ListSSHKeys(c *fiber.Ctx) error {
	user, ok := getCurrentUser(c)
	if !ok {
		return nil
	}

	keys, err := h.sshKeyService.ListSSHKeys(user.UserName)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(keys)
	return nil
}

// GetSSHKey gets an SSH key by ID
func (h *SSHKeyHandler) GetSSHKey(c *fiber.Ctx) error {
	user, ok := getCurrentUser(c)
	if !ok {
		return nil
	}

	sshKeyID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid SSH key ID")
		return nil
	}

	key, err := h.sshKeyService.GetSSHKey(user.UserName, sshKeyID)
	if err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(key)
	return nil
}

// CreateSSHKey creates a new SSH key
func (h *SSHKeyHandler) CreateSSHKey(c *fiber.Ctx) error {
	user, ok := getCurrentUser(c)
	if !ok {
		return nil
	}

	var req struct {
		Title     string `json:"title" validate:"required"`
		PublicKey string `json:"publicKey" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return nil
	}

	key, err := h.sshKeyService.CreateSSHKey(user.UserName, req.Title, req.PublicKey)
	if err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	c.Status(http.StatusCreated).JSON(key)
	return nil
}

// DeleteSSHKey deletes an SSH key
func (h *SSHKeyHandler) DeleteSSHKey(c *fiber.Ctx) error {
	user, ok := getCurrentUser(c)
	if !ok {
		return nil
	}

	sshKeyID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid SSH key ID")
		return nil
	}

	if err := h.sshKeyService.DeleteSSHKey(user.UserName, sshKeyID); err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "SSH key deleted successfully"})
	return nil
}
