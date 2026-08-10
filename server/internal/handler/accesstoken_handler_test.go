package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/model"

	"github.com/gofiber/fiber/v2"
)

func TestNewAccessTokenHandler(t *testing.T) {
	handler := NewAccessTokenHandler(nil)
	if handler == nil {
		t.Error("NewAccessTokenHandler returned nil")
	}
}

func TestAccessTokenHandler_ListAccessTokens_Unauthorized(t *testing.T) {
	app := fiber.New()
	handler := NewAccessTokenHandler(nil)

	app.Get("/access-tokens", handler.ListAccessTokens)

	req := httptest.NewRequest("GET", "/access-tokens", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestAccessTokenHandler_GetAccessToken_Unauthorized(t *testing.T) {
	app := fiber.New()
	handler := NewAccessTokenHandler(nil)

	app.Get("/access-tokens/:id", handler.GetAccessToken)

	req := httptest.NewRequest("GET", "/access-tokens/1", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestAccessTokenHandler_GetAccessToken_InvalidID(t *testing.T) {
	app := fiber.New()
	handler := NewAccessTokenHandler(nil)

	app.Use(func(c *fiber.Ctx) error {
		user := &model.Account{UserName: "testuser"}
		c.Locals(contextutil.FiberUserKey, user)
		return c.Next()
	})
	app.Get("/access-tokens/:id", handler.GetAccessToken)

	req := httptest.NewRequest("GET", "/access-tokens/invalid", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestAccessTokenHandler_CreateAccessToken_Unauthorized(t *testing.T) {
	app := fiber.New()
	handler := NewAccessTokenHandler(nil)

	app.Post("/access-tokens", handler.CreateAccessToken)

	body := map[string]string{
		"note": "Test Token",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/access-tokens", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestAccessTokenHandler_CreateAccessToken_InvalidBody(t *testing.T) {
	app := fiber.New()
	handler := NewAccessTokenHandler(nil)

	app.Use(func(c *fiber.Ctx) error {
		user := &model.Account{UserName: "testuser"}
		c.Locals(contextutil.FiberUserKey, user)
		return c.Next()
	})
	app.Post("/access-tokens", handler.CreateAccessToken)

	req := httptest.NewRequest("POST", "/access-tokens", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestAccessTokenHandler_DeleteAccessToken_Unauthorized(t *testing.T) {
	app := fiber.New()
	handler := NewAccessTokenHandler(nil)

	app.Delete("/access-tokens/:id", handler.DeleteAccessToken)

	req := httptest.NewRequest("DELETE", "/access-tokens/1", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestAccessTokenHandler_DeleteAccessToken_InvalidID(t *testing.T) {
	app := fiber.New()
	handler := NewAccessTokenHandler(nil)

	app.Use(func(c *fiber.Ctx) error {
		user := &model.Account{UserName: "testuser"}
		c.Locals(contextutil.FiberUserKey, user)
		return c.Next()
	})
	app.Delete("/access-tokens/:id", handler.DeleteAccessToken)

	req := httptest.NewRequest("DELETE", "/access-tokens/invalid", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}
