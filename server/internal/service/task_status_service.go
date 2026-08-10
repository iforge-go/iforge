package service

import (
	"errors"
	"time"

	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

var (
	ErrTaskStatusNotFound       = errors.New("task status not found")
	ErrTaskStatusExists         = errors.New("task status already exists")
	ErrInvalidTransition        = errors.New("invalid status transition: not allowed by project workflow rules")
	ErrTransitionStatusNotFound = errors.New("transition references unknown status slug")
)

// TaskStatusService handles task status related operations
type TaskStatusService struct {
	db *gorm.DB
}

// NewTaskStatusService creates a new TaskStatusService
func NewTaskStatusService(db *gorm.DB) *TaskStatusService {
	return &TaskStatusService{db: db}
}

// CreateTaskStatus creates a new task status
func (s *TaskStatusService) CreateTaskStatus(projectID int, name, slug, color, category string, position int, isDefault, isClosed bool, wipLimit *int) (*model.TaskStatus, error) {
	// Check if status with same slug already exists
	var existing model.TaskStatus
	err := s.db.Where("project_id = ? AND slug = ?", projectID, slug).First(&existing).Error
	if err == nil {
		return nil, ErrTaskStatusExists
	}

	// Normalize category: default invalid/empty values to todo
	if category != "todo" && category != "in_progress" && category != "done" {
		if isClosed {
			category = "done"
		} else {
			category = "todo"
		}
	}

	now := time.Now()
	status := &model.TaskStatus{
		ProjectID: projectID,
		Name:      name,
		Slug:      slug,
		Color:     color,
		Position:  position,
		IsDefault: isDefault,
		IsClosed:  isClosed,
		Category:  category,
		WipLimit:  wipLimit,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.db.Create(status).Error; err != nil {
		return nil, err
	}

	return status, nil
}

// GetTaskStatus retrieves a task status by ID
func (s *TaskStatusService) GetTaskStatus(id int) (*model.TaskStatus, error) {
	var status model.TaskStatus
	err := s.db.Where("id = ?", id).First(&status).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrTaskStatusNotFound
		}
		return nil, err
	}
	return &status, nil
}

// GetTaskStatusBySlug retrieves a task status by slug
func (s *TaskStatusService) GetTaskStatusBySlug(projectID int, slug string) (*model.TaskStatus, error) {
	var status model.TaskStatus
	err := s.db.Where("project_id = ? AND slug = ?", projectID, slug).First(&status).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrTaskStatusNotFound
		}
		return nil, err
	}
	return &status, nil
}

// ListTaskStatuses lists all task statuses for a project
func (s *TaskStatusService) ListTaskStatuses(projectID int) ([]*model.TaskStatus, error) {
	var statuses []*model.TaskStatus
	err := s.db.Where("project_id = ?", projectID).Order("position ASC, id ASC").Find(&statuses).Error
	if err != nil {
		return nil, err
	}
	// Deduplicate: keep first occurrence of each slug (prevents historical data duplicates)
	seen := make(map[string]bool)
	unique := make([]*model.TaskStatus, 0, len(statuses))
	for _, st := range statuses {
		if !seen[st.Slug] {
			seen[st.Slug] = true
			unique = append(unique, st)
		}
	}
	return unique, nil
}

// UpdateTaskStatus updates a task status
func (s *TaskStatusService) UpdateTaskStatus(status *model.TaskStatus) error {
	// Normalize category: default invalid values based on IsClosed (same as CreateTaskStatus)
	if status.Category != "todo" && status.Category != "in_progress" && status.Category != "done" {
		if status.IsClosed {
			status.Category = "done"
		} else {
			status.Category = "todo"
		}
	}
	status.UpdatedAt = time.Now()
	return s.db.Save(status).Error
}

// DeleteTaskStatus deletes a task status
func (s *TaskStatusService) DeleteTaskStatus(id int) error {
	return s.db.Where("id = ?", id).Delete(&model.TaskStatus{}).Error
}

// InitializeDefaultStatuses initializes default task statuses for a project
// 4 statuses: To Do / In Progress / Review / Done
func (s *TaskStatusService) InitializeDefaultStatuses(projectID int) error {
	defaultStatuses := []struct {
		name     string
		slug     string
		color    string
		position int
		isClosed bool
		category string
	}{
		{"待处理", "todo", "#daa724", 0, false, "todo"},
		{"进行中", "in_progress", "#079a0d", 1, false, "in_progress"},
		{"评审中", "review", "#9333ea", 2, false, "in_progress"},
		{"已完成", "done", "#8c023f", 3, true, "done"},
	}

	for _, ds := range defaultStatuses {
		status := &model.TaskStatus{
			ProjectID: projectID,
			Name:      ds.name,
			Slug:      ds.slug,
			Color:     ds.color,
			Position:  ds.position,
			IsDefault: true,
			IsClosed:  ds.isClosed,
			Category:  ds.category,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := s.db.Create(status).Error; err != nil {
			return err
		}
	}

	return nil
}

// ListTransitions lists all status transition rules configured for the project.
func (s *TaskStatusService) ListTransitions(projectID int) ([]*model.TaskStatusTransition, error) {
	var transitions []*model.TaskStatusTransition
	if err := s.db.Where("project_id = ?", projectID).Find(&transitions).Error; err != nil {
		return nil, err
	}
	// Populate FromSlug/ToSlug for frontend display (ProjectSlug is filled by handler layer using encodeID)
	statuses, err := s.ListTaskStatuses(projectID)
	if err != nil {
		return transitions, nil
	}
	slugByID := make(map[int]string, len(statuses))
	for _, st := range statuses {
		slugByID[st.ID] = st.Slug
	}
	for _, tr := range transitions {
		tr.FromSlug = slugByID[tr.FromStatusID]
		tr.ToSlug = slugByID[tr.ToStatusID]
	}
	return transitions, nil
}

// StatusTransitionPair represents a status transition rule (from→to), identified by slugs.
// Shared between handler and service to avoid type incompatibility from anonymous structs with different json tags.
type StatusTransitionPair struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// SetTransitions replaces all status transition rules for the project (aligned with Jira: configuration overwrites).
// Input is from/to slug pairs; empty slice clears rules (reverts to "allow all").
func (s *TaskStatusService) SetTransitions(projectID int, pairs []StatusTransitionPair) error {
	// slug→id mapping
	statuses, err := s.ListTaskStatuses(projectID)
	if err != nil {
		return err
	}
	idBySlug := make(map[string]int, len(statuses))
	for _, st := range statuses {
		idBySlug[st.Slug] = st.ID
	}

	// Build new rules (deduplicate + validate slugs)
	seen := make(map[[2]int]bool)
	var rows []*model.TaskStatusTransition
	for _, p := range pairs {
		fromID, ok1 := idBySlug[p.From]
		toID, ok2 := idBySlug[p.To]
		if !ok1 || !ok2 {
			return ErrTransitionStatusNotFound
		}
		if fromID == toID {
			continue // Self-loop is meaningless, skip
		}
		key := [2]int{fromID, toID}
		if seen[key] {
			continue
		}
		seen[key] = true
		rows = append(rows, &model.TaskStatusTransition{
			ProjectID:    projectID,
			FromStatusID: fromID,
			ToStatusID:   toID,
		})
	}

	// Transaction: delete old rules first, then insert new ones
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("project_id = ?", projectID).Delete(&model.TaskStatusTransition{}).Error; err != nil {
			return err
		}
		for _, r := range rows {
			if err := tx.Create(r).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// IsTransitionAllowed checks if fromSlug→toSlug is allowed by project workflow rules.
// Semantics (backward compatible):
//   - fromSlug or toSlug not found in task_status table (e.g., legacy closed/archived slugs): allow
//   - No rules configured for the project (empty table): allow
//   - Rules configured but (fromID→toID) doesn't match: deny
func (s *TaskStatusService) IsTransitionAllowed(projectID int, fromSlug, toSlug string) (bool, error) {
	if fromSlug == toSlug {
		return true, nil
	}
	var from, to model.TaskStatus
	if err := s.db.Where("project_id = ? AND slug = ?", projectID, fromSlug).First(&from).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return true, nil // Don't block unknown statuses (backward compatible with legacy slugs)
		}
		return false, err
	}
	if err := s.db.Where("project_id = ? AND slug = ?", projectID, toSlug).First(&to).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return true, nil
		}
		return false, err
	}

	// Empty rule set = allow all
	var count int64
	if err := s.db.Model(&model.TaskStatusTransition{}).Where("project_id = ?", projectID).Count(&count).Error; err != nil {
		return false, err
	}
	if count == 0 {
		return true, nil
	}

	// Match check
	var matched int64
	if err := s.db.Model(&model.TaskStatusTransition{}).
		Where("project_id = ? AND from_status_id = ? AND to_status_id = ?", projectID, from.ID, to.ID).
		Count(&matched).Error; err != nil {
		return false, err
	}
	return matched > 0, nil
}