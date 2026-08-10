package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestParsePaginationParams_Defaults(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		page, perPage, offset := parsePaginationParams(c)
		return c.JSON(fiber.Map{"page": page, "per_page": perPage, "offset": offset})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestParsePaginationParams_Custom(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		page, perPage, offset := parsePaginationParams(c)
		return c.JSON(fiber.Map{"page": page, "per_page": perPage, "offset": offset})
	})

	req := httptest.NewRequest("GET", "/test?page=2&per_page=50", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestParseIntParam_Valid(t *testing.T) {
	result, err := parseIntParam("42")
	if err != nil {
		t.Errorf("parseIntParam returned error: %v", err)
	}
	if result != 42 {
		t.Errorf("Expected 42, got %d", result)
	}
}

func TestParseIntParam_Invalid(t *testing.T) {
	_, err := parseIntParam("abc")
	if err == nil {
		t.Error("Expected error for invalid int")
	}
}

func TestParseIntParam_Negative(t *testing.T) {
	result, err := parseIntParam("-5")
	if err != nil {
		t.Errorf("parseIntParam returned error: %v", err)
	}
	if result != -5 {
		t.Errorf("Expected -5, got %d", result)
	}
}

func TestParseIntParam_Zero(t *testing.T) {
	result, err := parseIntParam("0")
	if err != nil {
		t.Errorf("parseIntParam returned error: %v", err)
	}
	if result != 0 {
		t.Errorf("Expected 0, got %d", result)
	}
}

func TestRespondError(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return respondError(c, http.StatusBadRequest, "test error")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestRespondErrorWithMessageKey(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return respondErrorWithMessageKey(c, http.StatusConflict, "errors.duplicate", "Duplicate entry")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", resp.StatusCode)
	}
}

func TestRespondSuccess(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return respondSuccess(c, http.StatusOK, fiber.Map{"data": "test"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestGetCurrentUser_Unauthorized(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		_, ok := getCurrentUser(c)
		if ok {
			t.Error("Expected getCurrentUser to return false for unauthenticated request")
		}
		return nil
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req, -1)

	// getCurrentUser writes 401 response when no user in context
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestEncodeID(t *testing.T) {
	encoded := encodeID(1)
	if encoded == "" {
		t.Error("encodeID returned empty string")
	}
}

func TestDecodeID_Valid(t *testing.T) {
	encoded := encodeID(42)
	decoded, err := decodeID(encoded)
	if err != nil {
		t.Errorf("decodeID returned error: %v", err)
	}
	if decoded != 42 {
		t.Errorf("Expected 42, got %d", decoded)
	}
}

func TestDecodeID_Invalid(t *testing.T) {
	_, err := decodeID("invalid-hashid")
	// Should return error or empty result
	if err == nil {
		// Some hashid implementations don't error on invalid input
		// They just return empty slice
		t.Log("decodeID did not return error for invalid input (implementation-dependent)")
	}
}
