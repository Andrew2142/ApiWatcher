package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"url-checker/internal/daemon"
)

const (
	Version = "1.0.0"
)

func main() {
	// Command line flags
	dataDir := flag.String("data-dir", getDefaultDataDir(), "Data directory for daemon state and logs")
	port := flag.String("port", "9876", "Port to listen on (localhost only)")
	version := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *version {
		fmt.Printf("apiwatcher-daemon version %s\n", Version)
		os.Exit(0)
	}

	log.Printf("Starting apiwatcher daemon version %s", Version)
	log.Printf("Data directory: %s", *dataDir)

	// Create daemon
	d, err := daemon.New(*dataDir)
	if err != nil {
		log.Fatalf("Failed to create daemon: %v", err)
	}

	// Create and start server
	address := fmt.Sprintf("localhost:%s", *port)
	server := daemon.NewServer(d, address)

	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Printf("Daemon is running")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan

	log.Println("Shutting down...")

	// Stop monitoring if running
	if d.GetState() == daemon.StateRunning {
		log.Println("Stopping monitoring...")
		if err := d.Stop(); err != nil {
			log.Printf("Error stopping monitoring: %v", err)
		}
	}

	// Stop server
	server.Stop()

	log.Println("Daemon stopped")
}

func getDefaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".apiwatcher"
	}
	return filepath.Join(home, ".apiwatcher")
}
