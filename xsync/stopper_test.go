package xsync

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewStopper(t *testing.T) {
	t.Parallel()

	timeout := time.Second * 5
	stopper := NewStopper(timeout)

	assert.NotNil(t, stopper)
	assert.NotNil(t, stopper.state)
	assert.NotNil(t, stopper.trigger)
	assert.NotNil(t, stopper.stoppedChan)
	assert.Equal(t, timeout, stopper.timeout)
	assert.Equal(t, int32(stateIdle), stopper.state.Load())
}

func TestStopper_StopTriggered(t *testing.T) {
	t.Parallel()

	stopper := NewStopper(time.Second)

	// Initially not triggered
	select {
	case <-stopper.StopTriggered():
		t.Fatal("StopTriggered should not be closed initially")
	default:
		t.Log("expected behavior")
	}

	// Trigger stop
	stopper.triggerStop()

	// Should be triggered now
	select {
	case <-stopper.StopTriggered():
		// Expected behavior
	case <-time.After(time.Millisecond * 100):
		t.Fatal("StopTriggered should be closed after triggering")
	}
}

func TestStopper_TriggerStopIdempotent(t *testing.T) {
	t.Parallel()

	stopper := NewStopper(time.Second)

	// Trigger multiple times
	stopper.triggerStop()
	stopper.triggerStop()
	stopper.triggerStop()

	// State should be triggered
	assert.Equal(t, int32(stateTriggered), stopper.state.Load())

	// Channel should be closed
	select {
	case <-stopper.StopTriggered():
		// Expected behavior
	default:
		t.Fatal("StopTriggered should be closed")
	}
}

func TestStopper_Stop(t *testing.T) {
	t.Parallel()

	stopper := NewStopper(time.Second)
	ctx := context.Background()

	err := stopper.Stop(ctx)
	assert.NoError(t, err)

	// Should be in closed state
	assert.Equal(t, int32(stateClosed), stopper.state.Load())

	// StopTriggered should be closed
	select {
	case <-stopper.StopTriggered():
		// Expected behavior
	default:
		t.Fatal("StopTriggered should be closed after Stop")
	}

	// WaitStopped should be closed
	select {
	case <-stopper.WaitStopped():
		// Expected behavior
	default:
		t.Fatal("WaitStopped should be closed after Stop")
	}
}

func TestStopper_TurnOffWithFunction(t *testing.T) {
	t.Parallel()

	stopper := NewStopper(time.Second)
	ctx := context.Background()

	executed := false
	err := stopper.TurnOff(ctx, func(ctx context.Context) error {
		executed = true
		return nil
	})

	assert.NoError(t, err)
	assert.True(t, executed)
	assert.Equal(t, int32(stateClosed), stopper.state.Load())
}

func TestStopper_TurnOffWithTimeout(t *testing.T) {
	t.Parallel()

	stopper := NewStopper(time.Millisecond * 100)
	ctx := context.Background()

	executed := make(chan struct{})
	err := stopper.TurnOff(ctx, func(ctx context.Context) error {
		// Simulate long-running function
		select {
		case <-time.After(time.Second):
			close(executed)
		case <-ctx.Done():
			// Function should be cancelled by timeout
			return nil
		}

		return nil
	})

	assert.Equal(t, ErrCloseTimeout, err)
	assert.Equal(t, int32(stateClosed), stopper.state.Load())

	// Function should not complete
	select {
	case <-executed:
		t.Fatal("Function should not complete due to timeout")
	case <-time.After(time.Millisecond * 50):
		t.Log("expected behavior")
	}
}

func TestStopper_TurnOffZeroTimeout(t *testing.T) {
	t.Parallel()

	stopper := NewStopper(0)
	ctx := context.Background()

	executed := false
	err := stopper.TurnOff(ctx, func(ctx context.Context) error {
		executed = true
		// Even with long operation, should not timeout
		time.Sleep(time.Millisecond * 100)

		return nil
	})

	assert.NoError(t, err)
	assert.True(t, executed)
	assert.Equal(t, int32(stateClosed), stopper.state.Load())
}

func TestStopper_OnStopping(t *testing.T) {
	t.Parallel()

	stopper := NewStopper(time.Second)

	// Initially not stopping
	assert.False(t, stopper.OnStopping())

	// After triggering, still not stopping (only triggered)
	stopper.triggerStop()
	assert.False(t, stopper.OnStopping())

	// After entering closing state
	stopper.toClosingState()
	assert.True(t, stopper.OnStopping())
}

func TestStopper_WaitStopped(t *testing.T) {
	t.Parallel()

	stopper := NewStopper(time.Second)

	// Initially not stopped
	select {
	case <-stopper.WaitStopped():
		t.Fatal("WaitStopped should not be closed initially")
	default:
		t.Log("expected behavior")
	}

	// Stop the stopper
	ctx := context.Background()
	err := stopper.Stop(ctx)
	assert.NoError(t, err)

	// Should be stopped now
	select {
	case <-stopper.WaitStopped():
		// Expected behavior
	case <-time.After(time.Millisecond * 100):
		t.Fatal("WaitStopped should be closed after stopping")
	}
}

func TestStopper_MultipleStops(t *testing.T) {
	t.Parallel()

	stopper := NewStopper(time.Second)
	ctx := context.Background()

	// First stop
	err1 := stopper.Stop(ctx)
	assert.NoError(t, err1)

	// Second stop should be no-op
	err2 := stopper.Stop(ctx)
	assert.NoError(t, err2)

	assert.Equal(t, int32(stateClosed), stopper.state.Load())
}

func TestStopper_ConcurrentStops(t *testing.T) {
	t.Parallel()

	stopper := NewStopper(time.Second)
	ctx := context.Background()

	const numGoroutines = 10
	errors := make(chan error, numGoroutines)

	// Start multiple goroutines trying to stop
	for range numGoroutines {
		go func() {
			err := stopper.Stop(ctx)
			errors <- err
		}()
	}

	// Collect all errors
	for range numGoroutines {
		err := <-errors
		assert.NoError(t, err)
	}

	// Should be in closed state
	assert.Equal(t, int32(stateClosed), stopper.state.Load())
}

func TestStopper_ConcurrentTurnOff(t *testing.T) {
	t.Parallel()

	stopper := NewStopper(time.Second)
	ctx := context.Background()

	const numGoroutines = 5

	executed := &atomic.Int32{}
	errors := make(chan error, numGoroutines)

	// Start multiple goroutines trying to turn off
	for range numGoroutines {
		go func() {
			err := stopper.TurnOff(ctx, func(ctx context.Context) error {
				executed.Add(1)
				time.Sleep(time.Millisecond * 10)

				return nil
			})
			errors <- err
		}()
	}

	// Collect all errors
	for range numGoroutines {
		err := <-errors
		assert.NoError(t, err)
	}

	// Only one function should have executed
	assert.Equal(t, int32(1), executed.Load())
	assert.Equal(t, int32(stateClosed), stopper.state.Load())
}

func TestStopper_StateTransitions(t *testing.T) {
	t.Parallel()

	stopper := NewStopper(time.Second)

	// Initial state: idle
	assert.Equal(t, int32(stateIdle), stopper.state.Load())

	// Trigger -> triggered
	stopper.triggerStop()
	assert.Equal(t, int32(stateTriggered), stopper.state.Load())

	// To closing state
	success := stopper.toClosingState()
	assert.True(t, success)
	assert.Equal(t, int32(stateClosing), stopper.state.Load())

	// Try to go to closing again (should fail)
	success = stopper.toClosingState()
	assert.False(t, success)
	assert.Equal(t, int32(stateClosing), stopper.state.Load())

	// To closed state
	stopper.toClosedState()
	assert.Equal(t, int32(stateClosed), stopper.state.Load())

	// Try to go to closed again (should be idempotent)
	stopper.toClosedState()
	assert.Equal(t, int32(stateClosed), stopper.state.Load())
}

func TestStopper_ContextCancellation(t *testing.T) {
	t.Parallel()

	stopper := NewStopper(time.Millisecond * 100)
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context before calling TurnOff
	cancel()

	executed := false
	err := stopper.TurnOff(ctx, func(ctx context.Context) error {
		// Check if context is cancelled in the function
		select {
		case <-ctx.Done():
			// Expected - context should be cancelled due to timeout inheritance
			return nil
		default:
			executed = true
		}

		return nil
	})

	// Since the parent context is cancelled, TurnOff will return ErrCloseTimeout
	assert.Equal(t, ErrCloseTimeout, err)
	assert.False(t, executed)
}

func TestStopper_TurnOffAlreadyClosing(t *testing.T) {
	t.Parallel()

	stopper := NewStopper(time.Second)

	// Manually set to closing state
	stopper.triggerStop()
	stopper.toClosingState()

	executed := false
	err := stopper.TurnOff(context.Background(), func(ctx context.Context) error {
		executed = true

		return nil
	})

	// Should return no error but not execute function
	assert.NoError(t, err)
	assert.False(t, executed)
}

func BenchmarkStopper_TriggerStop(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			stopper := NewStopper(time.Second)
			stopper.triggerStop()
		}
	})
}

func BenchmarkStopper_Stop(b *testing.B) {
	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			stopper := NewStopper(time.Second)
			err := stopper.Stop(ctx)
			assert.NoError(b, err)
		}
	})
}
