package bus

import (
	"context"
	"errors"
	"fmt"
	"net/rpc"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
)

// CircuitState represents the current state of the circuit breaker
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern for resilient client connections
type CircuitBreaker struct {
	mu               sync.RWMutex
	state            CircuitState
	failureCount     int64
	successCount     int64
	lastFailureTime  time.Time
	lastSuccessTime  time.Time
	maxFailures      int64
	resetTimeout     time.Duration
	halfOpenMaxCalls int64
	halfOpenCalls    int64
}

// ConnectionPool manages a pool of RPC connections for load balancing and resilience
type ConnectionPool struct {
	mu          sync.RWMutex
	connections []*rpc.Client
	size        int
	current     int64
	address     string
	logger      *logging.Logger
}

// ResilientClient provides a production-ready bus client with resilience features
type ResilientClient struct {
	pool           *ConnectionPool
	circuitBreaker *CircuitBreaker
	logger         *logging.Logger
	ctx            context.Context
	cancel         context.CancelFunc
	retryConfig    RetryConfig
}

// RetryConfig configures retry behavior
type RetryConfig struct {
	MaxRetries      int
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
}

// DefaultRetryConfig returns sensible defaults for retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:      3,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     5 * time.Second,
		Multiplier:      2.0,
	}
}

// NewCircuitBreaker creates a new circuit breaker with default settings
func NewCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{
		state:            CircuitClosed,
		maxFailures:      5,
		resetTimeout:     60 * time.Second,
		halfOpenMaxCalls: 3,
	}
}

// Execute runs a function through the circuit breaker
func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.allowRequest() {
		return errors.New("circuit breaker is open")
	}

	err := fn()

	if err != nil {
		cb.recordFailure()
		return err
	}

	cb.recordSuccess()
	return nil
}

func (cb *CircuitBreaker) allowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.state = CircuitHalfOpen
			cb.halfOpenCalls = 0
			return true
		}
		return false
	case CircuitHalfOpen:
		return cb.halfOpenCalls < cb.halfOpenMaxCalls
	default:
		return false
	}
}

func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	atomic.AddInt64(&cb.successCount, 1)
	cb.lastSuccessTime = time.Now()

	if cb.state == CircuitHalfOpen {
		cb.halfOpenCalls++
		if cb.halfOpenCalls >= cb.halfOpenMaxCalls {
			cb.state = CircuitClosed
			atomic.StoreInt64(&cb.failureCount, 0)
		}
	}
}

func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	atomic.AddInt64(&cb.failureCount, 1)
	cb.lastFailureTime = time.Now()

	if cb.state == CircuitHalfOpen || atomic.LoadInt64(&cb.failureCount) >= cb.maxFailures {
		cb.state = CircuitOpen
	}
}

// GetState returns the current circuit breaker state
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(address string, size int, logger *logging.Logger) (*ConnectionPool, error) {
	if size <= 0 {
		size = 5 // default pool size
	}

	pool := &ConnectionPool{
		connections: make([]*rpc.Client, 0, size),
		size:        size,
		address:     address,
		logger:      logger,
	}

	// Initialize connections
	for i := 0; i < size; i++ {
		conn, err := rpc.Dial("tcp", address)
		if err != nil {
			// Close any successful connections before returning error
			pool.Close()
			return nil, fmt.Errorf("failed to create connection %d: %w", i, err)
		}
		pool.connections = append(pool.connections, conn)
	}

	logger.Info("Connection pool created", "address", address, "size", size)
	return pool, nil
}

// Get returns a connection from the pool using round-robin
func (p *ConnectionPool) Get() *rpc.Client {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.connections) == 0 {
		return nil
	}

	index := atomic.AddInt64(&p.current, 1) % int64(len(p.connections))
	return p.connections[index]
}

// Close closes all connections in the pool
func (p *ConnectionPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, conn := range p.connections {
		if conn != nil {
			conn.Close()
			p.logger.Debug("Closed pool connection", "index", i)
		}
	}
	p.connections = nil
}

// Health checks if the pool has healthy connections
func (p *ConnectionPool) Health() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.connections) > 0
}

// NewResilientClient creates a new resilient bus client
func NewResilientClient(logger *logging.Logger) (*ResilientClient, error) {
	return NewResilientClientWithConfig(logger, 5, DefaultRetryConfig())
}

// NewResilientClientWithConfig creates a resilient client with custom configuration
func NewResilientClientWithConfig(logger *logging.Logger, poolSize int, retryConfig RetryConfig) (*ResilientClient, error) {
	ctx, cancel := context.WithCancel(context.Background())

	pool, err := NewConnectionPool("127.0.0.1:12345", poolSize, logger)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	client := &ResilientClient{
		pool:           pool,
		circuitBreaker: NewCircuitBreaker(),
		logger:         logger,
		ctx:            ctx,
		cancel:         cancel,
		retryConfig:    retryConfig,
	}

	logger.Info("Resilient bus client created", "poolSize", poolSize)
	return client, nil
}

// Close gracefully closes the resilient client
func (rc *ResilientClient) Close() error {
	rc.cancel()
	rc.pool.Close()
	rc.logger.Info("Resilient bus client closed")
	return nil
}

// ExecuteWithRetry executes an RPC call with retry logic and circuit breaking
func (rc *ResilientClient) ExecuteWithRetry(operation func(*rpc.Client) error) error {
	return rc.circuitBreaker.Execute(func() error {
		return rc.retryOperation(operation)
	})
}

func (rc *ResilientClient) retryOperation(operation func(*rpc.Client) error) error {
	var lastErr error
	interval := rc.retryConfig.InitialInterval

	for attempt := 0; attempt <= rc.retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-rc.ctx.Done():
				return rc.ctx.Err()
			case <-time.After(interval):
				// Continue with retry
			}
		}

		conn := rc.pool.Get()
		if conn == nil {
			lastErr = errors.New("no available connections in pool")
			rc.logger.Warn("No available connections", "attempt", attempt)
			continue
		}

		err := operation(conn)
		if err == nil {
			return nil // Success
		}

		lastErr = err
		rc.logger.Debug("Operation failed, will retry", "attempt", attempt, "error", err)

		// Exponential backoff
		if interval < rc.retryConfig.MaxInterval {
			interval = time.Duration(float64(interval) * rc.retryConfig.Multiplier)
			if interval > rc.retryConfig.MaxInterval {
				interval = rc.retryConfig.MaxInterval
			}
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", rc.retryConfig.MaxRetries, lastErr)
}

// SignalResourceCompletion signals resource completion with resilience
func (rc *ResilientClient) SignalResourceCompletion(resourceID, status string, data map[string]interface{}) error {
	return rc.ExecuteWithRetry(func(client *rpc.Client) error {
		return SignalResourceCompletion(client, resourceID, status, data)
	})
}

// WaitForResourceCompletion waits for resource completion with resilience
func (rc *ResilientClient) WaitForResourceCompletion(resourceID string, timeoutSeconds int64) (*ResourceState, error) {
	var result *ResourceState
	err := rc.ExecuteWithRetry(func(client *rpc.Client) error {
		state, err := WaitForResourceCompletion(client, resourceID, timeoutSeconds)
		if err != nil {
			return err
		}
		result = state
		return nil
	})
	return result, err
}

// PublishEvent publishes an event with resilience
func (rc *ResilientClient) PublishEvent(eventType, payload, resourceID string, data map[string]interface{}) error {
	return rc.ExecuteWithRetry(func(client *rpc.Client) error {
		return PublishEvent(client, eventType, payload, resourceID, data)
	})
}

// WaitForCleanupSignal waits for cleanup with resilience
func (rc *ResilientClient) WaitForCleanupSignal(timeoutSeconds int64) error {
	return rc.ExecuteWithRetry(func(client *rpc.Client) error {
		return WaitForCleanupSignal(client, rc.logger, timeoutSeconds)
	})
}

// HealthCheck performs a health check on the bus service
func (rc *ResilientClient) HealthCheck() (*HealthStatus, error) {
	var status *HealthStatus
	err := rc.ExecuteWithRetry(func(client *rpc.Client) error {
		var req HealthCheckRequest
		var resp HealthCheckResponse

		err := client.Call("BusService.HealthCheck", req, &resp)
		if err != nil {
			return err
		}
		if resp.Error != "" {
			return fmt.Errorf("health check error: %s", resp.Error)
		}

		status = &resp.Status
		return nil
	})
	return status, err
}

// GetMetrics returns circuit breaker and pool metrics
func (rc *ResilientClient) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"circuit_breaker_state":     rc.circuitBreaker.GetState(),
		"circuit_breaker_failures":  atomic.LoadInt64(&rc.circuitBreaker.failureCount),
		"circuit_breaker_successes": atomic.LoadInt64(&rc.circuitBreaker.successCount),
		"connection_pool_healthy":   rc.pool.Health(),
		"connection_pool_size":      rc.pool.size,
	}
}
