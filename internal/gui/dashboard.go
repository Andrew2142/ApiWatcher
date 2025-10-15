package gui

import (
	"fmt"
	"strings"

	"url-checker/internal/daemon"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// showDashboardScreen shows the main monitoring dashboard with site stats
func (s *AppState) showDashboardScreen() {
	// Get initial status and website stats
	status, err := s.daemonClient.GetStatus()
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to get status: %v", err), s.window)
		return
	}

	websiteStats, err := s.daemonClient.GetWebsiteStats()
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to get website stats: %v", err), s.window)
		return
	}

	// Cache the stats
	s.cachedWebsiteStats = websiteStats

	// Title and connection info
	title := widget.NewLabelWithStyle("Monitoring Dashboard", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	connInfo := widget.NewLabel("Connected to: " + s.sshConn.Config().Host)
	connInfo.TextStyle.Italic = true

	// Status indicator
	statusIndicator := createStatusIndicator(strings.ToUpper(string(status.State)))

	// Refresh button
	var siteList *widget.List
	var detailsContainer *fyne.Container
	var logArea *widget.Label
	var selectedSiteIndex *int

	refreshButton := widget.NewButton("Refresh", func() {
		s.refreshDashboard(siteList, detailsContainer, logArea, statusIndicator, selectedSiteIndex)
	})

	// Header row
	headerRow := container.NewBorder(
		nil, nil,
		container.NewHBox(title, connInfo),
		container.NewHBox(statusIndicator, refreshButton),
	)

	// SMTP warning if not configured
	var smtpWarning *fyne.Container
	if !status.HasSMTP {
		warningLabel := widget.NewLabel("⚠️  SMTP not configured on daemon - email alerts will not work!")
		warningLabel.TextStyle.Bold = true
		configureBtn := widget.NewButton("Configure Now", func() {
			s.showEditSMTPScreen()
		})
		smtpWarning = container.NewBorder(nil, nil, warningLabel, configureBtn)
	}

	// Create site list and details panel
	selectedIndex := -1
	selectedSiteIndex = &selectedIndex
	detailsContainer = createEmptyState("Select a site to view details")

	siteList = widget.NewList(
		func() int {
			return len(s.cachedWebsiteStats)
		},
		func() fyne.CanvasObject {
			// Just a label for the URL, no dot
			return widget.NewLabel("")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(s.cachedWebsiteStats) {
				return
			}

			stats := s.cachedWebsiteStats[id]
			urlLabel := obj.(*widget.Label)

			// Just show the URL
			urlLabel.Text = stats.URL
			if int(id) == *selectedSiteIndex {
				urlLabel.TextStyle.Bold = true
			} else {
				urlLabel.TextStyle.Bold = false
			}
		},
	)

	siteList.OnSelected = func(id widget.ListItemID) {
		*selectedSiteIndex = int(id)
		if int(id) < len(s.cachedWebsiteStats) {
			// Update details panel with selected site
			detailsContainer.Objects = createDetailsPanel(s.cachedWebsiteStats[id]).Objects
			detailsContainer.Refresh()
			siteList.Refresh()
		}
	}

	// Set minimum size for site list
	siteListContainer := container.NewBorder(
		widget.NewLabelWithStyle("Monitored Sites", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nil, nil, nil,
		siteList,
	)

	// Details panel with scroll
	detailsScroll := container.NewScroll(detailsContainer)
	detailsScroll.SetMinSize(fyne.NewSize(400, 400))

	// Split view for site list and details
	splitView := container.NewHSplit(
		siteListContainer,
		detailsScroll,
	)
	splitView.Offset = 0.3 // 30% for list, 70% for details

	// Recent Activity Log section
	logs, _ := s.daemonClient.GetLogs(50)
	reverseLogs(logs)
	logText := strings.Join(logs, "\n")
	logArea = widget.NewLabel(logText)
	logArea.Wrapping = fyne.TextWrapWord

	logScroll := container.NewScroll(logArea)
	logScroll.SetMinSize(fyne.NewSize(600, 150))

	logSection := container.NewBorder(
		widget.NewLabelWithStyle("Recent Activity", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nil, nil, nil,
		logScroll,
	)

	// Control buttons
	stopBtn := widget.NewButton("Stop Monitoring", func() {
		if err := s.daemonClient.Stop(); err != nil {
			dialog.ShowError(fmt.Errorf("failed to stop: %v", err), s.window)
			return
		}
		s.showDaemonStoppedScreen()
	})

	clearLogsBtn := widget.NewButton("Clear Logs", func() {
		if err := s.daemonClient.ClearLogs(); err != nil {
			dialog.ShowError(fmt.Errorf("failed to clear logs: %v", err), s.window)
			return
		}
		logArea.SetText("")
		logArea.Refresh()
	})

	disconnectBtn := widget.NewButton("Disconnect", func() {
		s.disconnect()
		s.showSSHConnectionScreen()
	})

	smtpBtn := widget.NewButton("SMTP Settings", func() {
		s.showEditSMTPScreen()
	})

	controlButtons := container.NewHBox(
		stopBtn,
		clearLogsBtn,
		smtpBtn,
		layout.NewSpacer(),
		disconnectBtn,
	)

	// Main layout
	topSection := container.NewVBox(headerRow)
	if smtpWarning != nil {
		topSection.Add(smtpWarning)
	}
	topSection.Add(widget.NewSeparator())

	content := container.NewBorder(
		topSection,
		container.NewVBox(
			widget.NewSeparator(),
			controlButtons,
		),
		nil, nil,
		container.NewVBox(
			splitView,
			widget.NewSeparator(),
			logSection,
		),
	)

	s.window.SetContent(content)
}

// refreshDashboard manually refreshes dashboard data
func (s *AppState) refreshDashboard(siteList *widget.List, detailsContainer *fyne.Container, logArea *widget.Label, statusIndicator *fyne.Container, selectedSiteIndex *int) {
	// Get updated status
	status, err := s.daemonClient.GetStatus()
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to refresh status: %v", err), s.window)
		return
	}

	// Update status indicator
	newIndicator := createStatusIndicator(strings.ToUpper(string(status.State)))
	statusIndicator.Objects = newIndicator.Objects
	statusIndicator.Refresh()

	// Get updated website stats
	websiteStats, err := s.daemonClient.GetWebsiteStats()
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to refresh website stats: %v", err), s.window)
		return
	}

	// Store updated stats for list
	s.cachedWebsiteStats = websiteStats

	// Refresh site list
	siteList.Refresh()

	// If a site is selected, update details
	if selectedSiteIndex != nil && *selectedSiteIndex >= 0 && *selectedSiteIndex < len(s.cachedWebsiteStats) {
		detailsContainer.Objects = createDetailsPanel(s.cachedWebsiteStats[*selectedSiteIndex]).Objects
		detailsContainer.Refresh()
	}

	// Update logs
	logs, err := s.daemonClient.GetLogs(50)
	if err == nil {
		reverseLogs(logs)
		logText := strings.Join(logs, "\n")
		logArea.SetText(logText)
		logArea.Refresh()
	}
}

// createDetailsPanel creates the details panel for a selected site
func createDetailsPanel(stats daemon.WebsiteStatsResponse) *fyne.Container {

	// Simple grid layout - 2 columns: label | value
	grid := container.New(layout.NewGridLayout(2))

	// Add essential information only
	addGridRow(grid, "", "") // spacer
	addGridRow(grid, "Status:", stats.CurrentStatus)
	addGridRow(grid, "Last Check:", formatRelativeTime(stats.LastCheckTime))
	addGridRow(grid, "", "") // spacer

	addGridRow(grid, "Total Checks:", fmt.Sprintf("%d", stats.TotalChecks))
	addGridRow(grid, "Failed Checks:", fmt.Sprintf("%d", stats.FailedChecks))
	addGridRow(grid, "Consecutive Failures:", fmt.Sprintf("%d", stats.ConsecutiveFailures))
	addGridRow(grid, "", "") // spacer

	addGridRow(grid, "Uptime (24h):", formatPercentage(stats.UptimeLast24Hours))
	addGridRow(grid, "Uptime (7d):", formatPercentage(stats.UptimeLast7Days))
	addGridRow(grid, "", "") // spacer

	addGridRow(grid, "Emails Sent:", fmt.Sprintf("%d", stats.EmailsSent))
	addGridRow(grid, "Last Alert:", formatRelativeTime(stats.LastAlertSent))
	addGridRow(grid, "", "") // spacer

	addGridRow(grid, "Monitored Since:", formatAbsoluteTime(stats.FirstMonitoredAt))
	addGridRow(grid, "Last Success:", formatRelativeTime(stats.LastSuccessTime))
	addGridRow(grid, "Last Failure:", formatRelativeTime(stats.LastFailureTime))
	addGridRow(grid, "", "") // spacer

	// Combine header and grid
	return container.NewVBox(
		widget.NewSeparator(),
		grid,
	)
}

// addGridRow adds a label-value pair to a grid
func addGridRow(grid *fyne.Container, label, value string) {
	labelWidget := widget.NewLabel(label)
	labelWidget.TextStyle.Bold = true

	valueWidget := widget.NewLabel(value)

	grid.Add(labelWidget)
	grid.Add(valueWidget)
}

// reverseLogs reverses a slice of strings in place
func reverseLogs(logs []string) {
	for i, j := 0, len(logs)-1; i < j; i, j = i+1, j-1 {
		logs[i], logs[j] = logs[j], logs[i]
	}
}
