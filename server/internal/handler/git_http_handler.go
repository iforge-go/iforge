package handler

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"iforge/iforge/internal/event"
	"iforge/iforge/internal/git"
	"iforge/iforge/internal/model"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type GitHTTPHandler struct {
	gitClient           *git.Client
	repoService         *service.RepositoryService
	authService         *service.AccountService
	activityService     *service.ActivityService
	collaboratorService *service.CollaboratorService
	accessTokenService  *service.AccessTokenService
	eventBus            *event.Bus
	logger              *git.Logger
}

func NewGitHTTPHandler(gitClient *git.Client, repoService *service.RepositoryService, authService *service.AccountService, activityService *service.ActivityService, collaboratorService *service.CollaboratorService, accessTokenService *service.AccessTokenService, eventBus *event.Bus) *GitHTTPHandler {
	return &GitHTTPHandler{
		gitClient:           gitClient,
		repoService:         repoService,
		authService:         authService,
		activityService:     activityService,
		collaboratorService: collaboratorService,
		accessTokenService:  accessTokenService,
		eventBus:            eventBus,
		logger:              git.GetLogger(),
	}
}

func (h *GitHTTPHandler) ServeHTTP(c *fiber.Ctx) error {
	urlPath := c.Path()
	path := strings.TrimPrefix(urlPath, "/")
	parts := strings.Split(path, "/")

	// Handle /git/ prefix (legacy route)
	if len(parts) >= 1 && parts[0] == "git" {
		parts = parts[1:]
	}

	if len(parts) < 2 {
		respondError(c, http.StatusNotFound, "Invalid path")
		return nil
	}

	owner := parts[0]
	repoName := strings.TrimSuffix(parts[1], ".git")
	remainingPath := ""
	if len(parts) > 2 {
		remainingPath = strings.Join(parts[2:], "/")
	}

	// Path validation: consistent with SSH parseRepoPath/isValidName to prevent path traversal
	if !isValidGitName(owner) || !isValidGitName(repoName) {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	repo, err := h.repoService.GetRepository(owner, repoName)
	if err != nil {
		h.logger.Warn("Repository not found in DB", map[string]interface{}{"owner": owner, "repo": repoName, "error": err.Error()})
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	repoPath := h.gitClient.RepositoryPath(owner, repoName)

	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		respondError(c, http.StatusNotFound, "Repository not found on disk")
		return nil
	}

	if remainingPath == "info/refs" {
		svc := c.Query("service")
		if svc == "git-upload-pack" || svc == "git-receive-pack" {
			if svc == "git-receive-pack" {
				if _, ok := h.authenticate(c, owner, repo, true); !ok {
					return nil
				}
			} else if repo.IsPrivate {
				if _, ok := h.authenticate(c, owner, repo, false); !ok {
					return nil
				}
			}
			h.getInfoRefs(c, repoPath, svc)
			return nil
		}
		h.serveGitFile(c, repoPath, remainingPath, owner, repo)
		return nil
	}

	if remainingPath == "git-upload-pack" {
		if repo.IsPrivate {
			if _, ok := h.authenticate(c, owner, repo, false); !ok {
				return nil
			}
		}
		h.uploadPack(c, repoPath)
		return nil
	}

	if remainingPath == "git-receive-pack" {
		user, ok := h.authenticate(c, owner, repo, true)
		if !ok {
			h.logger.Warn("Git HTTP authentication failed")
			return nil
		}
		h.receivePack(c, repoPath, owner, repoName, user.UserName)
		return nil
	}

	if remainingPath != "" {
		h.serveGitFile(c, repoPath, remainingPath, owner, repo)
		return nil
	}

	respondError(c, http.StatusNotFound, "Not found")
	return nil
}

func (h *GitHTTPHandler) authenticate(c *fiber.Ctx, owner string, repo *model.Repository, requireWrite bool) (*model.Account, bool) {
	auth := c.Get("Authorization")
	if auth == "" {
		c.Set("WWW-Authenticate", `Basic realm="iForge"`)
		respondError(c, http.StatusUnauthorized, "Authentication required")
		return nil, false
	}

	if !strings.HasPrefix(auth, "Basic ") {
		respondError(c, http.StatusUnauthorized, "Invalid authentication")
		return nil, false
	}

	// Decode Basic Auth credentials
	decoded, err := base64.StdEncoding.DecodeString(auth[6:])
	if err != nil {
		respondError(c, http.StatusUnauthorized, "Invalid authentication")
		return nil, false
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		respondError(c, http.StatusUnauthorized, "Invalid authentication")
		return nil, false
	}

	username, password := parts[0], parts[1]

	var user *model.Account
	if h.accessTokenService != nil {
		user, err = h.accessTokenService.ValidateAccessToken(password)
	}
	if user == nil {
		user, err = h.authService.Authenticate(username, password)
	}
	if err != nil {
		c.Set("WWW-Authenticate", `Basic realm="iForge"`)
		respondError(c, http.StatusUnauthorized, "Invalid username or password")
		return nil, false
	}

	if requireWrite {
		if !h.canPush(user.UserName, owner, repo) {
			respondError(c, http.StatusForbidden, "You do not have permission to push to this repository")
			return nil, false
		}
	} else {
		if !h.canPull(user.UserName, owner, repo) {
			respondError(c, http.StatusForbidden, "You do not have permission to access this repository")
			return nil, false
		}
	}

	return user, true
}

// Requires Developer role or higher
func (h *GitHTTPHandler) canPush(username, owner string, repo *model.Repository) bool {
	user, err := h.authService.GetAccountByUsername(username)
	if err != nil {
		return false
	}
	return h.repoService.HasMemberRole(repo, user)
}

// Requires Guest role or higher for private repos, anyone for public repos
func (h *GitHTTPHandler) canPull(username, owner string, repo *model.Repository) bool {
	// Public repositories can be pulled by anyone
	if !repo.IsPrivate {
		return true
	}

	user, err := h.authService.GetAccountByUsername(username)
	if err != nil {
		return false
	}
	return h.repoService.HasViewerRole(repo, user)
}

func (h *GitHTTPHandler) getInfoRefs(c *fiber.Ctx, repoPath, serviceName string) {
	cmd := exec.Command(serviceName, "--stateless-rpc", "--advertise-refs", repoPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		h.logger.Error("Git command failed", err, map[string]interface{}{"stderr": stderr.String()})
		respondError(c, http.StatusInternalServerError, "Git command failed")
		return
	}

	c.Status(http.StatusOK)
	contentType := fmt.Sprintf("application/x-%s-advertisement", serviceName)
	c.Set("Content-Type", contentType)

	serviceLine := fmt.Sprintf("# service=%s\n", serviceName)
	pktLen := len(serviceLine) + 4
	fmt.Fprintf(c.Response().BodyWriter(), "%04x%s", pktLen, serviceLine)

	c.Response().BodyWriter().Write([]byte("0000"))
	c.Response().BodyWriter().Write(stdout.Bytes())
}

func (h *GitHTTPHandler) uploadPack(c *fiber.Ctx, repoPath string) {
	c.Set("Content-Type", "application/x-git-upload-pack-result")
	h.runGitService(c, repoPath, "upload-pack")
}

func (h *GitHTTPHandler) receivePack(c *fiber.Ctx, repoPath, owner, repoName, pusherName string) {
	refUpdates := parseRefUpdatesFromBody(c.Body())

	for _, update := range refUpdates {
		if update.IsDeleted {
			continue
		}
		branch := strings.TrimPrefix(update.RefName, "refs/heads/")
		if branch == update.RefName {
			continue
		}
		canPush, err := h.repoService.CanPushToBranch(owner, repoName, branch, pusherName)
		if err != nil {
			h.logger.Warn("Failed to check branch protection", map[string]interface{}{"branch": branch, "error": err.Error()})
			continue
		}
		if !canPush {
			respondError(c, http.StatusForbidden, fmt.Sprintf("Push to protected branch '%s' is not allowed", branch))
			return
		}
	}

	c.Set("Content-Type", "application/x-git-receive-pack-result")
	h.runGitService(c, repoPath, "receive-pack")

	parts := strings.Split(repoPath, string(filepath.Separator))
	if len(parts) >= 2 {
		cacheOwner := parts[len(parts)-2]
		cacheRepo := strings.TrimSuffix(parts[len(parts)-1], ".git")
		git.InvalidateRepoCaches(cacheOwner, cacheRepo)
	}

	var additionalInfo *string
	if h.gitClient != nil {
		for i, update := range refUpdates {
			if update.IsDeleted {
				refUpdates[i].Commits = []git.PushCommitInfo{}
				continue
			}
			commits, err := h.gitClient.ListCommitsBetween(owner, repoName, update.OldSHA, update.NewSHA, 50)
			if err == nil {
				refUpdates[i].Commits = commits
			}
		}
		if len(refUpdates) > 0 {
			pushDetail := git.PushDetail{RefUpdates: refUpdates}
			if jsonBytes, err := json.Marshal(pushDetail); err == nil {
				jsonStr := string(jsonBytes)
				additionalInfo = &jsonStr
			}
		}
	}

	if h.activityService != nil {
		h.activityService.CreateActivity(
			owner, repoName, pusherName, "push",
			repoName, additionalInfo,
		)
	}

	if h.eventBus != nil {
		repoFullName := owner + "/" + repoName
		repoModel := &model.Repository{UserName: owner, RepositoryName: repoName}
		sender := &model.Account{UserName: pusherName}
		for _, update := range refUpdates {
			if update.IsDeleted {
				continue
			}
			if strings.HasPrefix(update.RefName, "refs/heads/") {
				h.eventBus.Publish(event.NewBranchPushedEvent(
					repoModel, sender,
					repoFullName, update.BranchName, pusherName,
					update.Commits, update.IsNewBranch,
				))
				continue
			}
			if strings.HasPrefix(update.RefName, "refs/tags/") {
				h.eventBus.Publish(event.NewTagPushedEvent(
					repoModel, sender,
					repoFullName, update.BranchName, pusherName,
					update.Commits, update.IsNewBranch,
				))
			}
		}
	}
}

// Parse git pack protocol pkt-lines from receive-pack stdin
func parseRefUpdatesFromBody(body []byte) []git.RefUpdate {
	var updates []git.RefUpdate
	offset := 0
	for offset+4 <= len(body) {
		lenHex := string(body[offset : offset+4])
		pktLen, err := strconv.ParseInt(lenHex, 16, 32)
		if err != nil || pktLen == 0 {
			break
		}
		if offset+int(pktLen) > len(body) {
			break
		}
		payload := string(body[offset+4 : offset+int(pktLen)])
		offset += int(pktLen)

		if idx := strings.IndexByte(payload, '\x00'); idx >= 0 {
			payload = payload[:idx]
		}
		payload = strings.TrimSpace(payload)

		parts := strings.Fields(payload)
		if len(parts) < 3 {
			continue
		}
		oldSHA, newSHA, refName := parts[0], parts[1], parts[2]
		branchName := strings.TrimPrefix(refName, "refs/heads/")
		branchName = strings.TrimPrefix(branchName, "refs/tags/")

		updates = append(updates, git.RefUpdate{
			RefName:     refName,
			BranchName:  branchName,
			OldSHA:      oldSHA,
			NewSHA:      newSHA,
			IsNewBranch: oldSHA == "0000000000000000000000000000000000000000",
			IsDeleted:   newSHA == "0000000000000000000000000000000000000000",
		})
	}
	return updates
}

func (h *GitHTTPHandler) runGitService(c *fiber.Ctx, repoPath, serviceName string) {
	rawBody := c.Body()
	var body io.Reader = bytes.NewReader(rawBody)

	if c.Get("Content-Encoding") == "gzip" && len(rawBody) >= 2 && rawBody[0] == 0x1f && rawBody[1] == 0x8b {
		gzReader, err := gzip.NewReader(bytes.NewReader(rawBody))
		if err != nil {
			h.logger.Warn("Failed to decode gzip request body", map[string]interface{}{"error": err.Error()})
			respondError(c, http.StatusBadRequest, "Invalid gzip")
			return
		}
		defer gzReader.Close()
		body = gzReader
	}

	c.Set("Content-Type", "application/x-git-"+serviceName+"-result")
	c.Status(http.StatusOK)

	cmd := exec.Command("git", serviceName, "--stateless-rpc", repoPath)
	cmd.Stdin = body
	cmd.Stdout = c.Response().BodyWriter()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		h.logger.Error("Git service command failed", err, map[string]interface{}{"stderr": stderr.String()})
		return
	}
}

func (h *GitHTTPHandler) serveGitFile(c *fiber.Ctx, repoPath, requestPath string, owner string, repo *model.Repository) {
	if repo.IsPrivate {
		if _, ok := h.authenticate(c, owner, repo, false); !ok {
			return
		}
	}

	cleanPath := filepath.Clean(requestPath)
	if strings.Contains(cleanPath, "..") {
		respondError(c, http.StatusForbidden, "Access denied")
		return
	}

	filePath := filepath.Join(repoPath, cleanPath)

	if !strings.HasPrefix(filepath.Clean(filePath), filepath.Clean(repoPath)) {
		respondError(c, http.StatusForbidden, "Access denied")
		return
	}

	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		respondError(c, http.StatusNotFound, "File not found")
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to stat file")
		return
	}

	if info.IsDir() {
		respondError(c, http.StatusNotFound, "File not found")
		return
	}

	ext := filepath.Ext(filePath)
	switch ext {
	case ".pack":
		c.Set("Content-Type", "application/x-git-packed-objects")
	case ".idx":
		c.Set("Content-Type", "application/x-git-packed-objects-toc")
	default:
		c.Set("Content-Type", "application/octet-stream")
	}

	c.SendFile(filePath)
}

// isValidGitName validates owner/repoName contains only safe characters.
// Consistent with SSH server's isValidName to prevent path traversal attacks.
func isValidGitName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_' || r == '-' || r == '.':
		default:
			return false
		}
	}
	return true
}
