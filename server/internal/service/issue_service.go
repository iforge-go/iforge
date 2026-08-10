package service

import (
	"errors"
	"time"

	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

// IssueWithLabels extends Issue with preloaded labels
type IssueWithLabels struct {
	model.Issue
	Labels               []model.Label `json:"labels"`
	CommentsCount        int           `json:"commentsCount"`
	ParticipantUserNames []string      `json:"participantUserNames"`
}

var (
	ErrIssueNotFound = errors.New("issue not found")
)

// IssueService handles issue-related operations
type IssueService struct {
	db *gorm.DB
}

// NewIssueService creates a new IssueService
func NewIssueService(db *gorm.DB) *IssueService {
	return &IssueService{
		db: db,
	}
}

// CreateIssue creates a new issue
func (s *IssueService) CreateIssue(
	owner, repo, openedUser, title string,
	content *string,
	milestoneID, priorityID *int,
	isMergeRequest bool,
) (*model.Issue, error) {
	var createdIssue *model.Issue
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var counter model.IssueIDCounter
		err := tx.Where("user_name = ? AND repository_name = ?", owner, repo).First(&counter).Error

		var issueID int
		if err == gorm.ErrRecordNotFound {
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

		now := time.Now()
		issue := &model.Issue{
			UserName:       owner,
			RepositoryName: repo,
			IssueID:        issueID,
			OpenedUserName: openedUser,
			MilestoneID:    milestoneID,
			PriorityID:     priorityID,
			Title:          title,
			Content:        content,
			Closed:         false,
			RegisteredDate: now,
			UpdatedDate:    now,
			IsMergeRequest: isMergeRequest,
		}

		if err := tx.Create(issue).Error; err != nil {
			return err
		}

		createdIssue = issue
		return nil
	})
	return createdIssue, err
}

// GetIssue retrieves an issue by ID
func (s *IssueService) GetIssue(owner, repo string, issueID int) (*model.Issue, error) {
	var issue model.Issue
	err := s.db.Where("user_name = ? AND repository_name = ? AND issue_id = ?", owner, repo, issueID).First(&issue).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrIssueNotFound
		}
		return nil, err
	}
	return &issue, nil
}

// UpdateIssue updates an issue
func (s *IssueService) UpdateIssue(issue *model.Issue) error {
	issue.UpdatedDate = time.Now()
	return s.db.Where("user_name = ? AND repository_name = ? AND issue_id = ?",
		issue.UserName, issue.RepositoryName, issue.IssueID).
		Save(issue).Error
}

// ListIssues lists issues for a repository
func (s *IssueService) ListIssues(owner, repo string, closed bool, limit, offset int) ([]*model.Issue, error) {
	var issues []*model.Issue
	query := s.db.Where("user_name = ? AND repository_name = ? AND closed = ? AND merge_request = ?",
		owner, repo, closed, false)

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Order("updated_date DESC").Find(&issues).Error
	return issues, err
}

// ListIssuesWithLabels returns issues with their labels preloaded.
// Optimized: uses 3 constant queries instead of N+1 (1 issues + 1 issue_label + 1 label).
func (s *IssueService) ListIssuesWithLabels(owner, repo string, closed bool, limit, offset int) ([]IssueWithLabels, error) {
	issues, err := s.ListIssues(owner, repo, closed, limit, offset)
	if err != nil {
		return nil, err
	}
	if len(issues) == 0 {
		return []IssueWithLabels{}, nil
	}

	// Collect all issue IDs for batch query
	issueIDs := make([]int, len(issues))
	for i, iss := range issues {
		issueIDs[i] = iss.IssueID
	}

	// First batch query: all issue_label associations for the issue IDs
	var allIssueLabels []model.IssueLabel
	if err := s.db.Where("user_name = ? AND repository_name = ? AND issue_id IN ?",
		owner, repo, issueIDs).Find(&allIssueLabels).Error; err != nil {
		// Fallback on query failure: return issues without labels
		result := make([]IssueWithLabels, len(issues))
		for i, iss := range issues {
			result[i] = IssueWithLabels{Issue: *iss, Labels: []model.Label{}}
		}
		return result, nil
	}

	// Build issueID -> []labelID mapping and collect all label IDs
	issueIDToLabelIDs := make(map[int][]int, len(issues))
	labelIDSet := make(map[int]bool)
	for _, il := range allIssueLabels {
		issueIDToLabelIDs[il.IssueID] = append(issueIDToLabelIDs[il.IssueID], il.LabelID)
		labelIDSet[il.LabelID] = true
	}

	// Second batch query: fetch all label details
	labelMap := make(map[int]model.Label)
	if len(labelIDSet) > 0 {
		allLabelIDs := make([]int, 0, len(labelIDSet))
		for id := range labelIDSet {
			allLabelIDs = append(allLabelIDs, id)
		}
		var allLabels []model.Label
		if err := s.db.Where("user_name = ? AND repository_name = ? AND label_id IN ?",
			owner, repo, allLabelIDs).Find(&allLabels).Error; err == nil {
			for _, l := range allLabels {
				labelMap[l.LabelID] = l
			}
		}
	}

	// Assemble results
	result := make([]IssueWithLabels, len(issues))
	for i, iss := range issues {
		labelIDs := issueIDToLabelIDs[iss.IssueID]
		labels := make([]model.Label, 0, len(labelIDs))
		for _, lid := range labelIDs {
			if l, ok := labelMap[lid]; ok {
				labels = append(labels, l)
			}
		}
		result[i] = IssueWithLabels{Issue: *iss, Labels: labels}
	}
	return result, nil
}

// AddComment adds a comment to an issue
func (s *IssueService) AddComment(owner, repo string, issueID int, commentedUser, action, content string) (*model.IssueComment, error) {
	now := time.Now()
	comment := &model.IssueComment{
		UserName:          owner,
		RepositoryName:    repo,
		IssueID:           issueID,
		CommentedUserName: commentedUser,
		Action:            action,
		Content:           content,
		RegisteredDate:    now,
		UpdatedDate:       now,
	}

	if err := s.db.Create(comment).Error; err != nil {
		return nil, err
	}

	// Update issue updated_date
	if err := s.db.Model(&model.Issue{}).
		Where("user_name = ? AND repository_name = ? AND issue_id = ?", owner, repo, issueID).
		Update("updated_date", now).Error; err != nil {
		return nil, err
	}

	return comment, nil
}

// GetComments retrieves all comments for an issue
func (s *IssueService) GetComments(owner, repo string, issueID int) ([]*model.IssueComment, error) {
	var comments []*model.IssueComment
	err := s.db.Where("user_name = ? AND repository_name = ? AND issue_id = ?", owner, repo, issueID).
		Order("registered_date ASC").
		Find(&comments).Error
	return comments, err
}

// LockIssue locks an issue to prevent further comments
func (s *IssueService) LockIssue(owner, repo string, issueID int) error {
	return s.db.Model(&model.Issue{}).
		Where("user_name = ? AND repository_name = ? AND issue_id = ?", owner, repo, issueID).
		Update("locked", true).Error
}

// UnlockIssue unlocks an issue to allow comments
func (s *IssueService) UnlockIssue(owner, repo string, issueID int) error {
	return s.db.Model(&model.Issue{}).
		Where("user_name = ? AND repository_name = ? AND issue_id = ?", owner, repo, issueID).
		Update("locked", false).Error
}

// AddLabel adds a label to an issue
func (s *IssueService) AddLabel(owner, repo string, issueID, labelID int) error {
	issueLabel := &model.IssueLabel{
		UserName:       owner,
		RepositoryName: repo,
		IssueID:        issueID,
		LabelID:        labelID,
	}
	return s.db.Create(issueLabel).Error
}

// RemoveLabel removes a label from an issue
func (s *IssueService) RemoveLabel(owner, repo string, issueID, labelID int) error {
	return s.db.Where("user_name = ? AND repository_name = ? AND issue_id = ? AND label_id = ?",
		owner, repo, issueID, labelID).
		Delete(&model.IssueLabel{}).Error
}

// GetLabels retrieves all labels for an issue
func (s *IssueService) GetLabels(owner, repo string, issueID int) ([]*model.Label, error) {
	var labels []*model.Label
	err := s.db.Joins("JOIN issue_label ON label.label_id = issue_label.label_id").
		Where("issue_label.user_name = ? AND issue_label.repository_name = ? AND issue_label.issue_id = ?",
			owner, repo, issueID).
		Find(&labels).Error
	return labels, err
}

// AddAssignee assigns a user to an issue
func (s *IssueService) AddAssignee(owner, repo string, issueID int, assigneeUserName string) error {
	assignment := &model.IssueAssignment{
		UserName:         owner,
		RepositoryName:   repo,
		IssueID:          issueID,
		AssigneeUserName: assigneeUserName,
	}
	return s.db.Create(assignment).Error
}

// RemoveAssignee removes an assignee from an issue
func (s *IssueService) RemoveAssignee(owner, repo string, issueID int, assigneeUserName string) error {
	return s.db.Where("user_name = ? AND repository_name = ? AND issue_id = ? AND assignee_user_name = ?",
		owner, repo, issueID, assigneeUserName).
		Delete(&model.IssueAssignment{}).Error
}

// SetAssignees replaces all assignees of an issue
func (s *IssueService) SetAssignees(owner, repo string, issueID int, assigneeUserNames []string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_name = ? AND repository_name = ? AND issue_id = ?",
			owner, repo, issueID).
			Delete(&model.IssueAssignment{}).Error; err != nil {
			return err
		}
		for _, username := range assigneeUserNames {
			assignment := &model.IssueAssignment{
				UserName:         owner,
				RepositoryName:   repo,
				IssueID:          issueID,
				AssigneeUserName: username,
			}
			if err := tx.Create(assignment).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetAssignees retrieves all assignees (as Account) for an issue
func (s *IssueService) GetAssignees(owner, repo string, issueID int) ([]*model.Account, error) {
	var accounts []*model.Account
	err := s.db.Joins("JOIN issue_assignment ON account.user_name = issue_assignment.assignee_user_name").
		Where("issue_assignment.user_name = ? AND issue_assignment.repository_name = ? AND issue_assignment.issue_id = ?",
			owner, repo, issueID).
		Find(&accounts).Error
	return accounts, err
}

// CreateLabel creates a new label
func (s *IssueService) CreateLabel(owner, repo, name, color string) (*model.Label, error) {
	label := &model.Label{
		UserName:       owner,
		RepositoryName: repo,
		LabelName:      name,
		Color:          color,
	}
	if err := s.db.Create(label).Error; err != nil {
		return nil, err
	}
	return label, nil
}

// GetRepositoryLabels retrieves all labels for a repository
func (s *IssueService) GetRepositoryLabels(owner, repo string) ([]*model.Label, error) {
	var labels []*model.Label
	err := s.db.Where("user_name = ? AND repository_name = ?", owner, repo).Find(&labels).Error
	return labels, err
}

// BatchUpdateIssues performs batch updates on issues. Supports: close/reopen, set milestone, set priority, add/remove labels
func (s *IssueService) BatchUpdateIssues(
	owner, repo string,
	issueIDs []int,
	closed *bool,
	milestoneID *int,
	priorityID *int,
	addLabelIDs, removeLabelIDs []int,
) error {
	if len(issueIDs) == 0 {
		return nil
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// Batch update main issue table
		updates := map[string]interface{}{
			"updated_date": time.Now(),
		}
		if closed != nil {
			updates["closed"] = *closed
		}
		if milestoneID != nil {
			updates["milestone_id"] = *milestoneID
		}
		if priorityID != nil {
			updates["priority_id"] = *priorityID
		}

		if err := tx.Model(&model.Issue{}).
			Where("user_name = ? AND repository_name = ? AND issue_id IN ?", owner, repo, issueIDs).
			Updates(updates).Error; err != nil {
			return err
		}

		// Batch add labels (optimized: replaced M*N Count queries with 1 query + 1 batch insert)
		if len(addLabelIDs) > 0 && len(issueIDs) > 0 {
			// Query all existing (issue_id, label_id) associations in one query
			var existing []model.IssueLabel
			if err := tx.Where("user_name = ? AND repository_name = ? AND issue_id IN ? AND label_id IN ?",
				owner, repo, issueIDs, addLabelIDs).Find(&existing).Error; err != nil {
				return err
			}
			// Track existing associations in a set to avoid duplicate inserts
			existingSet := make(map[uint64]bool, len(existing))
			for _, e := range existing {
				// Use (issue_id << 32) | label_id as composite key (both are int)
				key := uint64(e.IssueID)<<32 | uint64(uint32(e.LabelID))
				existingSet[key] = true
			}
			// Build rows to insert
			newRows := make([]model.IssueLabel, 0, len(addLabelIDs)*len(issueIDs))
			for _, labelID := range addLabelIDs {
				for _, issueID := range issueIDs {
					key := uint64(issueID)<<32 | uint64(uint32(labelID))
					if existingSet[key] {
						continue
					}
					newRows = append(newRows, model.IssueLabel{
						UserName:       owner,
						RepositoryName: repo,
						IssueID:        issueID,
						LabelID:        labelID,
					})
				}
			}
			// Batch insert
			if len(newRows) > 0 {
				if err := tx.CreateInBatches(newRows, 100).Error; err != nil {
					return err
				}
			}
		}

		// Batch remove labels (single DELETE, unchanged)
		for _, labelID := range removeLabelIDs {
			if err := tx.Where("user_name = ? AND repository_name = ? AND label_id = ? AND issue_id IN ?",
				owner, repo, labelID, issueIDs).
				Delete(&model.IssueLabel{}).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// GetIssueAuthors returns distinct users who have opened issues or merge
// requests in the given repository. When mergeRequest is true, only merge
// request authors are returned; otherwise only issue authors.
func (s *IssueService) GetIssueAuthors(owner, repo string, mergeRequest bool) ([]*model.Account, error) {
	var accounts []*model.Account
	err := s.db.Distinct("account.*").
		Joins("JOIN issue ON issue.opened_user_name = account.user_name").
		Where("issue.user_name = ? AND issue.repository_name = ? AND issue.merge_request = ?",
			owner, repo, mergeRequest).
		Order("account.user_name ASC").
		Find(&accounts).Error
	return accounts, err
}

// GetIssueAssignees returns distinct users who are assigned to issues or
// merge requests in the given repository. When mergeRequest is true, only
// merge request assignees are returned; otherwise only issue assignees.
func (s *IssueService) GetIssueAssignees(owner, repo string, mergeRequest bool) ([]*model.Account, error) {
	var accounts []*model.Account
	err := s.db.Distinct("account.*").
		Joins("JOIN issue_assignment ON issue_assignment.assignee_user_name = account.user_name").
		Joins("JOIN issue ON issue.user_name = issue_assignment.user_name AND issue.repository_name = issue_assignment.repository_name AND issue.issue_id = issue_assignment.issue_id").
		Where("issue_assignment.user_name = ? AND issue_assignment.repository_name = ? AND issue.merge_request = ?",
			owner, repo, mergeRequest).
		Order("account.user_name ASC").
		Find(&accounts).Error
	return accounts, err
}

// DeleteIssue deletes an issue and all related data
func (s *IssueService) DeleteIssue(owner, repo string, issueID int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Delete issue labels
		if err := tx.Where("user_name = ? AND repository_name = ? AND issue_id = ?",
			owner, repo, issueID).
			Delete(&model.IssueLabel{}).Error; err != nil {
			return err
		}

		// Delete issue assignments
		if err := tx.Where("user_name = ? AND repository_name = ? AND issue_id = ?",
			owner, repo, issueID).
			Delete(&model.IssueAssignment{}).Error; err != nil {
			return err
		}

		// Delete issue comments
		if err := tx.Where("user_name = ? AND repository_name = ? AND issue_id = ?",
			owner, repo, issueID).
			Delete(&model.IssueComment{}).Error; err != nil {
			return err
		}

		// Delete issue
		if err := tx.Where("user_name = ? AND repository_name = ? AND issue_id = ?",
			owner, repo, issueID).
			Delete(&model.Issue{}).Error; err != nil {
			return err
		}

		return nil
	})
}

// DeleteComment deletes an issue comment
func (s *IssueService) DeleteComment(owner, repo string, commentID int) error {
	return s.db.Where("user_name = ? AND repository_name = ? AND comment_id = ?",
		owner, repo, commentID).
		Delete(&model.IssueComment{}).Error
}

// GetComment retrieves a comment by ID
func (s *IssueService) GetComment(owner, repo string, commentID int) (*model.IssueComment, error) {
	var comment model.IssueComment
	err := s.db.Where("user_name = ? AND repository_name = ? AND comment_id = ?",
		owner, repo, commentID).
		First(&comment).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

// UpdateComment updates an issue comment
func (s *IssueService) UpdateComment(comment *model.IssueComment) error {
	return s.db.Save(comment).Error
}
