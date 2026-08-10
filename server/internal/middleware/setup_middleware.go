package middleware

import (
	"strings"

	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

// SetupMiddleware blocks access to all non-setup API routes when the system
// is not yet initialized. Setup routes and health checks are always allowed.
func SetupMiddleware(setupService *service.SetupService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if setupService.IsInitialized() {
			return c.Next()
		}

		path := c.Path()
		method := c.Method()

		if method == "OPTIONS" {
			return c.Next()
		}

		if strings.HasPrefix(path, "/api/v1/setup") {
			return c.Next()
		}

		if path == "/health" {
			return c.Next()
		}

		if path == "/api/v1/settings/ssh" || path == "/api/v1/settings/oidc/status" {
			return c.Next()
		}

		if path == "/api/v1/register/check-username" {
			return c.Next()
		}

		if strings.HasPrefix(path, "/api/v1") {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "System is not initialized. Please complete the setup wizard.",
			})
		}

		return c.Next()
	}
}
