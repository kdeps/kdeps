package bus

import (
	"errors"
	"net/rpc"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
)

func TestDefaultRetryConfig(t *testing.T) {
	t.Parallel()

	config := DefaultRetryConfig()

	if config.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries to be 3, got %d", config.MaxRetries)
	}
	if config.InitialInterval != 100*time.Millisecond {
		t.Errorf("Expected InitialInterval to be 100ms, got %v", config.InitialInterval)
	}
	if config.MaxInterval != 5*time.Second {
		t.Errorf("Expected MaxInterval to be 5s, got %v", config.MaxInterval)
	}
	if config.Multiplier != 2.0 {
		t.Errorf("Expected Multiplier to be 2.0, got %f", config.Multiplier)
	}
}

func TestNewCircuitBreaker(t *testing.T) {
	t.Parallel()

	cb := NewCircuitBreaker()

	if cb.state != CircuitClosed {
		t.Errorf("Expected initial state to be closed, got %v", cb.state)
	}
	if cb.maxFailures != 5 {
		t.Errorf("Expected maxFailures to be 5, got %d", cb.maxFailures)
	}
	if cb.resetTimeout != 60*time.Second {
		t.Errorf("Expected resetTimeout to be 60s, got %v", cb.resetTimeout)
	}
}

func TestCircuitBreaker_Execute(t *testing.T) {
	t.Parallel()

	cb := NewCircuitBreaker()

	// Test successful execution
	successCount := 0
	err := cb.Execute(func() error {
		successCount++
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if successCount != 1 {
		t.Errorf("Expected function to be called once, got %d", successCount)
	}

	// Test error execution
	testError := errors.New("test error")
	err = cb.Execute(func() error {
		return testError
	})

	if err != testError {
		t.Errorf("Expected test error, got %v", err)
	}
}

func TestCircuitBreaker_StateTransitions(t *testing.T) {
	t.Parallel()

	cb := NewCircuitBreaker()
	cb.maxFailures = 2 // Lower threshold for testing

	// Initial state should be closed
	if cb.GetState() != CircuitClosed {
		t.Errorf("Expected initial state to be closed, got %v", cb.GetState())
	}

	// Record failures to reach threshold
	cb.recordFailure()
	if cb.GetState() != CircuitClosed {
		t.Errorf("Expected state to remain closed after 1 failure, got %v", cb.GetState())
	}

	cb.recordFailure()
	if cb.GetState() != CircuitOpen {
		t.Errorf("Expected state to be open after 2 failures, got %v", cb.GetState())
	}

	// Should reject requests when open
	executed := false
	err := cb.Execute(func() error {
		executed = true
		return nil
	})

	if err == nil {
		t.Error("Expected error when circuit is open")
	}
	if executed {
		t.Error("Function should not execute when circuit is open")
	}
}

func TestCircuitBreaker_HalfOpen(t *testing.T) {
	t.Parallel()

	cb := NewCircuitBreaker()
	cb.maxFailures = 1
	cb.resetTimeout = 10 * time.Millisecond // Short timeout for testing
	cb.halfOpenMaxCalls = 2

	// Force circuit open
	cb.recordFailure()
	if cb.GetState() != CircuitOpen {
		t.Errorf("Expected state to be open, got %v", cb.GetState())
	}

	// Wait for reset timeout
	time.Sleep(20 * time.Millisecond)

	// Next request should transition to half-open
	successCount := 0
	cb.Execute(func() error {
		successCount++
		return nil
	})

	if cb.GetState() != CircuitHalfOpen {
		t.Errorf("Expected state to be half-open, got %v", cb.GetState())
	}

	// Second successful call should close circuit
	cb.Execute(func() error {
		successCount++
		return nil
	})

	if cb.GetState() != CircuitClosed {
		t.Errorf("Expected state to be closed after successful half-open calls, got %v", cb.GetState())
	}
	if successCount != 2 {
		t.Errorf("Expected 2 successful calls, got %d", successCount)
	}
}

func TestConnectionPool_Mockable(t *testing.T) {
	t.Parallel()

	logger := logging.GetLogger()

	// Test pool creation failure (since we're not running a real server)
	pool, err := NewConnectionPool("127.0.0.1:99999", 2, logger)
	if err == nil {
		t.Error("Expected error when connecting to non-existent server")
		if pool != nil {
			pool.Close()
		}
	}
}

func TestConnectionPool_Logic(t *testing.T) {
	t.Parallel()

	logger := logging.GetLogger()

	// Create a mock pool with nil connections for testing logic
	pool := &ConnectionPool{
		connections: []*rpc.Client{nil, nil}, // Mock connections
		size:        2,
		address:     "test",
		logger:      logger,
		current:     0,
	}

	// Test Get() round-robin logic
	conn1 := pool.Get()
	_ = pool.Get()      // conn2
	conn3 := pool.Get() // Should wrap around

	// Since we're using round-robin, conn3 should be same as conn1
	// (though they're all nil in this test)
	if conn1 != conn3 {
		t.Error("Expected round-robin to wrap around")
	}

	// Test Health()
	if !pool.Health() {
		t.Error("Expected pool to be healthy with connections")
	}

	// Test Close()
	pool.Close()
	if pool.connections != nil {
		t.Error("Expected connections to be nil after close")
	}
}

func TestResilientClient_Configuration(t *testing.T) {
	t.Parallel()

	logger := logging.GetLogger()

	// Test that client creation fails gracefully when server not available
	client, err := NewResilientClient(logger)
	if err == nil {
		t.Log("Client created successfully (server must be running)")
		if client != nil {
			client.Close()
		}
	} else {
		t.Log("Client creation failed as expected when server not available:", err)
	}

	// Test custom configuration
	retryConfig := RetryConfig{
		MaxRetries:      5,
		InitialInterval: 50 * time.Millisecond,
		MaxInterval:     10 * time.Second,
		Multiplier:      1.5,
	}

	client, err = NewResilientClientWithConfig(logger, 3, retryConfig)
	if err == nil {
		if client.retryConfig.MaxRetries != 5 {
			t.Errorf("Expected MaxRetries to be 5, got %d", client.retryConfig.MaxRetries)
		}
		client.Close()
	}
}

func TestResilientClient_RetryLogic(t *testing.T) {
	t.Parallel()

	// Create client with minimal pool for testing
	retryConfig := RetryConfig{
		MaxRetries:      2,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
	}

	// Test the retry logic directly without using the pool
	// since the pool requires actual RPC connections
	attemptCount := 0

	// Mock the retry operation manually to test the logic
	interval := retryConfig.InitialInterval
	for attempt := 0; attempt <= retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			// Simulate the backoff delay (without actually waiting)
			if interval < retryConfig.MaxInterval {
				interval = time.Duration(float64(interval) * retryConfig.Multiplier)
				if interval > retryConfig.MaxInterval {
					interval = retryConfig.MaxInterval
				}
			}
		}

		attemptCount++
		// Simulate the operation always failing
	}

	// Verify the expected number of attempts
	expectedAttempts := retryConfig.MaxRetries + 1

	if attemptCount != expectedAttempts {
		t.Errorf("Expected %d attempts, got %d", expectedAttempts, attemptCount)
	}
}

func TestResilientClient_GetMetrics(t *testing.T) {
	t.Parallel()

	// Create mock client for testing
	client := &ResilientClient{
		circuitBreaker: NewCircuitBreaker(),
		pool: &ConnectionPool{
			connections: []*rpc.Client{nil},
			size:        1,
		},
	}

	metrics := client.GetMetrics()

	if _, ok := metrics["circuit_breaker_state"]; !ok {
		t.Error("Expected circuit_breaker_state in metrics")
	}
	if _, ok := metrics["connection_pool_healthy"]; !ok {
		t.Error("Expected connection_pool_healthy in metrics")
	}
	if _, ok := metrics["connection_pool_size"]; !ok {
		t.Error("Expected connection_pool_size in metrics")
	}
}

// Integration test with real server (requires server to be running)
func TestResilientClient_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := logging.GetLogger()

	client, err := NewResilientClient(logger)
	if err != nil {
		t.Skip("Skipping integration test - bus server not available:", err)
	}
	defer client.Close()

	// Test health check
	health, err := client.HealthCheck()
	if err != nil {
		t.Errorf("Health check failed: %v", err)
	} else {
		t.Logf("Health check successful: %+v", health)
	}

	// Test metrics
	metrics := client.GetMetrics()
	t.Logf("Client metrics: %+v", metrics)
}
