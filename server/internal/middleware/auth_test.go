package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/model"

	"github.com/gofiber/fiber/v2"
)

func TestRequireAuth_Unauthorized(t *testing.T) {
	app := fiber.New()
	app.Use(RequireAuth())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestRequireAuth_Authorized(t *testing.T) {
	app := fiber.New()

	// Set user in context
	app.Use(func(c *fiber.Ctx) error {
		user := &model.Account{UserName: "testuser", IsAdmin: false}
		c.Locals(contextutil.FiberUserKey, user)
		return c.Next()
	})
	app.Use(RequireAuth())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestRequireAdmin_Unauthorized(t *testing.T) {
	app := fiber.New()
	app.Use(RequireAdmin())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", resp.StatusCode)
	}
}

func TestRequireAdmin_NonAdmin(t *testing.T) {
	app := fiber.New()

	// Set non-admin user in context
	app.Use(func(c *fiber.Ctx) error {
		user := &model.Account{UserName: "testuser", IsAdmin: false}
		c.Locals(contextutil.FiberUserKey, user)
		return c.Next()
	})
	app.Use(RequireAdmin())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", resp.StatusCode)
	}
}

func TestRequireAdmin_Admin(t *testing.T) {
	app := fiber.New()

	// Set admin user in context
	app.Use(func(c *fiber.Ctx) error {
		user := &model.Account{UserName: "admin", IsAdmin: true}
		c.Locals(contextutil.FiberUserKey, user)
		return c.Next()
	})
	app.Use(RequireAdmin())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestCORSMiddleware_DefaultOrigins(t *testing.T) {
	app := fiber.New()
	app.Use(CORSMiddleware())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	if allowOrigin != "http://localhost:3000" {
		t.Errorf("Expected Access-Control-Allow-Origin to be 'http://localhost:3000', got '%s'", allowOrigin)
	}
}

func TestCORSMiddleware_DisallowedOrigin(t *testing.T) {
	app := fiber.New()
	app.Use(CORSMiddleware())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://evil.com")
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	if allowOrigin != "" {
		t.Errorf("Expected Access-Control-Allow-Origin to be empty, got '%s'", allowOrigin)
	}
}

func TestCORSMiddleware_OptionsRequest(t *testing.T) {
	app := fiber.New()
	app.Use(CORSMiddleware())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	allowMethods := resp.Header.Get("Access-Control-Allow-Methods")
	if allowMethods == "" {
		t.Error("Expected Access-Control-Allow-Methods to be set")
	}
}

func TestGetAllowedOrigins_Defaults(t *testing.T) {
	origins := getAllowedOrigins()

	if !origins["http://localhost:3000"] {
		t.Error("Expected http://localhost:3000 to be allowed")
	}
	if !origins["http://localhost:3001"] {
		t.Error("Expected http://localhost:3001 to be allowed")
	}
	if !origins["http://localhost:8081"] {
		t.Error("Expected http://localhost:8081 to be allowed")
	}
}

func TestGetAllowedOrigins_WildcardEnv(t *testing.T) {
	t.Setenv("IFORGE_CORS_ORIGINS", "*")
	origins := getAllowedOrigins()

	if !origins["*"] {
		t.Error("Expected wildcard origin to be set")
	}
}

func TestGetAllowedOrigins_CustomEnv(t *testing.T) {
	t.Setenv("IFORGE_CORS_ORIGINS", "http://example.com,http://test.com")
	origins := getAllowedOrigins()

	if !origins["http://example.com"] {
		t.Error("Expected http://example.com to be allowed")
	}
	if !origins["http://test.com"] {
		t.Error("Expected http://test.com to be allowed")
	}
}

func TestIsAllowedOrigin_Wildcard(t *testing.T) {
	allowed := map[string]bool{"*": true}

	if !isAllowedOrigin("http://any.com", allowed) {
		t.Error("Expected any origin to be allowed with wildcard")
	}
}

func TestIsAllowedOrigin_Specific(t *testing.T) {
	allowed := map[string]bool{"http://example.com": true}

	if !isAllowedOrigin("http://example.com", allowed) {
		t.Error("Expected http://example.com to be allowed")
	}
	if isAllowedOrigin("http://evil.com", allowed) {
		t.Error("Expected http://evil.com to be disallowed")
	}
}
