package progress

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestProgressBar_New(t *testing.T) {
	total := 100
	pb := New(total)

	assert.Equal(t, total, pb.total)
	assert.Equal(t, 0, pb.current)
	assert.Equal(t, 50, pb.width)
	assert.True(t, pb.startTime.After(time.Time{}))
}

func TestProgressBar_Increment(t *testing.T) {
	pb := New(10)

	assert.Equal(t, 0, pb.current)

	pb.Increment()
	assert.Equal(t, 1, pb.current)

	pb.Increment()
	assert.Equal(t, 2, pb.current)

	// Test multiple increments
	for i := 0; i < 5; i++ {
		pb.Increment()
	}
	assert.Equal(t, 7, pb.current)
}

func TestProgressBar_Increment_ThreadSafe(t *testing.T) {
	pb := New(1000)

	// Simulate concurrent increments
	done := make(chan bool)
	numGoroutines := 10
	incrementsPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < incrementsPerGoroutine; j++ {
				pb.Increment()
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	expectedTotal := numGoroutines * incrementsPerGoroutine
	assert.Equal(t, expectedTotal, pb.current)
}

func TestProgressBar_Finish(t *testing.T) {
	pb := New(100)

	// Increment to 50
	for i := 0; i < 50; i++ {
		pb.Increment()
	}
	assert.Equal(t, 50, pb.current)

	// Finish should set current to total
	pb.Finish()
	assert.Equal(t, 100, pb.current)
}

func TestProgressBar_Finish_AlreadyComplete(t *testing.T) {
	pb := New(10)

	// Complete all iterations
	for i := 0; i < 10; i++ {
		pb.Increment()
	}
	assert.Equal(t, 10, pb.current)

	// Finish should still work
	pb.Finish()
	assert.Equal(t, 10, pb.current)
}

func TestProgressBar_Increment_BeyondTotal(t *testing.T) {
	pb := New(5)

	// Increment beyond total
	for i := 0; i < 10; i++ {
		pb.Increment()
	}

	// Should not exceed total when finished
	pb.Finish()
	assert.Equal(t, 5, pb.current)
}

func TestProgressBar_ZeroTotal(t *testing.T) {
	pb := New(0)

	assert.Equal(t, 0, pb.total)
	assert.Equal(t, 0, pb.current)

	// Should handle zero total gracefully
	pb.Increment()
	assert.Equal(t, 1, pb.current)

	pb.Finish()
	assert.Equal(t, 0, pb.current) // Finish sets to total (0)
}

func TestProgressBar_SingleIncrement(t *testing.T) {
	pb := New(1)

	pb.Increment()
	assert.Equal(t, 1, pb.current)

	pb.Finish()
	assert.Equal(t, 1, pb.current)
}

func TestProgressBar_LargeTotal(t *testing.T) {
	largeTotal := 1000000
	pb := New(largeTotal)

	assert.Equal(t, largeTotal, pb.total)
	assert.Equal(t, 0, pb.current)

	// Test a few increments
	for i := 0; i < 1000; i++ {
		pb.Increment()
	}
	assert.Equal(t, 1000, pb.current)

	pb.Finish()
	assert.Equal(t, largeTotal, pb.current)
}

func TestProgressBar_TimingBehavior(t *testing.T) {
	pb := New(100)
	startTime := pb.startTime

	// Small delay to ensure time passes
	time.Sleep(1 * time.Millisecond)

	pb.Increment()

	// Start time should remain unchanged
	assert.Equal(t, startTime, pb.startTime)

	// Current should be updated
	assert.Equal(t, 1, pb.current)
}

func TestProgressBar_MultipleFinish(t *testing.T) {
	pb := New(10)

	// Increment to 5
	for i := 0; i < 5; i++ {
		pb.Increment()
	}

	pb.Finish()
	assert.Equal(t, 10, pb.current)

	// Multiple finish calls should be safe
	pb.Finish()
	assert.Equal(t, 10, pb.current)

	pb.Finish()
	assert.Equal(t, 10, pb.current)
}

// Benchmark tests for performance
func BenchmarkProgressBar_Increment(b *testing.B) {
	pb := New(b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pb.Increment()
	}
}

func BenchmarkProgressBar_IncrementConcurrent(b *testing.B) {
	pb := New(b.N)

	b.ResetTimer()
	b.RunParallel(func(pb2 *testing.PB) {
		for pb2.Next() {
			pb.Increment()
		}
	})
}
