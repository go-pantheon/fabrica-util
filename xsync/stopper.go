package xsync

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-pantheon/fabrica-util/errors"
)

var (
	// ErrIsStopped is returned when the stopper is already stopped
	ErrIsStopped = errors.New("stopper is already stopped")
	// ErrStopByTrigger is returned when close is triggered
	ErrStopByTrigger = errors.New("stop by trigger")
	// ErrSignalStop is returned when the stopper is stopped by signal
	ErrSignalStop = errors.New("stop by signal")
	// ErrCloseTimeout is returned when the close function timed out
	ErrCloseTimeout = errors.New("close function timed out")
)

// Stoppable lifecycle close manager interface
type Stoppable interface {
	StopTriggerable
	StopWaitable

	// Stop is the stop function should wrap the TurnOff function
	Stop(ctx context.Context) error
	// TurnOff executes the f function with timeout protection
	TurnOff(ctx context.Context, f func(ctx context.Context) error) error
}

// StopTriggerable trigger close interface
type StopTriggerable interface {
	// CloseTriggered returns channel that's closed when close is triggered
	StopTriggered() <-chan struct{}
}

type StopWaitable interface {
	// OnStopping checks if the stop process has started
	OnStopping() bool
	// WaitStopped blocks until the stopper has completed stopping
	WaitStopped() <-chan struct{}
}

var _ Stoppable = (*Stopper)(nil)

// Stopper implements graceful shutdown with timeout
type Stopper struct {
	mu    sync.Mutex
	state *atomic.Int32 // 0=idle, 1=triggered, 2=closing, 3=closed

	trigger     chan struct{} // closed when close is triggered
	stoppedChan chan struct{} // closed when closed

	timeout time.Duration
}

const (
	stateIdle = iota
	stateTriggered
	stateClosing
	stateClosed
)

// NewStopper creates a new Stopper implements Stoppable interface
func NewStopper(timeout time.Duration) *Stopper {
	return &Stopper{
		state:       &atomic.Int32{},
		trigger:     make(chan struct{}),
		stoppedChan: make(chan struct{}),
		timeout:     timeout,
	}
}

func (s *Stopper) TurnOff(ctx context.Context, f func(ctx context.Context) error) (err error) {
	s.triggerStop()

	if !s.toClosingState() {
		return nil // Already closing or closed
	}

	defer s.toClosedState()

	if s.timeout <= 0 {
		return f(ctx)
	}

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)

		err = f(ctx)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ErrCloseTimeout
	}
}

// TriggerStop triggers the stop process (idempotent)
func (s *Stopper) triggerStop() {
	if s.state.CompareAndSwap(stateIdle, stateTriggered) {
		close(s.trigger)
	}
}

// StopTriggered returns a channel that's closed when stop is triggered
func (s *Stopper) StopTriggered() <-chan struct{} {
	return s.trigger
}

// Stop triggers the stop process
func (s *Stopper) Stop(ctx context.Context) error {
	return s.TurnOff(ctx, func(ctx context.Context) error {
		return nil
	})
}

// OnStopping checks if the stop process has started
func (s *Stopper) OnStopping() bool {
	return s.state.Load() >= stateClosing
}

// WaitStopped blocks until the stopper has completed stopping
func (s *Stopper) WaitStopped() <-chan struct{} {
	return s.stoppedChan
}

// stateToClosing attempts to transition to closing state
func (s *Stopper) toClosingState() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	currentState := s.state.Load()
	if currentState >= stateClosing {
		return false // Already closing or closed
	}

	return s.state.CompareAndSwap(currentState, stateClosing)
}

// stateToClosed transitions to closed state
func (s *Stopper) toClosedState() {
	if s.state.CompareAndSwap(stateClosing, stateClosed) {
		close(s.stoppedChan)
	}
}
