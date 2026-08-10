package model

import "time"

// RepositoryOptions contains repository configuration options
type RepositoryOptions struct {
	IssuesOption       string  `json:"issuesOption"`
	ExternalIssuesURL  *string `json:"externalIssuesUrl"`
	WikiOption         string  `json:"wikiOption"`
	ExternalWikiURL    *string `json:"externalWikiUrl"`
	AllowFork          bool    `json:"allowFork"`
	AllowMerge         bool    `json:"allowMerge"`
	AllowRebase        bool    `json:"allowRebase"`
	AllowRebaseMerge   bool    `json:"allowRebaseMerge"`
	AllowSquash        bool    `json:"allowSquash"`
	MergeOptions       string  `json:"mergeOptions"`
	DefaultMergeOption string  `json:"defaultMergeOption"`
	SafeMode           bool    `json:"safeMode"`
}

// Repository represents a git repository
type Repository struct {
	UserName       string `gorm:"primaryKey;column:user_name;not null;size:100" json:"userName"`
	RepositoryName string `gorm:"primaryKey;column:repository_name;not null;size:100" json:"repositoryName"`
	// Index on private covers public-repo filtering in GetVisibleRepositories
	IsPrivate      bool      `gorm:"column:private;index:idx_repo_private;not null;default:false" json:"isPrivate"`
	Description    *string   `gorm:"column:description;size:500" json:"description"`
	DefaultBranch  string    `gorm:"column:default_branch;not null;size:100;default:main" json:"defaultBranch"`
	RegisteredDate time.Time `gorm:"column:registered_date;index:idx_repo_registered;not null" json:"registeredDate"`
	UpdatedDate    time.Time `gorm:"column:updated_date;not null" json:"updatedDate"`
	// Index on last_activity_date covers ORDER BY last_activity_date DESC
	LastActivityDate     time.Time `gorm:"column:last_activity_date;index:idx_repo_activity;not null" json:"lastActivityDate"`
	OriginUserName       *string   `gorm:"column:origin_user_name;size:100" json:"originUserName"`
	OriginRepositoryName *string   `gorm:"column:origin_repository_name;size:100" json:"originRepositoryName"`
	// Composite index (parent_user_name, parent_repository_name) covers GetForks / GetForkCount
	ParentUserName       *string `gorm:"column:parent_user_name;index:idx_repo_parent,priority:1;size:100" json:"parentUserName"`
	ParentRepositoryName *string `gorm:"column:parent_repository_name;index:idx_repo_parent,priority:2;size:100" json:"parentRepositoryName"`
	IsArchived           bool    `gorm:"column:is_archived;not null;default:false" json:"isArchived"`
	IsTemplate           bool    `gorm:"column:is_template;not null;default:false" json:"isTemplate"`
	// IsDeleted is a soft-delete marker
	IsDeleted          bool              `gorm:"column:is_deleted;index:idx_repo_deleted;not null;default:false" json:"isDeleted"`
	Options            RepositoryOptions `gorm:"-" json:"options"`
	IssuesOption       string            `gorm:"column:issues_option;size:20" json:"-"`
	ExternalIssuesURL  *string           `gorm:"column:external_issues_url;size:500" json:"-"`
	WikiOption         string            `gorm:"column:wiki_option;size:20" json:"-"`
	ExternalWikiURL    *string           `gorm:"column:external_wiki_url;size:500" json:"-"`
	AllowFork          bool              `gorm:"column:allow_fork;not null;default:true" json:"-"`
	MergeOptions       string            `gorm:"column:merge_options;size:100" json:"-"`
	DefaultMergeOption string            `gorm:"column:default_merge_option;size:20" json:"-"`
	SafeMode           bool              `gorm:"column:safe_mode;not null;default:true" json:"-"`
	DefaultPriorityID  *int              `gorm:"column:default_priority_id" json:"-"`
}

func (Repository) TableName() string { return "repository" }

// RepositoryInfo is a composite type used in services
type RepositoryInfo struct {
	Repository  Repository
	IssueCount  int
	PullCount   int
	ForkedCount int
	IsForked    bool
}

// Collaborator represents a repository collaborator
type Collaborator struct {
	UserName       string `gorm:"primaryKey;column:user_name" json:"userName"`
	RepositoryName string `gorm:"primaryKey;column:repository_name" json:"repositoryName"`
	// Single-column index on collaborator_name covers EXISTS subquery (reverse lookup of collaborated repos by user)
	CollaboratorName string `gorm:"primaryKey;column:collaborator_name;index:idx_collab_name" json:"collaboratorName"`
	Role             string `gorm:"column:role" json:"role"`
}

func (Collaborator) TableName() string { return "collaborator" }

// CollaboratorWithUser extends Collaborator with account info for display
type CollaboratorWithUser struct {
	Collaborator
	FullName       string  `gorm:"column:full_name" json:"fullName"`
	AvatarURL      *string `gorm:"column:image" json:"avatarUrl"`
	Email          string  `gorm:"column:mail_address" json:"email"`
	IsOrganization bool    `gorm:"column:is_organization" json:"isOrganization"`
}

// RepositoryStar represents a repository star
type RepositoryStar struct {
	UserName       string    `gorm:"primaryKey;column:user_name" json:"userName"`
	RepositoryName string    `gorm:"primaryKey;column:repository_name" json:"repositoryName"`
	Starer         string    `gorm:"primaryKey;column:starrer" json:"starer"`
	RegisteredDate time.Time `gorm:"column:registered_date" json:"registeredDate"`
}

func (RepositoryStar) TableName() string { return "repository_star" }

// RepositoryWatch represents a repository watch
type RepositoryWatch struct {
	UserName       string `gorm:"primaryKey;column:user_name" json:"userName"`
	RepositoryName string `gorm:"primaryKey;column:repository_name" json:"repositoryName"`
	Watcher        string `gorm:"primaryKey;column:watcher" json:"watcher"`
	Notification   bool   `gorm:"column:notification" json:"notification"`
}

func (RepositoryWatch) TableName() string { return "repository_watch" }

// RepositoryMirror represents a repository mirror configuration
type RepositoryMirror struct {
	UserName       string    `gorm:"primaryKey;column:user_name" json:"userName"`
	RepositoryName string    `gorm:"primaryKey;column:repository_name" json:"repositoryName"`
	MirrorURL      string    `gorm:"column:mirror_url" json:"mirrorUrl"`
	SyncInterval   int       `gorm:"column:sync_interval" json:"syncInterval"`
	LastSyncDate   time.Time `gorm:"column:last_sync_date" json:"lastSyncDate"`
	NextSyncDate   time.Time `gorm:"column:next_sync_date" json:"nextSyncDate"`
	Enabled        bool      `gorm:"column:enabled" json:"enabled"`
	SyncOnPush     bool      `gorm:"column:sync_on_push" json:"syncOnPush"`
	Authentication string    `gorm:"column:authentication" json:"authentication"`
	Username       *string   `gorm:"column:username" json:"username,omitempty"`
	Password       *string   `gorm:"column:password" json:"password,omitempty"`
	SSHKey         *string   `gorm:"column:ssh_key" json:"sshKey,omitempty"`
}

func (RepositoryMirror) TableName() string { return "repository_mirror" }

// ProtectedBranch represents a protected branch configuration
type ProtectedBranch struct {
	UserName       string `gorm:"primaryKey;column:user_name" json:"userName"`
	RepositoryName string `gorm:"primaryKey;column:repository_name" json:"repositoryName"`
	Branch         string `gorm:"primaryKey;column:branch" json:"branch"`
	Status         string `gorm:"column:status" json:"status"`
}

func (ProtectedBranch) TableName() string { return "protected_branch" }

// DeployKey represents a deploy key
type DeployKey struct {
	// Composite index on (user_name, repository_name) covers per-repository deploy key queries
	UserName       string    `gorm:"column:user_name;index:idx_deploykey_repo,priority:1" json:"userName"`
	RepositoryName string    `gorm:"column:repository_name;index:idx_deploykey_repo,priority:2" json:"repositoryName"`
	DeployKeyID    int       `gorm:"primaryKey;autoIncrement;column:deploy_key_id" json:"deployKeyId"`
	Title          string    `gorm:"column:title" json:"title"`
	PublicKey      string    `gorm:"column:public_key" json:"publicKey"`
	AllowWrite     bool      `gorm:"column:allow_write" json:"allowWrite"`
	RegisteredDate time.Time `gorm:"column:registered_date" json:"registeredDate"`
}

func (DeployKey) TableName() string { return "deploy_key" }
