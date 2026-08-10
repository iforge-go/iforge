package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

var localLoc = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("CST", 8*3600)
	}
	return loc
}()

var (
	ErrActivityNotFound = errors.New("activity not found")
)

// ActivityService handles activity-related operations
type ActivityService struct {
	db *gorm.DB
}

// NewActivityService creates a new ActivityService
func NewActivityService(db *gorm.DB) *ActivityService {
	return &ActivityService{db: db}
}

// CreateActivity creates a new activity
func (s *ActivityService) CreateActivity(
	userName, repositoryName, activityUserName, activityType, message string,
	additionalInfo *string,
) (*model.Activity, error) {
	now := time.Now().In(localLoc)
	activity := &model.Activity{
		ActivityID:       generateActivityID(),
		UserName:         userName,
		RepositoryName:   repositoryName,
		ActivityUserName: activityUserName,
		ActivityType:     activityType,
		Message:          message,
		AdditionalInfo:   additionalInfo,
		ActivityDate:     now,
		ActivityDateDay:  now.Format("2006-01-02"),
	}

	if err := s.db.Create(activity).Error; err != nil {
		return nil, err
	}

	return activity, nil
}

// GetActivities retrieves activities for a repository
func (s *ActivityService) GetActivities(userName, repositoryName string, limit, offset int) ([]*model.Activity, error) {
	var activities []*model.Activity

	query := s.db.Where("user_name = ? AND repository_name = ?", userName, repositoryName).
		Order("activity_date DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&activities).Error; err != nil {
		return nil, err
	}

	return activities, nil
}

// GetUserActivities retrieves activities for a user across all repositories
func (s *ActivityService) GetUserActivities(userName string, limit, offset int) ([]*model.Activity, error) {
	var activities []*model.Activity

	query := s.db.Where("activity_user_name = ?", userName).
		Order("activity_date DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&activities).Error; err != nil {
		return nil, err
	}

	return activities, nil
}

// GetRecentActivities retrieves recent activities for a specific user (for dashboard).
//
// Optimized: Split 4 OR conditions into 4 independent subqueries using UNION, each utilizing its own index:
//  1. activity_user_name = ?          → idx_activity_user_date
//  2. user_name = ?                   → idx_activity_repo (user_name prefix)
//  3. user_name IN (org subquery)     → idx_activity_repo + organization_member
//  4. EXISTS collaborator             → idx_collab_name + idx_activity_repo
//
// Original OR + EXISTS approach couldn't short-circuit in SQLite, causing full table scans when any condition lacked an index.
// Uses UNION (not UNION ALL) for deduplication: user's own repo activities hit both branches 1 and 2,
// UNION ALL would show duplicates. UNION adds sorting overhead but branches use indexes, still much faster overall.
func (s *ActivityService) GetRecentActivities(userName string, limit, offset int) ([]*model.Activity, error) {
	// Query user's organization names once (avoid repeated subquery execution in UNION)
	var orgNames []string
	if err := s.db.Model(&model.OrganizationMember{}).
		Where("user_name = ?", userName).
		Pluck("organization_name", &orgNames).Error; err != nil {
		return nil, err
	}

	// Build UNION query: each branch uses its own index
	// Branch 1: activity_user_name = ? (user's own activities)
	unionSQL := `
		SELECT * FROM activity WHERE activity_user_name = ?
		UNION
		SELECT * FROM activity WHERE user_name = ?
	`
	args := []interface{}{userName, userName}

	// Branch 3: user_name IN (user's organizations)
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
		unionSQL += " UNION SELECT * FROM activity WHERE user_name IN (" + inClause + ")"
		args = append(args, placeholders...)
	}

	// Branch 4: activities from repos where user is a collaborator
	unionSQL += `
		UNION
		SELECT a.* FROM activity a
		INNER JOIN collaborator c
		  ON c.user_name = a.user_name AND c.repository_name = a.repository_name
		WHERE c.collaborator_name = ?
	`
	args = append(args, userName)

	// Outer ORDER BY + pagination
	orderClause := " ORDER BY activity_date DESC"
	if limit > 0 {
		orderClause += " LIMIT ?"
		args = append(args, limit)
		if offset > 0 {
			orderClause += " OFFSET ?"
			args = append(args, offset)
		}
	}

	var activities []*model.Activity
	if err := s.db.Raw("SELECT * FROM ("+unionSQL+") AS merged"+orderClause, args...).
		Scan(&activities).Error; err != nil {
		return nil, err
	}

	return activities, nil
}

// GetUserContributionDays retrieves user contribution counts by day for the past year.
//
// Optimized: Uses redundant column activity_date_day instead of DATE(activity_date, 'localtime') function,
// allowing GROUP BY to use the (activity_user_name, activity_date_day) index.
// Original approach wrapping column with DATE() function invalidated indexes completely, requiring full table scan + function computation.
func (s *ActivityService) GetUserContributionDays(userName string) ([]*model.ContributionDay, error) {
	oneYearAgo := time.Now().In(localLoc).AddDate(-1, 0, 0)
	oneYearAgoDay := oneYearAgo.Format("2006-01-02")

	var contributions []*model.ContributionDay
	// Use activity_date_day string comparison instead of DATE() function; keeps activity_date >= ? as fallback
	// Uses Model(&Activity{}) instead of Table("activity") to leverage model abstraction layer
	result := s.db.Model(&model.Activity{}).
		Select("activity_date_day as date, COUNT(*) as count").
		Where("activity_user_name = ? AND activity_date_day >= ?", userName, oneYearAgoDay).
		Group("activity_date_day").
		Scan(&contributions)
	if result.Error != nil {
		return nil, result.Error
	}

	return contributions, nil
}

// generateActivityID generates 16-byte random hex string as primary key.
// Uses crypto/rand instead of time.Now().UnixNano() to avoid collisions on Windows (15ms precision).
func generateActivityID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp+random if rand fails
		return time.Now().Format("20060102150405") + hex.EncodeToString(b)
	}
	return hex.EncodeToString(b)
}
