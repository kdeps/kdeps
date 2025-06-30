package bus

import (
	"context"
	"fmt"
	"net"
	"net/rpc"
	"sync"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
)

type BusService struct {
	logger *logging.Logger
	subs   map[string]chan Event // Map of subscription ID to event channel
	mu     sync.Mutex
	nextID int
	// Add storage for resource states and completion tracking
	resourceStates map[string]ResourceState
	completions    map[string]bool
	// Add health monitoring
	healthChecker *HealthChecker
	ctx           context.Context
	cancel        context.CancelFunc
}

type Event struct {
	Type    string
	Payload string
	// Add metadata for different event types
	ResourceID string
	Timestamp  int64
	Data       map[string]interface{}
}

type ResourceState struct {
	ResourceID string
	Status     string // "running", "completed", "failed"
	Timestamp  int64
	Data       map[string]interface{}
}

type SubscribeRequest struct{}

type SubscribeResponse struct {
	ID    string
	Error string
}

type EventRequest struct {
	ID string
}

type EventResponse struct {
	Event Event
	Error string
}

// New RPC methods for enhanced IPC
type SignalCompletionRequest struct {
	ResourceID string
	Status     string
	Data       map[string]interface{}
}

type SignalCompletionResponse struct {
	Success bool
	Error   string
}

type WaitForCompletionRequest struct {
	ResourceID string
	Timeout    int64 // timeout in seconds
}

type WaitForCompletionResponse struct {
	Success bool
	Status  string
	Error   string
	Data    map[string]interface{}
}

type PublishEventRequest struct {
	Event Event
}

type PublishEventResponse struct {
	Success bool
	Error   string
}

// Health check RPC methods
type HealthCheckRequest struct{}

type HealthCheckResponse struct {
	Status HealthStatus
	Error  string
}

func (s *BusService) Subscribe(req SubscribeRequest, resp *SubscribeResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("sub-%d", s.nextID)
	s.nextID++
	s.subs[id] = make(chan Event, 10) // Buffered to prevent blocking
	resp.ID = id
	s.logger.Info("Client subscribed", "id", id)

	// Record metrics
	if s.healthChecker != nil {
		s.healthChecker.RecordConnection()
	}

	return nil
}

func (s *BusService) GetEvent(req EventRequest, resp *EventResponse) error {
	start := time.Now()
	defer func() {
		if s.healthChecker != nil {
			s.healthChecker.UpdateLatency(time.Since(start))
			s.healthChecker.RecordEventProcessed()
		}
	}()

	s.mu.Lock()
	ch, ok := s.subs[req.ID]
	s.mu.Unlock()
	if !ok {
		resp.Error = "invalid subscription ID"
		if s.healthChecker != nil {
			s.healthChecker.RecordError()
		}
		return nil
	}
	select {
	case event := <-ch:
		resp.Event = event
		s.logger.Debug("Delivering event to client", "id", req.ID, "type", event.Type, "payload", event.Payload)
		if s.healthChecker != nil {
			s.healthChecker.RecordMessageDelivered()
		}
	case <-time.After(5 * time.Second):
		resp.Error = "no events available"
	}
	return nil
}

// SignalCompletion signals completion of a resource or operation
func (s *BusService) SignalCompletion(req SignalCompletionRequest, resp *SignalCompletionResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	timestamp := time.Now().Unix()
	s.resourceStates[req.ResourceID] = ResourceState{
		ResourceID: req.ResourceID,
		Status:     req.Status,
		Timestamp:  timestamp,
		Data:       req.Data,
	}
	s.completions[req.ResourceID] = true

	// Publish completion event
	event := Event{
		Type:       "completion",
		Payload:    fmt.Sprintf("Resource %s completed with status: %s", req.ResourceID, req.Status),
		ResourceID: req.ResourceID,
		Timestamp:  timestamp,
		Data:       req.Data,
	}
	s.publishEventInternal(event)

	resp.Success = true
	s.logger.Info("Resource completion signaled", "resourceID", req.ResourceID, "status", req.Status)

	// Record metrics
	if s.healthChecker != nil {
		s.healthChecker.RecordResourceCompletion()
	}

	return nil
}

// WaitForCompletion waits for a resource to complete
func (s *BusService) WaitForCompletion(req WaitForCompletionRequest, resp *WaitForCompletionResponse) error {
	timeout := time.Duration(req.Timeout) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second // default timeout
	}

	start := time.Now()
	for {
		s.mu.Lock()
		if state, ok := s.resourceStates[req.ResourceID]; ok {
			s.mu.Unlock()
			resp.Success = true
			resp.Status = state.Status
			resp.Data = state.Data
			s.logger.Info("Resource completion detected", "resourceID", req.ResourceID, "status", state.Status)
			return nil
		}
		s.mu.Unlock()

		if time.Since(start) > timeout {
			resp.Error = fmt.Sprintf("timeout waiting for resource %s to complete", req.ResourceID)
			s.logger.Warn("Timeout waiting for resource completion", "resourceID", req.ResourceID)
			if s.healthChecker != nil {
				s.healthChecker.RecordError()
			}
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}
}

// PublishEvent allows external publishing of events
func (s *BusService) PublishEvent(req PublishEventRequest, resp *PublishEventResponse) error {
	s.publishEventInternal(req.Event)
	resp.Success = true

	// Record metrics
	if s.healthChecker != nil {
		s.healthChecker.RecordMessagePublished()
	}

	return nil
}

// HealthCheck returns current health status
func (s *BusService) HealthCheck(req HealthCheckRequest, resp *HealthCheckResponse) error {
	if s.healthChecker != nil {
		resp.Status = s.healthChecker.GetHealth()
	} else {
		resp.Error = "health checker not available"
	}
	return nil
}

func (s *BusService) publishEventInternal(event Event) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if event.Timestamp == 0 {
		event.Timestamp = time.Now().Unix()
	}

	s.logger.Info("Publishing event", "type", event.Type, "payload", event.Payload, "resourceID", event.ResourceID)
	for id, ch := range s.subs {
		select {
		case ch <- event:
			s.logger.Debug("Sent event to subscriber", "id", id)
		default:
			s.logger.Warn("Subscriber channel full", "id", id)
			if s.healthChecker != nil {
				s.healthChecker.RecordError()
			}
		}
	}
}

// Legacy method for backwards compatibility
func (s *BusService) PublishEventLegacy(event Event) {
	s.publishEventInternal(event)
}

// Shutdown gracefully shuts down the bus service
func (s *BusService) Shutdown() {
	if s.cancel != nil {
		s.cancel()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Close all subscriber channels
	for id, ch := range s.subs {
		close(ch)
		s.logger.Debug("Closed subscriber channel", "id", id)
		if s.healthChecker != nil {
			s.healthChecker.RecordDisconnection()
		}
	}

	s.logger.Info("Bus service shutdown completed")
}

func StartBusServer(logger *logging.Logger) error {
	return StartBusServerWithContext(context.Background(), logger)
}

func StartBusServerWithContext(ctx context.Context, logger *logging.Logger) error {
	busCtx, cancel := context.WithCancel(ctx)

	// Initialize health checker
	healthChecker := NewHealthChecker(logger, 30*time.Second)
	healthChecker.Start(busCtx)

	service := &BusService{
		logger:         logger,
		subs:           make(map[string]chan Event),
		resourceStates: make(map[string]ResourceState),
		completions:    make(map[string]bool),
		healthChecker:  healthChecker,
		ctx:            busCtx,
		cancel:         cancel,
	}

	if err := rpc.Register(service); err != nil {
		cancel()
		return fmt.Errorf("failed to register RPC service: %w", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:12345")
	if err != nil {
		cancel()
		return fmt.Errorf("failed to listen on 127.0.0.1:12345: %w", err)
	}

	logger.Info("Message Bus RPC server started on 127.0.0.1:12345")

	// Handle graceful shutdown
	go func() {
		<-busCtx.Done()
		logger.Info("Shutting down bus server...")
		listener.Close()
		service.Shutdown()
	}()

	// Start accepting connections
	rpc.Accept(listener)
	return nil
}
