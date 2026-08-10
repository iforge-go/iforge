package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/model"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type LFSHandler struct {
	lfsService          *service.LFSService
	repoService         *service.RepositoryService
	accountService      *service.AccountService
	collaboratorService *service.CollaboratorService
}

func NewLFSHandler(
	lfsService *service.LFSService,
	repoService *service.RepositoryService,
	accountService *service.AccountService,
	collaboratorService *service.CollaboratorService,
) *LFSHandler {
	return &LFSHandler{
		lfsService:          lfsService,
		repoService:         repoService,
		accountService:      accountService,
		collaboratorService: collaboratorService,
	}
}

type lfsBatchRequest struct {
	Operation string      `json:"operation"`
	Transfers []string    `json:"transfers"`
	Objects   []lfsObject `json:"objects"`
}

type lfsObject struct {
	OID  string `json:"oid"`
	Size int64  `json:"size"`
}

type lfsBatchResponse struct {
	Transfer string              `json:"transfer"`
	Objects  []lfsResponseObject `json:"objects"`
}

type lfsResponseObject struct {
	OID           string               `json:"oid"`
	Size          int64                `json:"size"`
	Authenticated bool                 `json:"authenticated"`
	Actions       map[string]lfsAction `json:"actions,omitempty"`
	Error         *lfsError            `json:"error,omitempty"`
}

type lfsAction struct {
	Href   string            `json:"href"`
	Header map[string]string `json:"header,omitempty"`
}

type lfsError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (h *LFSHandler) ServeHTTP(c *fiber.Ctx) error {
	urlPath := c.Path()
	path := strings.TrimPrefix(urlPath, "/")
	parts := strings.Split(path, "/")

	// Strip leading "git/" prefix if present
	if len(parts) >= 1 && parts[0] == "git" {
		parts = parts[1:]
	}
	if len(parts) < 2 {
		respondError(c, http.StatusNotFound, "Invalid path")
		return nil
	}

	owner := parts[0]
	repoName := strings.TrimSuffix(parts[1], ".git")

	// remaining path after owner/repo
	remaining := ""
	if len(parts) > 2 {
		remaining = strings.Join(parts[2:], "/")
	}

	switch {
	case remaining == "info/lfs/batch" && c.Method() == http.MethodPost:
		return h.Batch(c, owner, repoName)
	case strings.HasPrefix(remaining, "info/lfs/objects/") && c.Method() == http.MethodPut:
		oid := strings.TrimPrefix(remaining, "info/lfs/objects/")
		return h.UploadObject(c, owner, repoName, oid)
	case strings.HasPrefix(remaining, "info/lfs/objects/") && c.Method() == http.MethodGet:
		oid := strings.TrimPrefix(remaining, "info/lfs/objects/")
		return h.DownloadObject(c, owner, repoName, oid)
	case remaining == "info/lfs/verify" && c.Method() == http.MethodPost:
		c.Status(http.StatusOK).JSON(fiber.Map{"message": "ok"})
		return nil
	}

	respondError(c, http.StatusNotFound, "Not found")
	return nil
}

// authenticate handles LFS HTTP Basic Auth
func (h *LFSHandler) authenticate(c *fiber.Ctx, owner string, repo *model.Repository, requireWrite bool) (*model.Account, bool) {
	auth := c.Get("Authorization")
	if auth == "" || !strings.HasPrefix(auth, "Basic ") {
		c.Set("WWW-Authenticate", `Basic realm="iForge LFS"`)
		respondError(c, http.StatusUnauthorized, "Authentication required")
		return nil, false
	}

	decoded, err := base64.StdEncoding.DecodeString(auth[6:])
	if err != nil {
		respondError(c, http.StatusUnauthorized, "Invalid authentication")
		return nil, false
	}

	credParts := strings.SplitN(string(decoded), ":", 2)
	if len(credParts) != 2 {
		respondError(c, http.StatusUnauthorized, "Invalid authentication")
		return nil, false
	}

	username, password := credParts[0], credParts[1]
	user, err := h.accountService.Authenticate(username, password)
	if err != nil {
		c.Set("WWW-Authenticate", `Basic realm="iForge LFS"`)
		respondError(c, http.StatusUnauthorized, "Invalid username or password")
		return nil, false
	}

	if requireWrite {
		if !h.canPush(user.UserName, owner, repo) {
			respondError(c, http.StatusForbidden, "Push permission required")
			return nil, false
		}
	} else {
		if !h.canPull(user.UserName, owner, repo) {
			respondError(c, http.StatusForbidden, "Access denied")
			return nil, false
		}
	}
	return user, true
}

func (h *LFSHandler) canPush(username, owner string, repo *model.Repository) bool {
	if username == owner {
		return true
	}
	user, err := h.accountService.GetAccountByUsername(username)
	if err != nil {
		return false
	}
	if user.IsAdmin {
		return true
	}
	role, err := h.collaboratorService.GetCollaboratorRole(owner, repo.RepositoryName, username)
	if err != nil {
		return false
	}
	// Write access requires admin or member (was previously "ADMIN"/"WRITE"
	// string literal which mismatched the canonical DEVELOPER constant — bug fix).
	return model.HasRoleAtLeast(role, model.RoleMember)
}

func (h *LFSHandler) canPull(username, owner string, repo *model.Repository) bool {
	if !repo.IsPrivate {
		return true
	}
	if username == owner {
		return true
	}
	user, err := h.accountService.GetAccountByUsername(username)
	if err != nil {
		return false
	}
	if user.IsAdmin {
		return true
	}
	role, err := h.collaboratorService.GetCollaboratorRole(owner, repo.RepositoryName, username)
	if err != nil {
		return false
	}
	// Read access requires at least viewer (was previously "ADMIN"/"WRITE"/"READ"
	// string literal — bug fix, now uses canonical role constants).
	return model.HasRoleAtLeast(role, model.RoleViewer)
}

// baseURL constructs the absolute base URL (scheme + host) from the request
func (h *LFSHandler) baseURL(c *fiber.Ctx) string {
	scheme := c.Protocol()
	if host := c.Get("Host"); host != "" {
		return scheme + "://" + host
	}
	return scheme + "://" + c.Hostname()
}

// Batch handles POST /info/lfs/batch
func (h *LFSHandler) Batch(c *fiber.Ctx, owner, repoName string) error {
	repo, err := h.repoService.GetRepository(owner, repoName)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	// Authenticate; download requires read, upload requires write
	requireWrite := false
	var req lfsBatchRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body")
		return nil
	}
	if req.Operation == "upload" {
		requireWrite = true
	}
	if _, ok := h.authenticate(c, owner, repo, requireWrite); !ok {
		return nil
	}

	base := h.baseURL(c)
	basePath := fmt.Sprintf("%s/%s/%s.git/info/lfs", base, owner, repoName)

	resp := lfsBatchResponse{Transfer: "basic", Objects: []lfsResponseObject{}}
	for _, obj := range req.Objects {
		respObj := lfsResponseObject{OID: obj.OID, Size: obj.Size, Authenticated: true}
		actions := map[string]lfsAction{}

		switch req.Operation {
		case "download":
			if !h.lfsService.MetaExists(owner, repoName, obj.OID) {
				respObj.Error = &lfsError{Code: 404, Message: "Object not found"}
				break
			}
			actions["download"] = lfsAction{
				Href: fmt.Sprintf("%s/objects/%s", basePath, obj.OID),
			}
		case "upload":
			if h.lfsService.MetaExists(owner, repoName, obj.OID) {
				// already exists; no actions needed
				break
			}
			actions["upload"] = lfsAction{
				Href: fmt.Sprintf("%s/objects/%s", basePath, obj.OID),
			}
			actions["verify"] = lfsAction{
				Href: fmt.Sprintf("%s/verify", basePath),
			}
		}

		if len(actions) > 0 {
			respObj.Actions = actions
		}
		resp.Objects = append(resp.Objects, respObj)
	}

	c.Set("Content-Type", "application/vnd.git-lfs+json")
	return c.Status(http.StatusOK).JSON(resp)
}

// UploadObject handles PUT /info/lfs/objects/:oid
func (h *LFSHandler) UploadObject(c *fiber.Ctx, owner, repoName, oid string) error {
	repo, err := h.repoService.GetRepository(owner, repoName)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}
	if _, ok := h.authenticate(c, owner, repo, true); !ok {
		return nil
	}

	// Stream body to disk
	body := bytes.NewReader(c.Body())
	if err := h.lfsService.WriteObjectContent(owner, repoName, oid, body); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to store object")
		return nil
	}

	// Save metadata (size from Content-Length)
	size := int64(len(c.Body()))
	if cl := c.Get("Content-Length"); cl != "" {
		var clInt int64
		if _, err := fmt.Sscanf(cl, "%d", &clInt); err == nil && clInt > 0 {
			size = clInt
		}
	}
	if err := h.lfsService.SaveMeta(owner, repoName, oid, size); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to save metadata")
		return nil
	}

	c.Status(http.StatusOK).SendString("")
	return nil
}

// DownloadObject handles GET /info/lfs/objects/:oid
func (h *LFSHandler) DownloadObject(c *fiber.Ctx, owner, repoName, oid string) error {
	repo, err := h.repoService.GetRepository(owner, repoName)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}
	if _, ok := h.authenticate(c, owner, repo, false); !ok {
		return nil
	}

	if !h.lfsService.MetaExists(owner, repoName, oid) {
		respondError(c, http.StatusNotFound, "Object not found")
		return nil
	}

	reader, size, err := h.lfsService.ReadObjectContent(owner, repoName, oid)
	if err != nil {
		respondError(c, http.StatusNotFound, "Object not found on disk")
		return nil
	}
	defer reader.Close()

	c.Set("Content-Type", "application/octet-stream")
	c.Set("Content-Length", fmt.Sprintf("%d", size))
	_, _ = io.Copy(c.Response().BodyWriter(), reader)
	return nil
}

// @Router /api/v1/repos/{owner}/{repo}/lfs/objects [get]
func (h *LFSHandler) ListLFSObjects(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	repo, err := h.repoService.GetRepository(owner, repoName)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}
	user := contextutil.GetUserFromContext(c)
	if repo.IsPrivate {
		if user == nil || !h.repoService.HasViewerRole(repo, user) {
			respondError(c, http.StatusNotFound, "Repository not found")
			return nil
		}
	}

	page, limit := parseLFSPageParams(c)
	objects, total, err := h.lfsService.ListObjects(owner, repoName, page, limit)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list LFS objects")
		return nil
	}
	if objects == nil {
		objects = []*model.LFSObject{}
	}
	c.Status(http.StatusOK).JSON(fiber.Map{"objects": objects, "total": total})
	return nil
}

// @Router /api/v1/repos/{owner}/{repo}/lfs/objects/{oid} [delete]
func (h *LFSHandler) DeleteLFSObject(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")
	oid := c.Params("oid")

	if _, _, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repoName); !ok {
		return nil
	}

	if !h.lfsService.MetaExists(owner, repoName, oid) {
		respondError(c, http.StatusNotFound, "LFS object not found")
		return nil
	}
	if err := h.lfsService.DeleteObject(owner, repoName, oid); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to delete LFS object")
		return nil
	}
	c.Status(http.StatusOK).JSON(fiber.Map{"message": "LFS object deleted"})
	return nil
}

// parseLFSPageParams extracts page/limit query params with sensible defaults
func parseLFSPageParams(c *fiber.Ctx) (int, int) {
	page := 1
	limit := 50
	if p := c.Query("page"); p != "" {
		var v int
		if _, err := fmt.Sscanf(p, "%d", &v); err == nil && v > 0 {
			page = v
		}
	}
	if l := c.Query("limit"); l != "" {
		var v int
		if _, err := fmt.Sscanf(l, "%d", &v); err == nil && v > 0 && v <= 200 {
			limit = v
		}
	}
	return page, limit
}
