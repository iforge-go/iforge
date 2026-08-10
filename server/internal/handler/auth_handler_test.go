package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"iforge/iforge/internal/model"
	"iforge/iforge/internal/service"
	"iforge/iforge/internal/session"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var testDBCounter int64

func init() {
	session.SetSecret("test-secret-key-for-handler-tests")
}

func setupTestDB(t *testing.T) *gorm.DB {
	n := atomic.AddInt64(&testDBCounter, 1)
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:testdb_handler_%d?mode=memory&cache=shared", n)), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Migrate necessary tables
	err = db.AutoMigrate(
		&model.Account{},
		&model.OrganizationMember{},
		&model.SSHKey{},
		&model.GPGKey{},
		&model.DeployKey{},
		&model.AccessToken{},
		&model.Collaborator{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}

func TestNewAuthHandler(t *testing.T) {
	db := setupTestDB(t)
	accountService := service.NewAccountService(db, nil)
	settingsService := service.NewSystemSettingsService(db)
	handler := NewAuthHandler(accountService, settingsService)

	if handler == nil {
		t.Error("NewAuthHandler returned nil")
	}
}

func TestAuthHandler_Login_InvalidBody(t *testing.T) {
	db := setupTestDB(t)
	accountService := service.NewAccountService(db, nil)
	settingsService := service.NewSystemSettingsService(db)
	handler := NewAuthHandler(accountService, settingsService)

	app := fiber.New()
	app.Post("/login", handler.Login)

	req := httptest.NewRequest("POST", "/login", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	db := setupTestDB(t)
	accountService := service.NewAccountService(db, nil)
	settingsService := service.NewSystemSettingsService(db)
	handler := NewAuthHandler(accountService, settingsService)

	app := fiber.New()
	app.Post("/login", handler.Login)

	loginReq := LoginRequest{
		Username: "nonexistent",
		Password: "wrongpassword",
	}
	body, _ := json.Marshal(loginReq)

	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestAuthHandler_Login_Success(t *testing.T) {
	db := setupTestDB(t)
	accountService := service.NewAccountService(db, nil)

	// Create test user using CreateAccount (which hashes the password)
	_, err := accountService.CreateAccount("testuser", "password123", "Test User", "test@example.com", false, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create test account: %v", err)
	}

	settingsService := service.NewSystemSettingsService(db)
	handler := NewAuthHandler(accountService, settingsService)

	app := fiber.New()
	app.Post("/login", handler.Login)

	loginReq := LoginRequest{
		Username: "testuser",
		Password: "password123",
	}
	body, _ := json.Marshal(loginReq)

	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if loginResp.Token == "" {
		t.Error("Expected token in response")
	}
}

func TestAuthHandler_Register_InvalidBody(t *testing.T) {
	db := setupTestDB(t)
	accountService := service.NewAccountService(db, nil)
	settingsService := service.NewSystemSettingsService(db)
	handler := NewAuthHandler(accountService, settingsService)

	app := fiber.New()
	app.Post("/register", handler.Register)

	req := httptest.NewRequest("POST", "/register", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestAuthHandler_Register_ReservedName(t *testing.T) {
	db := setupTestDB(t)
	accountService := service.NewAccountService(db, nil)
	settingsService := service.NewSystemSettingsService(db)
	handler := NewAuthHandler(accountService, settingsService)

	app := fiber.New()
	app.Post("/register", handler.Register)

	// Reserved name should return 400
	registerReq := RegisterRequest{
		Username:    "admin",
		Password:    "password123",
		FullName:    "Test User",
		MailAddress: "test@example.com",
	}
	body, _ := json.Marshal(registerReq)

	req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestAuthHandler_Register_Success(t *testing.T) {
	db := setupTestDB(t)
	accountService := service.NewAccountService(db, nil)
	settingsService := service.NewSystemSettingsService(db)
	handler := NewAuthHandler(accountService, settingsService)

	app := fiber.New()
	app.Post("/register", handler.Register)

	registerReq := RegisterRequest{
		Username:    "newuser",
		Password:    "password123",
		FullName:    "New User",
		MailAddress: "newuser@example.com",
	}
	body, _ := json.Marshal(registerReq)

	req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}
}

func TestAuthHandler_Register_DuplicateUser(t *testing.T) {
	db := setupTestDB(t)
	accountService := service.NewAccountService(db, nil)

	// Create existing user
	existingAccount := &model.Account{
		UserName:    "existinguser",
		Password:    "password123",
		MailAddress: "existing@example.com",
		FullName:    "Existing User",
		IsRemoved:   false,
	}
	if err := db.Create(existingAccount).Error; err != nil {
		t.Fatalf("Failed to create existing account: %v", err)
	}

	settingsService := service.NewSystemSettingsService(db)
	handler := NewAuthHandler(accountService, settingsService)

	app := fiber.New()
	app.Post("/register", handler.Register)

	registerReq := RegisterRequest{
		Username:    "existinguser",
		Password:    "password123",
		FullName:    "New User",
		MailAddress: "newuser@example.com",
	}
	body, _ := json.Marshal(registerReq)

	req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", resp.StatusCode)
	}
}

func TestAuthHandler_Logout(t *testing.T) {
	db := setupTestDB(t)
	accountService := service.NewAccountService(db, nil)
	settingsService := service.NewSystemSettingsService(db)
	handler := NewAuthHandler(accountService, settingsService)

	app := fiber.New()
	app.Post("/logout", handler.Logout)

	req := httptest.NewRequest("POST", "/logout", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}
