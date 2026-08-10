package handler

import (
	"net/http"
	"strconv"

	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type UserHandler struct {
	accountService *service.AccountService
}

func NewUserHandler(accountService *service.AccountService) *UserHandler {
	return &UserHandler{
		accountService: accountService,
	}
}

func (h *UserHandler) GetUser(c *fiber.Ctx) error {
	username := c.Params("username")
	user, err := h.accountService.GetAccountByUsername(username)
	if err != nil {
		respondError(c, http.StatusNotFound, "User not found")
		return nil
	}

	c.Status(http.StatusOK).JSON(user)
	return nil
}

func (h *UserHandler) SearchUsers(c *fiber.Ctx) error {
	keyword := c.Query("q")
	if keyword == "" {
		respondError(c, http.StatusBadRequest, "Query parameter 'q' is required")
		return nil
	}

	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	users, err := h.accountService.SearchUsers(keyword, limit)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"users": users})
	return nil
}

func (h *UserHandler) UpdateUser(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	username := c.Params("username")
	if user.UserName != username {
		respondError(c, http.StatusForbidden, "Cannot update other users")
		return nil
	}

	var req struct {
		FullName    *string `json:"fullName"`
		MailAddress *string `json:"mailAddress"`
		URL         *string `json:"url"`
		Description *string `json:"description"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body")
		return nil
	}

	updates := make(map[string]interface{})
	if req.FullName != nil {
		updates["fullName"] = *req.FullName
	}
	if req.MailAddress != nil {
		updates["mailAddress"] = *req.MailAddress
	}
	if req.URL != nil {
		updates["url"] = *req.URL
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}

	if err := h.accountService.UpdateAccount(username, updates); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to update user")
		return nil
	}

	updatedUser, err := h.accountService.GetAccountByUsername(username)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to fetch updated user")
		return nil
	}

	c.Status(http.StatusOK).JSON(updatedUser)
	return nil
}
