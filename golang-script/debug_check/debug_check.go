package main

import (
	"booker-bot/client"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
)

const (
	Username        = "amritacharya"
	Password        = "12345678"
	TargetGirlID    = "18037583"
	TargetShopID    = "2310001233"
	TargetCourseID  = "253139"
	AreaPath        = "niigata/A1501/A150101"
	ShopDir         = "arabiannight"
	S6URLFormat     = "https://www.cityheaven.net/niigata/A1501/A150101/arabiannight/S6ShareToReservationLogin/?forward=F1&girl_id=%s&pcmode=sp"
	CourseSelectURL = "https://yoyaku.cityheaven.net/select_course/niigata/A1501/A150101/arabiannight"
	ProfileInputURL = "https://yoyaku.cityheaven.net/input_profile/niigata/A1501/A150101/arabiannight"
)

func main() {
	fmt.Println("Starting Debug Check...")

	// 1. Verify Login & CSRF
	fmt.Println("Step 1: Login")
	_, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := client.NewLowLatencyClient(cancel, 0, nil, nil, nil, false)

	if err := client.Login(Username, Password); err != nil {
		fmt.Printf("Login failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Login successful.")

	// Check cookies after login
	client.DebugCookies("https://www.cityheaven.net")
	client.DebugCookies("https://yoyaku.cityheaven.net")

	// 2. Fetch Calendar for specific girl
	targetURL := fmt.Sprintf(S6URLFormat, TargetGirlID)
	fmt.Printf("Step 2: Fetching Calendar for Girl %s from %s\n", TargetGirlID, targetURL)

	slots, err := client.FetchCalendar(targetURL)
	if err != nil {
		fmt.Printf("FetchCalendar returned error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("FetchCalendar returned %d slots.\n", len(slots))
	for _, s := range slots {
		fmt.Printf(" - Slot: %s %s\n", s.Date, s.DayTime)
	}

	if len(slots) == 0 {
		fmt.Println("No slots found, cannot proceed with sequence test.")
		return
	}
	targetSlot := slots[0]

	// 3. Run Sequence
	fmt.Println("Step 3: Simulate Reservation Sequence")

	// A. Select Slot
	// Format time "10:00" -> "1000"
	// Ensure length is sufficient
	if len(targetSlot.DayTime) < 5 {
		fmt.Printf("Invalid time format: %s\n", targetSlot.DayTime)
		os.Exit(1)
	}
	rawTime := targetSlot.DayTime[0:2] + targetSlot.DayTime[3:5]
	fmt.Printf("A. Selecting Slot: %s %s (API: %s)\n", targetSlot.Date, targetSlot.DayTime, rawTime)

	// Check cookies before SelectSlot
	client.DebugCookies("https://yoyaku.cityheaven.net")

	if err := client.SelectSlot(AreaPath, ShopDir, TargetGirlID, targetSlot.Date, rawTime); err != nil {
		fmt.Printf("SelectSlot failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Slot selected.")

	// B. Select Course (Checks CSRF)
	fmt.Println("B. Selecting Course...")
	client.DebugCookies("https://yoyaku.cityheaven.net")

	if err := client.SelectCourse(CourseSelectURL, TargetCourseID); err != nil {
		fmt.Printf("SelectCourse failed: %v\n", err)
		// Dump HTML if CSRF error
		fmt.Println("Dumping Course Page...")
		dumpPage(client, CourseSelectURL, "debug_course_error.html")
		os.Exit(1)
	}
	fmt.Println("Course selected.")

	// C. Submit Profile (Checks CSRF)
	fmt.Println("C. Checking Profile Page (CSRF)...")
	token, err := client.GetCSRFToken(ProfileInputURL)
	if err != nil {
		fmt.Printf("Profile Page CSRF failed: %v\n", err)
		dumpPage(client, ProfileInputURL, "debug_profile_error.html")
		os.Exit(1)
	}
	fmt.Printf("Profile Page CSRF Token found: %s\n", token)
	fmt.Println("SUCCESS: Full sequence CSRF checks passed!")

	// 4. Check Reservation History
	fmt.Println("Step 4: Check Reservation History")
	reservations, err := client.CheckReservations()
	if err != nil {
		fmt.Printf("CheckReservations failed: %v\n", err)
	} else if len(reservations) == 0 {
		fmt.Println("No reservations found.")
	} else {
		fmt.Printf("Found %d reservations:\n", len(reservations))
		for _, r := range reservations {
			fmt.Printf(" - %s at %s (%s) - %s\n", r.GirlName, r.ShopName, r.Date, r.Status)
		}
	}
}

func dumpPage(c *client.LowLatencyClient, urlStr, filename string) {
	req, _ := http.NewRequest("GET", urlStr, nil)
	resp, err := c.Do(req)
	if err == nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		os.WriteFile(filename, body, 0644)
		fmt.Printf("Saved page content to %s\n", filename)
	}
}
