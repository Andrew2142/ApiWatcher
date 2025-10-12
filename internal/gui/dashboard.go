package gui

import (
	"fmt"
	"strings"
	"time"
	"url-checker/internal/daemon"

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

	// Control buttons
	var pauseResumeBtn *widget.Button
	if status.State == daemon.StateRunning {
		pauseResumeBtn = widget.NewButton("Pause Monitoring", func() {
			if err := s.daemonClient.Stop(); err != nil {
				dialog.ShowError(fmt.Errorf("failed to pause: %v", err), s.window)
				return
			}
			s.showDashboardScreen() // Refresh
		})
	} else if status.State == daemon.StatePaused {
		pauseResumeBtn = widget.NewButton("Resume Monitoring", func() {
			if err := s.daemonClient.Start(); err != nil {
				dialog.ShowError(fmt.Errorf("failed to resume: %v", err), s.window)
				return
			}
			s.showDashboardScreen() // Refresh
		})
	}

	stopBtn := widget.NewButton("Stop Monitoring", func() {
		if err := s.daemonClient.Stop(); err != nil {
			dialog.ShowError(fmt.Errorf("failed to stop: %v", err), s.window)
			return
		}
		s.showDaemonStoppedScreen()
	})

	refreshBtn := widget.NewButton("Refresh", func() {
		s.showDashboardScreen()
	})

	configBtn := widget.NewButton("Edit Configuration", func() {
		// Can only edit when stopped
		if status.State != daemon.StateStopped {
			dialog.ShowInformation("Cannot Edit",
				"Please stop monitoring before editing configuration.", s.window)
			return
		}
		s.showLoadConfigScreen()
	})

	disconnectBtn := widget.NewButton("Disconnect", func() {
		s.disconnect()
		s.showSSHConnectionScreen()
	})

	// Auto-refresh timer
	go s.autoRefreshDashboard(logArea, statusIndicator, infoLabel)

	// Layout
	controlButtons := container.NewHBox(
		pauseResumeBtn,
		stopBtn,
		refreshBtn,
		configBtn,
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

	for {
		select {
		case <-ticker.C:
			if s.daemonClient == nil {
				return // Disconnected
			}

			// Get updated status
			status, err := s.daemonClient.GetStatus()
			if err != nil {
				continue // Skip this update
			}

			// Update status indicator
			statusIndicator.SetText(fmt.Sprintf("● %s", strings.ToUpper(string(status.State))))

			// Update info
			infoText := fmt.Sprintf(
				"Monitoring %d websites\nEmail alerts: %s\n\nTotal checks: %d\nLast check: %s",
				status.WebsiteCount,
				status.Email,
				status.Stats.TotalChecks,
				formatTime(status.Stats.LastCheckTime),
			)
			infoLabel.SetText(infoText)

			// Update logs
			logs, err := s.daemonClient.GetLogs(50)
			if err == nil {
				logText := strings.Join(logs, "\n")
				logArea.SetText(logText)
			}
		}
	}
}

// formatTime formats a time for display
func formatTime(t time.Time) string {
	if t.IsZero() {
		return "Never"
	}
	return t.Format("2006-01-02 15:04:05")
}

