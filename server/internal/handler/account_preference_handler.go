package handler

import (
	"net/http"

	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type AccountPreferenceHandler struct {
	preferenceService *service.AccountPreferenceService
}

func NewAccountPreferenceHandler(preferenceService *service.AccountPreferenceService) *AccountPreferenceHandler {
	return &AccountPreferenceHandler{
		preferenceService: preferenceService,
	}
}

func (h *AccountPreferenceHandler) GetPreference(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	pref, err := h.preferenceService.GetPreference(user.UserName)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to get preferences")
		return nil
	}

	c.Status(http.StatusOK).JSON(pref)
	return nil
}

func (h *AccountPreferenceHandler) UpdatePreference(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	var req struct {
		HighlighterTheme string `json:"highlighterTheme"`
		Notification     *bool  `json:"notification"`
		Timezone         string `json:"timezone"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request")
		return nil
	}

	pref, err := h.preferenceService.GetPreference(user.UserName)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to get preferences")
		return nil
	}

	if req.HighlighterTheme != "" {
		pref.HighlighterTheme = req.HighlighterTheme
	}
	if req.Notification != nil {
		pref.Notification = *req.Notification
	}
	if req.Timezone != "" {
		pref.Timezone = req.Timezone
	}

	if err := h.preferenceService.UpdatePreference(pref); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to update preferences")
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Preferences updated successfully"})
	return nil
}
