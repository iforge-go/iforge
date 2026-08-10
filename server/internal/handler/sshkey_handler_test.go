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

func TestNewSSHKeyHandler(t *testing.T) {
	handler := NewSSHKeyHandler(nil)
	if handler == nil {
		t.Error("NewSSHKeyHandler returned nil")
	}
}

func TestSSHKeyHandler_ListSSHKeys_Unauthorized(t *testing.T) {
	app := fiber.New()
	handler := NewSSHKeyHandler(nil)

	app.Get("/ssh-keys", handler.ListSSHKeys)

	req := httptest.NewRequest("GET", "/ssh-keys", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestSSHKeyHandler_GetSSHKey_Unauthorized(t *testing.T) {
	app := fiber.New()
	handler := NewSSHKeyHandler(nil)

	app.Get("/ssh-keys/:id", handler.GetSSHKey)

	req := httptest.NewRequest("GET", "/ssh-keys/1", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestSSHKeyHandler_GetSSHKey_InvalidID(t *testing.T) {
	app := fiber.New()
	handler := NewSSHKeyHandler(nil)

	app.Use(func(c *fiber.Ctx) error {
		user := &model.Account{UserName: "testuser"}
		c.Locals(contextutil.FiberUserKey, user)
		return c.Next()
	})
	app.Get("/ssh-keys/:id", handler.GetSSHKey)

	req := httptest.NewRequest("GET", "/ssh-keys/invalid", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestSSHKeyHandler_CreateSSHKey_Unauthorized(t *testing.T) {
	app := fiber.New()
	handler := NewSSHKeyHandler(nil)

	app.Post("/ssh-keys", handler.CreateSSHKey)

	body := map[string]string{
		"title":     "Test Key",
		"publicKey": "ssh-rsa AAAA...",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/ssh-keys", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestSSHKeyHandler_CreateSSHKey_InvalidBody(t *testing.T) {
	app := fiber.New()
	handler := NewSSHKeyHandler(nil)

	app.Use(func(c *fiber.Ctx) error {
		user := &model.Account{UserName: "testuser"}
		c.Locals(contextutil.FiberUserKey, user)
		return c.Next()
	})
	app.Post("/ssh-keys", handler.CreateSSHKey)

	req := httptest.NewRequest("POST", "/ssh-keys", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestSSHKeyHandler_DeleteSSHKey_Unauthorized(t *testing.T) {
	app := fiber.New()
	handler := NewSSHKeyHandler(nil)

	app.Delete("/ssh-keys/:id", handler.DeleteSSHKey)

	req := httptest.NewRequest("DELETE", "/ssh-keys/1", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestSSHKeyHandler_DeleteSSHKey_InvalidID(t *testing.T) {
	app := fiber.New()
	handler := NewSSHKeyHandler(nil)

	app.Use(func(c *fiber.Ctx) error {
		user := &model.Account{UserName: "testuser"}
		c.Locals(contextutil.FiberUserKey, user)
		return c.Next()
	})
	app.Delete("/ssh-keys/:id", handler.DeleteSSHKey)

	req := httptest.NewRequest("DELETE", "/ssh-keys/invalid", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}
