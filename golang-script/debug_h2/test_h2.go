package main

import (
	"booker-bot/client"
	"fmt"
	"net/http"
	"time"
)

// Copies from main.go
const (
	SmartproxyUser     = "smart-b3ufblq8e30y_area-JP_state-tokyo"
	SmartproxyPass     = "3FgT4tkDlv9CMd4t"
	SmartproxyEndpoint = "proxy.smartproxy.net:3120"

	TargetURL = "https://www.cityheaven.net/"
)

func main() {
	fmt.Println("Starting HTTP/2 Verification (Standard Transport + Smartproxy)...")

	// Initialize Manager
	pm := client.NewProxyManager()
	pm.EnableSmartproxy(SmartproxyUser, SmartproxyPass, SmartproxyEndpoint)

	// Create client with ForceStandardTransport = true
	c := client.NewLowLatencyClient(func() {}, 0, pm, nil, nil, true)

	req, _ := http.NewRequest("GET", TargetURL, nil)

	start := time.Now()
	// Note: We use c.Do which has debug prints enabled in client.go
	resp, err := c.Do(req)
	if err != nil {
		fmt.Printf("❌ Request failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("✅ Request Successful\n")
	fmt.Printf("   Time: %v\n", time.Since(start))
	fmt.Printf("   Protocol: %s\n", resp.Proto)
	fmt.Printf("   Status: %s\n", resp.Status)

	if resp.ProtoMajor == 2 {
		fmt.Println("\n🚀 HTTP/2 Confirmed! We can leverage multiplexing.")
	} else {
		fmt.Println("\n⚠️  HTTP/1.1 Detected. Server might not support H2 or ALPN failed.")
	}
}
