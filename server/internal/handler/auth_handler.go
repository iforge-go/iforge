package handler

import (
	"net/http"
	"os"
	"strings"
	"time"

	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/service"
	"iforge/iforge/internal/session"

	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	accountService        *service.AccountService
	systemSettingsService *service.SystemSettingsService
}

func NewAuthHandler(accountService *service.AccountService, systemSettingsService *service.SystemSettingsService) *AuthHandler {
	return &AuthHandler{
		accountService:        accountService,
		systemSettingsService: systemSettingsService,
	}
}

type LoginRequest struct {
	Username string `json:"userName" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	Token string      `json:"token"`
	User  interface{} `json:"user"`
}

type RegisterRequest struct {
	Username    string `json:"username" validate:"required"`
	Password    string `json:"password" validate:"required"`
	FullName    string `json:"fullName" validate:"required"`
	MailAddress string `json:"mailAddress" validate:"required"`
}

// @Router /api/v1/login [post]
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		respondErrorWithMessageKey(c, http.StatusBadRequest, "errors.invalidRequest", "Invalid request body")
		return nil
	}

	user, err := h.accountService.Authenticate(req.Username, req.Password)
	if err != nil {
		respondErrorWithMessageKey(c, http.StatusUnauthorized, "errors.invalidCredentials", "Invalid credentials")
		return nil
	}

	signedToken := session.Sign(user.UserName, 24*time.Hour)
	secureCookie := os.Getenv("IFORGE_SECURE_COOKIE") == "true"
	cookie := &fiber.Cookie{
		Name:     "session",
		Value:    signedToken,
		MaxAge:   86400,
		Path:     "/",
		HTTPOnly: true,
		Secure:   secureCookie,
		SameSite: "strict",
	}
	c.Cookie(cookie)

	c.Status(http.StatusOK).JSON(LoginResponse{
		Token: signedToken,
		User:  user,
	})
	return nil
}

// @Router /api/v1/register [post]
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	// Check if registration is allowed
	settings, err := h.systemSettingsService.GetGeneralSettings()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to get settings")
		return nil
	}
	if !settings.AllowRegistration {
		respondErrorWithMessageKey(c, http.StatusForbidden, "errors.registrationDisabled", "Registration is disabled")
		return nil
	}

	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		respondErrorWithMessageKey(c, http.StatusBadRequest, "errors.invalidRequest", "Invalid request body")
		return nil
	}

	user, err := h.accountService.CreateAccount(req.Username, req.Password, req.FullName, req.MailAddress, false, nil, nil)
	if err != nil {
		if err == service.ErrUserAlreadyExists {
			respondErrorWithMessageKey(c, http.StatusConflict, "errors.userExists", "User already exists")
			return nil
		}
		if err == service.ErrReservedName {
			respondErrorWithMessageKey(c, http.StatusBadRequest, "errors.usernameReserved", "This username is reserved and cannot be used")
			return nil
		}
		respondErrorWithMessageKey(c, http.StatusInternalServerError, "errors.userCreateFailed", "Failed to create user")
		return nil
	}

	c.Status(http.StatusCreated).JSON(user)
	return nil
}

func (h *AuthHandler) CheckUsername(c *fiber.Ctx) error {
	username := strings.TrimSpace(c.Query("username"))
	if username == "" {
		return c.Status(http.StatusOK).JSON(fiber.Map{"available": false, "reason": "invalid"})
	}
	if len(username) < 3 || len(username) > 30 || !isValidUsername(username) {
		return c.Status(http.StatusOK).JSON(fiber.Map{"available": false, "reason": "invalid"})
	}
	if service.IsReservedName(username) {
		return c.Status(http.StatusOK).JSON(fiber.Map{"available": false, "reason": "reserved"})
	}
	if _, err := h.accountService.GetAccountByUsername(username); err == nil {
		return c.Status(http.StatusOK).JSON(fiber.Map{"available": false, "reason": "taken"})
	}
	return c.Status(http.StatusOK).JSON(fiber.Map{"available": true, "reason": ""})
}

func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	cookie := &fiber.Cookie{
		Name:     "session",
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		HTTPOnly: true,
		SameSite: "lax",
	}
	c.Cookie(cookie)
	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Logged out"})
	return nil
}

// @Router /api/v1/user [get]
func (h *AuthHandler) GetCurrentUser(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondErrorWithMessageKey(c, http.StatusUnauthorized, "errors.unauthorized", "Unauthorized")
		return nil
	}

	c.Status(http.StatusOK).JSON(user)
	return nil
}

// @Router /api/v1/user/password [put]
func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondErrorWithMessageKey(c, http.StatusUnauthorized, "errors.unauthorized", "Unauthorized")
		return nil
	}

	var req struct {
		CurrentPassword string `json:"currentPassword" validate:"required"`
		NewPassword     string `json:"newPassword" validate:"required,min=6"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondErrorWithMessageKey(c, http.StatusBadRequest, "errors.invalidRequest", "Invalid request body")
		return nil
	}

	_, err := h.accountService.Authenticate(user.UserName, req.CurrentPassword)
	if err != nil {
		respondErrorWithMessageKey(c, http.StatusUnauthorized, "errors.passwordIncorrect", "Current password is incorrect")
		return nil
	}

	updates := map[string]interface{}{
		"password": req.NewPassword,
	}

	if err := h.accountService.UpdateAccount(user.UserName, updates); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to change password")
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Password changed successfully"})
	return nil
}

// @Router /api/v1/user [put]
func (h *AuthHandler) UpdateCurrentUser(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
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

	if err := h.accountService.UpdateAccount(user.UserName, updates); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to update user")
		return nil
	}

	updatedUser, err := h.accountService.GetAccountByUsername(user.UserName)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to fetch updated user")
		return nil
	}

	c.Status(http.StatusOK).JSON(updatedUser)
	return nil
}
