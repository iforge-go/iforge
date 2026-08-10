package subscriber

import (
	"fmt"
	"strconv"

	"iforge/iforge/internal/event"
	"iforge/iforge/internal/model"
	"iforge/iforge/internal/service"
)

// ActivitySubscriber is an event.Handler that records user-facing activity
// entries (the activity feed) for repository lifecycle events.
//
// It owns the mapping from domain event types to activity types, so handlers
// only need to publish events without knowing the activity schema.
type ActivitySubscriber struct {
	activityService *service.ActivityService
}

// NewActivitySubscriber creates a new ActivitySubscriber.
func NewActivitySubscriber(activityService *service.ActivityService) *ActivitySubscriber {
	return &ActivitySubscriber{activityService: activityService}
}

// Register subscribes this handler to the relevant event types on bus.
func (s *ActivitySubscriber) Register(bus *event.Bus) {
	bus.Subscribe(event.TypeRepositoryCreated, s)
	bus.Subscribe(event.TypeRepositoryDeleted, s)
	bus.Subscribe(event.TypeRepositoryForked, s)
	bus.Subscribe(event.TypeIssues, s)
	bus.Subscribe(event.TypeIssueComment, s)
	bus.Subscribe(event.TypeMergeRequest, s)
	bus.Subscribe(event.TypeCustomFieldCreated, s)
	bus.Subscribe(event.TypePriorityCreated, s)
	bus.Subscribe(event.TypePushBatch, s)
}

// HandleEvent implements event.Handler.
func (s *ActivitySubscriber) HandleEvent(evt event.Event) {
	switch evt.Type {
	case event.TypeRepositoryCreated:
		s.handleRepositoryCreated(evt)
	case event.TypeRepositoryDeleted:
		s.handleRepositoryDeleted(evt)
	case event.TypeRepositoryForked:
		s.handleRepositoryForked(evt)
	case event.TypeIssues:
		s.handleIssue(evt)
	case event.TypeIssueComment:
		s.handleIssueComment(evt)
	case event.TypeMergeRequest:
		s.handleMergeRequest(evt)
	case event.TypeCustomFieldCreated:
		s.handleCustomFieldCreated(evt)
	case event.TypePriorityCreated:
		s.handlePriorityCreated(evt)
	case event.TypePushBatch:
		s.handlePushBatch(evt)
	}
}

// handleRepositoryCreated records a "create_repository" activity.
// The activity message is the repository name.
func (s *ActivitySubscriber) handleRepositoryCreated(evt event.Event) {
	s.record(evt, "create_repository", evt.Repo.RepositoryName, nil)
}

// handleRepositoryDeleted records a "delete_repository" activity.
// The activity message is the repository name.
func (s *ActivitySubscriber) handleRepositoryDeleted(evt event.Event) {
	s.record(evt, "delete_repository", evt.Repo.RepositoryName, nil)
}

// handleRepositoryForked records a "fork_repository" activity.
// The activity message is the source repository path (owner/repo).
func (s *ActivitySubscriber) handleRepositoryForked(evt event.Event) {
	d, ok := evt.Data.(event.ForkData)
	if !ok {
		return
	}
	s.record(evt, "fork_repository", fmt.Sprintf("%s/%s", d.SourceOwner, d.SourceRepo), nil)
}

// handleIssue records an issue activity based on evt.Action.
//
//   - opened  → activity type "open_issue"
//   - closed  → activity type "issue_close"
//   - reopened→ activity type "issue_reopen"
//
// The message is the issue number (as string). additionalInfo carries the
// issue number and title as JSON, matching the pre-refactor schema.
func (s *ActivitySubscriber) handleIssue(evt event.Event) {
	issue, ok := evt.Data.(*model.Issue)
	if !ok {
		return
	}

	var activityType string
	switch evt.Action {
	case "opened":
		activityType = "open_issue"
	case "closed":
		activityType = "issue_close"
	case "reopened":
		activityType = "issue_reopen"
	default:
		return
	}

	message := strconv.Itoa(issue.IssueID)
	additionalInfo := fmt.Sprintf(`{"issueNumber":%d,"issueTitle":"%s"}`, issue.IssueID, issue.Title)
	s.record(evt, activityType, message, &additionalInfo)
}

// handleIssueComment records a comment activity.
// If the issue is a merge request, the activity type is "merge_request_comment";
// otherwise it is "issue_comment".
// The message is the issue number; additionalInfo carries issue number and title.
func (s *ActivitySubscriber) handleIssueComment(evt event.Event) {
	d, ok := evt.Data.(event.IssueCommentData)
	if !ok || d.Issue == nil {
		return
	}
	activityType := "issue_comment"
	if d.Issue.IsMergeRequest {
		activityType = "merge_request_comment"
	}
	message := strconv.Itoa(d.Issue.IssueID)
	additionalInfo := fmt.Sprintf(`{"issueNumber":%d,"issueTitle":"%s"}`, d.Issue.IssueID, d.Issue.Title)
	s.record(evt, activityType, message, &additionalInfo)
}

// handleMergeRequest records merge request lifecycle activities.
//   - opened   → activity type "open_merge_request"
//   - closed   → activity type "merge_request_close"
//   - reopened → activity type "merge_request_reopen"
//   - merged   → activity type "merge_request_merge"
//
// The message is the issue number; additionalInfo carries issue number and title.
func (s *ActivitySubscriber) handleMergeRequest(evt event.Event) {
	d, ok := evt.Data.(event.MergeRequestData)
	if !ok || d.Issue == nil {
		return
	}

	var activityType string
	switch evt.Action {
	case "opened":
		activityType = "open_merge_request"
	case "closed":
		activityType = "merge_request_close"
	case "reopened":
		activityType = "merge_request_reopen"
	case "merged":
		activityType = "merge_request_merge"
	default:
		return
	}

	message := strconv.Itoa(d.Issue.IssueID)
	additionalInfo := fmt.Sprintf(`{"issueNumber":%d,"issueTitle":"%s"}`, d.Issue.IssueID, d.Issue.Title)
	s.record(evt, activityType, message, &additionalInfo)
}

// handleCustomFieldCreated records a "create_custom_field" activity.
// The message is the field name; no additionalInfo.
func (s *ActivitySubscriber) handleCustomFieldCreated(evt event.Event) {
	d, ok := evt.Data.(event.CustomFieldCreatedData)
	if !ok {
		return
	}
	s.record(evt, "create_custom_field", d.FieldName, nil)
}

// handlePriorityCreated records a "create_priority" activity.
// The message is the priority name; no additionalInfo.
func (s *ActivitySubscriber) handlePriorityCreated(evt event.Event) {
	d, ok := evt.Data.(event.PriorityCreatedData)
	if !ok {
		return
	}
	s.record(evt, "create_priority", d.PriorityName, nil)
}

// handlePushBatch records a "push" activity for SSH multi-ref pushes.
// The message is the repository name; additionalInfo is the pre-serialized
// ref-update JSON carried by the event.
func (s *ActivitySubscriber) handlePushBatch(evt event.Event) {
	d, ok := evt.Data.(event.PushBatchData)
	if !ok {
		return
	}
	var additionalInfo *string
	if d.AdditionalInfo != "" {
		info := d.AdditionalInfo
		additionalInfo = &info
	}
	s.record(evt, "push", evt.Repo.RepositoryName, additionalInfo)
}

// record is the shared helper that maps event fields to CreateActivity args.
func (s *ActivitySubscriber) record(evt event.Event, activityType, message string, additionalInfo *string) {
	_, _ = s.activityService.CreateActivity(
		evt.Repo.UserName,       // userName (owner of the repo the activity is about)
		evt.Repo.RepositoryName, // repositoryName
		evt.Sender.UserName,     // activityUserName (actor)
		activityType,
		message,
		additionalInfo,
	)
}
