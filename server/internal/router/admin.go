package router

import (
	"iforge/iforge/internal/container"
	"iforge/iforge/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

// registerSystemSettingsRoutes registers system settings routes
func registerSystemSettingsRoutes(api fiber.Router, c *container.Container) {
	admin := api.Group("/admin")
	admin.Use(middleware.RequireAdmin())

	admin.Get("/settings/general", c.SystemSettingsHandler().GetGeneralSettings)
	admin.Put("/settings/general", c.SystemSettingsHandler().UpdateGeneralSettings)
	admin.Get("/settings/smtp", c.SystemSettingsHandler().GetSMTPSettings)
	admin.Put("/settings/smtp", c.SystemSettingsHandler().UpdateSMTPSettings)
	admin.Post("/settings/smtp/test", c.SystemSettingsHandler().TestSMTPSettings)
	admin.Get("/settings/ldap", c.SystemSettingsHandler().GetLDAPSettings)
	admin.Put("/settings/ldap", c.SystemSettingsHandler().UpdateLDAPSettings)
	admin.Post("/settings/ldap/test", c.SystemSettingsHandler().TestLDAPSettings)
	admin.Get("/settings/ssh", c.SystemSettingsHandler().GetSSHSettings)
	admin.Put("/settings/ssh", c.SystemSettingsHandler().UpdateSSHSettings)
	admin.Get("/settings/webhook", c.SystemSettingsHandler().GetWebhookSettings)
	admin.Put("/settings/webhook", c.SystemSettingsHandler().UpdateWebhookSettings)
	admin.Get("/settings/upload", c.SystemSettingsHandler().GetUploadSettings)
	admin.Put("/settings/upload", c.SystemSettingsHandler().UpdateUploadSettings)
	admin.Get("/settings/repository", c.SystemSettingsHandler().GetRepositorySettings)
	admin.Put("/settings/repository", c.SystemSettingsHandler().UpdateRepositorySettings)
	admin.Get("/settings/ai", c.SystemSettingsHandler().GetAISettings)
	admin.Put("/settings/ai", c.SystemSettingsHandler().UpdateAISettings)
	admin.Post("/settings/ai/test", c.SystemSettingsHandler().TestAISettings)
	// AI model configuration CRUD
	admin.Get("/settings/ai/models", c.AIModelConfigHandler().List)
	admin.Post("/settings/ai/models", c.AIModelConfigHandler().Create)
	admin.Get("/settings/ai/models/:id", c.AIModelConfigHandler().Get)
	admin.Put("/settings/ai/models/:id", c.AIModelConfigHandler().Update)
	admin.Delete("/settings/ai/models/:id", c.AIModelConfigHandler().Delete)
	admin.Post("/settings/ai/models/:id/default", c.AIModelConfigHandler().SetDefault)
	admin.Post("/settings/ai/models/:id/test", c.AIModelConfigHandler().Test)
	admin.Get("/settings", c.SystemSettingsHandler().GetAllSettings)
}

// registerOIDCRoutes registers OIDC routes
func registerOIDCRoutes(api fiber.Router, c *container.Container) {
	api.Get("/auth/oidc/url", c.OIDCHandler().GetAuthURL)
	api.Post("/auth/oidc/callback", c.OIDCHandler().Callback)

	admin := api.Group("/admin")
	admin.Use(middleware.RequireAdmin())

	admin.Get("/settings/oidc", c.OIDCHandler().GetOIDCSettings)
	admin.Put("/settings/oidc", c.OIDCHandler().UpdateOIDCSettings)
	admin.Post("/settings/oidc/test", c.OIDCHandler().TestOIDCConnection)
}

// registerDatabaseViewerRoutes registers database viewer routes
func registerDatabaseViewerRoutes(api fiber.Router, c *container.Container) {
	admin := api.Group("/admin")
	admin.Use(middleware.RequireAdmin())

	admin.Get("/dbviewer/tables", c.DatabaseViewerHandler().ListTables)
	admin.Get("/dbviewer/tables/:table/schema", c.DatabaseViewerHandler().GetTableSchema)
	admin.Post("/dbviewer/tables/:table/query", c.DatabaseViewerHandler().QueryTable)
	// ExecuteQuery endpoint removed for security. Use QueryTable instead.
}

// registerPluginRoutes registers plugin routes
func registerPluginRoutes(api fiber.Router, c *container.Container) {
	admin := api.Group("/admin")
	admin.Use(middleware.RequireAdmin())

	admin.Get("/plugins", c.PluginHandler().ListPlugins)
	admin.Post("/plugins", c.PluginHandler().RegisterPlugin)
	admin.Get("/plugins/:id", c.PluginHandler().GetPlugin)
	admin.Patch("/plugins/:id", c.PluginHandler().UpdatePlugin)
	admin.Delete("/plugins/:id", c.PluginHandler().UnregisterPlugin)
	admin.Post("/plugins/:id/enable", c.PluginHandler().EnablePlugin)
	admin.Post("/plugins/:id/disable", c.PluginHandler().DisablePlugin)
	admin.Get("/plugins/:id/status", c.PluginHandler().GetPluginStatus)
	admin.Get("/plugins/status", c.PluginHandler().GetAllPluginStatuses)

	// Plugin template routes
	admin.Get("/plugin-templates", c.PluginHandler().ListTemplates)
	admin.Get("/plugin-templates/:templateId", c.PluginHandler().GetTemplate)
	admin.Post("/plugins/install", c.PluginHandler().InstallFromTemplate)
}

// registerAdminRoutes registers admin routes
func registerAdminRoutes(api fiber.Router, c *container.Container) {
	admin := api.Group("/admin")
	admin.Use(middleware.RequireAdmin())
	{
		admin.Get("/users", c.AdminHandler().ListUsers)
		admin.Post("/users", c.AdminHandler().CreateUser)
		admin.Put("/users/:username", c.AdminHandler().UpdateUser)
		admin.Patch("/users/:username", c.AdminHandler().UpdateUser)
		admin.Delete("/users/:username", c.AdminHandler().DeleteUser)
		admin.Get("/system", c.AdminHandler().GetSystemInfo)
		admin.Get("/system/metrics", c.MonitorHandler().GetMetrics)
		admin.Get("/system/metrics/prometheus", c.MonitorHandler().GetPrometheusMetrics)
		admin.Get("/system/logs", c.MonitorHandler().GetLogs)

		// Organization management
		admin.Get("/organizations", c.OrganizationHandler().ListOrganizations)
		admin.Post("/organizations", c.OrganizationHandler().CreateOrganization)
		admin.Put("/organizations/:organizationName", c.OrganizationHandler().UpdateOrganization)
		admin.Patch("/organizations/:organizationName", c.OrganizationHandler().UpdateOrganization)
		admin.Delete("/organizations/:organizationName", c.OrganizationHandler().DeleteOrganization)

		// Repository management
		admin.Get("/repos", c.AdminHandler().ListRepositories)
		admin.Get("/repos/:owner/:repo", c.AdminHandler().GetRepository)
		admin.Delete("/repos/:owner/:repo", c.AdminHandler().DeleteRepository)

		// Audit logs
		admin.Get("/audit-logs", c.AuditHandler().ListAuditLogs)
	}
}
