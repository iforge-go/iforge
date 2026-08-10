package subscriber

import (
	"fmt"
	"time"

	"iforge/iforge/internal/event"
	"iforge/iforge/internal/model"
	"iforge/iforge/internal/service"
	"iforge/iforge/internal/util"
)

// NotificationSubscriber is an event.Handler that creates in-app
// notifications for the affected users.
//
// It owns the notification-specific concerns (message wording, self-action
// filtering, recipient selection) so that handlers only need to publish
// domain events without knowing how notifications are shaped or stored.
type NotificationSubscriber struct {
	notificationService *service.NotificationService
	issueService        *service.IssueService
	accountService      *service.AccountService
}

// NewNotificationSubscriber creates a new NotificationSubscriber.
//   - issueService:   looks up historical commenters for comment fan-out
//   - accountService: validates @mentioned usernames exist before notifying
func NewNotificationSubscriber(
	notificationService *service.NotificationService,
	issueService *service.IssueService,
	accountService *service.AccountService,
) *NotificationSubscriber {
	return &NotificationSubscriber{
		notificationService: notificationService,
		issueService:        issueService,
		accountService:      accountService,
	}
}

// HandleEvent implements event.Handler.
func (s *NotificationSubscriber) HandleEvent(evt event.Event) {
	switch evt.Type {
	case event.TypeCollaboratorAdded:
		s.handleCollaboratorAdded(evt)
	case event.TypeIssues:
		s.handleIssue(evt)
	case event.TypeIssueComment:
		s.handleIssueComment(evt)
	case event.TypeMergeRequest:
		s.handleMergeRequest(evt)
	}
}

// Register subscribes this handler to the relevant event types on bus.
func (s *NotificationSubscriber) Register(bus *event.Bus) {
	bus.Subscribe(event.TypeCollaboratorAdded, s)
	bus.Subscribe(event.TypeIssues, s)
	bus.Subscribe(event.TypeIssueComment, s)
	bus.Subscribe(event.TypeMergeRequest, s)
}

// handleMention parses @username mentions in content and notifies each
// mentioned user (excluding the sender). Validates user existence via
// accountService to avoid creating notifications for non-existent users.
//
// repoUserName/repoRepositoryName/issueID/commentID provide context for the
// notification deep-link (so the mentioned user can jump to the source).
//
// Batch optimization: 1 query to validate + 1 INSERT for all notifications, replacing the original N+N DB round trips (N+1 anti-pattern).
func (s *NotificationSubscriber) handleMention(
	evt event.Event,
	content string,
	repoUserName, repoRepositoryName string,
	issueID, commentID *int,
) {
	if s.accountService == nil || content == "" {
		return
	}

	mentions := util.ParseMentions(content)
	if len(mentions) == 0 {
		return
	}

	// Exclude the sender themselves (ParseMentions already deduplicates and
	// preserves order; we only filter out sender here).
	filtered := make([]string, 0, len(mentions))
	for _, u := range mentions {
		if u != evt.Sender.UserName {
			filtered = append(filtered, u)
		}
	}
	if len(filtered) == 0 {
		return
	}

	// Single batch query: filter to usernames that actually exist.
	existing, err := s.accountService.FilterExistingUserNames(filtered)
	if err != nil || len(existing) == 0 {
		return
	}

	// Build notification array
	now := time.Now()
	message := fmt.Sprintf("%s mentioned you in %s/%s",
		evt.Sender.UserName, repoUserName, repoRepositoryName)
	notifications := make([]*model.Notification, 0, len(existing))
	for _, username := range existing {
		notifications = append(notifications, &model.Notification{
			RecipientUserName:  username,
			RepositoryUserName: repoUserName,
			RepositoryName:     repoRepositoryName,
			NotificationType:   "mention",
			IssueID:            issueID,
			CommentID:          commentID,
			Actor:              evt.Sender.UserName,
			Message:            message,
			RegisteredDate:     now,
		})
	}
	// 1 batch write + individual real-time push
	_ = s.notificationService.BatchCreateNotifications(notifications)
}

// handleCollaboratorAdded notifies the added collaborator.
// Self-additions (actor == collaborator) are skipped: there is no point
// notifying someone that they added themselves.
func (s *NotificationSubscriber) handleCollaboratorAdded(evt event.Event) {
	d, ok := evt.Data.(event.CollaboratorAddedData)
	if !ok {
		return
	}

	// Skip self-additions.
	if d.CollaboratorName == evt.Sender.UserName {
		return
	}

	message := fmt.Sprintf("%s added you as %s to %s/%s",
		evt.Sender.UserName, d.Role, evt.Repo.UserName, evt.Repo.RepositoryName)

	_ = s.notificationService.CreateNotification(
		d.CollaboratorName,
		evt.Repo.UserName,
		evt.Repo.RepositoryName,
		string(evt.Type), // "collaborator_added"
		nil,              // issueID
		nil,              // commentID
		evt.Sender.UserName,
		message,
	)
}

// handleIssue dispatches issue notifications based on evt.Action.
//
//   - opened  : notify repo owner (unless they are the creator)
//   - closed  : notify issue author (unless they are the closer)
//   - reopened: notify issue author (unless they are the reopener)
func (s *NotificationSubscriber) handleIssue(evt event.Event) {
	issue, ok := evt.Data.(*model.Issue)
	if !ok {
		return
	}

	var recipient, notificationType, verb string
	switch evt.Action {
	case "opened":
		recipient = evt.Repo.UserName
		notificationType = "issue_open"
		verb = "created"
	case "closed":
		recipient = issue.OpenedUserName
		notificationType = "issue_close"
		verb = "closed"
	case "reopened":
		recipient = issue.OpenedUserName
		notificationType = "issue_reopen"
		verb = "reopened"
	default:
		return
	}

	// Skip self-actions (creator/actor is the same person) for the direct
	// issue notification. Mention parsing still runs below so that other
	// @mentioned users are notified even when the actor is the recipient.
	issueID := issue.IssueID

	if recipient != evt.Sender.UserName {
		message := fmt.Sprintf("%s %s issue #%d", evt.Sender.UserName, verb, issue.IssueID)
		_ = s.notificationService.CreateNotification(
			recipient,
			evt.Repo.UserName,
			evt.Repo.RepositoryName,
			notificationType,
			&issueID,
			nil,
			evt.Sender.UserName,
			message,
		)
	}

	// Parse @mentions in the issue content. Only the opened action introduces
	// new content (closed/reopened typically edit only state).
	if evt.Action == "opened" && issue.Content != nil {
		s.handleMention(evt, *issue.Content, evt.Repo.UserName, evt.Repo.RepositoryName, &issueID, nil)
	}
}

// handleIssueComment notifies the issue author and all past commenters
// about a new comment. The commenter themselves is never notified.
//
// Fan-out logic:
//  1. Notify the issue author (if different from commenter).
//  2. Query all historical comments on the issue.
//  3. Notify every unique past commenter (excluding commenter and author,
//     who was already notified in step 1).
func (s *NotificationSubscriber) handleIssueComment(evt event.Event) {
	d, ok := evt.Data.(event.IssueCommentData)
	if !ok || d.Issue == nil || d.Comment == nil {
		return
	}

	issueID := d.Issue.IssueID
	commentID := d.Comment.CommentID
	message := fmt.Sprintf("%s commented on issue #%d", evt.Sender.UserName, issueID)

	// Track who has already been notified to avoid duplicates.
	notified := make(map[string]bool)
	notified[evt.Sender.UserName] = true // never notify the commenter

	// 1. Notify issue author (if not the commenter).
	if d.Issue.OpenedUserName != evt.Sender.UserName {
		notified[d.Issue.OpenedUserName] = true
		s.sendCommentNotification(d.Issue.OpenedUserName, evt, issueID, commentID, message)
	}

	// 2. Parse @mentions in the comment content. Runs before (and independent
	//    of) the historical-commenter fan-out so a GetComments failure does
	//    not block mention notifications.
	s.handleMention(evt, d.Comment.Content, evt.Repo.UserName, evt.Repo.RepositoryName, &issueID, &commentID)

	// 3. Fan out to past commenters.
	comments, err := s.issueService.GetComments(evt.Repo.UserName, evt.Repo.RepositoryName, issueID)
	if err != nil {
		return
	}
	for _, c := range comments {
		if !notified[c.CommentedUserName] {
			notified[c.CommentedUserName] = true
			s.sendCommentNotification(c.CommentedUserName, evt, issueID, commentID, message)
		}
	}
}

// sendCommentNotification is a thin wrapper for the repeated CreateNotification call.
func (s *NotificationSubscriber) sendCommentNotification(recipient string, evt event.Event, issueID, commentID int, message string) {
	_ = s.notificationService.CreateNotification(
		recipient,
		evt.Repo.UserName,
		evt.Repo.RepositoryName,
		"issue_comment",
		&issueID,
		&commentID,
		evt.Sender.UserName,
		message,
	)
}

// handleMergeRequest dispatches merge-request notifications based on evt.Action.
// Standard behavior:
//   - opened  : notify repo owner (unless they are the creator)
//   - merged  : notify MR author (unless they are the merger)
//   - closed  : notify MR author (unless they are the closer)
//   - reopened: notify MR author (unless they are the reopener)
func (s *NotificationSubscriber) handleMergeRequest(evt event.Event) {
	d, ok := evt.Data.(event.MergeRequestData)
	if !ok || d.Issue == nil {
		return
	}

	var recipient, notificationType, verb string
	switch evt.Action {
	case "opened":
		recipient = evt.Repo.UserName
		notificationType = "open_merge_request"
		verb = "opened merge request"
	case "merged":
		recipient = d.Issue.OpenedUserName
		notificationType = "merge_request_merge"
		verb = "merged merge request"
	case "closed":
		recipient = d.Issue.OpenedUserName
		notificationType = "merge_request_close"
		verb = "closed merge request"
	case "reopened":
		recipient = d.Issue.OpenedUserName
		notificationType = "merge_request_reopen"
		verb = "reopened merge request"
	default:
		return
	}

	// Skip self-actions (actor is the same as recipient) for the direct MR
	// notification. Mention parsing still runs below so other @mentioned
	// users are notified even when the actor is the recipient.
	issueID := d.Issue.IssueID

	if recipient != evt.Sender.UserName {
		message := fmt.Sprintf("%s %s #%d", evt.Sender.UserName, verb, d.Issue.IssueID)
		_ = s.notificationService.CreateNotification(
			recipient,
			evt.Repo.UserName,
			evt.Repo.RepositoryName,
			notificationType,
			&issueID,
			nil,
			evt.Sender.UserName,
			message,
		)
	}

	// Parse @mentions in the MR body. Only the opened action introduces new
	// content (merged/closed/reopened typically edit only state).
	if evt.Action == "opened" && d.Issue.Content != nil {
		s.handleMention(evt, *d.Issue.Content, evt.Repo.UserName, evt.Repo.RepositoryName, &issueID, nil)
	}
}
