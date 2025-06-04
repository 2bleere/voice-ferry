package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/2bleere/voice-ferry/pkg/config"
	"github.com/2bleere/voice-ferry/pkg/rtpengine"
)

func main() {
	fmt.Println("=== Testing RTPEngine Client ===")

	// Create RTPEngine client configuration
	rtpConfig := config.RTPEngineConfig{
		Instances: []config.RTPEngineInstance{
			{
				ID:      "rtpengine-1",
				Address: "192.168.1.208:22222",
				Enabled: true,
			},
		},
		Timeout: 5 * time.Second,
	}

	// Create client
	client, err := rtpengine.NewClient(rtpConfig)
	if err != nil {
		log.Fatalf("Failed to create RTPEngine client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Test ping
	fmt.Println("Testing ping...")
	response, err := client.Ping(ctx, "rtpengine-1")
	if err != nil {
		log.Printf("Ping failed: %v", err)
	} else {
		fmt.Printf("✓ Ping successful: %+v\n", response)
	}

	fmt.Println("RTPEngine client test completed!")
}
