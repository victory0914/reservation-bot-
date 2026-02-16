package client

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"
)

// FingerprintManager handles User-Agent and header randomization.
type FingerprintManager struct {
	userAgents []string
	mu         sync.Mutex
	random     *rand.Rand
}

// NewFingerprintManager creates a new FingerprintManager.
func NewFingerprintManager() *FingerprintManager {
	return &FingerprintManager{
		userAgents: []string{
			// Default fallback
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		},
		random: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// LoadUserAgents loads user agents from a file (one per line).
func (fm *FingerprintManager) LoadUserAgents(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	var loaded []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			loaded = append(loaded, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	if len(loaded) > 0 {
		fm.mu.Lock()
		fm.userAgents = loaded
		fm.mu.Unlock()
		fmt.Printf("Loaded %d user agents from %s\n", len(loaded), path)
	}

	return nil
}

// GetRandomUserAgent returns a random User-Agent string.
func (fm *FingerprintManager) GetRandomUserAgent() string {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	return fm.userAgents[fm.random.Intn(len(fm.userAgents))]
}

// GetRandomHeaders returns a map of common browser headers with randomized values.
func (fm *FingerprintManager) GetRandomHeaders() map[string]string {
	headers := make(map[string]string)

	// Randomize Accept-Language
	languages := []string{"en-US,en;q=0.9", "en-GB,en;q=0.9", "ja-JP,ja;q=0.9,en-US;q=0.8,en;q=0.7"}
	headers["Accept-Language"] = languages[fm.random.Intn(len(languages))]

	// Randomize Accept
	accepts := []string{
		"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
		"text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
	}
	headers["Accept"] = accepts[fm.random.Intn(len(accepts))]

	// Add other common headers
	headers["Connection"] = "keep-alive"
	headers["Upgrade-Insecure-Requests"] = "1"
	headers["Sec-Fetch-Dest"] = "document"
	headers["Sec-Fetch-Mode"] = "navigate"
	headers["Sec-Fetch-Site"] = "none"
	headers["Sec-Fetch-User"] = "?1"

	return headers
}
