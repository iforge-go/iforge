package middleware

import (
	"time"

	"iforge/iforge/internal/contextutil"
	gitsvc "iforge/iforge/internal/git"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

// MetricsMiddleware records metrics for HTTP requests
func MetricsMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		path := c.Path()

		// Process request
		err := c.Next()

		duration := time.Since(start)
		statusCode := c.Response().StatusCode()
		method := c.Method()

		// Record metrics
		service.RecordRequest(method, path, statusCode, duration)

		// Log request
		logger := gitsvc.GetLogger()
		var username string
		if user := contextutil.GetUserFromContext(c); user != nil {
			username = user.UserName
		}
		logger.LogRequest(method, path, statusCode, duration, username)

		return err
	}
}

// LoggingMiddleware logs all HTTP requests
func LoggingMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		path := c.Path()

		// Process request
		err := c.Next()

		duration := time.Since(start)
		statusCode := c.Response().StatusCode()
		method := c.Method()

		// Log request
		logger := gitsvc.GetLogger()
		var username string
		if user := contextutil.GetUserFromContext(c); user != nil {
			username = user.UserName
		}
		logger.LogRequest(method, path, statusCode, duration, username)

		return err
	}
}
