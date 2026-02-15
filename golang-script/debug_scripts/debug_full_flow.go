package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
)

func main() {
	jar, _ := cookiejar.New(nil)

	// Standard http.Client WITH redirect following and a cookie jar
	client := &http.Client{
		Jar: jar,
	}

	ua := "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36"

	dumpCookies := func(label string) {
		u, _ := url.Parse("https://www.cityheaven.net")
		fmt.Printf("\n--- Cookies after %s ---\n", label)
		for _, ck := range jar.Cookies(u) {
			fmt.Printf("  %s = %s\n", ck.Name, ck.Value)
		}
		// Also check img.cityheaven.net cookies
		u2, _ := url.Parse("http://img.cityheaven.net")
		for _, ck := range jar.Cookies(u2) {
			fmt.Printf("  [img] %s = %s\n", ck.Name, ck.Value)
		}
		// Also check .cityheaven.net (domain cookie)
		u3, _ := url.Parse("https://cityheaven.net")
		for _, ck := range jar.Cookies(u3) {
			fmt.Printf("  [domain] %s = %s\n", ck.Name, ck.Value)
		}
		fmt.Println()
	}

	doGet := func(label, urlStr string, extraHeaders map[string]string) string {
		fmt.Printf("========================================\n")
		fmt.Printf("=== %s: GET %s ===\n", label, urlStr)
		req, _ := http.NewRequest("GET", urlStr, nil)
		req.Header.Set("User-Agent", ua)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
		req.Header.Set("Accept-Language", "ja,en-US;q=0.7,en;q=0.3")
		req.Header.Set("Accept-Encoding", "identity") // No compression for readability
		for k, v := range extraHeaders {
			req.Header.Set(k, v)
		}

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("  ERROR: %v\n", err)
			return ""
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)

		fmt.Printf("  Status: %d\n", resp.StatusCode)
		fmt.Printf("  Final URL: %s\n", resp.Request.URL.String())
		fmt.Printf("  Body length: %d\n", len(body))

		// Check for key content
		if strings.Contains(bodyStr, "nenrei=y") {
			fmt.Println("  >>> CONTAINS: age gate (nenrei=y button)")
		}
		if strings.Contains(bodyStr, "風俗で遊ぶ") {
			fmt.Println("  >>> CONTAINS: age gate ENTER button text")
		}
		if strings.Contains(bodyStr, "ログイン") {
			fmt.Println("  >>> CONTAINS: login link/form")
		}
		if strings.Contains(bodyStr, "マイページ") {
			fmt.Println("  >>> CONTAINS: My Page (logged in)")
		}
		if strings.Contains(bodyStr, "login_id") || strings.Contains(bodyStr, "name=\"user\"") || strings.Contains(bodyStr, "name=\"pass\"") {
			fmt.Println("  >>> CONTAINS: login form fields")
		}
		if strings.Contains(bodyStr, "<form") {
			fmt.Println("  >>> CONTAINS: <form> element")
			// Extract form action
			idx := strings.Index(bodyStr, "<form")
			end := strings.Index(bodyStr[idx:], ">")
			if end > 0 {
				fmt.Printf("  >>> Form tag: %s\n", bodyStr[idx:idx+end+1])
			}
		}

		dumpCookies(label)
		return bodyStr
	}

	// ============================================================
	// Step 1: Hit the root page first to see if we get an age gate
	// ============================================================
	body1 := doGet("Step 1 - Root Page", "https://www.cityheaven.net/", nil)
	os.WriteFile("debug_flow_step1.html", []byte(body1), 0644)

	// ============================================================
	// Step 2: Hit the age verify bypass URL (following redirects)
	// Use the exact URL from the age gate ENTER button
	// ============================================================
	body2 := doGet("Step 2 - Age Bypass", "https://www.cityheaven.net/niigata/?nenrei=y", map[string]string{
		"Referer": "https://www.cityheaven.net/",
	})
	os.WriteFile("debug_flow_step2.html", []byte(body2), 0644)

	// ============================================================
	// Step 3: Now try to access the login page
	// ============================================================
	body3 := doGet("Step 3 - Login Page", "https://www.cityheaven.net/niigata/login/", map[string]string{
		"Referer": "https://www.cityheaven.net/niigata/",
	})
	os.WriteFile("debug_flow_step3.html", []byte(body3), 0644)

	// ============================================================
	// Step 4: Also try global login page
	// ============================================================
	body4 := doGet("Step 4 - Global Login", "https://www.cityheaven.net/login/", map[string]string{
		"Referer": "https://www.cityheaven.net/niigata/",
	})
	os.WriteFile("debug_flow_step4.html", []byte(body4), 0644)

	// ============================================================
	// Step 5: Try manually setting nenrei cookie and re-fetching login
	// The server might check for a specific cookie name
	// ============================================================
	fmt.Println("\n========================================")
	fmt.Println("=== Step 5 - Setting manual nenrei cookie and retrying ===")
	// Set a nenrei cookie manually
	u, _ := url.Parse("https://www.cityheaven.net")
	jar.SetCookies(u, []*http.Cookie{
		{Name: "nenrei", Value: "y", Path: "/"},
		{Name: "age_check", Value: "1", Path: "/"},
		{Name: "over18", Value: "1", Path: "/"},
	})
	dumpCookies("Step 5 - After manual cookie set")

	body5 := doGet("Step 5 - Login with manual cookies", "https://www.cityheaven.net/niigata/login/", map[string]string{
		"Referer": "https://www.cityheaven.net/niigata/",
	})
	os.WriteFile("debug_flow_step5.html", []byte(body5), 0644)

	// ============================================================
	// Step 6: Inspect what img.cityheaven.net/cs/nenrei/ looks like
	// This is the redirect target when age check fails
	// ============================================================
	body6 := doGet("Step 6 - nenrei redirect target", "http://img.cityheaven.net/cs/nenrei/", nil)
	os.WriteFile("debug_flow_step6.html", []byte(body6), 0644)

	// ============================================================
	// Step 7: Try using a no-redirect client to see the exact response from
	// the bypass URL - see if there's a redirect chain we're missing
	// ============================================================
	fmt.Println("\n========================================")
	fmt.Println("=== Step 7 - Bypass URL with no-redirect client ===")
	jar2, _ := cookiejar.New(nil)
	noRedirectClient := &http.Client{
		Jar: jar2,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req7, _ := http.NewRequest("GET", "https://www.cityheaven.net/niigata/?nenrei=y", nil)
	req7.Header.Set("User-Agent", ua)
	req7.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req7.Header.Set("Accept-Language", "ja,en-US;q=0.7,en;q=0.3")
	req7.Header.Set("Referer", "https://www.cityheaven.net/")

	resp7, err := noRedirectClient.Do(req7)
	if err != nil {
		fmt.Printf("  ERROR: %v\n", err)
	} else {
		fmt.Printf("  Status: %d\n", resp7.StatusCode)
		fmt.Printf("  Location: %s\n", resp7.Header.Get("Location"))
		fmt.Println("  Set-Cookie headers:")
		for _, sc := range resp7.Header["Set-Cookie"] {
			fmt.Printf("    %s\n", sc)
		}
		fmt.Println("  All Response Headers:")
		for k, vv := range resp7.Header {
			for _, v := range vv {
				fmt.Printf("    %s: %s\n", k, v)
			}
		}
		body7, _ := io.ReadAll(resp7.Body)
		resp7.Body.Close()
		fmt.Printf("  Body length: %d\n", len(body7))
		// Check first 500 chars
		if len(body7) > 500 {
			fmt.Printf("  Body start: %s...\n", string(body7[:500]))
		} else {
			fmt.Printf("  Body: %s\n", string(body7))
		}
	}

	fmt.Println("\n=== DONE ===")
}
