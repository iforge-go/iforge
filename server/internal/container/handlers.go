package container

import (
	gitsvc "iforge/iforge/internal/git"
	"iforge/iforge/internal/handler"
	"iforge/iforge/internal/realtime"
	"iforge/iforge/internal/service"
)

// Handler accessor methods with lazy initialization

func (c *Container) AuthHandler() *handler.AuthHandler {
	if c.authHandler == nil {
		c.authHandler = handler.NewAuthHandler(c.AccountService(), c.SystemSettingsService())
	}
	return c.authHandler
}

func (c *Container) UserHandler() *handler.UserHandler {
	if c.userHandler == nil {
		c.userHandler = handler.NewUserHandler(c.AccountService())
	}
	return c.userHandler
}

// AuditService returns the audit log service, lazily initializing it.
func (c *Container) AuditService() *service.AuditService {
	if c.auditService == nil {
		c.auditService = service.NewAuditService(c.DB())
	}
	return c.auditService
}

// AuditHandler returns the audit log admin handler, lazily initializing it.
func (c *Container) AuditHandler() *handler.AuditHandler {
	if c.auditHandler == nil {
		c.auditHandler = handler.NewAuditHandler(c.AuditService())
	}
	return c.auditHandler
}

// CICDHandler returns the CI/CD handler, lazily initializing it.
// Depends on Pipeline/Runner/Secret/Deployment/Cron services + RepoService (permission check).
func (c *Container) CICDHandler() *handler.CICDHandler {
	if c.cicdHandler == nil {
		c.cicdHandler = handler.NewCICDHandler(
			c.PipelineService(),
			c.RunnerService(),
			c.SecretService(),
			c.DeploymentService(),
			c.RepoService(),
			c.CronService(),
		)
	}
	return c.cicdHandler
}

// CICDRunnerHandler returns the external runner API handler, lazily initializing it.
// Depends on RunnerService (claim/register) + SecretService (inject decrypted secrets) +
// PipelineService (refresh pipeline status).
func (c *Container) CICDRunnerHandler() *handler.CICDRunnerHandler {
	if c.cicdRunnerHandler == nil {
		c.cicdRunnerHandler = handler.NewCICDRunnerHandler(
			c.RunnerService(),
			c.SecretService(),
			c.PipelineService(),
			c.ExternalURL(),
		)
	}
	return c.cicdRunnerHandler
}

func (c *Container) OrganizationHandler() *handler.OrganizationHandler {
	if c.organizationHandler == nil {
		c.organizationHandler = handler.NewOrganizationHandler(c.AccountService())
	}
	return c.organizationHandler
}

func (c *Container) RepoHandler() *handler.RepositoryHandler {
	if c.repoHandler == nil {
		c.repoHandler = handler.NewRepositoryHandler(
			c.RepoService(),
			c.AccountService(),
			c.GitClient(),
			c.EventBus(),
		)
	}
	return c.repoHandler
}

func (c *Container) IssueHandler() *handler.IssueHandler {
	if c.issueHandler == nil {
		c.issueHandler = handler.NewIssueHandler(
			c.IssueService(),
			c.RepoService(),
			c.GitClient(),
			c.LabelService(),
			c.MilestoneService(),
			c.EventBus(),
			c.AccountService(),
		)
	}
	return c.issueHandler
}

func (c *Container) MergeRequestHandler() *handler.MergeRequestHandler {
	if c.mergeRequestHandler == nil {
		c.mergeRequestHandler = handler.NewMergeRequestHandler(
			c.MergeRequestService(),
			c.IssueService(),
			c.GitClient(),
			c.NotificationService(),
			c.RepoService(),
			c.EventBus(),
			c.AccountService(),
		)
	}
	return c.mergeRequestHandler
}

func (c *Container) ActivityHandler() *handler.ActivityHandler {
	if c.activityHandler == nil {
		c.activityHandler = handler.NewActivityHandler(
			c.ActivityService(),
			c.RepoService(),
			c.AccountService(),
		)
	}
	return c.activityHandler
}

func (c *Container) WebhookHandler() *handler.WebhookHandler {
	if c.webhookHandler == nil {
		c.webhookHandler = handler.NewWebhookHandler(
			c.WebhookService(),
			c.RepoService(),
		)
	}
	return c.webhookHandler
}

func (c *Container) LabelHandler() *handler.LabelHandler {
	if c.labelHandler == nil {
		c.labelHandler = handler.NewLabelHandler(
			c.LabelService(),
			c.RepoService(),
			c.IssueService(),
		)
	}
	return c.labelHandler
}

func (c *Container) MilestoneHandler() *handler.MilestoneHandler {
	if c.milestoneHandler == nil {
		c.milestoneHandler = handler.NewMilestoneHandler(
			c.MilestoneService(),
			c.RepoService(),
		)
	}
	return c.milestoneHandler
}

func (c *Container) CollaboratorHandler() *handler.CollaboratorHandler {
	if c.collaboratorHandler == nil {
		c.collaboratorHandler = handler.NewCollaboratorHandler(
			c.CollaboratorService(),
			c.RepoService(),
			c.EventBus(),
		)
	}
	return c.collaboratorHandler
}

func (c *Container) ReleaseHandler() *handler.ReleaseHandler {
	if c.releaseHandler == nil {
		c.releaseHandler = handler.NewReleaseHandler(
			c.ReleaseService(),
			c.RepoService(),
			c.ReposPath(),
		)
	}
	return c.releaseHandler
}

func (c *Container) StarHandler() *handler.StarHandler {
	if c.starHandler == nil {
		c.starHandler = handler.NewStarHandler(c.StarService(), c.RepoService())
	}
	return c.starHandler
}

func (c *Container) WatchHandler() *handler.WatchHandler {
	if c.watchHandler == nil {
		c.watchHandler = handler.NewWatchHandler(c.WatchService(), c.RepoService())
	}
	return c.watchHandler
}

func (c *Container) WikiHandler() *handler.WikiHandler {
	if c.wikiHandler == nil {
		c.wikiHandler = handler.NewWikiHandler(c.WikiService(), c.RepoService())
	}
	return c.wikiHandler
}

func (c *Container) SSHKeyHandler() *handler.SSHKeyHandler {
	if c.sshKeyHandler == nil {
		c.sshKeyHandler = handler.NewSSHKeyHandler(c.SSHKeyService())
	}
	return c.sshKeyHandler
}

func (c *Container) AccessTokenHandler() *handler.AccessTokenHandler {
	if c.accessTokenHandler == nil {
		c.accessTokenHandler = handler.NewAccessTokenHandler(c.AccessTokenService())
	}
	return c.accessTokenHandler
}

func (c *Container) NotificationHandler() *handler.NotificationHandler {
	if c.notificationHandler == nil {
		c.notificationHandler = handler.NewNotificationHandler(c.NotificationService(), c.AccountService())
	}
	return c.notificationHandler
}

func (c *Container) CommitStatusHandler() *handler.CommitStatusHandler {
	if c.commitStatusHandler == nil {
		c.commitStatusHandler = handler.NewCommitStatusHandler(
			c.CommitStatusService(),
			c.RepoService(),
		)
	}
	return c.commitStatusHandler
}

func (c *Container) DeployKeyHandler() *handler.DeployKeyHandler {
	if c.deployKeyHandler == nil {
		c.deployKeyHandler = handler.NewDeployKeyHandler(
			c.DeployKeyService(),
			c.RepoService(),
		)
	}
	return c.deployKeyHandler
}

func (c *Container) ReviewHandler() *handler.ReviewHandler {
	if c.reviewHandler == nil {
		c.reviewHandler = handler.NewReviewHandler(c.ReviewService(), c.RepoService())
	}
	return c.reviewHandler
}

func (c *Container) CodeQualityHandler() *handler.CodeQualityHandler {
	if c.codeQualityHandler == nil {
		c.codeQualityHandler = handler.NewCodeQualityHandler(c.CodeQualityService(), c.RepoService())
	}
	return c.codeQualityHandler
}

func (c *Container) MirrorHandler() *handler.MirrorHandler {
	if c.mirrorHandler == nil {
		c.mirrorHandler = handler.NewMirrorHandler(
			c.MirrorService(),
			c.RepoService(),
		)
	}
	return c.mirrorHandler
}

func (c *Container) ContributorHandler() *handler.ContributorHandler {
	if c.contributorHandler == nil {
		c.contributorHandler = handler.NewContributorHandler(c.ContributorService(), c.RepoService())
	}
	return c.contributorHandler
}

func (c *Container) StatsHandler() *handler.StatsHandler {
	if c.statsHandler == nil {
		c.statsHandler = handler.NewStatsHandler(c.StatsService(), c.RepoService())
	}
	return c.statsHandler
}

func (c *Container) GitHandler() *handler.GitHandler {
	if c.gitHandler == nil {
		c.gitHandler = handler.NewGitHandler(
			c.GitClient(),
			c.RepoService(),
			c.EventBus(),
		)
	}
	return c.gitHandler
}

func (c *Container) AdminHandler() *handler.AdminHandler {
	if c.adminHandler == nil {
		c.adminHandler = handler.NewAdminHandler(
			c.AccountService(), c.RepoService(), c.ProjectService(), c.SystemSettingsService(),
			c.HomeDir(), c.HTTPPort(), c.SSHPort(), c.SSHEnabled(), c.DBDriver(),
		)
	}
	return c.adminHandler
}

func (c *Container) DatabaseViewerHandler() *handler.DatabaseViewerHandler {
	if c.databaseViewerHandler == nil {
		c.databaseViewerHandler = handler.NewDatabaseViewerHandler(c.DatabaseViewerService())
	}
	return c.databaseViewerHandler
}

func (c *Container) OIDCHandler() *handler.OIDCHandler {
	if c.oidcHandler == nil {
		c.oidcHandler = handler.NewOIDCHandler(
			c.OIDCService(),
			c.AccountService(),
		)
	}
	return c.oidcHandler
}

func (c *Container) GPGKeyHandler() *handler.GPGKeyHandler {
	if c.gpgKeyHandler == nil {
		c.gpgKeyHandler = handler.NewGPGKeyHandler(c.GPGKeyService())
	}
	return c.gpgKeyHandler
}

func (c *Container) CustomFieldHandler() *handler.CustomFieldHandler {
	if c.customFieldHandler == nil {
		c.customFieldHandler = handler.NewCustomFieldHandler(c.CustomFieldService(), c.RepoService(), c.EventBus())
	}
	return c.customFieldHandler
}

func (c *Container) SearchHandler() *handler.SearchHandler {
	if c.searchHandler == nil {
		c.searchHandler = handler.NewSearchHandler(c.SearchService(), c.RepoService(), c.AccountService())
	}
	return c.searchHandler
}

func (c *Container) AtomHandler() *handler.AtomHandler {
	if c.atomHandler == nil {
		c.atomHandler = handler.NewAtomHandler(
			c.ActivityService(),
			c.RepoService(),
		)
	}
	return c.atomHandler
}

func (c *Container) ImportHandler() *handler.ImportHandler {
	if c.importHandler == nil {
		c.importHandler = handler.NewImportHandler(
			c.ImportService(),
			c.RepoService(),
		)
	}
	return c.importHandler
}

func (c *Container) PriorityHandler() *handler.PriorityHandler {
	if c.priorityHandler == nil {
		c.priorityHandler = handler.NewPriorityHandler(c.PriorityService(), c.RepoService(), c.EventBus())
	}
	return c.priorityHandler
}

func (c *Container) ExtraMailHandler() *handler.ExtraMailAddressHandler {
	if c.extraMailHandler == nil {
		c.extraMailHandler = handler.NewExtraMailAddressHandler(c.ExtraMailService())
	}
	return c.extraMailHandler
}

func (c *Container) PreferenceHandler() *handler.AccountPreferenceHandler {
	if c.preferenceHandler == nil {
		c.preferenceHandler = handler.NewAccountPreferenceHandler(c.PreferenceService())
	}
	return c.preferenceHandler
}

func (c *Container) ArchiveHandler() *handler.ArchiveHandler {
	if c.archiveHandler == nil {
		c.archiveHandler = handler.NewArchiveHandler(c.GitClient(), c.RepoService())
	}
	return c.archiveHandler
}

func (c *Container) PasswordResetHandler() *handler.PasswordResetHandler {
	if c.passwordResetHandler == nil {
		c.passwordResetHandler = handler.NewPasswordResetHandler(
			c.PasswordResetService(),
			c.AccountService(),
			c.MailService(),
		)
	}
	return c.passwordResetHandler
}

func (c *Container) SystemSettingsHandler() *handler.SystemSettingsHandler {
	if c.systemSettingsHandler == nil {
		c.systemSettingsHandler = handler.NewSystemSettingsHandler(c.SystemSettingsService(), c.AIService(), c.AIModelConfigService())
	}
	return c.systemSettingsHandler
}

func (c *Container) AIModelConfigHandler() *handler.AIModelConfigHandler {
	if c.aiModelConfigHandler == nil {
		c.aiModelConfigHandler = handler.NewAIModelConfigHandler(c.AIModelConfigService(), c.AIService())
	}
	return c.aiModelConfigHandler
}

func (c *Container) PluginHandler() *handler.PluginHandler {
	if c.pluginHandler == nil {
		c.pluginHandler = handler.NewPluginHandler(c.PluginManager(), c.TemplateEngine(), c.PluginEventEngine())
	}
	return c.pluginHandler
}

func (c *Container) MonitorHandler() *handler.MonitorHandler {
	if c.monitorHandler == nil {
		logger := gitsvc.GetLogger()
		metrics := service.GetMetrics()
		c.monitorHandler = handler.NewMonitorHandler(logger, metrics)
	}
	return c.monitorHandler
}

func (c *Container) SetupHandler() *handler.SetupHandler {
	if c.setupHandler == nil {
		c.setupHandler = handler.NewSetupHandler(
			c.SetupService(),
			c.AccountService(),
			c.SystemSettingsService(),
		)
	}
	return c.setupHandler
}

func (c *Container) LFSHandler() *handler.LFSHandler {
	if c.lfsHandler == nil {
		c.lfsHandler = handler.NewLFSHandler(
			c.LFSService(),
			c.RepoService(),
			c.AccountService(),
			c.CollaboratorService(),
		)
	}
	return c.lfsHandler
}

// Project management handlers

func (c *Container) ProjectHandler() *handler.ProjectHandler {
	if c.projectHandler == nil {
		c.projectHandler = handler.NewProjectHandler(
			c.ProjectService(),
			c.AccountService(),
			c.RepoService(),
		)
	}
	return c.projectHandler
}

func (c *Container) SprintHandler() *handler.SprintHandler {
	if c.sprintHandler == nil {
		c.sprintHandler = handler.NewSprintHandler(
			c.SprintService(),
			c.ProjectService(),
			c.ScrumActivityService(),
			c.AIService(),
		)
	}
	return c.sprintHandler
}

func (c *Container) UserStoryHandler() *handler.UserStoryHandler {
	if c.userStoryHandler == nil {
		c.userStoryHandler = handler.NewUserStoryHandler(
			c.UserStoryService(),
			c.ProjectService(),
			c.ScrumActivityService(),
			c.AIService(),
			c.NotificationService(),
			c.EpicService(),
		)
	}
	return c.userStoryHandler
}

func (c *Container) EpicHandler() *handler.EpicHandler {
	if c.epicHandler == nil {
		c.epicHandler = handler.NewEpicHandler(
			c.EpicService(),
			c.ProjectService(),
			c.ScrumActivityService(),
		)
	}
	return c.epicHandler
}

func (c *Container) TaskHandler() *handler.TaskHandler {
	if c.taskHandler == nil {
		c.taskHandler = handler.NewTaskHandler(
			c.TaskService(),
			c.ProjectService(),
			c.NotificationService(),
			c.AccountService(),
			c.ScrumActivityService(),
			c.GitClient(),
			c.RepoService(),
			c.EventBus(),
			c.AIService(),
		)
	}
	return c.taskHandler
}

func (c *Container) TaskStatusHandler() *handler.TaskStatusHandler {
	if c.taskStatusHandler == nil {
		c.taskStatusHandler = handler.NewTaskStatusHandler(
			c.TaskStatusService(),
			c.ProjectService(),
		)
	}
	return c.taskStatusHandler
}

func (c *Container) ScrumActivityHandler() *handler.ScrumActivityHandler {
	if c.scrumActivityHandler == nil {
		c.scrumActivityHandler = handler.NewScrumActivityHandler(c.ScrumActivityService(), c.AccountService())
	}
	return c.scrumActivityHandler
}

// Hub returns the realtime hub, lazily initializing it and injecting it into
// NotificationService so CreateNotification can push to online recipients.
func (c *Container) Hub() *realtime.Hub {
	if c.hub == nil {
		c.hub = realtime.NewHub()
		c.NotificationService().SetHub(c.hub)
	}
	return c.hub
}

// WebSocketHandler returns the WebSocket handler, lazily initializing it.
func (c *Container) WebSocketHandler() *handler.WebSocketHandler {
	if c.webSocketHandler == nil {
		c.webSocketHandler = handler.NewWebSocketHandler(c.Hub())
	}
	return c.webSocketHandler
}
