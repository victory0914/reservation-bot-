package client

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// SafetyManager handles emergency stops and health checks.
type SafetyManager struct {
	mu            sync.RWMutex
	Triggered     bool
	TriggerReason string
	TriggeredAt   time.Time

	// Thresholds
	MaxConsecutiveErrors int
	ErrorCount           int
}

// NewSafetyManager creates a new SafetyManager.
func NewSafetyManager() *SafetyManager {
	return &SafetyManager{
		MaxConsecutiveErrors: 5,
	}
}

// CheckResponse inspects a response for ban signals (403, 429).
// Returns true if safe to proceed, false if safety trigger pulled.
func (sm *SafetyManager) CheckResponse(resp *http.Response) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.Triggered {
		return false
	}

	if resp.StatusCode == 403 || resp.StatusCode == 429 {
		sm.triggerLocked(fmt.Sprintf("HTTP %d Detected", resp.StatusCode))
		return false
	}

	if resp.StatusCode >= 500 {
		sm.ErrorCount++
		if sm.ErrorCount >= sm.MaxConsecutiveErrors {
			sm.triggerLocked("Too many consecutive 5xx errors")
			return false
		}
	} else if resp.StatusCode == 200 {
		sm.ErrorCount = 0 // Reset on success
	}

	return true
}

func (sm *SafetyManager) CheckError(err error) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// TODO: Analyze error text for network vs logic issues
	sm.ErrorCount++
	return !sm.Triggered
}

func (sm *SafetyManager) triggerLocked(reason string) {
	if !sm.Triggered {
		sm.Triggered = true
		sm.TriggerReason = reason
		sm.TriggeredAt = time.Now()
		fmt.Printf("\n🚨 SAFETY TRIGGER ACTIVATED: %s 🚨\n", reason)
	}
}

// IsTriggered checks status without locking if possible or just RLock
func (sm *SafetyManager) IsTriggered() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.Triggered
}
