package bus

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
)

// HealthChecker monitors bus service health and performance metrics
type HealthChecker struct {
	logger        *logging.Logger
	mu            sync.RWMutex
	metrics       *BusMetrics
	isHealthy     bool
	lastCheck     time.Time
	checkInterval time.Duration
}

// BusMetrics tracks performance and usage statistics
type BusMetrics struct {
	TotalConnections    int64
	ActiveConnections   int64
	MessagesPublished   int64
	MessagesDelivered   int64
	EventsProcessed     int64
	AverageLatency      time.Duration
	ErrorCount          int64
	ResourceCompletions int64
	UptimeStart         time.Time
}

// HealthStatus represents the current health state
type HealthStatus struct {
	Healthy      bool        `json:"healthy"`
	LastCheck    time.Time   `json:"last_check"`
	Uptime       string      `json:"uptime"`
	Metrics      *BusMetrics `json:"metrics"`
	ErrorMessage string      `json:"error_message,omitempty"`
}

// NewHealthChecker creates a new health monitoring system
func NewHealthChecker(logger *logging.Logger, checkInterval time.Duration) *HealthChecker {
	if checkInterval == 0 {
		checkInterval = 30 * time.Second
	}

	return &HealthChecker{
		logger:        logger,
		isHealthy:     true,
		checkInterval: checkInterval,
		metrics: &BusMetrics{
			UptimeStart: time.Now(),
		},
	}
}

// Start begins health monitoring in a background goroutine
func (h *HealthChecker) Start(ctx context.Context) {
	go h.monitorHealth(ctx)
	h.logger.Info("Bus health checker started", "interval", h.checkInterval)
}

// GetHealth returns current health status
func (h *HealthChecker) GetHealth() HealthStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()

	uptime := time.Since(h.metrics.UptimeStart).Round(time.Second)

	return HealthStatus{
		Healthy:   h.isHealthy,
		LastCheck: h.lastCheck,
		Uptime:    uptime.String(),
		Metrics:   h.copyMetrics(),
	}
}

// RecordConnection increments connection metrics
func (h *HealthChecker) RecordConnection() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.metrics.TotalConnections++
	h.metrics.ActiveConnections++
}

// RecordDisconnection decrements active connections
func (h *HealthChecker) RecordDisconnection() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.metrics.ActiveConnections > 0 {
		h.metrics.ActiveConnections--
	}
}

// RecordMessagePublished increments message publication count
func (h *HealthChecker) RecordMessagePublished() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.metrics.MessagesPublished++
}

// RecordMessageDelivered increments message delivery count
func (h *HealthChecker) RecordMessageDelivered() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.metrics.MessagesDelivered++
}

// RecordEventProcessed increments event processing count
func (h *HealthChecker) RecordEventProcessed() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.metrics.EventsProcessed++
}

// RecordResourceCompletion increments resource completion count
func (h *HealthChecker) RecordResourceCompletion() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.metrics.ResourceCompletions++
}

// RecordError increments error count and may affect health status
func (h *HealthChecker) RecordError() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.metrics.ErrorCount++

	// Mark as unhealthy if error rate is too high
	if h.metrics.ErrorCount > 100 && h.metrics.EventsProcessed > 0 {
		errorRate := float64(h.metrics.ErrorCount) / float64(h.metrics.EventsProcessed)
		if errorRate > 0.1 { // 10% error rate threshold
			h.isHealthy = false
			h.logger.Warn("Bus marked unhealthy due to high error rate",
				"errorRate", fmt.Sprintf("%.2f%%", errorRate*100))
		}
	}
}

// UpdateLatency updates the average latency metric
func (h *HealthChecker) UpdateLatency(latency time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Simple exponential moving average
	if h.metrics.AverageLatency == 0 {
		h.metrics.AverageLatency = latency
	} else {
		h.metrics.AverageLatency = time.Duration(
			float64(h.metrics.AverageLatency)*0.9 + float64(latency)*0.1,
		)
	}
}

// monitorHealth performs periodic health checks
func (h *HealthChecker) monitorHealth(ctx context.Context) {
	ticker := time.NewTicker(h.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			h.logger.Debug("Health checker stopping due to context cancellation")
			return
		case <-ticker.C:
			h.performHealthCheck()
		}
	}
}

// performHealthCheck executes a single health check
func (h *HealthChecker) performHealthCheck() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.lastCheck = time.Now()
	previousHealth := h.isHealthy

	// Start with healthy state and check for issues
	healthyStatus := true

	// Check error rate
	if h.metrics.ErrorCount > 100 && h.metrics.EventsProcessed > 0 {
		errorRate := float64(h.metrics.ErrorCount) / float64(h.metrics.EventsProcessed)
		if errorRate > 0.1 { // 10% error rate threshold
			healthyStatus = false
		}
	}

	// Check for reasonable activity levels
	if h.metrics.EventsProcessed == 0 && time.Since(h.metrics.UptimeStart) > 5*time.Minute {
		h.logger.Warn("Bus appears inactive - no events processed in 5 minutes")
	}

	// Check average latency
	if h.metrics.AverageLatency > 5*time.Second {
		h.logger.Warn("High bus latency detected", "avgLatency", h.metrics.AverageLatency)
	}

	// Check connection health
	if h.metrics.ActiveConnections > 1000 {
		h.logger.Warn("High number of active connections", "count", h.metrics.ActiveConnections)
	}

	// Update health status
	h.isHealthy = healthyStatus

	// Log health status change
	if previousHealth != h.isHealthy {
		if h.isHealthy {
			h.logger.Info("Bus health recovered")
		} else {
			h.logger.Error("Bus health degraded")
		}
	}

	// Log periodic metrics
	h.logger.Debug("Bus health check completed",
		"healthy", h.isHealthy,
		"connections", h.metrics.ActiveConnections,
		"eventsProcessed", h.metrics.EventsProcessed,
		"avgLatency", h.metrics.AverageLatency,
		"errorCount", h.metrics.ErrorCount,
	)
}

// copyMetrics creates a safe copy of metrics for external access
func (h *HealthChecker) copyMetrics() *BusMetrics {
	return &BusMetrics{
		TotalConnections:    h.metrics.TotalConnections,
		ActiveConnections:   h.metrics.ActiveConnections,
		MessagesPublished:   h.metrics.MessagesPublished,
		MessagesDelivered:   h.metrics.MessagesDelivered,
		EventsProcessed:     h.metrics.EventsProcessed,
		AverageLatency:      h.metrics.AverageLatency,
		ErrorCount:          h.metrics.ErrorCount,
		ResourceCompletions: h.metrics.ResourceCompletions,
		UptimeStart:         h.metrics.UptimeStart,
	}
}

// ResetMetrics clears all metrics (useful for testing)
func (h *HealthChecker) ResetMetrics() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.metrics = &BusMetrics{
		UptimeStart: time.Now(),
	}
	h.isHealthy = true
	h.logger.Info("Bus metrics reset")
}
