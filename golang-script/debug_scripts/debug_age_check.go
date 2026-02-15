package main

import (
	"booker-bot/client"
	"fmt"
	"net/http"
	"net/url"
)

func main() {
	fmt.Println("Debugging Age Verification...")
	c := client.NewLowLatencyClient(func() {}, 0, "")

	bypassURL := "https://www.cityheaven.net/niigata/?nenrei=y"
	req, _ := http.NewRequest("GET", bypassURL, nil)
	req.Header.Set("Referer", "https://www.cityheaven.net/niigata/")

	// Create a custom Transport/Client or just use c.Do but inspect the response *before* body close
	// client.Do wraps handling, but returns *http.Response

	resp, err := c.Do(req)
	if err != nil {
		fmt.Printf("Request failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Response Status: %s\n", resp.Status)
	fmt.Printf("Final URL: %s\n", resp.Request.URL.String())

	fmt.Println("Response Headers:")
	for k, v := range resp.Header {
		fmt.Printf("  %s: %v\n", k, v)
	}

	fmt.Println("Cookies in Jar after request:")
	u, _ := url.Parse("https://www.cityheaven.net")
	for _, ck := range c.CookieJar().Cookies(u) {
		fmt.Printf("  %s = %s\n", ck.Name, ck.Value)
	}
}
