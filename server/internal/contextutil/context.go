package contextutil

import (
	"context"

	"iforge/iforge/internal/model"

	"github.com/gofiber/fiber/v2"
)

// FiberUserKey is the key used to store user in Fiber context
const FiberUserKey = "user"

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

// UserContextKey is the key used to store user in standard context
const UserContextKey contextKey = UserContextKeyStr

// UserContextKeyStr is the string representation of UserContextKey
const UserContextKeyStr = "user"

// GetUserFromContext retrieves user from Fiber context
func GetUserFromContext(c *fiber.Ctx) *model.Account {
	val := c.Locals(FiberUserKey)
	if val == nil {
		return nil
	}
	user, ok := val.(*model.Account)
	if !ok {
		return nil
	}
	return user
}

// GetUserFromStdContext retrieves user from standard context
func GetUserFromStdContext(ctx context.Context) *model.Account {
	user, ok := ctx.Value(UserContextKey).(*model.Account)
	if !ok {
		return nil
	}
	return user
}
