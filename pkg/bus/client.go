package bus

import (
	"fmt"
	"net/rpc"
	"time"

	"github.com/kdeps/kdeps/pkg/logging" // Correct import.
)

// busAddr is the address the client connects to; configurable for testing.
var busAddr = "127.0.0.1:12345"

// WaitForEvents listens to the message bus for events.
func WaitForEvents(client *rpc.Client, logger *logging.Logger, eventHandler func(Event) bool) error {
	if client == nil {
		return fmt.Errorf("nil client provided")
	}

	logger.Debug("Subscribing to message bus...")

	var subResp SubscribeResponse
	err := client.Call("BusService.Subscribe", SubscribeRequest{}, &subResp)
	if err != nil {
		return fmt.Errorf("failed to subscribe to bus: %w", err)
	}
	if subResp.Error != "" {
		return fmt.Errorf("subscription error: %s", subResp.Error)
	}
	subID := subResp.ID

	logger.Debug("Waiting for events from bus...")

	timeout := time.After(5 * time.Second)

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for events")
		default:
			var resp EventResponse
			err := client.Call("BusService.GetEvent", EventRequest{ID: subID}, &resp)
			if err != nil {
				return fmt.Errorf("failed to get event from bus: %w", err)
			}
			if resp.Error != "" {
				logger.Debug("No events available", "error", resp.Error)
				time.Sleep(500 * time.Millisecond)
				continue
			}

			logger.Info("Received event", "type", resp.Event.Type, "payload", resp.Event.Payload)
			if eventHandler(resp.Event) {
				return nil
			}
		}
	}
}

// StartBusClient initializes and returns an RPC client to connect to the bus.
func StartBusClient() (*rpc.Client, error) {
	client, err := rpc.Dial("tcp", busAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to bus RPC server at %s: %w", busAddr, err)
	}
	return client, nil
}
