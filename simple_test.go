package voice_ferry_test

import (
	"testing"
	"time"
)

// TestGoroutineExecution tests basic goroutine execution without hanging
func TestGoroutineExecution(t *testing.T) {
	t.Log("Testing basic Go test execution...")

	done := make(chan bool, 1)

	go func() {
		t.Log("Goroutine started")
		time.Sleep(100 * time.Millisecond)
		t.Log("Goroutine finished")
		done <- true
	}()

	select {
	case <-done:
		t.Log("Success - no hang detected")
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout - something is hanging")
	}
}

// TestConcurrentGoroutines tests multiple goroutines running concurrently
func TestConcurrentGoroutines(t *testing.T) {
	const numGoroutines = 5
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			t.Logf("Goroutine %d started", id)
			time.Sleep(50 * time.Millisecond)
			t.Logf("Goroutine %d finished", id)
			done <- true
		}(i)
	}

	completed := 0
	timeout := time.After(3 * time.Second)

	for completed < numGoroutines {
		select {
		case <-done:
			completed++
		case <-timeout:
			t.Fatalf("Timeout waiting for goroutines. Only %d/%d completed", completed, numGoroutines)
		}
	}

	t.Logf("All %d goroutines completed successfully", numGoroutines)
}

// BenchmarkGoroutineCreation benchmarks goroutine creation and completion
func BenchmarkGoroutineCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		done := make(chan bool, 1)
		go func() {
			done <- true
		}()
		<-done
	}
}
