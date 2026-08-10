package handler

import (
	"net/http"
	"strconv"

	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type GPGKeyHandler struct {
	gpgKeyService *service.GPGKeyService
}

func NewGPGKeyHandler(gpgKeyService *service.GPGKeyService) *GPGKeyHandler {
	return &GPGKeyHandler{
		gpgKeyService: gpgKeyService,
	}
}

func (h *GPGKeyHandler) ListGPGKeys(c *fiber.Ctx) error {
	user, ok := getCurrentUser(c)
	if !ok {
		return nil
	}

	keys, err := h.gpgKeyService.ListGPGKeys(user.UserName)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(keys)
	return nil
}

func (h *GPGKeyHandler) GetGPGKey(c *fiber.Ctx) error {
	user, ok := getCurrentUser(c)
	if !ok {
		return nil
	}

	keyID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid key ID")
		return nil
	}

	key, err := h.gpgKeyService.GetGPGKey(user.UserName, keyID)
	if err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(key)
	return nil
}

func (h *GPGKeyHandler) AddGPGKey(c *fiber.Ctx) error {
	user, ok := getCurrentUser(c)
	if !ok {
		return nil
	}

	var req struct {
		Title     string `json:"title" validate:"required"`
		PublicKey string `json:"publicKey" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	key, err := h.gpgKeyService.AddGPGKey(user.UserName, req.Title, req.PublicKey)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusCreated).JSON(key)
	return nil
}

func (h *GPGKeyHandler) DeleteGPGKey(c *fiber.Ctx) error {
	user, ok := getCurrentUser(c)
	if !ok {
		return nil
	}

	keyID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid key ID")
		return nil
	}

	if err := h.gpgKeyService.DeleteGPGKey(user.UserName, keyID); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "GPG key deleted"})
	return nil
}
