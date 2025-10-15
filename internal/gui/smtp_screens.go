package gui

import (
	"fmt"
	"log"
	"strings"
	"url-checker/internal/config"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// showSMTPConfigScreen displays SMTP configuration screen
// nextScreen parameter determines where to go after saving
func (s *AppState) showSMTPConfigScreen(nextScreen func()) {
	title := widget.NewLabel("SMTP Configuration")
	title.TextStyle.Bold = true

	// Load existing config from daemon (or use defaults)
	var existingHost, existingPort, existingUsername, existingFrom string
	if s.daemonClient != nil {
		smtpData, err := s.daemonClient.GetSMTP()
		if err == nil && smtpData != nil {
			existingHost = smtpData["host"]
			existingPort = smtpData["port"]
			existingUsername = smtpData["username"]
			existingFrom = smtpData["from"]
		}
	}

	// Set defaults if not loaded
	if existingHost == "" {
		existingHost = "smtp.gmail.com"
	}
	if existingPort == "" {
		existingPort = "587"
	}

	// Create form entries
	hostEntry := widget.NewEntry()
	hostEntry.SetText(existingHost)
	hostEntry.SetPlaceHolder("smtp.gmail.com")

	portEntry := widget.NewEntry()
	portEntry.SetText(existingPort)
	portEntry.SetPlaceHolder("587")

	usernameEntry := widget.NewEntry()
	usernameEntry.SetText(existingUsername)
	usernameEntry.SetPlaceHolder("your-email@gmail.com")

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("your-app-password")

	fromEntry := widget.NewEntry()
	fromEntry.SetText(existingFrom)
	fromEntry.SetPlaceHolder("alerts@yourdomain.com")

	// Help text for common providers
	helpText := widget.NewLabel(`Common SMTP Settings:
Gmail: smtp.gmail.com:587 (use app password)
Outlook: smtp-mail.outlook.com:587
Yahoo: smtp.mail.yahoo.com:587
Office365: smtp.office365.com:587`)
	helpText.Wrapping = fyne.TextWrapWord

	saveBtn := widget.NewButton("Save Configuration", func() {
		newConfig := &config.SMTPConfig{
			Host:     strings.TrimSpace(hostEntry.Text),
			Port:     strings.TrimSpace(portEntry.Text),
			Username: strings.TrimSpace(usernameEntry.Text),
			Password: passwordEntry.Text,
			From:     strings.TrimSpace(fromEntry.Text),
		}

		// Validate
		if err := config.ValidateSMTPConfig(newConfig); err != nil {
			dialog.ShowError(fmt.Errorf("validation error: %v", err), s.window)
			return
		}

		// Check daemon connection
		if s.daemonClient == nil {
			dialog.ShowError(fmt.Errorf("not connected to daemon"), s.window)
			return
		}

		// Send to daemon
		if err := s.daemonClient.SetSMTP(newConfig.Host, newConfig.Port, newConfig.Username, newConfig.Password, newConfig.From); err != nil {
			log.Printf("⚠️  Failed to save SMTP config to daemon: %v", err)
			dialog.ShowError(fmt.Errorf("failed to save SMTP config: %v", err), s.window)
			return
		}

		log.Println("✅ SMTP configuration saved to daemon")
		dialog.ShowInformation("Success", "SMTP configuration saved to daemon successfully!", s.window)

		// Proceed to next screen
		if nextScreen != nil {
			nextScreen()
		}
	})

	skipBtn := widget.NewButton("Skip for Now", func() {
		if nextScreen != nil {
			nextScreen()
		}
	})

	// Show existing config status
	var statusMsg string
	if existingUsername != "" {
		statusMsg = fmt.Sprintf("Current: %s (%s:%s)", existingUsername, existingHost, existingPort)
	} else {
		statusMsg = "No SMTP configuration found on daemon. Please set it up to receive alerts."
	}
	statusLabel := widget.NewLabel(statusMsg)
	statusLabel.Wrapping = fyne.TextWrapWord

	scroll := container.NewScroll(container.NewVBox(
		title,
		widget.NewLabel(""),
		statusLabel,
		widget.NewLabel(""),
		widget.NewLabel("SMTP Host:"),
		hostEntry,
		widget.NewLabel("SMTP Port:"),
		portEntry,
		widget.NewLabel(""),
		widget.NewLabel("Username (Email):"),
		usernameEntry,
		widget.NewLabel("Password (App Password):"),
		passwordEntry,
		widget.NewLabel(""),
		widget.NewLabel("From Address:"),
		fromEntry,
		widget.NewLabel(""),
		widget.NewSeparator(),
		helpText,
		widget.NewSeparator(),
		widget.NewLabel(""),
		saveBtn,
		skipBtn,
	))

	s.window.SetContent(scroll)
}

// showEditSMTPScreen displays SMTP edit screen (accessible from dashboard)
func (s *AppState) showEditSMTPScreen() {
	title := widget.NewLabel("Edit SMTP Configuration")
	title.TextStyle.Bold = true

	// Load existing config from daemon
	var existingHost, existingPort, existingUsername, existingFrom string
	if s.daemonClient != nil {
		smtpData, err := s.daemonClient.GetSMTP()
		if err == nil && smtpData != nil {
			existingHost = smtpData["host"]
			existingPort = smtpData["port"]
			existingUsername = smtpData["username"]
			existingFrom = smtpData["from"]
		}
	}

	// Set defaults if not loaded
	if existingHost == "" {
		existingHost = "smtp.gmail.com"
	}
	if existingPort == "" {
		existingPort = "587"
	}

	// Create form entries
	hostEntry := widget.NewEntry()
	hostEntry.SetText(existingHost)
	hostEntry.SetPlaceHolder("smtp.gmail.com")

	portEntry := widget.NewEntry()
	portEntry.SetText(existingPort)
	portEntry.SetPlaceHolder("587")

	usernameEntry := widget.NewEntry()
	usernameEntry.SetText(existingUsername)
	usernameEntry.SetPlaceHolder("your-email@gmail.com")

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("your-app-password (leave empty to keep current)")

	fromEntry := widget.NewEntry()
	fromEntry.SetText(existingFrom)
	fromEntry.SetPlaceHolder("alerts@yourdomain.com")

	// Help text for common providers
	helpText := widget.NewLabel(`Common SMTP Settings:
Gmail: smtp.gmail.com:587 (use app password)
Outlook: smtp-mail.outlook.com:587
Yahoo: smtp.mail.yahoo.com:587
Office365: smtp.office365.com:587`)
	helpText.Wrapping = fyne.TextWrapWord

	saveBtn := widget.NewButton("Save Changes", func() {
		// If password is empty and we had existing config, we need to handle it
		password := passwordEntry.Text
		if password == "" && existingUsername != "" {
			dialog.ShowError(fmt.Errorf("password is required (cannot retrieve existing password for security)"), s.window)
			return
		}

		newConfig := &config.SMTPConfig{
			Host:     strings.TrimSpace(hostEntry.Text),
			Port:     strings.TrimSpace(portEntry.Text),
			Username: strings.TrimSpace(usernameEntry.Text),
			Password: password,
			From:     strings.TrimSpace(fromEntry.Text),
		}

		// Validate
		if err := config.ValidateSMTPConfig(newConfig); err != nil {
			dialog.ShowError(fmt.Errorf("validation error: %v", err), s.window)
			return
		}

		// Check daemon connection
		if s.daemonClient == nil {
			dialog.ShowError(fmt.Errorf("not connected to daemon"), s.window)
			return
		}

		// Send to daemon
		if err := s.daemonClient.SetSMTP(newConfig.Host, newConfig.Port, newConfig.Username, newConfig.Password, newConfig.From); err != nil {
			log.Printf("⚠️  Failed to save SMTP config to daemon: %v", err)
			dialog.ShowError(fmt.Errorf("failed to save SMTP config: %v", err), s.window)
			return
		}

		log.Println("✅ SMTP configuration saved to daemon")
		dialog.ShowInformation("Success", "SMTP configuration updated on daemon successfully!", s.window)
	})

	testBtn := widget.NewButton("Test Email", func() {
		// This would be a nice feature to add - send a test email
		dialog.ShowInformation("Test Email", "Test email feature coming soon!", s.window)
	})

	backBtn := widget.NewButton("Back to Dashboard", func() {
		s.showDashboardScreen()
	})

	scroll := container.NewScroll(container.NewVBox(
		title,
		widget.NewLabel(""),
		widget.NewLabel("Configure your SMTP settings for email alerts:"),
		widget.NewLabel(""),
		widget.NewLabel("SMTP Host:"),
		hostEntry,
		widget.NewLabel("SMTP Port:"),
		portEntry,
		widget.NewLabel(""),
		widget.NewLabel("Username (Email):"),
		usernameEntry,
		widget.NewLabel("Password (App Password):"),
		passwordEntry,
		widget.NewLabel(""),
		widget.NewLabel("From Address:"),
		fromEntry,
		widget.NewLabel(""),
		widget.NewSeparator(),
		helpText,
		widget.NewSeparator(),
		widget.NewLabel(""),
		container.NewHBox(saveBtn, testBtn),
		backBtn,
	))

	s.window.SetContent(scroll)
}

// showSMTPSetupPrompt shows a prompt to set up SMTP (during initial connection)
func (s *AppState) showSMTPSetupPrompt(nextScreen func()) {
	// Check if SMTP is already configured on daemon
	var alreadyConfigured bool
	if s.daemonClient != nil {
		smtpData, err := s.daemonClient.GetSMTP()
		if err == nil && smtpData != nil && smtpData["username"] != "" {
			// Already configured, skip to next screen
			alreadyConfigured = true
		}
	}

	if alreadyConfigured {
		if nextScreen != nil {
			nextScreen()
		}
		return
	}

	title := widget.NewLabel("SMTP Setup")
	title.TextStyle.Bold = true

	setupBtn := widget.NewButton("Configure SMTP Now", func() {
		s.showSMTPConfigScreen(nextScreen)
	})

	skipBtn := widget.NewButton("Skip and Configure Later", func() {
		if nextScreen != nil {
			nextScreen()
		}
	})

	content := container.NewVBox(
		title,
		widget.NewLabel(""),
		widget.NewLabel("Would you like to configure email alerts?"),
		widget.NewLabel(""),
		widget.NewLabel("You can set up SMTP settings now or configure them later from the dashboard."),
		widget.NewLabel("Email alerts will notify you when monitored URLs have issues."),
		widget.NewLabel(""),
		setupBtn,
		skipBtn,
	)

	s.window.SetContent(content)
}

