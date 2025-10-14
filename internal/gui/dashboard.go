package gui

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// showDashboardScreen shows the main monitoring dashboard
func (s *AppState) showDashboardScreen() {
	title := widget.NewLabel("Monitoring Dashboard")
	title.TextStyle.Bold = true

	// Get initial status
	status, err := s.daemonClient.GetStatus()
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to get status: %v", err), s.window)
		return
	}

	// Connection info
	connInfo := widget.NewLabel("Connected to: " + s.sshConn.Config().Host)
	connInfo.TextStyle.Italic = true

	// Status indicator
	statusIndicator := widget.NewLabel(fmt.Sprintf("● %s", strings.ToUpper(string(status.State))))
	statusIndicator.TextStyle.Bold = true

	// Monitoring info
	infoText := fmt.Sprintf(
		"Monitoring %d websites\nEmail alerts: %s\n\nTotal checks: %d\nLast check: %s",
		status.WebsiteCount,
		status.Email,
		status.Stats.TotalChecks,
		formatTime(status.Stats.LastCheckTime),
	)
	infoLabel := widget.NewLabel(infoText)

	// Log area
	logs, _ := s.daemonClient.GetLogs(50)
	logText := strings.Join(logs, "\n")
	logArea := widget.NewLabel(logText)
	logArea.Wrapping = fyne.TextWrapWord

	logScroll := container.NewScroll(logArea)
	logScroll.SetMinSize(fyne.NewSize(600, 200))

	stopBtn := widget.NewButton("Stop Monitoring", func() {
		// Stop is now instant - daemon kills workers immediately
		if err := s.daemonClient.Stop(); err != nil {
			dialog.ShowError(fmt.Errorf("failed to stop: %v", err), s.window)
			return
		}
		s.stopDashboardRefresh()
		s.showDaemonStoppedScreen()
	})

	clearLogsBtn := widget.NewButton("Clear Buffer Log", func() {
		if err := s.daemonClient.ClearLogs(); err != nil {
			dialog.ShowError(fmt.Errorf("failed to clear logs: %v", err), s.window)
			return
		}
		// Immediately refresh the log area to show it's cleared
		logArea.SetText("")
	})

	disconnectBtn := widget.NewButton("Disconnect", func() {
		s.stopDashboardRefresh()
		s.disconnect()
		s.showSSHConnectionScreen()
	})

	// Stop any existing auto-refresh goroutine and wait for it to finish
	s.stopDashboardRefresh()

	// Create new stop channels and start auto-refresh timer
	s.dashboardStopChan = make(chan bool)
	s.dashboardStopped = make(chan bool)
	go s.autoRefreshDashboard(logArea, statusIndicator, infoLabel)

	// Layout
	controlButtons := container.NewHBox(
		stopBtn,
		clearLogsBtn,
	)

	content := container.NewVBox(
		title,
		connInfo,
		widget.NewLabel(""),
		statusIndicator,
		widget.NewLabel(""),
		infoLabel,
		widget.NewLabel(""),
		widget.NewLabel("Recent Activity:"),
		logScroll,
		widget.NewLabel(""),
		controlButtons,
		widget.NewLabel(""),
		disconnectBtn,
	)

	s.window.SetContent(content)
}

// autoRefreshDashboard periodically refreshes the dashboard
func (s *AppState) autoRefreshDashboard(logArea, statusIndicator, infoLabel *widget.Label) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	defer func() {
		// Signal that we've stopped
		if s.dashboardStopped != nil {
			close(s.dashboardStopped)
		}
	}()

	for {
		select {
		case <-s.dashboardStopChan:
			return // Stop signal received
		case <-ticker.C:
			if s.daemonClient == nil {
				return // Disconnected
			}

			// Get updated status (with error handling to prevent crashes)
			status, err := s.daemonClient.GetStatus()
			if err != nil {
				continue // Skip this update if daemon is busy or error occurs
			}

			// Update status indicator
			fyne.Do(func() {
				if statusIndicator != nil {
					statusIndicator.SetText(fmt.Sprintf("● %s", strings.ToUpper(string(status.State))))
				}
			})

			// Update info
			infoText := fmt.Sprintf(
				"Monitoring %d websites\nEmail alerts: %s\n\nTotal checks: %d\nLast check: %s",
				status.WebsiteCount,
				status.Email,
				status.Stats.TotalChecks,
				formatTime(status.Stats.LastCheckTime),
			)
			fyne.Do(func() {
				if infoLabel != nil {
					infoLabel.SetText(infoText)
				}
			})

			// Update logs
			logs, err := s.daemonClient.GetLogs(50)
			if err == nil {
				logText := strings.Join(logs, "\n")
				fyne.Do(func() {
					if logArea != nil {
						logArea.SetText(logText)
					}
				})
			}
		}
	}
}

// stopDashboardRefresh stops any running dashboard refresh goroutine and waits for it to finish
func (s *AppState) stopDashboardRefresh() {
	if s.dashboardStopChan != nil {
		// Send stop signal
		select {
		case s.dashboardStopChan <- true:
			// Signal sent successfully, now wait for goroutine to finish
			if s.dashboardStopped != nil {
				select {
				case <-s.dashboardStopped:
					// Goroutine has stopped
				case <-time.After(2 * time.Second):
					// Timeout waiting for goroutine to stop
				}
			}
		default:
			// Channel already closed or receiver not listening
		}
		s.dashboardStopChan = nil
		s.dashboardStopped = nil
	}
}

// formatTime formats a time for display
func formatTime(t time.Time) string {
	if t.IsZero() {
		return "Never"
	}
	return t.Format("2006-01-02 15:04:05")
}
