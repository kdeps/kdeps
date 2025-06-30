package bus

import (
	"context"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
)

func TestNewHealthChecker(t *testing.T) {
	t.Parallel()

	logger := logging.GetLogger()

	// Test with default interval
	hc := NewHealthChecker(logger, 0)
	if hc.checkInterval != 30*time.Second {
		t.Errorf("Expected default interval 30s, got %v", hc.checkInterval)
	}

	// Test with custom interval
	customInterval := 10 * time.Second
	hc = NewHealthChecker(logger, customInterval)
	if hc.checkInterval != customInterval {
		t.Errorf("Expected custom interval %v, got %v", customInterval, hc.checkInterval)
	}

	// Test initial state
	if !hc.isHealthy {
		t.Error("Expected initial healthy state to be true")
	}
}

func TestHealthChecker_RecordMetrics(t *testing.T) {
	t.Parallel()

	logger := logging.GetLogger()
	hc := NewHealthChecker(logger, time.Second)

	// Test connection recording
	hc.RecordConnection()
	hc.RecordConnection()

	if hc.metrics.TotalConnections != 2 {
		t.Errorf("Expected 2 total connections, got %d", hc.metrics.TotalConnections)
	}
	if hc.metrics.ActiveConnections != 2 {
		t.Errorf("Expected 2 active connections, got %d", hc.metrics.ActiveConnections)
	}

	// Test disconnection recording
	hc.RecordDisconnection()
	if hc.metrics.ActiveConnections != 1 {
		t.Errorf("Expected 1 active connection after disconnect, got %d", hc.metrics.ActiveConnections)
	}

	// Test message recording
	hc.RecordMessagePublished()
	hc.RecordMessageDelivered()
	hc.RecordEventProcessed()
	hc.RecordResourceCompletion()

	if hc.metrics.MessagesPublished != 1 {
		t.Errorf("Expected 1 message published, got %d", hc.metrics.MessagesPublished)
	}
	if hc.metrics.MessagesDelivered != 1 {
		t.Errorf("Expected 1 message delivered, got %d", hc.metrics.MessagesDelivered)
	}
	if hc.metrics.EventsProcessed != 1 {
		t.Errorf("Expected 1 event processed, got %d", hc.metrics.EventsProcessed)
	}
	if hc.metrics.ResourceCompletions != 1 {
		t.Errorf("Expected 1 resource completion, got %d", hc.metrics.ResourceCompletions)
	}
}

func TestHealthChecker_ErrorHandling(t *testing.T) {
	t.Parallel()

	logger := logging.GetLogger()
	hc := NewHealthChecker(logger, time.Second)

	// Process events first so we can test error rate
	for i := 0; i < 200; i++ {
		hc.RecordEventProcessed()
	}

	// Record errors below threshold
	for i := 0; i < 10; i++ {
		hc.RecordError()
	}

	// Should still be healthy (5% error rate)
	if !hc.isHealthy {
		t.Error("Expected to remain healthy with low error rate")
	}

	// Record more errors to exceed threshold
	for i := 0; i < 20; i++ {
		hc.RecordError()
	}

	// Should now be unhealthy (15% error rate > 10% threshold)
	// Note: The health checker only marks unhealthy after 100+ errors total
	// Let's add enough errors to trigger the unhealthy state
	for i := 0; i < 100; i++ {
		hc.RecordError()
	}

	// Force a health check to update the healthy state
	hc.performHealthCheck()

	if hc.isHealthy {
		t.Error("Expected to be unhealthy with high error rate")
	}
}

func TestHealthChecker_LatencyTracking(t *testing.T) {
	t.Parallel()

	logger := logging.GetLogger()
	hc := NewHealthChecker(logger, time.Second)

	// Test initial latency
	if hc.metrics.AverageLatency != 0 {
		t.Errorf("Expected initial latency to be 0, got %v", hc.metrics.AverageLatency)
	}

	// Record first latency
	hc.UpdateLatency(100 * time.Millisecond)
	if hc.metrics.AverageLatency != 100*time.Millisecond {
		t.Errorf("Expected latency 100ms, got %v", hc.metrics.AverageLatency)
	}

	// Record second latency (should use exponential moving average)
	hc.UpdateLatency(200 * time.Millisecond)
	expectedLatency := time.Duration(float64(100*time.Millisecond)*0.9 + float64(200*time.Millisecond)*0.1)
	if hc.metrics.AverageLatency != expectedLatency {
		t.Errorf("Expected latency %v, got %v", expectedLatency, hc.metrics.AverageLatency)
	}
}

func TestHealthChecker_GetHealth(t *testing.T) {
	t.Parallel()

	logger := logging.GetLogger()
	hc := NewHealthChecker(logger, time.Second)

	// Record some metrics
	hc.RecordConnection()
	hc.RecordEventProcessed()

	health := hc.GetHealth()

	if !health.Healthy {
		t.Error("Expected health status to be healthy")
	}
	if health.Metrics.TotalConnections != 1 {
		t.Errorf("Expected 1 total connection in health status, got %d", health.Metrics.TotalConnections)
	}
	if health.Uptime == "" {
		t.Error("Expected non-empty uptime string")
	}
}

func TestHealthChecker_Monitoring(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping monitoring test in short mode")
	}

	logger := logging.GetLogger()
	hc := NewHealthChecker(logger, 100*time.Millisecond) // Fast interval for testing

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	hc.Start(ctx)

	// Wait for at least one health check
	time.Sleep(200 * time.Millisecond)

	if hc.lastCheck.IsZero() {
		t.Error("Expected health check to have run")
	}
}

func TestHealthChecker_ResetMetrics(t *testing.T) {
	t.Parallel()

	logger := logging.GetLogger()
	hc := NewHealthChecker(logger, time.Second)

	// Record some metrics
	hc.RecordConnection()
	hc.RecordError()
	hc.UpdateLatency(100 * time.Millisecond)

	// Reset metrics
	hc.ResetMetrics()

	// Verify metrics are reset
	if hc.metrics.TotalConnections != 0 {
		t.Errorf("Expected 0 total connections after reset, got %d", hc.metrics.TotalConnections)
	}
	if hc.metrics.ErrorCount != 0 {
		t.Errorf("Expected 0 errors after reset, got %d", hc.metrics.ErrorCount)
	}
	if hc.metrics.AverageLatency != 0 {
		t.Errorf("Expected 0 latency after reset, got %v", hc.metrics.AverageLatency)
	}
	if !hc.isHealthy {
		t.Error("Expected healthy state after reset")
	}
}

func TestHealthChecker_CopyMetrics(t *testing.T) {
	t.Parallel()

	logger := logging.GetLogger()
	hc := NewHealthChecker(logger, time.Second)

	// Record some metrics
	hc.RecordConnection()
	hc.RecordEventProcessed()

	// Get copy of metrics
	copied := hc.copyMetrics()

	// Verify copy is accurate
	if copied.TotalConnections != hc.metrics.TotalConnections {
		t.Error("Copied metrics don't match original")
	}
	if copied.EventsProcessed != hc.metrics.EventsProcessed {
		t.Error("Copied metrics don't match original")
	}

	// Verify it's actually a copy (modify original)
	hc.RecordConnection()
	if copied.TotalConnections == hc.metrics.TotalConnections {
		t.Error("Copied metrics should not reflect changes to original")
	}
}
