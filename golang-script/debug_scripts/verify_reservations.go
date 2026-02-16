package main

import (
	"booker-bot/client"
	"context"
	"log"
)

func main() {
	log.Println("Verifying Reservation History Helper...")

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Client
	c := client.NewLowLatencyClient(cancel, 0, "")

	// 1. Login
	username := "amritacharya"
	password := "12345678" // Use your actual login

	log.Println("Logging in...")
	if err := c.Login(username, password); err != nil {
		log.Fatalf("Login failed: %v", err)
	}

	// 2. Check Reservations
	log.Println("Fetching reservation history...")
	reservations, err := c.CheckReservations()
	if err != nil {
		log.Fatalf("Failed to check reservations: %v", err)
	}

	if len(reservations) == 0 {
		log.Println("Result: No reservations found.")
	} else {
		log.Printf("Result: Found %d reservations:", len(reservations))
		for _, res := range reservations {
			log.Printf("  - [%s] %s at %s (%s) - Status: %s", res.Date, res.GirlName, res.ShopName, res.Time, res.Status)
		}
	}
}
