package bus

import (
	"testing"
	"time"

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

		// Publish events with slight delays.
		go func() {
			time.Sleep(100 * time.Millisecond)
			req1 := PublishEventRequest{Event: Event{Type: "progress", Payload: "Working"}}
			var resp1 PublishEventResponse
			testService.PublishEvent(req1, &resp1)
			time.Sleep(100 * time.Millisecond)
			req2 := PublishEventRequest{Event: Event{Type: "ready", Payload: "Done"}}
			var resp2 PublishEventResponse
			testService.PublishEvent(req2, &resp2)
		}()

		// Handler to process events.
		handler := func(event Event) bool {
			if event.Type == "progress" {
				return false
			}
			if event.Type == "ready" {
				return true
			}
			t.Errorf("Unexpected event type: %s", event.Type)
			return false
		}

		err = WaitForEvents(client, logger, handler)
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
		if err.Error() != "timeout waiting for events" {
			t.Errorf("Expected timeout error, got: %v", err)
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
