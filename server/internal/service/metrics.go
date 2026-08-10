package service

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// MetricType represents the type of metric
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
)

// Metric represents a single metric
type Metric struct {
	Name      string            `json:"name"`
	Type      MetricType        `json:"type"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
	// Histogram fields
	Count   int64     `json:"count,omitempty"`
	Sum     float64   `json:"sum,omitempty"`
	Buckets []float64 `json:"buckets,omitempty"`
}

// MetricsCollector collects and manages metrics
type MetricsCollector struct {
	mu      sync.RWMutex
	metrics map[string]*Metric
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: make(map[string]*Metric),
	}
}

// Increment increments a counter metric
func (c *MetricsCollector) Increment(name string, labels map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.buildKey(name, labels)
	if m, exists := c.metrics[key]; exists {
		m.Value++
		m.Timestamp = time.Now()
	} else {
		c.metrics[key] = &Metric{
			Name:      name,
			Type:      MetricTypeCounter,
			Value:     1,
			Labels:    labels,
			Timestamp: time.Now(),
		}
	}
}

// Decrement decrements a gauge metric
func (c *MetricsCollector) Decrement(name string, labels map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.buildKey(name, labels)
	if m, exists := c.metrics[key]; exists {
		m.Value--
		m.Timestamp = time.Now()
	} else {
		c.metrics[key] = &Metric{
			Name:      name,
			Type:      MetricTypeGauge,
			Value:     -1,
			Labels:    labels,
			Timestamp: time.Now(),
		}
	}
}

// Set sets a gauge metric to a specific value
func (c *MetricsCollector) Set(name string, value float64, labels map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.buildKey(name, labels)
	c.metrics[key] = &Metric{
		Name:      name,
		Type:      MetricTypeGauge,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
	}
}

// Add adds a value to a counter or gauge metric
func (c *MetricsCollector) Add(name string, value float64, labels map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.buildKey(name, labels)
	if m, exists := c.metrics[key]; exists {
		m.Value += value
		m.Timestamp = time.Now()
	} else {
		c.metrics[key] = &Metric{
			Name:      name,
			Type:      MetricTypeGauge,
			Value:     value,
			Labels:    labels,
			Timestamp: time.Now(),
		}
	}
}

// Observe records a value for a histogram metric
func (c *MetricsCollector) Observe(name string, value float64, labels map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.buildKey(name, labels)
	if m, exists := c.metrics[key]; exists {
		// For histogram, accumulate count and sum
		m.Count++
		m.Sum += value
		m.Value = value // Keep latest value for compatibility
		m.Timestamp = time.Now()
	} else {
		c.metrics[key] = &Metric{
			Name:      name,
			Type:      MetricTypeHistogram,
			Value:     value,
			Labels:    labels,
			Timestamp: time.Now(),
			Count:     1,
			Sum:       value,
		}
	}
}

// Get returns a metric by name and labels
func (c *MetricsCollector) Get(name string, labels map[string]string) *Metric {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := c.buildKey(name, labels)
	if m, exists := c.metrics[key]; exists {
		return m
	}
	return nil
}

// GetAll returns all metrics
func (c *MetricsCollector) GetAll() []*Metric {
	c.mu.RLock()
	defer c.mu.RUnlock()

	metrics := make([]*Metric, 0, len(c.metrics))
	for _, m := range c.metrics {
		metrics = append(metrics, m)
	}
	return metrics
}

// Reset resets all metrics
func (c *MetricsCollector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metrics = make(map[string]*Metric)
}

// buildKey builds a unique key for a metric
func (c *MetricsCollector) buildKey(name string, labels map[string]string) string {
	key := name
	if len(labels) > 0 {
		for k, v := range labels {
			key += "|" + k + "=" + v
		}
	}
	return key
}

// Global metrics collector
var defaultMetrics = NewMetricsCollector()

// GetMetrics returns the global metrics collector
func GetMetrics() *MetricsCollector {
	return defaultMetrics
}

// Common metrics
func RecordRequest(method, path string, statusCode int, duration time.Duration) {
	labels := map[string]string{
		"method": method,
		"path":   path,
		"status": string(rune(statusCode)),
	}
	defaultMetrics.Increment("http_requests_total", labels)
	defaultMetrics.Observe("http_request_duration_seconds", duration.Seconds(), labels)
}

func RecordError(module, errorType string) {
	labels := map[string]string{
		"module": module,
		"type":   errorType,
	}
	defaultMetrics.Increment("errors_total", labels)
}

func SetActiveUsers(count float64) {
	defaultMetrics.Set("active_users", count, nil)
}

func SetActiveRepositories(count float64) {
	defaultMetrics.Set("active_repositories", count, nil)
}

func RecordGitOperation(operation string, duration time.Duration) {
	labels := map[string]string{
		"operation": operation,
	}
	defaultMetrics.Increment("git_operations_total", labels)
	defaultMetrics.Observe("git_operation_duration_seconds", duration.Seconds(), labels)
}

func RecordDatabaseQuery(queryType string, duration time.Duration) {
	labels := map[string]string{
		"type": queryType,
	}
	defaultMetrics.Increment("database_queries_total", labels)
	defaultMetrics.Observe("database_query_duration_seconds", duration.Seconds(), labels)
}

// ExportPrometheus exports metrics in Prometheus text format
func (c *MetricsCollector) ExportPrometheus() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result string
	metricsByName := make(map[string][]*Metric)

	// Group metrics by name
	for _, m := range c.metrics {
		metricsByName[m.Name] = append(metricsByName[m.Name], m)
	}

	for name, metrics := range metricsByName {
		if len(metrics) == 0 {
			continue
		}

		// Determine metric type
		metricType := metrics[0].Type
		switch metricType {
		case MetricTypeCounter:
			result += fmt.Sprintf("# TYPE %s counter\n", name)
			for _, m := range metrics {
				result += formatPrometheusMetric(name, m.Value, m.Labels)
			}
		case MetricTypeGauge:
			result += fmt.Sprintf("# TYPE %s gauge\n", name)
			for _, m := range metrics {
				result += formatPrometheusMetric(name, m.Value, m.Labels)
			}
		case MetricTypeHistogram:
			result += fmt.Sprintf("# TYPE %s histogram\n", name)
			for _, m := range metrics {
				result += formatPrometheusHistogram(name, m)
			}
		}
	}

	return result
}

// formatPrometheusMetric formats a single metric line
func formatPrometheusMetric(name string, value float64, labels map[string]string) string {
	if len(labels) == 0 {
		return fmt.Sprintf("%s %f\n", name, value)
	}

	labelStr := formatLabels(labels)
	return fmt.Sprintf("%s{%s} %f\n", name, labelStr, value)
}

// formatPrometheusHistogram formats a histogram metric
func formatPrometheusHistogram(name string, m *Metric) string {
	var result string
	labelStr := formatLabels(m.Labels)

	// _bucket lines (simplified - just using count and sum)
	result += fmt.Sprintf("%s_bucket{%s,le=\"+Inf\"} %d\n", name, labelStr, m.Count)
	result += fmt.Sprintf("%s_sum{%s} %f\n", name, labelStr, m.Sum)
	result += fmt.Sprintf("%s_count{%s} %d\n", name, labelStr, m.Count)

	return result
}

// formatLabels formats labels as Prometheus label string
func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}

	var parts []string
	for k, v := range labels {
		parts = append(parts, fmt.Sprintf("%s=\"%s\"", k, v))
	}
	return strings.Join(parts, ",")
}

// ExportPrometheus exports metrics from the global collector in Prometheus format
func ExportPrometheus() string {
	return defaultMetrics.ExportPrometheus()
}
