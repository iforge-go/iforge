// Package subscriber hosts event subscribers (adapters between the
// transport-agnostic event.Bus and downstream consumers such as webhook
// delivery, notifications, audit logging, plugins, ...).
//
// Subscribers depend on event + service + model, but nothing in service
// depends back on subscriber, keeping the dependency graph acyclic and
// the service package focused on business logic.
package subscriber

import (
	"iforge/iforge/internal/event"
	"iforge/iforge/internal/model"
	"iforge/iforge/internal/service"
)

// WebhookSubscriber is an event.Handler that translates domain events into
// standard webhook payloads and delivers them via WebhookService.
type WebhookSubscriber struct {
	webhookService *service.WebhookService
}

// NewWebhookSubscriber creates a new WebhookSubscriber.
func NewWebhookSubscriber(webhookService *service.WebhookService) *WebhookSubscriber {
	return &WebhookSubscriber{webhookService: webhookService}
}

// HandleEvent implements event.Handler.
func (s *WebhookSubscriber) HandleEvent(evt event.Event) {
	payload := s.buildPayload(evt)
	s.webhookService.TriggerRepositoryWebhooks(
		evt.Repo.UserName,
		evt.Repo.RepositoryName,
		string(evt.Type),
		evt.Action,
		payload,
	)
}

// buildPayload assembles a standard webhook payload for evt.
func (s *WebhookSubscriber) buildPayload(evt event.Event) map[string]interface{} {
	payload := map[string]interface{}{
		"action": evt.Action,
		"repository": map[string]interface{}{
			"name":      evt.Repo.RepositoryName,
			"full_name": evt.Repo.UserName + "/" + evt.Repo.RepositoryName,
			"private":   evt.Repo.IsPrivate,
			"html_url":  "/" + evt.Repo.UserName + "/" + evt.Repo.RepositoryName,
			"owner": map[string]interface{}{
				"login":      evt.Repo.UserName,
				"avatar_url": "/avatars/" + evt.Repo.UserName,
			},
		},
		"sender": map[string]interface{}{
			"login":      evt.Sender.UserName,
			"avatar_url": "/avatars/" + evt.Sender.UserName,
		},
	}

	switch evt.Type {
	case event.TypePush:
		s.mergePush(payload, evt.Data)
	case event.TypeIssues:
		s.mergeIssue(payload, evt.Data)
	case event.TypeIssueComment:
		s.mergeIssueComment(payload, evt.Data)
	case event.TypeMergeRequest:
		s.mergeMergeRequest(payload, evt.Data)
	case event.TypeRelease:
		s.mergeRelease(payload, evt.Data)
	case event.TypePing:
		s.mergePing(payload, evt.Data)
	}

	return payload
}

func (s *WebhookSubscriber) mergePush(payload map[string]interface{}, data interface{}) {
	d, ok := data.(event.PushData)
	if !ok {
		return
	}
	payload["ref"] = "refs/heads/" + d.Branch
	payload["commits"] = d.Commits
}

func (s *WebhookSubscriber) mergeIssue(payload map[string]interface{}, data interface{}) {
	issue, ok := data.(*model.Issue)
	if !ok {
		return
	}
	state := "open"
	if issue.Closed {
		state = "closed"
	}
	content := ""
	if issue.Content != nil {
		content = *issue.Content
	}
	payload["issue"] = map[string]interface{}{
		"number": issue.IssueID,
		"title":  issue.Title,
		"state":  state,
		"body":   content,
		"user": map[string]interface{}{
			"login":      issue.OpenedUserName,
			"avatar_url": "/avatars/" + issue.OpenedUserName,
		},
	}
}

func (s *WebhookSubscriber) mergeIssueComment(payload map[string]interface{}, data interface{}) {
	d, ok := data.(event.IssueCommentData)
	if !ok {
		return
	}
	if d.Issue != nil {
		payload["issue"] = map[string]interface{}{
			"number": d.Issue.IssueID,
			"title":  d.Issue.Title,
		}
	}
	if d.Comment != nil {
		payload["comment"] = map[string]interface{}{
			"id":   d.Comment.CommentID,
			"body": d.Comment.Content,
			"user": map[string]interface{}{
				"login":      d.Comment.CommentedUserName,
				"avatar_url": "/avatars/" + d.Comment.CommentedUserName,
			},
		}
	}
}

func (s *WebhookSubscriber) mergeMergeRequest(payload map[string]interface{}, data interface{}) {
	d, ok := data.(event.MergeRequestData)
	if !ok || d.Issue == nil {
		return
	}

	state := "open"
	if d.Issue.Closed {
		state = "closed"
	}

	content := ""
	if d.Issue.Content != nil {
		content = *d.Issue.Content
	}

	var requestBranch, baseBranch string
	var merged bool
	switch v := d.MR.(type) {
	case *model.MergeRequest:
		requestBranch = v.RequestBranch
		baseBranch = v.Branch
		merged = v.MergedCommitIDs != nil
	case *model.MergeRequestDetail:
		requestBranch = v.RequestBranch
		baseBranch = v.Branch
		merged = v.Merged
	}
	if merged {
		state = "merged"
	}

	payload["pull_request"] = map[string]interface{}{
		"number": d.Issue.IssueID,
		"title":  d.Issue.Title,
		"state":  state,
		"body":   content,
		"head": map[string]interface{}{
			"ref": requestBranch,
		},
		"base": map[string]interface{}{
			"ref": baseBranch,
		},
		"merged": merged,
		"user": map[string]interface{}{
			"login":      d.Issue.OpenedUserName,
			"avatar_url": "/avatars/" + d.Issue.OpenedUserName,
		},
	}
}

func (s *WebhookSubscriber) mergeRelease(payload map[string]interface{}, data interface{}) {
	release, ok := data.(*model.ReleaseTag)
	if !ok {
		return
	}
	content := ""
	if release.Content != nil {
		content = *release.Content
	}
	payload["release"] = map[string]interface{}{
		"tag_name":   release.Tag,
		"name":       release.Name,
		"body":       content,
		"draft":      false,
		"prerelease": false,
		"author": map[string]interface{}{
			"login":      release.Author,
			"avatar_url": "/avatars/" + release.Author,
		},
	}
}

func (s *WebhookSubscriber) mergePing(payload map[string]interface{}, data interface{}) {
	d, ok := data.(event.PingData)
	if !ok {
		return
	}
	payload["hook_id"] = d.WebhookID
	payload["hook_type"] = "Repository"
}
