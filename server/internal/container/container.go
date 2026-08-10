package container

import (
	"iforge/iforge/internal/event"
	"iforge/iforge/internal/git"
	"iforge/iforge/internal/handler"
	"iforge/iforge/internal/realtime"
	"iforge/iforge/internal/service"

	"gorm.io/gorm"
)

// Container holds all application dependencies
type Container struct {
	// Configuration
	reposPath   string
	homeDir     string
	httpPort    int
	sshPort     int
	sshEnabled  bool
	externalURL string
	dbDriver    string

	// Database
	db *gorm.DB

	// Event bus
	eventBus *event.Bus

	// Realtime
	hub              *realtime.Hub
	webSocketHandler *handler.WebSocketHandler

	// Core services
	accountService        *service.AccountService
	repoService           *service.RepositoryService
	issueService          *service.IssueService
	mergeRequestService   *service.MergeRequestService
	activityService       *service.ActivityService
	webhookService        *service.WebhookService
	labelService          *service.LabelService
	milestoneService      *service.MilestoneService
	collaboratorService   *service.CollaboratorService
	releaseService        *service.ReleaseService
	starService           *service.StarService
	watchService          *service.WatchService
	wikiService           *service.WikiService
	sshKeyService         *service.SSHKeyService
	accessTokenService    *service.AccessTokenService
	notificationService   *service.NotificationService
	commitStatusService   *service.CommitStatusService
	deployKeyService      *service.DeployKeyService
	reviewService         *service.ReviewService
	codeQualityService    *service.CodeQualityService
	mirrorService         *service.MirrorService
	databaseViewerService *service.DatabaseViewerService
	gpgKeyService         *service.GPGKeyService
	customFieldService    *service.CustomFieldService
	searchService         *service.SearchService
	priorityService       *service.PriorityService
	extraMailService      *service.ExtraMailAddressService
	preferenceService     *service.AccountPreferenceService
	passwordResetService  *service.PasswordResetService
	pluginManager         *service.PluginManager
	systemSettingsService *service.SystemSettingsService
	ldapService           *service.LDAPService
	oidcService           *service.OIDCService
	mailService           *service.MailService
	importService         *service.ImportService
	aiService             *service.AIService
	aiModelConfigService  *service.AIModelConfigService
	gitClient             *git.Client
	statsService          *service.StatsService
	contributorService    *service.ContributorService
	setupService          *service.SetupService
	lfsService            *service.LFSService

	// Project management services
	projectService       *service.ProjectService
	sprintService        *service.SprintService
	userStoryService     *service.UserStoryService
	taskService          *service.TaskService
	taskStatusService    *service.TaskStatusService
	scrumActivityService *service.ScrumActivityService
	epicService          *service.EpicService

	// Security & compliance services
	auditService *service.AuditService

	// CI/CD services (built-in Pipeline/Job/Runner/Deployment/Cron full pipeline)
	pipelineService   *service.PipelineService
	runnerService     *service.RunnerService
	secretService     *service.SecretService
	deploymentService *service.DeploymentService
	cronService       *service.CronService

	// Plugin services
	templateEngine *service.TemplateEngine
	eventEngine    *service.PluginEventEngine

	// Handlers (lazily initialized)
	authHandler           *handler.AuthHandler
	organizationHandler   *handler.OrganizationHandler
	userHandler           *handler.UserHandler
	repoHandler           *handler.RepositoryHandler
	issueHandler          *handler.IssueHandler
	mergeRequestHandler   *handler.MergeRequestHandler
	activityHandler       *handler.ActivityHandler
	webhookHandler        *handler.WebhookHandler
	labelHandler          *handler.LabelHandler
	milestoneHandler      *handler.MilestoneHandler
	collaboratorHandler   *handler.CollaboratorHandler
	releaseHandler        *handler.ReleaseHandler
	starHandler           *handler.StarHandler
	watchHandler          *handler.WatchHandler
	wikiHandler           *handler.WikiHandler
	sshKeyHandler         *handler.SSHKeyHandler
	accessTokenHandler    *handler.AccessTokenHandler
	notificationHandler   *handler.NotificationHandler
	commitStatusHandler   *handler.CommitStatusHandler
	deployKeyHandler      *handler.DeployKeyHandler
	reviewHandler         *handler.ReviewHandler
	codeQualityHandler    *handler.CodeQualityHandler
	mirrorHandler         *handler.MirrorHandler
	contributorHandler    *handler.ContributorHandler
	statsHandler          *handler.StatsHandler
	gitHandler            *handler.GitHandler
	adminHandler          *handler.AdminHandler
	databaseViewerHandler *handler.DatabaseViewerHandler
	oidcHandler           *handler.OIDCHandler
	gpgKeyHandler         *handler.GPGKeyHandler
	customFieldHandler    *handler.CustomFieldHandler
	searchHandler         *handler.SearchHandler
	atomHandler           *handler.AtomHandler
	importHandler         *handler.ImportHandler
	priorityHandler       *handler.PriorityHandler
	extraMailHandler      *handler.ExtraMailAddressHandler
	preferenceHandler     *handler.AccountPreferenceHandler
	archiveHandler        *handler.ArchiveHandler
	passwordResetHandler  *handler.PasswordResetHandler
	systemSettingsHandler *handler.SystemSettingsHandler
	aiModelConfigHandler  *handler.AIModelConfigHandler
	pluginHandler         *handler.PluginHandler
	monitorHandler        *handler.MonitorHandler
	setupHandler          *handler.SetupHandler
	lfsHandler            *handler.LFSHandler

	// Project management handlers
	projectHandler       *handler.ProjectHandler
	sprintHandler        *handler.SprintHandler
	userStoryHandler     *handler.UserStoryHandler
	taskHandler          *handler.TaskHandler
	taskStatusHandler    *handler.TaskStatusHandler
	scrumActivityHandler *handler.ScrumActivityHandler
	epicHandler          *handler.EpicHandler

	// Security & compliance handlers
	auditHandler *handler.AuditHandler

	// CI/CD handlers
	cicdHandler       *handler.CICDHandler
	cicdRunnerHandler *handler.CICDRunnerHandler
}

// NewContainer creates a new dependency injection container
func NewContainer(reposPath, homeDir string, httpPort, sshPort int, sshEnabled bool) *Container {
	return &Container{
		reposPath:  reposPath,
		homeDir:    homeDir,
		httpPort:   httpPort,
		sshPort:    sshPort,
		sshEnabled: sshEnabled,
	}
}

// ReposPath returns the repositories path
func (c *Container) ReposPath() string {
	return c.reposPath
}

// HomeDir returns the home directory
func (c *Container) HomeDir() string {
	return c.homeDir
}

// HTTPPort returns the HTTP server port
func (c *Container) HTTPPort() int {
	return c.httpPort
}

// SSHPort returns the SSH server port
func (c *Container) SSHPort() int {
	return c.sshPort
}

// SSHEnabled returns whether SSH server is enabled
func (c *Container) SSHEnabled() bool {
	return c.sshEnabled
}

// ExternalURL returns the external URL for reverse proxy
func (c *Container) ExternalURL() string {
	return c.externalURL
}

// SetExternalURL sets the external URL
func (c *Container) SetExternalURL(url string) {
	c.externalURL = url
}

// DB returns the database connection
func (c *Container) DB() *gorm.DB {
	return c.db
}

// SetDB sets the database connection (called after database.InitGORMDB)
func (c *Container) SetDB(db *gorm.DB) {
	c.db = db
}

// DBDriver returns the database driver name
func (c *Container) DBDriver() string {
	return c.dbDriver
}

// SetDBDriver sets the database driver name
func (c *Container) SetDBDriver(driver string) {
	c.dbDriver = driver
}
