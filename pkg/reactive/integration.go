package reactive

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
)

// ReactiveResolver provides reactive extensions to the dependency resolver
type ReactiveResolver struct {
	logger *logging.Logger
	store  *Store[AppState]
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex

	// Event streams
	operationStream *Subject[OperationEvent]
	resourceStream  *Subject[ResourceEvent]
	logStream       *Subject[LogEvent]
	errorStream     *Subject[ErrorEvent]
}

// Event types
type OperationEvent struct {
	Type      string                 `json:"type"`
	Operation Operation              `json:"operation"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type ResourceEvent struct {
	Type     string                 `json:"type"`
	Resource Resource               `json:"resource"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type LogEvent struct {
	Entry LogEntry `json:"entry"`
}

type ErrorEvent struct {
	Error    string                 `json:"error"`
	Source   string                 `json:"source"`
	Context  map[string]interface{} `json:"context,omitempty"`
	Severity string                 `json:"severity"`
}

// NewReactiveResolver creates a new reactive resolver
func NewReactiveResolver(logger *logging.Logger) *ReactiveResolver {
	ctx, cancel := context.WithCancel(context.Background())

	store := NewStore(InitialAppState(), AppReducer)
	store.Use(LoggingMiddleware[AppState]())
	store.Use(ThunkMiddleware[AppState]())

	resolver := &ReactiveResolver{
		logger:          logger,
		store:           store,
		ctx:             ctx,
		cancel:          cancel,
		operationStream: NewSubject[OperationEvent](),
		resourceStream:  NewSubject[ResourceEvent](),
		logStream:       NewSubject[LogEvent](),
		errorStream:     NewSubject[ErrorEvent](),
	}

	// Wire up event streams to state store
	resolver.wireEventStreams()

	return resolver
}

// wireEventStreams connects event streams to the state store
func (r *ReactiveResolver) wireEventStreams() {
	// Operation events -> Store
	r.operationStream.Subscribe(r.ctx, ObserverFunc[OperationEvent]{
		NextFunc: func(event OperationEvent) {
			switch event.Type {
			case "created":
				r.store.Dispatch(AddOperation(event.Operation))
			case "updated":
				r.store.Dispatch(UpdateOperation(event.Operation))
			case "removed":
				r.store.Dispatch(RemoveOperation(event.Operation.ID))
			}
		},
		ErrorFunc: func(err error) {
			r.logger.Error("Operation stream error", "error", err)
		},
	})

	// Resource events -> Store
	r.resourceStream.Subscribe(r.ctx, ObserverFunc[ResourceEvent]{
		NextFunc: func(event ResourceEvent) {
			switch event.Type {
			case "created":
				r.store.Dispatch(AddResource(event.Resource))
			case "updated":
				r.store.Dispatch(UpdateResource(event.Resource))
			case "removed":
				r.store.Dispatch(RemoveResource(event.Resource.ID))
			}
		},
		ErrorFunc: func(err error) {
			r.logger.Error("Resource stream error", "error", err)
		},
	})

	// Log events -> Store
	r.logStream.Subscribe(r.ctx, ObserverFunc[LogEvent]{
		NextFunc: func(event LogEvent) {
			r.store.Dispatch(AddLog(event.Entry))
		},
		ErrorFunc: func(err error) {
			r.logger.Error("Log stream error", "error", err)
		},
	})

	// Error events -> Store
	r.errorStream.Subscribe(r.ctx, ObserverFunc[ErrorEvent]{
		NextFunc: func(event ErrorEvent) {
			r.store.Dispatch(SetError(event.Error))
		},
		ErrorFunc: func(err error) {
			r.logger.Error("Error stream error", "error", err)
		},
	})
}

// State access methods
func (r *ReactiveResolver) GetState() AppState {
	return r.store.GetState()
}

func (r *ReactiveResolver) Subscribe(ctx context.Context, observer Observer[AppState]) Subscription {
	return r.store.Subscribe(ctx, observer)
}

func (r *ReactiveResolver) SelectOperations() *Observable[map[string]Operation] {
	selected := r.store.Select(func(state AppState) interface{} {
		return state.Operations
	})
	return Map(selected, func(ops interface{}) map[string]Operation {
		return ops.(map[string]Operation)
	})
}

func (r *ReactiveResolver) SelectResources() *Observable[map[string]Resource] {
	selected := r.store.Select(func(state AppState) interface{} {
		return state.Resources
	})
	return Map(selected, func(res interface{}) map[string]Resource {
		return res.(map[string]Resource)
	})
}

func (r *ReactiveResolver) SelectLogs() *Observable[[]LogEntry] {
	selected := r.store.Select(func(state AppState) interface{} {
		return state.Logs
	})
	return Map(selected, func(logs interface{}) []LogEntry {
		return logs.([]LogEntry)
	})
}

func (r *ReactiveResolver) SelectError() *Observable[string] {
	selected := r.store.Select(func(state AppState) interface{} {
		return state.Error
	})
	return Map(selected, func(err interface{}) string {
		return err.(string)
	})
}

func (r *ReactiveResolver) SelectLoading() *Observable[bool] {
	selected := r.store.Select(func(state AppState) interface{} {
		return state.Loading
	})
	return Map(selected, func(loading interface{}) bool {
		return loading.(bool)
	})
}

// Event emission methods
func (r *ReactiveResolver) EmitOperationCreated(op Operation, metadata map[string]interface{}) {
	r.operationStream.OnNext(OperationEvent{
		Type:      "created",
		Operation: op,
		Metadata:  metadata,
	})
}

func (r *ReactiveResolver) EmitOperationUpdated(op Operation, metadata map[string]interface{}) {
	r.operationStream.OnNext(OperationEvent{
		Type:      "updated",
		Operation: op,
		Metadata:  metadata,
	})
}

func (r *ReactiveResolver) EmitOperationRemoved(op Operation, metadata map[string]interface{}) {
	r.operationStream.OnNext(OperationEvent{
		Type:      "removed",
		Operation: op,
		Metadata:  metadata,
	})
}

func (r *ReactiveResolver) EmitResourceCreated(res Resource, metadata map[string]interface{}) {
	r.resourceStream.OnNext(ResourceEvent{
		Type:     "created",
		Resource: res,
		Metadata: metadata,
	})
}

func (r *ReactiveResolver) EmitResourceUpdated(res Resource, metadata map[string]interface{}) {
	r.resourceStream.OnNext(ResourceEvent{
		Type:     "updated",
		Resource: res,
		Metadata: metadata,
	})
}

func (r *ReactiveResolver) EmitResourceRemoved(res Resource, metadata map[string]interface{}) {
	r.resourceStream.OnNext(ResourceEvent{
		Type:     "removed",
		Resource: res,
		Metadata: metadata,
	})
}

func (r *ReactiveResolver) EmitLog(level, message, source string, data interface{}) {
	r.logStream.OnNext(LogEvent{
		Entry: LogEntry{
			Level:     level,
			Message:   message,
			Timestamp: time.Now().Unix(),
			Source:    source,
			Data:      data,
		},
	})
}

func (r *ReactiveResolver) EmitError(err, source string, context map[string]interface{}, severity string) {
	r.errorStream.OnNext(ErrorEvent{
		Error:    err,
		Source:   source,
		Context:  context,
		Severity: severity,
	})
}

// Async operations
func (r *ReactiveResolver) StartAsyncOperation(id, opType string, task func() (interface{}, error)) {
	op := Operation{
		ID:        id,
		Type:      opType,
		Status:    "running",
		Progress:  0.0,
		StartTime: time.Now().Unix(),
	}

	r.EmitOperationCreated(op, nil)

	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				op.Status = "error"
				op.EndTime = time.Now().Unix()
				r.EmitOperationUpdated(op, map[string]interface{}{
					"error": fmt.Sprintf("Panic: %v", rec),
				})
				r.EmitError(fmt.Sprintf("Operation %s panicked: %v", id, rec), "async-operation",
					map[string]interface{}{"operationId": id}, "critical")
			}
		}()

		result, err := task()
		op.EndTime = time.Now().Unix()

		if err != nil {
			op.Status = "error"
			r.EmitOperationUpdated(op, map[string]interface{}{
				"error": err.Error(),
			})
			r.EmitError(err.Error(), "async-operation",
				map[string]interface{}{"operationId": id}, "error")
		} else {
			op.Status = "completed"
			op.Progress = 1.0
			r.EmitOperationUpdated(op, map[string]interface{}{
				"result": result,
			})
		}
	}()
}

// Progress tracking
func (r *ReactiveResolver) UpdateOperationProgress(id string, progress float64) {
	state := r.store.GetState()
	if op, exists := state.Operations[id]; exists {
		op.Progress = progress
		if progress >= 1.0 {
			op.Status = "completed"
			op.EndTime = time.Now().Unix()
		}
		r.EmitOperationUpdated(op, nil)
	}
}

// Resource monitoring
func (r *ReactiveResolver) MonitorResourceChanges() *Observable[ResourceEvent] {
	return &Observable[ResourceEvent]{
		subscribe: func(ctx context.Context, observer Observer[ResourceEvent]) Subscription {
			return r.resourceStream.Subscribe(ctx, observer)
		},
	}
}

// Operation monitoring
func (r *ReactiveResolver) MonitorOperationChanges() *Observable[OperationEvent] {
	return &Observable[OperationEvent]{
		subscribe: func(ctx context.Context, observer Observer[OperationEvent]) Subscription {
			return r.operationStream.Subscribe(ctx, observer)
		},
	}
}

// Log monitoring
func (r *ReactiveResolver) MonitorLogs() *Observable[LogEvent] {
	return &Observable[LogEvent]{
		subscribe: func(ctx context.Context, observer Observer[LogEvent]) Subscription {
			return r.logStream.Subscribe(ctx, observer)
		},
	}
}

// Error monitoring
func (r *ReactiveResolver) MonitorErrors() *Observable[ErrorEvent] {
	return &Observable[ErrorEvent]{
		subscribe: func(ctx context.Context, observer Observer[ErrorEvent]) Subscription {
			return r.errorStream.Subscribe(ctx, observer)
		},
	}
}

// State persistence
func (r *ReactiveResolver) EnableStatePersistence(interval time.Duration) {
	// Auto-save state periodically
	Timer(interval).
		Subscribe(r.ctx, ObserverFunc[time.Time]{
			NextFunc: func(t time.Time) {
				r.logger.Debug("Auto-saving state", "timestamp", t)
				// Could save to file system, database, etc.
			},
		})
}

// Metrics collection
func (r *ReactiveResolver) CollectMetrics() *Observable[map[string]interface{}] {
	intervalObs := Interval(5 * time.Second)
	return Map(intervalObs, func(t time.Time) map[string]interface{} {
		state := r.store.GetState()

		completedOps := 0
		runningOps := 0
		errorOps := 0

		for _, op := range state.Operations {
			switch op.Status {
			case "completed":
				completedOps++
			case "running":
				runningOps++
			case "error":
				errorOps++
			}
		}

		return map[string]interface{}{
			"timestamp":            t.Unix(),
			"operations_total":     len(state.Operations),
			"operations_completed": completedOps,
			"operations_running":   runningOps,
			"operations_error":     errorOps,
			"resources_total":      len(state.Resources),
			"logs_total":           len(state.Logs),
			"has_error":            state.Error != "",
			"is_loading":           state.Loading,
		}
	})
}

// Cleanup
func (r *ReactiveResolver) Close() {
	r.cancel()
	r.store.Close()
	r.operationStream.OnComplete()
	r.resourceStream.OnComplete()
	r.logStream.OnComplete()
	r.errorStream.OnComplete()
}

// Helper functions for creating reactive wrappers

// WrapAsyncFunc wraps a function to make it reactive
func (r *ReactiveResolver) WrapAsyncFunc(id, opType string, fn func() (interface{}, error)) func() {
	return func() {
		r.StartAsyncOperation(id, opType, fn)
	}
}

// WrapProgressFunc wraps a function with progress reporting
func (r *ReactiveResolver) WrapProgressFunc(id string, fn func(func(float64)) error) func() error {
	return func() error {
		return fn(func(progress float64) {
			r.UpdateOperationProgress(id, progress)
		})
	}
}

// CreateResourceObserver creates an observer for resource changes
func (r *ReactiveResolver) CreateResourceObserver(resourceID string) *Observable[Resource] {
	resources := r.SelectResources()
	mapped1 := Map(resources, func(resources map[string]Resource) interface{} {
		if res, exists := resources[resourceID]; exists {
			return res
		}
		return nil
	})
	filtered := Filter(mapped1, func(res interface{}) bool {
		return res != nil
	})
	return Map(filtered, func(res interface{}) Resource {
		return res.(Resource)
	})
}

// CreateOperationObserver creates an observer for operation changes
func (r *ReactiveResolver) CreateOperationObserver(operationID string) *Observable[Operation] {
	operations := r.SelectOperations()
	mapped1 := Map(operations, func(operations map[string]Operation) interface{} {
		if op, exists := operations[operationID]; exists {
			return op
		}
		return nil
	})
	filtered := Filter(mapped1, func(op interface{}) bool {
		return op != nil
	})
	return Map(filtered, func(op interface{}) Operation {
		return op.(Operation)
	})
}
