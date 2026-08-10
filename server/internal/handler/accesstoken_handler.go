package handler

import (
	"net/http"
	"strconv"

	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type AccessTokenHandler struct {
	accessTokenService *service.AccessTokenService
}

func NewAccessTokenHandler(accessTokenService *service.AccessTokenService) *AccessTokenHandler {
	return &AccessTokenHandler{
		accessTokenService: accessTokenService,
	}
}

func (h *AccessTokenHandler) ListAccessTokens(c *fiber.Ctx) error {
	user, ok := getCurrentUser(c)
	if !ok {
		return nil
	}

	tokens, err := h.accessTokenService.ListAccessTokens(user.UserName)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(tokens)
	return nil
}

func (h *AccessTokenHandler) GetAccessToken(c *fiber.Ctx) error {
	user, ok := getCurrentUser(c)
	if !ok {
		return nil
	}

	tokenID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid token ID")
		return nil
	}

	token, err := h.accessTokenService.GetAccessToken(user.UserName, tokenID)
	if err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(token)
	return nil
}

func (h *AccessTokenHandler) CreateAccessToken(c *fiber.Ctx) error {
	user, ok := getCurrentUser(c)
	if !ok {
		return nil
	}

	var req struct {
		Note string `json:"note" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return nil
	}

	token, tokenString, err := h.accessTokenService.CreateAccessToken(user.UserName, req.Note)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusCreated).JSON(fiber.Map{
		"token": token,
		"value": tokenString,
	})
	return nil
}

func (h *AccessTokenHandler) DeleteAccessToken(c *fiber.Ctx) error {
	user, ok := getCurrentUser(c)
	if !ok {
		return nil
	}

	tokenID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid token ID")
		return nil
	}

	if err := h.accessTokenService.DeleteAccessToken(tokenID, user.UserName); err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "access token deleted successfully"})
	return nil
}
