package handler

import (
	"net/http"
	"strconv"
	"time"

	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type ScrumActivityHandler struct {
	activityService *service.ScrumActivityService
	accountService  *service.AccountService
}

func NewScrumActivityHandler(activityService *service.ScrumActivityService, accountService *service.AccountService) *ScrumActivityHandler {
	return &ScrumActivityHandler{
		activityService: activityService,
		accountService:  accountService,
	}
}

func (h *ScrumActivityHandler) GetActivities(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		return respondError(c, http.StatusUnauthorized, "Unauthorized")
	}

	projectID, err := decodeID(c.Params("projectSlug"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, "Invalid project ID")
	}

	entityType := c.Params("entityType")
	entityIDStr := c.Params("entityId")

	// Try numeric first, then try slug decode
	entityID, err := strconv.Atoi(entityIDStr)
	if err != nil {
		entityID, err = decodeID(entityIDStr)
		if err != nil {
			return respondError(c, http.StatusBadRequest, "Invalid entity ID")
		}
	}

	activities, err := h.activityService.GetActivities(projectID, entityType, entityID)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "Failed to get activities")
	}

	// Sideload: batch-prefetch operator avatar info to avoid per-user /users/:name
	// requests from the frontend. The task detail drawer's ActivityTimeline shows
	// each operator's avatar; originally only letter avatars were available, but
	// after sideloading we can serve the real avatar image URL and fullName.
	var userNames []string
	seen := make(map[string]bool)
	for _, a := range activities {
		if a.UserName != "" && !seen[a.UserName] {
			seen[a.UserName] = true
			userNames = append(userNames, a.UserName)
		}
	}
	avatarInfo, _ := h.accountService.GetUserAvatarInfo(userNames)
	participants := h.accountService.BuildParticipants(avatarInfo, userNames)

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"activities":   activities,
		"participants": participants,
	})
}

// ptrStringValue converts a *string to "" when nil, otherwise the dereferenced value.
// Used to render pointer fields as comparable strings for activity logging.
func ptrStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ptrIntValue converts a *int to "" when nil, otherwise its decimal representation.
// Used to render pointer fields as comparable strings for activity logging.
func ptrIntValue(i *int) string {
	if i == nil {
		return ""
	}
	return strconv.Itoa(*i)
}

// ptrFloat64Value converts a *float64 to "" when nil, otherwise its decimal representation.
// Used to render pointer fields as comparable strings for activity logging.
func ptrFloat64Value(f *float64) string {
	if f == nil {
		return ""
	}
	return strconv.FormatFloat(*f, 'f', -1, 64)
}

// ptrIntEqual compares two *int pointers by value.
// nil == nil -> true; nil != non-nil -> false; otherwise compare dereferenced values.
// Used before activity logging to detect whether a *int field changed (e.g. Story.EpicID).
func ptrIntEqual(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// ptrTimeValue converts a *time.Time to "" when nil, otherwise its RFC3339 representation.
// Used to render pointer fields as comparable strings for activity logging.
func ptrTimeValue(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}
