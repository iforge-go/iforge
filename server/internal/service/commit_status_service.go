package service

import (
	"errors"
	"time"

	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

var (
	ErrCommitStatusNotFound = errors.New("commit status not found")
)

type CommitStatusService struct {
	db *gorm.DB
}

func NewCommitStatusService(db *gorm.DB) *CommitStatusService {
	return &CommitStatusService{db: db}
}

// CreateOrUpdateStatus creates or updates a commit status
func (s *CommitStatusService) CreateOrUpdateStatus(
	userName, repoName, commitID, context, state string,
	targetURL, description *string,
	creator string,
) (*model.CommitStatus, error) {
	now := time.Now()
	status := &model.CommitStatus{
		UserName:       userName,
		RepositoryName: repoName,
		CommitID:       commitID,
		Context:        context,
		State:          state,
		TargetURL:      targetURL,
		Description:    description,
		UpdatedDate:    now,
		Creator:        creator,
	}

	// Try to update existing status
	result := s.db.Model(&model.CommitStatus{}).
		Where("user_name = ? AND repository_name = ? AND commit_id = ? AND context = ?",
			userName, repoName, commitID, context).
		Updates(map[string]interface{}{
			"state":        status.State,
			"target_url":   status.TargetURL,
			"description":  status.Description,
			"updated_date": status.UpdatedDate,
			"creator":      status.Creator,
		})

	if result.Error != nil {
		return nil, result.Error
	}

	// If no rows were affected, insert new status
	if result.RowsAffected == 0 {
		if err := s.db.Create(status).Error; err != nil {
			return nil, err
		}
	}

	return status, nil
}

// GetStatusesForCommit retrieves all statuses for a commit
func (s *CommitStatusService) GetStatusesForCommit(userName, repoName, commitID string) ([]*model.CommitStatus, error) {
	var statuses []*model.CommitStatus
	err := s.db.
		Where("user_name = ? AND repository_name = ? AND commit_id = ?", userName, repoName, commitID).
		Order("context ASC").
		Find(&statuses).Error
	return statuses, err
}

// GetCombinedStatus retrieves the combined status for a commit
func (s *CommitStatusService) GetCombinedStatus(userName, repoName, commitID string) (*model.CombinedStatus, error) {
	statuses, err := s.GetStatusesForCommit(userName, repoName, commitID)
	if err != nil {
		return nil, err
	}

	if len(statuses) == 0 {
		return &model.CombinedStatus{
			UserName:       userName,
			RepositoryName: repoName,
			CommitID:       commitID,
			State:          "pending",
			Statuses:       []*model.CommitStatus{},
		}, nil
	}

	// Determine combined state
	state := "success"
	hasPending := false
	hasFailure := false

	for _, status := range statuses {
		switch status.State {
		case "pending":
			hasPending = true
		case "failure", "error":
			hasFailure = true
		}
	}

	if hasFailure {
		state = "failure"
	} else if hasPending {
		state = "pending"
	}

	return &model.CombinedStatus{
		UserName:       userName,
		RepositoryName: repoName,
		CommitID:       commitID,
		State:          state,
		Statuses:       statuses,
	}, nil
}

// GetStatus retrieves a specific status by context
func (s *CommitStatusService) GetStatus(userName, repoName, commitID, context string) (*model.CommitStatus, error) {
	status := &model.CommitStatus{}
	err := s.db.
		Where("user_name = ? AND repository_name = ? AND commit_id = ? AND context = ?",
			userName, repoName, commitID, context).
		First(status).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrCommitStatusNotFound
		}
		return nil, err
	}

	return status, nil
}
