package middleware

import (
	"strings"

	"iforge/iforge/internal/model"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

const RunnerAuthKey = "runner"

func RunnerAuthMiddleware(runnerService *service.RunnerService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing authorization header"})
		}

		token := authHeader
		if len(token) >= 7 && strings.EqualFold(token[:7], "bearer ") {
			token = strings.TrimSpace(token[7:])
			if token == "" {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "empty bearer token"})
			}
		} else {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid authorization format, expected Bearer token"})
		}

		runner, err := runnerService.GetRunnerByToken(token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid runner token"})
		}

		c.Locals(RunnerAuthKey, runner)
		return c.Next()
	}
}

func GetRunnerFromContext(c *fiber.Ctx) *model.Runner {
	val := c.Locals(RunnerAuthKey)
	if val == nil {
		return nil
	}
	if runner, ok := val.(*model.Runner); ok {
		return runner
	}
	return nil
}
