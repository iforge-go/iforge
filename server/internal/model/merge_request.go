package model

import "time"

// MergeRequest represents a merge request
type MergeRequest struct {
	// (user_name, repository_name) is the composite PK prefix; ListMergeRequests filters
	// by these two fields and JOINs issue - the PK index is sufficient.
	// request_user_name index supports reverse lookup of MRs by requester.
	UserName              string  `gorm:"primaryKey;column:user_name" json:"userName"`
	RepositoryName        string  `gorm:"primaryKey;column:repository_name" json:"repositoryName"`
	IssueID               int     `gorm:"primaryKey;column:issue_id" json:"issueId"`
	Branch                string  `gorm:"column:branch" json:"branch"`
	RequestUserName       string  `gorm:"column:request_user_name;index:idx_mr_request_user" json:"requestUserName"`
	RequestRepositoryName string  `gorm:"column:request_repository_name" json:"requestRepositoryName"`
	RequestBranch         string  `gorm:"column:request_branch" json:"requestBranch"`
	CommitIDFrom          string  `gorm:"column:commit_id_from" json:"commitIdFrom"`
	CommitIDTo            string  `gorm:"column:commit_id_to" json:"commitIdTo"`
	IsDraft               bool    `gorm:"column:is_draft" json:"isDraft"`
	MergedCommitIDs       *string `gorm:"column:merged_commit_ids" json:"mergedCommitIds"`
	MergedCommits         *string `gorm:"column:merged_commits" json:"mergedCommits"`
	MergedFileChanges     *string `gorm:"column:merged_file_changes" json:"mergedFileChanges"`
}

func (MergeRequest) TableName() string { return "merge_request" }

// MergeRequestListItem is the enriched MR data returned in list views
type MergeRequestListItem struct {
	IssueID         int       `json:"issueId"`
	Title           string    `json:"title"`
	State           string    `json:"state"`
	UserName        string    `json:"userName"`
	CreatedAt       time.Time `json:"createdAt"`
	Branch          string    `json:"branch"`
	RequestBranch   string    `json:"requestBranch"`
	RequestUserName string    `json:"requestUserName"`
	IsDraft         bool      `json:"isDraft"`
	Merged          bool      `json:"merged"`
	CommitIDFrom    string    `json:"commitIdFrom"`
}

// MergeRequestDetail is the enriched MR data for detail views
type MergeRequestDetail struct {
	IssueID               int       `json:"issueId"`
	Title                 string    `json:"title"`
	Content               *string   `json:"content"`
	State                 string    `json:"state"`
	UserName              string    `json:"userName"`
	RepositoryName        string    `json:"repositoryName"`
	CreatedAt             time.Time `json:"createdAt"`
	UpdatedAt             time.Time `json:"updatedAt"`
	Branch                string    `json:"branch"`
	RequestBranch         string    `json:"requestBranch"`
	RequestUserName       string    `json:"requestUserName"`
	RequestRepositoryName string    `json:"requestRepositoryName"`
	IsDraft               bool      `json:"isDraft"`
	CommitIDFrom          string    `json:"commitIdFrom"`
	CommitIDTo            string    `json:"commitIdTo"`
	MergedCommitIDs       *string   `json:"mergedCommitIds"`
	MergedCommits         *string   `json:"mergedCommits"`
	MergedFileChanges     *string   `json:"mergedFileChanges"`
	Merged                bool      `json:"merged"`
}

// MRBranch carries the branch-related fields shared by MergeRequest and
// MergeRequestDetail. It lets subscribers process MR events uniformly
// (via the MRBranch method) without type-switching on the concrete MR type.
type MRBranch struct {
	RequestUserName       string
	RequestRepositoryName string
	RequestBranch         string
	UserName              string
	RepositoryName        string
	Branch                string
	IsDraft               bool
}

// MRBranch returns the branch-related fields of the merge request.
func (m *MergeRequest) MRBranch() MRBranch {
	return MRBranch{
		RequestUserName:       m.RequestUserName,
		RequestRepositoryName: m.RequestRepositoryName,
		RequestBranch:         m.RequestBranch,
		UserName:              m.UserName,
		RepositoryName:        m.RepositoryName,
		Branch:                m.Branch,
		IsDraft:               m.IsDraft,
	}
}

// MRBranch returns the branch-related fields of the merge request detail.
func (m *MergeRequestDetail) MRBranch() MRBranch {
	return MRBranch{
		RequestUserName:       m.RequestUserName,
		RequestRepositoryName: m.RequestRepositoryName,
		RequestBranch:         m.RequestBranch,
		UserName:              m.UserName,
		RepositoryName:        m.RepositoryName,
		Branch:                m.Branch,
		IsDraft:               m.IsDraft,
	}
}
