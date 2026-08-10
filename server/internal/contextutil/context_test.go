package contextutil

import (
	"context"
	"net/http/httptest"
	"testing"

	"iforge/iforge/internal/model"

	"github.com/gofiber/fiber/v2"
)

func TestGetUserFromStdContext(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected *model.Account
	}{
		{
			name:     "empty context",
			ctx:      context.Background(),
			expected: nil,
		},
		{
			name: "context with user",
			ctx: context.WithValue(context.Background(), UserContextKey, &model.Account{
				UserName: "alice",
			}),
			expected: &model.Account{UserName: "alice"},
		},
		{
			name:     "context with wrong type",
			ctx:      context.WithValue(context.Background(), UserContextKey, "not a user"),
			expected: nil,
		},
		{
			name:     "context with nil user",
			ctx:      context.WithValue(context.Background(), UserContextKey, (*model.Account)(nil)),
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetUserFromStdContext(tt.ctx)
			if got == nil && tt.expected == nil {
				return
			}
			if got == nil || tt.expected == nil {
				t.Errorf("GetUserFromStdContext() = %v, want %v", got, tt.expected)
				return
			}
			if got.UserName != tt.expected.UserName {
				t.Errorf("GetUserFromStdContext().UserName = %v, want %v", got.UserName, tt.expected.UserName)
			}
		})
	}
}

func TestUserContextKey(t *testing.T) {
	if UserContextKey != contextKey(UserContextKeyStr) {
		t.Errorf("UserContextKey = %v, want %v", UserContextKey, UserContextKeyStr)
	}
}

func TestFiberUserKey(t *testing.T) {
	if FiberUserKey != "user" {
		t.Errorf("FiberUserKey = %v, want 'user'", FiberUserKey)
	}
}

func TestGetUserFromContext(t *testing.T) {
	tests := []struct {
		name     string
		setupCtx func(*fiber.Ctx)
		expected *model.Account
	}{
		{
			name: "no user in context",
			setupCtx: func(c *fiber.Ctx) {
				// Do not set any value
			},
			expected: nil,
		},
		{
			name: "user in context",
			setupCtx: func(c *fiber.Ctx) {
				user := &model.Account{UserName: "bob", IsAdmin: true}
				c.Locals(FiberUserKey, user)
			},
			expected: &model.Account{UserName: "bob", IsAdmin: true},
		},
		{
			name: "wrong type in context",
			setupCtx: func(c *fiber.Ctx) {
				c.Locals(FiberUserKey, "not a user")
			},
			expected: nil,
		},
		{
			name: "nil user in context",
			setupCtx: func(c *fiber.Ctx) {
				c.Locals(FiberUserKey, (*model.Account)(nil))
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/test", func(c *fiber.Ctx) error {
				tt.setupCtx(c)
				got := GetUserFromContext(c)
				if got == nil && tt.expected == nil {
					return c.SendString("ok")
				}
				if got == nil || tt.expected == nil {
					t.Errorf("GetUserFromContext() = %v, want %v", got, tt.expected)
					return c.SendString("fail")
				}
				if got.UserName != tt.expected.UserName {
					t.Errorf("GetUserFromContext().UserName = %v, want %v", got.UserName, tt.expected.UserName)
				}
				if got.IsAdmin != tt.expected.IsAdmin {
					t.Errorf("GetUserFromContext().IsAdmin = %v, want %v", got.IsAdmin, tt.expected.IsAdmin)
				}
				return c.SendString("ok")
			})

			req := httptest.NewRequest("GET", "/test", nil)
			resp, _ := app.Test(req)
			if resp.StatusCode != 200 {
				t.Errorf("Request failed with status %d", resp.StatusCode)
			}
		})
	}
}
