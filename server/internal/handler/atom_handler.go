package handler

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"time"

	"iforge/iforge/internal/contextutil"
	"iforge/iforge/internal/model"
	"iforge/iforge/internal/service"

	"github.com/gofiber/fiber/v2"
)

type AtomHandler struct {
	activityService *service.ActivityService
	repoService     *service.RepositoryService
}

func NewAtomHandler(activityService *service.ActivityService, repoService *service.RepositoryService) *AtomHandler {
	return &AtomHandler{
		activityService: activityService,
		repoService:     repoService,
	}
}

type AtomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Xmlns   string      `xml:"xmlns,attr"`
	Title   string      `xml:"title"`
	ID      string      `xml:"id"`
	Updated string      `xml:"updated"`
	Author  AtomAuthor  `xml:"author"`
	Links   []AtomLink  `xml:"link"`
	Entries []AtomEntry `xml:"entry"`
}

type AtomAuthor struct {
	Name string `xml:"name"`
}

type AtomLink struct {
	Rel  string `xml:"rel,attr,omitempty"`
	Href string `xml:"href,attr"`
	Type string `xml:"type,attr,omitempty"`
}

type AtomEntry struct {
	Title   string      `xml:"title"`
	ID      string      `xml:"id"`
	Updated string      `xml:"updated"`
	Author  AtomAuthor  `xml:"author"`
	Content AtomContent `xml:"content"`
	Link    AtomLink    `xml:"link"`
}

type AtomContent struct {
	Type string `xml:"type,attr"`
	Text string `xml:",chardata"`
}

func (h *AtomHandler) GetUserAtomFeed(c *fiber.Ctx) error {
	username := c.Params("username")

	activities, err := h.activityService.GetUserActivities(username, 20, 0)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	feed := h.buildFeed(
		fmt.Sprintf("%s's activities", username),
		fmt.Sprintf("/%s", username),
		activities,
	)

	c.Set("Content-Type", "application/atom+xml; charset=utf-8")
	c.Status(http.StatusOK).XML(feed)
	return nil
}

func (h *AtomHandler) GetRepositoryAtomFeed(c *fiber.Ctx) error {
	owner := c.Params("owner")
	repo := c.Params("repo")

	repository, err := h.repoService.GetRepository(owner, repo)
	if err != nil {
		respondError(c, http.StatusNotFound, "Repository not found")
		return nil
	}

	user := contextutil.GetUserFromContext(c)
	if repository.IsPrivate {
		if user == nil || !h.repoService.HasViewerRole(repository, user) {
			respondError(c, http.StatusNotFound, "Repository not found")
			return nil
		}
	}

	activities, err := h.activityService.GetActivities(owner, repo, 20, 0)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return nil
	}

	feed := h.buildFeed(
		fmt.Sprintf("%s/%s activities", owner, repo),
		fmt.Sprintf("/%s/%s", owner, repo),
		activities,
	)

	c.Set("Content-Type", "application/atom+xml; charset=utf-8")
	c.Status(http.StatusOK).XML(feed)
	return nil
}

func (h *AtomHandler) buildFeed(title, link string, activities []*model.Activity) AtomFeed {
	now := time.Now().Format(time.RFC3339)

	var entries []AtomEntry
	for _, activity := range activities {
		entry := AtomEntry{
			Title:   activity.Message,
			ID:      fmt.Sprintf("tag:iforge,%s:%s", activity.ActivityID, now),
			Updated: activity.ActivityDate.Format(time.RFC3339),
			Author: AtomAuthor{
				Name: activity.UserName,
			},
			Content: AtomContent{
				Type: "html",
				Text: activity.Message,
			},
			Link: AtomLink{
				Href: fmt.Sprintf("/%s/%s", activity.UserName, activity.RepositoryName),
			},
		}
		entries = append(entries, entry)
	}

	return AtomFeed{
		Xmlns:   "http://www.w3.org/2005/Atom",
		Title:   title,
		ID:      link,
		Updated: now,
		Author: AtomAuthor{
			Name: "iForge",
		},
		Links: []AtomLink{
			{Href: link},
			{Rel: "self", Href: link + ".atom"},
		},
		Entries: entries,
	}
}
