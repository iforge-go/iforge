package handler

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type ReleaseHandler struct {
	releaseService *service.ReleaseService
	repoService    *service.RepositoryService
	uploadDir      string
}

func NewReleaseHandler(releaseService *service.ReleaseService, repoService *service.RepositoryService, uploadDir string) *ReleaseHandler {
	dir := filepath.Join(uploadDir, "_releases")
	os.MkdirAll(dir, 0755)
	return &ReleaseHandler{
		releaseService: releaseService,
		repoService:    repoService,
		uploadDir:      dir,
	}
}

func (h *ReleaseHandler) ListReleases(c *fiber.Ctx) error {
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

	releases, err := h.releaseService.ListReleases(owner, repo)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(releases)
	return nil
}

func (h *ReleaseHandler) GetRelease(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	tag := c.Params("tag")
	if decodedTag, err := url.QueryUnescape(tag); err == nil {
		tag = decodedTag
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

	release, err := h.releaseService.GetRelease(owner, repo, tag)
	if err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(release)
	return nil
}

func (h *ReleaseHandler) CreateRelease(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	user, _, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	var req struct {
		Tag     string  `json:"tag" validate:"required"`
		Name    string  `json:"name" validate:"required"`
		Content *string `json:"content"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	content := ""
	if req.Content != nil {
		content = *req.Content
	}

	release, err := h.releaseService.CreateRelease(owner, repo, req.Tag, req.Name, content, user.UserName)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusCreated).JSON(release)
	return nil
}

func (h *ReleaseHandler) UpdateRelease(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	tag := c.Params("tag")
	if decodedTag, err := url.QueryUnescape(tag); err == nil {
		tag = decodedTag
	}

	_, _, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	var req struct {
		Name    string  `json:"name" validate:"required"`
		Content *string `json:"content"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	content := ""
	if req.Content != nil {
		content = *req.Content
	}

	if err := h.releaseService.UpdateRelease(owner, repo, tag, req.Name, content); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	release, err := h.releaseService.GetRelease(owner, repo, tag)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(release)
	return nil
}

func (h *ReleaseHandler) DeleteRelease(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	tag := c.Params("tag")
	if decodedTag, err := url.QueryUnescape(tag); err == nil {
		tag = decodedTag
	}

	_, _, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	if err := h.releaseService.DeleteRelease(owner, repo, tag); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Release deleted successfully"})
	return nil
}

func (h *ReleaseHandler) UploadAsset(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	tag := c.Params("tag")

	user, _, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	// Verify release exists
	_, err := h.releaseService.GetRelease(owner, repo, tag)
	if err != nil {
		respondError(c, http.StatusNotFound, "Release not found")
		return nil
	}

	// Get uploaded file using Fiber's API
	fileHeader, err := c.FormFile("file")
	if err != nil {
		respondError(c, http.StatusBadRequest, "Failed to get uploaded file")
		return nil
	}

	// Open the uploaded file
	file, err := fileHeader.Open()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to open uploaded file")
		return nil
	}
	defer file.Close()

	// Create asset directory
	assetDir := filepath.Join(h.uploadDir, owner, repo, tag)
	if err := os.MkdirAll(assetDir, 0755); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to create asset directory")
		return nil
	}

	// Sanitize filename to prevent path traversal
	safeFilename := filepath.Base(fileHeader.Filename)
	if safeFilename == "." || safeFilename == "/" {
		respondError(c, http.StatusBadRequest, "Invalid filename")
		return nil
	}

	// Save file
	filePath := filepath.Join(assetDir, safeFilename)
	dst, err := os.Create(filePath)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to create file")
		return nil
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to save file")
		return nil
	}

	// Get optional label using Fiber's API
	label := c.FormValue("label")
	var labelPtr *string
	if label != "" {
		labelPtr = &label
	}

	// Add asset to database
	asset, err := h.releaseService.AddReleaseAsset(owner, repo, tag, fileHeader.Filename, labelPtr, fileHeader.Size, user.UserName)
	if err != nil {
		os.Remove(filePath) // Clean up file on error
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusCreated).JSON(asset)
	return nil
}

func (h *ReleaseHandler) DownloadAsset(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	tag := c.Params("tag")
	assetIDStr := c.Params("assetId")

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

	assetID, err := strconv.Atoi(assetIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid asset ID")
		return nil
	}

	// Get asset info using new method
	asset, err := h.releaseService.GetAsset(owner, repo, tag, assetID)
	if err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return nil
	}

	// Serve file
	filePath := filepath.Join(h.uploadDir, owner, repo, tag, asset.FileName)
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", asset.FileName))
	c.Set("Content-Type", "application/octet-stream")
	c.SendFile(filePath)
	return nil
}

func (h *ReleaseHandler) ListAssets(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	tag := c.Params("tag")

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

	assets, err := h.releaseService.ListReleaseAssets(owner, repo, tag)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(assets)
	return nil
}

func (h *ReleaseHandler) DeleteAsset(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	tag := c.Params("tag")
	assetIDStr := c.Params("assetId")

	_, _, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	assetID, err := strconv.Atoi(assetIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid asset ID")
		return nil
	}

	// Get asset info to delete file
	asset, err := h.releaseService.GetAsset(owner, repo, tag, assetID)
	if err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return nil
	}

	// Delete from database
	if err := h.releaseService.DeleteReleaseAsset(owner, repo, tag, assetID); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	// Delete file
	filePath := filepath.Join(h.uploadDir, owner, repo, tag, asset.FileName)
	os.Remove(filePath)

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Asset deleted successfully"})
	return nil
}
