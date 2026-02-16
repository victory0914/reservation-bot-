package main

import (
	"booker-bot/client"
	"context"
	"fmt"
	"log"
)

func main() {
	log.Println("Verifying Reservation History Helper...")

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Client
	client := client.NewLowLatencyClient(cancel, 0, nil, nil, nil)

	// 1. Login
	username := "amritacharya"
	password := "12345678" // Use your actual login

	fmt.Printf("Logging in as %s...\n", username)            // Changed log.Println to fmt.Printf
	if err := client.Login(username, password); err != nil { // Changed c.Login to client.Login
		log.Fatalf("Login failed: %v", err)
	}
	fmt.Println("Login successful!") // Added new line

	// 2. Check Reservations
	fmt.Println("Checking reservations...")         // Changed log.Println to fmt.Println
	reservations, err := client.CheckReservations() // Changed c.CheckReservations to client.CheckReservations
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
