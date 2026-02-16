package main

import (
	"booker-bot/client"
	"fmt"
	"time"
)

func main() {
	fmt.Println("Starting Precision Timing Verification...")

	scheduler := client.NewScheduler()

	// Test 3 times
	for i := 1; i <= 3; i++ {
		target := time.Now().Add(1 * time.Second)
		fmt.Printf("\n[Test %d] Sleeping until: %s\n", i, target.Format("15:04:05.000000"))

		// Block
		drift := scheduler.SleepUntil(target)

		fmt.Printf("   -> Woke up at: %s\n", time.Now().Format("15:04:05.000000"))
		scheduler.LogDrift(drift)

		if drift > 2*time.Millisecond {
			fmt.Println("   ⚠️  Warning: High drift detected! CPU might be overloaded or GC pause.")
		}
	}
}
