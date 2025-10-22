package gui

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
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
			fyne.Do(func() {
				progress.Hide()
				dialog.ShowError(fmt.Errorf("failed to start tunnel: %v", err), s.window)
			})
			return
		}

		s.localTunnelPort = localPort

		// Create daemon client
		s.daemonClient = daemon.NewClient(fmt.Sprintf("localhost:%d", localPort))

		// Connect to daemon
		if err := s.daemonClient.Connect(); err != nil {
			fyne.Do(func() {
				progress.Hide()
				dialog.ShowError(fmt.Errorf("failed to connect to daemon: %v", err), s.window)
			})
			return
		}

		// Get daemon status
		status, err := s.daemonClient.GetStatus()
		if err != nil {
			fyne.Do(func() {
				progress.Hide()
				dialog.ShowError(fmt.Errorf("failed to get daemon status: %v", err), s.window)
			})
			return
		}

		log.Printf("Connected to daemon. State: %s, HasConfig: %v", status.State, status.HasConfig)

		fyne.Do(func() {
			progress.Hide()
		})

		// Show appropriate screen based on daemon state
		switch status.State {
		case daemon.StateRunning:
			// Monitoring is active
			fyne.Do(func() {
				s.showDashboardScreen()
			})
		case daemon.StatePaused:
			// Monitoring is paused
			fyne.Do(func() {
				s.showDashboardScreen()
			})
		case daemon.StateStopped:
			if status.HasConfig {
				// Has config but not running
				fyne.Do(func() {
					s.showDaemonStoppedScreen()
				})
			} else {
				// No config - need to configure
				fyne.Do(func() {
					s.showLoadConfigScreen()
				})
			}
		default:
			// Unknown state
			fyne.Do(func() {
				s.showLoadConfigScreen()
			})
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
		// Widget updates must be done on UI thread
		fyne.Do(func() {
			currentText := logArea.Text
			if currentText == "" {
				logArea.SetText(msg)
			} else {
				logArea.SetText(currentText + "\n" + msg)
			}
			logArea.Refresh()
		})
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

		fyne.Do(func() {
			statusLabel.SetText("❌ Requirements not met - Go installation required")
		})
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
	
	// Widget updates must be done on UI thread
	fyne.Do(func() {
		statusLabel.SetText("✅ Ready to install")
		statusLabel.Refresh()
		installBtn.Enable()
	})
}

// installDaemon installs the daemon on the remote server - FULLY AUTOMATED!
func (s *AppState) installDaemon(statusLabel, logArea *widget.Label) {
	// Widget updates must be done on UI thread
	fyne.Do(func() {
		statusLabel.SetText("Installing daemon...")
		statusLabel.Refresh()
	})

	updateLog := func(msg string) {
		log.Println(msg)
		// Widget updates must be done on UI thread
		fyne.Do(func() {
			currentText := logArea.Text
			if currentText == "" {
				logArea.SetText(msg)
			} else {
				logArea.SetText(currentText + "\n" + msg)
			}
			logArea.Refresh()
		})
	}

	go func() {
		updateLog("")
		updateLog("🚀 Starting AUTOMATED daemon installation...")
		updateLog("")

		// Create directory structure
		updateLog("📁 Creating directories on server...")
		_, err := s.sshConn.RunCommand("mkdir -p ~/.apiwatcher/bin ~/.apiwatcher/config ~/.apiwatcher/snapshots ~/.apiwatcher/logs")
		if err != nil {
			updateLog(fmt.Sprintf("❌ Failed to create directories: %v", err))
			fyne.Do(func() {
				statusLabel.SetText("❌ Installation failed")
				statusLabel.Refresh()
			})
			return
		}
		updateLog("✅ Directories created successfully")
		updateLog("")

		// Build daemon for Linux locally
		updateLog("🔨 Building daemon binary for Linux...")
		updateLog("   This will take 10-30 seconds...")
		
		err = s.buildDaemonBinary()
		if err != nil {
			updateLog(fmt.Sprintf("❌ Build failed: %v", err))
			updateLog("")
			updateLog("Please ensure you have Go installed locally and run:")
			updateLog("  cd /home/andy/Dev/url-checker")
			updateLog("  GOOS=linux GOARCH=amd64 go build -o apiwatcher-daemon-linux ./cmd/apiwatcher-daemon")
			fyne.Do(func() {
				statusLabel.SetText("❌ Build failed")
				statusLabel.Refresh()
			})
			return
		}
		updateLog("✅ Binary built successfully!")
		updateLog("")

		// Upload binary to server
		updateLog("📤 Uploading daemon binary to server...")
		updateLog("   This may take a few seconds...")
		
		err = s.uploadDaemonBinary()
		if err != nil {
			updateLog(fmt.Sprintf("❌ Upload failed: %v", err))
			fyne.Do(func() {
				statusLabel.SetText("❌ Upload failed")
				statusLabel.Refresh()
			})
			return
		}
		updateLog("✅ Binary uploaded successfully!")
		updateLog("")

		// Make executable
		updateLog("🔧 Setting executable permissions...")
		_, err = s.sshConn.RunCommand("chmod +x ~/.apiwatcher/bin/apiwatcher-daemon")
		if err != nil {
			updateLog(fmt.Sprintf("❌ Failed to set permissions: %v", err))
			fyne.Do(func() {
				statusLabel.SetText("❌ Installation failed")
				statusLabel.Refresh()
			})
			return
		}
		updateLog("✅ Permissions set")
		updateLog("")

	// Start daemon
	updateLog("▶️  Starting daemon...")
	_, err = s.sshConn.RunCommand("pkill -f apiwatcher-daemon 2>/dev/null; setsid nohup ~/.apiwatcher/bin/apiwatcher-daemon > ~/.apiwatcher/logs/daemon.log 2>&1 < /dev/null &")
	if err != nil {
		updateLog(fmt.Sprintf("⚠️  Start command returned error (may be OK): %v", err))
	}
		
		// Wait for daemon to start
		updateLog("⏳ Waiting for daemon to initialize...")
		time.Sleep(2 * time.Second)
		
		// Verify daemon is running
		output, err := s.sshConn.RunCommand("pgrep -f apiwatcher-daemon")
		if err != nil || output == "" {
			updateLog("⚠️  Could not verify daemon is running")
			updateLog("   Please check: ps aux | grep apiwatcher-daemon")
		} else {
			updateLog(fmt.Sprintf("✅ Daemon is running! (PID: %s)", strings.TrimSpace(output)))
		}
		updateLog("")

		updateLog("🎉 Installation complete!")
		updateLog("   The daemon is now running on your server")
		updateLog("   Click 'OK' to connect to the daemon...")
		updateLog("")

		// Widget updates must be done on UI thread
		fyne.Do(func() {
			statusLabel.SetText("✅ Installation complete!")
			statusLabel.Refresh()
		})

		// Wait a moment, then prompt for SMTP setup, then connect to daemon
		time.Sleep(2 * time.Second)
		fyne.Do(func() {
			// Show SMTP setup prompt, which will then connect to daemon
			s.showSMTPSetupPrompt(func() {
				s.connectToDaemon()
			})
		})
	}()
}

// showDaemonStoppedScreen shows when daemon is installed but monitoring is stopped
func (s *AppState) showDaemonStoppedScreen() {
	// Stop auto-refresh when leaving dashboard
	s.stopAutoRefresh()

	title := widget.NewLabel("Daemon Stopped")
	title.TextStyle.Bold = true

	status, err := s.daemonClient.GetStatus()
	var infoText string
	if err != nil || status == nil {
		infoText = "The daemon is installed but monitoring is not running."
	} else {
		infoText = fmt.Sprintf("The daemon is installed but monitoring is not running.\n\nLast known configuration:\n- Websites: %d\n- Email: %s",
			status.WebsiteCount, status.Email)
	}

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

// buildDaemonBinary builds the daemon binary for Linux on the local machine
func (s *AppState) buildDaemonBinary() error {
	// Get the working directory (should be the project root)
	workDir := "/home/andy/Dev/url-checker"
	
	// Build command for Linux
	cmd := fmt.Sprintf("cd %s && GOOS=linux GOARCH=amd64 go build -o apiwatcher-daemon-linux ./cmd/apiwatcher-daemon", workDir)
	
	// Execute the build command locally
	output, err := executeLocalCommand(cmd)
	if err != nil {
		return fmt.Errorf("build failed: %v\nOutput: %s", err, output)
	}
	
	return nil
}

// uploadDaemonBinary uploads the built daemon binary to the server
func (s *AppState) uploadDaemonBinary() error {
	localPath := "/home/andy/Dev/url-checker/apiwatcher-daemon-linux"
	remotePath := "~/.apiwatcher/bin/apiwatcher-daemon"
	
	// Use SCP to upload the file
	return s.sshConn.UploadFile(localPath, remotePath)
}

// executeLocalCommand executes a shell command on the local machine
func executeLocalCommand(command string) (string, error) {
	cmd := exec.Command("bash", "-c", command)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// disconnect closes connections to daemon and SSH
func (s *AppState) disconnect() {
	// Stop auto-refresh timer
	s.stopAutoRefresh()

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
