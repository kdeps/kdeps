package reactive

import (
	"context"
	"fmt"
	"time"
)

// Example: Basic Observable usage
func ExampleBasicObservable() {
	ctx := context.Background()

	// Create a subject (can both emit and be observed)
	subject := NewSubject[string]()

	// Subscribe to the subject
	subscription := subject.Subscribe(ctx, ObserverFunc[string]{
		NextFunc: func(value string) {
			fmt.Printf("Received: %s\n", value)
		},
		ErrorFunc: func(err error) {
			fmt.Printf("Error: %v\n", err)
		},
		CompleteFunc: func() {
			fmt.Println("Stream completed")
		},
	})
	defer subscription.Unsubscribe()

	// Emit values
	subject.OnNext("Hello")
	subject.OnNext("World")
	subject.OnComplete()
}

// Example: BehaviorSubject usage
func ExampleBehaviorSubject() {
	ctx := context.Background()

	// Create behavior subject with initial value
	behaviorSubject := NewBehaviorSubject("initial")

	// Subscribe (will immediately receive "initial")
	subscription := behaviorSubject.Subscribe(ctx, ObserverFunc[string]{
		NextFunc: func(value string) {
			fmt.Printf("Current value: %s\n", value)
		},
	})
	defer subscription.Unsubscribe()

	// Emit new values
	behaviorSubject.OnNext("updated")
	behaviorSubject.OnNext("final")
}

// Example: Operators usage
func ExampleOperators() {
	ctx := context.Background()

	// Create a subject that emits numbers
	numberSubject := NewSubject[int]()

	// Chain operators
	filtered := Filter(numberSubject, func(n int) bool { return n > 5 })
	mapped := Map(filtered, func(n int) string { return fmt.Sprintf("Number: %d", n) })
	subscription := mapped.Subscribe(ctx, ObserverFunc[string]{
		NextFunc: func(value string) {
			fmt.Println(value)
		},
	})
	defer subscription.Unsubscribe()

	// Emit values (only > 5 will pass through)
	for i := 1; i <= 10; i++ {
		numberSubject.OnNext(i)
	}

	numberSubject.OnComplete()
}

// Example: Timer and Interval observables
func ExampleTimerAndInterval() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Timer: emits once after delay
	timerSub := Timer(1*time.Second).Subscribe(ctx, ObserverFunc[time.Time]{
		NextFunc: func(t time.Time) {
			fmt.Printf("Timer fired at: %s\n", t.Format("15:04:05"))
		},
	})
	defer timerSub.Unsubscribe()

	// Interval: emits repeatedly
	intervalSub := Interval(500*time.Millisecond).Subscribe(ctx, ObserverFunc[time.Time]{
		NextFunc: func(t time.Time) {
			fmt.Printf("Interval tick: %s\n", t.Format("15:04:05.000"))
		},
	})
	defer intervalSub.Unsubscribe()

	// Wait for context timeout
	<-ctx.Done()
}

// Example: From Channel
func ExampleFromChannel() {
	ctx := context.Background()

	// Create a channel
	ch := make(chan string, 5)

	// Create observable from channel
	observable := FromChannel(ch)

	subscription := observable.Subscribe(ctx, ObserverFunc[string]{
		NextFunc: func(value string) {
			fmt.Printf("From channel: %s\n", value)
		},
		CompleteFunc: func() {
			fmt.Println("Channel closed")
		},
	})
	defer subscription.Unsubscribe()

	// Send values to channel
	ch <- "message 1"
	ch <- "message 2"
	ch <- "message 3"
	close(ch)

	// Give time for processing
	time.Sleep(100 * time.Millisecond)
}

// Example: Throttle and Debounce
func ExampleThrottleAndDebounce() {
	ctx := context.Background()

	subject := NewSubject[string]()

	// Throttle: limits emission rate
	throttled := Throttle(subject, 500*time.Millisecond)
	throttleSub := throttled.Subscribe(ctx, ObserverFunc[string]{
		NextFunc: func(value string) {
			fmt.Printf("Throttled: %s at %s\n", value, time.Now().Format("15:04:05.000"))
		},
	})
	defer throttleSub.Unsubscribe()

	// Debounce: waits for quiet period
	debounced := Debounce(subject, 300*time.Millisecond)
	debounceSub := debounced.Subscribe(ctx, ObserverFunc[string]{
		NextFunc: func(value string) {
			fmt.Printf("Debounced: %s at %s\n", value, time.Now().Format("15:04:05.000"))
		},
	})
	defer debounceSub.Unsubscribe()

	// Rapid emissions
	for i := 0; i < 10; i++ {
		subject.OnNext(fmt.Sprintf("rapid-%d", i))
		time.Sleep(100 * time.Millisecond)
	}

	// Wait for debounce
	time.Sleep(500 * time.Millisecond)
}

// Example: State Management
func ExampleStateManagement() {
	// Create store with initial state
	store := NewStore(InitialAppState(), AppReducer)
	defer store.Close()

	ctx := context.Background()

	// Subscribe to state changes
	subscription := store.Subscribe(ctx, ObserverFunc[AppState]{
		NextFunc: func(state AppState) {
			fmt.Printf("State update - Loading: %t, Operations: %d, Error: %s\n",
				state.Loading, len(state.Operations), state.Error)
		},
	})
	defer subscription.Unsubscribe()

	// Dispatch actions
	store.Dispatch(SetLoading(true))

	op := Operation{
		ID:        "example-op",
		Type:      "test",
		Status:    "running",
		Progress:  0.5,
		StartTime: time.Now().Unix(),
	}
	store.Dispatch(AddOperation(op))

	store.Dispatch(SetError("Something went wrong"))
	store.Dispatch(SetLoading(false))
}

// Example: Selector usage
func ExampleSelectors() {
	store := NewStore(InitialAppState(), AppReducer)
	defer store.Close()

	ctx := context.Background()

	// Select only operations
	operationsObs := store.Select(func(state AppState) interface{} {
		return state.Operations
	})

	subscription := operationsObs.Subscribe(ctx, ObserverFunc[interface{}]{
		NextFunc: func(ops interface{}) {
			operations := ops.(map[string]Operation)
			fmt.Printf("Operations changed: count = %d\n", len(operations))
		},
	})
	defer subscription.Unsubscribe()

	// Add operations
	for i := 0; i < 3; i++ {
		op := Operation{
			ID:        fmt.Sprintf("op-%d", i),
			Type:      "example",
			Status:    "running",
			StartTime: time.Now().Unix(),
		}
		store.Dispatch(AddOperation(op))
	}
}

// Example: ReactiveResolver integration
func ExampleReactiveResolver() {
	// This would typically be created by the main application
	resolver := &ReactiveResolver{
		// ... initialization would happen here
	}
	defer resolver.Close()

	ctx := context.Background()

	// Monitor operations
	operationSub := resolver.MonitorOperationChanges().Subscribe(ctx, ObserverFunc[OperationEvent]{
		NextFunc: func(event OperationEvent) {
			fmt.Printf("Operation %s: %s\n", event.Type, event.Operation.ID)
		},
	})
	defer operationSub.Unsubscribe()

	// Monitor errors
	errorSub := resolver.MonitorErrors().Subscribe(ctx, ObserverFunc[ErrorEvent]{
		NextFunc: func(event ErrorEvent) {
			fmt.Printf("Error from %s: %s\n", event.Source, event.Error)
		},
	})
	defer errorSub.Unsubscribe()

	// Emit some events (would normally come from operations)
	resolver.EmitOperationCreated(Operation{
		ID:        "test-op",
		Type:      "example",
		Status:    "running",
		StartTime: time.Now().Unix(),
	}, nil)

	resolver.EmitError("Test error", "example", nil, "warning")
}

// Example: Async operations with progress
func ExampleAsyncOperationWithProgress() {
	resolver := &ReactiveResolver{
		// ... initialization would happen here
	}
	defer resolver.Close()

	ctx := context.Background()

	// Monitor operation progress
	operationObs := resolver.CreateOperationObserver("async-task")
	subscription := operationObs.Subscribe(ctx, ObserverFunc[Operation]{
		NextFunc: func(op Operation) {
			fmt.Printf("Operation %s: %s (%.1f%%)\n", op.ID, op.Status, op.Progress*100)
		},
	})
	defer subscription.Unsubscribe()

	// Start async operation
	resolver.StartAsyncOperation("async-task", "file-processing", func() (interface{}, error) {
		// Simulate work with progress updates
		for i := 0; i <= 10; i++ {
			resolver.UpdateOperationProgress("async-task", float64(i)/10.0)
			time.Sleep(100 * time.Millisecond)
		}
		return "Processing complete", nil
	})

	// Wait for completion
	time.Sleep(2 * time.Second)
}

// Example: Metrics collection
func ExampleMetricsCollection() {
	resolver := &ReactiveResolver{
		// ... initialization would happen here
	}
	defer resolver.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Collect metrics
	metricsSub := resolver.CollectMetrics().Subscribe(ctx, ObserverFunc[map[string]interface{}]{
		NextFunc: func(metrics map[string]interface{}) {
			fmt.Printf("Metrics at %v: %+v\n",
				time.Unix(metrics["timestamp"].(int64), 0).Format("15:04:05"),
				metrics)
		},
	})
	defer metricsSub.Unsubscribe()

	// Simulate some operations
	for i := 0; i < 3; i++ {
		resolver.StartAsyncOperation(fmt.Sprintf("metric-op-%d", i), "test", func() (interface{}, error) {
			time.Sleep(2 * time.Second)
			return "done", nil
		})
	}

	// Wait for metrics
	<-ctx.Done()
}

// Example: Complete workflow
func ExampleCompleteWorkflow() {
	fmt.Println("=== Complete Reactive Workflow Example ===")

	resolver := &ReactiveResolver{
		// ... initialization would happen here
	}
	defer resolver.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Set up comprehensive monitoring
	stateSub := resolver.Subscribe(ctx, ObserverFunc[AppState]{
		NextFunc: func(state AppState) {
			fmt.Printf("[STATE] Loading: %t, Ops: %d, Resources: %d, Logs: %d\n",
				state.Loading, len(state.Operations), len(state.Resources), len(state.Logs))
		},
	})
	defer stateSub.Unsubscribe()

	// Start multiple operations
	resolver.StartAsyncOperation("download", "file-download", func() (interface{}, error) {
		for i := 0; i <= 5; i++ {
			resolver.UpdateOperationProgress("download", float64(i)/5.0)
			time.Sleep(200 * time.Millisecond)
		}
		return "Downloaded 1MB", nil
	})

	resolver.StartAsyncOperation("process", "data-processing", func() (interface{}, error) {
		for i := 0; i <= 3; i++ {
			resolver.UpdateOperationProgress("process", float64(i)/3.0)
			resolver.EmitLog("info", fmt.Sprintf("Processing step %d", i+1), "processor", nil)
			time.Sleep(300 * time.Millisecond)
		}
		return "Processed successfully", nil
	})

	// Add some resources
	resolver.EmitResourceCreated(Resource{
		ID:     "config.json",
		Type:   "config",
		Status: "loaded",
		Data:   map[string]interface{}{"setting": "value"},
	}, nil)

	resolver.EmitResourceCreated(Resource{
		ID:     "database",
		Type:   "connection",
		Status: "connected",
		Data:   "mysql://localhost:3306",
	}, nil)

	// Wait for completion
	<-ctx.Done()
	fmt.Println("Workflow complete")
}
