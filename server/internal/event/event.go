package event

import (
	"iforge/iforge/internal/git"
	"iforge/iforge/internal/model"
	"time"
)

// Type represents the category of an event. It follows standard webhook
// event naming conventions so webhook payloads stay compatible.
type Type string

const (
	TypePush               Type = "push"
	TypeIssues             Type = "issues"
	TypeIssueComment       Type = "issue_comment"
	TypeMergeRequest       Type = "pull_request" // external string kept for webhook compat
	TypeRelease            Type = "release"
	TypePing               Type = "ping"
	TypeCollaboratorAdded  Type = "collaborator_added"
	TypeRepositoryCreated  Type = "repository_created"
	TypeRepositoryDeleted  Type = "repository_deleted"
	TypeRepositoryForked   Type = "repository_forked"
	TypeCustomFieldCreated Type = "custom_field_created"
	TypePriorityCreated    Type = "priority_created"
	TypePushBatch          Type = "push_batch"
	// TypeTaskBranchCreated is published by the Scrum layer when a branch is
	// created and linked to a task. Repo is left nil on the Event; the branch
	// is identified by RepoFullName (owner/repo) in TaskBranchCreatedData.
	TypeTaskBranchCreated Type = "task_branch_created"
	// TypeBranchPushed is published by the VCS layer (HTTP + SSH push handlers)
	// after a successful git-receive-pack. It carries the pushed branch name and
	// commit list so Scrum-domain subscribers can react (e.g., auto-move linked
	// tasks from "todo" to "in_progress" and record commit activity).
	TypeBranchPushed Type = "branch_pushed"
	// TypeTagPushed is published by the VCS layer (HTTP + SSH push handlers)
	// after a successful git-receive-pack for a tag ref (refs/tags/*).
	// Consumed by CICDSubscriber to trigger tag-based pipelines (e.g., release
	// jobs that auto-publish a Release + assets).
	TypeTagPushed Type = "tag_pushed"
)

// Event is a domain event published on the Bus.
//
// Common fields (Repo, Sender, Action) are surfaced for all subscribers.
// Event-specific data is carried by Data; subscribers are responsible for
// shaping it into their own format (e.g. a standard webhook payload).
type Event struct {
	Type   Type
	Action string
	Repo   *model.Repository
	Sender *model.Account
	Data   interface{}
}

// PushData carries the branch and commit list for push events.
type PushData struct {
	Branch  string
	Commits []map[string]interface{}
}

// MergeRequestData carries the issue and merge-request view for merge_request events.
type MergeRequestData struct {
	Issue *model.Issue
	MR    interface{} // *model.MergeRequest or *model.MergeRequestDetail
}

// IssueCommentData carries the issue and comment for issue_comment events.
type IssueCommentData struct {
	Issue   *model.Issue
	Comment *model.IssueComment
}

// PingData carries the webhook id for ping events.
type PingData struct {
	WebhookID uint
}

// NewPushEvent constructs a push event.
func NewPushEvent(repo *model.Repository, sender *model.Account, branch string, commits []map[string]interface{}) Event {
	return Event{
		Type:   TypePush,
		Repo:   repo,
		Sender: sender,
		Data:   PushData{Branch: branch, Commits: commits},
	}
}

// NewIssueEvent constructs an issues event.
func NewIssueEvent(repo *model.Repository, sender *model.Account, action string, issue *model.Issue) Event {
	return Event{
		Type:   TypeIssues,
		Action: action,
		Repo:   repo,
		Sender: sender,
		Data:   issue,
	}
}

// NewIssueCommentEvent constructs an issue_comment event.
func NewIssueCommentEvent(repo *model.Repository, sender *model.Account, action string, issue *model.Issue, comment *model.IssueComment) Event {
	return Event{
		Type:   TypeIssueComment,
		Action: action,
		Repo:   repo,
		Sender: sender,
		Data:   IssueCommentData{Issue: issue, Comment: comment},
	}
}

// NewMergeRequestEvent constructs a merge_request event.
func NewMergeRequestEvent(repo *model.Repository, sender *model.Account, action string, issue *model.Issue, mr interface{}) Event {
	return Event{
		Type:   TypeMergeRequest,
		Action: action,
		Repo:   repo,
		Sender: sender,
		Data:   MergeRequestData{Issue: issue, MR: mr},
	}
}

// NewReleaseEvent constructs a release event.
func NewReleaseEvent(repo *model.Repository, sender *model.Account, action string, release *model.ReleaseTag) Event {
	return Event{
		Type:   TypeRelease,
		Action: action,
		Repo:   repo,
		Sender: sender,
		Data:   release,
	}
}

// NewPingEvent constructs a ping event.
func NewPingEvent(repo *model.Repository, sender *model.Account, webhookID uint) Event {
	return Event{
		Type:   TypePing,
		Repo:   repo,
		Sender: sender,
		Data:   PingData{WebhookID: webhookID},
	}
}

// CollaboratorAddedData carries the added user and role for collaborator_added events.
type CollaboratorAddedData struct {
	CollaboratorName string
	Role             string
}

// NewCollaboratorAddedEvent constructs a collaborator_added event.
func NewCollaboratorAddedEvent(repo *model.Repository, sender *model.Account, collaboratorName, role string) Event {
	return Event{
		Type:   TypeCollaboratorAdded,
		Repo:   repo,
		Sender: sender,
		Data:   CollaboratorAddedData{CollaboratorName: collaboratorName, Role: role},
	}
}

// ForkData carries the source repository info for repository_forked events.
type ForkData struct {
	SourceOwner string
	SourceRepo  string
}

// NewRepositoryCreatedEvent constructs a repository_created event.
func NewRepositoryCreatedEvent(repo *model.Repository, sender *model.Account) Event {
	return Event{
		Type:   TypeRepositoryCreated,
		Repo:   repo,
		Sender: sender,
	}
}

// NewRepositoryDeletedEvent constructs a repository_deleted event.
func NewRepositoryDeletedEvent(repo *model.Repository, sender *model.Account) Event {
	return Event{
		Type:   TypeRepositoryDeleted,
		Repo:   repo,
		Sender: sender,
	}
}

// NewRepositoryForkedEvent constructs a repository_forked event.
// repo is the newly created fork; source info is carried in Data.
func NewRepositoryForkedEvent(repo *model.Repository, sender *model.Account, sourceOwner, sourceRepo string) Event {
	return Event{
		Type:   TypeRepositoryForked,
		Repo:   repo,
		Sender: sender,
		Data:   ForkData{SourceOwner: sourceOwner, SourceRepo: sourceRepo},
	}
}

// CustomFieldCreatedData carries the field name for custom_field_created events.
type CustomFieldCreatedData struct {
	FieldName string
}

// NewCustomFieldCreatedEvent constructs a custom_field_created event.
func NewCustomFieldCreatedEvent(repo *model.Repository, sender *model.Account, fieldName string) Event {
	return Event{
		Type:   TypeCustomFieldCreated,
		Repo:   repo,
		Sender: sender,
		Data:   CustomFieldCreatedData{FieldName: fieldName},
	}
}

// PriorityCreatedData carries the priority name for priority_created events.
type PriorityCreatedData struct {
	PriorityName string
}

// NewPriorityCreatedEvent constructs a priority_created event.
func NewPriorityCreatedEvent(repo *model.Repository, sender *model.Account, priorityName string) Event {
	return Event{
		Type:   TypePriorityCreated,
		Repo:   repo,
		Sender: sender,
		Data:   PriorityCreatedData{PriorityName: priorityName},
	}
}

// PushBatchData carries the pre-serialized ref-update JSON for push_batch events.
// SSH push updates multiple refs at once; the JSON mirrors what was historically
// stored in Activity.AdditionalInfo so subscribers can persist it verbatim.
type PushBatchData struct {
	AdditionalInfo string
}

// NewPushBatchEvent constructs a push_batch event.
func NewPushBatchEvent(repo *model.Repository, sender *model.Account, additionalInfo string) Event {
	return Event{
		Type:   TypePushBatch,
		Repo:   repo,
		Sender: sender,
		Data:   PushBatchData{AdditionalInfo: additionalInfo},
	}
}

// TaskBranchCreatedData carries task + branch context for task_branch_created
// events. The branch is identified by RepoFullName ("owner/repo") plus
// BranchName, keeping the Scrum layer loosely coupled from the VCS tables.
type TaskBranchCreatedData struct {
	ProjectID    int
	TaskID       int
	TaskTitle    string
	RepoFullName string
	BranchName   string
}

// NewTaskBranchCreatedEvent constructs a task_branch_created event.
// Repo is intentionally left nil: this is a Scrum-domain event and the branch
// is identified by RepoFullName in Data.
func NewTaskBranchCreatedEvent(sender *model.Account, projectID, taskID int, taskTitle, repoFullName, branchName string) Event {
	return Event{
		Type:   TypeTaskBranchCreated,
		Action: "created",
		Sender: sender,
		Data: TaskBranchCreatedData{
			ProjectID:    projectID,
			TaskID:       taskID,
			TaskTitle:    taskTitle,
			RepoFullName: repoFullName,
			BranchName:   branchName,
		},
	}
}

// BranchPushedData carries the pushed branch + commit list for branch_pushed
// events. Published by VCS push handlers (HTTP + SSH), consumed by Scrum
// subscribers to auto-transition linked tasks and record commit activity.
//
// Commits uses the lightweight git.PushCommitInfo (no file-change payload).
// PushedAt is the wall-clock time of the push (not the commit author time).
type BranchPushedData struct {
	RepoFullName string // "owner/repo"
	BranchName   string
	PusherName   string
	Commits      []git.PushCommitInfo
	IsNewBranch  bool
	PushedAt     time.Time
}

// NewBranchPushedEvent constructs a branch_pushed event.
// Repo is set to a lightweight Repository{UserName, RepositoryName} so that
// subscribers reading evt.Repo also work; the canonical identifier is
// RepoFullName in Data.
func NewBranchPushedEvent(repo *model.Repository, sender *model.Account, repoFullName, branchName, pusherName string, commits []git.PushCommitInfo, isNewBranch bool) Event {
	return Event{
		Type:   TypeBranchPushed,
		Action: "pushed",
		Repo:   repo,
		Sender: sender,
		Data: BranchPushedData{
			RepoFullName: repoFullName,
			BranchName:   branchName,
			PusherName:   pusherName,
			Commits:      commits,
			IsNewBranch:  isNewBranch,
			PushedAt:     time.Now(),
		},
	}
}

// TagPushedData carries the pushed tag + commit list for tag_pushed events.
// Published by VCS push handlers (HTTP + SSH) when a tag ref is updated.
// Consumed by CICDSubscriber to trigger tag-based pipelines.
type TagPushedData struct {
	RepoFullName string // "owner/repo"
	TagName      string
	PusherName   string
	Commits      []git.PushCommitInfo
	IsNewTag     bool
	PushedAt     time.Time
}

// NewTagPushedEvent constructs a tag_pushed event.
func NewTagPushedEvent(repo *model.Repository, sender *model.Account, repoFullName, tagName, pusherName string, commits []git.PushCommitInfo, isNewTag bool) Event {
	return Event{
		Type:   TypeTagPushed,
		Action: "pushed",
		Repo:   repo,
		Sender: sender,
		Data: TagPushedData{
			RepoFullName: repoFullName,
			TagName:      tagName,
			PusherName:   pusherName,
			Commits:      commits,
			IsNewTag:     isNewTag,
			PushedAt:     time.Now(),
		},
	}
}
