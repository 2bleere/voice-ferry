package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// Command represents an rtpengine ng protocol command
type Command struct {
	Command string   `json:"command"`
	CallID  string   `json:"call-id"`
	FromTag string   `json:"from-tag,omitempty"`
	ToTag   string   `json:"to-tag,omitempty"`
	SDP     string   `json:"sdp,omitempty"`
	Flags   []string `json:"flags,omitempty"`
	Replace []string `json:"replace,omitempty"`
}

func generateCookie() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func main() {
	fmt.Println("=== RTPEngine Bencode Format Testing ===")
	
	// Test 1: Current implementation (what we think is correct)
	fmt.Println("\n1. Testing current implementation:")
	testCurrentImplementation()
	
	// Test 2: Alternative format with cookie in JSON
	fmt.Println("\n2. Testing cookie in JSON format:")
	testCookieInJSON()
	
	// Test 3: Minimal ping command
	fmt.Println("\n3. Testing minimal ping command:")
	testMinimalPing()
	
	// Test 4: Send actual test to RTPEngine
	fmt.Println("\n4. Testing actual RTPEngine connection:")
	testActualRTPEngine()
}

func testCurrentImplementation() {
	cmd := Command{
		Command: "ping",
		CallID:  "",
	}
	
	// Encode command as JSON
	cmdBytes, err := json.Marshal(cmd)
	if err != nil {
		fmt.Printf("Error marshaling: %v\n", err)
		return
	}
	
	// Generate cookie
	cookie := generateCookie()
	
	// Create bencode format: d6:cookie<len>:<cookie>7:command<len>:<json>e
	bencoded := fmt.Sprintf("d6:cookie%d:%s7:command%d:%se", len(cookie), cookie, len(cmdBytes), cmdBytes)
	
	fmt.Printf("Cookie: %s\n", cookie)
	fmt.Printf("JSON: %s\n", string(cmdBytes))
	fmt.Printf("Bencode: %s\n", bencoded)
	fmt.Printf("Length: %d bytes\n", len(bencoded))
}

func testCookieInJSON() {
	// Create command with cookie in JSON
	cookie := generateCookie()
	
	// Create a map to include cookie in JSON
	cmdMap := map[string]interface{}{
		"command": "ping",
		"call-id": "",
		"cookie":  cookie,
	}
	
	cmdBytes, err := json.Marshal(cmdMap)
	if err != nil {
		fmt.Printf("Error marshaling: %v\n", err)
		return
	}
	
	// Create bencode format with cookie also at bencode level
	bencoded := fmt.Sprintf("d6:cookie%d:%s7:command%d:%se", len(cookie), cookie, len(cmdBytes), cmdBytes)
	
	fmt.Printf("Cookie: %s\n", cookie)
	fmt.Printf("JSON: %s\n", string(cmdBytes))
	fmt.Printf("Bencode: %s\n", bencoded)
	fmt.Printf("Length: %d bytes\n", len(bencoded))
}

func testMinimalPing() {
	// Most minimal ping possible
	cookie := "testcookie"
	jsonCmd := `{"command":"ping"}`
	
	bencoded := fmt.Sprintf("d6:cookie%d:%s7:command%d:%se", len(cookie), cookie, len(jsonCmd), jsonCmd)
	
	fmt.Printf("Cookie: %s\n", cookie)
	fmt.Printf("JSON: %s\n", jsonCmd)
	fmt.Printf("Bencode: %s\n", bencoded)
	fmt.Printf("Length: %d bytes\n", len(bencoded))
}

func testActualRTPEngine() {
	// Try to connect to RTPEngine service
	rtpengineAddr := "rtpengine.voice-ferry.svc.cluster.local:22222"
	
	// If running outside cluster, try the service IP directly
	conn, err := net.DialTimeout("udp", rtpengineAddr, 5*time.Second)
	if err != nil {
		// Try direct IP if service name fails
		rtpengineAddr = "10.43.56.159:22222"
		conn, err = net.DialTimeout("udp", rtpengineAddr, 5*time.Second)
		if err != nil {
			fmt.Printf("Failed to connect to RTPEngine: %v\n", err)
			return
		}
	}
	defer conn.Close()
	
	fmt.Printf("Connected to RTPEngine at %s\n", rtpengineAddr)
	
	// Test different formats
	testFormats := []struct {
		name    string
		payload string
	}{
		{
			name:    "Current format",
			payload: createCurrentFormat(),
		},
		{
			name:    "Minimal format",
			payload: createMinimalFormat(),
		},
		{
			name:    "Cookie in JSON format", 
			payload: createCookieInJSONFormat(),
		},
	}
	
	for _, test := range testFormats {
		fmt.Printf("\nTesting %s:\n", test.name)
		fmt.Printf("Payload: %s\n", test.payload)
		
		// Set timeout
		conn.SetDeadline(time.Now().Add(3*time.Second))
		
		// Send command
		_, err := conn.Write([]byte(test.payload))
		if err != nil {
			fmt.Printf("Error sending: %v\n", err)
			continue
		}
		
		// Read response
		buffer := make([]byte, 1024)
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Printf("Error reading: %v\n", err)
			continue
		}
		
		response := string(buffer[:n])
		fmt.Printf("Response: %s\n", response)
		
		// Parse response to see if it's successful
		if contains(response, "result") && contains(response, "ok") {
			fmt.Printf("✅ SUCCESS: %s worked!\n", test.name)
		} else {
			fmt.Printf("❌ FAILED: %s did not work\n", test.name)
		}
	}
}

func createCurrentFormat() string {
	cmd := Command{
		Command: "ping",
		CallID:  "",
	}
	
	cmdBytes, _ := json.Marshal(cmd)
	cookie := generateCookie()
	return fmt.Sprintf("d6:cookie%d:%s7:command%d:%se", len(cookie), cookie, len(cmdBytes), cmdBytes)
}

func createMinimalFormat() string {
	cookie := "testcookie"
	jsonCmd := `{"command":"ping"}`
	return fmt.Sprintf("d6:cookie%d:%s7:command%d:%se", len(cookie), cookie, len(jsonCmd), jsonCmd)
}

func createCookieInJSONFormat() string {
	cookie := generateCookie()
	cmdMap := map[string]interface{}{
		"command": "ping",
		"call-id": "",
		"cookie":  cookie,
	}
	
	cmdBytes, _ := json.Marshal(cmdMap)
	return fmt.Sprintf("d6:cookie%d:%s7:command%d:%se", len(cookie), cookie, len(cmdBytes), cmdBytes)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    (len(s) > len(substr) && 
		     (s[:len(substr)] == substr || 
		      s[len(s)-len(substr):] == substr || 
		      containsMiddle(s, substr))))
}

func containsMiddle(s, substr string) bool {
	for i := 1; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
