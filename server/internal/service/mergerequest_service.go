package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"iforge/iforge/internal/git"
	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

var (
	ErrMergeRequestNotFound = errors.New("merge request not found")
)

// MergeRequestService handles merge request operations
type MergeRequestService struct {
	db         *gorm.DB
	gitClient  *git.Client
}

// NewMergeRequestService creates a new MergeRequestService
func NewMergeRequestService(db *gorm.DB, gitClient *git.Client) *MergeRequestService {
	return &MergeRequestService{db: db, gitClient: gitClient}
}

type mrDetailRow struct {
	IssueID               int
	Title                 string
	Content               *string
	Closed                bool
	OpenedUserName        string
	RegisteredDate        time.Time
	UpdatedDate           time.Time
	Branch                string
	RequestBranch         string
	RequestUserName       string
	RequestRepositoryName string
	IsDraft               bool
	CommitIDFrom          string
	CommitIDTo            string
	MergedCommitIDs       *string
	MergedCommits         *string
	MergedFileChanges     *string
}

// GetMergeRequest gets a merge request with enriched issue data
func (s *MergeRequestService) GetMergeRequest(owner, repo string, issueId int) (*model.MergeRequestDetail, error) {
	var row mrDetailRow
	err := s.db.
		Table("merge_request").
		Select("merge_request.issue_id, issue.title, issue.content, issue.closed, issue.opened_user_name, issue.registered_date, issue.updated_date, merge_request.branch, merge_request.request_branch, merge_request.request_user_name, merge_request.request_repository_name, merge_request.is_draft, merge_request.commit_id_from, merge_request.commit_id_to, merge_request.merged_commit_ids, merge_request.merged_commits, merge_request.merged_file_changes").
		Joins("INNER JOIN issue ON merge_request.user_name = issue.user_name AND merge_request.repository_name = issue.repository_name AND merge_request.issue_id = issue.issue_id").
		Where("merge_request.user_name = ? AND merge_request.repository_name = ? AND merge_request.issue_id = ?", owner, repo, issueId).
		Scan(&row).Error

	if err != nil {
		return nil, err
	}

	if row.IssueID == 0 {
		return nil, ErrMergeRequestNotFound
	}

	state := "open"
	if row.Closed {
		if row.MergedCommitIDs != nil && *row.MergedCommitIDs != "" {
			state = "merged"
		} else {
			state = "closed"
		}
	}

	return &model.MergeRequestDetail{
		IssueID:               row.IssueID,
		Title:                 row.Title,
		Content:               row.Content,
		State:                 state,
		UserName:              row.OpenedUserName,
		CreatedAt:             row.RegisteredDate,
		UpdatedAt:             row.UpdatedDate,
		Branch:                row.Branch,
		RequestBranch:         row.RequestBranch,
		RequestUserName:       row.RequestUserName,
		RequestRepositoryName: row.RequestRepositoryName,
		IsDraft:               row.IsDraft,
		CommitIDFrom:          row.CommitIDFrom,
		CommitIDTo:            row.CommitIDTo,
		MergedCommitIDs:       row.MergedCommitIDs,
		MergedCommits:         row.MergedCommits,
		MergedFileChanges:     row.MergedFileChanges,
		Merged:                row.MergedCommitIDs != nil && *row.MergedCommitIDs != "",
	}, nil
}

type mrListRow struct {
	IssueID         int
	Title           string
	Closed          bool
	OpenedUserName  string
	RegisteredDate  time.Time
	Branch          string
	RequestBranch   string
	RequestUserName string
	IsDraft         bool
	MergedCommitIDs *string
	CommitIDFrom    string
}

// ListMergeRequests lists merge requests for a repository
func (s *MergeRequestService) ListMergeRequests(owner, repo string, state string, limit, offset int) ([]model.MergeRequestListItem, error) {
	var rows []mrListRow

	query := s.db.
		Table("merge_request").
		Select("merge_request.issue_id, issue.title, issue.closed, issue.opened_user_name, issue.registered_date, merge_request.branch, merge_request.request_branch, merge_request.request_user_name, merge_request.is_draft, merge_request.merged_commit_ids, merge_request.commit_id_from").
		Joins("INNER JOIN issue ON merge_request.user_name = issue.user_name AND merge_request.repository_name = issue.repository_name AND merge_request.issue_id = issue.issue_id").
		Where("merge_request.user_name = ? AND merge_request.repository_name = ?", owner, repo)

	if state == "open" {
		query = query.Where("issue.closed = ?", false)
	} else if state == "closed" {
		query = query.Where("issue.closed = ?", true).Where("merge_request.merged_commit_ids IS NULL OR merge_request.merged_commit_ids = ?", "")
	} else if state == "merged" {
		query = query.Where("merge_request.merged_commit_ids IS NOT NULL AND merge_request.merged_commit_ids != ?", "")
	}

	err := query.
		Order("issue.registered_date DESC").
		Limit(limit).
		Offset(offset).
		Scan(&rows).Error

	items := make([]model.MergeRequestListItem, len(rows))
	for i, r := range rows {
		s := "open"
		if r.Closed {
			if r.MergedCommitIDs != nil && *r.MergedCommitIDs != "" {
				s = "merged"
			} else {
				s = "closed"
			}
		}
		items[i] = model.MergeRequestListItem{
			IssueID:         r.IssueID,
			Title:           r.Title,
			State:           s,
			UserName:        r.OpenedUserName,
			CreatedAt:       r.RegisteredDate,
			Branch:          r.Branch,
			RequestBranch:   r.RequestBranch,
			RequestUserName: r.RequestUserName,
			IsDraft:         r.IsDraft,
			Merged:          r.MergedCommitIDs != nil && *r.MergedCommitIDs != "",
			CommitIDFrom:    r.CommitIDFrom,
		}
	}

	return items, err
}

// CreateMergeRequest creates a new merge request
func (s *MergeRequestService) CreateMergeRequest(
	owner, repo, requestUser, requestRepo, branch, requestBranch, title, content string,
	commitIdFrom, commitIdTo string, isDraft bool, openedUser string,
) (*model.MergeRequest, error) {
	var mr *model.MergeRequest

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Get next issue ID from counter
		var counter model.IssueIDCounter
		err := tx.Where("user_name = ? AND repository_name = ?", owner, repo).First(&counter).Error

		var issueID int
		if errors.Is(err, gorm.ErrRecordNotFound) {
			issueID = 1
			counter = model.IssueIDCounter{
				UserName:       owner,
				RepositoryName: repo,
				IssueID:        1,
			}
			if err := tx.Create(&counter).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			issueID = counter.IssueID + 1
			if err := tx.Model(&counter).Update("issue_id", issueID).Error; err != nil {
				return err
			}
		}

		// Create issue
		now := time.Now()
		issue := &model.Issue{
			UserName:       owner,
			RepositoryName: repo,
			IssueID:        issueID,
			OpenedUserName: openedUser,
			Title:          title,
			Content:        &content,
			Closed:         false,
			RegisteredDate: now,
			UpdatedDate:    now,
			IsMergeRequest: true,
		}

		if err := tx.Create(issue).Error; err != nil {
			return err
		}

		// Create merge request
		mr = &model.MergeRequest{
			UserName:              owner,
			RepositoryName:        repo,
			IssueID:               issueID,
			Branch:                branch,
			RequestUserName:       requestUser,
			RequestRepositoryName: requestRepo,
			RequestBranch:         requestBranch,
			CommitIDFrom:          commitIdFrom,
			CommitIDTo:            commitIdTo,
			IsDraft:               isDraft,
		}

		if err := tx.Create(mr).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return mr, nil
}

// UpdateMergeRequest updates a merge request
func (s *MergeRequestService) UpdateMergeRequest(owner, repo string, issueId int, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	gormUpdates := make(map[string]interface{})

	if draft, ok := updates["is_draft"].(bool); ok {
		gormUpdates["is_draft"] = draft
	}
	if v, ok := updates["commit_id_from"]; ok {
		gormUpdates["commit_id_from"] = v
	}
	if v, ok := updates["commit_id_to"]; ok {
		gormUpdates["commit_id_to"] = v
	}

	if len(gormUpdates) == 0 {
		return nil
	}

	return s.db.
		Model(&model.MergeRequest{}).
		Where("user_name = ? AND repository_name = ? AND issue_id = ?", owner, repo, issueId).
		Updates(gormUpdates).Error
}

// MergeMergeRequest marks a merge request as merged
func (s *MergeRequestService) MergeMergeRequest(owner, repo string, issueId int, mergedCommitIds string, mergedCommits *string, mergedFileChanges *string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Update merge request
		updates := map[string]interface{}{
			"merged_commit_ids": mergedCommitIds,
		}
		if mergedCommits != nil {
			updates["merged_commits"] = *mergedCommits
		}
		if mergedFileChanges != nil {
			updates["merged_file_changes"] = *mergedFileChanges
		}

		if err := tx.Model(&model.MergeRequest{}).
			Where("user_name = ? AND repository_name = ? AND issue_id = ?", owner, repo, issueId).
			Updates(updates).Error; err != nil {
			return err
		}

		// Close the issue
		return tx.Model(&model.Issue{}).
			Where("user_name = ? AND repository_name = ? AND issue_id = ?", owner, repo, issueId).
			Updates(map[string]interface{}{
				"closed":       true,
				"updated_date": time.Now(),
			}).Error
	})
}

// MergeStrategy represents the merge strategy to use
type MergeStrategy string

const (
	MergeStrategyMergeCommit MergeStrategy = "merge-commit"
	MergeStrategySquash      MergeStrategy = "squash"
	MergeStrategyRebase      MergeStrategy = "rebase"
)

// MergeMergeRequestWithStrategy merges a merge request using the specified strategy
func (s *MergeRequestService) MergeMergeRequestWithStrategy(
	owner, repo string,
	issueId int,
	strategy MergeStrategy,
	message string,
	authorName, authorEmail string,
) (string, error) {
	mr, err := s.GetMergeRequest(owner, repo, issueId)
	if err != nil {
		return "", err
	}

	// Check if MR is already merged
	if mr.MergedCommitIDs != nil && *mr.MergedCommitIDs != "" {
		return "", errors.New("merge request already merged")
	}

	// Check if MR is draft
	if mr.IsDraft {
		return "", errors.New("cannot merge draft merge request")
	}

	// Determine if this is a cross-repo MR
	isCrossRepo := mr.RequestUserName != owner || mr.RequestRepositoryName != repo

	// For cross-repo MR, fetch the latest changes to refs/mr/{issueId}/head
	headRef := mr.RequestBranch
	if isCrossRepo {
		mrRef := fmt.Sprintf("refs/mr/%d/head", issueId)
		err = s.gitClient.FetchBranch(
			owner, repo,
			mr.RequestUserName, mr.RequestRepositoryName,
			mr.RequestBranch,
			mrRef,
		)
		if err != nil {
			return "", fmt.Errorf("failed to fetch source branch: %w", err)
		}
		headRef = mrRef
	}

	// Capture compare data before merging (branches will be identical after merge)
	compareResult, _ := s.gitClient.Compare(owner, repo, mr.Branch, headRef)
	var mergedCommitsJSON *string
	var mergedFileChangesJSON *string
	if compareResult != nil {
		commits := compareResult.Commits
		if commits == nil {
			commits = []git.CommitInfo{}
		}
		commitsBytes, _ := json.Marshal(commits)
		mergedCommitsJSON = strPtr(string(commitsBytes))
		files := compareResult.FileChanges
		if files == nil {
			files = []git.FileChange{}
		}
		filesBytes, _ := json.Marshal(files)
		mergedFileChangesJSON = strPtr(string(filesBytes))
	}

	// Use injected git service to perform the merge
	var mergeCommitId string
	var mergeMessage string

	switch strategy {
	case MergeStrategyMergeCommit:
		if message == "" {
			mergeMessage = fmt.Sprintf("Merge MR #%d", issueId)
		} else {
			mergeMessage = message
		}
		mergeCommitId, err = s.gitClient.MergeBranch(
			owner, repo,
			mr.Branch, headRef,
			mergeMessage, authorName, authorEmail,
		)
		if err != nil {
			return "", err
		}

	case MergeStrategySquash:
		if message == "" {
			mergeMessage = fmt.Sprintf("Squash and merge MR #%d", issueId)
		} else {
			mergeMessage = message
		}
		mergeCommitId, err = s.gitClient.SquashMerge(
			owner, repo,
			mr.Branch, headRef,
			mergeMessage, authorName, authorEmail,
		)
		if err != nil {
			return "", err
		}

	case MergeStrategyRebase:
		if message == "" {
			mergeMessage = fmt.Sprintf("Rebase and merge MR #%d", issueId)
		} else {
			mergeMessage = message
		}
		mergeCommitId, err = s.gitClient.RebaseMerge(
			owner, repo,
			mr.Branch, headRef,
			authorName, authorEmail,
		)
		if err != nil {
			return "", err
		}

	default:
		return "", errors.New("invalid merge strategy")
	}

	// Update MR status with captured data
	err = s.MergeMergeRequest(owner, repo, issueId, mergeCommitId, mergedCommitsJSON, mergedFileChangesJSON)
	if err != nil {
		return "", err
	}

	// Cleanup temporary ref for cross-repo MR
	if isCrossRepo {
		if cleanupErr := s.gitClient.CleanupMRRef(owner, repo, issueId); cleanupErr != nil {
			// Log error but don't fail the merge
			fmt.Printf("Warning: failed to cleanup MR ref: %v\n", cleanupErr)
		}
	}

	return mergeCommitId, nil
}

func strPtr(s string) *string {
	return &s
}
