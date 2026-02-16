package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"booker-bot/client"
)

// Configuration constants
const (
	TargetShopID   = "2310001233" // Taken from JSON analysis
	TargetGirlID   = "52809022"   // Example girl ID (Optional priority)
	TargetCourseID = "253139"     // Example course ID

	// Area/shop path components (used by SelectSlot and SelectGirl)
	AreaPath = "niigata/A1501/A150101"
	ShopDir  = "arabiannight"

	// Credentials
	Username = "amritacharya"
	Password = "12345678" // PLEASE CHANGE THIS OR LOAD FROM ENV

	// URLs
	BaseURL = "https://www.cityheaven.net/niigata/A1501/A150101/arabiannight/"

	// CalendarBaseFormat for S6 URL (User suggested, returns JSON)
	// Uses %[2]s to pick the girlID (2nd arg), ignoring the week number (1st arg)
	// The S6 URL returns 2 weeks of data, so we might only need to call it once per girl.
	CalendarBaseFormat = "https://www.cityheaven.net/niigata/A1501/A150101/arabiannight/S6ShareToReservationLogin/?forward=F1&girl_id=%[2]s&pcmode=sp"

	CourseSelectURL = "https://yoyaku.cityheaven.net/select_course/niigata/A1501/A150101/arabiannight"
	ProfileInputURL = "https://yoyaku.cityheaven.net/input_profile/niigata/A1501/A150101/arabiannight"
	ConfirmURL      = "https://yoyaku.cityheaven.net/Confirm/ConfirmList/niigata/A1501/A150101/arabiannight"

	PollInterval = 2000 * time.Millisecond // Slower poll for safety when iterating list
	DryRun       =  false                   // Set to false to actually book
)

func main() {
	log.Println("Starting City Heaven Low-Latency Client (Go)...")
	log.Println("Mode: Auto-Discovery & Polling (Verbose Slot Logging)")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Client
	c := client.NewLowLatencyClient(cancel, 0, "")

	// 1. Login & Age Verification
	log.Println("Step 1: Performing Age Verification & Login...")
	if err := c.HandleAgeVerification(); err != nil {
		log.Printf("Warning: Age verification check failed: %v (might already be verified)", err)
	}

	if err := c.Login(Username, Password); err != nil {
		log.Fatalf("Critical: Login failed: %v", err)
	}
	log.Println("Login successful.")

	// 1b. Check existing reservations
	log.Println("Checking existing reservations...")
	existing, err := c.CheckReservations()
	if err != nil {
		log.Printf("Warning: Could not check reservation history: %v", err)
	} else if len(existing) == 0 {
		log.Println("No active reservations found on My Page.")
	} else {
		log.Printf("Found %d active reservations:", len(existing))
		for _, res := range existing {
			log.Printf("  - [%s] %s at %s (%s) - Status: %s", res.Date, res.GirlName, res.ShopName, res.Time, res.Status)
		}
	}

	// 2. Polling Loop
	log.Println("Step 2: Starting Polling Loop with Auto-Discovery...")

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// A. Dynamic Girl Discovery
			log.Println("Fetching girl list from shop page...")
			girls, err := c.ListGirls(BaseURL)
			if err != nil {
				log.Printf("Error listing girls: %v", err)
				time.Sleep(PollInterval)
				continue
			}
			log.Printf("Found %d girls on page.", len(girls))

			// B. Iterate through each girl
			for i, girlID := range girls {
				log.Printf("[%d/%d] Checking GirlID: %s", i+1, len(girls), girlID)

				// Check availability (Max 1 set of weeks since S6 returns 2 weeks)
				weeksToCheck := 1
				foundSlots := false

				for week := 1; week <= weeksToCheck; week++ {
					targetURL := fmt.Sprintf(CalendarBaseFormat, week, girlID)

					log.Printf("  -> Checking Schedule...")

					slots, err := c.FetchCalendar(targetURL)
					if err != nil {
						log.Printf("Error fetching calendar for girl %s: %v", girlID, err)
						continue
					}

					if len(slots) > 0 {
						log.Printf("SUCCESS: Found %d available slots for GirlID %s!", len(slots), girlID)
						foundSlots = true

						targetSlot := slots[0]
						log.Printf("Targeting Slot: %s %s", targetSlot.Date, targetSlot.DayTime)

						RunReservationSequence(c, girlID, targetSlot)

						if !DryRun {
							// break
						}
					}

					time.Sleep(200 * time.Millisecond)
				}

				if !foundSlots {
					// log.Printf("  No slots found for GirlID %s.", girlID)
				}

				time.Sleep(500 * time.Millisecond)
			}

			log.Println("Finished one full pass of all girls. Sleeping...")
			time.Sleep(PollInterval)
		}
	}
}

func RunReservationSequence(c *client.LowLatencyClient, girlID string, slot client.Slot) {
	log.Println("Step 3: Starting Reservation Sequence...")

	// ── Step 1: Select Slot (POST /calendar/SelectedList/) ──
	// This locks the time slot in the server-side session.
	// The API expects day_time in HH:MM format (e.g. "10:00").
	log.Printf("  -> Step 3a: Selecting Slot: %s %s", slot.Date, slot.DayTime)

	if err := c.SelectSlot(AreaPath, ShopDir, girlID, slot.Date, slot.DayTime); err != nil {
		log.Printf("Failed to select slot: %v", err)
		return
	}
	log.Println("  -> Slot selected (SelectedList).")

	// ── Step 2: Select Girl (POST /Selectvacancygirl/SelectedGirl) ──
	// This confirms the girl selection after the slot has been locked.
	// Without this step, SelectCourse returns an error page (no CSRF token).
	log.Printf("  -> Step 3b: Selecting Girl: %s", girlID)

	if err := c.SelectGirl(TargetShopID, girlID, slot.Date, slot.DayTime); err != nil {
		log.Printf("Failed to select girl: %v", err)
		return
	}
	log.Println("  -> Girl selected (SelectedGirl).")

	// ── Step 3: Select Course ──
	// Now the session is correctly established, so the course page will
	// render with the _csrf token.
	log.Println("  -> Step 3c: Selecting Course...")
	if err := c.SelectCourse(CourseSelectURL, TargetCourseID); err != nil {
		log.Printf("Failed to select course: %v", err)
		return
	}
	log.Println("  -> Course selected.")

	// ── Step 4: Input Profile ──
	log.Println("  -> Step 3d: Submitting Profile...")
	config := client.ReservationConfig{
		ShopID:   TargetShopID,
		GirlID:   girlID,
		CourseID: TargetCourseID,
		AreaPath: AreaPath,
		ShopDir:  ShopDir,
		Name:     "Test User",
		Phone:    "09012345678",
		Email:    "test@example.com",
	}
	body, err := c.SubmitProfile(ProfileInputURL, config)
	if err != nil {
		log.Printf("Failed to submit profile: %v", err)
		return
	}
	log.Println("  -> Profile submitted.")

	// ── Step 5: Confirm ──
	log.Println("  -> Step 3e: Confirming Reservation...")
	if err := c.ConfirmReservation(ConfirmURL, body, DryRun); err != nil {
		log.Printf("Failed to confirm: %v", err)
		return
	}

	if DryRun {
		log.Println("SUCCESS: Helper sequence finished (Dry Run).")
	} else {
		log.Println("SUCCESS: Reservation Confirmed!")
	}
}
