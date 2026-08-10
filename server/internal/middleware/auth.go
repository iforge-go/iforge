package middleware

import (
	"os"
	"strings"

	"iforge/iforge/internal/contextutil"
	gitsvc "iforge/iforge/internal/git"
	"iforge/iforge/internal/service"
	"iforge/iforge/internal/session"

	"github.com/gofiber/fiber/v2"
)

// AuthMiddleware handles authentication for Fiber
func AuthMiddleware(accountService *service.AccountService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Try session cookie first
		sessionCookie := c.Cookies("session")
		if sessionCookie != "" {
			username, err := session.Verify(sessionCookie)
			if err == nil {
				user, err := accountService.GetAccountByUsername(username)
				if err == nil && !user.IsRemoved {
					c.Locals(contextutil.FiberUserKey, user)
					return c.Next()
				}
			}
			// Invalid or expired session, clear cookie
			c.ClearCookie("session")
		}

		// Fallback to Authorization header (signed session token)
		authHeader := c.Get("Authorization")
		if authHeader != "" {
			token := authHeader
			if len(token) > 7 && token[:7] == "Bearer " {
				token = token[7:]
			}
			username, err := session.Verify(token)
			if err != nil {
				// Log verification error for debugging
				gitsvc.GetLogger().Warn("Session verification failed", map[string]interface{}{"error": err.Error()})
			} else {
				user, err := accountService.GetAccountByUsername(username)
				if err != nil {
					gitsvc.GetLogger().Warn("GetAccountByUsername failed", map[string]interface{}{"username": username, "error": err.Error()})
				} else if user.IsRemoved {
					gitsvc.GetLogger().Warn("Removed user attempted access", map[string]interface{}{"username": username})
				} else {
					c.Locals(contextutil.FiberUserKey, user)
					return c.Next()
				}
			}
		}

		return c.Next()
	}
}

// RequireAuth middleware requires authentication
func RequireAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		user := contextutil.GetUserFromContext(c)
		if user == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
		}
		return c.Next()
	}
}

// RequireAdmin middleware requires admin privileges
func RequireAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		user := contextutil.GetUserFromContext(c)
		if user == nil || !user.IsAdmin {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
		}
		return c.Next()
	}
}

// CORSMiddleware handles CORS for Fiber
func CORSMiddleware() fiber.Handler {
	allowedOrigins := getAllowedOrigins()
	return func(c *fiber.Ctx) error {
		origin := c.Get("Origin")
		if origin != "" && isAllowedOrigin(origin, allowedOrigins) {
			c.Set("Access-Control-Allow-Origin", origin)
			c.Set("Access-Control-Allow-Credentials", "true")
		}
		c.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Method() == "OPTIONS" {
			return c.SendStatus(fiber.StatusOK)
		}

		return c.Next()
	}
}

func getAllowedOrigins() map[string]bool {
	origins := map[string]bool{}
	if env := os.Getenv("IFORGE_CORS_ORIGINS"); env != "" {
		if strings.TrimSpace(env) == "*" {
			origins["*"] = true
			return origins
		}
		for _, o := range strings.Split(env, ",") {
			o = strings.TrimSpace(o)
			if o != "" {
				origins[o] = true
			}
		}
	}
	if len(origins) == 0 {
		origins["http://localhost:3001"] = true
		origins["http://localhost:3000"] = true
		origins["http://localhost:8081"] = true
	}
	return origins
}

// isAllowedOrigin checks if the given origin is in the allowed set.
func isAllowedOrigin(origin string, allowed map[string]bool) bool {
	if allowed["*"] {
		return true
	}
	return allowed[origin]
}
