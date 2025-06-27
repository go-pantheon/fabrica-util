package xsync

import (
	"bytes"
	"log/slog"
	"runtime"
	"runtime/debug"

	"github.com/go-pantheon/fabrica-util/errors"
)

// DefaultStackSize is the default size for stack traces
const DefaultStackSize = 64 << 10 // 64KB

const (
	initialRoutineIDBuffer = 128
)

// errorWithStack is an error that holds a stack trace.
type errorWithStack struct {
	err   error
	stack []byte
}

// Error returns the message of the original error.
func (e *errorWithStack) Error() string {
	return e.err.Error()
}

// Stack returns the captured stack trace.
func (e *errorWithStack) Stack() []byte {
	return e.stack
}

// Unwrap returns the original wrapped error.
func (e *errorWithStack) Unwrap() error {
	return e.err
}

// Go executes a function in a separate goroutine with panic recovery.
// It logs any errors that occur during execution.
// msg: descriptive message for logging
// fn: function to execute safely
func Go(msg string, fn func() error, filters ...func(err error) bool) {
	filter := func(err error) bool {
		for _, f := range filters {
			if f(err) {
				return true
			}
		}

		return false
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				err := CatchErr(r)
				var stack string
				// Check if the error has a stack trace.
				if st, ok := err.(interface{ Stack() []byte }); ok {
					stack = string(st.Stack())
				}
				slog.Error("goroutine panic recovered",
					"message", msg,
					"error", err.Error(),
					"stack", stack,
				)
			}
		}()

		if err := Run(fn); err != nil {
			if !filter(err) {
				var stack string
				// Check if the error has a stack trace.
				if st, ok := err.(interface{ Stack() []byte }); ok {
					stack = string(st.Stack())
				}
				slog.Error("goroutine error occurred.",
					"message", msg,
					"error", err.Error(),
					"stack", stack,
				)
			}
		}
	}()
}

// Run executes a function with panic recovery.
// Returns the error from the function or a wrapped error if a panic occurred.
func Run(fn func() error) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = CatchErr(p)
		}
	}()

	return fn()
}

// RoutineID returns the current goroutine ID.
// Warning: Only for debug purposes, never use it in production.
// The implementation is based on parsing the runtime stack.
func RoutineID() uint64 {
	buf := make([]byte, initialRoutineIDBuffer)
	n := runtime.Stack(buf, false)
	stack := buf[:n]

	const prefix = "goroutine "
	if !bytes.HasPrefix(stack, []byte(prefix)) {
		return 0
	}

	stack = stack[len(prefix):]
	end := bytes.IndexByte(stack, ' ')

	if end == -1 {
		return 0
	}

	var id uint64

	for _, c := range stack[:end] {
		if c < '0' || c > '9' {
			return 0
		}

		id = id*10 + uint64(c-'0')
	}

	return id
}

// CatchErr creates an error with stack trace from a recovered panic.
// It captures the current stack trace and formats it as part of the error message.
func CatchErr(r any) error {
	if r == nil {
		return nil
	}

	var err error
	switch t := r.(type) {
	case error:
		err = errors.Wrap(t, "goroutine panic recovered")
	case string:
		err = errors.New(t)
	default:
		err = errors.Errorf("%v", r)
	}

	return &errorWithStack{
		err:   err,
		stack: debug.Stack(),
	}
}

// CatchErrWithSize creates an error with a custom sized stack trace from a recovered panic.
// stackSize: the maximum size of the runtime stack in bytes (currently unused)
// Deprecated: stackSize is no longer used, this function is equivalent to CatchErr.
func CatchErrWithSize(r interface{}, _ int) error {
	// Implementation is the same as CatchErr to maintain backwards compatibility
	// while keeping the function signature stable
	return CatchErr(r)
}
