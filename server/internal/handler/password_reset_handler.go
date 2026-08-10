package handler

import (
	"iforge/iforge/internal/service"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

type PasswordResetHandler struct {
	passwordResetService *service.PasswordResetService
	accountService       *service.AccountService
	mailService          *service.MailService
	rateLimit            *resetRateLimit
}

type resetRateLimit struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
}

func newResetRateLimit() *resetRateLimit {
	return &resetRateLimit{
		attempts: make(map[string][]time.Time),
	}
}

func (r *resetRateLimit) check(key string, maxAttempts int, window time.Duration) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-window)

	// Clean old attempts
	attempts := r.attempts[key]
	var valid []time.Time
	for _, t := range attempts {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= maxAttempts {
		r.attempts[key] = valid
		return false
	}

	r.attempts[key] = append(valid, now)
	return true
}

func NewPasswordResetHandler(
	passwordResetService *service.PasswordResetService,
	accountService *service.AccountService,
	mailService *service.MailService,
) *PasswordResetHandler {
	return &PasswordResetHandler{
		passwordResetService: passwordResetService,
		accountService:       accountService,
		mailService:          mailService,
		rateLimit:            newResetRateLimit(),
	}
}

func (h *PasswordResetHandler) RequestReset(c *fiber.Ctx) error {
	clientIP := c.IP()
	if !h.rateLimit.check(clientIP, 5, time.Hour) {
		respondError(c, http.StatusTooManyRequests, "Too many requests, please try again later")
		return nil
	}

	var req struct {
		Email string `json:"email" validate:"required,email"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	// Find user by email
	user, err := h.accountService.GetAccountByEmail(req.Email)
	if err != nil {
		// Don't reveal if email exists or not
		c.Status(http.StatusOK).JSON(fiber.Map{"message": "If the email exists, a reset link has been sent"})
		return nil
	}

	// Generate reset token
	token, err := h.passwordResetService.CreateResetToken(user.UserName)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to create reset token")
		return nil
	}

	// Build reset URL from request Origin (or Referer as fallback) so the
	// link points to the actual frontend host instead of a hardcoded value.
	frontendBase := c.Get("Origin")
	if frontendBase == "" {
		frontendBase = c.Get("Referer")
		if idx := strings.Index(frontendBase, "?"); idx >= 0 {
			frontendBase = frontendBase[:idx]
		}
		// Strip trailing slash from referer path component, keep origin only
		if idx := strings.Index(frontendBase[8:], "/"); idx >= 0 {
			// Keep scheme://host[:port]
			schemeLen := 8
			if strings.HasPrefix(frontendBase, "https://") {
				schemeLen = 9
			}
			frontendBase = frontendBase[:schemeLen+idx]
		}
	}
	if frontendBase == "" {
		// Final fallback: derive from request host
		scheme := c.Protocol()
		frontendBase = scheme + "://" + c.Hostname() + ":3001"
	}

	resetURL := frontendBase + "/reset-password?token=" + token
	subject := "Password Reset Request"
	body := "Click the following link to reset your password:\n\n" + resetURL + "\n\nThis link will expire in 24 hours."

	msg := &service.EmailMessage{
		To:      []string{user.MailAddress},
		Subject: subject,
		Body:    body,
		IsHTML:  false,
	}

	err = h.mailService.SendMail(msg)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to send reset email")
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "If the email exists, a reset link has been sent"})
	return nil
}

// ValidateResetToken validates a password reset token
func (h *PasswordResetHandler) ValidateResetToken(c *fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		respondError(c, http.StatusBadRequest, "Token is required")
		return nil
	}

	_, err := h.passwordResetService.ValidateToken(token)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid or expired token")
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{
		"valid": true,
	})
	return nil
}

// ResetPassword handles password reset with token
func (h *PasswordResetHandler) ResetPassword(c *fiber.Ctx) error {
	var req struct {
		Token       string `json:"token" validate:"required"`
		NewPassword string `json:"newPassword" validate:"required,min=8"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	err := h.passwordResetService.ResetPassword(req.Token, req.NewPassword)
	if err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Password reset successfully"})
	return nil
}
