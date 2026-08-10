package container

import (
	"os"
	"path/filepath"

	"iforge/iforge/internal/event"
	"iforge/iforge/internal/git"
	gitsvc "iforge/iforge/internal/git"
	"iforge/iforge/internal/service"
	"iforge/iforge/internal/subscriber"
)

// Service accessor methods with lazy initialization

// EventBus returns the shared event bus, lazily initializing it and
// registering built-in subscribers on first access.
func (c *Container) EventBus() *event.Bus {
	if c.eventBus == nil {
		c.eventBus = event.NewBus()
		// Webhook delivery subscribes to all webhook-relevant event types.
		c.eventBus.SubscribeAll(subscriber.NewWebhookSubscriber(c.WebhookService()))
		// In-app notifications subscribe to their specific event types.
		// accountService is used by handleMention to validate @mentioned usernames.
		subscriber.NewNotificationSubscriber(c.NotificationService(), c.IssueService(), c.AccountService()).Register(c.eventBus)
		// Activity feed subscribes to repository lifecycle events.
		subscriber.NewActivitySubscriber(c.ActivityService()).Register(c.eventBus)
		// Scrum-layer events (task-branch creation): Scrum activity feed + task notifications.
		subscriber.NewScrumSubscriber(c.ScrumActivityService(), c.NotificationService(), c.TaskService(), c.RepoService(), git.GetLogger()).Register(c.eventBus)
		// CI/CD layer: push events automatically trigger pipelines defined in .iforge-ci.yml.
		// Depends on PipelineService (starts Executor worker) + GitClient (reads yaml) + Logger.
		subscriber.NewCICDSubscriber(c.PipelineService(), c.GitClient(), git.GetLogger()).Register(c.eventBus)
		git.GetLogger().Info("Event bus initialized", nil)
	}
	return c.eventBus
}

func (c *Container) SystemSettingsService() *service.SystemSettingsService {
	if c.systemSettingsService == nil {
		c.systemSettingsService = service.NewSystemSettingsService(c.DB())
	}
	return c.systemSettingsService
}

func (c *Container) LDAPService() *service.LDAPService {
	if c.ldapService == nil {
		c.ldapService = service.NewLDAPService(c.SystemSettingsService())
	}
	return c.ldapService
}

func (c *Container) OIDCService() *service.OIDCService {
	if c.oidcService == nil {
		c.oidcService = service.NewOIDCService(c.SystemSettingsService())
	}
	return c.oidcService
}

func (c *Container) AIModelConfigService() *service.AIModelConfigService {
	if c.aiModelConfigService == nil {
		c.aiModelConfigService = service.NewAIModelConfigService(c.DB())
	}
	return c.aiModelConfigService
}

func (c *Container) AIService() *service.AIService {
	if c.aiService == nil {
		c.aiService = service.NewAIService(c.AIModelConfigService())
	}
	return c.aiService
}

func (c *Container) AccountService() *service.AccountService {
	if c.accountService == nil {
		c.accountService = service.NewAccountService(c.DB(), c.LDAPService())
	}
	return c.accountService
}

func (c *Container) RepoService() *service.RepositoryService {
	if c.repoService == nil {
		c.repoService = service.NewRepositoryService(c.DB(), c.AccountService(), c.GitClient())
	}
	return c.repoService
}

func (c *Container) IssueService() *service.IssueService {
	if c.issueService == nil {
		c.issueService = service.NewIssueService(c.DB())
	}
	return c.issueService
}

func (c *Container) MergeRequestService() *service.MergeRequestService {
	if c.mergeRequestService == nil {
		c.mergeRequestService = service.NewMergeRequestService(c.DB(), c.GitClient())
	}
	return c.mergeRequestService
}

func (c *Container) ActivityService() *service.ActivityService {
	if c.activityService == nil {
		c.activityService = service.NewActivityService(c.DB())
	}
	return c.activityService
}

func (c *Container) WebhookService() *service.WebhookService {
	if c.webhookService == nil {
		c.webhookService = service.NewWebhookService(c.DB())
	}
	return c.webhookService
}

func (c *Container) LabelService() *service.LabelService {
	if c.labelService == nil {
		c.labelService = service.NewLabelService(c.DB())
	}
	return c.labelService
}

func (c *Container) MilestoneService() *service.MilestoneService {
	if c.milestoneService == nil {
		c.milestoneService = service.NewMilestoneService(c.DB())
	}
	return c.milestoneService
}

func (c *Container) CollaboratorService() *service.CollaboratorService {
	if c.collaboratorService == nil {
		c.collaboratorService = service.NewCollaboratorService(c.DB())
	}
	return c.collaboratorService
}

func (c *Container) ReleaseService() *service.ReleaseService {
	if c.releaseService == nil {
		c.releaseService = service.NewReleaseService(c.DB())
	}
	return c.releaseService
}

func (c *Container) StarService() *service.StarService {
	if c.starService == nil {
		c.starService = service.NewStarService(c.DB())
	}
	return c.starService
}

func (c *Container) WatchService() *service.WatchService {
	if c.watchService == nil {
		c.watchService = service.NewWatchService(c.DB())
	}
	return c.watchService
}

func (c *Container) WikiService() *service.WikiService {
	if c.wikiService == nil {
		c.wikiService = service.NewWikiService(c.DB())
	}
	return c.wikiService
}

func (c *Container) SSHKeyService() *service.SSHKeyService {
	if c.sshKeyService == nil {
		c.sshKeyService = service.NewSSHKeyService(c.DB())
	}
	return c.sshKeyService
}

func (c *Container) AccessTokenService() *service.AccessTokenService {
	if c.accessTokenService == nil {
		c.accessTokenService = service.NewAccessTokenService(c.DB())
	}
	return c.accessTokenService
}

func (c *Container) NotificationService() *service.NotificationService {
	if c.notificationService == nil {
		c.notificationService = service.NewNotificationService(c.DB())
	}
	return c.notificationService
}

func (c *Container) CommitStatusService() *service.CommitStatusService {
	if c.commitStatusService == nil {
		c.commitStatusService = service.NewCommitStatusService(c.DB())
	}
	return c.commitStatusService
}

func (c *Container) DeployKeyService() *service.DeployKeyService {
	if c.deployKeyService == nil {
		c.deployKeyService = service.NewDeployKeyService(c.DB())
	}
	return c.deployKeyService
}

func (c *Container) ReviewService() *service.ReviewService {
	if c.reviewService == nil {
		c.reviewService = service.NewReviewService(c.DB())
	}
	return c.reviewService
}

func (c *Container) CodeQualityService() *service.CodeQualityService {
	if c.codeQualityService == nil {
		c.codeQualityService = service.NewCodeQualityService(c.DB(), c.GitClient())
	}
	return c.codeQualityService
}

func (c *Container) MirrorService() *service.MirrorService {
	if c.mirrorService == nil {
		c.mirrorService = service.NewMirrorService(c.DB())
	}
	return c.mirrorService
}

func (c *Container) DatabaseViewerService() *service.DatabaseViewerService {
	if c.databaseViewerService == nil {
		c.databaseViewerService = service.NewDatabaseViewerService(c.DB())
	}
	return c.databaseViewerService
}

func (c *Container) GPGKeyService() *service.GPGKeyService {
	if c.gpgKeyService == nil {
		c.gpgKeyService = service.NewGPGKeyService(c.DB())
	}
	return c.gpgKeyService
}

func (c *Container) CustomFieldService() *service.CustomFieldService {
	if c.customFieldService == nil {
		c.customFieldService = service.NewCustomFieldService(c.DB())
	}
	return c.customFieldService
}

func (c *Container) SearchService() *service.SearchService {
	if c.searchService == nil {
		c.searchService = service.NewSearchService(c.DB())
	}
	return c.searchService
}

func (c *Container) PriorityService() *service.PriorityService {
	if c.priorityService == nil {
		c.priorityService = service.NewPriorityService(c.DB())
	}
	return c.priorityService
}

func (c *Container) ExtraMailService() *service.ExtraMailAddressService {
	if c.extraMailService == nil {
		c.extraMailService = service.NewExtraMailAddressService(c.DB())
	}
	return c.extraMailService
}

func (c *Container) PreferenceService() *service.AccountPreferenceService {
	if c.preferenceService == nil {
		c.preferenceService = service.NewAccountPreferenceService(c.DB())
	}
	return c.preferenceService
}

func (c *Container) PasswordResetService() *service.PasswordResetService {
	if c.passwordResetService == nil {
		c.passwordResetService = service.NewPasswordResetService(c.DB())
	}
	return c.passwordResetService
}

func (c *Container) PluginManager() *service.PluginManager {
	if c.pluginManager == nil {
		c.pluginManager = service.NewPluginManager(c.DB())
	}
	return c.pluginManager
}

// TemplateEngine returns the plugin template engine, lazily initializing it.
func (c *Container) TemplateEngine() *service.TemplateEngine {
	if c.templateEngine == nil {
		c.templateEngine = service.NewTemplateEngine(c.DB())
		// Load builtin templates on first access
		if err := c.templateEngine.LoadBuiltinTemplates(); err != nil {
			logger := gitsvc.GetLogger()
			logger.Error("Failed to load builtin plugin templates: "+err.Error(), nil)
		}
	}
	return c.templateEngine
}

// PluginEventEngine returns the plugin event engine, lazily initializing it.
func (c *Container) PluginEventEngine() *service.PluginEventEngine {
	if c.eventEngine == nil {
		c.eventEngine = service.NewPluginEventEngine(c.DB(), c.MailService())
		// Load handlers from database on first access
		if err := c.eventEngine.LoadHandlers(); err != nil {
			logger := gitsvc.GetLogger()
			logger.Error("Failed to load plugin event handlers: "+err.Error(), nil)
		}
	}
	return c.eventEngine
}

func (c *Container) MailService() *service.MailService {
	if c.mailService == nil {
		c.mailService = service.NewMailService(&service.SMTPConfig{})
	}
	return c.mailService
}

func (c *Container) ImportService() *service.ImportService {
	if c.importService == nil {
		c.importService = service.NewImportService(c.GitClient())
	}
	return c.importService
}

func (c *Container) GitClient() *git.Client {
	if c.gitClient == nil {
		c.gitClient = git.NewClient(c.reposPath)
	}
	return c.gitClient
}

func (c *Container) LFSService() *service.LFSService {
	if c.lfsService == nil {
		c.lfsService = service.NewLFSService(c.DB(), c.GitClient())
	}
	return c.lfsService
}

func (c *Container) StatsService() *service.StatsService {
	if c.statsService == nil {
		c.statsService = service.NewStatsService(c.DB(), c.GitClient())
	}
	return c.statsService
}

func (c *Container) ContributorService() *service.ContributorService {
	if c.contributorService == nil {
		c.contributorService = service.NewContributorService(c.GitClient())
	}
	return c.contributorService
}

func (c *Container) SetupService() *service.SetupService {
	if c.setupService == nil {
		c.setupService = service.NewSetupService(c.DB())
	}
	return c.setupService
}

// Project management services

func (c *Container) ProjectService() *service.ProjectService {
	if c.projectService == nil {
		c.projectService = service.NewProjectService(c.DB())
	}
	return c.projectService
}

func (c *Container) SprintService() *service.SprintService {
	if c.sprintService == nil {
		c.sprintService = service.NewSprintService(c.DB())
	}
	return c.sprintService
}

func (c *Container) UserStoryService() *service.UserStoryService {
	if c.userStoryService == nil {
		c.userStoryService = service.NewUserStoryService(c.DB())
	}
	return c.userStoryService
}

func (c *Container) TaskService() *service.TaskService {
	if c.taskService == nil {
		c.taskService = service.NewTaskService(c.DB())
	}
	return c.taskService
}

func (c *Container) TaskStatusService() *service.TaskStatusService {
	if c.taskStatusService == nil {
		c.taskStatusService = service.NewTaskStatusService(c.DB())
	}
	return c.taskStatusService
}

func (c *Container) ScrumActivityService() *service.ScrumActivityService {
	if c.scrumActivityService == nil {
		c.scrumActivityService = service.NewScrumActivityService(c.DB())
	}
	return c.scrumActivityService
}

func (c *Container) EpicService() *service.EpicService {
	if c.epicService == nil {
		c.epicService = service.NewEpicService(c.DB())
	}
	return c.epicService
}

// CI/CD services (built-in Pipeline/Job/Runner/Deployment full pipeline)

// PipelineService returns the Pipeline service, lazily initialized.
// On first access, also starts the built-in Executor (a goroutine that schedules pending jobs serially).
func (c *Container) PipelineService() *service.PipelineService {
	if c.pipelineService == nil {
		c.pipelineService = service.NewPipelineService(c.DB())
		// Inject SecretService (used to decrypt secrets into env vars at runtime)
		c.pipelineService.Executor().SetSecretService(c.SecretService())
		// Inject GitClient (used to clone repositories to working directories)
		c.pipelineService.Executor().SetGitClient(c.GitClient())
		// Inject dataDir (used for artifact/cache storage)
		c.pipelineService.Executor().SetDataDir(c.HomeDir())
		// Inject notification services (notify trigger when pipeline reaches terminal state)
		// Both PipelineService and Executor need these (each has its own refreshPipelineStatus)
		c.pipelineService.SetNotificationService(c.NotificationService())
		c.pipelineService.SetMailService(c.MailService())
		c.pipelineService.SetWebhookService(c.WebhookService())
		c.pipelineService.Executor().SetNotificationService(c.NotificationService())
		c.pipelineService.Executor().SetMailService(c.MailService())
		c.pipelineService.Executor().SetWebhookService(c.WebhookService())
		// Inject Release service (auto-create Release + upload assets when release job completes)
		// uploadDir uses the same directory as ReleaseHandler ({reposPath}/_releases)
		c.pipelineService.Executor().SetReleaseService(c.ReleaseService(), filepath.Join(c.ReposPath(), "_releases"))
		// Start built-in Executor worker (can be disabled via IFORGE_DISABLE_BUILTIN_EXECUTOR=true;
		// when disabled, all jobs are handled by external runners)
		if os.Getenv("IFORGE_DISABLE_BUILTIN_EXECUTOR") != "true" {
			c.pipelineService.Executor().Start()
		}
	}
	return c.pipelineService
}

// RunnerService returns the Runner service, lazily initialized.
// On first access, ensures the built-in Shell Runner exists (system-level runner, works out of the box).
func (c *Container) RunnerService() *service.RunnerService {
	if c.runnerService == nil {
		c.runnerService = service.NewRunnerService(c.DB())
		// Ensure built-in runner exists (idempotent; returns immediately if already present)
		// Can be skipped via IFORGE_DISABLE_BUILTIN_EXECUTOR=true (when using external runners)
		if os.Getenv("IFORGE_DISABLE_BUILTIN_EXECUTOR") != "true" {
			if _, err := c.runnerService.EnsureBuiltinRunner("builtin-shell", "shell"); err != nil {
				// Don't block startup; just log the error
				git.GetLogger().Error("Failed to ensure builtin shell runner: "+err.Error(), nil)
			}
		}
		// Start stale runner detection goroutine (scans every 30s; marks runners offline after 90s without heartbeat)
		go c.runnerService.StartStaleChecker()
	}
	return c.runnerService
}

// SecretService returns the encrypted variables service, lazily initialized.
func (c *Container) SecretService() *service.SecretService {
	if c.secretService == nil {
		c.secretService = service.NewSecretService(c.DB())
	}
	return c.secretService
}

// DeploymentService returns the deployment records service, lazily initialized.
func (c *Container) DeploymentService() *service.DeploymentService {
	if c.deploymentService == nil {
		c.deploymentService = service.NewDeploymentService(c.DB())
	}
	return c.deploymentService
}

// CronService returns the scheduled trigger service, lazily initialized.
// On first access, starts the scheduler goroutine (checks for due crons every minute).
// Depends on PipelineService (creates pipelines) + GitClient (reads .iforge-ci.yml and branch heads) + Logger.
func (c *Container) CronService() *service.CronService {
	if c.cronService == nil {
		c.cronService = service.NewCronService(c.DB(), c.PipelineService(), c.GitClient(), git.GetLogger())
		c.cronService.Start()
	}
	return c.cronService
}
