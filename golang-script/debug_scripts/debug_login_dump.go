package main

import (
	"booker-bot/client"
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	fmt.Println("Fetching Login Page (with Bypass)...")
	c := client.NewLowLatencyClient(func() {}, 0, "")

	// 1. Bypass Age
	bypassURL := "https://www.cityheaven.net/niigata/?nenrei=y"
	reqBP, _ := http.NewRequest("GET", bypassURL, nil)
	respBP, err := c.Do(reqBP)
	if err != nil {
		fmt.Printf("Bypass failed: %v\n", err)
		return
	}
	respBP.Body.Close()
	fmt.Println("Bypass request sent.")

	// 2. Fetch Login Page
	loginURL := "https://www.cityheaven.net/login/"
	req, _ := http.NewRequest("GET", loginURL, nil)
	resp, err := c.Do(req)
	if err != nil {
		fmt.Printf("Failed to fetch login page: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	os.WriteFile("debug_login_real.html", body, 0644)
	fmt.Printf("Saved real login page to debug_login_real.html (%d bytes)\n", len(body))

	// Log final URL to check redirects
	fmt.Printf("Final URL: %s\n", resp.Request.URL.String())
}
