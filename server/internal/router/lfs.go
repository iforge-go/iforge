package router

import (
	"iforge/iforge/internal/container"

	"github.com/gofiber/fiber/v2"
)

// registerLFSRoutes registers LFS management routes (under /api/v1)
// The LFS protocol endpoints (batch/upload/download/verify) are routed
// separately by the git HTTP middleware in router.go.
func registerLFSRoutes(api fiber.Router, c *container.Container) {
	api.Get("/repos/:owner/:repo/lfs/objects", c.LFSHandler().ListLFSObjects)
	api.Delete("/repos/:owner/:repo/lfs/objects/:oid", c.LFSHandler().DeleteLFSObject)
}
