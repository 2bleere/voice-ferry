package main

import (
	"testing"
	"time"

	"github.com/2bleere/voice-ferry/pkg/metrics"
)

func TestMetricsCollectorCreation(t *testing.T) {
	t.Log("Testing metrics collector creation...")

	// Try to create a metrics collector
	done := make(chan bool, 1)

	go func() {
		t.Log("Creating metrics collector...")
		collector := metrics.NewMetricsCollector()
		if collector == nil {
			t.Error("Failed to create metrics collector")
		} else {
			t.Log("Successfully created metrics collector")
		}
		done <- true
	}()

	select {
	case <-done:
		t.Log("Test completed successfully!")
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout! Metrics collector creation is hanging.")
	}
}
