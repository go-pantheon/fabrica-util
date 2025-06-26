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
	// ErrStopByTrigger is returned when stop is triggered
	ErrStopByTrigger = errors.New("stop by trigger")
	// ErrSignalStop is returned when the stopper is stopped by signal
	ErrSignalStop = errors.New("stop by signal")
	// ErrTurnOffTimeout is returned when the turn off function timed out
	ErrTurnOffTimeout = errors.New("turn off function timed out")
)

// Stoppable lifecycle stop manager interface
type Stoppable interface {
	StopTriggerable
	StopWaitable
	GoStopWaitable

	// Stop is the stop function should wrap the TurnOff function
	Stop(ctx context.Context) error
	// TurnOff executes the f function with timeout protection
	TurnOff(f func() error) error
}

// StopTriggerable trigger stop interface
type StopTriggerable interface {
	// StopTriggered returns channel that's stopped when stop is triggered
	StopTriggered() <-chan struct{}
}

type StopWaitable interface {
	// OnStopping checks if the stop process has started
	OnStopping() bool
	// WaitStopped blocks until the stopper has completed stopping
	WaitStopped() <-chan struct{}
}

type GoStopWaitable interface {
	Go(msg string, fn func() error, filters ...func(err error) bool)
	GoWaitStop(msg string, fn func() error)
	GoAndQuickStop(msg string, fn func() error, stop func() error, filters ...func(err error) bool)
	GoAndFinalStop(msg string, fn func() error, stop func() error, filters ...func(err error) bool)
}

var _ Stoppable = (*Stopper)(nil)

// Stopper implements graceful shutdown with timeout
type Stopper struct {
	mu    sync.Mutex
	state *atomic.Int32 // 0=idle, 1=triggered, 2=stopping, 3=stopped

	trigger     chan struct{} // closed when stop is triggered
	stoppedChan chan struct{} // closed when stopped

	cond  *sync.Cond
	ready bool

	timeout time.Duration
}

const (
	stateIdle = iota
	stateTriggered
	stateStopping
	stateStopped
)

// NewStopper creates a new Stopper implements Stoppable interface
func NewStopper(timeout time.Duration) *Stopper {
	return &Stopper{
		state:       &atomic.Int32{},
		trigger:     make(chan struct{}),
		stoppedChan: make(chan struct{}),
		cond:        sync.NewCond(&sync.Mutex{}),
		timeout:     timeout,
	}
}

func (s *Stopper) TurnOff(f func() error) (err error) {
	s.triggerStop()

	if !s.toStoppingState() {
		return nil // Already stopping or stopped
	}

	defer s.toStoppedState()

	if s.timeout <= 0 {
		return f()
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)

		err = f()
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ErrTurnOffTimeout
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
	return s.TurnOff(func() error {
		return nil
	})
}

// OnStopping checks if the stop process has started
func (s *Stopper) OnStopping() bool {
	return s.state.Load() >= stateStopping
}

// WaitStopped blocks until the stopper has completed stopping
func (s *Stopper) WaitStopped() <-chan struct{} {
	return s.stoppedChan
}

// stateToStopping attempts to transition to stopping state
func (s *Stopper) toStoppingState() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	currentState := s.state.Load()
	if currentState >= stateStopping {
		return false // Already stopping or stopped
	}

	return s.state.CompareAndSwap(currentState, stateStopping)
}

// stateToStopped transitions to stopped state
func (s *Stopper) toStoppedState() {
	if s.state.CompareAndSwap(stateStopping, stateStopped) {
		close(s.stoppedChan)
	}
}

func (s *Stopper) GoAndQuickStop(msg string, fn func() error, stop func() error, filters ...func(err error) bool) {
	s.GoWaitStop(msg, stop)
	s.Go(msg, fn, filters...)
}

func (s *Stopper) Go(msg string, fn func() error, filters ...func(err error) bool) {
	Go(msg, func() error {
		s.cond.L.Lock()

		defer func() {
			s.ready = true
			s.cond.Broadcast()
			s.cond.L.Unlock()
		}()

		return fn()
	}, filters...)
}

func (s *Stopper) GoWaitStop(msg string, fn func() error) {
	Go(msg, func() error {
		s.cond.L.Lock()

		for !s.ready {
			s.cond.Wait()
		}

		s.cond.L.Unlock()

		return fn()
	})
}

func (s *Stopper) GoAndFinalStop(msg string, fn func() error, stop func() error, filters ...func(err error) bool) {
	panic("not implemented")
}
