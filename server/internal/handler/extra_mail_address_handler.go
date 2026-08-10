package handler

import (
	"iforge/iforge/internal/service"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

type ExtraMailAddressHandler struct {
	mailService *service.ExtraMailAddressService
}

func NewExtraMailAddressHandler(mailService *service.ExtraMailAddressService) *ExtraMailAddressHandler {
	return &ExtraMailAddressHandler{
		mailService: mailService,
	}
}

func (h *ExtraMailAddressHandler) ListExtraMailAddresses(c *fiber.Ctx) error {
	user, ok := getCurrentUser(c)
	if !ok {
		return nil
	}

	addresses, err := h.mailService.ListExtraMailAddresses(user.UserName)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list mail addresses")
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"addresses": addresses})
	return nil
}

func (h *ExtraMailAddressHandler) AddExtraMailAddress(c *fiber.Ctx) error {
	user, ok := getCurrentUser(c)
	if !ok {
		return nil
	}

	var req struct {
		MailAddress string `json:"mailAddress" validate:"required,email"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid mail address")
		return nil
	}

	if err := h.mailService.AddExtraMailAddress(user.UserName, req.MailAddress); err != nil {
		if err == service.ErrMailAddressExists {
			respondError(c, http.StatusConflict, "mail address already exists")
			return nil
		}
		respondError(c, http.StatusInternalServerError, "failed to add mail address")
		return nil
	}

	c.Status(http.StatusCreated).JSON(fiber.Map{"message": "mail address added successfully"})
	return nil
}

func (h *ExtraMailAddressHandler) DeleteExtraMailAddress(c *fiber.Ctx) error {
	user, ok := getCurrentUser(c)
	if !ok {
		return nil
	}

	mailAddress := c.Params("mailAddress")
	if mailAddress == "" {
		respondError(c, http.StatusBadRequest, "mail address is required")
		return nil
	}

	if err := h.mailService.DeleteExtraMailAddress(user.UserName, mailAddress); err != nil {
		if err == service.ErrMailAddressNotFound {
			respondError(c, http.StatusNotFound, "mail address not found")
			return nil
		}
		respondError(c, http.StatusInternalServerError, "failed to delete mail address")
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "mail address deleted successfully"})
	return nil
}

func (h *ExtraMailAddressHandler) GetPrimaryMailAddress(c *fiber.Ctx) error {
	user, ok := getCurrentUser(c)
	if !ok {
		return nil
	}

	mailAddress, err := h.mailService.GetPrimaryMailAddress(user.UserName)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to get primary mail address")
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"mailAddress": mailAddress})
	return nil
}
