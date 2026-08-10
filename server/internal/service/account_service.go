package service

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"iforge/iforge/internal/model"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrInvalidPassword   = errors.New("invalid password")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrReservedName      = errors.New("username is reserved and cannot be used")
)

// reservedNames contains all words that cannot be used as usernames or organization names.
// These conflict with frontend route paths and would cause URL ambiguity.
var reservedNames = map[string]bool{
	// Frontend route reserved words (top-level dirs under web/src/app/)
	"admin":           true,
	"auth":            true,
	"dashboard":       true,
	"forgot-password": true,
	"login":           true,
	"new":             true,
	"notifications":   true,
	"organizations":   true,
	"pms":             true,
	"projects":        true,
	"register":        true,
	"repos":           true,
	"reset-password":  true,
	"search":          true,
	"settings":        true,
	"setup":           true,
	"vcs":             true,

	// Backend route reserved words
	"api":    true,
	"health": true,

	// Git/system reserved words
	"git":           true,
	"www":           true,
	"administrator": true,
	"system":        true,
	"support":       true,
	"help":          true,
	"about":         true,
	"info":          true,
	"security":      true,
}

// IsReservedName checks if a name is reserved (case-insensitive).
func IsReservedName(name string) bool {
	return reservedNames[strings.ToLower(strings.TrimSpace(name))]
}

// AccountService handles account-related operations
type AccountService struct {
	db          *gorm.DB
	ldapService *LDAPService
}

// NewAccountService creates a new AccountService
func NewAccountService(db *gorm.DB, ldapService *LDAPService) *AccountService {
	return &AccountService{
		db:          db,
		ldapService: ldapService,
	}
}

// Authenticate authenticates a user with username and password
// It first tries local authentication, then falls back to LDAP if enabled
func (s *AccountService) Authenticate(username, password string) (*model.Account, error) {
	// Try local authentication first
	account, err := s.GetAccountByUsername(username)
	if err == nil {
		if account.IsRemoved {
			return nil, ErrUserNotFound
		}

		if verifyPassword(account.Password, password) {
			s.UpdateLastLoginDate(username)
			return account, nil
		}
	}

	// If local auth failed and LDAP is enabled, try LDAP
	if s.ldapService != nil {
		ldapUser, ldapErr := s.ldapService.Authenticate(username, password)
		if ldapErr == nil && ldapUser != nil {
			// LDAP authentication succeeded
			// Check if user exists locally, if not create it
			account, err = s.GetAccountByUsername(username)
			if err != nil {
				// User doesn't exist locally, create it
				account, err = s.CreateAccount(
					username,
					password, // Store the password locally too
					ldapUser.FullName,
					ldapUser.Email,
					false, // LDAP users are not admins by default
					nil,
					nil,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create LDAP user locally: %w", err)
				}
			}

			// Update last login date
			s.UpdateLastLoginDate(username)
			return account, nil
		}
	}

	// Both local and LDAP authentication failed
	if err == gorm.ErrRecordNotFound {
		return nil, ErrUserNotFound
	}
	return nil, ErrInvalidPassword
}

// verifyPassword verifies a password against a hash.
// Supports bcrypt (new), PBKDF2 (legacy), and SHA-1 (legacy).
// When a legacy hash matches, it upgrades the stored hash to bcrypt.
func verifyPassword(hash, password string) bool {
	if strings.HasPrefix(hash, "$2a$") || strings.HasPrefix(hash, "$2b$") || strings.HasPrefix(hash, "$2y$") {
		return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
	}
	if strings.HasPrefix(hash, "$pbkdf2-sha256$") {
		parts := strings.Split(hash, "$")
		if len(parts) != 5 {
			return false
		}
		return hash == pbkdf2Hash(password, parts[2], parts[3])
	}
	return hash == sha1Hash(password)
}

func sha1Hash(password string) string {
	h := sha1.New()
	h.Write([]byte(password))
	return hex.EncodeToString(h.Sum(nil))
}

func pbkdf2Hash(password, salt, _ string) string {
	// Simplified PBKDF2 - in production use golang.org/x/crypto/pbkdf2
	h := sha1.New()
	h.Write([]byte(password + salt))
	return hex.EncodeToString(h.Sum(nil))
}

// hashPassword hashes a password using bcrypt.
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CreateAccount creates a new account
func (s *AccountService) CreateAccount(username, password, fullName, mailAddress string, isAdmin bool, url, description *string) (*model.Account, error) {
	if IsReservedName(username) {
		return nil, ErrReservedName
	}

	var count int64
	if err := s.db.Model(&model.Account{}).Where("user_name = ?", username).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrUserAlreadyExists
	}

	hashedPassword, err := hashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	now := time.Now()

	account := &model.Account{
		UserName:       username,
		FullName:       fullName,
		MailAddress:    mailAddress,
		Password:       hashedPassword,
		IsAdmin:        isAdmin,
		URL:            url,
		RegisteredDate: now,
		UpdatedDate:    now,
		Description:    description,
	}

	if err := s.db.Create(account).Error; err != nil {
		return nil, err
	}

	return account, nil
}

// GetAccountByUsername retrieves an account by username
func (s *AccountService) GetAccountByUsername(username string) (*model.Account, error) {
	account := &model.Account{}
	err := s.db.Where("user_name = ? AND removed = ?", username, false).First(account).Error
	if err != nil {
		return nil, err
	}
	return account, nil
}

// FilterExistingUserNames batch-validates username existence, returning usernames
// that exist (and are not soft-deleted) from the input. Preserves input order and deduplicates.
// Used for @mentions: 1 query replaces N calls to GetAccountByUsername, eliminating N+1.
func (s *AccountService) FilterExistingUserNames(userNames []string) ([]string, error) {
	if len(userNames) == 0 {
		return nil, nil
	}
	var existing []string
	err := s.db.Model(&model.Account{}).
		Where("user_name IN ? AND removed = ?", userNames, false).
		Pluck("user_name", &existing).Error
	if err != nil {
		return nil, err
	}
	existingSet := make(map[string]bool, len(existing))
	for _, n := range existing {
		existingSet[n] = true
	}
	// Preserve input order and deduplicate
	result := make([]string, 0, len(userNames))
	seen := make(map[string]bool, len(userNames))
	for _, n := range userNames {
		if existingSet[n] && !seen[n] {
			seen[n] = true
			result = append(result, n)
		}
	}
	return result, nil
}

// GetAccountByEmail retrieves an account by email
func (s *AccountService) GetAccountByEmail(email string) (*model.Account, error) {
	account := &model.Account{}
	err := s.db.Where("mail_address = ?", email).First(account).Error
	if err != nil {
		return nil, err
	}
	return account, nil
}

// UpdateLastLoginDate updates the last login date for a user
func (s *AccountService) UpdateLastLoginDate(username string) error {
	return s.db.Model(&model.Account{}).
		Where("user_name = ?", username).
		Update("last_login_date", time.Now()).Error
}

// GetAllUsers retrieves all users
func (s *AccountService) GetAllUsers(includeRemoved bool) ([]*model.Account, error) {
	var accounts []*model.Account
	query := s.db.Where("is_organization = ?", false)
	if !includeRemoved {
		query = query.Where("removed = ?", false)
	}
	err := query.Order("user_name").Find(&accounts).Error
	return accounts, err
}

// SearchUsers searches non-removed personal accounts by username or full name.
// Returns at most `limit` results (default 20).
//
// Optimization: Returns empty when keyword length <2 to avoid leading wildcard LIKE %x% matching too many rows;
// LIMIT hard-capped at 50 to prevent malicious full-table fetches.
//
// Cross-database compatibility: Uses LOWER(col) LIKE LOWER(?) for case-insensitive matching.
// SQLite/MySQL LIKE is case-insensitive by default, but PostgreSQL LIKE is case-sensitive,
// requiring LOWER() or ILIKE for consistent behavior. Since LIKE '%xxx%' cannot use indexes
// (leading wildcard), adding LOWER() incurs no additional performance cost.
func (s *AccountService) SearchUsers(keyword string, limit int) ([]*model.Account, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	// Search only when keyword is non-empty (single characters also supported)
	// LIMIT hard-capped at 50, single-character LIKE performance is acceptable
	if len(strings.TrimSpace(keyword)) == 0 {
		return []*model.Account{}, nil
	}
	var accounts []*model.Account
	like := "%" + strings.ToLower(keyword) + "%"
	err := s.db.
		Where("is_organization = ? AND removed = ? AND (LOWER(user_name) LIKE ? OR LOWER(full_name) LIKE ?)", false, false, like, like).
		Order("user_name").
		Limit(limit).
		Find(&accounts).Error
	return accounts, err
}

// CreateOrganization creates a new organization account and adds the creator as a manager
func (s *AccountService) CreateOrganization(creatorUserName, organizationName, description string) (*model.Account, error) {
	if IsReservedName(organizationName) {
		return nil, ErrReservedName
	}

	now := time.Now()
	account := &model.Account{
		UserName:       organizationName,
		FullName:       organizationName,
		MailAddress:    fmt.Sprintf("%s@localhost", organizationName),
		IsOrganization: true,
		RegisteredDate: now,
		UpdatedDate:    now,
		Description:    &description,
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(account).Error; err != nil {
			return err
		}

		member := &model.OrganizationMember{
			OrganizationName: organizationName,
			UserName:         creatorUserName,
			IsManager:        true,
			JoinedDate:       now,
		}
		return tx.Create(member).Error
	})
	if err != nil {
		return nil, err
	}

	return account, nil
}

// ListOrganizations retrieves all organization accounts
func (s *AccountService) ListOrganizations() ([]*model.Account, error) {
	var organizations []*model.Account
	err := s.db.Where("is_organization = ? AND removed = ?", true, false).
		Order("user_name").
		Find(&organizations).Error
	return organizations, err
}

// AddOrganizationMember adds a user to an organization
func (s *AccountService) AddOrganizationMember(organizationName, userName string, isManager bool) error {
	member := &model.OrganizationMember{
		OrganizationName: organizationName,
		UserName:         userName,
		IsManager:        isManager,
		JoinedDate:       time.Now(),
	}
	return s.db.Create(member).Error
}

// RemoveOrganizationMember removes a user from an organization
func (s *AccountService) RemoveOrganizationMember(organizationName, userName string) error {
	return s.db.Where("organization_name = ? AND user_name = ?", organizationName, userName).
		Delete(&model.OrganizationMember{}).Error
}

// IsOrganizationManager checks if a user is a manager of an organization
func (s *AccountService) IsOrganizationManager(organizationName, userName string) (bool, error) {
	var count int64
	err := s.db.Model(&model.OrganizationMember{}).
		Where("organization_name = ? AND user_name = ? AND is_manager = ?", organizationName, userName, true).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetManagedOrganizationNames returns the organization names where the given user is a manager
func (s *AccountService) GetManagedOrganizationNames(userName string) ([]string, error) {
	var names []string
	err := s.db.Model(&model.OrganizationMember{}).
		Where("user_name = ? AND is_manager = ?", userName, true).
		Pluck("organization_name", &names).Error
	return names, err
}

// IsOrganizationMember checks if a user is a member of an organization
func (s *AccountService) IsOrganizationMember(organizationName, userName string) (bool, error) {
	var count int64
	err := s.db.Model(&model.OrganizationMember{}).
		Where("organization_name = ? AND user_name = ?", organizationName, userName).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetOrganizationMembers retrieves all members of an organization with account info
func (s *AccountService) GetOrganizationMembers(organizationName string) ([]*model.OrganizationMemberDetail, error) {
	var members []*model.OrganizationMemberDetail
	err := s.db.Table("organization_member").
		Select("organization_member.organization_name, organization_member.user_name, organization_member.is_manager, organization_member.joined_date, account.full_name, account.image, account.administrator").
		Joins("LEFT JOIN account ON account.user_name = organization_member.user_name").
		Where("organization_member.organization_name = ? AND account.removed = ?", organizationName, false).
		Order("organization_member.joined_date ASC").
		Find(&members).Error
	return members, err
}

// GetUserOrganizations retrieves all organizations a user belongs to
func (s *AccountService) GetUserOrganizations(userName string) ([]*model.Account, error) {
	var organizations []*model.Account
	err := s.db.
		Joins("INNER JOIN organization_member ON account.user_name = organization_member.organization_name").
		Where("organization_member.user_name = ? AND account.is_organization = ? AND account.removed = ?", userName, true, false).
		Order("account.user_name").
		Find(&organizations).Error
	return organizations, err
}

// UpdateAccount updates user account information
func (s *AccountService) UpdateAccount(username string, updates map[string]interface{}) error {
	gormUpdates := map[string]interface{}{}

	if fullName, ok := updates["fullName"].(string); ok {
		gormUpdates["full_name"] = fullName
	}
	if mailAddress, ok := updates["mailAddress"].(string); ok {
		gormUpdates["mail_address"] = mailAddress
	}
	if url, ok := updates["url"].(string); ok {
		gormUpdates["url"] = url
	}
	if description, ok := updates["description"].(string); ok {
		gormUpdates["description"] = description
	}
	if image, ok := updates["image"].(string); ok {
		gormUpdates["image"] = image
	}
	if isAdmin, ok := updates["isAdmin"].(bool); ok {
		gormUpdates["administrator"] = isAdmin
	}
	if isRemoved, ok := updates["isRemoved"].(bool); ok {
		gormUpdates["removed"] = isRemoved
	}
	if password, ok := updates["password"].(string); ok && password != "" {
		hashed, err := hashPassword(password)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}
		gormUpdates["password"] = hashed
	}

	if len(gormUpdates) == 0 {
		return nil
	}

	gormUpdates["updated_date"] = time.Now()

	return s.db.Model(&model.Account{}).
		Where("user_name = ?", username).
		Updates(gormUpdates).Error
}

// DeleteAccount physically deletes a user (and cleans up related data).
// Note: Soft delete (setting isRemoved=true via UpdateAccount) is usually recommended;
// this method is for admin-forced user deletion.
func (s *AccountService) DeleteAccount(username string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Delete user's SSH keys
		if err := tx.Where("user_name = ?", username).Delete(&model.SSHKey{}).Error; err != nil {
			return err
		}
		// Delete user's GPG keys
		if err := tx.Where("user_name = ?", username).Delete(&model.GPGKey{}).Error; err != nil {
			return err
		}
		// Delete user's access tokens
		if err := tx.Where("user_name = ?", username).Delete(&model.AccessToken{}).Error; err != nil {
			return err
		}
		// Delete user's collaborator records
		if err := tx.Where("collaborator_name = ?", username).Delete(&model.Collaborator{}).Error; err != nil {
			return err
		}
		// Delete organization memberships
		if err := tx.Where("user_name = ?", username).Delete(&model.OrganizationMember{}).Error; err != nil {
			return err
		}
		// Delete user account
		if err := tx.Where("user_name = ?", username).Delete(&model.Account{}).Error; err != nil {
			return err
		}

		return nil
	})
}

// UserAvatarInfo holds avatar-related fields for a user.
type UserAvatarInfo struct {
	Image    *string
	FullName string
}

// GetUserAvatarInfo returns a map of username → avatar info (image + full name).
func (s *AccountService) GetUserAvatarInfo(userNames []string) (map[string]UserAvatarInfo, error) {
	result := make(map[string]UserAvatarInfo)
	if len(userNames) == 0 {
		return result, nil
	}
	type accountRow struct {
		UserName string  `gorm:"column:user_name"`
		Image    *string `gorm:"column:image"`
		FullName string  `gorm:"column:full_name"`
	}
	var rows []accountRow
	// Use Model(&Account{}) instead of Table("account") to leverage GORM's model abstraction layer
	err := s.db.Model(&model.Account{}).
		Select("user_name, image, full_name").
		Where("user_name IN ?", userNames).
		Find(&rows).Error
	if err != nil {
		return result, err
	}
	for _, r := range rows {
		result[r.UserName] = UserAvatarInfo{Image: r.Image, FullName: r.FullName}
	}
	return result, nil
}

// Participant represents a user in a sideloaded participants list.
type Participant struct {
	UserName string  `json:"userName"`
	FullName string  `json:"fullName"`
	Image    *string `json:"image"`
}

// BuildParticipants builds a deduplicated participant list from avatar info, preserving insertion order.
func (s *AccountService) BuildParticipants(avatarInfo map[string]UserAvatarInfo, userNames []string) []Participant {
	seen := make(map[string]bool)
	result := make([]Participant, 0, len(userNames))
	for _, name := range userNames {
		if seen[name] {
			continue
		}
		seen[name] = true
		info := avatarInfo[name]
		result = append(result, Participant{
			UserName: name,
			FullName: info.FullName,
			Image:    info.Image,
		})
	}
	return result
}

// BuildParticipantsForUsers fetches avatar info for the given userNames and
// builds a participant list in one call. Convenience wrapper around
// GetUserAvatarInfo + BuildParticipants. The avatar-info fetch error is
// ignored (matching the existing handler pattern): on error the map is empty
// and participants still contain the usernames with zero-value avatar fields.
func (s *AccountService) BuildParticipantsForUsers(userNames []string) []Participant {
	avatarInfo, _ := s.GetUserAvatarInfo(userNames)
	return s.BuildParticipants(avatarInfo, userNames)
}
