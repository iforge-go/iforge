package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type OIDCHandler struct {
	oidcService    *service.OIDCService
	accountService *service.AccountService
}

func NewOIDCHandler(oidcService *service.OIDCService, accountService *service.AccountService) *OIDCHandler {
	return &OIDCHandler{
		oidcService:    oidcService,
		accountService: accountService,
	}
}

func (h *OIDCHandler) GetOIDCSettings(c *fiber.Ctx) error {
	settings, err := h.oidcService.GetOIDCSettings()
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	// Don't expose client secret
	settings.ClientSecret = ""

	c.Status(http.StatusOK).JSON(settings)
	return nil
}

func (h *OIDCHandler) GetOIDCStatus(c *fiber.Ctx) error {
	settings, err := h.oidcService.GetOIDCSettings()
	if err != nil {
		c.Status(http.StatusOK).JSON(fiber.Map{"enabled": false})
		return nil
	}
	c.Status(http.StatusOK).JSON(fiber.Map{"enabled": settings.Enabled})
	return nil
}

func (h *OIDCHandler) UpdateOIDCSettings(c *fiber.Ctx) error {
	var settings service.OIDCSettings
	if err := c.BodyParser(&settings); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	// Preserve existing client secret if not provided
	existingSettings, err := h.oidcService.GetOIDCSettings()
	if err == nil && settings.ClientSecret == "" {
		settings.ClientSecret = existingSettings.ClientSecret
	}

	if err := h.oidcService.SetOIDCSettings(&settings); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "OIDC settings updated"})
	return nil
}

func (h *OIDCHandler) TestOIDCConnection(c *fiber.Ctx) error {
	if err := h.oidcService.TestConnection(); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "OIDC connection successful"})
	return nil
}

func (h *OIDCHandler) GetAuthURL(c *fiber.Ctx) error {
	// Generate random state
	state := generateRandomState()

	// Store state in service-side store so callbacks can be validated
	// even when the frontend and backend are on different origins (where
	// cookies would not be sent by the browser). We still set the cookie
	// for backwards compatibility with same-origin deployments.
	h.oidcService.SaveState(state)
	c.Cookie(&fiber.Cookie{
		Name:     "oidc_state",
		Value:    state,
		MaxAge:   300,
		Path:     "/",
		HTTPOnly: true,
		SameSite: "lax",
	})

	authURL, err := h.oidcService.GetAuthURL(state)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"authUrl": authURL, "state": state})
	return nil
}

// Callback handles the OAuth2 callback
func (h *OIDCHandler) Callback(c *fiber.Ctx) error {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		respondError(c, http.StatusBadRequest, "missing code or state")
		return nil
	}

	// Verify state. Prefer the service-side store (works across origins);
	// fall back to the cookie for legacy same-origin deployments.
	if !h.oidcService.ConsumeState(state) {
		storedState := c.Cookies("oidc_state")
		if storedState == "" || storedState != state {
			respondError(c, http.StatusBadRequest, "invalid state")
			return nil
		}
	}

	// Clear state cookie if present
	c.Cookie(&fiber.Cookie{
		Name:     "oidc_state",
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		HTTPOnly: true,
		SameSite: "lax",
	})

	// Exchange code for user info
	user, err := h.oidcService.Exchange(code)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	// Check if user exists, if not create new user
	username := user.PreferredName
	if username == "" {
		username = user.Subject
	}

	account, err := h.accountService.GetAccountByUsername(username)
	if err != nil {
		// User doesn't exist, create new account
		account, err = h.accountService.CreateAccount(
			username,
			"", // No password for OIDC users
			user.Name,
			user.Email,
			false,
			nil,
			nil,
		)
		if err != nil {
			if err == service.ErrReservedName {
				respondErrorWithMessageKey(c, http.StatusBadRequest, "errors.usernameReserved", "OIDC username is reserved, please contact administrator")
				return nil
			}
			respondError(c, http.StatusInternalServerError, "failed to create user: "+err.Error())
			return nil
		}
	}

	// Generate session token
	token := generateSessionToken()

	// Set token in cookie
	c.Cookie(&fiber.Cookie{
		Name:     "session",
		Value:    token,
		MaxAge:   86400,
		Path:     "/",
		HTTPOnly: true,
		SameSite: "lax",
	})

	c.Status(http.StatusOK).JSON(fiber.Map{
		"token": token,
		"user":  account,
	})
	return nil
}

// generateSessionToken generates a random session token
func generateSessionToken() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		// Return empty string on error instead of hardcoded fallback
		return ""
	}
	return hex.EncodeToString(bytes)
}

// generateRandomState generates a random state string
func generateRandomState() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Return empty string on error instead of hardcoded fallback
		return ""
	}
	return hex.EncodeToString(bytes)
}
