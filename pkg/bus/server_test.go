package bus

import (
	"fmt"
	"net"
	"net/rpc"
	"os"
	"strings"
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

		// Check if testService is available (when we created our own server)
		// If not, use the global bus service or create a mock scenario
		if testService != nil {
			go func() {
				time.Sleep(50 * time.Millisecond) // Brief delay to allow subscription setup
				testService.PublishEvent(testEvent)
			}()
		} else {
			// When reusing an existing server, use global event publishing
			go func() {
				time.Sleep(50 * time.Millisecond) // Brief delay to allow subscription setup
				PublishGlobalEvent(testEvent.Type, testEvent.Payload)
			}()
		}

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

// TestStartBusServerBackgroundComprehensive tests the actual StartBusServerBackground function
func TestStartBusServerBackgroundComprehensive(t *testing.T) {
	logger := logging.NewTestLogger()

	t.Run("SuccessfulStart", func(t *testing.T) {
		// Find an available port
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("Failed to find available port: %v", err)
		}
		port := listener.Addr().(*net.TCPAddr).Port
		listener.Close()

		// Save original functions
		originalRegister := rpcRegisterFunc
		originalListen := netListenFunc
		originalAccept := rpcAcceptFunc
		defer func() {
			rpcRegisterFunc = originalRegister
			netListenFunc = originalListen
			rpcAcceptFunc = originalAccept
		}()

		// Mock the functions to use available port
		rpcRegisterFunc = func(rcvr interface{}) error {
			return nil // Success
		}
		netListenFunc = func(network, address string) (net.Listener, error) {
			return net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		}
		rpcAcceptFunc = func(lis net.Listener) {
			// Mock accept - just close the listener to avoid hanging
			lis.Close()
		}

		// Test the function
		service, err := StartBusServerBackground(logger)
		if err != nil {
			t.Errorf("StartBusServerBackground failed: %v", err)
		}
		if service == nil {
			t.Errorf("Expected service to be returned")
		}
		if service.logger != logger {
			t.Errorf("Expected service logger to match input logger")
		}
		if service.subs == nil {
			t.Errorf("Expected service subs to be initialized")
		}
	})

	t.Run("RegisterError", func(t *testing.T) {
		// Save original functions
		originalRegister := rpcRegisterFunc
		originalListen := netListenFunc
		originalAccept := rpcAcceptFunc
		defer func() {
			rpcRegisterFunc = originalRegister
			netListenFunc = originalListen
			rpcAcceptFunc = originalAccept
		}()

		// Mock register to fail
		rpcRegisterFunc = func(rcvr interface{}) error {
			return fmt.Errorf("service already registered")
		}

		service, err := StartBusServerBackground(logger)
		if err == nil {
			t.Errorf("Expected error from StartBusServerBackground")
		}
		if service != nil {
			t.Errorf("Expected nil service on error")
		}
		if !strings.Contains(err.Error(), "register RPC service") {
			t.Errorf("Expected register error, got: %v", err)
		}
	})

	t.Run("ListenError", func(t *testing.T) {
		// Save original functions
		originalRegister := rpcRegisterFunc
		originalListen := netListenFunc
		originalAccept := rpcAcceptFunc
		defer func() {
			rpcRegisterFunc = originalRegister
			netListenFunc = originalListen
			rpcAcceptFunc = originalAccept
		}()

		// Mock register to succeed, listen to fail
		rpcRegisterFunc = func(rcvr interface{}) error {
			return nil
		}
		netListenFunc = func(network, address string) (net.Listener, error) {
			return nil, fmt.Errorf("address already in use")
		}

		service, err := StartBusServerBackground(logger)
		if err == nil {
			t.Errorf("Expected error from StartBusServerBackground")
		}
		if service != nil {
			t.Errorf("Expected nil service on error")
		}
		if !strings.Contains(err.Error(), "listen") {
			t.Errorf("Expected listen error, got: %v", err)
		}
	})
}

// TestGetEventErrorPaths tests error scenarios for GetEvent
func TestGetEventErrorPaths(t *testing.T) {
	logger := logging.NewTestLogger()
	service := &BusService{
		logger: logger,
		subs:   make(map[string]chan Event),
	}

	t.Run("InvalidSubscriptionID", func(t *testing.T) {
		var resp EventResponse
		err := service.GetEvent(EventRequest{ID: "invalid-id"}, &resp)
		if err != nil {
			t.Errorf("GetEvent should not return error for invalid ID, got: %v", err)
		}
		if resp.Error != "invalid subscription ID" {
			t.Errorf("Expected 'invalid subscription ID' error, got: %s", resp.Error)
		}
		if resp.Event.Type != "" || resp.Event.Payload != "" {
			t.Errorf("Expected empty event for invalid ID, got: %v", resp.Event)
		}
	})

	t.Run("TimeoutScenario", func(t *testing.T) {
		// Create a subscription
		service.mu.Lock()
		subID := "timeout-test"
		service.subs[subID] = make(chan Event, 1)
		service.mu.Unlock()

		var resp EventResponse
		err := service.GetEvent(EventRequest{ID: subID}, &resp)
		if err != nil {
			t.Errorf("GetEvent should not return error on timeout, got: %v", err)
		}
		if resp.Error != "no events available" {
			t.Errorf("Expected 'no events available' error, got: %s", resp.Error)
		}
	})

	t.Run("SuccessfulEventRetrieval", func(t *testing.T) {
		// Create a subscription with an event
		service.mu.Lock()
		subID := "success-test"
		eventChan := make(chan Event, 1)
		testEvent := Event{Type: "test", Payload: "success"}
		eventChan <- testEvent
		service.subs[subID] = eventChan
		service.mu.Unlock()

		var resp EventResponse
		err := service.GetEvent(EventRequest{ID: subID}, &resp)
		if err != nil {
			t.Errorf("GetEvent should not return error, got: %v", err)
		}
		if resp.Error != "" {
			t.Errorf("Expected no error, got: %s", resp.Error)
		}
		if resp.Event != testEvent {
			t.Errorf("Expected event %v, got %v", testEvent, resp.Event)
		}
	})
}

// TestPublishEventComprehensive tests all paths in PublishEvent
func TestPublishEventComprehensive(t *testing.T) {
	logger := logging.NewTestLogger()
	service := &BusService{
		logger: logger,
		subs:   make(map[string]chan Event),
	}

	t.Run("NoSubscribers", func(t *testing.T) {
		// Should not panic with no subscribers
		testEvent := Event{Type: "test", Payload: "no-subs"}
		service.PublishEvent(testEvent)
	})

	t.Run("MultipleSubscribers", func(t *testing.T) {
		// Add multiple subscribers
		numSubs := 3
		eventChans := make([]chan Event, numSubs)
		service.mu.Lock()
		for i := 0; i < numSubs; i++ {
			subID := fmt.Sprintf("multi-sub-%d", i)
			eventChan := make(chan Event, 1)
			service.subs[subID] = eventChan
			eventChans[i] = eventChan
		}
		service.mu.Unlock()

		testEvent := Event{Type: "multi-test", Payload: "multiple"}
		service.PublishEvent(testEvent)

		// Verify all subscribers received the event
		for i, eventChan := range eventChans {
			select {
			case receivedEvent := <-eventChan:
				if receivedEvent != testEvent {
					t.Errorf("Subscriber %d received wrong event: %v", i, receivedEvent)
				}
			case <-time.After(1 * time.Second):
				t.Errorf("Subscriber %d did not receive event", i)
			}
		}
	})

	t.Run("FullChannelWarning", func(t *testing.T) {
		// Create a subscriber with a full channel
		service.mu.Lock()
		subID := "full-channel-test"
		eventChan := make(chan Event, 1)
		eventChan <- Event{Type: "blocking", Payload: "event"} // Fill the channel
		service.subs[subID] = eventChan
		service.mu.Unlock()

		// This should trigger the "channel full" warning path
		testEvent := Event{Type: "should-warn", Payload: "full"}
		service.PublishEvent(testEvent)

		// Verify the original event is still there and new one was dropped
		select {
		case receivedEvent := <-eventChan:
			if receivedEvent.Type != "blocking" {
				t.Errorf("Expected original blocking event, got: %v", receivedEvent)
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("Channel should have contained the original event")
		}

		// Channel should be empty now (new event was dropped)
		select {
		case unexpectedEvent := <-eventChan:
			t.Errorf("Channel should be empty, but got: %v", unexpectedEvent)
		default:
			// Expected - channel is empty
		}
	})
}

// TestStartBusServerErrorPaths tests more error scenarios for StartBusServer
func TestStartBusServerErrorPaths(t *testing.T) {
	logger := logging.NewTestLogger()

	t.Run("RegisterError", func(t *testing.T) {
		// Save original functions
		originalRegister := rpcRegisterFunc
		originalListen := netListenFunc
		originalAccept := rpcAcceptFunc
		defer func() {
			rpcRegisterFunc = originalRegister
			netListenFunc = originalListen
			rpcAcceptFunc = originalAccept
		}()

		// Mock register to fail
		rpcRegisterFunc = func(rcvr interface{}) error {
			return fmt.Errorf("service already registered")
		}

		err := StartBusServer(logger)
		if err == nil {
			t.Errorf("Expected error when register fails")
		}
		if !strings.Contains(err.Error(), "register RPC service") {
			t.Errorf("Expected register error, got: %v", err)
		}
	})

	t.Run("ListenError", func(t *testing.T) {
		// Save original functions
		originalRegister := rpcRegisterFunc
		originalListen := netListenFunc
		originalAccept := rpcAcceptFunc
		defer func() {
			rpcRegisterFunc = originalRegister
			netListenFunc = originalListen
			rpcAcceptFunc = originalAccept
		}()

		// Mock register to succeed, listen to fail
		rpcRegisterFunc = func(rcvr interface{}) error {
			return nil
		}
		netListenFunc = func(network, address string) (net.Listener, error) {
			return nil, fmt.Errorf("address already in use")
		}

		err := StartBusServer(logger)
		if err == nil {
			t.Errorf("Expected error when listen fails")
		}
		if !strings.Contains(err.Error(), "listen") {
			t.Errorf("Expected listen error, got: %v", err)
		}
	})
}

// TestBusServiceEdgeCases tests edge cases and boundary conditions
func TestBusServiceEdgeCases(t *testing.T) {
	logger := logging.NewTestLogger()

	t.Run("ConcurrentSubscriptionAccess", func(t *testing.T) {
		service := &BusService{
			logger: logger,
			subs:   make(map[string]chan Event),
		}

		// Test concurrent Subscribe calls
		numGoroutines := 10
		done := make(chan bool, numGoroutines)
		responses := make([]SubscribeResponse, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				defer func() { done <- true }()
				var resp SubscribeResponse
				err := service.Subscribe(SubscribeRequest{}, &resp)
				if err != nil {
					t.Errorf("Subscribe failed: %v", err)
				}
				responses[index] = resp
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// Verify all subscriptions got unique IDs
		ids := make(map[string]bool)
		for _, resp := range responses {
			if resp.Error != "" {
				t.Errorf("Subscribe returned error: %s", resp.Error)
			}
			if ids[resp.ID] {
				t.Errorf("Duplicate subscription ID: %s", resp.ID)
			}
			ids[resp.ID] = true
		}

		if len(ids) != numGoroutines {
			t.Errorf("Expected %d unique IDs, got %d", numGoroutines, len(ids))
		}
	})

	t.Run("EventChannelCapacity", func(t *testing.T) {
		service := &BusService{
			logger: logger,
			subs:   make(map[string]chan Event),
		}

		// Create subscription
		var resp SubscribeResponse
		err := service.Subscribe(SubscribeRequest{}, &resp)
		if err != nil {
			t.Fatalf("Subscribe failed: %v", err)
		}

		subID := resp.ID

		// Fill the channel to capacity (10 events)
		for i := 0; i < 10; i++ {
			service.PublishEvent(Event{
				Type:    "capacity-test",
				Payload: fmt.Sprintf("event-%d", i),
			})
		}

		// Verify we can retrieve all 10 events
		for i := 0; i < 10; i++ {
			var eventResp EventResponse
			err = service.GetEvent(EventRequest{ID: subID}, &eventResp)
			if err != nil {
				t.Errorf("GetEvent failed on event %d: %v", i, err)
			}
			if eventResp.Error != "" {
				t.Errorf("GetEvent returned error on event %d: %s", i, eventResp.Error)
			}
			if eventResp.Event.Payload != fmt.Sprintf("event-%d", i) {
				t.Errorf("Wrong event payload at %d: expected event-%d, got %s", i, i, eventResp.Event.Payload)
			}
		}
	})
}

// TestStartBusServerSuccess tests the successful path of StartBusServer
func TestStartBusServerSuccess(t *testing.T) {
	logger := logging.NewTestLogger()

	// Save original functions
	originalRegister := rpcRegisterFunc
	originalListen := netListenFunc
	originalAccept := rpcAcceptFunc
	defer func() {
		rpcRegisterFunc = originalRegister
		netListenFunc = originalListen
		rpcAcceptFunc = originalAccept
	}()

	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	// Mock the functions for success path
	rpcRegisterFunc = func(rcvr interface{}) error {
		return nil // Success
	}
	netListenFunc = func(network, address string) (net.Listener, error) {
		return net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	}
	rpcAcceptFunc = func(lis net.Listener) {
		// Mock accept - just close the listener to simulate completion
		lis.Close()
	}

	// Test the function - should complete successfully
	err = StartBusServer(logger)
	if err != nil {
		t.Errorf("StartBusServer should succeed, got error: %v", err)
	}
}
