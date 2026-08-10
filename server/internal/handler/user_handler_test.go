package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"iforge/iforge/internal/model"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

func TestNewUserHandler(t *testing.T) {
	db := setupTestDB(t)
	accountService := service.NewAccountService(db, nil)
	handler := NewUserHandler(accountService)

	if handler == nil {
		t.Error("NewUserHandler returned nil")
	}
}

func TestUserHandler_GetUser_Success(t *testing.T) {
	db := setupTestDB(t)
	accountService := service.NewAccountService(db, nil)

	// Create test user
	_, err := accountService.CreateAccount("testuser", "password123", "Test User", "test@example.com", false, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create test account: %v", err)
	}

	handler := NewUserHandler(accountService)

	app := fiber.New()
	app.Get("/users/:username", handler.GetUser)

	req := httptest.NewRequest("GET", "/users/testuser", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var user model.Account
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if user.UserName != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", user.UserName)
	}
}

func TestUserHandler_GetUser_NotFound(t *testing.T) {
	db := setupTestDB(t)
	accountService := service.NewAccountService(db, nil)
	handler := NewUserHandler(accountService)

	app := fiber.New()
	app.Get("/users/:username", handler.GetUser)

	req := httptest.NewRequest("GET", "/users/nonexistent", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestUserHandler_SearchUsers_Success(t *testing.T) {
	db := setupTestDB(t)
	accountService := service.NewAccountService(db, nil)

	// Create test users
	_, _ = accountService.CreateAccount("alice", "password123", "Alice Smith", "alice@example.com", false, nil, nil)
	_, _ = accountService.CreateAccount("bob", "password123", "Bob Johnson", "bob@example.com", false, nil, nil)

	handler := NewUserHandler(accountService)

	app := fiber.New()
	app.Get("/users/search", handler.SearchUsers)

	req := httptest.NewRequest("GET", "/users/search?q=alice", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string][]*model.Account
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if len(result["users"]) != 1 {
		t.Errorf("Expected 1 user, got %d", len(result["users"]))
	}
}

func TestUserHandler_SearchUsers_MissingQuery(t *testing.T) {
	db := setupTestDB(t)
	accountService := service.NewAccountService(db, nil)
	handler := NewUserHandler(accountService)

	app := fiber.New()
	app.Get("/users/search", handler.SearchUsers)

	req := httptest.NewRequest("GET", "/users/search", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestUserHandler_SearchUsers_EmptyQuery(t *testing.T) {
	db := setupTestDB(t)
	accountService := service.NewAccountService(db, nil)
	handler := NewUserHandler(accountService)

	app := fiber.New()
	app.Get("/users/search", handler.SearchUsers)

	req := httptest.NewRequest("GET", "/users/search?q=", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}
