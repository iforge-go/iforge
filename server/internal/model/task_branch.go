package model

import "time"

// TaskBranch represents the loose coupling between a task and a Git branch.
//
// Design notes (decoupled from VCS):
//   - Does not reference VCS's branches table (iforge's VCS has no branches table; branches are plain strings).
//   - repo_full_name uses the "owner/repo" format (e.g. "guanshiliang/iforge"), an industry-standard
//     format that naturally supports fork workflows: PMS only records the branch's physical location,
//     without distinguishing upstream from fork.
//   - user_name records the user who created the association (iforge uses user_name site-wide, not user_id).
//   - Composite unique index (project_id, task_id, repo_full_name, branch_name) prevents duplicate
//     associations while allowing the same branch to be linked to multiple tasks (many-to-many)
//     and the same task to link to multiple branches.
type TaskBranch struct {
	ID           uint   `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	ProjectID    int    `gorm:"column:project_id;uniqueIndex:idx_task_branch" json:"-"`
	ProjectSlug  string `gorm:"-" json:"projectSlug"`
	TaskID       int    `gorm:"column:task_id;uniqueIndex:idx_task_branch" json:"taskId"`
	RepoFullName string `gorm:"column:repo_full_name;type:varchar(255);uniqueIndex:idx_task_branch" json:"repoFullName"`
	BranchName   string `gorm:"column:branch_name;type:varchar(255);uniqueIndex:idx_task_branch" json:"branchName"`
	// BaseSHA records the base commit SHA at branch creation (i.e. HEAD of fromBranch when created).
	// Used by GetTaskCommits to correctly list branch-unique commits even after MR merge:
	// after merge the merge base catches up to the branch HEAD and ListCommitsBetween returns empty,
	// but BaseSHA is frozen at creation time and unaffected by subsequent merges.
	// Older records without this field fall back to merge base / CreatedAt timestamp filtering.
	BaseSHA   string    `gorm:"column:base_sha" json:"baseSha"`
	UserName  string    `gorm:"column:user_name" json:"userName"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
}

func (TaskBranch) TableName() string { return "task_branch" }
