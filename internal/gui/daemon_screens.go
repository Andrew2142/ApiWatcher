package gui

import (
	"fmt"
	"log"
	"strings"
	"url-checker/internal/daemon"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// connectToDaemon establishes connection to the daemon and checks its status
func (s *AppState) connectToDaemon() {
	progress := dialog.NewProgressInfinite("Connecting", "Connecting to daemon...", s.window)
	progress.Show()

	go func() {
		// Start SSH tunnel
		localPort, err := s.sshConn.StartTunnel()
		if err != nil {
			progress.Hide()
			dialog.ShowError(fmt.Errorf("failed to start tunnel: %v", err), s.window)
			return
		}

		s.localTunnelPort = localPort

		// Create daemon client
		s.daemonClient = daemon.NewClient(fmt.Sprintf("localhost:%d", localPort))

		// Connect to daemon
		if err := s.daemonClient.Connect(); err != nil {
			progress.Hide()
			dialog.ShowError(fmt.Errorf("failed to connect to daemon: %v", err), s.window)
			return
		}

		// Get daemon status
		status, err := s.daemonClient.GetStatus()
		if err != nil {
			progress.Hide()
			dialog.ShowError(fmt.Errorf("failed to get daemon status: %v", err), s.window)
			return
		}

		log.Printf("Connected to daemon. State: %s, HasConfig: %v", status.State, status.HasConfig)

		progress.Hide()

		// Show appropriate screen based on daemon state
		switch status.State {
		case daemon.StateRunning:
			// Monitoring is active
			s.showDashboardScreen()
		case daemon.StatePaused:
			// Monitoring is paused
			s.showDashboardScreen()
		case daemon.StateStopped:
			if status.HasConfig {
				// Has config but not running
				s.showDaemonStoppedScreen()
			} else {
				// No config - need to configure
				s.showLoadConfigScreen()
			}
		default:
			// Unknown state
			s.showLoadConfigScreen()
		}
	}()
}

// showDaemonSetupScreen shows the daemon installation wizard
func (s *AppState) showDaemonSetupScreen() {
	title := widget.NewLabel("Daemon Setup")
	title.TextStyle.Bold = true

	statusLabel := widget.NewLabel("Checking server requirements...")
	logArea := widget.NewLabel("")
	logArea.Wrapping = fyne.TextWrapWord

	// Create a much larger scroll area for logs
	logScroll := container.NewScroll(logArea)
	logScroll.SetMinSize(fyne.NewSize(600, 300)) // Much larger log area

	installBtn := widget.NewButton("Install Daemon", func() {
		s.installDaemon(statusLabel, logArea)
	})
	installBtn.Disable()

	cancelBtn := widget.NewButton("Cancel", func() {
		s.showSSHConnectionScreen()
	})

	// Use border layout to give more space to logs
	topSection := container.NewVBox(
		title,
		widget.NewLabel(""),
		widget.NewLabel("The daemon is not installed on the server."),
		widget.NewLabel("Let's set it up!"),
		widget.NewLabel(""),
		statusLabel,
		widget.NewLabel(""),
	)

	bottomSection := container.NewVBox(
		widget.NewLabel(""),
		container.NewHBox(installBtn, cancelBtn),
	)

	content := container.NewBorder(
		topSection,    // top
		bottomSection, // bottom
		nil,           // left
		nil,           // right
		logScroll,     // center - takes up most space
	)

	s.window.SetContent(content)

	// Check requirements in background
	go s.checkServerRequirements(statusLabel, logArea, installBtn)
}

// checkServerRequirements checks if the server has required dependencies
func (s *AppState) checkServerRequirements(statusLabel, logArea *widget.Label, installBtn *widget.Button) {
	updateLog := func(msg string) {
		log.Println(msg)
		// Widget updates are thread-safe, just need to refresh
		currentText := logArea.Text
		if currentText == "" {
			logArea.SetText(msg)
		} else {
			logArea.SetText(currentText + "\n" + msg)
		}
		logArea.Refresh()
	}

	updateLog("🔍 Starting server requirements check...")
	updateLog("")

	// Check Go - try with PATH set
	updateLog("📦 Checking for Go installation...")
	// Try common Go paths for non-interactive shells
	output, err := s.sshConn.RunCommand("source ~/.bashrc 2>/dev/null; source ~/.profile 2>/dev/null; export PATH=$PATH:/usr/local/go/bin:~/go/bin; go version")

	if err != nil {
		updateLog("❌ Go not found. Please install Go 1.19+ on the server.")
		updateLog("")
		updateLog("📋 Installation instructions available!")

		// Show copyable installation commands
		s.showCopyableInstructions("Go Installation Commands (Install the latest Go version on your server)", []string{
			"# Download the latest stable Go version",
			"wget https://go.dev/dl/go1.25.2.linux-amd64.tar.gz",
			"",

			"# Remove any previous Go installation",
			"sudo rm -rf /usr/local/go",
			"",

			"# Extract Go to /usr/local",
			"sudo tar -C /usr/local -xzf go1.25.2.linux-amd64.tar.gz",
			"",

			"# Permanently add Go to PATH for future sessions",
			"echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc",
			"echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.zshrc",
			"",

			"# Reload your shell configuration",
			"source ~/.bashrc || source ~/.zshrc",
			"",

			"# Verify the installation",
			"go version",
		})

		statusLabel.SetText("❌ Requirements not met - Go installation required")
		return
	}
	updateLog(fmt.Sprintf("✅ Go found: %s", strings.TrimSpace(output)))
	updateLog("")

	// Check Chrome (for snapshots)
	updateLog("🌐 Checking for Chrome/Chromium...")
	_, err = s.sshConn.RunCommand("which google-chrome || which chromium-browser")
	if err != nil {
		updateLog("⚠️  Chrome not found. Snapshot mode will not work.")
		updateLog("   (This is optional - basic monitoring will still work)")
	} else {
		updateLog("✅ Chrome/Chromium found")
	}
	updateLog("")

	// Check disk space
	updateLog("💾 Checking disk space...")
	output, err = s.sshConn.RunCommand("df -h ~ | tail -1 | awk '{print $4}'")
	if err == nil {
		updateLog(fmt.Sprintf("✅ Available disk space: %s", strings.TrimSpace(output)))
	} else {
		updateLog("⚠️  Could not check disk space")
	}
	updateLog("")

	// Check network connectivity
	updateLog("🌍 Checking network connectivity...")
	_, err = s.sshConn.RunCommand("curl -s --max-time 5 https://www.google.com > /dev/null")
	if err != nil {
		updateLog("⚠️  Network connectivity test failed")
		updateLog("   (Monitoring may not work properly)")
	} else {
		updateLog("✅ Network connectivity OK")
	}
	updateLog("")

	updateLog("🎉 Server is ready for daemon installation!")
	updateLog("")
	updateLog("Click 'Install Daemon' to proceed...")
	
	// Widget updates are thread-safe, just need to refresh
	statusLabel.SetText("✅ Ready to install")
	statusLabel.Refresh()
	installBtn.Enable()
}

// installDaemon installs the daemon on the remote server
func (s *AppState) installDaemon(statusLabel, logArea *widget.Label) {
	// Widget updates are thread-safe, just need to refresh
	statusLabel.SetText("Installing daemon...")
	statusLabel.Refresh()

	updateLog := func(msg string) {
		log.Println(msg)
		// Widget updates are thread-safe, just need to refresh
		currentText := logArea.Text
		if currentText == "" {
			logArea.SetText(msg)
		} else {
			logArea.SetText(currentText + "\n" + msg)
		}
		logArea.Refresh()
	}

	go func() {
		updateLog("")
		updateLog("🚀 Starting daemon installation...")
		updateLog("")

		// Create directory structure
		updateLog("📁 Creating directories on server...")
		_, err := s.sshConn.RunCommand("mkdir -p ~/.apiwatcher/bin ~/.apiwatcher/config ~/.apiwatcher/snapshots ~/.apiwatcher/logs")
		if err != nil {
			updateLog(fmt.Sprintf("❌ Failed to create directories: %v", err))
			statusLabel.SetText("❌ Installation failed")
			return
		}
		updateLog("✅ Directories created successfully")
		updateLog("")

		// Build daemon for Linux
		updateLog("🔨 Building daemon binary for Linux...")
		updateLog("   This may take a moment...")

		// Note: This is a placeholder for now
		updateLog("⚠️  Automatic build not yet implemented")
		updateLog("")
		updateLog("📋 Manual steps required!")
		updateLog("   → A popup with copyable commands will appear")
		updateLog("   → Commands are customized for your server")

		// Show copyable daemon installation commands
		serverHost := s.sshConn.Config().Host
		serverUser := s.sshConn.Config().Username

		s.showCopyableInstructions("Daemon Installation Commands", []string{
			"# 1. Build daemon for Linux (run on your local machine):",
			"GOOS=linux GOARCH=amd64 go build -o apiwatcher-daemon-linux ./cmd/apiwatcher-daemon",
			"",
			"# 2. Upload to server:",
			fmt.Sprintf("scp apiwatcher-daemon-linux %s@%s:~/.apiwatcher/bin/apiwatcher-daemon", serverUser, serverHost),
			"",
			"# 3. Make executable:",
			fmt.Sprintf("ssh %s@%s 'chmod +x ~/.apiwatcher/bin/apiwatcher-daemon'", serverUser, serverHost),
			"",
			"# 4. Start daemon:",
			fmt.Sprintf("ssh %s@%s 'nohup ~/.apiwatcher/bin/apiwatcher-daemon > ~/.apiwatcher/logs/daemon.log 2>&1 &'", serverUser, serverHost),
		})

		updateLog("")

		// For now, just create a placeholder
		updateLog("📝 Creating placeholder daemon script...")
		script := `#!/bin/bash
echo "Daemon placeholder - please upload real binary"
echo "See installation instructions above"
sleep 10
`
		_, err = s.sshConn.RunCommand(fmt.Sprintf("cat > ~/.apiwatcher/bin/apiwatcher-daemon << 'EOF'\n%sEOF", script))
		if err != nil {
			updateLog(fmt.Sprintf("❌ Failed to create placeholder: %v", err))
		}

		// Make executable
		updateLog("🔧 Setting permissions...")
		_, err = s.sshConn.RunCommand("chmod +x ~/.apiwatcher/bin/apiwatcher-daemon")
		if err != nil {
			updateLog(fmt.Sprintf("⚠️  Could not set permissions: %v", err))
		} else {
			updateLog("✅ Permissions set")
		}
		updateLog("")

		updateLog("⚠️  Manual daemon upload required")
		updateLog("   Please follow the manual steps above to complete installation")
		updateLog("")
		updateLog("💡 Future versions will automate this process")

		// Widget updates are thread-safe, just need to refresh
		statusLabel.SetText("⚠️  Manual steps required")
		statusLabel.Refresh()
	}()
}

// showDaemonStoppedScreen shows when daemon is installed but monitoring is stopped
func (s *AppState) showDaemonStoppedScreen() {
	title := widget.NewLabel("Daemon Stopped")
	title.TextStyle.Bold = true

	status, _ := s.daemonClient.GetStatus()
	infoText := fmt.Sprintf("The daemon is installed but monitoring is not running.\n\nLast known configuration:\n- Websites: %d\n- Email: %s",
		status.WebsiteCount, status.Email)

	startBtn := widget.NewButton("Start Monitoring", func() {
		if err := s.daemonClient.Start(); err != nil {
			dialog.ShowError(fmt.Errorf("failed to start monitoring: %v", err), s.window)
			return
		}
		s.showDashboardScreen()
	})

	configureBtn := widget.NewButton("Change Configuration", func() {
		s.showLoadConfigScreen()
	})

	disconnectBtn := widget.NewButton("Disconnect", func() {
		s.disconnect()
		s.showSSHConnectionScreen()
	})

	content := container.NewVBox(
		title,
		widget.NewLabel(""),
		widget.NewLabel(infoText),
		widget.NewLabel(""),
		startBtn,
		configureBtn,
		widget.NewLabel(""),
		disconnectBtn,
	)

	s.window.SetContent(content)
}

// showCopyableInstructions shows a popup with selectable/copyable text
func (s *AppState) showCopyableInstructions(title string, commands []string) {
	// Join all commands with newlines
	allText := strings.Join(commands, "\n")

	// Create a multiline entry that's read-only but selectable
	textEntry := widget.NewEntry()
	textEntry.MultiLine = true
	textEntry.Wrapping = fyne.TextWrapWord
	textEntry.SetText(allText)

	// Make it large enough to see the content
	textEntry.Resize(fyne.NewSize(600, 300))

	// Create scroll container
	scroll := container.NewScroll(textEntry)
	scroll.SetMinSize(fyne.NewSize(600, 300))

	// Create content with buttons
	content := container.NewVBox(
		widget.NewLabel(title),
		widget.NewLabel("Select text to copy individual commands"),
		scroll,
	)

	// Show as custom dialog
	d := dialog.NewCustom("Installation Commands", "Close", content, s.window)
	d.Resize(fyne.NewSize(700, 450))
	d.Show()
}

// disconnect closes connections to daemon and SSH
func (s *AppState) disconnect() {
	if s.daemonClient != nil {
		s.daemonClient.Close()
		s.daemonClient = nil
	}
	if s.sshConn != nil {
		s.sshConn.Close()
		s.sshConn = nil
	}
	s.localTunnelPort = 0
}
