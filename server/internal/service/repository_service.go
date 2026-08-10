package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	gitsvc "iforge/iforge/internal/git"
	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

var (
	ErrRepositoryNotFound    = errors.New("repository not found")
	ErrRepositoryExists      = errors.New("repository already exists")
	ErrInvalidRepositoryName = errors.New("invalid repository name")
)

// RepositoryService handles repository-related operations
type RepositoryService struct {
	db             *gorm.DB
	accountService *AccountService
	gitClient      *gitsvc.Client
}

// NewRepositoryService creates a new RepositoryService
func NewRepositoryService(db *gorm.DB, accountService *AccountService, gitClient *gitsvc.Client) *RepositoryService {
	return &RepositoryService{
		db:             db,
		accountService: accountService,
		gitClient:      gitClient,
	}
}

// CreateRepository creates a new repository
func (s *RepositoryService) CreateRepository(
	owner, name string,
	isPrivate bool,
	description *string,
	defaultBranch string,
	originUser, originRepo, parentUser, parentRepo *string,
	creator string,
) (*model.Repository, error) {
	var count int64
	if err := s.db.Model(&model.Repository{}).
		Where("user_name = ? AND repository_name = ? AND is_deleted = false", owner, name).
		Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrRepositoryExists
	}

	now := time.Now()
	repo := &model.Repository{
		UserName:             owner,
		RepositoryName:       name,
		IsPrivate:            isPrivate,
		Description:          description,
		DefaultBranch:        defaultBranch,
		RegisteredDate:       now,
		UpdatedDate:          now,
		LastActivityDate:     now,
		OriginUserName:       originUser,
		OriginRepositoryName: originRepo,
		ParentUserName:       parentUser,
		ParentRepositoryName: parentRepo,
		IsDeleted:            false,
		IssuesOption:         "PUBLIC",
		WikiOption:           "PUBLIC",
		AllowFork:            true,
		MergeOptions:         "merge-commit,squash,rebase",
		DefaultMergeOption:   "merge-commit",
		SafeMode:             true,
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(repo).Error; err != nil {
			return err
		}
		// Initialize issue ID counter
		counter := &model.IssueIDCounter{
			UserName:       owner,
			RepositoryName: name,
			IssueID:        0,
		}
		if err := tx.Create(counter).Error; err != nil {
			return err
		}
		// If creator is different from owner (e.g., creating repo for organization),
		// add creator as admin collaborator
		if creator != "" && creator != owner {
			collaborator := &model.Collaborator{
				UserName:         owner,
				RepositoryName:   name,
				CollaboratorName: creator,
				Role:             model.RoleAdmin,
			}
			if err := tx.Create(collaborator).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return repo, nil
}

// GetRepository retrieves a repository by owner and name
func (s *RepositoryService) GetRepository(owner, name string) (*model.Repository, error) {
	// Check cache first
	cacheKey := gitsvc.RepositoryCacheKey(owner, name)
	if cached, ok := gitsvc.RepositoryCache.Get(cacheKey); ok {
		// Return deep copy to prevent cache pollution or data races
		return cloneRepository(cached.(*model.Repository)), nil
	}

	// Fetch from database
	repo := &model.Repository{}
	if err := s.db.Where("user_name = ? AND repository_name = ? AND is_deleted = false", owner, name).First(repo).Error; err != nil {
		return nil, err
	}

	// Populate Options from flat fields
	repo.Options.IssuesOption = repo.IssuesOption
	repo.Options.ExternalIssuesURL = repo.ExternalIssuesURL
	repo.Options.WikiOption = repo.WikiOption
	repo.Options.ExternalWikiURL = repo.ExternalWikiURL
	repo.Options.AllowFork = repo.AllowFork
	repo.Options.MergeOptions = repo.MergeOptions
	repo.Options.DefaultMergeOption = repo.DefaultMergeOption
	repo.Options.SafeMode = repo.SafeMode

	// Store a clone in cache to prevent external mutation
	gitsvc.RepositoryCache.Set(cacheKey, cloneRepository(repo))
	return repo, nil
}

// cloneRepository returns a deep copy of repo.
// Repository has multiple pointer fields; a shallow copy would share underlying data.
func cloneRepository(repo *model.Repository) *model.Repository {
	if repo == nil {
		return nil
	}
	clone := *repo // Copy value fields
	// Allocate independent copies for pointer fields
	clone.Description = cloneStringPtr(repo.Description)
	clone.OriginUserName = cloneStringPtr(repo.OriginUserName)
	clone.OriginRepositoryName = cloneStringPtr(repo.OriginRepositoryName)
	clone.ParentUserName = cloneStringPtr(repo.ParentUserName)
	clone.ParentRepositoryName = cloneStringPtr(repo.ParentRepositoryName)
	clone.ExternalIssuesURL = cloneStringPtr(repo.ExternalIssuesURL)
	clone.ExternalWikiURL = cloneStringPtr(repo.ExternalWikiURL)
	clone.DefaultPriorityID = cloneIntPtr(repo.DefaultPriorityID)
	// Options also has pointer fields
	clone.Options.ExternalIssuesURL = cloneStringPtr(repo.Options.ExternalIssuesURL)
	clone.Options.ExternalWikiURL = cloneStringPtr(repo.Options.ExternalWikiURL)
	return &clone
}

func cloneStringPtr(p *string) *string {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func cloneIntPtr(p *int) *int {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

// UpdateRepository updates a repository
func (s *RepositoryService) UpdateRepository(repo *model.Repository) error {
	repo.UpdatedDate = time.Now()

	// Update flat fields from Options
	repo.IssuesOption = repo.Options.IssuesOption
	repo.ExternalIssuesURL = repo.Options.ExternalIssuesURL
	repo.WikiOption = repo.Options.WikiOption
	repo.ExternalWikiURL = repo.Options.ExternalWikiURL
	repo.AllowFork = repo.Options.AllowFork
	repo.MergeOptions = repo.Options.MergeOptions
	repo.DefaultMergeOption = repo.Options.DefaultMergeOption
	repo.SafeMode = repo.Options.SafeMode

	if err := s.db.Where("user_name = ? AND repository_name = ?", repo.UserName, repo.RepositoryName).
		Save(repo).Error; err != nil {
		return err
	}

	// Invalidate cache
	gitsvc.InvalidateRepositoryCache(repo.UserName, repo.RepositoryName)
	return nil
}

// DeleteRepository deletes a repository (soft delete - sets is_deleted = true)
func (s *RepositoryService) DeleteRepository(owner, name string) error {
	// Delete git repository directory on disk
	repoPath := s.gitClient.RepositoryPath(owner, name)
	os.RemoveAll(repoPath)

	// Invalidate all caches for this repository (per-repo, not global)
	gitsvc.InvalidateRepoCaches(owner, name)

	// Soft delete: set is_deleted = true instead of physical deletion
	now := time.Now()
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Mark repository as deleted
		if err := tx.Model(&model.Repository{}).
			Where("user_name = ? AND repository_name = ?", owner, name).
			Updates(map[string]interface{}{
				"is_deleted":   true,
				"updated_date": now,
			}).Error; err != nil {
			return err
		}

		// Optionally clean up related records (or keep them for audit trail)
		// For now, we keep related records but they won't be accessible via the soft-deleted repo
		return nil
	})
}

// ForkRepository forks a repository to a new owner
func (s *RepositoryService) ForkRepository(originalOwner, originalName, newOwner string) (*model.Repository, error) {
	// Get original repository
	originalRepo, err := s.GetRepository(originalOwner, originalName)
	if err != nil {
		return nil, err
	}

	// Check if repository already exists for new owner
	var count int64
	if err := s.db.Model(&model.Repository{}).
		Where("user_name = ? AND repository_name = ? AND is_deleted = false", newOwner, originalName).
		Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrRepositoryExists
	}

	now := time.Now()
	forkedRepo := &model.Repository{
		UserName:             newOwner,
		RepositoryName:       originalName,
		IsPrivate:            originalRepo.IsPrivate,
		Description:          originalRepo.Description,
		DefaultBranch:        originalRepo.DefaultBranch,
		RegisteredDate:       now,
		UpdatedDate:          now,
		LastActivityDate:     now,
		OriginUserName:       &originalOwner,
		OriginRepositoryName: &originalName,
		ParentUserName:       &originalOwner,
		ParentRepositoryName: &originalName,
		IsDeleted:            false,
		IssuesOption:         originalRepo.IssuesOption,
		ExternalIssuesURL:    originalRepo.ExternalIssuesURL,
		WikiOption:           originalRepo.WikiOption,
		ExternalWikiURL:      originalRepo.ExternalWikiURL,
		AllowFork:            originalRepo.AllowFork,
		MergeOptions:         originalRepo.MergeOptions,
		DefaultMergeOption:   originalRepo.DefaultMergeOption,
		SafeMode:             originalRepo.SafeMode,
	}

	if err := s.db.Create(forkedRepo).Error; err != nil {
		return nil, err
	}

	// Initialize issue ID counter
	counter := &model.IssueIDCounter{
		UserName:       newOwner,
		RepositoryName: originalName,
		IssueID:        0,
	}
	if err := s.db.Create(counter).Error; err != nil {
		return nil, err
	}

	return forkedRepo, nil
}

// GetVisibleRepositories retrieves repositories visible to a user.
//
// Optimized: splits 4 OR conditions into UNION subqueries, each using its own index:
//  1. private = FALSE                       → idx_repo_private
//  2. user_name = ?                         → primary key index
//  3. user_name IN (user's orgs)            → primary key index + organization_member
//  4. EXISTS collaborator                   → idx_collab_name + primary key
//
// Original OR + EXISTS + IN subquery cannot short-circuit in SQLite, causing full table scan.
// UNION (not UNION ALL) ensures deduplication.
func (s *RepositoryService) GetVisibleRepositories(username *string, limit int) ([]*model.Repository, error) {
	var repos []*model.Repository

	// Anonymous users: only public repos (single branch via idx_repo_private)
	if username == nil {
		query := s.db.Model(&model.Repository{}).Where("private = FALSE AND is_deleted = false")
		query = query.Order("last_activity_date DESC")
		if limit > 0 {
			query = query.Limit(limit)
		}
		if err := query.Find(&repos).Error; err != nil {
			return nil, err
		}
		s.populateRepoOptions(repos)
		return repos, nil
	}

	// Logged-in users: fetch org names once to avoid repeated subqueries in UNION
	var orgNames []string
	if err := s.db.Model(&model.OrganizationMember{}).
		Where("user_name = ?", *username).
		Pluck("organization_name", &orgNames).Error; err != nil {
		return nil, err
	}

	// Build UNION query: 4 branches each using their own index, all filtering is_deleted = false
	unionSQL := `
		SELECT * FROM repository WHERE private = FALSE AND is_deleted = false
		UNION
		SELECT * FROM repository WHERE user_name = ? AND is_deleted = false
	`
	args := []interface{}{*username}

	if len(orgNames) > 0 {
		placeholders := make([]interface{}, len(orgNames))
		inClause := ""
		for i, name := range orgNames {
			if i > 0 {
				inClause += ","
			}
			inClause += "?"
			placeholders[i] = name
		}
		unionSQL += " UNION SELECT * FROM repository WHERE user_name IN (" + inClause + ") AND is_deleted = false"
		args = append(args, placeholders...)
	}

	unionSQL += `
		UNION
		SELECT r.* FROM repository r
		INNER JOIN collaborator c
		  ON c.user_name = r.user_name AND c.repository_name = r.repository_name
		WHERE c.collaborator_name = ? AND r.is_deleted = false
	`
	args = append(args, *username)

	// Outer sort + pagination
	orderClause := " ORDER BY last_activity_date DESC"
	if limit > 0 {
		orderClause += " LIMIT ?"
		args = append(args, limit)
	}

	if err := s.db.Raw("SELECT * FROM ("+unionSQL+") AS merged"+orderClause, args...).
		Scan(&repos).Error; err != nil {
		return nil, err
	}

	s.populateRepoOptions(repos)
	return repos, nil
}

// populateRepoOptions backfills flat fields into Options struct (shared logic)
func (s *RepositoryService) populateRepoOptions(repos []*model.Repository) {
	for _, repo := range repos {
		repo.Options.IssuesOption = repo.IssuesOption
		repo.Options.ExternalIssuesURL = repo.ExternalIssuesURL
		repo.Options.WikiOption = repo.WikiOption
		repo.Options.ExternalWikiURL = repo.ExternalWikiURL
		repo.Options.AllowFork = repo.AllowFork
		repo.Options.MergeOptions = repo.MergeOptions
		repo.Options.DefaultMergeOption = repo.DefaultMergeOption
		repo.Options.SafeMode = repo.SafeMode
	}
}

// GetMyRepositories retrieves repositories the user owns or collaborates on.
// Unlike GetVisibleRepositories, this does NOT include all public repos — only
// repos where the user is the owner or an added collaborator.
func (s *RepositoryService) GetMyRepositories(userName string) ([]*model.Repository, error) {
	var repos []*model.Repository

	err := s.db.Where(
		"(user_name = ? OR EXISTS (SELECT 1 FROM collaborator WHERE collaborator.user_name = repository.user_name AND collaborator.repository_name = repository.repository_name AND collaborator.collaborator_name = ?)) AND is_deleted = false",
		userName, userName,
	).Order("last_activity_date DESC").Find(&repos).Error
	if err != nil {
		return nil, err
	}

	for _, repo := range repos {
		repo.Options.IssuesOption = repo.IssuesOption
		repo.Options.ExternalIssuesURL = repo.ExternalIssuesURL
		repo.Options.WikiOption = repo.WikiOption
		repo.Options.ExternalWikiURL = repo.ExternalWikiURL
		repo.Options.AllowFork = repo.AllowFork
		repo.Options.MergeOptions = repo.MergeOptions
		repo.Options.DefaultMergeOption = repo.DefaultMergeOption
		repo.Options.SafeMode = repo.SafeMode
	}

	return repos, nil
}

// GetRepositoriesByUser retrieves repositories owned by a specific user
func (s *RepositoryService) GetRepositoriesByUser(owner string, currentUser *string) ([]*model.Repository, error) {
	var repos []*model.Repository

	query := s.db.Where("user_name = ? AND is_deleted = false", owner)

	// If not the owner, only show public repos or repos where user is collaborator
	if currentUser != nil && *currentUser != owner {
		query = query.Where("private = FALSE OR EXISTS (SELECT 1 FROM collaborator WHERE collaborator.user_name = repository.user_name AND collaborator.repository_name = repository.repository_name AND collaborator.collaborator_name = ?)", *currentUser)
	} else if currentUser == nil {
		query = query.Where("private = FALSE")
	}

	query = query.Order("last_activity_date DESC")

	if err := query.Find(&repos).Error; err != nil {
		return nil, err
	}

	// Populate Options for each repository
	for _, repo := range repos {
		repo.Options.IssuesOption = repo.IssuesOption
		repo.Options.ExternalIssuesURL = repo.ExternalIssuesURL
		repo.Options.WikiOption = repo.WikiOption
		repo.Options.ExternalWikiURL = repo.ExternalWikiURL
		repo.Options.AllowFork = repo.AllowFork
		repo.Options.MergeOptions = repo.MergeOptions
		repo.Options.DefaultMergeOption = repo.DefaultMergeOption
		repo.Options.SafeMode = repo.SafeMode
	}

	return repos, nil
}

// AddCollaborator adds a collaborator to a repository
func (s *RepositoryService) AddCollaborator(owner, repo, collaborator, role string) error {
	collab := &model.Collaborator{
		UserName:         owner,
		RepositoryName:   repo,
		CollaboratorName: collaborator,
		Role:             role,
	}
	return s.db.Create(collab).Error
}

// RemoveCollaborator removes a collaborator from a repository
func (s *RepositoryService) RemoveCollaborator(owner, repo, collaborator string) error {
	return s.db.Where("user_name = ? AND repository_name = ? AND collaborator_name = ?", owner, repo, collaborator).
		Delete(&model.Collaborator{}).Error
}

// GetCollaborators retrieves all collaborators of a repository
func (s *RepositoryService) GetCollaborators(owner, repo string) ([]*model.Collaborator, error) {
	var collaborators []*model.Collaborator
	err := s.db.Where("user_name = ? AND repository_name = ?", owner, repo).Find(&collaborators).Error
	return collaborators, err
}

// GetForkCount returns the number of forks for a repository
func (s *RepositoryService) GetForkCount(owner, repo string) (int64, error) {
	var count int64
	err := s.db.Model(&model.Repository{}).
		Where("parent_user_name = ? AND parent_repository_name = ? AND is_deleted = false", owner, repo).
		Count(&count).Error
	return count, err
}

// CountRepositories returns the total number of repositories
func (s *RepositoryService) CountRepositories() (int64, error) {
	var count int64
	err := s.db.Model(&model.Repository{}).Where("is_deleted = false").Count(&count).Error
	return count, err
}

// ListAllRepositories returns all repositories with pagination (admin only, no visibility filter)
func (s *RepositoryService) ListAllRepositories(page, limit int) ([]*model.Repository, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var total int64
	if err := s.db.Model(&model.Repository{}).Where("is_deleted = false").Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var repos []*model.Repository
	offset := (page - 1) * limit
	if err := s.db.Model(&model.Repository{}).
		Where("is_deleted = false").
		Order("registered_date DESC").
		Limit(limit).
		Offset(offset).
		Find(&repos).Error; err != nil {
		return nil, 0, err
	}

	for _, repo := range repos {
		repo.Options.IssuesOption = repo.IssuesOption
		repo.Options.ExternalIssuesURL = repo.ExternalIssuesURL
		repo.Options.WikiOption = repo.WikiOption
		repo.Options.ExternalWikiURL = repo.ExternalWikiURL
		repo.Options.AllowFork = repo.AllowFork
		repo.Options.MergeOptions = repo.MergeOptions
		repo.Options.DefaultMergeOption = repo.DefaultMergeOption
		repo.Options.SafeMode = repo.SafeMode
	}

	return repos, total, nil
}

// GetForks retrieves all forks of a repository
func (s *RepositoryService) GetForks(owner, repo string, currentUser *string) ([]*model.Repository, error) {
	var repos []*model.Repository

	query := s.db.Where("parent_user_name = ? AND parent_repository_name = ? AND is_deleted = false", owner, repo)

	// Filter by visibility
	if currentUser != nil {
		query = query.Where("private = FALSE OR user_name = ? OR EXISTS (SELECT 1 FROM collaborator WHERE collaborator.user_name = repository.user_name AND collaborator.repository_name = repository.repository_name AND collaborator.collaborator_name = ?)",
			*currentUser, *currentUser)
	} else {
		query = query.Where("private = FALSE")
	}

	query = query.Order("last_activity_date DESC")

	if err := query.Find(&repos).Error; err != nil {
		return nil, err
	}

	// Populate Options for each repository
	for _, repo := range repos {
		repo.Options.IssuesOption = repo.IssuesOption
		repo.Options.ExternalIssuesURL = repo.ExternalIssuesURL
		repo.Options.WikiOption = repo.WikiOption
		repo.Options.ExternalWikiURL = repo.ExternalWikiURL
		repo.Options.AllowFork = repo.AllowFork
		repo.Options.MergeOptions = repo.MergeOptions
		repo.Options.DefaultMergeOption = repo.DefaultMergeOption
		repo.Options.SafeMode = repo.SafeMode
	}

	return repos, nil
}

// IsForked checks if a user has forked a repository
func (s *RepositoryService) IsForked(owner, repo, userName string) (bool, error) {
	var count int64
	err := s.db.Model(&model.Repository{}).
		Where("user_name = ? AND parent_user_name = ? AND parent_repository_name = ? AND is_deleted = false", userName, owner, repo).
		Count(&count).Error
	return count > 0, err
}

// TransferRepository transfers a repository to a new owner
func (s *RepositoryService) TransferRepository(oldOwner, repoName, newOwner string) error {
	// Check if repository exists
	_, err := s.GetRepository(oldOwner, repoName)
	if err != nil {
		return ErrRepositoryNotFound
	}

	// Check if repository already exists for new owner
	var count int64
	if err := s.db.Model(&model.Repository{}).
		Where("user_name = ? AND repository_name = ? AND is_deleted = false", newOwner, repoName).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrRepositoryExists
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		// Update repository owner
		if err := tx.Model(&model.Repository{}).
			Where("user_name = ? AND repository_name = ? AND is_deleted = false", oldOwner, repoName).
			Updates(map[string]interface{}{
				"user_name":    newOwner,
				"updated_date": now,
			}).Error; err != nil {
			return err
		}

		// Update all related tables
		tables := []interface{}{
			&model.Issue{},
			&model.IssueComment{},
			&model.CommitComment{},
			&model.IssueLabel{},
			&model.IssueIDCounter{},
			&model.MergeRequest{},
			&model.Label{},
			&model.Milestone{},
			&model.Priority{},
			&model.Collaborator{},
			&model.Webhook{},
			&model.Activity{},
			&model.ReleaseAsset{},
			&model.ReleaseTag{},
			&model.ProtectedBranch{},
			&model.DeployKey{},
			&model.CommitStatus{},
		}

		for _, table := range tables {
			if err := tx.Model(table).
				Where("user_name = ? AND repository_name = ?", oldOwner, repoName).
				Update("user_name", newOwner).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// RenameRepository renames a repository
func (s *RepositoryService) RenameRepository(owner, oldName, newName string) error {
	// Check if repository exists
	_, err := s.GetRepository(owner, oldName)
	if err != nil {
		return ErrRepositoryNotFound
	}

	// Check if new name already exists
	var count int64
	if err := s.db.Model(&model.Repository{}).
		Where("user_name = ? AND repository_name = ? AND is_deleted = false", owner, newName).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrRepositoryExists
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		// Update repository name
		if err := tx.Model(&model.Repository{}).
			Where("user_name = ? AND repository_name = ? AND is_deleted = false", owner, oldName).
			Updates(map[string]interface{}{
				"repository_name": newName,
				"updated_date":    now,
			}).Error; err != nil {
			return err
		}

		// Update all related tables
		tables := []interface{}{
			&model.Issue{},
			&model.IssueComment{},
			&model.CommitComment{},
			&model.IssueLabel{},
			&model.IssueIDCounter{},
			&model.MergeRequest{},
			&model.Label{},
			&model.Milestone{},
			&model.Priority{},
			&model.Collaborator{},
			&model.Webhook{},
			&model.Activity{},
			&model.ReleaseAsset{},
			&model.ReleaseTag{},
			&model.ProtectedBranch{},
			&model.DeployKey{},
			&model.CommitStatus{},
		}

		for _, table := range tables {
			if err := tx.Model(table).
				Where("user_name = ? AND repository_name = ?", owner, oldName).
				Update("repository_name", newName).Error; err != nil {
				return err
			}
		}

		// Rename Git repository directory on filesystem
		oldPath := s.gitClient.RepositoryPath(owner, oldName)
		newPath := s.gitClient.RepositoryPath(owner, newName)
		if _, err := os.Stat(oldPath); err == nil {
			if err := os.Rename(oldPath, newPath); err != nil {
				return fmt.Errorf("failed to rename git repository directory: %w", err)
			}
		}

		// Invalidate caches for both old and new names
		gitsvc.InvalidateRepositoryCache(owner, oldName)
		gitsvc.InvalidateRepositoryCache(owner, newName)
		return nil
	})
}

// ArchiveRepository archives or unarchives a repository
func (s *RepositoryService) ArchiveRepository(owner, repoName string, isArchived bool) error {
	// Check if repository exists
	_, err := s.GetRepository(owner, repoName)
	if err != nil {
		return ErrRepositoryNotFound
	}

	now := time.Now()
	if err := s.db.Model(&model.Repository{}).
		Where("user_name = ? AND repository_name = ? AND is_deleted = false", owner, repoName).
		Updates(map[string]interface{}{
			"is_archived":  isArchived,
			"updated_date": now,
		}).Error; err != nil {
		return err
	}
	gitsvc.InvalidateRepositoryCache(owner, repoName)
	return nil
}

// SetTemplateRepository sets or unsets a repository as a template
func (s *RepositoryService) SetTemplateRepository(owner, repoName string, isTemplate bool) error {
	// Check if repository exists
	_, err := s.GetRepository(owner, repoName)
	if err != nil {
		return ErrRepositoryNotFound
	}

	now := time.Now()
	if err := s.db.Model(&model.Repository{}).
		Where("user_name = ? AND repository_name = ? AND is_deleted = false", owner, repoName).
		Updates(map[string]interface{}{
			"is_template":  isTemplate,
			"updated_date": now,
		}).Error; err != nil {
		return err
	}
	gitsvc.InvalidateRepositoryCache(owner, repoName)
	return nil
}

// CreateFromTemplate creates a new repository from a template
func (s *RepositoryService) CreateFromTemplate(templateOwner, templateRepo, newOwner, newRepo string, isPrivate bool) (*model.Repository, error) {
	// Get template repository
	template, err := s.GetRepository(templateOwner, templateRepo)
	if err != nil {
		return nil, ErrRepositoryNotFound
	}

	// Check if template is actually a template
	if !template.IsTemplate {
		return nil, fmt.Errorf("repository is not a template")
	}

	// Check permissions - must be able to read template
	// For private templates, need collaborator access
	if template.IsPrivate && template.UserName != newOwner {
		var count int64
		err = s.db.Model(&model.Collaborator{}).
			Where("user_name = ? AND repository_name = ? AND collaborator_name = ?", templateOwner, templateRepo, newOwner).
			Count(&count).Error
		if err != nil || count == 0 {
			return nil, fmt.Errorf("no permission to use this template")
		}
	}

	// Create new repository record
	now := time.Now()
	newRepoModel := &model.Repository{
		UserName:           newOwner,
		RepositoryName:     newRepo,
		IsPrivate:          isPrivate,
		Description:        template.Description,
		DefaultBranch:      template.DefaultBranch,
		RegisteredDate:     now,
		UpdatedDate:        now,
		LastActivityDate:   now,
		IsTemplate:         false, // New repo is not a template by default
		IssuesOption:       template.IssuesOption,
		ExternalIssuesURL:  template.ExternalIssuesURL,
		WikiOption:         template.WikiOption,
		ExternalWikiURL:    template.ExternalWikiURL,
		AllowFork:          template.AllowFork,
		MergeOptions:       template.MergeOptions,
		DefaultMergeOption: template.DefaultMergeOption,
		SafeMode:           template.SafeMode,
	}

	if err := s.db.Create(newRepoModel).Error; err != nil {
		return nil, err
	}

	// Copy files from template using git clone --bare
	templateRepoPath := s.gitClient.RepositoryPath(templateOwner, templateRepo)

	// Clone template as bare repository
	ctx := context.Background()
	_, err = s.gitClient.CloneBare(ctx, newOwner, newRepo, templateRepoPath)
	if err != nil {
		// Clean up on failure
		os.RemoveAll(s.gitClient.RepositoryPath(newOwner, newRepo))
		return nil, fmt.Errorf("failed to clone template: %w", err)
	}

	return newRepoModel, nil
}

// HasOwnerRole checks if a user has owner-level permissions (highest level)
// Aligned with GitBucket: System Admin OR Repository Owner OR Organization Manager OR ADMIN Collaborator
func (s *RepositoryService) HasOwnerRole(repo *model.Repository, user *model.Account) bool {
	return s.hasRole(repo, user, model.RoleAdmin)
}

// HasMemberRole checks if a user has member-level permissions (write access)
// Aligned with GitBucket: System Admin OR Repository Owner OR Organization Member OR ADMIN/MEMBER Collaborator
// Renamed from HasDeveloperRole to align with PMS role naming (admin/member/viewer).
func (s *RepositoryService) HasMemberRole(repo *model.Repository, user *model.Account) bool {
	return s.hasRole(repo, user, model.RoleMember)
}

// HasViewerRole checks if a user has viewer-level permissions (lowest level)
// Aligned with GitBucket: System Admin OR Repository Owner OR Organization Member OR Any Collaborator
// Renamed from HasGuestRole to align with PMS role naming (admin/member/viewer).
func (s *RepositoryService) HasViewerRole(repo *model.Repository, user *model.Account) bool {
	return s.hasRole(repo, user, model.RoleViewer)
}

// hasRole is the shared implementation behind HasOwnerRole/HasMemberRole/HasViewerRole.
// required must be one of model.RoleAdmin / model.RoleMember / model.RoleViewer.
// Aligned with GitBucket role checks:
//   - System Admin OR Repository Owner short-circuits to true.
//   - Organization: Manager (for RoleAdmin) or any Member (otherwise).
//   - Collaborator: role must be at least the required level.
func (s *RepositoryService) hasRole(repo *model.Repository, user *model.Account, required string) bool {
	if user == nil {
		return false
	}

	// System administrator has every role for all repositories
	if user.IsAdmin {
		return true
	}

	// Repository owner always has every role
	if repo.UserName == user.UserName {
		return true
	}

	// Check if repository owner is an organization:
	// - For owner-level (admin) role, the user must be an organization manager.
	// - For lower roles, any organization member qualifies.
	if s.accountService != nil {
		ownerAccount, err := s.accountService.GetAccountByUsername(repo.UserName)
		if err == nil && ownerAccount != nil && ownerAccount.IsOrganization {
			if required == model.RoleAdmin {
				if isManager, err := s.accountService.IsOrganizationManager(repo.UserName, user.UserName); err == nil && isManager {
					return true
				}
			} else {
				if isMember, err := s.accountService.IsOrganizationMember(repo.UserName, user.UserName); err == nil && isMember {
					return true
				}
			}
		}
	}

	// Check if user is a collaborator with a role at least the required level
	collab := &model.Collaborator{}
	err := s.db.Where("user_name = ? AND repository_name = ? AND collaborator_name = ?",
		repo.UserName, repo.RepositoryName, user.UserName).First(collab).Error

	if err != nil {
		return false
	}

	return model.HasRoleAtLeast(collab.Role, required)
}

// IsIssueEditable tests whether a logged-in user can create issues / merge requests.
// Aligned with GitBucket's IssueCreationService.isIssueEditable:
//   - ALL     => !isPrivate && loginAccount.isDefined
//   - PUBLIC  => hasGuestRole
//   - PRIVATE => hasDeveloperRole
//   - DISABLE => false
//
// Default (empty IssuesOption) uses ALL mode — any logged-in
// user can create issues/PRs in public repositories.
func (s *RepositoryService) IsIssueEditable(repo *model.Repository, user *model.Account) bool {
	if user == nil {
		return false
	}
	option := repo.IssuesOption
	if option == "" {
		option = "ALL" // Default to ALL
	}
	switch option {
	case "ALL":
		// Any logged-in user can create issues, but only for non-private repos
		return !repo.IsPrivate
	case "PUBLIC":
		return s.HasViewerRole(repo, user)
	case "PRIVATE":
		return s.HasMemberRole(repo, user)
	case "DISABLE":
		return false
	}
	return false
}

// IsIssueManageable tests whether a logged-in user can manage issues
// (assign assignees / milestones / labels). Aligned with GitBucket's
// IssueCreationService.isIssueManageable: hasMemberRole (was hasDeveloperRole).
func (s *RepositoryService) IsIssueManageable(repo *model.Repository, user *model.Account) bool {
	return s.HasMemberRole(repo, user)
}

// IsIssueCommentManageable tests whether a logged-in user can manage
// issue comments (e.g. delete others' comments). Aligned with GitBucket's
// IssueCreationService.isIssueCommentManageable: hasOwnerRole.
func (s *RepositoryService) IsIssueCommentManageable(repo *model.Repository, user *model.Account) bool {
	return s.HasOwnerRole(repo, user)
}

// IsBranchProtected checks if a branch is protected
func (s *RepositoryService) IsBranchProtected(owner, repo, branch string) (bool, error) {
	var count int64
	err := s.db.Model(&model.ProtectedBranch{}).
		Where("user_name = ? AND repository_name = ? AND branch = ?", owner, repo, branch).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// CanPushToBranch checks if user can push to a specific branch (considering protection)
func (s *RepositoryService) CanPushToBranch(owner, repo, branch, userName string) (bool, error) {
	// Check if branch is protected
	isProtected, err := s.IsBranchProtected(owner, repo, branch)
	if err != nil {
		return false, err
	}

	// If not protected, allow push
	if !isProtected {
		return true, nil
	}

	// Branch is protected - only repository admins can push
	repository, err := s.GetRepository(owner, repo)
	if err != nil {
		return false, err
	}

	user, err := s.accountService.GetAccountByUsername(userName)
	if err != nil {
		return false, nil
	}

	return s.HasOwnerRole(repository, user), nil
}

// ListProtectedBranches lists all protected branches for a repository
func (s *RepositoryService) ListProtectedBranches(owner, repo string) ([]gitsvc.ProtectedBranchInfo, error) {
	branches := []model.ProtectedBranch{}
	err := s.db.
		Where("user_name = ? AND repository_name = ?", owner, repo).
		Find(&branches).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query protected branches: %w", err)
	}
	protected := make([]gitsvc.ProtectedBranchInfo, len(branches))
	for i, pb := range branches {
		protected[i] = gitsvc.ProtectedBranchInfo{
			BranchName: pb.Branch,
			Status:     pb.Status,
		}
	}
	return protected, nil
}

// ProtectBranch protects a branch
func (s *RepositoryService) ProtectBranch(owner, repo, branchName string) error {
	pb := &model.ProtectedBranch{
		UserName:       owner,
		RepositoryName: repo,
		Branch:         branchName,
		Status:         "enabled",
	}
	if err := s.db.Save(pb).Error; err != nil {
		return fmt.Errorf("failed to protect branch: %w", err)
	}
	return nil
}

// UnprotectBranch unprotects a branch
func (s *RepositoryService) UnprotectBranch(owner, repo, branchName string) error {
	err := s.db.
		Where("user_name = ? AND repository_name = ? AND branch = ?", owner, repo, branchName).
		Delete(&model.ProtectedBranch{}).Error
	if err != nil {
		return fmt.Errorf("failed to unprotect branch: %w", err)
	}
	return nil
}
