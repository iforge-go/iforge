package router

import (
	"iforge/iforge/internal/container"

	"github.com/gofiber/fiber/v2"
)

// registerRepoRoutes registers repository routes
func registerRepoRoutes(api fiber.Router, c *container.Container) {
	api.Get("/repos", c.RepoHandler().ListRepositories)
	api.Post("/repos", c.RepoHandler().CreateRepository)
	api.Post("/repos/import", c.ImportHandler().ImportFromURL)
	api.Get("/repos/parse-url", c.ImportHandler().ParseURL)
	api.Get("/repos/:owner/:repo", c.RepoHandler().GetRepository)
	api.Patch("/repos/:owner/:repo", c.RepoHandler().UpdateRepository)
	api.Delete("/repos/:owner/:repo", c.RepoHandler().DeleteRepository)
	api.Post("/repos/:owner/:repo/fork", c.RepoHandler().ForkRepository)
	api.Get("/repos/:owner/:repo/forks", c.RepoHandler().GetForks)
	api.Get("/repos/:owner/:repo/fork/count", c.RepoHandler().GetForkCount)
	api.Get("/repos/:owner/:repo/fork/isForked", c.RepoHandler().IsForked)
	api.Get("/repos/:owner/:repo/fork/status", c.RepoHandler().GetForkStatus)
	api.Post("/repos/:owner/:repo/fork/sync", c.RepoHandler().SyncFork)
	api.Get("/repos/:owner/:repo/role", c.RepoHandler().GetUserRole)
	api.Post("/repos/:owner/:repo/transfer", c.RepoHandler().TransferRepository)
	api.Post("/repos/:owner/:repo/rename", c.RepoHandler().RenameRepository)
	api.Post("/repos/:owner/:repo/archive", c.RepoHandler().ArchiveRepository)
	api.Post("/repos/:owner/:repo/template", c.RepoHandler().SetTemplateRepository)
	api.Post("/repos/template/create", c.RepoHandler().CreateFromTemplate)
}

// registerGitRoutes registers Git operation routes
func registerGitRoutes(api fiber.Router, c *container.Container) {
	api.Get("/repos/:owner/:repo/files", c.GitHandler().ListFiles)
	api.Get("/repos/:owner/:repo/files/all", c.GitHandler().ListAllFiles)
	api.Get("/repos/:owner/:repo/file", c.GitHandler().GetFileContent)
	api.Post("/repos/:owner/:repo/files", c.GitHandler().CreateFile)
	api.Put("/repos/:owner/:repo/files", c.GitHandler().UpdateFile)
	api.Delete("/repos/:owner/:repo/files", c.GitHandler().DeleteFile)
	api.Get("/repos/:owner/:repo/commits", c.GitHandler().ListCommits)
	api.Get("/repos/:owner/:repo/commits/:commit", c.GitHandler().GetCommit)
	api.Get("/repos/:owner/:repo/branches", c.GitHandler().ListBranches)
	api.Post("/repos/:owner/:repo/branches", c.GitHandler().CreateBranch)
	api.Delete("/repos/:owner/:repo/branches/:branch", c.GitHandler().DeleteBranch)
	api.Post("/repos/:owner/:repo/branches/:branch/rename", c.GitHandler().RenameBranch)
	api.Get("/repos/:owner/:repo/branches/protection", c.GitHandler().ListProtectedBranches)
	api.Post("/repos/:owner/:repo/branches/protection", c.GitHandler().ProtectBranch)
	api.Delete("/repos/:owner/:repo/branches/protection/:branch", c.GitHandler().UnprotectBranch)
	api.Get("/repos/:owner/:repo/search", c.GitHandler().SearchCode)
	api.Get("/repos/:owner/:repo/tags", c.GitHandler().ListTags)
	api.Post("/repos/:owner/:repo/tags", c.GitHandler().CreateTag)
	api.Delete("/repos/:owner/:repo/tags/:tag", c.GitHandler().DeleteTag)
	api.Get("/repos/:owner/:repo/patch/:commit", c.GitHandler().DownloadPatch)
	api.Get("/repos/:owner/:repo/raw/:ref/*", c.GitHandler().DownloadRawFile)
	api.Get("/repos/:owner/:repo/blame", c.GitHandler().BlameFile)
	api.Post("/repos/:owner/:repo/upload", c.GitHandler().UploadFile)
}

// registerReleaseRoutes registers release routes
func registerReleaseRoutes(api fiber.Router, c *container.Container) {
	api.Get("/repos/:owner/:repo/releases", c.ReleaseHandler().ListReleases)
	api.Post("/repos/:owner/:repo/releases", c.ReleaseHandler().CreateRelease)
	api.Get("/repos/:owner/:repo/releases/:tag", c.ReleaseHandler().GetRelease)
	api.Patch("/repos/:owner/:repo/releases/:tag", c.ReleaseHandler().UpdateRelease)
	api.Delete("/repos/:owner/:repo/releases/:tag", c.ReleaseHandler().DeleteRelease)
	api.Get("/repos/:owner/:repo/releases/:tag/assets", c.ReleaseHandler().ListAssets)
	api.Post("/repos/:owner/:repo/releases/:tag/assets", c.ReleaseHandler().UploadAsset)
	api.Get("/repos/:owner/:repo/releases/:tag/assets/:assetId", c.ReleaseHandler().DownloadAsset)
	api.Delete("/repos/:owner/:repo/releases/:tag/assets/:assetId", c.ReleaseHandler().DeleteAsset)
}

// registerCollaboratorRoutes registers collaborator routes
func registerCollaboratorRoutes(api fiber.Router, c *container.Container) {
	api.Get("/repos/:owner/:repo/collaborators", c.CollaboratorHandler().ListCollaborators)
	api.Post("/repos/:owner/:repo/collaborators", c.CollaboratorHandler().AddCollaborator)
	api.Patch("/repos/:owner/:repo/collaborators/:collaborator", c.CollaboratorHandler().UpdateCollaboratorRole)
	api.Delete("/repos/:owner/:repo/collaborators/:collaborator", c.CollaboratorHandler().RemoveCollaborator)
}

// registerDeployKeyRoutes registers deploy key routes
func registerDeployKeyRoutes(api fiber.Router, c *container.Container) {
	api.Get("/repos/:owner/:repo/deploy_keys", c.DeployKeyHandler().ListDeployKeys)
	api.Post("/repos/:owner/:repo/deploy_keys", c.DeployKeyHandler().CreateDeployKey)
	api.Get("/repos/:owner/:repo/deploy_keys/:id", c.DeployKeyHandler().GetDeployKey)
	api.Delete("/repos/:owner/:repo/deploy_keys/:id", c.DeployKeyHandler().DeleteDeployKey)
}

// registerMirrorRoutes registers mirror routes
func registerMirrorRoutes(api fiber.Router, c *container.Container) {
	api.Get("/repos/:owner/:repo/mirror", c.MirrorHandler().GetMirror)
	api.Post("/repos/:owner/:repo/mirror", c.MirrorHandler().CreateMirror)
	api.Put("/repos/:owner/:repo/mirror", c.MirrorHandler().UpdateMirror)
	api.Delete("/repos/:owner/:repo/mirror", c.MirrorHandler().DeleteMirror)
	api.Post("/repos/:owner/:repo/mirror/sync", c.MirrorHandler().SyncMirror)
}

// registerWebhookRoutes registers webhook routes
func registerWebhookRoutes(api fiber.Router, c *container.Container) {
	api.Get("/repos/:owner/:repo/webhooks", c.WebhookHandler().ListWebhooks)
	api.Post("/repos/:owner/:repo/webhooks", c.WebhookHandler().CreateWebhook)
	api.Get("/repos/:owner/:repo/webhooks/:id/deliveries", c.WebhookHandler().ListWebhookDeliveries)
	api.Post("/repos/:owner/:repo/webhooks/:id/test", c.WebhookHandler().TestWebhook)
	api.Delete("/repos/:owner/:repo/webhooks/:id", c.WebhookHandler().DeleteWebhook)
}

// registerCommitStatusRoutes registers commit status routes
func registerCommitStatusRoutes(api fiber.Router, c *container.Container) {
	api.Post("/repos/:owner/:repo/statuses/:commit", c.CommitStatusHandler().CreateStatus)
	api.Get("/repos/:owner/:repo/statuses/:commit", c.CommitStatusHandler().ListStatuses)
	api.Get("/repos/:owner/:repo/commits/:commit/status", c.CommitStatusHandler().GetCombinedStatus)
}

// registerStarRoutes registers star routes
func registerStarRoutes(api fiber.Router, c *container.Container) {
	api.Post("/repos/:owner/:repo/star", c.StarHandler().StarRepository)
	api.Delete("/repos/:owner/:repo/star", c.StarHandler().UnstarRepository)
	api.Get("/repos/:owner/:repo/star", c.StarHandler().IsStarred)
	api.Get("/repos/:owner/:repo/stargazers", c.StarHandler().GetStargazers)
}

// registerWatchRoutes registers watch routes
func registerWatchRoutes(api fiber.Router, c *container.Container) {
	api.Post("/repos/:owner/:repo/watch", c.WatchHandler().WatchRepository)
	api.Delete("/repos/:owner/:repo/watch", c.WatchHandler().UnwatchRepository)
	api.Get("/repos/:owner/:repo/watch", c.WatchHandler().IsWatching)
	api.Get("/repos/:owner/:repo/watchers", c.WatchHandler().GetWatchers)
}

// registerStatsRoutes registers statistics routes
func registerStatsRoutes(api fiber.Router, c *container.Container) {
	api.Get("/repos/:owner/:repo/stats", c.StatsHandler().GetRepositoryStats)
	api.Get("/repos/:owner/:repo/contributors", c.ContributorHandler().GetContributors)
	api.Get("/repos/:owner/:repo/code-quality", c.CodeQualityHandler().AnalyzeCodeQuality)
}
