package bus

import (
	"errors"
	"fmt"
	"net/rpc"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
)

// busAddr is the address the client connects to; configurable for testing.
var busAddr = "127.0.0.1:12345"

// WaitForEvents listens to the message bus for events.
func WaitForEvents(client *rpc.Client, logger *logging.Logger, eventHandler func(Event) bool) error {
	if client == nil {
		return errors.New("nil client provided")
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
			return errors.New("timeout waiting for events")
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

// SignalResourceCompletion signals that a resource has completed
func SignalResourceCompletion(client *rpc.Client, resourceID, status string, data map[string]interface{}) error {
	if client == nil {
		return errors.New("nil client provided")
	}

	req := SignalCompletionRequest{
		ResourceID: resourceID,
		Status:     status,
		Data:       data,
	}

	var resp SignalCompletionResponse
	err := client.Call("BusService.SignalCompletion", req, &resp)
	if err != nil {
		return fmt.Errorf("failed to signal completion: %w", err)
	}
	if resp.Error != "" {
		return fmt.Errorf("completion signal error: %s", resp.Error)
	}
	if !resp.Success {
		return errors.New("completion signal failed")
	}
	return nil
}

// WaitForResourceCompletion waits for a resource to complete
func WaitForResourceCompletion(client *rpc.Client, resourceID string, timeoutSeconds int64) (*ResourceState, error) {
	if client == nil {
		return nil, errors.New("nil client provided")
	}

	req := WaitForCompletionRequest{
		ResourceID: resourceID,
		Timeout:    timeoutSeconds,
	}

	var resp WaitForCompletionResponse
	err := client.Call("BusService.WaitForCompletion", req, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for completion: %w", err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("wait for completion error: %s", resp.Error)
	}
	if !resp.Success {
		return nil, errors.New("wait for completion failed")
	}

	return &ResourceState{
		ResourceID: resourceID,
		Status:     resp.Status,
		Data:       resp.Data,
	}, nil
}

// PublishEvent publishes an event to the bus
func PublishEvent(client *rpc.Client, eventType, payload, resourceID string, data map[string]interface{}) error {
	if client == nil {
		return errors.New("nil client provided")
	}

	event := Event{
		Type:       eventType,
		Payload:    payload,
		ResourceID: resourceID,
		Data:       data,
		Timestamp:  time.Now().Unix(),
	}

	req := PublishEventRequest{Event: event}
	var resp PublishEventResponse
	err := client.Call("BusService.PublishEvent", req, &resp)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}
	if resp.Error != "" {
		return fmt.Errorf("publish event error: %s", resp.Error)
	}
	if !resp.Success {
		return errors.New("publish event failed")
	}
	return nil
}

// WaitForCleanupSignal waits for cleanup signal instead of file-based approach
func WaitForCleanupSignal(client *rpc.Client, logger *logging.Logger, timeoutSeconds int64) error {
	return WaitForEvents(client, logger, func(event Event) bool {
		if event.Type == "cleanup" || event.Type == "dockercleanup" {
			logger.Info("Cleanup signal received via bus", "payload", event.Payload)
			return true
		}
		return false
	})
}

// StartBusClient initializes and returns an RPC client to connect to the bus.
func StartBusClient() (*rpc.Client, error) {
	client, err := rpc.Dial("tcp", busAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to bus RPC server at %s: %w", busAddr, err)
	}
	return client, nil
}
