// Integration verification program to test etcd and Redis connectivity
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/2bleere/voice-ferry/pkg/config"
	"github.com/2bleere/voice-ferry/pkg/etcd"
	"github.com/2bleere/voice-ferry/pkg/redis"
	v1 "github.com/2bleere/voice-ferry/proto/gen/b2bua/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== Testing etcd and Redis Integrations ===")

	// Test etcd integration
	fmt.Println("\n1. Testing etcd integration...")

	etcdConfig := &config.EtcdConfig{
		Enabled:     true,
		Endpoints:   []string{"http://localhost:2379"},
		DialTimeout: 5 * time.Second,
	}

	etcdClient, err := etcd.NewClient(etcdConfig)
	if err != nil {
		log.Printf("Failed to create etcd client: %v", err)
	} else {
		defer etcdClient.Close()

		// Test etcd health check
		if err := etcdClient.HealthCheck(ctx); err != nil {
			log.Printf("Etcd health check failed: %v", err)
		} else {
			fmt.Println("   ✓ Etcd health check passed")
		}

		// Test routing rule storage
		testRule := &v1.RoutingRule{
			RuleId:      "test-integration-rule",
			Name:        "Integration Test Rule",
			Priority:    100,
			Description: "Test rule for integration verification",
			Enabled:     true,
			Conditions: &v1.RoutingConditions{
				RequestUriRegex: "^sip:.*@test.com$",
				FromUriRegex:    "^sip:.*@client.com$",
				SourceIps:       []string{"192.168.1.0/24"},
				HeaderConditions: map[string]string{
					"User-Agent": "TestClient.*",
				},
			},
			Actions: &v1.RoutingActions{
				NextHopUri: "sip:gateway@192.168.1.100:5060",
				AddHeaders: map[string]string{
					"X-Test-Header": "integration-test",
				},
				RtpengineFlags: "trust-address replace-origin",
			},
			CreatedAt: timestamppb.Now(),
			UpdatedAt: timestamppb.Now(),
		}

		if err := etcdClient.StoreRoutingRule(ctx, testRule); err != nil {
			log.Printf("Failed to store routing rule: %v", err)
		} else {
			fmt.Println("   ✓ Routing rule stored successfully")

			// Retrieve the rule
			if retrievedRule, err := etcdClient.GetRoutingRule(ctx, testRule.RuleId); err != nil {
				log.Printf("Failed to retrieve routing rule: %v", err)
			} else {
				fmt.Printf("   ✓ Routing rule retrieved: %s\n", retrievedRule.Name)
			}

			// Cleanup
			if err := etcdClient.DeleteRoutingRule(ctx, testRule.RuleId); err != nil {
				log.Printf("Failed to delete routing rule: %v", err)
			} else {
				fmt.Println("   ✓ Routing rule deleted successfully")
			}
		}

		// Test configuration storage
		testConfig := map[string]interface{}{
			"test_setting": "integration_test",
			"test_number":  42,
			"timestamp":    time.Now().Unix(),
		}

		if err := etcdClient.StoreConfig(ctx, "integration_test", testConfig); err != nil {
			log.Printf("Failed to store config: %v", err)
		} else {
			fmt.Println("   ✓ Configuration stored successfully")

			var retrievedConfig map[string]interface{}
			if err := etcdClient.GetConfig(ctx, "integration_test", &retrievedConfig); err != nil {
				log.Printf("Failed to retrieve config: %v", err)
			} else {
				fmt.Printf("   ✓ Configuration retrieved: %v\n", retrievedConfig["test_setting"])
			}
		}
	}

	// Test Redis integration
	fmt.Println("\n2. Testing Redis integration...")

	redisConfig := &config.RedisConfig{
		Enabled:      true,
		Host:         "localhost",
		Port:         6379,
		Database:     0,
		PoolSize:     10,
		MinIdleConns: 5,
		Timeout:      5,
	}

	redisClient, err := redis.NewClient(redisConfig)
	if err != nil {
		log.Printf("Failed to create Redis client: %v", err)
	} else {
		defer redisClient.Close()

		// Test Redis health check
		if err := redisClient.HealthCheck(ctx); err != nil {
			log.Printf("Redis health check failed: %v", err)
		} else {
			fmt.Println("   ✓ Redis health check passed")
		}

		// Test session management
		sessionID := "test-call-123"
		sessionData := map[string]interface{}{
			"call_id":     "test-call-123",
			"user_agent":  "Test UA",
			"remote_addr": "192.168.1.100",
			"state":       "active",
			"created_at":  time.Now().Unix(),
		}

		if err := redisClient.StoreSession(ctx, sessionID, sessionData, 1*time.Hour); err != nil {
			log.Printf("Failed to store session: %v", err)
		} else {
			fmt.Println("   ✓ Session stored successfully")

			var retrievedSession map[string]interface{}
			if err := redisClient.GetSessionData(ctx, sessionID, &retrievedSession); err != nil {
				log.Printf("Failed to retrieve session: %v", err)
			} else {
				fmt.Printf("   ✓ Session retrieved: %s\n", sessionID)
			}

			if err := redisClient.DeleteSession(ctx, sessionID); err != nil {
				log.Printf("Failed to delete session: %v", err)
			} else {
				fmt.Println("   ✓ Session deleted successfully")
			}
		}

		// Test call state management
		callID := "test-call-456"
		callState := map[string]interface{}{
			"session_id":  sessionID,
			"call_id":     callID,
			"state":       "ringing",
			"direction":   "inbound",
			"start_time":  time.Now().Unix(),
			"remote_addr": "192.168.1.200",
		}

		if err := redisClient.StoreCallState(ctx, callID, callState); err != nil {
			log.Printf("Failed to store call state: %v", err)
		} else {
			fmt.Println("   ✓ Call state stored successfully")

			var retrievedCallState map[string]interface{}
			if err := redisClient.GetCallState(ctx, callID, &retrievedCallState); err != nil {
				log.Printf("Failed to retrieve call state: %v", err)
			} else {
				fmt.Printf("   ✓ Call state retrieved: %v\n", retrievedCallState["state"])
			}

			if err := redisClient.DeleteCallState(ctx, callID); err != nil {
				log.Printf("Failed to delete call state: %v", err)
			} else {
				fmt.Println("   ✓ Call state deleted successfully")
			}
		}

		// Test cache operations
		cacheKey := "test-cache-key"
		cacheData := map[string]string{
			"user_id":   "user123",
			"username":  "testuser",
			"last_seen": time.Now().Format(time.RFC3339),
		}

		if err := redisClient.StoreCache(ctx, cacheKey, cacheData, 30*time.Minute); err != nil {
			log.Printf("Failed to store cache data: %v", err)
		} else {
			fmt.Println("   ✓ Cache data stored successfully")

			var retrievedCacheData map[string]string
			if err := redisClient.GetCache(ctx, cacheKey, &retrievedCacheData); err != nil {
				log.Printf("Failed to retrieve cache data: %v", err)
			} else {
				fmt.Printf("   ✓ Cache data retrieved: %s\n", retrievedCacheData["user_id"])
			}

			if err := redisClient.DeleteCache(ctx, cacheKey); err != nil {
				log.Printf("Failed to delete cache data: %v", err)
			} else {
				fmt.Println("   ✓ Cache data deleted successfully")
			}
		}

		// Test metrics
		metricKey := "test:metric:calls"
		if err := redisClient.IncrementMetric(ctx, metricKey); err != nil {
			log.Printf("Failed to increment metric: %v", err)
		} else {
			fmt.Println("   ✓ Metric incremented successfully")

			// Increment again to test
			if err := redisClient.IncrementMetric(ctx, metricKey); err != nil {
				log.Printf("Failed to increment metric: %v", err)
			} else {
				if value, err := redisClient.GetMetric(ctx, metricKey); err != nil {
					log.Printf("Failed to retrieve metric: %v", err)
				} else {
					fmt.Printf("   ✓ Metric value retrieved: %d\n", value)
				}
			}
		}

		// Test pub/sub
		testChannel := "test:channel"
		testMessage := map[string]string{
			"event": "test_event",
			"data":  "test data",
		}

		if err := redisClient.Publish(ctx, testChannel, testMessage); err != nil {
			log.Printf("Failed to publish message: %v", err)
		} else {
			fmt.Println("   ✓ Message published successfully")
		}
	}

	fmt.Println("\n=== Integration Tests Completed ===")
}
