package handler

import (
	"net/http"

	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type NotificationHandler struct {
	notificationService *service.NotificationService
	accountService      *service.AccountService
}

func NewNotificationHandler(notificationService *service.NotificationService, accountService *service.AccountService) *NotificationHandler {
	return &NotificationHandler{
		notificationService: notificationService,
		accountService:      accountService,
	}
}

func (h *NotificationHandler) ListNotifications(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return nil
	}

	unreadOnly := c.Query("unread") == "true"

	notifications, err := h.notificationService.ListNotifications(user.UserName, unreadOnly)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	for i := range notifications {
		notifications[i].Slug = encodeID(notifications[i].NotificationID)
		if notifications[i].ProjectID != nil {
			notifications[i].ProjectSlug = encodeID(*notifications[i].ProjectID)
		}
		if notifications[i].StoryID != nil {
			notifications[i].StorySlug = encodeID(*notifications[i].StoryID)
		}
	}

	var userNames []string
	seen := make(map[string]bool)
	for _, n := range notifications {
		if n.Actor != "" && !seen[n.Actor] {
			seen[n.Actor] = true
			userNames = append(userNames, n.Actor)
		}
	}
	avatarInfo, _ := h.accountService.GetUserAvatarInfo(userNames)
	participants := h.accountService.BuildParticipants(avatarInfo, userNames)

	c.Status(http.StatusOK).JSON(fiber.Map{
		"notifications": notifications,
		"participants":  participants,
	})
	return nil
}

func (h *NotificationHandler) GetUnreadCount(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return nil
	}

	count, err := h.notificationService.GetUnreadCount(user.UserName)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"count": count})
	return nil
}

func (h *NotificationHandler) MarkAsRead(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return nil
	}

	notificationID, err := decodeID(c.Params("notificationSlug"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid notification ID")
		return nil
	}

	if err := h.notificationService.MarkAsRead(notificationID, user.UserName); err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "notification marked as read"})
	return nil
}

// MarkAllAsRead marks all notifications as read for the current user
func (h *NotificationHandler) MarkAllAsRead(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return nil
	}

	if err := h.notificationService.MarkAllAsRead(user.UserName); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "all notifications marked as read"})
	return nil
}

// DeleteNotification deletes a notification
func (h *NotificationHandler) DeleteNotification(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return nil
	}

	notificationID, err := decodeID(c.Params("notificationSlug"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid notification ID")
		return nil
	}

	if err := h.notificationService.DeleteNotification(notificationID, user.UserName); err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "notification deleted"})
	return nil
}
