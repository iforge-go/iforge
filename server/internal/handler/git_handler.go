package handler

import (
	"context"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"iforge/iforge/internal/event"
	"iforge/iforge/internal/git"
	"iforge/iforge/internal/model"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type GitHandler struct {
	gitClient   *git.Client
	repoService *service.RepositoryService
	eventBus    *event.Bus
}

func NewGitHandler(gitClient *git.Client, repoService *service.RepositoryService, eventBus *event.Bus) *GitHandler {
	return &GitHandler{
		gitClient:   gitClient,
		repoService: repoService,
		eventBus:    eventBus,
	}
}

func (h *GitHandler) ListFiles(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	_, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	ref := c.Query("ref", "HEAD")
	path := c.Query("path", "")

	files, err := h.gitClient.ListFiles(owner, repo, ref, path)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(files)
	return nil
}

// ListAllFiles recursively lists all files in the repository tree.
// Used by the "Go to file" feature.
func (h *GitHandler) ListAllFiles(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	_, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	ref := c.Query("ref", "HEAD")

	files, err := h.gitClient.ListAllFiles(owner, repo, ref)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(files)
	return nil
}

func (h *GitHandler) GetFileContent(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	_, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	ref := c.Query("ref", "HEAD")
	path := c.Query("path")

	if path == "" {
		respondError(c, http.StatusBadRequest, "path is required")
		return nil
	}

	content, err := h.gitClient.GetFileContent(owner, repo, ref, path)
	if err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(content)
	return nil
}

// DownloadRawFile downloads raw file content
func (h *GitHandler) DownloadRawFile(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	_, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	ref := c.Params("ref")
	path := c.Params("*")

	if path == "" {
		respondError(c, http.StatusBadRequest, "path is required")
		return nil
	}

	path = strings.TrimPrefix(path, "/")
	content, err := h.gitClient.GetRawFileContent(owner, repo, ref, path)
	if err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return nil
	}

	parts := strings.Split(path, "/")
	filename := parts[len(parts)-1]

	contentType := mime.TypeByExtension(filepath.Ext(filename))
	if contentType == "" {
		ext := strings.ToLower(filepath.Ext(filename))
		switch ext {
		case ".md", ".markdown":
			contentType = "text/markdown; charset=utf-8"
		case ".txt", ".log", ".text":
			contentType = "text/plain; charset=utf-8"
		case ".yml", ".yaml":
			contentType = "text/yaml; charset=utf-8"
		case ".json":
			contentType = "application/json; charset=utf-8"
		case ".xml":
			contentType = "application/xml; charset=utf-8"
		case ".toml":
			contentType = "text/toml; charset=utf-8"
		case ".ini", ".cfg", ".conf":
			contentType = "text/plain; charset=utf-8"
		case ".sh", ".bash":
			contentType = "text/x-shellscript; charset=utf-8"
		case ".py":
			contentType = "text/x-python; charset=utf-8"
		case ".go":
			contentType = "text/x-go; charset=utf-8"
		case ".js", ".mjs":
			contentType = "text/javascript; charset=utf-8"
		case ".ts", ".tsx":
			contentType = "text/typescript; charset=utf-8"
		case ".jsx":
			contentType = "text/jsx; charset=utf-8"
		case ".css":
			contentType = "text/css; charset=utf-8"
		case ".html", ".htm":
			contentType = "text/html; charset=utf-8"
		case ".svg":
			contentType = "image/svg+xml"
		case ".csv":
			contentType = "text/csv; charset=utf-8"
		default:
			contentType = "application/octet-stream"
		}
	}

	c.Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filename))
	c.Set("Content-Type", contentType)
	c.Status(http.StatusOK).Send(content)
	return nil
}

// ListCommits lists commits
func (h *GitHandler) ListCommits(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	_, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	ref := c.Query("sha", c.Query("ref", "HEAD"))
	limit := 50

	commits, err := h.gitClient.ListCommits(owner, repo, ref, limit)
	if err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(commits)
	return nil
}

func (h *GitHandler) GetCommit(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	_, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	commitId := c.Params("commit")

	commit, err := h.gitClient.GetCommit(owner, repo, commitId)
	if err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(commit)
	return nil
}

func (h *GitHandler) ListProtectedBranches(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	_, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	protected, err := h.repoService.ListProtectedBranches(owner, repo)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(protected)
	return nil
}

func (h *GitHandler) ProtectBranch(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	_, _, ok := getUserAndCheckPermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	var req struct {
		BranchName string `json:"branchName" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	if err := h.repoService.ProtectBranch(owner, repo, req.BranchName); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Branch protected successfully"})
	return nil
}

func (h *GitHandler) UnprotectBranch(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	branchName := c.Params("branch")
	if decodedBranch, err := url.QueryUnescape(branchName); err == nil {
		branchName = decodedBranch
	}

	_, _, ok := getUserAndCheckPermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	if err := h.repoService.UnprotectBranch(owner, repo, branchName); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Branch unprotected successfully"})
	return nil
}

func (h *GitHandler) ListBranches(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	repository, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	branches, err := h.gitClient.ListBranches(owner, repo, repository.DefaultBranch)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(branches)
	return nil
}

func (h *GitHandler) CreateBranch(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	_, _, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	var req struct {
		BranchName string `json:"branchName" validate:"required"`
		From       string `json:"from" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	if err := h.gitClient.CreateBranch(owner, repo, req.BranchName, req.From); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusCreated).JSON(fiber.Map{"message": "Branch created successfully"})
	return nil
}

func (h *GitHandler) DeleteBranch(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	branchName := c.Params("branch")
	if decodedBranch, err := url.QueryUnescape(branchName); err == nil {
		branchName = decodedBranch
	}

	_, _, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	if err := h.gitClient.DeleteBranch(owner, repo, branchName); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Branch deleted successfully"})
	return nil
}

func (h *GitHandler) RenameBranch(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	oldName := c.Params("branch")
	if decodedBranch, err := url.QueryUnescape(oldName); err == nil {
		oldName = decodedBranch
	}

	_, _, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	var req struct {
		NewName string `json:"newName" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	if err := h.gitClient.RenameBranch(owner, repo, oldName, req.NewName); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Branch renamed successfully"})
	return nil
}

func (h *GitHandler) SearchCode(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	_, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	ref := c.Query("ref", "HEAD")
	query := c.Query("q")

	if query == "" {
		respondError(c, http.StatusBadRequest, "query parameter 'q' is required")
		return nil
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	resp, err := h.gitClient.SearchCode(ctx, owner, repo, ref, query)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(resp)
	return nil
}

func (h *GitHandler) CreateFile(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	user, _, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	var req struct {
		Branch  string `json:"branch" validate:"required"`
		Path    string `json:"path" validate:"required"`
		Content string `json:"content" validate:"required"`
		Message string `json:"message" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	if err := h.gitClient.CreateFile(owner, repo, req.Branch, req.Path, req.Content, req.Message, user.FullName, user.MailAddress); err != nil {
		git.GetLogger().Error("CreateFile failed", err, map[string]interface{}{
			"owner":  owner,
			"repo":   repo,
			"branch": req.Branch,
			"path":   req.Path,
			"author": user.FullName,
			"email":  user.MailAddress,
		})
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	publishPushEvent(h, owner, repo, user, req.Branch, req.Path, "added")

	c.Status(http.StatusCreated).JSON(fiber.Map{"message": "File created successfully"})
	return nil
}

func (h *GitHandler) UpdateFile(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	user, _, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	var req struct {
		Branch  string `json:"branch" validate:"required"`
		Path    string `json:"path" validate:"required"`
		Content string `json:"content" validate:"required"`
		Message string `json:"message" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	if err := h.gitClient.UpdateFile(owner, repo, req.Branch, req.Path, req.Content, req.Message, user.FullName, user.MailAddress); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	publishPushEvent(h, owner, repo, user, req.Branch, req.Path, "modified")

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "File updated successfully"})
	return nil
}

func (h *GitHandler) DeleteFile(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	user, _, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	var req struct {
		Branch  string `json:"branch" validate:"required"`
		Path    string `json:"path" validate:"required"`
		Message string `json:"message" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	if err := h.gitClient.DeleteFile(owner, repo, req.Branch, req.Path, req.Message, user.FullName, user.MailAddress); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	publishPushEvent(h, owner, repo, user, req.Branch, req.Path, "removed")

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "File deleted successfully"})
	return nil
}

func (h *GitHandler) ListTags(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	_, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	tags, err := h.gitClient.ListTags(owner, repo)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(tags)
	return nil
}

func (h *GitHandler) CreateTag(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	user, _, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	var req struct {
		TagName string `json:"tagName" validate:"required"`
		Target  string `json:"target"`
		Message string `json:"message"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	err := h.gitClient.CreateTag(owner, repo, req.TagName, req.Target, req.Message, user.UserName, user.MailAddress)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusCreated).JSON(fiber.Map{"message": "Tag created successfully"})
	return nil
}

func (h *GitHandler) DeleteTag(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	tagName := c.Params("tag")
	if decodedTag, err := url.QueryUnescape(tagName); err == nil {
		tagName = decodedTag
	}

	_, _, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	if err := h.gitClient.DeleteTag(owner, repo, tagName); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Tag deleted successfully"})
	return nil
}

func (h *GitHandler) DownloadPatch(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	_, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	commitID := c.Params("commit")

	patch, err := h.gitClient.GeneratePatch(owner, repo, commitID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	filename := fmt.Sprintf("%s.patch", commitID[:7])
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Status(http.StatusOK).Send(patch)
	return nil
}

func (h *GitHandler) UploadFile(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	if _, err := c.Request().MultipartForm(); err != nil {
		respondError(c, http.StatusBadRequest, "Failed to parse multipart form")
		return nil
	}

	branch := c.FormValue("branch")
	if branch == "" {
		branch = "main"
	}
	path := c.FormValue("path")
	message := c.FormValue("message")

	fileHeader, err := c.FormFile("file")
	if err != nil {
		respondError(c, http.StatusBadRequest, "Failed to get uploaded file")
		return nil
	}

	file, err := fileHeader.Open()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to open uploaded file")
		return nil
	}
	defer file.Close()

	content := make([]byte, fileHeader.Size)
	if _, err := file.Read(content); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to read file content")
		return nil
	}

	user, _, ok := getUserAndCheckWritePermission(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	safeFilename := filepath.Base(fileHeader.Filename)
	if safeFilename == "." || safeFilename == "/" {
		respondError(c, http.StatusBadRequest, "Invalid filename")
		return nil
	}

	filePath := path
	if filePath == "" {
		filePath = safeFilename
	} else {
		filePath = path + "/" + safeFilename
	}

	if message == "" {
		message = fmt.Sprintf("Upload %s", fileHeader.Filename)
	}

	if err := h.gitClient.CreateFile(owner, repo, branch, filePath, string(content), message, user.FullName, user.MailAddress); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusCreated).JSON(fiber.Map{
		"message":  "File uploaded successfully",
		"filename": fileHeader.Filename,
		"path":     filePath,
	})
	return nil
}

func (h *GitHandler) BlameFile(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	_, _, ok := getRepoAndCheckAccess(c, h.repoService, owner, repo)
	if !ok {
		return nil
	}

	ref := c.Query("ref", "HEAD")
	path := c.Query("path")

	if path == "" {
		respondError(c, http.StatusBadRequest, "path is required")
		return nil
	}

	blame, err := h.gitClient.BlameFile(owner, repo, ref, path)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(blame)
	return nil
}

// publishPushEvent publishes a push event to the event bus
func publishPushEvent(h *GitHandler, owner, repo string, user *model.Account, branch, path, action string) {
	if h.eventBus == nil {
		return
	}

	repository, err := h.repoService.GetRepository(owner, repo)
	if err != nil {
		return
	}

	// Create commit info for the event
	commitInfo := map[string]interface{}{
		"message": fmt.Sprintf("%s %s", action, path),
		"author":  user.FullName,
		"path":    path,
		"action":  action,
	}

	evt := event.NewPushEvent(repository, user, branch, []map[string]interface{}{commitInfo})
	h.eventBus.Publish(evt)
}
