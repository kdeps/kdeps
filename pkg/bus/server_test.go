package bus

import (
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
			req := PublishEventRequest{Event: testEvent}
			var resp PublishEventResponse
			testService.PublishEvent(req, &resp)
		}()

		var eventResp EventResponse
		err = client.Call("BusService.GetEvent", EventRequest{ID: subID}, &eventResp)
		if err != nil {
			t.Errorf("GetEvent failed: %v", err)
		}
		if eventResp.Error != "" {
			t.Errorf("GetEvent returned error: %s", eventResp.Error)
		}
		if eventResp.Event.Type != testEvent.Type || eventResp.Event.Payload != testEvent.Payload {
			t.Errorf("Expected event %+v, got %+v", testEvent, eventResp.Event)
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
