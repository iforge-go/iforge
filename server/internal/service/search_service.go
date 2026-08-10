package service

import (
	"log"
	"strings"
	"time"

	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

// SearchService handles search operations
type SearchService struct {
	db *gorm.DB
}

// NewSearchService creates a new SearchService
func NewSearchService(db *gorm.DB) *SearchService {
	return &SearchService{db: db}
}

// SearchQuery represents a search query
type SearchQuery struct {
	Keyword       string
	Owner         string
	Repo          string
	Type          string // "issue" or "mr"
	State         string // "open" or "closed"
	Author        string
	Assignee      string
	Label         string
	Milestone     string
	Sort          string // "created-asc", "created-desc", "updated-asc", "updated-desc"
	CreatedBefore *time.Time
	CreatedAfter  *time.Time
	UpdatedBefore *time.Time
	UpdatedAfter  *time.Time
	Limit         int
	Offset        int
}

// IssueSearchResult represents search results
type IssueSearchResult struct {
	Issues []IssueWithLabels
	Total  int
}

// SearchIssues performs advanced issue search.
// When Type is empty, defaults to "issue" (excludes merge requests).
func (s *SearchService) SearchIssues(query *SearchQuery) (*IssueSearchResult, error) {
	// Start building query
	tx := s.db.Model(&model.Issue{})

	// Repository filter
	if query.Owner != "" && query.Repo != "" {
		tx = tx.Where("issue.user_name = ? AND issue.repository_name = ?", query.Owner, query.Repo)
	}

	// Type filter (issue or mr). Default to issue when unspecified so the
	// issues list page doesn't accidentally return merge requests.
	typeFilter := query.Type
	if typeFilter == "" {
		typeFilter = "issue"
	}
	if typeFilter == "mr" {
		tx = tx.Where("issue.merge_request = ?", true)
	} else if typeFilter == "issue" {
		tx = tx.Where("issue.merge_request = ?", false)
	}

	// State filter
	if query.State != "" {
		if query.State == "open" {
			tx = tx.Where("issue.closed = ?", false)
		} else if query.State == "closed" {
			tx = tx.Where("issue.closed = ?", true)
		} else if query.State == "merged" {
			tx = tx.Where("issue.closed = ? AND EXISTS (SELECT 1 FROM merge_request mr WHERE mr.user_name = issue.user_name AND mr.repository_name = issue.repository_name AND mr.issue_id = issue.issue_id AND mr.merged_commit_ids IS NOT NULL AND mr.merged_commit_ids != '')", true)
		}
	}

	// Author filter
	if query.Author != "" {
		tx = tx.Where("issue.opened_user_name = ?", query.Author)
	}

	// Assignee filter (EXISTS subquery on issue_assignment table)
	if query.Assignee != "" {
		tx = tx.Where(`
			EXISTS (
				SELECT 1 FROM issue_assignment ia
				WHERE ia.user_name = issue.user_name
				AND ia.repository_name = issue.repository_name
				AND ia.issue_id = issue.issue_id
				AND ia.assignee_user_name = ?
			)
		`, query.Assignee)
	}

	// Label filter (using EXISTS subquery)
	if query.Label != "" {
		tx = tx.Where(`
			EXISTS (
				SELECT 1 FROM issue_label il
				JOIN label l ON il.label_id = l.label_id
				WHERE il.user_name = issue.user_name
				AND il.repository_name = issue.repository_name
				AND il.issue_id = issue.issue_id
				AND l.label_name = ?
			)
		`, query.Label)
	}

	// Milestone filter (using EXISTS subquery)
	if query.Milestone != "" {
		tx = tx.Where(`
			EXISTS (
				SELECT 1 FROM milestone m
				WHERE m.user_name = issue.user_name
				AND m.repository_name = issue.repository_name
				AND m.milestone_id = issue.milestone_id
				AND m.title = ?
			)
		`, query.Milestone)
	}

	// Date filters
	if query.CreatedBefore != nil {
		tx = tx.Where("issue.registered_date <= ?", *query.CreatedBefore)
	}
	if query.CreatedAfter != nil {
		tx = tx.Where("issue.registered_date >= ?", *query.CreatedAfter)
	}
	if query.UpdatedBefore != nil {
		tx = tx.Where("issue.updated_date <= ?", *query.UpdatedBefore)
	}
	if query.UpdatedAfter != nil {
		tx = tx.Where("issue.updated_date >= ?", *query.UpdatedAfter)
	}

	// Keyword search via LIKE. Min length 2 enforced (leading wildcard forces full scan).
	// LOWER() ensures cross-DB case-insensitivity (PostgreSQL LIKE is case-sensitive by default).
	if len(query.Keyword) >= 2 {
		keyword := "%" + strings.ToLower(query.Keyword) + "%"
		tx = tx.Where("(LOWER(issue.title) LIKE ? OR LOWER(issue.content) LIKE ?)", keyword, keyword)
	}

	// Get total count
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, err
	}

	// Get issues with pagination
	limit := query.Limit
	if limit == 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}
	offset := query.Offset
	if offset < 0 {
		offset = 0
	}

	var issues []*model.Issue
	// Apply sort. Default to updated_date DESC (most recently updated first).
	orderClause := "issue.updated_date DESC"
	switch query.Sort {
	case "created-asc":
		orderClause = "issue.registered_date ASC"
	case "created-desc":
		orderClause = "issue.registered_date DESC"
	case "updated-asc":
		orderClause = "issue.updated_date ASC"
	case "updated-desc":
		orderClause = "issue.updated_date DESC"
	}
	if err := tx.Order(orderClause).Limit(limit).Offset(offset).Find(&issues).Error; err != nil {
		return nil, err
	}

	// Batch query comment counts and participant usernames for each issue (avoid N+1 queries)
	// Only count action='comment' records, excluding system events like close/reopen/label
	type commentCountRow struct {
		IssueID int `gorm:"column:issue_id"`
		Cnt     int `gorm:"column:cnt"`
	}
	type participantRow struct {
		IssueID           int    `gorm:"column:issue_id"`
		CommentedUserName string `gorm:"column:commented_user_name"`
	}

	commentCountMap := make(map[int]int)
	participantMap := make(map[int][]string) // Maintain insertion order

	if len(issues) > 0 {
		issueIDs := make([]int, len(issues))
		for i, issue := range issues {
			issueIDs[i] = issue.IssueID
		}

		// Comment count: group by issue_id
		// Subquery failure should not block main search results: log and fill with 0
		var countRows []commentCountRow
		if err := s.db.Model(&model.IssueComment{}).
			Select("issue_id, COUNT(*) as cnt").
			Where("user_name = ? AND repository_name = ? AND issue_id IN ? AND action = ?",
				query.Owner, query.Repo, issueIDs, "comment").
			Group("issue_id").
			Find(&countRows).Error; err != nil {
			log.Printf("warn: SearchIssues comment count failed: %v", err)
		}
		for _, r := range countRows {
			commentCountMap[r.IssueID] = r.Cnt
		}

		// Participants: deduplicate by (issue_id, commented_user_name), order by first comment time
		var partRows []participantRow
		if err := s.db.Model(&model.IssueComment{}).
			Select("issue_id, commented_user_name").
			Where("user_name = ? AND repository_name = ? AND issue_id IN ?",
				query.Owner, query.Repo, issueIDs).
			Group("issue_id, commented_user_name").
			Order("issue_id, MIN(registered_date)").
			Find(&partRows).Error; err != nil {
			log.Printf("warn: SearchIssues participants failed: %v", err)
		}
		for _, r := range partRows {
			// Defensive deduplication
			exists := false
			for _, name := range participantMap[r.IssueID] {
				if name == r.CommentedUserName {
					exists = true
					break
				}
			}
			if !exists {
				participantMap[r.IssueID] = append(participantMap[r.IssueID], r.CommentedUserName)
			}
		}
	}

	// Batch query labels for all issues (avoid N+1, 2 queries instead of 2N)
	issueLabelMap := make(map[int][]int) // issueId -> []labelId
	labelIDSet := make(map[int]bool)
	if len(issues) > 0 {
		issueIDs := make([]int, len(issues))
		for i, issue := range issues {
			issueIDs[i] = issue.IssueID
		}
		var allIssueLabels []model.IssueLabel
		if err := s.db.Where("user_name = ? AND repository_name = ? AND issue_id IN ?",
			query.Owner, query.Repo, issueIDs).Find(&allIssueLabels).Error; err != nil {
			log.Printf("warn: SearchIssues issue labels failed: %v", err)
		}
		for _, il := range allIssueLabels {
			issueLabelMap[il.IssueID] = append(issueLabelMap[il.IssueID], il.LabelID)
			labelIDSet[il.LabelID] = true
		}
	}
	// Batch query all used labels
	labelMap := make(map[int]model.Label)
	if len(labelIDSet) > 0 {
		allLabelIDs := make([]int, 0, len(labelIDSet))
		for id := range labelIDSet {
			allLabelIDs = append(allLabelIDs, id)
		}
		var allLabels []model.Label
		if err := s.db.Where("user_name = ? AND repository_name = ? AND label_id IN ?",
			query.Owner, query.Repo, allLabelIDs).Find(&allLabels).Error; err != nil {
			log.Printf("warn: SearchIssues labels failed: %v", err)
		}
		for _, l := range allLabels {
			labelMap[l.LabelID] = l
		}
	}

	// Preload labels for each issue (backfill from batch query results)
	result := make([]IssueWithLabels, len(issues))
	for i, issue := range issues {
		labels := make([]model.Label, 0)
		for _, lid := range issueLabelMap[issue.IssueID] {
			if l, ok := labelMap[lid]; ok {
				labels = append(labels, l)
			}
		}
		result[i] = IssueWithLabels{
			Issue:                *issue,
			Labels:               labels,
			CommentsCount:        commentCountMap[issue.IssueID],
			ParticipantUserNames: buildIssueParticipants(issue.OpenedUserName, participantMap[issue.IssueID]),
		}
	}

	return &IssueSearchResult{
		Issues: result,
		Total:  int(total),
	}, nil
}

// buildIssueParticipants builds participant username list for a single issue, creator first
func buildIssueParticipants(openedUserName string, commenters []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, 1+len(commenters))
	// Creator first
	if openedUserName != "" {
		seen[openedUserName] = true
		result = append(result, openedUserName)
	}
	// Append commenters (deduplicated)
	for _, name := range commenters {
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		result = append(result, name)
	}
	return result
}

// ParseSearchQuery parses advanced search syntax
// Supports: is:issue, is:mr, is:open, is:closed, author:xxx, assignee:xxx,
// label:xxx, milestone:xxx, sort:created-asc|created-desc|updated-asc|updated-desc
func (s *SearchService) ParseSearchQuery(searchText string) *SearchQuery {
	query := &SearchQuery{
		Keyword: "",
	}

	parts := strings.Fields(searchText)
	remainingParts := []string{}

	for _, part := range parts {
		if strings.Contains(part, ":") {
			kv := strings.SplitN(part, ":", 2)
			if len(kv) == 2 {
				key := strings.ToLower(kv[0])
				value := kv[1]

				switch key {
				case "is":
					if value == "issue" || value == "mr" {
						query.Type = value
					} else if value == "open" || value == "closed" || value == "merged" {
						query.State = value
					}
				case "author":
					query.Author = value
				case "assignee":
					query.Assignee = value
				case "label":
					query.Label = value
				case "milestone":
					query.Milestone = value
				case "sort":
					query.Sort = strings.ToLower(value)
				default:
					remainingParts = append(remainingParts, part)
				}
			} else {
				remainingParts = append(remainingParts, part)
			}
		} else {
			remainingParts = append(remainingParts, part)
		}
	}

	// Remaining text is the keyword
	query.Keyword = strings.Join(remainingParts, " ")

	return query
}
