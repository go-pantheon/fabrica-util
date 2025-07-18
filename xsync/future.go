package xsync

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
)

var (
	// ErrFutureCancelled is returned when a Future is cancelled
	ErrFutureCancelled = errors.New("future was cancelled")
)

// Future represents a value that may not be available yet
type Future[T any] struct {
	mu       sync.RWMutex
	done     chan struct{}
	value    T
	err      error
	complete bool
}

// NewFuture creates a new Future instance
func NewFuture[T any]() *Future[T] {
	return &Future[T]{
		done: make(chan struct{}),
	}
}

// Complete sets the value and error for the Future
func (f *Future[T]) Complete(value T, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.complete {
		return
	}

	f.value = value
	f.err = err
	f.complete = true

	close(f.done)
}

// Get waits for the Future to complete and returns its value and error
func (f *Future[T]) Get() (T, error) {
	<-f.done

	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.value, f.err
}

// GetWithContext waits for the Future to complete with a context
func (f *Future[T]) GetWithContext(ctx context.Context) (T, error) {
	select {
	case <-f.done:
		f.mu.RLock()
		defer f.mu.RUnlock()

		return f.value, f.err
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	}
}

// GetWithTimeout waits for the Future to complete with a timeout
func (f *Future[T]) GetWithTimeout(timeout time.Duration) (T, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return f.GetWithContext(ctx)
}

// IsComplete returns whether the Future has completed
func (f *Future[T]) IsComplete() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.complete
}

// Cancel cancels the Future
func (f *Future[T]) Cancel() {
	f.Complete(f.value, ErrFutureCancelled)
}

// Then creates a new Future that will be completed with the result of the given function
func Then[T, U any](f *Future[T], fn func(T) (U, error)) *Future[U] {
	result := NewFuture[U]()

	Go("future.Then", func() error {
		value, err := f.Get()
		if err != nil {
			result.Complete(result.value, err)
			return nil
		}

		newValue, newErr := fn(value)

		result.Complete(newValue, newErr)

		return nil
	})

	return result
}
