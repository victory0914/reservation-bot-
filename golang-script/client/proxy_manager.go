package client

import (
	"bufio"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"sync"
	"time"
)

// ProxyManager handles loading and rotating proxies.
type ProxyManager struct {
	proxies      []string
	currentIndex int
	mu           sync.Mutex
	random       *rand.Rand

	// Smartproxy Config
	useSmartproxy bool
	smartUser     string
	smartPass     string
	smartEndpoint string

	// Sticky session: reuses the same proxy URL until explicitly rotated
	stickyURL string
}

// NewProxyManager creates a new ProxyManager.
func NewProxyManager() *ProxyManager {
	return &ProxyManager{
		proxies: []string{},
		random:  rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// LoadProxies loads proxies from a file (one per line).
// Format: scheme://ip:port or scheme://user:pass@ip:port
func (pm *ProxyManager) LoadProxies(path string) error {
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
			// Basic validation/parsing check
			if _, err := url.Parse(line); err == nil {
				loaded = append(loaded, line)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.proxies = loaded
	// Shuffle efficiently
	pm.random.Shuffle(len(pm.proxies), func(i, j int) {
		pm.proxies[i], pm.proxies[j] = pm.proxies[j], pm.proxies[i]
	})

	fmt.Printf("Loaded %d proxies from %s\n", len(pm.proxies), path)
	return nil
}

// GetNext returns the next proxy.
// If Smartproxy is enabled, returns a generated rotating URL.
// Else returns from the loaded list (round-robin).
func (pm *ProxyManager) GetNext() string {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.useSmartproxy {
		// Generate random session ID for Smartproxy rotation
		sessionID := pm.random.Intn(9000000) + 1000000 // 7 digit random
		// Format: http://username-session-ID:pass@endpoint
		return fmt.Sprintf("http://%s-session-%d:%s@%s",
			pm.smartUser, sessionID, pm.smartPass, pm.smartEndpoint)
	}

	if len(pm.proxies) == 0 {
		return ""
	}

	proxy := pm.proxies[pm.currentIndex]
	pm.currentIndex = (pm.currentIndex + 1) % len(pm.proxies)
	return proxy
}

// GetRandom returns a random proxy from the list.
func (pm *ProxyManager) GetRandom() string {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if len(pm.proxies) == 0 {
		return ""
	}

	return pm.proxies[pm.random.Intn(len(pm.proxies))]
}

// MarkBad can be used to temporarily remove or deprioritize a proxy (optional implementation).
func (pm *ProxyManager) MarkBad(proxy string) {
	// For now, we just log it. Advanced logic could remove it from the rotation.
	fmt.Printf("Marking proxy as bad: %s\n", proxy)
}

// EnableSmartproxy configures the manager to use Smartproxy with rotating sessions.
// endpoint should be like "gate.smartproxy.com:7000"
func (pm *ProxyManager) EnableSmartproxy(user, pass, endpoint string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.smartUser = user
	pm.smartPass = pass
	pm.smartEndpoint = endpoint
	pm.useSmartproxy = true

	fmt.Printf("Smartproxy enabled on endpoint: %s\n", endpoint)
}

// UseSmartproxy switches the manager to use Smartproxy mode.
func (pm *ProxyManager) UseSmartproxy() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if pm.smartUser != "" && pm.smartPass != "" {
		pm.useSmartproxy = true
	}
}

// UseFileProxies switches the manager to use file-loaded proxies.
func (pm *ProxyManager) UseFileProxies() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.useSmartproxy = false
}

// HasFileProxies returns true if file-loaded proxies are available.
func (pm *ProxyManager) HasFileProxies() bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return len(pm.proxies) > 0
}

// IsSmartproxyActive returns true if Smartproxy mode is active.
func (pm *ProxyManager) IsSmartproxyActive() bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.useSmartproxy
}

// GetCurrentProxyInfo returns a human-readable label for the current proxy mode and the proxy URL (masked).
func (pm *ProxyManager) GetCurrentProxyInfo() string {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.useSmartproxy {
		return fmt.Sprintf("SmartProxy (%s, JP-Tokyo rotating)", pm.smartEndpoint)
	}

	if len(pm.proxies) > 0 {
		proxy := pm.proxies[pm.currentIndex]
		if u, err := url.Parse(proxy); err == nil {
			return fmt.Sprintf("File Proxy (%s)", u.Host)
		}
		return fmt.Sprintf("File Proxy (%s)", proxy)
	}

	return "DIRECT (no proxy)"
}

// GetSticky returns the same proxy URL until RotateSticky() is called.
// This ensures redirects and multi-step flows use the same IP.
func (pm *ProxyManager) GetSticky() string {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.stickyURL != "" {
		return pm.stickyURL
	}

	// Generate a new sticky URL based on current mode
	if pm.useSmartproxy {
		sessionID := pm.random.Intn(9000000) + 1000000
		pm.stickyURL = fmt.Sprintf("http://%s-session-%d:%s@%s",
			pm.smartUser, sessionID, pm.smartPass, pm.smartEndpoint)
	} else if len(pm.proxies) > 0 {
		pm.stickyURL = pm.proxies[pm.currentIndex]
		pm.currentIndex = (pm.currentIndex + 1) % len(pm.proxies)
	}

	return pm.stickyURL
}

// RotateSticky clears the sticky proxy so the next GetSticky() call picks a new one.
func (pm *ProxyManager) RotateSticky() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.stickyURL = ""
}
