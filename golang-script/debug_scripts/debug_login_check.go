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

	url := "https://www.cityheaven.net/login/?nenrei=y"
	fmt.Printf("Fetching %s...\n", url)
	req, _ := http.NewRequest("GET", url, nil)
	// Add Referer just in case
	req.Header.Set("Referer", "https://www.cityheaven.net/niigata/?nenrei=y")

	resp, err := c.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	os.WriteFile("debug_login_check.html", body, 0644)
	fmt.Printf("Saved %s (%d bytes)\n", "debug_login_check.html", len(body))
}
