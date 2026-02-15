package main

import (
	"booker-bot/client"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func main() {
	c := client.NewLowLatencyClient(func() {}, 0, "")

	// 1. Bypass Age
	bypassURL := "https://www.cityheaven.net/niigata/?nenrei=y"
	reqBP, _ := http.NewRequest("GET", bypassURL, nil)
	reqBP.Header.Set("Referer", "https://www.cityheaven.net/niigata/")
	respBP, _ := c.Do(reqBP)
	respBP.Body.Close()
	fmt.Println("Bypass done.")

	// 2. Try Login (SMS Auth Endpoint)
	loginURL := "https://www.cityheaven.net/niigata/smsauth/smsauthlogin"
	data := url.Values{}
	data.Set("user", "amritacharya")
	data.Set("pass", "12345678")
	data.Set("save_on", "1")

	req, _ := http.NewRequest("POST", loginURL, strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", "https://www.cityheaven.net/niigata/")
	req.Header.Set("Origin", "https://www.cityheaven.net")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := c.Do(req)
	if err != nil {
		fmt.Printf("Login Request Failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	fmt.Printf("Login Status: %s\n", resp.Status)
	fmt.Printf("Final URL: %s\n", resp.Request.URL.String())

	fmt.Println("Response Headers:")
	for k, v := range resp.Header {
		fmt.Printf("  %s: %v\n", k, v)
	}

	if strings.Contains(bodyStr, "マイページ") {
		fmt.Println("SUCCESS: Found 'My Page'")
	} else if strings.Contains(bodyStr, "IDまたはパスワードが違います") {
		fmt.Println("FAILURE: Invalid Credentials")
	} else {
		// Check for lo cookie
		foundLo := false
		u, _ := url.Parse("https://www.cityheaven.net")
		for _, ck := range c.CookieJar().Cookies(u) {
			if ck.Name == "lo" { // Maybe "lo" or "login"
				foundLo = true
				fmt.Printf("COOKIE FOUND: %s=%s\n", ck.Name, ck.Value)
			}
		}
		if foundLo {
			fmt.Println("SUCCESS: 'lo' cookie found (Logged In)")
		} else {
			fmt.Println("UNKNOWN: Check debug_login_api_response.html")
		}
	}

	// Dump cookies
	u, _ := url.Parse("https://www.cityheaven.net")
	fmt.Println("All Cookies:")
	for _, ck := range c.CookieJar().Cookies(u) {
		fmt.Printf("  %s=%s\n", ck.Name, ck.Value)
	}

	os.WriteFile("debug_login_api_response.html", body, 0644)
}
