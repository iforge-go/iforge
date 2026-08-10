package subscriber

import (
	"fmt"
	"strings"

	"iforge/iforge/internal/event"
	"iforge/iforge/internal/git"
	"iforge/iforge/internal/model"
	"iforge/iforge/internal/service"
)

// ScrumSubscriber handles Scrum-domain events (task-branch creation, etc.) by
// recording Scrum activities and notifying task participants asynchronously.
//
// It is the Scrum-layer counterpart to NotificationSubscriber/ActivitySubscriber
// (which are VCS-focused): it uses ScrumActivityService and task-scoped
// notifications (CreateTaskNotification) rather than the VCS activity feed and
// repo-scoped notifications. Handlers only publish domain events; this subscriber
// owns the side-effect concerns (message wording, recipient selection).
//
// All handlers are best-effort: failures are logged via logger (when non-nil)
// but never propagated, since this runs in a background goroutine and must not
// affect the originating request.
type ScrumSubscriber struct {
	activityService     *service.ScrumActivityService
	notificationService *service.NotificationService
	taskService         *service.TaskService
	repoService         *service.RepositoryService
	logger              *git.Logger
}

// NewScrumSubscriber creates a new ScrumSubscriber.
// logger may be nil; when non-nil, best-effort operation failures are logged.
func NewScrumSubscriber(activityService *service.ScrumActivityService, notificationService *service.NotificationService, taskService *service.TaskService, repoService *service.RepositoryService, logger *git.Logger) *ScrumSubscriber {
	return &ScrumSubscriber{
		activityService:     activityService,
		notificationService: notificationService,
		taskService:         taskService,
		repoService:         repoService,
		logger:              logger,
	}
}

// logErr logs a best-effort operation failure. Used as `s.logErr("Op", err)`
// in place of `_ = op()` so failures stay observable without being propagated.
func (s *ScrumSubscriber) logErr(op string, err error) {
	if err != nil && s.logger != nil {
		s.logger.Error("scrum subscriber: "+op+" failed", err)
	}
}

// Register subscribes this handler to the relevant event types on bus.
func (s *ScrumSubscriber) Register(bus *event.Bus) {
	bus.Subscribe(event.TypeTaskBranchCreated, s)
	bus.Subscribe(event.TypeBranchPushed, s)
	bus.Subscribe(event.TypeMergeRequest, s)
}

// HandleEvent implements event.Handler.
func (s *ScrumSubscriber) HandleEvent(evt event.Event) {
	switch evt.Type {
	case event.TypeTaskBranchCreated:
		s.handleTaskBranchCreated(evt)
	case event.TypeBranchPushed:
		s.handleBranchPushed(evt)
	case event.TypeMergeRequest:
		s.handleMergeRequest(evt)
	}
}

// notifyTaskAssignees sends a task-scoped notification to all assignees except
// the actor (to avoid self-notification). Best-effort: errors are logged but
// not propagated.
//
// Consolidates the duplicated "fetch assignees → loop → skip self → notify"
// pattern that previously appeared in handleTaskBranchCreated,
// handleBranchPushed, and handleMergeRequest.
func (s *ScrumSubscriber) notifyTaskAssignees(sender string, projectID, taskID int, notificationType, msg string) {
	if s.notificationService == nil || s.taskService == nil {
		return
	}
	assignees, err := s.taskService.GetAssignees(projectID, taskID)
	if err != nil {
		s.logErr("GetAssignees", err)
		return
	}
	if assignees == nil {
		return
	}
	for _, a := range assignees {
		if a.UserName == sender {
			continue
		}
		if err := s.notificationService.CreateTaskNotification(
			a.UserName,
			notificationType,
			sender,
			msg,
			&projectID,
			&taskID,
		); err != nil {
			s.logErr("CreateTaskNotification", err)
		}
	}
}

// handleTaskBranchCreated records a "branch_created" Scrum activity and notifies
// task assignees (except the creator) that a branch was created for their task.
//
// Both actions are best-effort: errors are logged but not propagated because
// this runs in a background goroutine and must not affect the request.
func (s *ScrumSubscriber) handleTaskBranchCreated(evt event.Event) {
	d, ok := evt.Data.(event.TaskBranchCreatedData)
	if !ok {
		return
	}
	sender := evt.Sender.UserName
	branchRef := fmt.Sprintf("%s:%s", d.RepoFullName, d.BranchName)

	// 1. Asynchronously record Scrum activity
	if s.activityService != nil {
		s.logErr("LogActivity(branch_created)", s.activityService.LogActivity(sender, d.ProjectID, "task", d.TaskID,
			"branch_created", "", "", branchRef))
	}

	// 2. Notify task assignees (skip creator to avoid self-notification)
	msg := fmt.Sprintf("%s created branch %s for task #%d", sender, branchRef, d.TaskID)
	s.notifyTaskAssignees(sender, d.ProjectID, d.TaskID, "task_branch_created", msg)
}

// handleBranchPushed reacts to pushes on branches linked to tasks:
//  1. Auto-transitions the task from "todo" → "in_progress" (with status
//     history) when the first push lands on the associated branch.
//  2. Records a "commit_pushed" Scrum activity so the push shows up in the
//     task activity timeline.
//  3. Notifies task assignees (except the pusher) about the new commits.
//
// All actions are best-effort: errors are logged but not propagated because
// this runs in a background goroutine and must not affect the push request.
func (s *ScrumSubscriber) handleBranchPushed(evt event.Event) {
	d, ok := evt.Data.(event.BranchPushedData)
	if !ok {
		return
	}
	if s.taskService == nil {
		return
	}

	// Reverse-lookup: find all tasks linked to this repo + branch.
	links, err := s.taskService.FindTaskBranchesByRepoBranch(d.RepoFullName, d.BranchName)
	if err != nil || len(links) == 0 {
		return
	}

	pusher := d.PusherName
	if evt.Sender != nil && evt.Sender.UserName != "" {
		pusher = evt.Sender.UserName
	}
	branchRef := fmt.Sprintf("%s:%s", d.RepoFullName, d.BranchName)
	commitCount := len(d.Commits)

	for _, link := range links {
		projectID := link.ProjectID
		taskID := link.TaskID

		// 1. Record commit_pushed activity (always, for every push). This comes
		//    before the status change record—the push is the cause, the auto
		//    status change is the effect; causes appear earlier in the timeline.
		if s.activityService != nil {
			s.logErr("LogActivity(commit_pushed)", s.activityService.LogActivity(pusher, projectID, "task", taskID,
				"commit_pushed", "branch", "", branchRef))
		}

		// 2. Auto-transition todo → in_progress (configurable: category=="todo" triggers, target slug resolved by GetInProgressStatusSlug; no-op guard prevents phantom activities).
		task, err := s.taskService.GetTask(projectID, taskID)
		if err == nil && s.taskService.GetStatusCategory(projectID, task.Status) == "todo" {
			inProgressSlug := s.taskService.GetInProgressStatusSlug(projectID)
			if inProgressSlug != task.Status {
				if updateErr := s.taskService.UpdateStatus(projectID, taskID, inProgressSlug, nil); updateErr == nil {
					comment := fmt.Sprintf("Auto: %d commit(s) pushed to %s", commitCount, branchRef)
					s.logErr("RecordStatusHistory(todo→in_progress)", s.taskService.RecordStatusHistory(projectID, taskID, task.Status, inProgressSlug, pusher, &comment))
					if s.activityService != nil {
						s.logErr("LogActivity(status_changed)", s.activityService.LogActivity(pusher, projectID, "task", taskID,
							"status_changed", "status", task.Status, inProgressSlug))
					}
				}
			}
		}

		// 3. Notify assignees (skip pusher to avoid self-notification).
		var msg string
		if commitCount > 0 {
			msg = fmt.Sprintf("%s pushed %d commit(s) to %s for task #%d", pusher, commitCount, branchRef, taskID)
		} else {
			msg = fmt.Sprintf("%s pushed to %s for task #%d", pusher, branchRef, taskID)
		}
		s.notifyTaskAssignees(pusher, projectID, taskID, "task_branch_pushed", msg)
	}
}

// mrBranchInfo extracts source/target branch info from an MR event.
type mrBranchInfo struct {
	SourceRepoFullName string
	SourceBranch       string
	TargetRepoFullName string
	TargetBranch       string
	IsDraft            bool
}

// mrBranchProvider is satisfied by *model.MergeRequest and
// *model.MergeRequestDetail (both implement MRBranch()), letting
// extractMRBranchInfo access branch fields uniformly without duplicating
// field access across type-switch cases.
type mrBranchProvider interface {
	MRBranch() model.MRBranch
}

// extractMRBranchInfo extracts branch info from MergeRequestData.MR (interface{}).
// Works with any type satisfying mrBranchProvider.
func extractMRBranchInfo(mr interface{}) (mrBranchInfo, bool) {
	p, ok := mr.(mrBranchProvider)
	if !ok {
		return mrBranchInfo{}, false
	}
	b := p.MRBranch()
	return mrBranchInfo{
		SourceRepoFullName: b.RequestUserName + "/" + b.RequestRepositoryName,
		SourceBranch:       b.RequestBranch,
		TargetRepoFullName: b.UserName + "/" + b.RepositoryName,
		TargetBranch:       b.Branch,
		IsDraft:            b.IsDraft,
	}, true
}

// parseRepoFullName splits an "owner/repo" string into owner and repo.
// Returns ok=false if the format is invalid. Used instead of strings.Split
// to avoid panics on malformed input.
func parseRepoFullName(repoFullName string) (owner, repo string, ok bool) {
	parts := strings.SplitN(repoFullName, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || strings.Contains(parts[1], "/") {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// handleMergeRequest reacts to merge-request lifecycle events on branches linked to tasks:
//  1. MR opened (non-draft, target=default branch) -> task: in_progress -> review
//  2. MR merged -> task: review/in_progress -> done
//  3. MR closed (unmerged) -> task: review -> in_progress (rollback)
//  4. MR reopened -> task: in_progress -> review
//
// Only MRs targeting the repository's default branch trigger transitions, to keep
// the semantics clean ("merged into mainline" = task done). Draft MRs are skipped
// on "opened" since they indicate work not yet ready for review.
//
// All actions are best-effort: errors are logged but not propagated because
// this runs in a background goroutine and must not affect the request.
func (s *ScrumSubscriber) handleMergeRequest(evt event.Event) {
	d, ok := evt.Data.(event.MergeRequestData)
	if !ok || d.MR == nil {
		return
	}
	if s.taskService == nil || s.repoService == nil {
		return
	}

	info, ok := extractMRBranchInfo(d.MR)
	if !ok {
		return
	}

	// Target repo info from evt.Repo (MergeRequestDetail may not have RepositoryName populated)
	if evt.Repo != nil {
		info.TargetRepoFullName = evt.Repo.UserName + "/" + evt.Repo.RepositoryName
	}

	// Only process MRs targeting the default branch (keep semantics clear: merging to mainline counts as task completion)
	owner, repoName, ok := parseRepoFullName(info.TargetRepoFullName)
	if !ok {
		return
	}
	targetRepo, err := s.repoService.GetRepository(owner, repoName)
	if err != nil {
		return
	}
	if info.TargetBranch != targetRepo.DefaultBranch {
		return
	}

	// Reverse lookup: check if source branch is linked to a task
	links, err := s.taskService.FindTaskBranchesByRepoBranch(info.SourceRepoFullName, info.SourceBranch)
	if err != nil || len(links) == 0 {
		return
	}

	sender := ""
	if evt.Sender != nil {
		sender = evt.Sender.UserName
	}
	mrRef := fmt.Sprintf("%s:%s→%s:%s", info.SourceRepoFullName, info.SourceBranch,
		info.TargetRepoFullName, info.TargetBranch)
	issueID := 0
	if d.Issue != nil {
		issueID = d.Issue.IssueID
	}

	for _, link := range links {
		projectID := link.ProjectID
		taskID := link.TaskID

		// Determine target status from action (configurable resolution).
		var toStatus, activityAction string
		switch evt.Action {
		case "opened":
			// Draft MRs don't trigger review
			if info.IsDraft {
				continue
			}
			toStatus = s.taskService.GetReviewStatusSlug(projectID)
			activityAction = "mr_created"
		case "merged":
			// On merge, transition to done-category regardless of current state
			toStatus = s.taskService.GetStatusSlugByCategory(projectID, "done")
			activityAction = "mr_merged"
		case "closed":
			// Close unmerged: roll back from review to first in_progress status
			toStatus = s.taskService.GetInProgressStatusSlug(projectID)
			activityAction = "mr_closed"
		case "reopened":
			// Reopen: re-enter review
			toStatus = s.taskService.GetReviewStatusSlug(projectID)
			activityAction = "mr_created"
		default:
			continue
		}

		// Only transition when current state matches (avoid incorrect transitions)
		task, err := s.taskService.GetTask(projectID, taskID)
		if err != nil {
			continue
		}

		// Merge scenario: any non-closed state transitions to a done-category
		// target (and target != current to avoid no-op). Other scenarios: only
		// transition when current category == "in_progress" (and target !=
		// current). The no-op guard prevents phantom status_changed activities.
		shouldTransition := false
		actualFrom := task.Status
		if evt.Action == "merged" {
			if !s.taskService.IsClosedStatusByConfig(projectID, task.Status) && toStatus != task.Status {
				shouldTransition = true
			}
		} else if s.taskService.GetStatusCategory(projectID, task.Status) == "in_progress" && toStatus != task.Status {
			shouldTransition = true
		}

		// Record MR activity (always recorded). Before status change record—MR event is the cause,
		// status change is the effect; cause should appear earlier in timeline, effect (later timestamp / larger id) appears above.
		if s.activityService != nil {
			value := fmt.Sprintf("MR #%d", issueID)
			s.logErr("LogActivity(MR)", s.activityService.LogActivity(sender, projectID, "task", taskID,
				activityAction, "", "", value))
		}

		// Execute status transition (only when current state matches)
		if shouldTransition {
			// Optimistic lock: only write toStatus when DB state is still actualFrom.
			// Concurrent losers get RowsAffected=0, preventing duplicate activities.
			changed, err := s.taskService.UpdateStatusIfCurrent(projectID, taskID, actualFrom, toStatus, nil)
			s.logErr("UpdateStatusIfCurrent(MR)", err)
			if changed {
				comment := fmt.Sprintf("Auto: MR #%d %s (%s)", issueID, evt.Action, mrRef)
				s.logErr("RecordStatusHistory(MR)", s.taskService.RecordStatusHistory(projectID, taskID, actualFrom, toStatus, sender, &comment))
				if s.activityService != nil {
					s.logErr("LogActivity(status_changed MR)", s.activityService.LogActivity(sender, projectID, "task", taskID,
						"status_changed", "status", actualFrom, toStatus))
				}
			}
		}

		// Notify assignees (skip the actor to avoid self-notification).
		msg := fmt.Sprintf("%s %s merge request #%d for task #%d", sender, evt.Action, issueID, taskID)
		s.notifyTaskAssignees(sender, projectID, taskID, "task_mr_updated", msg)
	}
}
