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

func TestNewGPGKeyHandler(t *testing.T) {
	handler := NewGPGKeyHandler(nil)
	if handler == nil {
		t.Error("NewGPGKeyHandler returned nil")
	}
}

func TestGPGKeyHandler_ListGPGKeys_Unauthorized(t *testing.T) {
	app := fiber.New()
	handler := NewGPGKeyHandler(nil)

	app.Get("/gpg-keys", handler.ListGPGKeys)

	req := httptest.NewRequest("GET", "/gpg-keys", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestGPGKeyHandler_GetGPGKey_Unauthorized(t *testing.T) {
	app := fiber.New()
	handler := NewGPGKeyHandler(nil)

	app.Get("/gpg-keys/:id", handler.GetGPGKey)

	req := httptest.NewRequest("GET", "/gpg-keys/1", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestGPGKeyHandler_GetGPGKey_InvalidID(t *testing.T) {
	app := fiber.New()
	handler := NewGPGKeyHandler(nil)

	app.Use(func(c *fiber.Ctx) error {
		user := &model.Account{UserName: "testuser"}
		c.Locals(contextutil.FiberUserKey, user)
		return c.Next()
	})
	app.Get("/gpg-keys/:id", handler.GetGPGKey)

	req := httptest.NewRequest("GET", "/gpg-keys/invalid", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestGPGKeyHandler_AddGPGKey_Unauthorized(t *testing.T) {
	app := fiber.New()
	handler := NewGPGKeyHandler(nil)

	app.Post("/gpg-keys", handler.AddGPGKey)

	body := map[string]string{
		"title":     "Test Key",
		"publicKey": "-----BEGIN PGP PUBLIC KEY BLOCK-----",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/gpg-keys", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestGPGKeyHandler_AddGPGKey_InvalidBody(t *testing.T) {
	app := fiber.New()
	handler := NewGPGKeyHandler(nil)

	app.Use(func(c *fiber.Ctx) error {
		user := &model.Account{UserName: "testuser"}
		c.Locals(contextutil.FiberUserKey, user)
		return c.Next()
	})
	app.Post("/gpg-keys", handler.AddGPGKey)

	req := httptest.NewRequest("POST", "/gpg-keys", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestGPGKeyHandler_DeleteGPGKey_Unauthorized(t *testing.T) {
	app := fiber.New()
	handler := NewGPGKeyHandler(nil)

	app.Delete("/gpg-keys/:id", handler.DeleteGPGKey)

	req := httptest.NewRequest("DELETE", "/gpg-keys/1", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestGPGKeyHandler_DeleteGPGKey_InvalidID(t *testing.T) {
	app := fiber.New()
	handler := NewGPGKeyHandler(nil)

	app.Use(func(c *fiber.Ctx) error {
		user := &model.Account{UserName: "testuser"}
		c.Locals(contextutil.FiberUserKey, user)
		return c.Next()
	})
	app.Delete("/gpg-keys/:id", handler.DeleteGPGKey)

	req := httptest.NewRequest("DELETE", "/gpg-keys/invalid", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}
