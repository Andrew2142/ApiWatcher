package gui

import (
	"fmt"
	"log"
	"net"
	"time"
	"url-checker/internal/daemon"
)

// checkLocalDaemonRunning checks if the daemon is running on localhost:9876
func (s *AppState) checkLocalDaemonRunning() bool {
	conn, err := net.DialTimeout("tcp", "localhost:9876", 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// connectToLocalDaemon attempts to connect to a locally running daemon
func (s *AppState) connectToLocalDaemon() error {
	log.Println("Checking for local daemon on localhost:9876...")
	
	// Check if daemon is running
	if !s.checkLocalDaemonRunning() {
		return fmt.Errorf("daemon not running on localhost:9876. Please start the daemon first using: ./apiwatcher-daemon")
	}
	
	log.Println("Local daemon found, connecting...")
	s.isLocalMode = true
	
	// Create daemon client pointing to localhost
	s.daemonClient = daemon.NewClient("localhost:9876")
	
	// Connect to daemon
	if err := s.daemonClient.Connect(); err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	
	log.Println("Successfully connected to local daemon")
	return nil
}

