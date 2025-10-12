package main

import (
	"fmt"
	"log"
	"time"
	"url-checker/internal/daemon"
)

// Simple test program to verify daemon works
func main() {
	address := "localhost:9876"
	
	log.Println("Connecting to daemon at", address)
	client := daemon.NewClient(address)
	
	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()
	
	log.Println("✅ Connected to daemon")
	
	// Test ping
	if err := client.Ping(); err != nil {
		log.Fatalf("Ping failed: %v", err)
	}
	log.Println("✅ Ping successful")
	
	// Get initial status
	status, err := client.GetStatus()
	if err != nil {
		log.Fatalf("Failed to get status: %v", err)
	}
	log.Printf("✅ Status: %s, HasConfig: %v", status.State, status.HasConfig)
	
	// Set configuration
	websites := []string{"https://www.google.com", "https://www.github.com"}
	log.Println("Setting configuration...")
	if err := client.SetConfig("test@example.com", websites, nil); err != nil {
		log.Fatalf("Failed to set config: %v", err)
	}
	log.Println("✅ Configuration set")
	
	// Start monitoring
	log.Println("Starting monitoring...")
	if err := client.Start(); err != nil {
		log.Fatalf("Failed to start: %v", err)
	}
	log.Println("✅ Monitoring started")
	
	// Wait a bit
	time.Sleep(2 * time.Second)
	
	// Get status again
	status, err = client.GetStatus()
	if err != nil {
		log.Fatalf("Failed to get status: %v", err)
	}
	log.Printf("✅ Status: %s, Monitoring %d websites", status.State, status.WebsiteCount)
	
	// Get logs
	logs, err := client.GetLogs(10)
	if err != nil {
		log.Fatalf("Failed to get logs: %v", err)
	}
	fmt.Println("\n📋 Recent logs:")
	for _, line := range logs {
		fmt.Println("  ", line)
	}
	
	// Stop monitoring
	log.Println("\nStopping monitoring...")
	if err := client.Stop(); err != nil {
		log.Fatalf("Failed to stop: %v", err)
	}
	log.Println("✅ Monitoring stopped")
	
	// Final status
	status, err = client.GetStatus()
	if err != nil {
		log.Fatalf("Failed to get status: %v", err)
	}
	log.Printf("✅ Final status: %s", status.State)
	
	fmt.Println("\n🎉 All tests passed!")
}

