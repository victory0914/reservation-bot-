package main

import (
	"booker-bot/client"
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	c := client.NewLowLatencyClient(func() {}, 0, "")

	// 1. Fetch Bypass directly
	url1 := "https://www.cityheaven.net/niigata/?nenrei=y"
	fmt.Printf("Fetching %s...\n", url1)
	req1, _ := http.NewRequest("GET", url1, nil)
	req1.Header.Set("Referer", "https://www.cityheaven.net/niigata/")
	resp1, err := c.Do(req1)
	if err != nil {
		panic(err)
	}
	defer resp1.Body.Close()
	body1, _ := io.ReadAll(resp1.Body)
	os.WriteFile("debug_bypass.html", body1, 0644)
	fmt.Printf("Saved %s (%d bytes)\n", "debug_bypass.html", len(body1))

	// 2. Fetch Top Page (Should be clear now if cookie worked)
	url2 := "https://www.cityheaven.net/niigata/"
	fmt.Printf("Fetching %s...\n", url2)
	req2, _ := http.NewRequest("GET", url2, nil)
	resp2, err := c.Do(req2)
	if err != nil {
		panic(err)
	}
	defer resp2.Body.Close()
	body2, _ := io.ReadAll(resp2.Body)
	os.WriteFile("debug_top.html", body2, 0644)
	fmt.Printf("Saved %s (%d bytes)\n", "debug_top.html", len(body2))
}
