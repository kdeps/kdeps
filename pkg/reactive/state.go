package reactive

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// Action represents a state action
type Action interface {
	Type() string
	Payload() interface{}
}

// SimpleAction is a basic implementation of Action
type SimpleAction struct {
	ActionType    string      `json:"type"`
	ActionPayload interface{} `json:"payload"`
}

func (a SimpleAction) Type() string {
	return a.ActionType
}

func (a SimpleAction) Payload() interface{} {
	return a.ActionPayload
}

func NewAction(actionType string, payload interface{}) Action {
	return SimpleAction{
		ActionType:    actionType,
		ActionPayload: payload,
	}
}

// Reducer function type
type Reducer[T any] func(state T, action Action) T

// Middleware function type
type Middleware[T any] func(store *Store[T], action Action, next func())

// Store represents reactive state management
type Store[T any] struct {
	state       T
	reducer     Reducer[T]
	subject     *BehaviorSubject[T]
	middlewares []Middleware[T]
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewStore creates a new reactive store
func NewStore[T any](initialState T, reducer Reducer[T]) *Store[T] {
	ctx, cancel := context.WithCancel(context.Background())

	store := &Store[T]{
		state:       initialState,
		reducer:     reducer,
		subject:     NewBehaviorSubject(initialState),
		middlewares: make([]Middleware[T], 0),
		ctx:         ctx,
		cancel:      cancel,
	}

	return store
}

// GetState returns the current state
func (s *Store[T]) GetState() T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// Subscribe to state changes
func (s *Store[T]) Subscribe(ctx context.Context, observer Observer[T]) Subscription {
	return s.subject.Subscribe(ctx, observer)
}

// Select creates an observable that emits when a specific part of state changes
func (s *Store[T]) Select(selector func(T) interface{}) *Observable[interface{}] {
	return &Observable[interface{}]{
		subscribe: func(ctx context.Context, observer Observer[interface{}]) Subscription {
			var lastValue interface{}
			var hasValue bool

			return s.subject.Subscribe(ctx, ObserverFunc[T]{
				NextFunc: func(state T) {
					newValue := selector(state)
					if !hasValue || !deepEqual(lastValue, newValue) {
						lastValue = newValue
						hasValue = true
						observer.OnNext(newValue)
					}
				},
				ErrorFunc:    observer.OnError,
				CompleteFunc: observer.OnComplete,
			})
		},
	}
}

// Dispatch an action
func (s *Store[T]) Dispatch(action Action) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create middleware chain
	chain := s.createMiddlewareChain(action)
	chain()
}

func (s *Store[T]) createMiddlewareChain(action Action) func() {
	if len(s.middlewares) == 0 {
		return func() {
			s.executeAction(action)
		}
	}

	return func() {
		index := 0
		var next func()
		next = func() {
			if index >= len(s.middlewares) {
				s.executeAction(action)
				return
			}
			middleware := s.middlewares[index]
			index++
			middleware(s, action, next)
		}
		next()
	}
}

func (s *Store[T]) executeAction(action Action) {
	newState := s.reducer(s.state, action)
	if !deepEqual(s.state, newState) {
		s.state = newState
		s.subject.OnNext(newState)
	}
}

// Use middleware
func (s *Store[T]) Use(middleware Middleware[T]) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.middlewares = append(s.middlewares, middleware)
}

// Close the store
func (s *Store[T]) Close() {
	s.cancel()
	s.subject.OnComplete()
}

// Common middleware
func LoggingMiddleware[T any]() Middleware[T] {
	return func(store *Store[T], action Action, next func()) {
		fmt.Printf("Action: %s, Payload: %v\n", action.Type(), action.Payload())
		next()
	}
}

func ThunkMiddleware[T any]() Middleware[T] {
	return func(store *Store[T], action Action, next func()) {
		if thunk, ok := action.Payload().(func(*Store[T])); ok {
			thunk(store)
		} else {
			next()
		}
	}
}

// Helper functions
func deepEqual(a, b interface{}) bool {
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return string(aJSON) == string(bJSON)
}

// Common state patterns

// AppState represents the global application state
type AppState struct {
	Loading    bool                   `json:"loading"`
	Error      string                 `json:"error,omitempty"`
	Operations map[string]Operation   `json:"operations"`
	Resources  map[string]Resource    `json:"resources"`
	Logs       []LogEntry             `json:"logs"`
	Config     map[string]interface{} `json:"config"`
}

type Operation struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Status    string                 `json:"status"`
	Progress  float64                `json:"progress"`
	StartTime int64                  `json:"startTime"`
	EndTime   int64                  `json:"endTime,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type Resource struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Status   string                 `json:"status"`
	Data     interface{}            `json:"data"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type LogEntry struct {
	Level     string      `json:"level"`
	Message   string      `json:"message"`
	Timestamp int64       `json:"timestamp"`
	Source    string      `json:"source,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

// Action types
const (
	SetLoadingAction      = "SET_LOADING"
	SetErrorAction        = "SET_ERROR"
	ClearErrorAction      = "CLEAR_ERROR"
	AddOperationAction    = "ADD_OPERATION"
	UpdateOperationAction = "UPDATE_OPERATION"
	RemoveOperationAction = "REMOVE_OPERATION"
	AddResourceAction     = "ADD_RESOURCE"
	UpdateResourceAction  = "UPDATE_RESOURCE"
	RemoveResourceAction  = "REMOVE_RESOURCE"
	AddLogAction          = "ADD_LOG"
	ClearLogsAction       = "CLEAR_LOGS"
	UpdateConfigAction    = "UPDATE_CONFIG"
)

// AppReducer is the main reducer for application state
func AppReducer(state AppState, action Action) AppState {
	switch action.Type() {
	case SetLoadingAction:
		return AppState{
			Loading:    action.Payload().(bool),
			Error:      state.Error,
			Operations: state.Operations,
			Resources:  state.Resources,
			Logs:       state.Logs,
			Config:     state.Config,
		}

	case SetErrorAction:
		return AppState{
			Loading:    state.Loading,
			Error:      action.Payload().(string),
			Operations: state.Operations,
			Resources:  state.Resources,
			Logs:       state.Logs,
			Config:     state.Config,
		}

	case ClearErrorAction:
		return AppState{
			Loading:    state.Loading,
			Error:      "",
			Operations: state.Operations,
			Resources:  state.Resources,
			Logs:       state.Logs,
			Config:     state.Config,
		}

	case AddOperationAction:
		operations := make(map[string]Operation)
		for k, v := range state.Operations {
			operations[k] = v
		}
		op := action.Payload().(Operation)
		operations[op.ID] = op

		return AppState{
			Loading:    state.Loading,
			Error:      state.Error,
			Operations: operations,
			Resources:  state.Resources,
			Logs:       state.Logs,
			Config:     state.Config,
		}

	case UpdateOperationAction:
		operations := make(map[string]Operation)
		for k, v := range state.Operations {
			operations[k] = v
		}
		op := action.Payload().(Operation)
		if _, exists := operations[op.ID]; exists {
			operations[op.ID] = op
		}

		return AppState{
			Loading:    state.Loading,
			Error:      state.Error,
			Operations: operations,
			Resources:  state.Resources,
			Logs:       state.Logs,
			Config:     state.Config,
		}

	case RemoveOperationAction:
		operations := make(map[string]Operation)
		for k, v := range state.Operations {
			operations[k] = v
		}
		opID := action.Payload().(string)
		delete(operations, opID)

		return AppState{
			Loading:    state.Loading,
			Error:      state.Error,
			Operations: operations,
			Resources:  state.Resources,
			Logs:       state.Logs,
			Config:     state.Config,
		}

	case AddResourceAction:
		resources := make(map[string]Resource)
		for k, v := range state.Resources {
			resources[k] = v
		}
		res := action.Payload().(Resource)
		resources[res.ID] = res

		return AppState{
			Loading:    state.Loading,
			Error:      state.Error,
			Operations: state.Operations,
			Resources:  resources,
			Logs:       state.Logs,
			Config:     state.Config,
		}

	case UpdateResourceAction:
		resources := make(map[string]Resource)
		for k, v := range state.Resources {
			resources[k] = v
		}
		res := action.Payload().(Resource)
		if _, exists := resources[res.ID]; exists {
			resources[res.ID] = res
		}

		return AppState{
			Loading:    state.Loading,
			Error:      state.Error,
			Operations: state.Operations,
			Resources:  resources,
			Logs:       state.Logs,
			Config:     state.Config,
		}

	case RemoveResourceAction:
		resources := make(map[string]Resource)
		for k, v := range state.Resources {
			resources[k] = v
		}
		resID := action.Payload().(string)
		delete(resources, resID)

		return AppState{
			Loading:    state.Loading,
			Error:      state.Error,
			Operations: state.Operations,
			Resources:  resources,
			Logs:       state.Logs,
			Config:     state.Config,
		}

	case AddLogAction:
		logs := make([]LogEntry, len(state.Logs))
		copy(logs, state.Logs)
		logEntry := action.Payload().(LogEntry)
		logs = append(logs, logEntry)

		// Keep only last 1000 logs
		if len(logs) > 1000 {
			logs = logs[len(logs)-1000:]
		}

		return AppState{
			Loading:    state.Loading,
			Error:      state.Error,
			Operations: state.Operations,
			Resources:  state.Resources,
			Logs:       logs,
			Config:     state.Config,
		}

	case ClearLogsAction:
		return AppState{
			Loading:    state.Loading,
			Error:      state.Error,
			Operations: state.Operations,
			Resources:  state.Resources,
			Logs:       []LogEntry{},
			Config:     state.Config,
		}

	case UpdateConfigAction:
		config := make(map[string]interface{})
		for k, v := range state.Config {
			config[k] = v
		}
		configUpdate := action.Payload().(map[string]interface{})
		for k, v := range configUpdate {
			config[k] = v
		}

		return AppState{
			Loading:    state.Loading,
			Error:      state.Error,
			Operations: state.Operations,
			Resources:  state.Resources,
			Logs:       state.Logs,
			Config:     config,
		}

	default:
		return state
	}
}

// InitialAppState creates the initial application state
func InitialAppState() AppState {
	return AppState{
		Loading:    false,
		Error:      "",
		Operations: make(map[string]Operation),
		Resources:  make(map[string]Resource),
		Logs:       make([]LogEntry, 0),
		Config:     make(map[string]interface{}),
	}
}

// Action creators
func SetLoading(loading bool) Action {
	return NewAction(SetLoadingAction, loading)
}

func SetError(err string) Action {
	return NewAction(SetErrorAction, err)
}

func ClearError() Action {
	return NewAction(ClearErrorAction, nil)
}

func AddOperation(op Operation) Action {
	return NewAction(AddOperationAction, op)
}

func UpdateOperation(op Operation) Action {
	return NewAction(UpdateOperationAction, op)
}

func RemoveOperation(id string) Action {
	return NewAction(RemoveOperationAction, id)
}

func AddResource(res Resource) Action {
	return NewAction(AddResourceAction, res)
}

func UpdateResource(res Resource) Action {
	return NewAction(UpdateResourceAction, res)
}

func RemoveResource(id string) Action {
	return NewAction(RemoveResourceAction, id)
}

func AddLog(entry LogEntry) Action {
	return NewAction(AddLogAction, entry)
}

func ClearLogs() Action {
	return NewAction(ClearLogsAction, nil)
}

func UpdateConfig(config map[string]interface{}) Action {
	return NewAction(UpdateConfigAction, config)
}
