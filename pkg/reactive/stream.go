package reactive

import (
	"context"
	"sync"
	"time"
)

// Observer interface for reactive patterns
type Observer[T any] interface {
	OnNext(value T)
	OnError(err error)
	OnComplete()
}

// ObserverFunc is a functional implementation of Observer
type ObserverFunc[T any] struct {
	NextFunc     func(T)
	ErrorFunc    func(error)
	CompleteFunc func()
}

func (o ObserverFunc[T]) OnNext(value T) {
	if o.NextFunc != nil {
		o.NextFunc(value)
	}
}

func (o ObserverFunc[T]) OnError(err error) {
	if o.ErrorFunc != nil {
		o.ErrorFunc(err)
	}
}

func (o ObserverFunc[T]) OnComplete() {
	if o.CompleteFunc != nil {
		o.CompleteFunc()
	}
}

// IObservable represents the interface for reactive streams
type IObservable[T any] interface {
	Subscribe(ctx context.Context, observer Observer[T]) Subscription
	SubscribeFunc(ctx context.Context, onNext func(T), onError func(error), onComplete func()) Subscription
}

// Observable represents a reactive stream
type Observable[T any] struct {
	subscribe func(ctx context.Context, observer Observer[T]) Subscription
}

// Subscription represents a subscription to an observable
type Subscription interface {
	Unsubscribe()
	IsSubscribed() bool
}

type subscription struct {
	cancel     context.CancelFunc
	subscribed bool
	mu         sync.RWMutex
}

func (s *subscription) Unsubscribe() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.subscribed {
		s.cancel()
		s.subscribed = false
	}
}

func (s *subscription) IsSubscribed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.subscribed
}

// Subject is both Observable and Observer
type Subject[T any] struct {
	observers []Observer[T]
	mu        sync.RWMutex
	completed bool
	error     error
}

func NewSubject[T any]() *Subject[T] {
	return &Subject[T]{
		observers: make([]Observer[T], 0),
	}
}

func (s *Subject[T]) Subscribe(ctx context.Context, observer Observer[T]) Subscription {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.completed {
		if s.error != nil {
			go observer.OnError(s.error)
		} else {
			go observer.OnComplete()
		}
		return &subscription{
			cancel:     func() {},
			subscribed: false,
		}
	}

	s.observers = append(s.observers, observer)

	ctx, cancel := context.WithCancel(ctx)
	sub := &subscription{
		cancel:     cancel,
		subscribed: true,
	}

	// Remove observer when context is cancelled
	go func() {
		<-ctx.Done()
		s.removeObserver(observer)
	}()

	return sub
}

func (s *Subject[T]) removeObserver(target Observer[T]) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, observer := range s.observers {
		if observer == target {
			s.observers = append(s.observers[:i], s.observers[i+1:]...)
			break
		}
	}
}

func (s *Subject[T]) OnNext(value T) {
	s.mu.RLock()
	observers := make([]Observer[T], len(s.observers))
	copy(observers, s.observers)
	completed := s.completed
	s.mu.RUnlock()

	if completed {
		return
	}

	for _, observer := range observers {
		go observer.OnNext(value)
	}
}

func (s *Subject[T]) OnError(err error) {
	s.mu.Lock()
	if s.completed {
		s.mu.Unlock()
		return
	}

	observers := make([]Observer[T], len(s.observers))
	copy(observers, s.observers)
	s.completed = true
	s.error = err
	s.mu.Unlock()

	for _, observer := range observers {
		go observer.OnError(err)
	}
}

func (s *Subject[T]) OnComplete() {
	s.mu.Lock()
	if s.completed {
		s.mu.Unlock()
		return
	}

	observers := make([]Observer[T], len(s.observers))
	copy(observers, s.observers)
	s.completed = true
	s.mu.Unlock()

	for _, observer := range observers {
		go observer.OnComplete()
	}
}

// SubscribeFunc is a convenience method for Subject to implement IObservable
func (s *Subject[T]) SubscribeFunc(ctx context.Context, onNext func(T), onError func(error), onComplete func()) Subscription {
	return s.Subscribe(ctx, ObserverFunc[T]{
		NextFunc:     onNext,
		ErrorFunc:    onError,
		CompleteFunc: onComplete,
	})
}

// BehaviorSubject holds the last emitted value
type BehaviorSubject[T any] struct {
	*Subject[T]
	lastValue T
	hasValue  bool
}

func NewBehaviorSubject[T any](initialValue T) *BehaviorSubject[T] {
	return &BehaviorSubject[T]{
		Subject:   NewSubject[T](),
		lastValue: initialValue,
		hasValue:  true,
	}
}

func NewBehaviorSubjectEmpty[T any]() *BehaviorSubject[T] {
	return &BehaviorSubject[T]{
		Subject:  NewSubject[T](),
		hasValue: false,
	}
}

func (bs *BehaviorSubject[T]) Subscribe(ctx context.Context, observer Observer[T]) Subscription {
	bs.mu.RLock()
	hasValue := bs.hasValue
	lastValue := bs.lastValue
	bs.mu.RUnlock()

	// Emit last value immediately if available
	if hasValue {
		go observer.OnNext(lastValue)
	}

	return bs.Subject.Subscribe(ctx, observer)
}

func (bs *BehaviorSubject[T]) OnNext(value T) {
	bs.mu.Lock()
	bs.lastValue = value
	bs.hasValue = true
	bs.mu.Unlock()

	bs.Subject.OnNext(value)
}

func (bs *BehaviorSubject[T]) GetValue() (T, bool) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	return bs.lastValue, bs.hasValue
}

// SubscribeFunc for BehaviorSubject to implement IObservable
func (bs *BehaviorSubject[T]) SubscribeFunc(ctx context.Context, onNext func(T), onError func(error), onComplete func()) Subscription {
	return bs.Subscribe(ctx, ObserverFunc[T]{
		NextFunc:     onNext,
		ErrorFunc:    onError,
		CompleteFunc: onComplete,
	})
}

// Observable creation functions
func FromChannel[T any](ch <-chan T) *Observable[T] {
	return &Observable[T]{
		subscribe: func(ctx context.Context, observer Observer[T]) Subscription {
			ctx, cancel := context.WithCancel(ctx)
			sub := &subscription{
				cancel:     cancel,
				subscribed: true,
			}

			go func() {
				defer func() {
					if r := recover(); r != nil {
						observer.OnError(r.(error))
					}
				}()

				for {
					select {
					case <-ctx.Done():
						return
					case value, ok := <-ch:
						if !ok {
							observer.OnComplete()
							return
						}
						observer.OnNext(value)
					}
				}
			}()

			return sub
		},
	}
}

func Interval(duration time.Duration) *Observable[time.Time] {
	return &Observable[time.Time]{
		subscribe: func(ctx context.Context, observer Observer[time.Time]) Subscription {
			ctx, cancel := context.WithCancel(ctx)
			sub := &subscription{
				cancel:     cancel,
				subscribed: true,
			}

			go func() {
				ticker := time.NewTicker(duration)
				defer ticker.Stop()

				for {
					select {
					case <-ctx.Done():
						return
					case t := <-ticker.C:
						observer.OnNext(t)
					}
				}
			}()

			return sub
		},
	}
}

func Timer(duration time.Duration) *Observable[time.Time] {
	return &Observable[time.Time]{
		subscribe: func(ctx context.Context, observer Observer[time.Time]) Subscription {
			ctx, cancel := context.WithCancel(ctx)
			sub := &subscription{
				cancel:     cancel,
				subscribed: true,
			}

			go func() {
				timer := time.NewTimer(duration)
				defer timer.Stop()

				select {
				case <-ctx.Done():
					return
				case t := <-timer.C:
					observer.OnNext(t)
					observer.OnComplete()
				}
			}()

			return sub
		},
	}
}

// Map transforms values using a mapper function
func Map[T, U any](obs IObservable[T], mapper func(T) U) *Observable[U] {
	return &Observable[U]{
		subscribe: func(ctx context.Context, observer Observer[U]) Subscription {
			return obs.Subscribe(ctx, ObserverFunc[T]{
				NextFunc: func(value T) {
					observer.OnNext(mapper(value))
				},
				ErrorFunc:    observer.OnError,
				CompleteFunc: observer.OnComplete,
			})
		},
	}
}

// Filter filters values using a predicate function
func Filter[T any](obs IObservable[T], predicate func(T) bool) *Observable[T] {
	return &Observable[T]{
		subscribe: func(ctx context.Context, observer Observer[T]) Subscription {
			return obs.Subscribe(ctx, ObserverFunc[T]{
				NextFunc: func(value T) {
					if predicate(value) {
						observer.OnNext(value)
					}
				},
				ErrorFunc:    observer.OnError,
				CompleteFunc: observer.OnComplete,
			})
		},
	}
}

// Throttle limits the emission rate
func Throttle[T any](obs IObservable[T], duration time.Duration) *Observable[T] {
	return &Observable[T]{
		subscribe: func(ctx context.Context, observer Observer[T]) Subscription {
			var lastEmit time.Time
			var mu sync.Mutex

			return obs.Subscribe(ctx, ObserverFunc[T]{
				NextFunc: func(value T) {
					mu.Lock()
					now := time.Now()
					if now.Sub(lastEmit) >= duration {
						lastEmit = now
						mu.Unlock()
						observer.OnNext(value)
					} else {
						mu.Unlock()
					}
				},
				ErrorFunc:    observer.OnError,
				CompleteFunc: observer.OnComplete,
			})
		},
	}
}

// Debounce waits for a quiet period before emitting
func Debounce[T any](obs IObservable[T], duration time.Duration) *Observable[T] {
	return &Observable[T]{
		subscribe: func(ctx context.Context, observer Observer[T]) Subscription {
			var timer *time.Timer
			var mu sync.Mutex

			return obs.Subscribe(ctx, ObserverFunc[T]{
				NextFunc: func(value T) {
					mu.Lock()
					if timer != nil {
						timer.Stop()
					}
					timer = time.AfterFunc(duration, func() {
						observer.OnNext(value)
					})
					mu.Unlock()
				},
				ErrorFunc:    observer.OnError,
				CompleteFunc: observer.OnComplete,
			})
		},
	}
}

func (o *Observable[T]) Subscribe(ctx context.Context, observer Observer[T]) Subscription {
	return o.subscribe(ctx, observer)
}

// SubscribeFunc is a convenience method for Observable to implement IObservable
func (o *Observable[T]) SubscribeFunc(ctx context.Context, onNext func(T), onError func(error), onComplete func()) Subscription {
	return o.Subscribe(ctx, ObserverFunc[T]{
		NextFunc:     onNext,
		ErrorFunc:    onError,
		CompleteFunc: onComplete,
	})
}
