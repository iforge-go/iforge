package service

import (
	"testing"
	"time"
)

func TestOIDCService_SaveState(t *testing.T) {
	t.Run("valid state", func(t *testing.T) {
		service := NewOIDCService(nil)
		service.SaveState("test-state-123")

		if _, exists := service.stateStore["test-state-123"]; !exists {
			t.Error("State should be saved in stateStore")
		}
	})

	t.Run("empty state", func(t *testing.T) {
		service := NewOIDCService(nil)
		initialLen := len(service.stateStore)
		service.SaveState("")

		if len(service.stateStore) != initialLen {
			t.Error("Empty state should not be saved")
		}
	})
}

func TestOIDCService_ConsumeState(t *testing.T) {
	service := NewOIDCService(nil)

	// Test consuming non-existent state
	if service.ConsumeState("non-existent") {
		t.Error("Consuming non-existent state should return false")
	}

	// Test consuming empty state
	if service.ConsumeState("") {
		t.Error("Consuming empty state should return false")
	}

	// Test consuming valid state
	service.SaveState("valid-state")
	if !service.ConsumeState("valid-state") {
		t.Error("Consuming valid state should return true")
	}

	// Test consuming already consumed state
	if service.ConsumeState("valid-state") {
		t.Error("Consuming already consumed state should return false")
	}
}

func TestOIDCService_ConsumeState_Expired(t *testing.T) {
	service := NewOIDCService(nil)

	// Save a state
	service.SaveState("test-state")

	// Manually expire it
	service.stateMu.Lock()
	service.stateStore["test-state"] = time.Now().Add(-1 * time.Minute)
	service.stateMu.Unlock()

	// Consuming expired state should return false
	if service.ConsumeState("test-state") {
		t.Error("Consuming expired state should return false")
	}
}

func TestOIDCService_SaveState_GarbageCollection(t *testing.T) {
	service := NewOIDCService(nil)

	// Add some expired states
	service.stateMu.Lock()
	service.stateStore["expired1"] = time.Now().Add(-10 * time.Minute)
	service.stateStore["expired2"] = time.Now().Add(-5 * time.Minute)
	service.stateStore["valid"] = time.Now().Add(5 * time.Minute)
	service.stateMu.Unlock()

	// Save a new state, which should trigger garbage collection
	service.SaveState("new-state")

	// Check that expired states were removed
	service.stateMu.Lock()
	defer service.stateMu.Unlock()

	if _, exists := service.stateStore["expired1"]; exists {
		t.Error("Expired state 'expired1' should be removed")
	}
	if _, exists := service.stateStore["expired2"]; exists {
		t.Error("Expired state 'expired2' should be removed")
	}
	if _, exists := service.stateStore["valid"]; !exists {
		t.Error("Valid state 'valid' should still exist")
	}
	if _, exists := service.stateStore["new-state"]; !exists {
		t.Error("New state 'new-state' should exist")
	}
}

func TestOIDCService_NewOIDCService(t *testing.T) {
	settingsService := &SystemSettingsService{}
	service := NewOIDCService(settingsService)

	if service == nil {
		t.Fatal("Expected OIDCService to be created")
	}
	if service.settingsService != settingsService {
		t.Error("Expected settingsService to be set")
	}
	if service.stateStore == nil {
		t.Error("Expected stateStore to be initialized")
	}
	if service.provider != nil {
		t.Error("Expected provider to be nil initially")
	}
	if service.verifier != nil {
		t.Error("Expected verifier to be nil initially")
	}
	if service.oauth2Config != nil {
		t.Error("Expected oauth2Config to be nil initially")
	}
}

func TestOIDCService_StateConcurrency(t *testing.T) {
	service := NewOIDCService(nil)

	// Test concurrent state operations
	done := make(chan bool, 10)

	// Save states concurrently
	for i := 0; i < 5; i++ {
		go func(id int) {
			service.SaveState("state-" + string(rune('0'+id)))
			done <- true
		}(i)
	}

	// Consume states concurrently
	for i := 0; i < 5; i++ {
		go func(id int) {
			service.ConsumeState("state-" + string(rune('0'+id)))
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic or deadlock
}
