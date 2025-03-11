package bus

import (
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
}

type Event struct {
	Type    string
	Payload string
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

func (s *BusService) Subscribe(req SubscribeRequest, resp *SubscribeResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("sub-%d", s.nextID)
	s.nextID++
	s.subs[id] = make(chan Event, 10) // Buffered to prevent blocking
	resp.ID = id
	s.logger.Info("Client subscribed", "id", id)
	return nil
}

func (s *BusService) GetEvent(req EventRequest, resp *EventResponse) error {
	s.mu.Lock()
	ch, ok := s.subs[req.ID]
	s.mu.Unlock()
	if !ok {
		resp.Error = "invalid subscription ID"
		return nil
	}
	select {
	case event := <-ch:
		resp.Event = event
		s.logger.Debug("Delivering event to client", "id", req.ID, "type", event.Type, "payload", event.Payload)
	case <-time.After(5 * time.Second):
		resp.Error = "no events available"
	}
	return nil
}

func (s *BusService) PublishEvent(event Event) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Info("Publishing event", "type", event.Type, "payload", event.Payload)
	for id, ch := range s.subs {
		select {
		case ch <- event:
			s.logger.Debug("Sent event to subscriber", "id", id)
		default:
			s.logger.Warn("Subscriber channel full", "id", id)
		}
	}
}

func StartBusServer(logger *logging.Logger) error {
	service := &BusService{
		logger: logger,
		subs:   make(map[string]chan Event),
	}
	if err := rpc.Register(service); err != nil {
		return fmt.Errorf("failed to register RPC service: %w", err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:12345")
	if err != nil {
		return fmt.Errorf("failed to listen on 127.0.0.1:12345: %w", err)
	}
	logger.Info("Message Bus RPC server started on 127.0.0.1:12345")
	rpc.Accept(listener)
	return nil
}
