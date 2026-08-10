package router

import (
	"iforge/iforge/internal/container"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// registerUserRoutes registers user-related routes.
// api registers public access routes (viewing public user info), auth registers routes requiring forced authentication.
func registerUserRoutes(api fiber.Router, auth fiber.Router, c *container.Container) {
	// Public routes: unauthenticated users can also view public user info and organization lists
	api.Get("/users/search", c.UserHandler().SearchUsers)
	api.Get("/users/:username", c.UserHandler().GetUser)
	api.Get("/users/:username/repos", c.RepoHandler().GetUserRepositories)
	api.Get("/users/:username/starred", c.StarHandler().GetStarredRepos)
	api.Get("/organizations", c.OrganizationHandler().ListOrganizations)
	api.Get("/organizations/:organizationName", c.OrganizationHandler().GetOrganization)
	api.Get("/organizations/:organizationName/members", c.OrganizationHandler().GetOrganizationMembers)

	// Routes requiring authentication: current user's private data + write operations
	auth.Post("/logout", c.AuthHandler().Logout)
	auth.Get("/user", c.AuthHandler().GetCurrentUser)
	auth.Patch("/user", c.AuthHandler().UpdateCurrentUser)
	auth.Post("/user/change-password", c.AuthHandler().ChangePassword)
	auth.Patch("/users/:username", c.UserHandler().UpdateUser)
	auth.Get("/user/repos", c.RepoHandler().GetMyRepositories)
	auth.Get("/user/organizations", c.OrganizationHandler().ListMyOrganizations)
	auth.Get("/user/managed-organizations", c.OrganizationHandler().ListManagedOrganizations)
	auth.Post("/organizations", c.OrganizationHandler().CreateOrganization)
	auth.Put("/organizations/:organizationName", c.OrganizationHandler().UpdateOrganization)
	auth.Delete("/organizations/:organizationName", c.OrganizationHandler().DeleteOrganization)
	auth.Post("/organizations/:organizationName/members", c.OrganizationHandler().AddOrganizationMember)
	auth.Delete("/organizations/:organizationName/members/:userName", c.OrganizationHandler().RemoveOrganizationMember)
}

// registerSSHKeyRoutes registers SSH key routes
func registerSSHKeyRoutes(api fiber.Router, c *container.Container) {
	api.Get("/user/sshkeys", c.SSHKeyHandler().ListSSHKeys)
	api.Post("/user/sshkeys", c.SSHKeyHandler().CreateSSHKey)
	api.Get("/user/sshkeys/:id", c.SSHKeyHandler().GetSSHKey)
	api.Delete("/user/sshkeys/:id", c.SSHKeyHandler().DeleteSSHKey)
}

// registerGPGKeyRoutes registers GPG key routes
func registerGPGKeyRoutes(api fiber.Router, c *container.Container) {
	api.Get("/user/gpgkeys", c.GPGKeyHandler().ListGPGKeys)
	api.Post("/user/gpgkeys", c.GPGKeyHandler().AddGPGKey)
	api.Get("/user/gpgkeys/:id", c.GPGKeyHandler().GetGPGKey)
	api.Delete("/user/gpgkeys/:id", c.GPGKeyHandler().DeleteGPGKey)
}

// registerAccessTokenRoutes registers access token routes
func registerAccessTokenRoutes(api fiber.Router, c *container.Container) {
	api.Get("/user/tokens", c.AccessTokenHandler().ListAccessTokens)
	api.Post("/user/tokens", c.AccessTokenHandler().CreateAccessToken)
	api.Get("/user/tokens/:id", c.AccessTokenHandler().GetAccessToken)
	api.Delete("/user/tokens/:id", c.AccessTokenHandler().DeleteAccessToken)
}

// registerExtraMailRoutes registers additional email routes
func registerExtraMailRoutes(api fiber.Router, c *container.Container) {
	api.Get("/user/emails", c.ExtraMailHandler().ListExtraMailAddresses)
	api.Post("/user/emails", c.ExtraMailHandler().AddExtraMailAddress)
	api.Delete("/user/emails/:mailAddress", c.ExtraMailHandler().DeleteExtraMailAddress)
	api.Get("/user/emails/primary", c.ExtraMailHandler().GetPrimaryMailAddress)
}

// registerPreferenceRoutes registers preference routes
func registerPreferenceRoutes(api fiber.Router, c *container.Container) {
	api.Get("/user/preferences", c.PreferenceHandler().GetPreference)
	api.Put("/user/preferences", c.PreferenceHandler().UpdatePreference)
}

// registerNotificationRoutes registers notification routes
func registerNotificationRoutes(api fiber.Router, c *container.Container) {
	api.Get("/notifications", c.NotificationHandler().ListNotifications)
	api.Get("/notifications/unread/count", c.NotificationHandler().GetUnreadCount)
	api.Post("/notifications/:notificationSlug/read", c.NotificationHandler().MarkAsRead)
	api.Post("/notifications/read-all", c.NotificationHandler().MarkAllAsRead)
	api.Delete("/notifications/:notificationSlug", c.NotificationHandler().DeleteNotification)
}

// registerWebSocketRoutes registers WebSocket routes
func registerWebSocketRoutes(api fiber.Router, c *container.Container) {
	api.Get("/ws", websocket.New(c.WebSocketHandler().HandleWS))
}

// registerActivityRoutes registers activity routes
func registerActivityRoutes(api fiber.Router, c *container.Container) {
	api.Get("/activities", c.ActivityHandler().ListRecentActivities)
	api.Get("/repos/:owner/:repo/activities", c.ActivityHandler().ListActivities)
	api.Get("/users/:username/activities", c.ActivityHandler().ListUserActivities)
	api.Get("/users/:username/contributions", c.ActivityHandler().GetUserContributions)
}

// registerWikiRoutes registers Wiki routes
func registerWikiRoutes(api fiber.Router, c *container.Container) {
	api.Get("/repos/:owner/:repo/wiki", c.WikiHandler().ListWikiPages)
	api.Post("/repos/:owner/:repo/wiki", c.WikiHandler().CreateWikiPage)
	api.Get("/repos/:owner/:repo/wiki/:page", c.WikiHandler().GetWikiPage)
	api.Put("/repos/:owner/:repo/wiki/:page", c.WikiHandler().UpdateWikiPage)
	api.Delete("/repos/:owner/:repo/wiki/:page", c.WikiHandler().DeleteWikiPage)
}
