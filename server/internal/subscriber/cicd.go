package subscriber

import (
	"fmt"
	"strings"

	"iforge/iforge/internal/event"
	"iforge/iforge/internal/git"
	"iforge/iforge/internal/model"
	"iforge/iforge/internal/service"
)

// CICDSubscriber listens for push events and automatically triggers CI/CD
// pipelines.
//
// Trigger conditions:
//   - A .iforge-ci.yml file exists at the repository root.
//   - Branch push: triggers a regular pipeline (refs/heads/<branch>).
//   - Tag push: triggers a tag pipeline (refs/tags/<tag>), used for release
//     jobs that auto-publish.
//
// Flow:
//  1. Extract repo, ref, pusher and latest commit from BranchPushedData /
//     TagPushedData.
//  2. Read .iforge-ci.yml via GitClient.
//  3. Call PipelineService.CreatePipeline to create the pipeline and schedule
//     its jobs.
//
// Any error is logged only; the originating push has already succeeded and
// must not be affected.
type CICDSubscriber struct {
	pipelineService *service.PipelineService
	gitClient       *git.Client
	logger          *git.Logger
}

// NewCICDSubscriber creates a new CICDSubscriber
func NewCICDSubscriber(pipelineService *service.PipelineService, gitClient *git.Client, logger *git.Logger) *CICDSubscriber {
	return &CICDSubscriber{
		pipelineService: pipelineService,
		gitClient:       gitClient,
		logger:          logger,
	}
}

// Register subscribes this handler to the relevant event types on bus.
func (s *CICDSubscriber) Register(bus *event.Bus) {
	bus.Subscribe(event.TypeBranchPushed, s)
	bus.Subscribe(event.TypeTagPushed, s)
}

// HandleEvent implements event.Handler.
func (s *CICDSubscriber) HandleEvent(evt event.Event) {
	switch evt.Type {
	case event.TypeBranchPushed:
		s.handleBranchPushed(evt)
	case event.TypeTagPushed:
		s.handleTagPushed(evt)
	}
}

// handleBranchPushed handles a branch push event.
func (s *CICDSubscriber) handleBranchPushed(evt event.Event) {
	data, ok := evt.Data.(event.BranchPushedData)
	if !ok {
		return
	}

	owner, repoName, ok := splitRepoFullName(data.RepoFullName)
	if !ok {
		s.logger.Error(fmt.Sprintf("CICD: invalid repo full name: %s", data.RepoFullName), nil)
		return
	}

	// Commits[0] is the newest (go-git Log returns commits in reverse time order).
	if len(data.Commits) == 0 {
		s.logger.Info(fmt.Sprintf("CICD: skipping push to %s (no commits)", data.RepoFullName), nil)
		return
	}
	latestCommit := data.Commits[0]
	commitSHA := latestCommit.ID
	commitMessage := latestCommit.Message

	// Keep only the first line so long messages don't blow up list displays.
	if idx := strings.IndexByte(commitMessage, '\n'); idx > 0 {
		commitMessage = commitMessage[:idx]
	}

	// Read .iforge-ci.yml at the latest commit.
	yamlBytes, err := s.gitClient.GetRawFileContent(owner, repoName, commitSHA, ".iforge-ci.yml")
	if err != nil {
		// Missing config file is the normal case; not an error.
		s.logger.Info(fmt.Sprintf("CICD: no .iforge-ci.yml in %s@%s, skipping", data.RepoFullName, commitSHA[:8]), nil)
		return
	}
	yamlConfig := string(yamlBytes)

	pusher := data.PusherName
	if pusher == "" && evt.Sender != nil {
		pusher = evt.Sender.UserName
	}

	// pipeline.ref is stored in "refs/heads/<branch>" form.
	ref := "refs/heads/" + data.BranchName

	pipeline, err := s.pipelineService.CreatePipeline(
		owner, repoName,
		model.TriggerEventPush,
		ref,
		commitSHA,
		pusher,
		commitMessage,
		yamlConfig,
		nil, // merge_request_id
	)
	if err != nil {
		s.logger.Error(fmt.Sprintf("CICD: failed to create pipeline for %s@%s: %v",
			data.RepoFullName, commitSHA[:8], err), nil)
		return
	}

	s.logger.Info(fmt.Sprintf("CICD: pipeline #%d created for %s@%s (branch=%s, jobs triggered)",
		pipeline.ID, data.RepoFullName, commitSHA[:8], data.BranchName), nil)
}

// handleTagPushed handles a tag push event.
//
// Similar to handleBranchPushed, but:
//   - ref is "refs/tags/<tag>".
//   - Used to trigger release pipelines (jobs with a release field auto-create
//     a Release and upload assets).
//   - The commit list may be empty for lightweight tags; in that case the SHA
//     the tag itself points to is used.
func (s *CICDSubscriber) handleTagPushed(evt event.Event) {
	data, ok := evt.Data.(event.TagPushedData)
	if !ok {
		return
	}

	owner, repoName, ok := splitRepoFullName(data.RepoFullName)
	if !ok {
		s.logger.Error(fmt.Sprintf("CICD: invalid repo full name: %s", data.RepoFullName), nil)
		return
	}

	// Tag push commits may be empty (lightweight tag). Use the commit the tag
	// points to. If commits exist, take the first (newest); otherwise fall back
	// to the tag name as commit SHA. ListCommitsBetween may return empty on tag
	// push, so we handle both cases.
	var commitSHA, commitMessage string
	if len(data.Commits) > 0 {
		commitSHA = data.Commits[0].ID
		commitMessage = data.Commits[0].Message
		if idx := strings.IndexByte(commitMessage, '\n'); idx > 0 {
			commitMessage = commitMessage[:idx]
		}
	}

	// Read .iforge-ci.yml at the commit the tag points to.
	// If commitSHA is empty, try reading with the tag name (points to the commit).
	refForRead := commitSHA
	if refForRead == "" {
		refForRead = "refs/tags/" + data.TagName
	}
	yamlBytes, err := s.gitClient.GetRawFileContent(owner, repoName, refForRead, ".iforge-ci.yml")
	if err != nil {
		s.logger.Info(fmt.Sprintf("CICD: no .iforge-ci.yml in %s@%s, skipping tag push", data.RepoFullName, data.TagName), nil)
		return
	}
	yamlConfig := string(yamlBytes)

	// If no commit SHA, try resolving tag → commit SHA via git client
	if commitSHA == "" {
		if resolved, err := s.gitClient.ResolveRef(owner, repoName, "refs/tags/"+data.TagName); err == nil && resolved != "" {
			commitSHA = resolved
		} else {
			s.logger.Error(fmt.Sprintf("CICD: cannot resolve commit SHA for tag %s on %s: %v", data.TagName, data.RepoFullName, err), nil)
			return
		}
	}

	if commitMessage == "" {
		commitMessage = "Tag " + data.TagName
	}

	pusher := data.PusherName
	if pusher == "" && evt.Sender != nil {
		pusher = evt.Sender.UserName
	}

	// Normalize ref (pipeline.ref stores "refs/tags/<tag>" format)
	ref := "refs/tags/" + data.TagName

	pipeline, err := s.pipelineService.CreatePipeline(
		owner, repoName,
		model.TriggerEventPush,
		ref,
		commitSHA,
		pusher,
		commitMessage,
		yamlConfig,
		nil,
	)
	if err != nil {
		s.logger.Error(fmt.Sprintf("CICD: failed to create pipeline for %s@%s (tag=%s): %v",
			data.RepoFullName, commitSHA[:8], data.TagName, err), nil)
		return
	}

	s.logger.Info(fmt.Sprintf("CICD: pipeline #%d created for %s@%s (tag=%s, release pipeline triggered)",
		pipeline.ID, data.RepoFullName, commitSHA[:8], data.TagName), nil)
}

// splitRepoFullName splits "owner/repo" into (owner, repo).
// Returns (_, _, false) if the format is invalid.
func splitRepoFullName(fullName string) (string, string, bool) {
	parts := strings.SplitN(fullName, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}
