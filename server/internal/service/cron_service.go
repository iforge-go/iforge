package service

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"iforge/iforge/internal/git"
	"iforge/iforge/internal/model"

	"gorm.io/gorm"
)

// ============================================================================
// Error definitions
// ============================================================================

var (
	ErrCronNotFound    = errors.New("cron schedule not found")
	ErrCronNameExists  = errors.New("cron name already exists in this repository")
	ErrInvalidCronExpr = errors.New("invalid cron expression")
)

// ============================================================================
// CronService
// ============================================================================

// CronService manages CI/CD scheduled trigger configuration + scheduling
//
// Design:
//   - Scheduler goroutine wakes up every minute, queries cron with enabled=true and next_run_at <= now
//   - Triggers PipelineService.CreatePipeline(trigger_event=cron, triggered_by="system")
//   - Updates last_run_at and next_run_at after trigger (recalculates based on cron expression)
//   - Also ticks immediately on startup to catch up on missed crons during restart (next_run_at <= now)
//
// Note: Pipelines with triggered_by="system" don't send user notifications (to avoid sending
// notifications to non-existent system user), but still trigger repository webhooks.
type CronService struct {
	db              *gorm.DB
	pipelineService *PipelineService
	gitClient       *git.Client
	logger          *git.Logger
	stopCh          chan struct{}
	wg              sync.WaitGroup
}

// NewCronService creates CronService (doesn't start scheduler; must call Start explicitly)
func NewCronService(db *gorm.DB, pipelineService *PipelineService, gitClient *git.Client, logger *git.Logger) *CronService {
	return &CronService{
		db:              db,
		pipelineService: pipelineService,
		gitClient:       gitClient,
		logger:          logger,
		stopCh:          make(chan struct{}),
	}
}

// Start starts the scheduler goroutine
func (s *CronService) Start() {
	s.wg.Add(1)
	go s.scheduler()
}

// Stop stops the scheduler
func (s *CronService) Stop() {
	close(s.stopCh)
	s.wg.Wait()
}

// scheduler main loop: wakes up every minute
func (s *CronService) scheduler() {
	defer s.wg.Done()
	// Tick immediately on startup to catch up on missed crons during restart
	s.tick()
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.tick()
		}
	}
}

// tick checks all due crons and triggers pipelines
func (s *CronService) tick() {
	now := time.Now()
	var crons []model.CronSchedule
	// next_run_at IS NULL fallback: crons never scheduled trigger immediately (and calculate next time)
	if err := s.db.Where("enabled = ? AND (next_run_at IS NULL OR next_run_at <= ?)", true, now).
		Find(&crons).Error; err != nil {
		s.logError(fmt.Sprintf("Cron tick query failed: %v", err))
		return
	}
	if len(crons) == 0 {
		return
	}
	for _, cron := range crons {
		s.triggerCron(cron, now)
	}
}

// triggerCron triggers a single cron
func (s *CronService) triggerCron(cron model.CronSchedule, now time.Time) {
	commitSHA, err := s.getBranchHead(cron.UserName, cron.RepositoryName, cron.Branch)
	if err != nil {
		s.logError(fmt.Sprintf("Cron %q (%s/%s): get branch head failed: %v",
			cron.Name, cron.UserName, cron.RepositoryName, err))
		s.updateNextRun(cron, now)
		return
	}

	// Get yaml (use cron override config, or read repository's .iforge-ci.yml)
	yamlConfig := cron.YAMLConfig
	if yamlConfig == "" {
		yamlBytes, err := s.gitClient.GetRawFileContent(cron.UserName, cron.RepositoryName, commitSHA, ".iforge-ci.yml")
		if err != nil {
			s.logError(fmt.Sprintf("Cron %q (%s/%s): no .iforge-ci.yml at %s and no override: %v",
				cron.Name, cron.UserName, cron.RepositoryName, commitSHA[:8], err))
			s.updateNextRun(cron, now)
			return
		}
		yamlConfig = string(yamlBytes)
	}

	ref := "refs/heads/" + cron.Branch
	pipeline, err := s.pipelineService.CreatePipeline(
		cron.UserName, cron.RepositoryName,
		model.TriggerEventCron,
		ref, commitSHA,
		"system", // system trigger, skip user notifications
		fmt.Sprintf("Scheduled run (%s)", cron.Name),
		yamlConfig,
		nil,
	)
	if err != nil {
		s.logError(fmt.Sprintf("Cron %q (%s/%s): CreatePipeline failed: %v",
			cron.Name, cron.UserName, cron.RepositoryName, err))
	} else {
		if s.logger != nil {
			s.logger.Info(fmt.Sprintf("CICD: cron %q triggered pipeline #%d (branch=%s, commit=%s)",
				cron.Name, pipeline.ID, cron.Branch, commitSHA[:8]), nil)
		}
	}
	s.updateNextRun(cron, now)
}

// updateNextRun updates last_run_at and next_run_at
// Disables cron if cron expression is invalid (theoretically shouldn't happen; CreateCron/UpdateCron already validated) to avoid repeated triggers
func (s *CronService) updateNextRun(cron model.CronSchedule, now time.Time) {
	expr, err := ParseCron(cron.Schedule)
	if err != nil {
		s.logError(fmt.Sprintf("Cron %q: invalid schedule %q, disabling: %v",
			cron.Name, cron.Schedule, err))
		s.db.Model(&model.CronSchedule{}).Where("id = ?", cron.ID).
			Updates(map[string]interface{}{
				"enabled":     false,
				"last_run_at": now,
				"updated_at":  now,
			})
		return
	}
	nextRun := expr.NextTime(now)
	s.db.Model(&model.CronSchedule{}).Where("id = ?", cron.ID).
		Updates(map[string]interface{}{
			"last_run_at": now,
			"next_run_at": nextRun,
			"updated_at":  now,
		})
}

// getBranchHead gets the latest commit SHA for a branch
func (s *CronService) getBranchHead(owner, repo, branch string) (string, error) {
	if s.gitClient == nil {
		return "", fmt.Errorf("gitClient not configured")
	}
	return s.gitClient.GetBranchHead(owner, repo, branch)
}

func (s *CronService) logError(msg string) {
	if s.logger != nil {
		s.logger.Error(msg, nil, nil)
	} else {
		fmt.Printf("[error] %s\n", msg)
	}
}

// ============================================================================
// CRUD API
// ============================================================================

// CreateCron creates cron configuration
func (s *CronService) CreateCron(owner, repo, name, schedule, branch, yamlConfig, creator string) (*model.CronSchedule, error) {
	// Validate cron expression
	expr, err := ParseCron(schedule)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidCronExpr, err)
	}
	// Check name uniqueness (uniqueIndex will ensure this, but check early to return friendly error)
	var existing model.CronSchedule
	if err := s.db.Where("user_name = ? AND repository_name = ? AND name = ?", owner, repo, name).
		First(&existing).Error; err == nil {
		return nil, ErrCronNameExists
	}
	now := time.Now()
	nextRun := expr.NextTime(now)
	cron := &model.CronSchedule{
		UserName:       owner,
		RepositoryName: repo,
		Name:           name,
		Schedule:       schedule,
		Branch:         branch,
		YAMLConfig:     yamlConfig,
		Enabled:        true,
		Creator:        creator,
		NextRunAt:      &nextRun,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.db.Create(cron).Error; err != nil {
		return nil, err
	}
	return cron, nil
}

// ListCrons lists cron configurations for a repository
func (s *CronService) ListCrons(owner, repo string) ([]model.CronSchedule, error) {
	var crons []model.CronSchedule
	err := s.db.Where("user_name = ? AND repository_name = ?", owner, repo).
		Order("created_at DESC").Find(&crons).Error
	return crons, err
}

// GetCron queries a single cron
func (s *CronService) GetCron(id int64) (*model.CronSchedule, error) {
	var cron model.CronSchedule
	if err := s.db.First(&cron, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCronNotFound
		}
		return nil, err
	}
	return &cron, nil
}

// UpdateCron updates cron (recalculates next_run_at when schedule changes)
func (s *CronService) UpdateCron(id int64, schedule, branch, yamlConfig string, enabled bool) (*model.CronSchedule, error) {
	cron, err := s.GetCron(id)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	updates := map[string]interface{}{
		"branch":      branch,
		"yaml_config": yamlConfig,
		"enabled":     enabled,
		"updated_at":  now,
	}
	// Schedule changed; validate + recalculate next_run_at
	if schedule != cron.Schedule {
		expr, err := ParseCron(schedule)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidCronExpr, err)
		}
		updates["schedule"] = schedule
		if enabled {
			nextRun := expr.NextTime(now)
			updates["next_run_at"] = nextRun
		}
	}
	if err := s.db.Model(&model.CronSchedule{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.GetCron(id)
}

// DeleteCron deletes cron
func (s *CronService) DeleteCron(id int64) error {
	result := s.db.Delete(&model.CronSchedule{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrCronNotFound
	}
	return nil
}
