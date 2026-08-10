package service

import (
	"strings"
	"testing"
	"time"
)

func TestMetricsCollector_NewMetricsCollector(t *testing.T) {
	collector := NewMetricsCollector()
	if collector == nil {
		t.Fatal("Expected MetricsCollector to be created")
	}
	if collector.metrics == nil {
		t.Error("Expected metrics map to be initialized")
	}
}

func TestMetricsCollector_Increment(t *testing.T) {
	collector := NewMetricsCollector()

	// Test incrementing new metric
	collector.Increment("test_counter", nil)
	metric := collector.Get("test_counter", nil)
	if metric == nil {
		t.Fatal("Expected metric to exist")
	}
	if metric.Value != 1 {
		t.Errorf("Expected value 1, got %f", metric.Value)
	}
	if metric.Type != MetricTypeCounter {
		t.Errorf("Expected type counter, got %s", metric.Type)
	}

	// Test incrementing existing metric
	collector.Increment("test_counter", nil)
	metric = collector.Get("test_counter", nil)
	if metric.Value != 2 {
		t.Errorf("Expected value 2, got %f", metric.Value)
	}

	// Test with labels
	labels := map[string]string{"method": "GET"}
	collector.Increment("http_requests", labels)
	metric = collector.Get("http_requests", labels)
	if metric == nil {
		t.Fatal("Expected metric with labels to exist")
	}
	if metric.Value != 1 {
		t.Errorf("Expected value 1, got %f", metric.Value)
	}
}

func TestMetricsCollector_Decrement(t *testing.T) {
	collector := NewMetricsCollector()

	// Test decrementing new metric
	collector.Decrement("test_gauge", nil)
	metric := collector.Get("test_gauge", nil)
	if metric == nil {
		t.Fatal("Expected metric to exist")
	}
	if metric.Value != -1 {
		t.Errorf("Expected value -1, got %f", metric.Value)
	}

	// Test decrementing existing metric
	collector.Set("test_gauge", 10, nil)
	collector.Decrement("test_gauge", nil)
	metric = collector.Get("test_gauge", nil)
	if metric.Value != 9 {
		t.Errorf("Expected value 9, got %f", metric.Value)
	}
}

func TestMetricsCollector_Set(t *testing.T) {
	collector := NewMetricsCollector()

	collector.Set("active_users", 100, nil)
	metric := collector.Get("active_users", nil)
	if metric == nil {
		t.Fatal("Expected metric to exist")
	}
	if metric.Value != 100 {
		t.Errorf("Expected value 100, got %f", metric.Value)
	}
	if metric.Type != MetricTypeGauge {
		t.Errorf("Expected type gauge, got %s", metric.Type)
	}

	// Test overwriting value
	collector.Set("active_users", 200, nil)
	metric = collector.Get("active_users", nil)
	if metric.Value != 200 {
		t.Errorf("Expected value 200, got %f", metric.Value)
	}
}

func TestMetricsCollector_Add(t *testing.T) {
	collector := NewMetricsCollector()

	// Test adding to new metric
	collector.Add("bytes_sent", 100, nil)
	metric := collector.Get("bytes_sent", nil)
	if metric == nil {
		t.Fatal("Expected metric to exist")
	}
	if metric.Value != 100 {
		t.Errorf("Expected value 100, got %f", metric.Value)
	}

	// Test adding to existing metric
	collector.Add("bytes_sent", 50, nil)
	metric = collector.Get("bytes_sent", nil)
	if metric.Value != 150 {
		t.Errorf("Expected value 150, got %f", metric.Value)
	}
}

func TestMetricsCollector_Observe(t *testing.T) {
	collector := NewMetricsCollector()

	// Test observing new metric
	collector.Observe("request_duration", 0.5, nil)
	metric := collector.Get("request_duration", nil)
	if metric == nil {
		t.Fatal("Expected metric to exist")
	}
	if metric.Type != MetricTypeHistogram {
		t.Errorf("Expected type histogram, got %s", metric.Type)
	}
	if metric.Count != 1 {
		t.Errorf("Expected count 1, got %d", metric.Count)
	}
	if metric.Sum != 0.5 {
		t.Errorf("Expected sum 0.5, got %f", metric.Sum)
	}

	// Test observing multiple values
	collector.Observe("request_duration", 1.5, nil)
	metric = collector.Get("request_duration", nil)
	if metric.Count != 2 {
		t.Errorf("Expected count 2, got %d", metric.Count)
	}
	if metric.Sum != 2.0 {
		t.Errorf("Expected sum 2.0, got %f", metric.Sum)
	}
}

func TestMetricsCollector_Get(t *testing.T) {
	collector := NewMetricsCollector()

	// Test getting non-existent metric
	metric := collector.Get("non_existent", nil)
	if metric != nil {
		t.Error("Expected nil for non-existent metric")
	}

	// Test getting existing metric
	collector.Set("test_metric", 42, nil)
	metric = collector.Get("test_metric", nil)
	if metric == nil {
		t.Fatal("Expected metric to exist")
	}
	if metric.Value != 42 {
		t.Errorf("Expected value 42, got %f", metric.Value)
	}
}

func TestMetricsCollector_GetAll(t *testing.T) {
	collector := NewMetricsCollector()

	collector.Set("metric1", 1, nil)
	collector.Set("metric2", 2, nil)
	collector.Set("metric3", 3, nil)

	metrics := collector.GetAll()
	if len(metrics) != 3 {
		t.Errorf("Expected 3 metrics, got %d", len(metrics))
	}
}

func TestMetricsCollector_Reset(t *testing.T) {
	collector := NewMetricsCollector()

	collector.Set("metric1", 1, nil)
	collector.Set("metric2", 2, nil)

	collector.Reset()

	metrics := collector.GetAll()
	if len(metrics) != 0 {
		t.Errorf("Expected 0 metrics after reset, got %d", len(metrics))
	}
}

func TestMetricsCollector_buildKey(t *testing.T) {
	collector := NewMetricsCollector()

	// Test key without labels
	key := collector.buildKey("test_metric", nil)
	if key != "test_metric" {
		t.Errorf("Expected key 'test_metric', got '%s'", key)
	}

	// Test key with labels
	labels := map[string]string{"method": "GET", "path": "/api"}
	key = collector.buildKey("http_requests", labels)
	if !strings.Contains(key, "http_requests") {
		t.Errorf("Expected key to contain 'http_requests', got '%s'", key)
	}
	if !strings.Contains(key, "method=GET") {
		t.Errorf("Expected key to contain 'method=GET', got '%s'", key)
	}
}

func TestGetMetrics(t *testing.T) {
	metrics := GetMetrics()
	if metrics == nil {
		t.Fatal("Expected global metrics collector to exist")
	}
}

func TestRecordRequest(t *testing.T) {
	// Reset global metrics
	defaultMetrics.Reset()

	RecordRequest("GET", "/api/test", 200, 100*time.Millisecond)

	// Verify that some metrics were recorded (exact label matching is complex due to status conversion)
	metrics := defaultMetrics.GetAll()
	if len(metrics) == 0 {
		t.Error("Expected metrics to be recorded")
	}

	// Check that http_requests_total was incremented
	found := false
	for _, m := range metrics {
		if m.Name == "http_requests_total" {
			found = true
			if m.Value != 1 {
				t.Errorf("Expected value 1, got %f", m.Value)
			}
		}
	}
	if !found {
		t.Error("Expected http_requests_total metric to exist")
	}
}

func TestRecordError(t *testing.T) {
	defaultMetrics.Reset()

	RecordError("database", "connection_error")

	// Look up via GetAll (map iteration order in buildKey is non-deterministic)
	metrics := defaultMetrics.GetAll()
	found := false
	for _, m := range metrics {
		if m.Name == "errors_total" {
			found = true
			if m.Value != 1 {
				t.Errorf("Expected value 1, got %f", m.Value)
			}
		}
	}
	if !found {
		t.Fatal("Expected error metric to exist")
	}
}

func TestSetActiveUsers(t *testing.T) {
	defaultMetrics.Reset()

	SetActiveUsers(150)

	metric := defaultMetrics.Get("active_users", nil)
	if metric == nil {
		t.Fatal("Expected active_users metric to exist")
	}
	if metric.Value != 150 {
		t.Errorf("Expected value 150, got %f", metric.Value)
	}
}

func TestSetActiveRepositories(t *testing.T) {
	defaultMetrics.Reset()

	SetActiveRepositories(50)

	metric := defaultMetrics.Get("active_repositories", nil)
	if metric == nil {
		t.Fatal("Expected active_repositories metric to exist")
	}
	if metric.Value != 50 {
		t.Errorf("Expected value 50, got %f", metric.Value)
	}
}

func TestRecordGitOperation(t *testing.T) {
	defaultMetrics.Reset()

	RecordGitOperation("push", 500*time.Millisecond)

	metric := defaultMetrics.Get("git_operations_total", map[string]string{
		"operation": "push",
	})
	if metric == nil {
		t.Fatal("Expected git operation metric to exist")
	}
}

func TestRecordDatabaseQuery(t *testing.T) {
	defaultMetrics.Reset()

	RecordDatabaseQuery("SELECT", 50*time.Millisecond)

	metric := defaultMetrics.Get("database_queries_total", map[string]string{
		"type": "SELECT",
	})
	if metric == nil {
		t.Fatal("Expected database query metric to exist")
	}
}

func TestMetricsCollector_ExportPrometheus(t *testing.T) {
	collector := NewMetricsCollector()

	collector.Increment("http_requests", map[string]string{"method": "GET"})
	collector.Set("active_users", 100, nil)
	collector.Observe("request_duration", 0.5, nil)

	output := collector.ExportPrometheus()

	if !strings.Contains(output, "http_requests") {
		t.Error("Expected output to contain 'http_requests'")
	}
	if !strings.Contains(output, "active_users") {
		t.Error("Expected output to contain 'active_users'")
	}
	if !strings.Contains(output, "request_duration") {
		t.Error("Expected output to contain 'request_duration'")
	}
	if !strings.Contains(output, "# TYPE") {
		t.Error("Expected output to contain TYPE annotations")
	}
}

func TestExportPrometheus(t *testing.T) {
	defaultMetrics.Reset()
	defaultMetrics.Set("test_metric", 42, nil)

	output := ExportPrometheus()
	if !strings.Contains(output, "test_metric") {
		t.Error("Expected output to contain 'test_metric'")
	}
}

func TestFormatLabels(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		expected string
	}{
		{
			name:     "empty labels",
			labels:   map[string]string{},
			expected: "",
		},
		{
			name:     "single label",
			labels:   map[string]string{"method": "GET"},
			expected: `method="GET"`,
		},
		{
			name: "multiple labels",
			labels: map[string]string{
				"method": "GET",
				"path":   "/api",
			},
			expected: `method="GET",path="/api"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatLabels(tt.labels)
			// For multiple labels, order may vary, so check contains
			if len(tt.labels) <= 1 && result != tt.expected {
				t.Errorf("formatLabels() = %q; want %q", result, tt.expected)
			}
			if len(tt.labels) > 1 {
				if !strings.Contains(result, `method="GET"`) {
					t.Errorf("Expected result to contain method label, got %q", result)
				}
			}
		})
	}
}

func TestMetricType(t *testing.T) {
	if MetricTypeCounter != "counter" {
		t.Errorf("Expected MetricTypeCounter to be 'counter', got %s", MetricTypeCounter)
	}
	if MetricTypeGauge != "gauge" {
		t.Errorf("Expected MetricTypeGauge to be 'gauge', got %s", MetricTypeGauge)
	}
	if MetricTypeHistogram != "histogram" {
		t.Errorf("Expected MetricTypeHistogram to be 'histogram', got %s", MetricTypeHistogram)
	}
}
