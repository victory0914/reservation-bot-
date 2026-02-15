package main

import (
	"booker-bot/client"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func main() {
	c := client.NewLowLatencyClient(func() {}, 0, "")

	dumpCookies := func(label string) {
		u, _ := url.Parse("https://www.cityheaven.net")
		fmt.Printf("\n--- Cookies after %s ---\n", label)
		for _, ck := range c.CookieJar().Cookies(u) {
			fmt.Printf("  %s = %s\n", ck.Name, ck.Value)
		}
		fmt.Println()
	}

	doGet := func(label, urlStr string, extraHeaders map[string]string) string {
		fmt.Printf("========================================\n")
		fmt.Printf("=== %s: GET %s ===\n", label, urlStr)
		req, _ := http.NewRequest("GET", urlStr, nil)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
		req.Header.Set("Accept-Language", "ja,en-US;q=0.7,en;q=0.3")
		for k, v := range extraHeaders {
			req.Header.Set(k, v)
		}

		resp, err := c.Do(req)
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

		if strings.Contains(bodyStr, "nenrei=y") {
			fmt.Println("  >>> CONTAINS: age gate (nenrei=y button)")
		}
		if strings.Contains(bodyStr, "ログイン") {
			fmt.Println("  >>> CONTAINS: login link/text")
		}
		if strings.Contains(bodyStr, "login_form") {
			fmt.Println("  >>> CONTAINS: login_form (form name)")
		}
		if strings.Contains(bodyStr, "loginAuth") {
			fmt.Println("  >>> CONTAINS: loginAuth (form action)")
		}
		if strings.Contains(bodyStr, "<title>ログイン</title>") {
			fmt.Println("  >>> LOGIN PAGE CONFIRMED by <title>")
		}
		if strings.Contains(bodyStr, "マイページ") {
			fmt.Println("  >>> CONTAINS: My Page (logged in)")
		}

		dumpCookies(label)
		return bodyStr
	}

	// Step 1: Age Bypass
	body1 := doGet("Step 1 - Age Bypass", "https://www.cityheaven.net/niigata/?nenrei=y", map[string]string{
		"Referer": "https://www.cityheaven.net/",
	})
	os.WriteFile("debug_utls_step1.html", []byte(body1), 0644)

	// Step 2: Login Page
	body2 := doGet("Step 2 - Login Page", "https://www.cityheaven.net/niigata/login/", map[string]string{
		"Referer": "https://www.cityheaven.net/niigata/",
	})
	os.WriteFile("debug_utls_step2.html", []byte(body2), 0644)

	// Step 3: Actually POST the login form with correct fields
	if strings.Contains(body2, "login_form") || strings.Contains(body2, "loginAuth") {
		fmt.Println("\n=== Step 3: POST login form ===")
		mitapage := base64.StdEncoding.EncodeToString([]byte("https://www.cityheaven.net/niigata/"))

		data := url.Values{}
		data.Set("user", "amritacharya")
		data.Set("pass", "12345678")
		data.Set("login", "ログイン")
		data.Set("adprefflg", "0")
		data.Set("forwardTo", "")
		data.Set("mitagirl", "")
		data.Set("mitapage", mitapage)
		data.Set("message_flg", "0")
		data.Set("message_girl_id", "")
		data.Set("dummy", "")
		data.Set("myheavenflg", "0")
		data.Set("touhyouFlg", "0")
		data.Set("touhyouId", "")
		data.Set("touhyouNo", "")
		data.Set("touhyouDate", "")
		data.Set("pointcardurl", "")
		data.Set("targetPageUrl", "")
		data.Set("originalPageUrl", "/niigata/")
		data.Set("voidFlg", "")
		data.Set("favorite_url", "")
		data.Set("favorite_refer_url", "")
		data.Set("official_no_disp", "")

		reqPost, _ := http.NewRequest("POST", "https://www.cityheaven.net/niigata/login/loginAuth/", strings.NewReader(data.Encode()))
		reqPost.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		reqPost.Header.Set("Referer", "https://www.cityheaven.net/niigata/login/")
		reqPost.Header.Set("Origin", "https://www.cityheaven.net")

		resp, err := c.Do(reqPost)
		if err != nil {
			fmt.Printf("  POST ERROR: %v\n", err)
		} else {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			bodyStr := string(body)

			fmt.Printf("  Status: %d\n", resp.StatusCode)
			fmt.Printf("  Final URL: %s\n", resp.Request.URL.String())
			fmt.Printf("  Body length: %d\n", len(body))

			if strings.Contains(bodyStr, "nenrei=y") {
				fmt.Println("  >>> STILL ON AGE GATE")
			}
			if strings.Contains(bodyStr, "マイページ") {
				fmt.Println("  >>> SUCCESS: My Page found!")
			}
			if strings.Contains(bodyStr, "IDまたはパスワードが違います") {
				fmt.Println("  >>> FAILURE: Invalid credentials")
			}
			if strings.Contains(bodyStr, "ログイン") && !strings.Contains(bodyStr, "ログアウト") {
				fmt.Println("  >>> Login text found (may still be on login page)")
			}
			if strings.Contains(bodyStr, "ログアウト") {
				fmt.Println("  >>> SUCCESS: Logout link found!")
			}

			os.WriteFile("debug_utls_step3.html", body, 0644)
			dumpCookies("Step 3 - After Login POST")
		}
	} else {
		fmt.Println("\n=== Step 3: SKIPPED (login form not found in Step 2) ===")
		fmt.Println("The uTLS client did not get past the age gate.")
	}

	fmt.Println("\n=== DONE ===")
}
