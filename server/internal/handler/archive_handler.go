package handler

import (
	"bytes"
	"net/http"
	"strings"

	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/git"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type ArchiveHandler struct {
	gitClient   *git.Client
	repoService *service.RepositoryService
}

func NewArchiveHandler(gitClient *git.Client, repoService *service.RepositoryService) *ArchiveHandler {
	return &ArchiveHandler{
		gitClient:   gitClient,
		repoService: repoService,
	}
}

func (h *ArchiveHandler) DownloadArchive(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")
	filepath := c.Params("filepath")

	repoObj, err := h.repoService.GetRepository(owner, repo)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	if repoObj.IsPrivate {
		user := contextutil.GetUserFromContext(c)
		if user == nil {
			respondError(c, http.StatusUnauthorized, "Unauthorized")
			return nil
		}
		if !h.repoService.HasViewerRole(repoObj, user) {
			respondError(c, http.StatusForbidden, "Forbidden")
			return nil
		}
	}

	var ref, format string

	if strings.HasSuffix(filepath, ".tar.gz") {
		ref = strings.TrimSuffix(filepath, ".tar.gz")
		format = "tar.gz"
	} else if strings.HasSuffix(filepath, ".zip") {
		ref = strings.TrimSuffix(filepath, ".zip")
		format = "zip"
	} else {
		respondError(c, http.StatusBadRequest, "Unsupported archive format")
		return nil
	}

	contentType := "application/zip"
	if format == "tar.gz" {
		contentType = "application/gzip"
	}

	filename := ref + "." + format
	if format == "tar.gz" {
		filename = ref + ".tar.gz"
	} else {
		filename = ref + ".zip"
	}

	c.Set("Content-Type", contentType)
	c.Set("Content-Disposition", "attachment; filename="+filename)
	c.Status(http.StatusOK)

	buf := &bytes.Buffer{}
	_, err = h.gitClient.ArchiveRepository(buf, owner, repo, ref, format)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	c.Send(buf.Bytes())
	return nil
}
