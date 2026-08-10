package service

import (
	"testing"
	"time"
)

func TestSearchService_SearchIssues_Basic(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	issueService := NewIssueService(db)
	searchService := NewSearchService(db)

	// Create test issues
	for i := 1; i <= 3; i++ {
		_, err := issueService.CreateIssue("testuser", "test-repo", "testuser", "Test Issue", nil, nil, nil, false)
		if err != nil {
			t.Fatalf("CreateIssue %d failed: %v", i, err)
		}
	}

	// Search for issues
	query := &SearchQuery{
		Owner: "testuser",
		Repo:  "test-repo",
		Type:  "issue",
		Limit: 10,
	}
	result, err := searchService.SearchIssues(query)
	if err != nil {
		t.Fatalf("SearchIssues failed: %v", err)
	}
	if result.Total != 3 {
		t.Errorf("Expected 3 issues, got %d", result.Total)
	}
	if len(result.Issues) != 3 {
		t.Errorf("Expected 3 issues in result, got %d", len(result.Issues))
	}
}

func TestSearchService_SearchIssues_StateFilter(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	issueService := NewIssueService(db)
	searchService := NewSearchService(db)

	// Create open issue
	_, err := issueService.CreateIssue("testuser", "test-repo", "testuser", "Open Issue", nil, nil, nil, false)
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}

	// Create closed issue
	issue2, err := issueService.CreateIssue("testuser", "test-repo", "testuser", "Closed Issue", nil, nil, nil, false)
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}
	issue2.Closed = true
	if err := issueService.UpdateIssue(issue2); err != nil {
		t.Fatalf("UpdateIssue failed: %v", err)
	}

	// Search for open issues
	query := &SearchQuery{
		Owner: "testuser",
		Repo:  "test-repo",
		Type:  "issue",
		State: "open",
		Limit: 10,
	}
	result, err := searchService.SearchIssues(query)
	if err != nil {
		t.Fatalf("SearchIssues (open) failed: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("Expected 1 open issue, got %d", result.Total)
	}

	// Search for closed issues
	query.State = "closed"
	result, err = searchService.SearchIssues(query)
	if err != nil {
		t.Fatalf("SearchIssues (closed) failed: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("Expected 1 closed issue, got %d", result.Total)
	}
}

func TestSearchService_SearchIssues_AuthorFilter(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	issueService := NewIssueService(db)
	searchService := NewSearchService(db)

	// Create issues by different authors
	_, err := issueService.CreateIssue("testuser", "test-repo", "author1", "Issue 1", nil, nil, nil, false)
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}
	_, err = issueService.CreateIssue("testuser", "test-repo", "author2", "Issue 2", nil, nil, nil, false)
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}
	_, err = issueService.CreateIssue("testuser", "test-repo", "author1", "Issue 3", nil, nil, nil, false)
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}

	// Search by author
	query := &SearchQuery{
		Owner:  "testuser",
		Repo:   "test-repo",
		Type:   "issue",
		Author: "author1",
		Limit:  10,
	}
	result, err := searchService.SearchIssues(query)
	if err != nil {
		t.Fatalf("SearchIssues failed: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("Expected 2 issues by author1, got %d", result.Total)
	}
}

func TestSearchService_SearchIssues_KeywordFilter(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	issueService := NewIssueService(db)
	searchService := NewSearchService(db)

	// Create issues with different titles
	content1 := "This is about bugs"
	content2 := "This is about features"
	_, err := issueService.CreateIssue("testuser", "test-repo", "testuser", "Bug Report", &content1, nil, nil, false)
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}
	_, err = issueService.CreateIssue("testuser", "test-repo", "testuser", "Feature Request", &content2, nil, nil, false)
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}

	// Search by keyword in title
	query := &SearchQuery{
		Owner:   "testuser",
		Repo:    "test-repo",
		Type:    "issue",
		Keyword: "Bug",
		Limit:   10,
	}
	result, err := searchService.SearchIssues(query)
	if err != nil {
		t.Fatalf("SearchIssues failed: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("Expected 1 issue matching 'Bug', got %d", result.Total)
	}

	// Search by keyword in content
	query.Keyword = "features"
	result, err = searchService.SearchIssues(query)
	if err != nil {
		t.Fatalf("SearchIssues failed: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("Expected 1 issue matching 'features', got %d", result.Total)
	}
}

func TestSearchService_SearchIssues_Pagination(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	issueService := NewIssueService(db)
	searchService := NewSearchService(db)

	// Create 10 issues
	for i := 1; i <= 10; i++ {
		_, err := issueService.CreateIssue("testuser", "test-repo", "testuser", "Issue", nil, nil, nil, false)
		if err != nil {
			t.Fatalf("CreateIssue %d failed: %v", i, err)
		}
	}

	// Get first page
	query := &SearchQuery{
		Owner:  "testuser",
		Repo:   "test-repo",
		Type:   "issue",
		Limit:  3,
		Offset: 0,
	}
	result, err := searchService.SearchIssues(query)
	if err != nil {
		t.Fatalf("SearchIssues page 1 failed: %v", err)
	}
	if len(result.Issues) != 3 {
		t.Errorf("Expected 3 issues on page 1, got %d", len(result.Issues))
	}
	if result.Total != 10 {
		t.Errorf("Expected total 10, got %d", result.Total)
	}

	// Get second page
	query.Offset = 3
	result, err = searchService.SearchIssues(query)
	if err != nil {
		t.Fatalf("SearchIssues page 2 failed: %v", err)
	}
	if len(result.Issues) != 3 {
		t.Errorf("Expected 3 issues on page 2, got %d", len(result.Issues))
	}

	// Get last page
	query.Offset = 9
	result, err = searchService.SearchIssues(query)
	if err != nil {
		t.Fatalf("SearchIssues last page failed: %v", err)
	}
	if len(result.Issues) != 1 {
		t.Errorf("Expected 1 issue on last page, got %d", len(result.Issues))
	}
}

func TestSearchService_SearchIssues_Sort(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	issueService := NewIssueService(db)
	searchService := NewSearchService(db)

	// Create issues with time gaps to ensure distinct timestamps
	_, err := issueService.CreateIssue("testuser", "test-repo", "testuser", "Issue 1", nil, nil, nil, false)
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	_, err = issueService.CreateIssue("testuser", "test-repo", "testuser", "Issue 2", nil, nil, nil, false)
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	_, err = issueService.CreateIssue("testuser", "test-repo", "testuser", "Issue 3", nil, nil, nil, false)
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}

	// Sort by created-asc
	query := &SearchQuery{
		Owner: "testuser",
		Repo:  "test-repo",
		Type:  "issue",
		Sort:  "created-asc",
		Limit: 10,
	}
	result, err := searchService.SearchIssues(query)
	if err != nil {
		t.Fatalf("SearchIssues failed: %v", err)
	}
	if len(result.Issues) != 3 {
		t.Errorf("Expected 3 issues, got %d", len(result.Issues))
	}
	// Verify order (oldest first)
	if result.Issues[0].Title != "Issue 1" {
		t.Errorf("Expected first issue 'Issue 1', got '%s'", result.Issues[0].Title)
	}
	if result.Issues[2].Title != "Issue 3" {
		t.Errorf("Expected last issue 'Issue 3', got '%s'", result.Issues[2].Title)
	}

	// Sort by created-desc
	query.Sort = "created-desc"
	result, err = searchService.SearchIssues(query)
	if err != nil {
		t.Fatalf("SearchIssues failed: %v", err)
	}
	// Verify order (newest first)
	if result.Issues[0].Title != "Issue 3" {
		t.Errorf("Expected first issue 'Issue 3', got '%s'", result.Issues[0].Title)
	}
	if result.Issues[2].Title != "Issue 1" {
		t.Errorf("Expected last issue 'Issue 1', got '%s'", result.Issues[2].Title)
	}
}

func TestSearchService_SearchIssues_TypeFilter(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	issueService := NewIssueService(db)
	searchService := NewSearchService(db)

	// Create issues and merge requests
	_, err := issueService.CreateIssue("testuser", "test-repo", "testuser", "Issue 1", nil, nil, nil, false)
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}
	_, err = issueService.CreateIssue("testuser", "test-repo", "testuser", "MR 1", nil, nil, nil, true)
	if err != nil {
		t.Fatalf("CreateIssue (MR) failed: %v", err)
	}
	_, err = issueService.CreateIssue("testuser", "test-repo", "testuser", "Issue 2", nil, nil, nil, false)
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}

	// Search for issues only
	query := &SearchQuery{
		Owner: "testuser",
		Repo:  "test-repo",
		Type:  "issue",
		Limit: 10,
	}
	result, err := searchService.SearchIssues(query)
	if err != nil {
		t.Fatalf("SearchIssues (issue) failed: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("Expected 2 issues, got %d", result.Total)
	}

	// Search for merge requests only
	query.Type = "mr"
	result, err = searchService.SearchIssues(query)
	if err != nil {
		t.Fatalf("SearchIssues (mr) failed: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("Expected 1 MR, got %d", result.Total)
	}
}

func TestSearchService_ParseSearchQuery_Basic(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSearchService(db)

	// Parse simple keyword
	query := service.ParseSearchQuery("bug report")
	if query.Keyword != "bug report" {
		t.Errorf("Expected Keyword 'bug report', got '%s'", query.Keyword)
	}
}

func TestSearchService_ParseSearchQuery_TypeFilter(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSearchService(db)

	// Parse is:issue
	query := service.ParseSearchQuery("is:issue bug")
	if query.Type != "issue" {
		t.Errorf("Expected Type 'issue', got '%s'", query.Type)
	}
	if query.Keyword != "bug" {
		t.Errorf("Expected Keyword 'bug', got '%s'", query.Keyword)
	}

	// Parse is:mr
	query = service.ParseSearchQuery("is:mr feature")
	if query.Type != "mr" {
		t.Errorf("Expected Type 'mr', got '%s'", query.Type)
	}
	if query.Keyword != "feature" {
		t.Errorf("Expected Keyword 'feature', got '%s'", query.Keyword)
	}
}

func TestSearchService_ParseSearchQuery_StateFilter(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSearchService(db)

	// Parse is:open
	query := service.ParseSearchQuery("is:open bug")
	if query.State != "open" {
		t.Errorf("Expected State 'open', got '%s'", query.State)
	}
	if query.Keyword != "bug" {
		t.Errorf("Expected Keyword 'bug', got '%s'", query.Keyword)
	}

	// Parse is:closed
	query = service.ParseSearchQuery("is:closed feature")
	if query.State != "closed" {
		t.Errorf("Expected State 'closed', got '%s'", query.State)
	}
}

func TestSearchService_ParseSearchQuery_AuthorFilter(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSearchService(db)

	// Parse author:xxx
	query := service.ParseSearchQuery("author:john bug")
	if query.Author != "john" {
		t.Errorf("Expected Author 'john', got '%s'", query.Author)
	}
	if query.Keyword != "bug" {
		t.Errorf("Expected Keyword 'bug', got '%s'", query.Keyword)
	}
}

func TestSearchService_ParseSearchQuery_AssigneeFilter(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSearchService(db)

	// Parse assignee:xxx
	query := service.ParseSearchQuery("assignee:jane feature")
	if query.Assignee != "jane" {
		t.Errorf("Expected Assignee 'jane', got '%s'", query.Assignee)
	}
	if query.Keyword != "feature" {
		t.Errorf("Expected Keyword 'feature', got '%s'", query.Keyword)
	}
}

func TestSearchService_ParseSearchQuery_LabelFilter(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSearchService(db)

	// Parse label:xxx
	query := service.ParseSearchQuery("label:bug report")
	if query.Label != "bug" {
		t.Errorf("Expected Label 'bug', got '%s'", query.Label)
	}
	if query.Keyword != "report" {
		t.Errorf("Expected Keyword 'report', got '%s'", query.Keyword)
	}
}

func TestSearchService_ParseSearchQuery_MilestoneFilter(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSearchService(db)

	// Parse milestone:xxx
	query := service.ParseSearchQuery("milestone:v1.0 feature")
	if query.Milestone != "v1.0" {
		t.Errorf("Expected Milestone 'v1.0', got '%s'", query.Milestone)
	}
	if query.Keyword != "feature" {
		t.Errorf("Expected Keyword 'feature', got '%s'", query.Keyword)
	}
}

func TestSearchService_ParseSearchQuery_SortFilter(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSearchService(db)

	// Parse sort:created-asc
	query := service.ParseSearchQuery("sort:created-asc bug")
	if query.Sort != "created-asc" {
		t.Errorf("Expected Sort 'created-asc', got '%s'", query.Sort)
	}
	if query.Keyword != "bug" {
		t.Errorf("Expected Keyword 'bug', got '%s'", query.Keyword)
	}

	// Parse sort:updated-desc
	query = service.ParseSearchQuery("sort:updated-desc feature")
	if query.Sort != "updated-desc" {
		t.Errorf("Expected Sort 'updated-desc', got '%s'", query.Sort)
	}
}

func TestSearchService_ParseSearchQuery_Combined(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSearchService(db)

	// Parse complex query
	query := service.ParseSearchQuery("is:issue is:open author:john label:bug sort:created-desc bug report")
	if query.Type != "issue" {
		t.Errorf("Expected Type 'issue', got '%s'", query.Type)
	}
	if query.State != "open" {
		t.Errorf("Expected State 'open', got '%s'", query.State)
	}
	if query.Author != "john" {
		t.Errorf("Expected Author 'john', got '%s'", query.Author)
	}
	if query.Label != "bug" {
		t.Errorf("Expected Label 'bug', got '%s'", query.Label)
	}
	if query.Sort != "created-desc" {
		t.Errorf("Expected Sort 'created-desc', got '%s'", query.Sort)
	}
	if query.Keyword != "bug report" {
		t.Errorf("Expected Keyword 'bug report', got '%s'", query.Keyword)
	}
}

func TestSearchService_ParseSearchQuery_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)
	service := NewSearchService(db)

	// Parse empty query
	query := service.ParseSearchQuery("")
	if query.Keyword != "" {
		t.Errorf("Expected empty Keyword, got '%s'", query.Keyword)
	}
	if query.Type != "" {
		t.Errorf("Expected empty Type, got '%s'", query.Type)
	}
}
