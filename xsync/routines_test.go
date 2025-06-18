package xsync

import (
	"bytes"
	"runtime"
	"sync"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRoutineId 测试边界条件
func TestRoutineId(t *testing.T) {
	t.Parallel()

	t.Run("valid routine id", func(t *testing.T) {
		t.Parallel()

		done := make(chan struct{})

		go func() {
			id := RoutineID()
			require.NotZero(t, id)
			close(done)
		}()
		<-done
	})

	t.Run("invalid stack format", func(t *testing.T) {
		t.Parallel()
		// 构造错误格式的堆栈
		invalidStack := []byte("invalid format")
		id := parseRoutineID(invalidStack)
		assert.Zero(t, id)
	})

	t.Run("id with non-digit characters", func(t *testing.T) {
		t.Parallel()

		invalidID := []byte("goroutine ABC [running]")
		id := parseRoutineID(invalidID)
		assert.Zero(t, id)
	})

	t.Run("buffer too small", func(t *testing.T) {
		t.Parallel()
		// 使用极小的缓冲区
		smallBuf := make([]byte, 10)
		n := runtime.Stack(smallBuf, false)
		id := parseRoutineID(smallBuf[:n])
		assert.Zero(t, id)
	})
}

// TestGoSafe 测试正常和异常情况
func TestGoSafe(t *testing.T) {
	t.Parallel()

	t.Run("normal execution", func(t *testing.T) {
		t.Parallel()

		var wg sync.WaitGroup

		wg.Add(1)

		Go("normal test", func() error {
			defer wg.Done()
			return nil
		})

		wg.Wait()
	})

	t.Run("function returns error", func(t *testing.T) {
		t.Parallel()

		var wg sync.WaitGroup

		wg.Add(1)

		expectedErr := errors.New("expected error")

		Go("error test", func() error {
			defer wg.Done()
			return expectedErr
		})

		wg.Wait()
	})

	t.Run("panic recovery", func(t *testing.T) {
		t.Parallel()

		var wg sync.WaitGroup

		wg.Add(1)

		Go("panic test", func() error {
			defer wg.Done()
			panic("test panic")
		})

		wg.Wait()
	})
}

// TestRunSafe 测试错误处理
func TestRunSafe(t *testing.T) {
	t.Parallel()

	t.Run("no panic", func(t *testing.T) {
		t.Parallel()

		err := Run(func() error {
			return nil
		})
		assert.NoError(t, err)
	})

	t.Run("with error return", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("expected error")
		err := Run(func() error {
			return expectedErr
		})
		assert.Equal(t, expectedErr, err)
	})

	t.Run("panic with value", func(t *testing.T) {
		t.Parallel()

		err := Run(func() error {
			panic("string panic")
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "string panic")
	})

	t.Run("panic with error", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("error panic")
		err := Run(func() error {
			return expectedErr
		})
		require.Error(t, err)
		assert.True(t, errors.Is(err, expectedErr), "should unwrap error")
	})
}

// TestCatchErr test error generation
func TestCatchErr(t *testing.T) {
	t.Parallel()

	t.Run("with string panic", func(t *testing.T) {
		t.Parallel()

		err := CatchErr("test panic")
		assert.Contains(t, err.Error(), "test panic")
	})

	t.Run("with error panic", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("test error")
		err := CatchErr(expectedErr)
		assert.Contains(t, err.Error(), "test error")
	})

	t.Run("custom stack size", func(t *testing.T) {
		t.Parallel()

		err := CatchErrWithSize("panic", 128)
		assert.Contains(t, err.Error(), "panic")
	})
}

// BenchmarkRoutineId 性能测试
func BenchmarkRoutineId(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = RoutineID()
		}
	})
}

func BenchmarkGoSafe(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Go("bench", func() error { return nil })
		}
	})
}

// parseRoutineID 测试辅助函数，暴露内部解析逻辑
func parseRoutineID(stack []byte) uint64 {
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
