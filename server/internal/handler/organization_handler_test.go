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

func TestNewOrganizationHandler(t *testing.T) {
	db := setupTestDB(t)
	accountService := service.NewAccountService(db, nil)
	handler := NewOrganizationHandler(accountService)

	if handler == nil {
		t.Error("NewOrganizationHandler returned nil")
	}
}

func TestOrganizationHandler_ListOrganizations_Empty(t *testing.T) {
	db := setupTestDB(t)
	accountService := service.NewAccountService(db, nil)
	handler := NewOrganizationHandler(accountService)

	app := fiber.New()
	app.Get("/organizations", handler.ListOrganizations)

	req := httptest.NewRequest("GET", "/organizations", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var organizations []*model.Account
	if err := json.NewDecoder(resp.Body).Decode(&organizations); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if len(organizations) != 0 {
		t.Errorf("Expected 0 organizations, got %d", len(organizations))
	}
}

func TestOrganizationHandler_ListOrganizations_WithData(t *testing.T) {
	db := setupTestDB(t)
	accountService := service.NewAccountService(db, nil)

	// Create organization
	_, err := accountService.CreateOrganization("admin", "testorg", "Test Organization")
	if err != nil {
		t.Fatalf("Failed to create organization: %v", err)
	}

	handler := NewOrganizationHandler(accountService)

	app := fiber.New()
	app.Get("/organizations", handler.ListOrganizations)

	req := httptest.NewRequest("GET", "/organizations", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var organizations []*model.Account
	if err := json.NewDecoder(resp.Body).Decode(&organizations); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if len(organizations) != 1 {
		t.Errorf("Expected 1 organization, got %d", len(organizations))
	}

	if organizations[0].UserName != "testorg" {
		t.Errorf("Expected organization name 'testorg', got '%s'", organizations[0].UserName)
	}
}

func TestOrganizationHandler_ListMyOrganizations_Unauthorized(t *testing.T) {
	db := setupTestDB(t)
	accountService := service.NewAccountService(db, nil)
	handler := NewOrganizationHandler(accountService)

	app := fiber.New()
	app.Get("/user/organizations", handler.ListMyOrganizations)

	req := httptest.NewRequest("GET", "/user/organizations", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestOrganizationHandler_ListManagedOrganizations_Unauthorized(t *testing.T) {
	db := setupTestDB(t)
	accountService := service.NewAccountService(db, nil)
	handler := NewOrganizationHandler(accountService)

	app := fiber.New()
	app.Get("/user/managed-organizations", handler.ListManagedOrganizations)

	req := httptest.NewRequest("GET", "/user/managed-organizations", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestOrganizationHandler_CreateOrganization_Unauthorized(t *testing.T) {
	db := setupTestDB(t)
	accountService := service.NewAccountService(db, nil)
	handler := NewOrganizationHandler(accountService)

	app := fiber.New()
	app.Post("/organizations", handler.CreateOrganization)

	req := httptest.NewRequest("POST", "/organizations", nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}
