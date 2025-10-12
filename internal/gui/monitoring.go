package gui

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

// Step 9: Start monitoring - sends config and start command to remote daemon
func (s *AppState) startMonitoring() {
	// Show progress dialog
	progress := dialog.NewProgressInfinite("Starting Monitoring", "Sending configuration to server...", s.window)
	progress.Show()

	go func() {
		// Build snapshot IDs map
		snapshotIDs := make(map[string]string)
		for url, snap := range s.snapshotsByURL {
			if snap != nil {
				snapshotIDs[url] = snap.ID
			}
		}

		// Send configuration to daemon
		log.Println("Sending configuration to daemon...")
		err := s.daemonClient.SetConfig(s.cfg.Email, s.cfg.Websites, snapshotIDs)
		if err != nil {
			fyne.Do(func() {
				progress.Hide()
				dialog.ShowError(fmt.Errorf("failed to send configuration: %v", err), s.window)
			})
			return
		}

		// Start monitoring on daemon
		log.Println("Starting monitoring on remote daemon...")
		err = s.daemonClient.Start()
		if err != nil {
			fyne.Do(func() {
				progress.Hide()
				dialog.ShowError(fmt.Errorf("failed to start monitoring: %v", err), s.window)
			})
			return
		}

		fyne.Do(func() {
			progress.Hide()
		})

		log.Printf("✅ Monitoring started on server for %d websites", len(s.cfg.Websites))

		// Show the dashboard (which displays remote monitoring status)
		fyne.Do(func() {
			s.showDashboardScreen()
		})
	}()
}

