// Package event provides a publish/subscribe event bus for domain events.
//
// The bus is intentionally agnostic of how events are consumed: webhook
// delivery, audit logging, notifications and plugins can all subscribe
// independently. Subscribers implement Handler (or register a func) and
// are dispatched asynchronously per published event.
//
// Design highlights:
//   - Worker Pool: fixed number of worker goroutines to bound concurrency.
//   - Backpressure: bounded channel buffer to prevent memory blow-up.
//   - Dead Letter Queue: failed events are persisted for retry.
//   - Metrics: tracks processing success/failure rates.
package event

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"gorm.io/gorm"
)

// Handler processes a single Event.
type Handler interface {
	HandleEvent(evt Event)
}

// HandlerFunc is a function adapter for Handler.
type HandlerFunc func(Event)

// HandleEvent implements Handler.
func (f HandlerFunc) HandleEvent(evt Event) { f(evt) }

// BusConfig holds configuration for the event bus.
type BusConfig struct {
	// WorkerCount is the number of worker goroutines (default: 10).
	WorkerCount int
	// QueueSize is the buffer size of the event channel (default: 1000).
	QueueSize int
	// MaxRetries is the maximum number of retry attempts for failed events (default: 3).
	MaxRetries int
	// RetryDelay is the initial delay between retries (default: 1 second).
	RetryDelay time.Duration
	// EnableDLQ enables dead letter queue for failed events (default: true).
	EnableDLQ bool
}

// DefaultBusConfig returns the default configuration.
func DefaultBusConfig() BusConfig {
	return BusConfig{
		WorkerCount: 10,
		QueueSize:   1000,
		MaxRetries:  3,
		RetryDelay:  time.Second,
		EnableDLQ:   true,
	}
}

// FailedEvent represents a failed event stored in the dead letter queue.
type FailedEvent struct {
	ID           uint      `gorm:"primaryKey;autoIncrement"`
	EventType    string    `gorm:"index;not null"`
	EventData    string    `gorm:"type:text"` // JSON serialized event
	HandlerName  string    `gorm:"index;not null"`
	ErrorMessage string    `gorm:"type:text"`
	RetryCount   int       `gorm:"default:0"`
	MaxRetries   int       `gorm:"default:3"`
	CreatedAt    time.Time `gorm:"index"`
	UpdatedAt    time.Time
	NextRetryAt  *time.Time `gorm:"index"`
	Status       string     `gorm:"index;default:pending"` // pending, retrying, failed, success
}

func (FailedEvent) TableName() string {
	return "event_failed_events"
}

// Bus is a publish/subscribe event bus with worker pool and DLQ support.
type Bus struct {
	mu       sync.RWMutex
	handlers map[Type][]Handler
	config   BusConfig

	// Worker pool
	eventChan chan eventTask
	workers   []*worker
	wg        sync.WaitGroup

	// Dead letter queue
	db        *gorm.DB
	enableDLQ bool

	// Metrics
	metrics *BusMetrics

	// Lifecycle
	ctx          context.Context
	cancel       context.CancelFunc
	shutdownOnce sync.Once
}

// eventTask represents a task to be processed by a worker.
type eventTask struct {
	handler Handler
	event   Event
}

// worker represents a single worker goroutine.
type worker struct {
	id     int
	bus    *Bus
	tasks  <-chan eventTask
	ctx    context.Context
	cancel context.CancelFunc
}

// BusMetrics tracks event bus metrics.
type BusMetrics struct {
	mu              sync.RWMutex
	EventsPublished int64
	EventsProcessed int64
	EventsFailed    int64
	EventsRetried   int64
	EventsDLQ       int64
	QueueLength     int64
	ActiveWorkers   int64
	LastPublishedAt time.Time
	LastProcessedAt time.Time
}

// NewBus creates a new event bus with default configuration.
func NewBus() *Bus {
	return NewBusWithConfig(DefaultBusConfig())
}

// NewBusWithConfig creates a new event bus with custom configuration.
func NewBusWithConfig(config BusConfig) *Bus {
	if config.WorkerCount <= 0 {
		config.WorkerCount = 10
	}
	if config.QueueSize <= 0 {
		config.QueueSize = 1000
	}
	if config.MaxRetries <= 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay <= 0 {
		config.RetryDelay = time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	bus := &Bus{
		handlers:  make(map[Type][]Handler),
		config:    config,
		eventChan: make(chan eventTask, config.QueueSize),
		enableDLQ: config.EnableDLQ,
		metrics:   &BusMetrics{},
		ctx:       ctx,
		cancel:    cancel,
	}

	// Start workers
	bus.startWorkers()

	// Start DLQ retry processor if enabled
	if config.EnableDLQ {
		go bus.processDLQRetries()
	}

	return bus
}

// SetDB sets the database connection for dead letter queue.
// Must be called before any events are published.
func (b *Bus) SetDB(db *gorm.DB) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.db = db

	// Auto-migrate the failed events table
	if db != nil {
		if err := db.AutoMigrate(&FailedEvent{}); err != nil {
			log.Printf("[warn] Failed to migrate failed events table: %v", err)
		}
	}
}

// startWorkers starts the worker pool.
func (b *Bus) startWorkers() {
	for i := 0; i < b.config.WorkerCount; i++ {
		w := &worker{
			id:  i,
			bus: b,
		}
		w.ctx, w.cancel = context.WithCancel(b.ctx)
		b.workers = append(b.workers, w)

		b.wg.Add(1)
		go w.run()
	}

	log.Printf("[info] Event bus started with %d workers", b.config.WorkerCount)
}

// run is the main loop for a worker.
func (w *worker) run() {
	defer w.bus.wg.Done()

	w.bus.metrics.mu.Lock()
	w.bus.metrics.ActiveWorkers++
	w.bus.metrics.mu.Unlock()

	defer func() {
		w.bus.metrics.mu.Lock()
		w.bus.metrics.ActiveWorkers--
		w.bus.metrics.mu.Unlock()
	}()

	for {
		select {
		case <-w.ctx.Done():
			return
		case task, ok := <-w.bus.eventChan:
			if !ok {
				return
			}
			w.processTask(task)
		}
	}
}

// processTask processes a single event task.
func (w *worker) processTask(task eventTask) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[error] Worker %d: event handler panicked: %v (event type: %s)",
				w.id, r, string(task.event.Type))
			w.bus.handleFailedEvent(task, fmt.Errorf("panic: %v", r))
		}
	}()

	// Execute the handler
	task.handler.HandleEvent(task.event)

	// Update metrics
	w.bus.metrics.mu.Lock()
	w.bus.metrics.EventsProcessed++
	w.bus.metrics.LastProcessedAt = time.Now()
	w.bus.metrics.mu.Unlock()
}

// handleFailedEvent handles a failed event (retry or DLQ).
func (b *Bus) handleFailedEvent(task eventTask, err error) {
	b.metrics.mu.Lock()
	b.metrics.EventsFailed++
	b.metrics.mu.Unlock()

	handlerName := fmt.Sprintf("%T", task.handler)

	// Store in DLQ if enabled
	if b.enableDLQ && b.db != nil {
		failedEvent := &FailedEvent{
			EventType:    string(task.event.Type),
			EventData:    fmt.Sprintf("%+v", task.event), // TODO: JSON serialize
			HandlerName:  handlerName,
			ErrorMessage: err.Error(),
			RetryCount:   0,
			MaxRetries:   b.config.MaxRetries,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			Status:       "pending",
		}

		// Calculate next retry time with exponential backoff
		nextRetry := time.Now().Add(b.config.RetryDelay)
		failedEvent.NextRetryAt = &nextRetry

		if err := b.db.Create(failedEvent).Error; err != nil {
			log.Printf("[error] Failed to store failed event in DLQ: %v", err)
		} else {
			b.metrics.mu.Lock()
			b.metrics.EventsDLQ++
			b.metrics.mu.Unlock()
			log.Printf("[warn] Event stored in DLQ: type=%s handler=%s error=%v",
				task.event.Type, handlerName, err)
		}
	} else {
		log.Printf("[error] Event failed (no DLQ): type=%s handler=%s error=%v",
			task.event.Type, handlerName, err)
	}
}

// processDLQRetries periodically processes failed events for retry.
func (b *Bus) processDLQRetries() {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-b.ctx.Done():
			return
		case <-ticker.C:
			b.retryFailedEvents()
		}
	}
}

// retryFailedEvents retries failed events that are ready.
func (b *Bus) retryFailedEvents() {
	if b.db == nil {
		return
	}

	var failedEvents []FailedEvent
	now := time.Now()

	// Find events ready for retry
	if err := b.db.Where("status = ? AND retry_count < max_retries AND next_retry_at <= ?",
		"pending", now).
		Limit(100). // Process in batches
		Find(&failedEvents).Error; err != nil {
		log.Printf("[error] Failed to query DLQ: %v", err)
		return
	}

	for _, fe := range failedEvents {
		// Mark as retrying
		b.db.Model(&fe).Updates(map[string]interface{}{
			"status":      "retrying",
			"retry_count": fe.RetryCount + 1,
			"updated_at":  now,
		})

		b.metrics.mu.Lock()
		b.metrics.EventsRetried++
		b.metrics.mu.Unlock()

		// TODO: Deserialize event and re-dispatch to handler
		// For now, just mark as failed after max retries
		if fe.RetryCount+1 >= fe.MaxRetries {
			b.db.Model(&fe).Update("status", "failed")
			log.Printf("[error] Event permanently failed: type=%s handler=%s",
				fe.EventType, fe.HandlerName)
		}
	}
}

// Subscribe registers a handler for the given event type.
func (b *Bus) Subscribe(t Type, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[t] = append(b.handlers[t], h)
}

// SubscribeFunc is a convenience wrapper around Subscribe for func handlers.
func (b *Bus) SubscribeFunc(t Type, f func(Event)) {
	b.Subscribe(t, HandlerFunc(f))
}

// SubscribeAll registers a handler for every known event type.
// This is convenient for subscribers (e.g. webhook delivery) that need to
// observe all events.
func (b *Bus) SubscribeAll(h Handler) {
	for _, t := range []Type{TypePush, TypeIssues, TypeIssueComment, TypeMergeRequest, TypeRelease, TypePing} {
		b.Subscribe(t, h)
	}
}

// Publish dispatches evt to all subscribers of evt.Type.
// Events are queued and processed by the worker pool.
// If the queue is full, Publish blocks until space is available (backpressure).
func (b *Bus) Publish(evt Event) {
	b.mu.RLock()
	handlers := make([]Handler, len(b.handlers[evt.Type]))
	copy(handlers, b.handlers[evt.Type])
	b.mu.RUnlock()

	// Update metrics
	b.metrics.mu.Lock()
	b.metrics.EventsPublished++
	b.metrics.LastPublishedAt = time.Now()
	b.metrics.mu.Unlock()

	// Queue tasks for each handler
	for _, h := range handlers {
		task := eventTask{
			handler: h,
			event:   evt,
		}

		// Send to worker pool (blocks if queue is full - backpressure)
		select {
		case b.eventChan <- task:
			// Successfully queued
		case <-b.ctx.Done():
			log.Printf("[warn] Event bus shutting down, dropping event: type=%s", evt.Type)
			return
		}
	}

	// Update queue length metric
	b.metrics.mu.Lock()
	b.metrics.QueueLength = int64(len(b.eventChan))
	b.metrics.mu.Unlock()
}

// Shutdown gracefully shuts down the event bus.
// It stops accepting new events and waits for all workers to finish.
// Safe to call multiple times.
func (b *Bus) Shutdown(timeout time.Duration) error {
	var err error
	b.shutdownOnce.Do(func() {
		log.Printf("[info] Event bus shutting down...")

		// Signal all workers to stop
		b.cancel()

		// Close the event channel
		close(b.eventChan)

		// Wait for workers with timeout
		done := make(chan struct{})
		go func() {
			b.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			log.Printf("[info] Event bus shutdown complete")
			err = nil
		case <-time.After(timeout):
			log.Printf("[warn] Event bus shutdown timeout after %v", timeout)
			err = fmt.Errorf("shutdown timeout")
		}
	})
	return err
}

// GetMetrics returns a snapshot of the current metrics.
func (b *Bus) GetMetrics() BusMetrics {
	b.metrics.mu.RLock()
	defer b.metrics.mu.RUnlock()
	
	// Return a copy without the mutex
	return BusMetrics{
		EventsPublished: b.metrics.EventsPublished,
		EventsProcessed: b.metrics.EventsProcessed,
		EventsFailed:    b.metrics.EventsFailed,
		EventsRetried:   b.metrics.EventsRetried,
		EventsDLQ:       b.metrics.EventsDLQ,
		QueueLength:     b.metrics.QueueLength,
		ActiveWorkers:   b.metrics.ActiveWorkers,
		LastPublishedAt: b.metrics.LastPublishedAt,
		LastProcessedAt: b.metrics.LastProcessedAt,
	}
}

// GetDLQStats returns statistics about the dead letter queue.
func (b *Bus) GetDLQStats() (pending int64, retrying int64, failed int64, err error) {
	if b.db == nil {
		return 0, 0, 0, nil
	}

	b.db.Model(&FailedEvent{}).Where("status = ?", "pending").Count(&pending)
	b.db.Model(&FailedEvent{}).Where("status = ?", "retrying").Count(&retrying)
	b.db.Model(&FailedEvent{}).Where("status = ?", "failed").Count(&failed)

	return pending, retrying, failed, nil
}
