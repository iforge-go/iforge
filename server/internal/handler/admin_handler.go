package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

var startTime = time.Now()

type AdminHandler struct {
	accountService  *service.AccountService
	repoService     *service.RepositoryService
	projectService  *service.ProjectService
	settingsService *service.SystemSettingsService
	homeDir         string
	httpPort        int
	sshPort         int
	sshEnabled      bool
	dbDriver        string
}

func NewAdminHandler(
	accountService *service.AccountService,
	repoService *service.RepositoryService,
	projectService *service.ProjectService,
	settingsService *service.SystemSettingsService,
	homeDir string,
	httpPort, sshPort int,
	sshEnabled bool,
	dbDriver string,
) *AdminHandler {
	return &AdminHandler{
		accountService:  accountService,
		repoService:     repoService,
		projectService:  projectService,
		settingsService: settingsService,
		homeDir:         homeDir,
		httpPort:        httpPort,
		sshPort:         sshPort,
		sshEnabled:      sshEnabled,
		dbDriver:        dbDriver,
	}
}

func (h *AdminHandler) ListUsers(c *fiber.Ctx) error {
	users, err := h.accountService.GetAllUsers(false)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list users")
		return nil
	}

	for _, user := range users {
		user.Password = ""
	}

	c.Status(http.StatusOK).JSON(users)
	return nil
}

func (h *AdminHandler) CreateUser(c *fiber.Ctx) error {
	var req struct {
		UserName    string  `json:"userName" validate:"required"`
		Password    string  `json:"password" validate:"required"`
		FullName    string  `json:"fullName" validate:"required"`
		MailAddress string  `json:"mailAddress" validate:"required,email"`
		IsAdmin     bool    `json:"isAdmin"`
		URL         *string `json:"url"`
		Description *string `json:"description"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondErrorWithMessageKey(c, http.StatusBadRequest, "errors.invalidRequest", "Invalid request body")
		return nil
	}

	user, err := h.accountService.CreateAccount(req.UserName, req.Password, req.FullName, req.MailAddress, req.IsAdmin, req.URL, req.Description)
	if err != nil {
		if err == service.ErrUserAlreadyExists {
			respondErrorWithMessageKey(c, http.StatusConflict, "errors.userExists", "User already exists")
			return nil
		}
		if err == service.ErrReservedName {
			respondErrorWithMessageKey(c, http.StatusBadRequest, "errors.usernameReserved", "This username is reserved and cannot be used")
			return nil
		}
		respondErrorWithMessageKey(c, http.StatusInternalServerError, "errors.userCreateFailed", "Failed to create user")
		return nil
	}

	user.Password = ""
	c.Status(http.StatusCreated).JSON(user)
	return nil
}

func (h *AdminHandler) UpdateUser(c *fiber.Ctx) error {
	username := c.Params("username")

	var req struct {
		FullName    *string `json:"fullName"`
		MailAddress *string `json:"mailAddress"`
		IsAdmin     *bool   `json:"isAdmin"`
		URL         *string `json:"url"`
		Description *string `json:"description"`
		IsRemoved   *bool   `json:"isRemoved"`
		Password    *string `json:"password"`
	}

	if err := c.BodyParser(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return nil
	}

	if currentAccount := contextutil.GetUserFromContext(c); currentAccount != nil {
		if currentAccount.UserName == username {
			if req.IsAdmin != nil && !*req.IsAdmin {
				respondError(c, http.StatusBadRequest, "Cannot revoke your own admin privileges")
				return nil
			}
			if req.IsRemoved != nil && *req.IsRemoved {
				respondError(c, http.StatusBadRequest, "Cannot disable your own account")
				return nil
			}
		}
	}

	if req.IsAdmin != nil && !*req.IsAdmin {
		user, err := h.accountService.GetAccountByUsername(username)
		if err != nil {
			respondError(c, http.StatusNotFound, "User not found")
			return nil
		}
		if user.IsAdmin {
			allUsers, err := h.accountService.GetAllUsers(false)
			if err != nil {
				respondError(c, http.StatusInternalServerError, "Failed to check admin count")
				return nil
			}
			adminCount := 0
			for _, u := range allUsers {
				if u.IsAdmin {
					adminCount++
				}
			}
			if adminCount <= 1 {
				respondError(c, http.StatusBadRequest, "Cannot revoke the last admin's privileges")
				return nil
			}
		}
	}

	updates := make(map[string]interface{})
	if req.FullName != nil {
		updates["fullName"] = *req.FullName
	}
	if req.MailAddress != nil {
		updates["mailAddress"] = *req.MailAddress
	}
	if req.IsAdmin != nil {
		updates["isAdmin"] = *req.IsAdmin
	}
	if req.URL != nil {
		updates["url"] = *req.URL
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.IsRemoved != nil {
		updates["isRemoved"] = *req.IsRemoved
	}
	if req.Password != nil && *req.Password != "" {
		updates["password"] = *req.Password
	}

	if err := h.accountService.UpdateAccount(username, updates); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	updated, err := h.accountService.GetAccountByUsername(username)
	if err != nil {
		c.Status(http.StatusOK).JSON(fiber.Map{"message": "User updated"})
		return nil
	}
	updated.Password = ""
	c.Status(http.StatusOK).JSON(updated)
	return nil
}

func (h *AdminHandler) DeleteUser(c *fiber.Ctx) error {
	username := c.Params("username")

	if currentAccount := contextutil.GetUserFromContext(c); currentAccount != nil {
		if currentAccount.UserName == username {
			respondError(c, http.StatusBadRequest, "Cannot delete your own account")
			return nil
		}
	}

	account, err := h.accountService.GetAccountByUsername(username)
	if err != nil {
		respondError(c, http.StatusNotFound, "User not found")
		return nil
	}
	if account.IsOrganization {
		respondError(c, http.StatusBadRequest, "Cannot delete organization account via user API")
		return nil
	}

	if err := h.accountService.DeleteAccount(username); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "User deleted: " + username})
	return nil
}

// GetSystemInfo returns system information (admin only)
func (h *AdminHandler) GetSystemInfo(c *fiber.Ctx) error {
	users, err := h.accountService.GetAllUsers(false)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to get user count")
		return nil
	}

	organizations, err := h.accountService.ListOrganizations()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to get organization count")
		return nil
	}

	repoCount, err := h.repoService.CountRepositories()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to get repository count")
		return nil
	}

	projectCount, err := h.projectService.CountProjects()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to get project count")
		return nil
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	hostname, _ := os.Hostname()

	uptime := time.Since(startTime)

	configuredTimezone := "Local"
	if settings, err := h.settingsService.GetGeneralSettings(); err == nil && settings.Timezone != "" {
		if loc, err := time.LoadLocation(settings.Timezone); err == nil {
			configuredTimezone = settings.Timezone
			_ = loc
		}
	}

	var serverTime string
	var timezoneName string
	if configuredTimezone != "Local" {
		if loc, err := time.LoadLocation(configuredTimezone); err == nil {
			now := time.Now().In(loc)
			serverTime = now.Format("2006-01-02 15:04:05")
			timezoneName = configuredTimezone
		} else {
			serverTime = time.Now().Format("2006-01-02 15:04:05")
			timezoneName = getServerTimezone()
		}
	} else {
		serverTime = time.Now().Format("2006-01-02 15:04:05")
		timezoneName = getServerTimezone()
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"userCount":    len(users),
		"orgCount":     len(organizations),
		"repoCount":    repoCount,
		"projectCount": projectCount,
		"database": fiber.Map{
			"type": h.getDatabaseType(),
			"path": h.getDatabasePath(),
		},
		"uptime": fiber.Map{
			"seconds": int64(uptime.Seconds()),
			"human":   formatUptime(uptime),
		},
		"memory": fiber.Map{
			"alloc":      memStats.Alloc,
			"totalAlloc": memStats.TotalAlloc,
			"sys":        memStats.Sys,
			"numGC":      memStats.NumGC,
			"allocHuman": formatBytes(memStats.Alloc),
			"sysHuman":   formatBytes(memStats.Sys),
		},
		"cpu": fiber.Map{
			"numCPU":       runtime.NumCPU(),
			"numGoroutine": runtime.NumGoroutine(),
		},
		"os": fiber.Map{
			"name":      runtime.GOOS,
			"nameHuman": formatOS(runtime.GOOS),
			"arch":      runtime.GOARCH,
			"hostname":  hostname,
			"compiler":  runtime.Compiler,
		},
		"services": fiber.Map{
			"http": fiber.Map{
				"port":    h.httpPort,
				"address": fmt.Sprintf("http://localhost:%d", h.httpPort),
			},
			"ssh": fiber.Map{
				"enabled":  h.sshEnabled,
				"port":     h.sshPort,
				"address":  fmt.Sprintf("ssh://git@localhost:%d", h.sshPort),
				"protocol": "git-upload-pack / git-receive-pack",
			},
			"database": fiber.Map{
				"type": h.getDatabaseType(),
				"path": h.getDatabasePath(),
			},
			"reposPath": filepath.Join(h.homeDir, "repositories"),
			"homeDir":   h.homeDir,
		},
		"goVersion":  runtime.Version(),
		"version":    Version,
		"gitCommit":  GitCommit,
		"buildTime":  BuildTime,
		"dirty":      Dirty,
		"repository": "iforge",
		"serverTime": serverTime,
		"timezone":   timezoneName,
	})
}

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	suffix := []string{"KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%.2f %s", float64(b)/float64(div), suffix[exp])
}

func formatOS(goos string) string {
	switch goos {
	case "windows":
		return "Windows"
	case "linux":
		return "Linux"
	case "darwin":
		return "macOS"
	case "freebsd":
		return "FreeBSD"
	case "openbsd":
		return "OpenBSD"
	case "netbsd":
		return "NetBSD"
	default:
		return goos
	}
}

func formatUptime(d time.Duration) string {
	days := int(d / (24 * time.Hour))
	d -= time.Duration(days) * 24 * time.Hour
	hours := int(d / time.Hour)
	d -= time.Duration(hours) * time.Hour
	minutes := int(d / time.Minute)
	seconds := int(d / time.Second)
	if days > 0 {
		return fmt.Sprintf("%d days %d hours %d minutes", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%d hours %d minutes %d seconds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%d minutes %d seconds", minutes, seconds)
	}
	return fmt.Sprintf("%d seconds", seconds)
}

func getServerTimezone() string {
	now := time.Now()
	loc := now.Location()
	name := loc.String()
	if name != "Local" {
		return name
	}
	return "Local"
}

func (h *AdminHandler) getDatabaseType() string {
	switch h.dbDriver {
	case "mysql":
		return "MySQL"
	case "postgres":
		return "PostgreSQL"
	default:
		return "SQLite"
	}
}

func (h *AdminHandler) getDatabasePath() string {
	switch h.dbDriver {
	case "mysql":
		return "MySQL Server"
	case "postgres":
		return "PostgreSQL Server"
	default:
		return filepath.Join(h.homeDir, "iforge.db")
	}
}

func (h *AdminHandler) ListRepositories(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	repos, total, err := h.repoService.ListAllRepositories(page, limit)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list repositories")
		return nil
	}

	c.Status(http.StatusOK).JSON(fiber.Map{
		"repositories": repos,
		"total":        total,
		"page":         page,
		"limit":        limit,
	})
	return nil
}

func (h *AdminHandler) GetRepository(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	repo, err := h.repoService.GetRepository(owner, repoName)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	c.Status(http.StatusOK).JSON(repo)
	return nil
}

func (h *AdminHandler) DeleteRepository(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repoName := c.Params("repo")

	_, err := h.repoService.GetRepository(owner, repoName)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	if err := h.repoService.DeleteRepository(owner, repoName); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to delete repository")
		return nil
	}

	LogAudit(c, "repository.delete", "repository", owner+"/"+repoName,
		map[string]interface{}{"admin": true}, true)

	c.Status(http.StatusOK).JSON(fiber.Map{"message": "Repository deleted: " + owner + "/" + repoName})
	return nil
}
