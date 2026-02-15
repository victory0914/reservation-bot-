package main

import (
	"booker-bot/client"
	"fmt"
	"os"
)

func main() {
	// Credentials from context
	username := "amritacharya" // Replace with real one if needed, but using test creds
	password := "12345678"

	c := client.NewLowLatencyClient(func() {}, 0, "")

	fmt.Println("Attempting Login with updated logic...")
	err := c.Login(username, password)
	if err != nil {
		fmt.Printf("Login Failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Login Success!")
}
