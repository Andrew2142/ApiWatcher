package gui

import (
	"fmt"
	"log"
	"strings"
	"url-checker/internal/daemon"
	"url-checker/internal/remote"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// Step 1: SSH Connection Screen
func (s *AppState) showSSHConnectionScreen() {
	title := widget.NewLabel("API Watcher - Server Connection")
	title.TextStyle.Bold = true

	// Load saved profiles
	profiles, err := remote.ListProfiles()
	if err != nil {
		log.Printf("Failed to load profiles: %v", err)
	}

	// Load preferences to highlight last connected server
	prefs, _ := LoadPreferences()
	lastServer := prefs.LastConnectedServer

	var content *fyne.Container

	if len(profiles) > 0 {
		// Show saved servers
		var serverNames []string
		for _, profile := range profiles {
			info := fmt.Sprintf("%s (%s@%s)", profile.Name, profile.Config.Username, profile.Config.Host)
			serverNames = append(serverNames, info)
		}

		selectedIndex := 0
		radio := widget.NewRadioGroup(serverNames, func(value string) {
			for i, name := range serverNames {
				if name == value {
					selectedIndex = i
					break
				}
			}
		})

		// Select last connected server if available
		selectedName := serverNames[0]
		if lastServer != "" {
			for i, profile := range profiles {
				if profile.Name == lastServer {
					selectedName = serverNames[i]
					selectedIndex = i
					break
				}
			}
		}
		radio.SetSelected(selectedName)

		connectBtn := widget.NewButton("Connect to Selected Server", func() {
			profile := profiles[selectedIndex]
			s.saveLastConnection(profile.Name)
			s.connectToServer(profile.Config)
		})

		newBtn := widget.NewButton("Add New Server", func() {
			s.showNewServerScreen()
		})

		localBtn := widget.NewButton("Run Locally", func() {
			s.handleLocalConnection()
		})

		scroll := container.NewScroll(radio)
		scroll.SetMinSize(fyne.NewSize(400, 300))

		content = container.NewVBox(
			title,
			widget.NewLabel(""),
			widget.NewLabel("Select a server to connect to:"),
			scroll,
			widget.NewLabel(""),
			connectBtn,
			newBtn,
			widget.NewLabel(""),
			widget.NewLabel("Or run the daemon locally:"),
			localBtn,
		)
	} else {
		// No saved servers, show new server form
		newBtn := widget.NewButton("Add New Server", func() {
			s.showNewServerScreen()
		})

		localBtn := widget.NewButton("Run Locally", func() {
			s.handleLocalConnection()
		})

		content = container.NewVBox(
			title,
			widget.NewLabel(""),
			widget.NewLabel("No saved servers found."),
			widget.NewLabel(""),
			newBtn,
			widget.NewLabel(""),
			widget.NewLabel("Or run the daemon locally:"),
			localBtn,
		)
	}

	s.window.SetContent(content)
}

// Step 2: New Server Screen
func (s *AppState) showNewServerScreen() {
	title := widget.NewLabel("Add New Server")
	title.TextStyle.Bold = true

	// Server details
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("My Production Server")

	hostEntry := widget.NewEntry()
	hostEntry.SetPlaceHolder("192.168.1.100 or myserver.com")

	portEntry := widget.NewEntry()
	portEntry.SetText("22")

	usernameEntry := widget.NewEntry()
	usernameEntry.SetPlaceHolder("ubuntu")

	// Authentication method
	authMethod := "key"
	authRadio := widget.NewRadioGroup([]string{"SSH Key", "Password"}, func(value string) {
		if value == "SSH Key" {
			authMethod = "key"
		} else {
			authMethod = "password"
		}
	})
	authRadio.SetSelected("SSH Key")

	keyPathEntry := widget.NewEntry()
	keyPathEntry.SetPlaceHolder("/home/user/.ssh/id_rsa")

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Server password")
	passwordEntry.Hide()

	// Update visibility based on auth method
	authRadio.OnChanged = func(value string) {
		if value == "SSH Key" {
			authMethod = "key"
			keyPathEntry.Show()
			passwordEntry.Hide()
		} else {
			authMethod = "password"
			keyPathEntry.Hide()
			passwordEntry.Show()
		}
		s.window.Content().Refresh()
	}

	daemonPortEntry := widget.NewEntry()
	daemonPortEntry.SetText("9876")

	saveCheckbox := widget.NewCheck("Save this server for future use", nil)
	saveCheckbox.SetChecked(true)

	testBtn := widget.NewButton("Test Connection", func() {
		config := &remote.SSHConfig{
			Host:       strings.TrimSpace(hostEntry.Text),
			Port:       strings.TrimSpace(portEntry.Text),
			Username:   strings.TrimSpace(usernameEntry.Text),
			AuthMethod: authMethod,
			Password:   passwordEntry.Text,
			KeyPath:    strings.TrimSpace(keyPathEntry.Text),
			DaemonPort: strings.TrimSpace(daemonPortEntry.Text),
		}

		// Validate input
		if config.Host == "" || config.Username == "" {
			dialog.ShowError(fmt.Errorf("please fill in host and username"), s.window)
			return
		}

		if config.AuthMethod == "key" && config.KeyPath == "" {
			dialog.ShowError(fmt.Errorf("please specify SSH key path"), s.window)
			return
		}

		if config.AuthMethod == "password" && config.Password == "" {
			dialog.ShowError(fmt.Errorf("please enter password"), s.window)
			return
		}

		// Try to connect
		dialog.ShowInformation("Testing", "Connecting to server...", s.window)

		go func() {
			conn, err := remote.Connect(config)
			if err != nil {
				fyne.Do(func() {
					dialog.ShowError(fmt.Errorf("connection failed: %v", err), s.window)
				})
				return
			}
			defer conn.Close()

			if err := conn.TestConnection(); err != nil {
				fyne.Do(func() {
					dialog.ShowError(fmt.Errorf("connection test failed: %v", err), s.window)
				})
				return
			}

			fyne.Do(func() {
				dialog.ShowInformation("Success", "Connection successful!", s.window)
			})
		}()
	})

	connectBtn := widget.NewButton("Connect", func() {
		config := &remote.SSHConfig{
			Host:       strings.TrimSpace(hostEntry.Text),
			Port:       strings.TrimSpace(portEntry.Text),
			Username:   strings.TrimSpace(usernameEntry.Text),
			AuthMethod: authMethod,
			Password:   passwordEntry.Text,
			KeyPath:    strings.TrimSpace(keyPathEntry.Text),
			DaemonPort: strings.TrimSpace(daemonPortEntry.Text),
		}

		// Validate input
		if config.Host == "" || config.Username == "" {
			dialog.ShowError(fmt.Errorf("please fill in host and username"), s.window)
			return
		}

		if config.AuthMethod == "key" && config.KeyPath == "" {
			dialog.ShowError(fmt.Errorf("please specify SSH key path"), s.window)
			return
		}

		if config.AuthMethod == "password" && config.Password == "" {
			dialog.ShowError(fmt.Errorf("please enter password"), s.window)
			return
		}

		// Save profile if requested
		serverName := ""
		if saveCheckbox.Checked {
			serverName = strings.TrimSpace(nameEntry.Text)
			if serverName == "" {
				serverName = config.Host
			}
			if err := remote.SaveProfile(serverName, config); err != nil {
				log.Printf("Warning: failed to save profile: %v", err)
			}
			s.saveLastConnection(serverName)
		}

		// Connect
		s.connectToServer(config)
	})

	backBtn := widget.NewButton("Back", func() {
		s.showSSHConnectionScreen()
	})

	scroll := container.NewScroll(container.NewVBox(
		title,
		widget.NewLabel(""),
		widget.NewLabel("Server Name:"),
		nameEntry,
		widget.NewLabel(""),
		widget.NewLabel("Host:"),
		hostEntry,
		widget.NewLabel("Port:"),
		portEntry,
		widget.NewLabel("Username:"),
		usernameEntry,
		widget.NewLabel(""),
		widget.NewLabel("Authentication Method:"),
		authRadio,
		widget.NewLabel(""),
		widget.NewLabel("SSH Key Path:"),
		keyPathEntry,
		widget.NewLabel("Password:"),
		passwordEntry,
		widget.NewLabel(""),
		widget.NewLabel("Daemon Port (leave default):"),
		daemonPortEntry,
		widget.NewLabel(""),
		saveCheckbox,
		widget.NewLabel(""),
		container.NewHBox(testBtn, connectBtn),
		backBtn,
	))

	s.window.SetContent(scroll)
}

// connectToServer handles the connection process
func (s *AppState) connectToServer(config *remote.SSHConfig) {
	// Show connecting dialog
	progress := dialog.NewProgressInfinite("Connecting", "Connecting to server...", s.window)
	progress.Show()

	go func() {
		// Connect to SSH
		conn, err := remote.Connect(config)
		if err != nil {
			fyne.Do(func() {
				progress.Hide()
				dialog.ShowError(fmt.Errorf("connection failed: %v", err), s.window)
			})
			return
		}

		// Store connection
		s.sshConn = conn

		// Check daemon status
		installed, err := conn.CheckDaemonInstalled()
		if err != nil {
			fyne.Do(func() {
				progress.Hide()
				dialog.ShowError(fmt.Errorf("failed to check daemon: %v", err), s.window)
			})
			return
		}

		fyne.Do(func() {
			progress.Hide()
		})

		if !installed {
			// Daemon not installed - show setup wizard
			fyne.Do(func() {
				s.showDaemonSetupScreen()
			})
		} else {
			// Daemon installed - connect to it
			fyne.Do(func() {
				s.connectToDaemon()
			})
		}
	}()
}

// handleLocalConnection handles the local daemon connection flow
func (s *AppState) handleLocalConnection() {
	progress := dialog.NewProgressInfinite("Connecting", "Connecting to local daemon...", s.window)
	progress.Show()

	go func() {
		// Try to connect to local daemon
		err := s.connectToLocalDaemon()
		if err != nil {
			fyne.Do(func() {
				progress.Hide()
				dialog.ShowError(err, s.window)
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

		log.Printf("Connected to local daemon. State: %s, HasConfig: %v", status.State, status.HasConfig)

		fyne.Do(func() {
			progress.Hide()
		})

		// Show appropriate screen based on daemon state
		switch status.State {
		case daemon.StateRunning:
			fyne.Do(func() {
				s.showDashboardScreen()
			})
		case daemon.StatePaused:
			fyne.Do(func() {
				s.showDashboardScreen()
			})
		case daemon.StateStopped:
			if status.HasConfig {
				fyne.Do(func() {
					s.showDaemonStoppedScreen()
				})
			} else {
				fyne.Do(func() {
					s.showLoadConfigScreen()
				})
			}
		default:
			fyne.Do(func() {
				s.showLoadConfigScreen()
			})
		}
	}()
}
