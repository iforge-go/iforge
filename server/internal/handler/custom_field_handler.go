package handler

import (
	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/event"
	"iforge/iforge/internal/model"
	"iforge/iforge/internal/service"
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type CustomFieldHandler struct {
	customFieldService *service.CustomFieldService
	repoService        *service.RepositoryService
	eventBus           *event.Bus
}

func NewCustomFieldHandler(customFieldService *service.CustomFieldService, repoService *service.RepositoryService, eventBus *event.Bus) *CustomFieldHandler {
	return &CustomFieldHandler{
		customFieldService: customFieldService,
		repoService:        repoService,
		eventBus:           eventBus,
	}
}

func (h *CustomFieldHandler) ListCustomFields(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	repository, err := h.repoService.GetRepository(owner, repo)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	user := contextutil.GetUserFromContext(c)
	if repository.IsPrivate {
		if user == nil || !h.repoService.HasViewerRole(repository, user) {
			respondError(c, http.StatusNotFound, "Repository not found")
			return nil
		}
	}

	fields, err := h.customFieldService.ListCustomFields(owner, repo)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fields)
	return nil
}

func (h *CustomFieldHandler) GetCustomField(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	fieldID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid field ID")
		return nil
	}

	repository, err := h.repoService.GetRepository(owner, repo)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	user := contextutil.GetUserFromContext(c)
	if repository.IsPrivate {
		if user == nil || !h.repoService.HasViewerRole(repository, user) {
			respondError(c, http.StatusNotFound, "Repository not found")
			return nil
		}
	}

	field, err := h.customFieldService.GetCustomField(owner, repo, fieldID)
	if err != nil {
		respondError(c, http.StatusNotFound, "Custom field not found")
		return nil
	}

	c.Status(http.StatusOK).JSON(field)
	return nil
}

func (h *CustomFieldHandler) CreateCustomField(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	user, repository, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	var req struct {
		FieldName              string  `json:"fieldName" validate:"required"`
		FieldType              string  `json:"fieldType" validate:"required"`
		Constraints            *string `json:"constraints"`
		EnableForIssues        bool    `json:"enableForIssues"`
		EnableForMergeRequests bool    `json:"enableForMergeRequests"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	field := &model.CustomField{
		UserName:               owner,
		RepositoryName:         repo,
		FieldName:              req.FieldName,
		FieldType:              req.FieldType,
		Constraints:            req.Constraints,
		EnableForIssues:        req.EnableForIssues,
		EnableForMergeRequests: req.EnableForMergeRequests,
	}

	createdField, err := h.customFieldService.CreateCustomField(field)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	h.eventBus.Publish(event.NewCustomFieldCreatedEvent(repository, user, req.FieldName))

	c.Status(http.StatusCreated).JSON(createdField)
	return nil
}

func (h *CustomFieldHandler) UpdateCustomField(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	fieldID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid field ID")
		return nil
	}

	// Check write permission
	_, _, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	var req struct {
		FieldName              string  `json:"fieldName"`
		FieldType              string  `json:"fieldType"`
		Constraints            *string `json:"constraints"`
		EnableForIssues        *bool   `json:"enableForIssues"`
		EnableForMergeRequests *bool   `json:"enableForMergeRequests"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	updates := map[string]interface{}{}
	if req.FieldName != "" {
		updates["fieldName"] = req.FieldName
	}
	if req.FieldType != "" {
		updates["fieldType"] = req.FieldType
	}
	if req.Constraints != nil {
		updates["constraints"] = *req.Constraints
	}
	if req.EnableForIssues != nil {
		updates["enableForIssues"] = *req.EnableForIssues
	}
	if req.EnableForMergeRequests != nil {
		updates["enableForMergeRequests"] = *req.EnableForMergeRequests
	}

	err = h.customFieldService.UpdateCustomField(owner, repo, fieldID, updates)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Custom field updated successfully"})
	return nil
}

// DeleteCustomField deletes a custom field
func (h *CustomFieldHandler) DeleteCustomField(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	fieldID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid field ID")
		return nil
	}

	// Check admin permission
	_, _, ok := getUserAndCheckPermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	err = h.customFieldService.DeleteCustomField(owner, repo, fieldID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Custom field deleted successfully"})
	return nil
}

// GetIssueCustomFieldValues gets all custom field values for an issue
func (h *CustomFieldHandler) GetIssueCustomFieldValues(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	issueID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid issue ID")
		return nil
	}

	// Check repository access
	repository, err := h.repoService.GetRepository(owner, repo)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	user := contextutil.GetUserFromContext(c)
	if repository.IsPrivate {
		if user == nil || !h.repoService.HasViewerRole(repository, user) {
			respondError(c, http.StatusNotFound, "Repository not found")
			return nil
		}
	}

	values, err := h.customFieldService.GetIssueCustomFieldValues(owner, repo, issueID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(values)
	return nil
}

// SetIssueCustomFieldValue sets a custom field value for an issue
func (h *CustomFieldHandler) SetIssueCustomFieldValue(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	issueID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid issue ID")
		return nil
	}

	fieldID, err := strconv.Atoi(c.Params("fieldId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid field ID")
		return nil
	}

	// Check write permission
	_, _, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	var req struct {
		Value string `json:"value" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	err = h.customFieldService.SetIssueCustomFieldValue(owner, repo, issueID, fieldID, req.Value)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Custom field value set successfully"})
	return nil
}
