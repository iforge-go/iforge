package event

import (
	"sync"
	"testing"
	"time"
)

func TestBus_SubscribeAndPublish(t *testing.T) {
	bus := NewBus()
	defer bus.Shutdown(time.Second)

	var received Event
	var wg sync.WaitGroup
	wg.Add(1)

	bus.SubscribeFunc(TypePush, func(evt Event) {
		received = evt
		wg.Done()
	})

	evt := Event{
		Type:   TypePush,
		Action: "created",
	}

	bus.Publish(evt)

	// Wait for async processing
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for event")
	}

	if received.Type != TypePush {
		t.Errorf("expected TypePush, got %v", received.Type)
	}
	if received.Action != "created" {
		t.Errorf("expected action 'created', got %v", received.Action)
	}
}

func TestBus_MultipleSubscribers(t *testing.T) {
	bus := NewBus()
	defer bus.Shutdown(time.Second)

	count := 0
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(3)

	for i := 0; i < 3; i++ {
		bus.SubscribeFunc(TypeIssues, func(evt Event) {
			mu.Lock()
			count++
			mu.Unlock()
			wg.Done()
		})
	}

	bus.Publish(Event{Type: TypeIssues})

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for events")
	}

	mu.Lock()
	if count != 3 {
		t.Errorf("expected 3 handlers called, got %d", count)
	}
	mu.Unlock()
}

func TestBus_SubscribeAll(t *testing.T) {
	bus := NewBus()
	defer bus.Shutdown(time.Second)

	types := make(map[Type]bool)
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(4)

	bus.SubscribeAll(HandlerFunc(func(evt Event) {
		mu.Lock()
		types[evt.Type] = true
		mu.Unlock()
		wg.Done()
	}))

	bus.Publish(Event{Type: TypePush})
	bus.Publish(Event{Type: TypeIssues})
	bus.Publish(Event{Type: TypeMergeRequest})
	bus.Publish(Event{Type: TypeRelease})

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for events")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(types) != 4 {
		t.Errorf("expected 4 event types, got %d", len(types))
	}
}

func TestBus_GetMetrics(t *testing.T) {
	bus := NewBus()
	defer bus.Shutdown(time.Second)

	var wg sync.WaitGroup
	wg.Add(2)

	bus.SubscribeFunc(TypePush, func(evt Event) {
		wg.Done()
	})

	bus.Publish(Event{Type: TypePush})
	bus.Publish(Event{Type: TypePush})

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for events")
	}

	metrics := bus.GetMetrics()
	if metrics.EventsPublished != 2 {
		t.Errorf("expected 2 events published, got %d", metrics.EventsPublished)
	}
	if metrics.EventsProcessed != 2 {
		t.Errorf("expected 2 events processed, got %d", metrics.EventsProcessed)
	}
}

func TestBus_Shutdown(t *testing.T) {
	bus := NewBus()

	// Shutdown should complete within timeout
	err := bus.Shutdown(2 * time.Second)
	if err != nil {
		t.Errorf("shutdown failed: %v", err)
	}

	// Second shutdown should also work
	err = bus.Shutdown(time.Second)
	if err != nil {
		t.Errorf("second shutdown failed: %v", err)
	}
}

func TestDefaultBusConfig(t *testing.T) {
	config := DefaultBusConfig()

	if config.WorkerCount != 10 {
		t.Errorf("expected WorkerCount 10, got %d", config.WorkerCount)
	}
	if config.QueueSize != 1000 {
		t.Errorf("expected QueueSize 1000, got %d", config.QueueSize)
	}
	if config.MaxRetries != 3 {
		t.Errorf("expected MaxRetries 3, got %d", config.MaxRetries)
	}
	if config.RetryDelay != time.Second {
		t.Errorf("expected RetryDelay 1s, got %v", config.RetryDelay)
	}
	if !config.EnableDLQ {
		t.Error("expected EnableDLQ to be true")
	}
}

func TestBusConfig_Defaults(t *testing.T) {
	tests := []struct {
		name     string
		config   BusConfig
		expected BusConfig
	}{
		{
			name:   "zero values",
			config: BusConfig{},
			expected: BusConfig{
				WorkerCount: 10,
				QueueSize:   1000,
				MaxRetries:  3,
				RetryDelay:  time.Second,
			},
		},
		{
			name: "custom values",
			config: BusConfig{
				WorkerCount: 5,
				QueueSize:   500,
				MaxRetries:  5,
				RetryDelay:  2 * time.Second,
			},
			expected: BusConfig{
				WorkerCount: 5,
				QueueSize:   500,
				MaxRetries:  5,
				RetryDelay:  2 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus := NewBusWithConfig(tt.config)
			defer bus.Shutdown(time.Second)

			if bus.config.WorkerCount != tt.expected.WorkerCount {
				t.Errorf("WorkerCount: expected %d, got %d", tt.expected.WorkerCount, bus.config.WorkerCount)
			}
			if bus.config.QueueSize != tt.expected.QueueSize {
				t.Errorf("QueueSize: expected %d, got %d", tt.expected.QueueSize, bus.config.QueueSize)
			}
			if bus.config.MaxRetries != tt.expected.MaxRetries {
				t.Errorf("MaxRetries: expected %d, got %d", tt.expected.MaxRetries, bus.config.MaxRetries)
			}
		})
	}
}

func TestHandlerFunc(t *testing.T) {
	called := false
	h := HandlerFunc(func(evt Event) {
		called = true
	})

	h.HandleEvent(Event{})

	if !called {
		t.Error("HandlerFunc was not called")
	}
}

func BenchmarkBus_Publish(b *testing.B) {
	bus := NewBus()
	defer bus.Shutdown(time.Second)

	bus.SubscribeFunc(TypePush, func(evt Event) {})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Publish(Event{Type: TypePush})
	}
}

func BenchmarkBus_PublishMultipleHandlers(b *testing.B) {
	bus := NewBus()
	defer bus.Shutdown(time.Second)

	for i := 0; i < 10; i++ {
		bus.SubscribeFunc(TypePush, func(evt Event) {})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Publish(Event{Type: TypePush})
	}
}
