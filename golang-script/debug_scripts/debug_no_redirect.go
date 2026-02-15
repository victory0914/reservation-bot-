package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

func main() {
	jar, _ := cookiejar.New(nil)

	// Create client that does NOT follow redirects
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Stop on first redirect
		},
	}

	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

	// Step 1: GET the bypass page first to collect any cookies
	fmt.Println("=== Step 1: GET /niigata/?nenrei=y ===")
	req1, _ := http.NewRequest("GET", "https://www.cityheaven.net/niigata/?nenrei=y", nil)
	req1.Header.Set("User-Agent", ua)
	req1.Header.Set("Referer", "https://www.cityheaven.net/niigata/")
	resp1, err := client.Do(req1)
	if err != nil && !errors.Is(err, http.ErrUseLastResponse) {
		fmt.Printf("Step 1 error: %v\n", err)
		return
	}
	fmt.Printf("Status: %d\n", resp1.StatusCode)
	fmt.Printf("Location: %s\n", resp1.Header.Get("Location"))
	for _, c := range resp1.Header["Set-Cookie"] {
		fmt.Printf("  Set-Cookie: %s\n", c)
	}
	io.ReadAll(resp1.Body)
	resp1.Body.Close()

	// Dump cookies after step 1
	u, _ := url.Parse("https://www.cityheaven.net")
	fmt.Println("\nCookies after step 1:")
	for _, ck := range jar.Cookies(u) {
		fmt.Printf("  %s = %s\n", ck.Name, ck.Value)
	}

	// Step 2: POST to /login/exec/ (global, no niigata prefix)
	fmt.Println("\n=== Step 2: POST /login/exec/ (no redirect) ===")
	data := url.Values{}
	data.Set("user", "amritacharya")
	data.Set("pass", "12345678")
	data.Set("save_on", "1")

	req2, _ := http.NewRequest("POST", "https://www.cityheaven.net/login/exec/", strings.NewReader(data.Encode()))
	req2.Header.Set("User-Agent", ua)
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req2.Header.Set("Referer", "https://www.cityheaven.net/niigata/?nenrei=y")
	req2.Header.Set("Origin", "https://www.cityheaven.net")

	resp2, err := client.Do(req2)
	if err != nil && !errors.Is(err, http.ErrUseLastResponse) {
		fmt.Printf("Step 2 error: %v\n", err)
		return
	}
	fmt.Printf("Status: %d\n", resp2.StatusCode)
	fmt.Printf("Location: %s\n", resp2.Header.Get("Location"))
	for _, c := range resp2.Header["Set-Cookie"] {
		fmt.Printf("  Set-Cookie: %s\n", c)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	fmt.Printf("Body length: %d\n", len(body2))
	if len(body2) > 0 && len(body2) < 1000 {
		fmt.Printf("Body: %s\n", string(body2))
	}

	fmt.Println("\nCookies after step 2:")
	for _, ck := range jar.Cookies(u) {
		fmt.Printf("  %s = %s\n", ck.Name, ck.Value)
	}

	// Step 3: Also try POST to /niigata/login/exec/
	fmt.Println("\n=== Step 3: POST /niigata/login/exec/ (no redirect) ===")
	req3, _ := http.NewRequest("POST", "https://www.cityheaven.net/niigata/login/exec/", strings.NewReader(data.Encode()))
	req3.Header.Set("User-Agent", ua)
	req3.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req3.Header.Set("Referer", "https://www.cityheaven.net/niigata/?nenrei=y")
	req3.Header.Set("Origin", "https://www.cityheaven.net")

	resp3, err := client.Do(req3)
	if err != nil && !errors.Is(err, http.ErrUseLastResponse) {
		fmt.Printf("Step 3 error: %v\n", err)
		return
	}
	fmt.Printf("Status: %d\n", resp3.StatusCode)
	fmt.Printf("Location: %s\n", resp3.Header.Get("Location"))
	for _, c := range resp3.Header["Set-Cookie"] {
		fmt.Printf("  Set-Cookie: %s\n", c)
	}
	body3, _ := io.ReadAll(resp3.Body)
	resp3.Body.Close()
	fmt.Printf("Body length: %d\n", len(body3))
	if len(body3) > 0 && len(body3) < 1000 {
		fmt.Printf("Body: %s\n", string(body3))
	}

	fmt.Println("\nCookies after step 3:")
	for _, ck := range jar.Cookies(u) {
		fmt.Printf("  %s = %s\n", ck.Name, ck.Value)
	}

	// Step 4: Try POST to /n/Z1Login/ endpoint
	fmt.Println("\n=== Step 4: POST /n/Z1Login/ (no redirect) ===")
	req4, _ := http.NewRequest("POST", "https://www.cityheaven.net/n/Z1Login/", strings.NewReader(data.Encode()))
	req4.Header.Set("User-Agent", ua)
	req4.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req4.Header.Set("Referer", "https://www.cityheaven.net/niigata/?nenrei=y")
	req4.Header.Set("Origin", "https://www.cityheaven.net")

	resp4, err := client.Do(req4)
	if err != nil && !errors.Is(err, http.ErrUseLastResponse) {
		fmt.Printf("Step 4 error: %v\n", err)
		return
	}
	fmt.Printf("Status: %d\n", resp4.StatusCode)
	fmt.Printf("Location: %s\n", resp4.Header.Get("Location"))
	for _, c := range resp4.Header["Set-Cookie"] {
		fmt.Printf("  Set-Cookie: %s\n", c)
	}
	body4, _ := io.ReadAll(resp4.Body)
	resp4.Body.Close()
	fmt.Printf("Body length: %d\n", len(body4))
	if len(body4) > 0 && len(body4) < 1000 {
		fmt.Printf("Body: %s\n", string(body4))
	}

	fmt.Println("\nFinal cookies:")
	for _, ck := range jar.Cookies(u) {
		fmt.Printf("  %s = %s\n", ck.Name, ck.Value)
	}
}
