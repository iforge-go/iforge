package handler

import (
	"net/http"

	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/model"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type OrganizationHandler struct {
	accountService *service.AccountService
}

func NewOrganizationHandler(accountService *service.AccountService) *OrganizationHandler {
	return &OrganizationHandler{
		accountService: accountService,
	}
}

func (h *OrganizationHandler) ListOrganizations(c *fiber.Ctx) error {
	organizations, err := h.accountService.ListOrganizations()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list organizations")
		return nil
	}

	if organizations == nil {
		organizations = []*model.Account{}
	}

	c.Status(http.StatusOK).JSON(organizations)
	return nil
}

func (h *OrganizationHandler) ListMyOrganizations(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	organizations, err := h.accountService.GetUserOrganizations(user.UserName)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list organizations")
		return nil
	}

	if organizations == nil {
		organizations = []*model.Account{}
	}

	c.Status(http.StatusOK).JSON(organizations)
	return nil
}

func (h *OrganizationHandler) ListManagedOrganizations(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	names, err := h.accountService.GetManagedOrganizationNames(user.UserName)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list managed organizations")
		return nil
	}

	if names == nil {
		names = []string{}
	}

	c.Status(http.StatusOK).JSON(names)
	return nil
}

func (h *OrganizationHandler) CreateOrganization(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	if !user.IsAdmin {
		respondError(c, http.StatusForbidden, "Forbidden: Admin access required to create organizations")
		return nil
	}

	var req struct {
		OrganizationName string `json:"organizationName" validate:"required"`
		Description      string `json:"description"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body")
		return nil
	}

	organization, err := h.accountService.CreateOrganization(user.UserName, req.OrganizationName, req.Description)
	if err != nil {
		if err == service.ErrReservedName {
			respondErrorWithMessageKey(c, http.StatusBadRequest, "errors.orgNameReserved", "This organization name is reserved and cannot be used")
			return nil
		}
		respondErrorWithMessageKey(c, http.StatusInternalServerError, "errors.orgCreateFailed", "Failed to create organization")
		return nil
	}

	LogAudit(c, "organization.create", "organization", req.OrganizationName, nil, true)

	c.Status(http.StatusCreated).JSON(organization)
	return nil
}

func (h *OrganizationHandler) GetOrganization(c *fiber.Ctx) error {
	organizationName := c.Params("organizationName")
	organization, err := h.accountService.GetAccountByUsername(organizationName)
	if err != nil {
		respondError(c, http.StatusNotFound, "Organization not found")
		return nil
	}

	if !organization.IsOrganization {
		respondError(c, http.StatusNotFound, "Organization not found")
		return nil
	}

	c.Status(http.StatusOK).JSON(organization)
	return nil
}

func (h *OrganizationHandler) GetOrganizationMembers(c *fiber.Ctx) error {
	organizationName := c.Params("organizationName")
	members, err := h.accountService.GetOrganizationMembers(organizationName)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to get organization members")
		return nil
	}

	c.Status(http.StatusOK).JSON(members)
	return nil
}

func (h *OrganizationHandler) AddOrganizationMember(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	organizationName := c.Params("organizationName")

	if !user.IsAdmin {
		isManager, err := h.accountService.IsOrganizationManager(organizationName, user.UserName)
		if err != nil || !isManager {
			respondError(c, http.StatusForbidden, "Forbidden: Organization manager access required")
			return nil
		}
	}

	var req struct {
		UserName  string `json:"userName" validate:"required"`
		IsManager bool   `json:"isManager"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body")
		return nil
	}

	err := h.accountService.AddOrganizationMember(organizationName, req.UserName, req.IsManager)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to add organization member")
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Member added successfully"})
	return nil
}

func (h *OrganizationHandler) RemoveOrganizationMember(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	organizationName := c.Params("organizationName")
	userName := c.Params("userName")

	if !user.IsAdmin {
		isManager, err := h.accountService.IsOrganizationManager(organizationName, user.UserName)
		if err != nil || !isManager {
			respondError(c, http.StatusForbidden, "Forbidden: Organization manager access required")
			return nil
		}
	}

	err := h.accountService.RemoveOrganizationMember(organizationName, userName)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to remove organization member")
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Member removed successfully"})
	return nil
}

func (h *OrganizationHandler) UpdateOrganization(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	organizationName := c.Params("organizationName")

	if !user.IsAdmin {
		isManager, err := h.accountService.IsOrganizationManager(organizationName, user.UserName)
		if err != nil || !isManager {
			respondError(c, http.StatusForbidden, "Forbidden: Organization manager access required")
			return nil
		}
	}

	var req struct {
		Description *string `json:"description"`
		URL         *string `json:"url"`
	}
	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	updates := make(map[string]interface{})
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.URL != nil {
		updates["url"] = *req.URL
	}

	if err := h.accountService.UpdateAccount(organizationName, updates); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	updated, err := h.accountService.GetAccountByUsername(organizationName)
	if err != nil {
		c.Status(http.StatusOK).JSON(fiber.Map{"message": "Organization updated"})
		return nil
	}
	updated.Password = ""

	LogAudit(c, "organization.update", "organization", organizationName, nil, true)

	c.Status(http.StatusOK).JSON(updated)
	return nil
}

func (h *OrganizationHandler) DeleteOrganization(c *fiber.Ctx) error {
	user := contextutil.GetUserFromContext(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "Unauthorized")
		return nil
	}

	if !user.IsAdmin {
		respondError(c, http.StatusForbidden, "Forbidden: Admin access required to delete organizations")
		return nil
	}

	organizationName := c.Params("organizationName")

	account, err := h.accountService.GetAccountByUsername(organizationName)
	if err != nil {
		respondError(c, http.StatusNotFound, "Organization not found")
		return nil
	}
	if !account.IsOrganization {
		respondError(c, http.StatusBadRequest, "Not an organization account")
		return nil
	}

	if err := h.accountService.DeleteAccount(organizationName); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	LogAudit(c, "organization.delete", "organization", organizationName, nil, true)

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Organization deleted: " + organizationName})
	return nil
}
