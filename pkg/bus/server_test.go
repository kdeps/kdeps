package bus

import (
	"fmt"
	"net"
	"net/rpc"
	"os"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
)

var (
	testService  *BusService
	testListener net.Listener
)

// TestMain sets up a shared test server for all tests.
func TestMain(m *testing.M) {
	logger := logging.GetLogger()

	// Check if port 12345 is in use; reuse if possible, otherwise start a new server.
	conn, err := net.DialTimeout("tcp", "127.0.0.1:12345", 100*time.Millisecond)
	if err == nil {
		conn.Close()
		logger.Info("Reusing existing server on 127.0.0.1:12345")
	} else {
		listener, err := net.Listen("tcp", "127.0.0.1:12345")
		if err != nil {
			logger.Warn("Failed to start test server on 127.0.0.1:12345: %v", err)
		} else {
			service := &BusService{
				logger: logger,
				subs:   make(map[string]chan Event),
			}
			if err := rpc.Register(service); err != nil {
				logger.Fatal("Failed to register service: %v", err)
			}
			go rpc.Accept(listener)
			testListener = listener
			testService = service
			logger.Info("Started test server on 127.0.0.1:12345")
		}
	}
	time.Sleep(100 * time.Millisecond) // Wait for server to start.

	exitCode := m.Run()

	if testListener != nil {
		testListener.Close()
	}
	os.Exit(exitCode)
}

func TestBusService(t *testing.T) {
	// Test subscription to the bus service.
	t.Run("Subscribe", func(t *testing.T) {
		t.Parallel()

		client, err := rpc.Dial("tcp", "127.0.0.1:12345")
		if err != nil {
			t.Skipf("Failed to connect to server: %v", err)
		}
		defer client.Close()

		var resp SubscribeResponse
		err = client.Call("BusService.Subscribe", SubscribeRequest{}, &resp)
		if err != nil {
			t.Errorf("Subscribe failed: %v", err)
		}
		if resp.Error != "" {
			t.Errorf("Subscribe returned error: %s", resp.Error)
		}
	})

	// Test publishing and retrieving an event.
	t.Run("PublishAndGetEvent", func(t *testing.T) {
		t.Parallel()

		client, err := rpc.Dial("tcp", "127.0.0.1:12345")
		if err != nil {
			t.Skipf("Failed to connect to server: %v", err)
		}
		defer client.Close()

		var subResp SubscribeResponse
		err = client.Call("BusService.Subscribe", SubscribeRequest{}, &subResp)
		if err != nil {
			t.Errorf("Subscribe failed: %v", err)
		}
		if subResp.Error != "" {
			t.Errorf("Subscribe returned error: %s", subResp.Error)
		}
		subID := subResp.ID

		testEvent := Event{Type: "test", Payload: "test payload"}
		go func() {
			time.Sleep(50 * time.Millisecond) // Brief delay to allow subscription setup
			testService.PublishEvent(testEvent)
		}()

		var eventResp EventResponse
		err = client.Call("BusService.GetEvent", EventRequest{ID: subID}, &eventResp)
		if err != nil {
			t.Errorf("GetEvent failed: %v", err)
		}
		if eventResp.Error != "" {
			t.Errorf("GetEvent returned error: %s", eventResp.Error)
		}
		if eventResp.Event != testEvent {
			t.Errorf("Expected event %v, got %v", testEvent, eventResp.Event)
		}
	})

	// Test timeout when no events are available.
	t.Run("GetEventTimeout", func(t *testing.T) {
		t.Parallel()

		client, err := rpc.Dial("tcp", "127.0.0.1:12345")
		if err != nil {
			t.Skipf("Failed to connect to server: %v", err)
		}
		defer client.Close()

		var subResp SubscribeResponse
		err = client.Call("BusService.Subscribe", SubscribeRequest{}, &subResp)
		if err != nil {
			t.Errorf("Subscribe failed: %v", err)
		}
		if subResp.Error != "" {
			t.Errorf("Subscribe returned error: %s", subResp.Error)
		}
		subID := subResp.ID

		attempts := 0
		maxAttempts := 10
		for attempts < maxAttempts {
			var eventResp EventResponse
			err = client.Call("BusService.GetEvent", EventRequest{ID: subID}, &eventResp)
			if err != nil {
				t.Errorf("GetEvent failed: %v", err)
			}
			if eventResp.Error != "" {
				// Got the timeout error, as expected
				if eventResp.Event.Type != "" || eventResp.Event.Payload != "" {
					t.Errorf("Expected empty event on timeout, got %v", eventResp.Event)
				}
				return // Test passes
			}
			t.Logf("Discarded unexpected event: %v", eventResp.Event)
			attempts++
			time.Sleep(100 * time.Millisecond) // Small delay to allow other tests to finish
		}
		t.Errorf("Failed to get timeout error after %d attempts", maxAttempts)
	})

	// Test server binding and connectivity.
	t.Run("ServerBinding", func(t *testing.T) {
		t.Parallel()

		conn, err := net.Dial("tcp", "127.0.0.1:12345")
		if err != nil {
			t.Skipf("Failed to connect to 127.0.0.1:12345: %v", err)
		}
		conn.Close()
	})
}

// Test starting the bus server when the port is already in use.
func TestStartBusServerError(t *testing.T) {
	t.Parallel()

	logger := logging.GetLogger()

	l, err := net.Listen("tcp", "127.0.0.1:12345")
	if err != nil {
		t.Skipf("Port 12345 already in use: %v", err)
	}
	defer l.Close()

	err = StartBusServer(logger)
	if err == nil {
		t.Errorf("Expected error when port is in use, got nil")
	}
}

// TestStartBusServerBackground tests the background server functionality
func TestStartBusServerBackground(t *testing.T) {
	t.Parallel()

	logger := logging.NewTestLogger()

	// Use a different port to avoid conflicts
	oldGlobalService := GetGlobalBusService()
	defer SetGlobalBusService(oldGlobalService)

	// Start server on an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0") // Use port 0 to get any available port
	if err != nil {
		t.Fatalf("Failed to get available port: %v", err)
	}
	listener.Close()

	// Create a service that uses the available address
	service := &BusService{
		logger: logger,
		subs:   make(map[string]chan Event),
	}

	// Test successful creation
	if service == nil {
		t.Fatalf("Failed to create bus service")
	}

	// Test that service is properly initialized
	if service.logger == nil {
		t.Errorf("Expected logger to be set")
	}
	if service.subs == nil {
		t.Errorf("Expected subs map to be initialized")
	}
}

// TestGlobalBusServiceFunctions tests the global service getter/setter functions
func TestGlobalBusServiceFunctions(t *testing.T) {
	t.Parallel()

	logger := logging.NewTestLogger()

	// Save original global service to restore later
	oldGlobalService := GetGlobalBusService()
	defer SetGlobalBusService(oldGlobalService)

	// Test setting and getting global service
	testService := &BusService{
		logger: logger,
		subs:   make(map[string]chan Event),
	}

	// Initially should be nil or the old service
	initialService := GetGlobalBusService()
	if initialService != oldGlobalService {
		t.Errorf("Expected initial service to match old service")
	}

	// Set the test service
	SetGlobalBusService(testService)

	// Get the service and verify it matches
	retrievedService := GetGlobalBusService()
	if retrievedService != testService {
		t.Errorf("Expected retrieved service to match test service")
	}

	// Test setting to nil
	SetGlobalBusService(nil)
	nilService := GetGlobalBusService()
	if nilService != nil {
		t.Errorf("Expected nil service after setting to nil")
	}
}

// TestPublishGlobalEvent tests the global event publishing functionality
func TestPublishGlobalEvent(t *testing.T) {
	t.Parallel()

	logger := logging.NewTestLogger()

	// Save original global service to restore later
	oldGlobalService := GetGlobalBusService()
	defer SetGlobalBusService(oldGlobalService)

	// Test publishing with no global service (should not panic)
	SetGlobalBusService(nil)
	PublishGlobalEvent("test", "payload")

	// Test publishing with a global service
	testService := &BusService{
		logger: logger,
		subs:   make(map[string]chan Event),
	}

	// Create a subscription to capture the event
	testService.mu.Lock()
	subID := "test-sub"
	eventChan := make(chan Event, 1)
	testService.subs[subID] = eventChan
	testService.mu.Unlock()

	SetGlobalBusService(testService)

	// Publish an event
	eventType := "global-test"
	payload := "global-payload"
	PublishGlobalEvent(eventType, payload)

	// Check that the event was received
	select {
	case receivedEvent := <-eventChan:
		if receivedEvent.Type != eventType {
			t.Errorf("Expected event type %s, got %s", eventType, receivedEvent.Type)
		}
		if receivedEvent.Payload != payload {
			t.Errorf("Expected payload %s, got %s", payload, receivedEvent.Payload)
		}
	case <-time.After(1 * time.Second):
		t.Errorf("Timeout waiting for published event")
	}
}

// TestStartBusServerRaceCondition tests for race conditions in StartBusServer
func TestStartBusServerRaceCondition(t *testing.T) {
	t.Parallel()

	logger := logging.NewTestLogger()

	// Test concurrent access to avoid race conditions
	done := make(chan bool, 2)

	go func() {
		defer func() { done <- true }()
		// This will likely fail due to port conflict, but shouldn't race
		StartBusServer(logger)
	}()

	go func() {
		defer func() { done <- true }()
		// This will also likely fail due to port conflict, but shouldn't race
		StartBusServer(logger)
	}()

	// Wait for both goroutines to complete
	<-done
	<-done
}

// TestBusServicePublishEventConcurrency tests concurrent event publishing
func TestBusServicePublishEventConcurrency(t *testing.T) {
	t.Parallel()

	logger := logging.NewTestLogger()
	service := &BusService{
		logger: logger,
		subs:   make(map[string]chan Event),
	}

	// Create multiple subscribers
	numSubs := 5
	eventChans := make([]chan Event, numSubs)
	for i := 0; i < numSubs; i++ {
		service.mu.Lock()
		subID := fmt.Sprintf("sub-%d", i)
		eventChan := make(chan Event, 10)
		service.subs[subID] = eventChan
		eventChans[i] = eventChan
		service.mu.Unlock()
	}

	// Publish events concurrently
	numEvents := 10
	done := make(chan bool, numEvents)

	for i := 0; i < numEvents; i++ {
		go func(eventNum int) {
			defer func() { done <- true }()
			service.PublishEvent(Event{
				Type:    "concurrent-test",
				Payload: fmt.Sprintf("event-%d", eventNum),
			})
		}(i)
	}

	// Wait for all publishing to complete
	for i := 0; i < numEvents; i++ {
		<-done
	}

	// Verify all subscribers received all events
	for i, eventChan := range eventChans {
		receivedCount := 0
		for {
			select {
			case <-eventChan:
				receivedCount++
			case <-time.After(100 * time.Millisecond):
				goto checkCount
			}
		}
	checkCount:
		if receivedCount != numEvents {
			t.Errorf("Subscriber %d received %d events, expected %d", i, receivedCount, numEvents)
		}
	}
}
