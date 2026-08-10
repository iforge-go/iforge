package handler

import (
	"errors"
	"net/http"
	"regexp"

	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type SetupHandler struct {
	setupService          *service.SetupService
	accountService        *service.AccountService
	systemSettingsService *service.SystemSettingsService
}

func NewSetupHandler(
	setupService *service.SetupService,
	accountService *service.AccountService,
	systemSettingsService *service.SystemSettingsService,
) *SetupHandler {
	return &SetupHandler{
		setupService:          setupService,
		accountService:        accountService,
		systemSettingsService: systemSettingsService,
	}
}

type SetupStatusResponse struct {
	Initialized bool `json:"initialized"`
}

type SetupRequest struct {
	AdminUsername string `json:"adminUsername"`
	AdminPassword string `json:"adminPassword"`
	AdminEmail    string `json:"adminEmail"`
}

type SetupResponse struct {
	Message string `json:"message"`
}

// @Router /api/v1/setup/status [get]
func (h *SetupHandler) GetSetupStatus(c *fiber.Ctx) error {
	return c.Status(http.StatusOK).JSON(SetupStatusResponse{
		Initialized: h.setupService.IsInitialized(),
	})
}

// @Router /api/v1/setup [post]
func (h *SetupHandler) Setup(c *fiber.Ctx) error {
	if h.setupService.IsInitialized() {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "System is already initialized",
		})
	}

	var req SetupRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if len(req.AdminUsername) < 3 || len(req.AdminUsername) > 30 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Username must be between 3 and 30 characters",
		})
	}
	if !isValidUsername(req.AdminUsername) {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Username may only contain alphanumeric characters, hyphens, and underscores",
		})
	}
	if len(req.AdminPassword) < 8 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Password must be at least 8 characters",
		})
	}
	if !isValidEmail(req.AdminEmail) {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid email address",
		})
	}

	_, err := h.accountService.CreateAccount(
		req.AdminUsername,
		req.AdminPassword,
		req.AdminUsername,
		req.AdminEmail,
		true,
		nil, nil,
	)
	if err != nil {
		if errors.Is(err, service.ErrUserAlreadyExists) {
			return c.Status(http.StatusConflict).JSON(fiber.Map{
				"error":      "Username already exists",
				"messageKey": "errors.userExists",
			})
		}
		if errors.Is(err, service.ErrReservedName) {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{
				"error":      "This username is reserved and cannot be used",
				"messageKey": "errors.usernameReserved",
			})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error":      "Failed to create admin account",
			"messageKey": "errors.adminCreateFailed",
		})
	}

	if err := h.systemSettingsService.SetSetting("initialized", "true"); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to mark system as initialized",
		})
	}

	h.setupService.RefreshInitialized()

	return c.Status(http.StatusCreated).JSON(SetupResponse{
		Message: "Setup completed successfully",
	})
}

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func isValidUsername(s string) bool {
	return usernameRegex.MatchString(s)
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func isValidEmail(s string) bool {
	return emailRegex.MatchString(s)
}
