package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/2bleere/voice-ferry/pkg/metrics"
)

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	fmt.Println("Starting debug test...")

	// Try to create a metrics collector
	done := make(chan bool, 1)

	go func() {
		fmt.Println("Creating metrics collector...")
		collector := metrics.NewMetricsCollector()
		fmt.Printf("Created collector: %+v\n", collector != nil)
		done <- true
	}()

	select {
	case <-done:
		fmt.Println("Success!")
	case <-time.After(5 * time.Second):
		fmt.Println("Timeout! Metrics collector creation is hanging.")
		os.Exit(1)
	}
}
