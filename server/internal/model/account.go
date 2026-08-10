package model

import "time"

// Account represents a user or organization account
type Account struct {
	UserName string `gorm:"primaryKey;column:user_name;not null;size:100" json:"userName"`
	FullName string `gorm:"column:full_name;size:200" json:"fullName"`
	// Index on mail_address covers GetAccountByEmail (login lookup)
	MailAddress    string     `gorm:"column:mail_address;index:idx_account_email;not null;size:255" json:"mailAddress"`
	Password       string     `gorm:"column:password;not null;size:255" json:"-"`
	IsAdmin        bool       `gorm:"column:administrator;not null;default:false" json:"isAdmin"`
	URL            *string    `gorm:"column:url;size:500" json:"url"`
	RegisteredDate time.Time  `gorm:"column:registered_date;not null" json:"registeredDate"`
	UpdatedDate    time.Time  `gorm:"column:updated_date;not null" json:"updatedDate"`
	LastLoginDate  *time.Time `gorm:"column:last_login_date" json:"lastLoginDate"`
	Image          *string    `gorm:"column:image;size:500" json:"image"`
	// Composite index (is_organization, removed) covers filtering for GetAllUsers / ListOrganizations / SearchUsers
	IsOrganization bool    `gorm:"column:is_organization;index:idx_account_org_removed,priority:1;not null;default:false" json:"isOrganization"`
	IsRemoved      bool    `gorm:"column:removed;index:idx_account_org_removed,priority:2;not null;default:false" json:"isRemoved"`
	Description    *string `gorm:"column:description;size:1000" json:"description"`
}

func (Account) TableName() string { return "account" }

// AccountExtraMailAddress represents an extra mail address for an account
type AccountExtraMailAddress struct {
	UserName    string `gorm:"primaryKey;column:user_name" json:"userName"`
	MailAddress string `gorm:"primaryKey;column:mail_address" json:"mailAddress"`
}

func (AccountExtraMailAddress) TableName() string { return "account_extra_mail_address" }

// AccountPreference represents user preferences
type AccountPreference struct {
	UserName         string `gorm:"primaryKey;column:user_name" json:"userName"`
	HighlighterTheme string `gorm:"column:highlighter_theme" json:"highlighterTheme"`
	Notification     bool   `gorm:"column:notification" json:"notification"`
	Timezone         string `gorm:"column:timezone" json:"timezone"`
}

func (AccountPreference) TableName() string { return "account_preference" }

// OrganizationMember represents an organization member
type OrganizationMember struct {
	OrganizationName string `gorm:"primaryKey;column:organization_name" json:"organizationName"`
	// Single-column index on user_name supports reverse lookup of user's organizations (used by GetVisibleRepositories subquery)
	UserName   string    `gorm:"primaryKey;column:user_name;index:idx_orgmember_user" json:"userName"`
	IsManager  bool      `gorm:"column:is_manager" json:"isManager"`
	JoinedDate time.Time `gorm:"column:joined_date" json:"joinedDate"`
}

func (OrganizationMember) TableName() string { return "organization_member" }

// OrganizationMemberDetail represents an organization member with account info
type OrganizationMemberDetail struct {
	OrganizationName string    `gorm:"column:organization_name" json:"organizationName"`
	UserName         string    `gorm:"column:user_name" json:"userName"`
	IsManager        bool      `gorm:"column:is_manager" json:"isManager"`
	JoinedDate       time.Time `gorm:"column:joined_date" json:"joinedDate"`
	FullName         string    `gorm:"column:full_name" json:"fullName"`
	Image            *string   `gorm:"column:image" json:"image"`
	IsAdmin          bool      `gorm:"column:administrator" json:"isAdmin"`
}

// SSHKey represents an SSH key
type SSHKey struct {
	UserName       string    `gorm:"column:user_name" json:"userName"`
	SSHKeyID       int       `gorm:"primaryKey;autoIncrement;column:ssh_key_id" json:"sshKeyId"`
	Title          string    `gorm:"column:title" json:"title"`
	PublicKey      string    `gorm:"column:public_key" json:"publicKey"`
	RegisteredDate time.Time `gorm:"column:registered_date" json:"registeredDate"`
}

func (SSHKey) TableName() string { return "ssh_key" }

// GPGKey represents a GPG key
type GPGKey struct {
	UserName       string    `gorm:"primaryKey;column:user_name" json:"userName"`
	KeyID          int64     `gorm:"primaryKey;column:key_id" json:"keyId"`
	GpgKeyID       string    `gorm:"column:gpg_key_id" json:"gpgKeyId"`
	Title          string    `gorm:"column:title" json:"title"`
	PublicKey      string    `gorm:"column:public_key" json:"publicKey"`
	RegisteredDate time.Time `gorm:"column:registered_date" json:"registeredDate"`
}

func (GPGKey) TableName() string { return "gpg_key" }
