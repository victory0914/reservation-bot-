package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// CaptchaSolver defines the interface for CAPTCHA solving services.
type CaptchaSolver interface {
	Solve(siteKey, url string) (string, error)
}

// MockCaptchaSolver is a dummy solver for testing.
type MockCaptchaSolver struct{}

func (s *MockCaptchaSolver) Solve(siteKey, url string) (string, error) {
	fmt.Printf("[CAPTCHA] Simulated solving for %s on %s...\n", siteKey, url)
	time.Sleep(2 * time.Second) // Simulate delay
	return "MOCK_CAPTCHA_SOLUTION", nil
}

// TwoCaptchaSolver is a placeholder for 2Captcha implementation.
type TwoCaptchaSolver struct {
	APIKey string
}

func (s *TwoCaptchaSolver) Solve(siteKey, url string) (string, error) {
	// 1. Send CAPTCHA request
	// Assuming reCAPTCHA v2 for now based on typical targets, but could be others.
	// For now, we'll implement a generic method or assume similar parameters.
	// The user mentions "siteKey" and "url", which are typical for reCAPTCHA/hCaptcha.

	// Construct URL for in.php
	// http://2captcha.com/in.php?key=API_KEY&method=userrecaptcha&googlekey=SITE_KEY&pageurl=PAGE_URL&json=1

	u := fmt.Sprintf("http://2captcha.com/in.php?key=%s&method=userrecaptcha&googlekey=%s&pageurl=%s&json=1", s.APIKey, siteKey, url)

	resp, err := http.Get(u)
	if err != nil {
		return "", fmt.Errorf("2Captcha request failed: %w", err)
	}
	defer resp.Body.Close()

	var inResponse struct {
		Status  int    `json:"status"`
		Request string `json:"request"` // Contains ID or Error
	}

	if err := json.NewDecoder(resp.Body).Decode(&inResponse); err != nil {
		return "", fmt.Errorf("failed to decode 2Captcha response: %w", err)
	}

	if inResponse.Status != 1 {
		return "", fmt.Errorf("2Captcha error: %s", inResponse.Request)
	}

	requestID := inResponse.Request
	fmt.Printf("[2Captcha] Job submitted. ID: %s. Waiting for solution...\n", requestID)

	// 2. Poll for result
	// http://2captcha.com/res.php?key=API_KEY&action=get&id=ID&json=1

	for i := 0; i < 20; i++ { // Try for ~100 seconds
		time.Sleep(5 * time.Second)

		pollURL := fmt.Sprintf("http://2captcha.com/res.php?key=%s&action=get&id=%s&json=1", s.APIKey, requestID)
		pollResp, err := http.Get(pollURL)
		if err != nil {
			continue // Network error, retry
		}
		defer pollResp.Body.Close() // defer inside loop is warning, but minimal leak for 20 iters

		var pollResponse struct {
			Status  int    `json:"status"`
			Request string `json:"request"`
		}

		// Reset body reader
		if err := json.NewDecoder(pollResp.Body).Decode(&pollResponse); err != nil {
			continue
		}

		if pollResponse.Status == 1 {
			return pollResponse.Request, nil
		}

		if pollResponse.Request != "CAPCHA_NOT_READY" {
			return "", fmt.Errorf("2Captcha polling error: %s", pollResponse.Request)
		}

		fmt.Print(".")
	}

	return "", fmt.Errorf("2Captcha timeout")
}
