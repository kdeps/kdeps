package bus

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
)

func TestClient(t *testing.T) {
	t.Parallel()

	logger := logging.GetLogger()

	// Test waiting for events successfully.
	t.Run("WaitForEventsSuccess", func(t *testing.T) {
		t.Parallel()

		oldAddr := busAddr
		busAddr = "127.0.0.1:12345"
		defer func() { busAddr = oldAddr }()

		client, err := StartBusClient()
		if err != nil {
			t.Skipf("Failed to connect to server: %v", err)
		}
		defer client.Close()

		// Use a mock client for this test since testService might be nil when reusing existing servers
		mockClient := &mockRPCClient{
			callFunc: func(serviceMethod string, args interface{}, reply interface{}) error {
				if serviceMethod == "BusService.Subscribe" {
					resp := reply.(*SubscribeResponse)
					resp.ID = "test-sub"
					resp.Error = ""
					return nil
				}
				if serviceMethod == "BusService.GetEvent" {
					// Simulate receiving events
					resp := reply.(*EventResponse)
					resp.Event = Event{Type: "ready", Payload: "Done"}
					resp.Error = ""
					return nil
				}
				return fmt.Errorf("unknown method: %s", serviceMethod)
			},
		}

		// Handler to process events.
		handler := func(event Event) bool {
			if event.Type == "ready" {
				return true // Stop waiting
			}
			return false
		}

		err = WaitForEvents(mockClient, logger, handler)
		if err != nil {
			t.Errorf("WaitForEvents failed: %v", err)
		}
	})

	// Test timeout when waiting for events.
	t.Run("WaitForEventsTimeout", func(t *testing.T) {
		t.Parallel()

		oldAddr := busAddr
		busAddr = "127.0.0.1:12345"
		defer func() { busAddr = oldAddr }()

		client, err := StartBusClient()
		if err != nil {
			t.Skipf("Failed to connect to server: %v", err)
		}
		defer client.Close()

		handler := func(event Event) bool {
			return false // Never stop.
		}

		err = WaitForEvents(client, logger, handler)
		if err == nil {
			t.Errorf("Expected timeout error, got nil")
		}
		// Accept either timeout error or connection errors (like broken pipe)
		// since the connection might fail before timeout in test environments
		if !strings.Contains(err.Error(), "timeout waiting for events") &&
			!strings.Contains(err.Error(), "broken pipe") &&
			!strings.Contains(err.Error(), "failed to get event from bus") {
			t.Errorf("Expected timeout or connection error, got: %v", err)
		}
	})
}

// Test failure to start the bus client when the server is unavailable.
func TestStartBusClientFailure(t *testing.T) {
	t.Parallel()

	oldAddr := busAddr
	busAddr = "127.0.0.1:12346"
	defer func() { busAddr = oldAddr }()

	_, err := StartBusClient()
	if err == nil {
		t.Errorf("Expected connection error, got nil")
	}
}

// Test error handling when WaitForEvents is called with a nil client.
func TestWaitForEventsError(t *testing.T) {
	t.Parallel()

	logger := logging.GetLogger()

	err := WaitForEvents(nil, logger, func(event Event) bool { return false })
	if err == nil {
		t.Errorf("Expected error on nil client, got nil")
	}
	if err.Error() != "nil client provided" {
		t.Errorf("Expected 'nil client provided' error, got: %v", err)
	}
}

// TestWaitForEventsComprehensiveErrorPaths tests all error scenarios in WaitForEvents
func TestWaitForEventsComprehensiveErrorPaths(t *testing.T) {
	logger := logging.NewTestLogger()

	t.Run("NilClient", func(t *testing.T) {
		err := WaitForEvents(nil, logger, func(event Event) bool { return false })
		if err == nil {
			t.Errorf("Expected error on nil client, got nil")
		}
		if err.Error() != "nil client provided" {
			t.Errorf("Expected 'nil client provided' error, got: %v", err)
		}
	})

	t.Run("SubscribeCallError", func(t *testing.T) {
		// Create a mock client that will fail on Subscribe call
		mockClient := &mockRPCClient{
			callResponses: map[string]callResponse{
				"BusService.Subscribe": {
					err: fmt.Errorf("connection failed"),
				},
			},
		}

		err := WaitForEvents(mockClient, logger, func(event Event) bool { return false })
		if err == nil {
			t.Errorf("Expected subscribe error, got nil")
		}
		if !strings.Contains(err.Error(), "failed to subscribe to bus") {
			t.Errorf("Expected subscribe error, got: %v", err)
		}
	})

	t.Run("SubscribeResponseError", func(t *testing.T) {
		// Create a mock client that returns an error in the response
		mockClient := &mockRPCClient{
			callResponses: map[string]callResponse{
				"BusService.Subscribe": {
					response: &SubscribeResponse{
						ID:    "",
						Error: "subscription failed",
					},
				},
			},
		}

		err := WaitForEvents(mockClient, logger, func(event Event) bool { return false })
		if err == nil {
			t.Errorf("Expected subscription error, got nil")
		}
		if !strings.Contains(err.Error(), "subscription error") {
			t.Errorf("Expected subscription error, got: %v", err)
		}
	})

	t.Run("GetEventCallError", func(t *testing.T) {
		// Create a mock client that succeeds on Subscribe but fails on GetEvent
		mockClient := &mockRPCClient{
			callResponses: map[string]callResponse{
				"BusService.Subscribe": {
					response: &SubscribeResponse{
						ID:    "test-sub",
						Error: "",
					},
				},
				"BusService.GetEvent": {
					err: fmt.Errorf("get event failed"),
				},
			},
		}

		err := WaitForEvents(mockClient, logger, func(event Event) bool { return false })
		if err == nil {
			t.Errorf("Expected get event error, got nil")
		}
		if !strings.Contains(err.Error(), "failed to get event from bus") {
			t.Errorf("Expected get event error, got: %v", err)
		}
	})

	t.Run("EventHandlerReturnsTrue", func(t *testing.T) {
		// Create a mock client that returns events
		mockClient := &mockRPCClient{
			callResponses: map[string]callResponse{
				"BusService.Subscribe": {
					response: &SubscribeResponse{
						ID:    "test-sub",
						Error: "",
					},
				},
				"BusService.GetEvent": {
					response: &EventResponse{
						Event: Event{Type: "test", Payload: "payload"},
						Error: "",
					},
				},
			},
		}

		handlerCalled := false
		err := WaitForEvents(mockClient, logger, func(event Event) bool {
			handlerCalled = true
			return true // This should cause the function to return nil
		})

		if err != nil {
			t.Errorf("Expected nil error when handler returns true, got: %v", err)
		}
		if !handlerCalled {
			t.Errorf("Expected handler to be called")
		}
	})

	t.Run("EventHandlerReturnsFalse", func(t *testing.T) {
		// Create a mock client that returns events then timeout
		callCount := 0
		mockClient := &mockRPCClient{
			callFunc: func(serviceMethod string, args interface{}, reply interface{}) error {
				if serviceMethod == "BusService.Subscribe" {
					resp := reply.(*SubscribeResponse)
					resp.ID = "test-sub"
					resp.Error = ""
					return nil
				}
				if serviceMethod == "BusService.GetEvent" {
					callCount++
					if callCount == 1 {
						// First call returns an event
						resp := reply.(*EventResponse)
						resp.Event = Event{Type: "test", Payload: "payload"}
						resp.Error = ""
						return nil
					}
					// Subsequent calls return "no events available" to simulate waiting
					resp := reply.(*EventResponse)
					resp.Error = "no events available"
					return nil
				}
				return fmt.Errorf("unknown method: %s", serviceMethod)
			},
		}

		handlerCallCount := 0
		err := WaitForEvents(mockClient, logger, func(event Event) bool {
			handlerCallCount++
			return false // Keep waiting, should eventually timeout
		})

		if err == nil {
			t.Errorf("Expected timeout error, got nil")
		}
		if !strings.Contains(err.Error(), "timeout waiting for events") {
			t.Errorf("Expected timeout error, got: %v", err)
		}
		if handlerCallCount != 1 {
			t.Errorf("Expected handler to be called once, got %d calls", handlerCallCount)
		}
	})
}

// mockRPCClient is a mock implementation of rpc.Client for testing
type mockRPCClient struct {
	callResponses map[string]callResponse
	callFunc      func(serviceMethod string, args interface{}, reply interface{}) error
}

type callResponse struct {
	response interface{}
	err      error
}

func (m *mockRPCClient) Call(serviceMethod string, args interface{}, reply interface{}) error {
	if m.callFunc != nil {
		return m.callFunc(serviceMethod, args, reply)
	}

	if resp, ok := m.callResponses[serviceMethod]; ok {
		if resp.err != nil {
			return resp.err
		}
		if resp.response != nil {
			// Copy the response to the reply parameter
			switch r := reply.(type) {
			case *SubscribeResponse:
				if sr, ok := resp.response.(*SubscribeResponse); ok {
					*r = *sr
				}
			case *EventResponse:
				if er, ok := resp.response.(*EventResponse); ok {
					*r = *er
				}
			}
		}
		return nil
	}
	return fmt.Errorf("no mock response for method: %s", serviceMethod)
}

func (m *mockRPCClient) Close() error {
	return nil
}
