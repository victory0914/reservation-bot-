package client

import (
	"fmt"
	"time"
)

// Scheduler handles precise timing for request execution.
type Scheduler struct {
	// SpinDuration is the duration before target time to switch from sleeping to busy-waiting.
	// Default: 5ms
	SpinDuration time.Duration
}

// NewScheduler creates a new Scheduler.
func NewScheduler() *Scheduler {
	return &Scheduler{
		SpinDuration: 5 * time.Millisecond,
	}
}

// SleepUntil blocks until the target time is reached.
// It uses time.Sleep for the bulk of the wait, then busy-waits (spin lock)
// for the final milliseconds to ensure high precision (eliminating OS scheduler jitter).
// Returns the drift (actual wake time - target time).
func (s *Scheduler) SleepUntil(target time.Time) time.Duration {
	now := time.Now()

	// If already past target, return immediately with negative drift
	if now.After(target) {
		return now.Sub(target)
	}

	// Calculate remaining time
	remaining := target.Sub(now)

	// 1. Coarse Sleep
	// Sleep until SpinDuration before target
	if remaining > s.SpinDuration {
		sleepParams := remaining - s.SpinDuration
		time.Sleep(sleepParams)
	}

	// 2. Spin Lock (Busy Wait)
	// Continuously poll time.Now() until target is reached
	// This burns CPU for ~5ms but guarantees we catch the exact moment
	for {
		now = time.Now()
		if !now.Before(target) {
			break
		}
	}

	return now.Sub(target)
}

// LogDrift prints the drift in a readable format
func (s *Scheduler) LogDrift(drift time.Duration) {
	msg := fmt.Sprintf("⏱️  Precision Wake: Drift = %d µs", drift.Microseconds())

	// Colorize
	if drift > 1*time.Millisecond {
		// Red if > 1ms off
		fmt.Printf("\033[31m%s\033[0m\n", msg)
	} else {
		// Green if < 1ms (Ideal)
		fmt.Printf("\033[32m%s\033[0m\n", msg)
	}
}
