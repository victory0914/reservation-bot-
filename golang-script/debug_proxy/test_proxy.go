package main

import (
	"booker-bot/client"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Copies from main.go
const (
	SmartproxyUser     = "smart-b3ufblq8e30y_area-JP_state-tokyo"
	SmartproxyPass     = "3FgT4tkDlv9CMd4t"
	SmartproxyEndpoint = "proxy.smartproxy.net:3120"
)

func main() {
	fmt.Println("Starting Smartproxy Connection Test (ProxyManager Integration - HTTPS)...")

	// Initialize Manager
	pm := client.NewProxyManager()
	pm.EnableSmartproxy(SmartproxyUser, SmartproxyPass, SmartproxyEndpoint)

	// Create client
	c := client.NewLowLatencyClient(func() {}, 0, pm, nil, nil, true)

	// Make 2 requests
	for i := 1; i <= 2; i++ {
		fmt.Printf("\n--- Request %d ---\n", i)
		targetURL := "https://httpbin.org/ip" // HTTPS to verify CONNECT

		req, _ := http.NewRequest("GET", targetURL, nil)
		start := time.Now()

		resp, err := c.Do(req)
		if err != nil {
			fmt.Printf("❌ Request failed: %v\n", err)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		fmt.Printf("✅ Status: %s | Time: %v\n", resp.Status, time.Since(start))
		fmt.Printf("📍 Body: %s\n", string(body))
	}
}
