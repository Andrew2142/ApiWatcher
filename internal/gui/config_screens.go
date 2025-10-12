package gui

import (
	"fmt"
	"log"
	"strings"
	"url-checker/internal/config"
	"url-checker/internal/snapshot"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// Step 1: Load saved config or create new
func (s *AppState) showLoadConfigScreen() {
	title := widget.NewLabel("API Watcher - Configuration")
	title.TextStyle.Bold = true

	loadBtn := widget.NewButton("Load Saved Configuration", func() {
		s.showSelectSavedConfigScreen()
	})

	newBtn := widget.NewButton("Create New Configuration", func() {
		s.showNewConfigScreen()
	})

	content := container.NewVBox(
		title,
		widget.NewLabel(""),
		widget.NewLabel("Would you like to load a saved configuration or create a new one?"),
		widget.NewLabel(""),
		loadBtn,
		newBtn,
	)

	s.window.SetContent(content)
}

// Step 2a: Select from saved configs
func (s *AppState) showSelectSavedConfigScreen() {
	savedConfigs, err := config.LoadAllSavedConfigs()
	if err != nil {
		dialog.ShowError(fmt.Errorf("error loading saved configs: %v", err), s.window)
		s.showLoadConfigScreen()
		return
	}

	if len(savedConfigs) == 0 {
		dialog.ShowInformation("No Saved Configurations",
			"No saved configurations found. Please create a new one.", s.window)
		s.showNewConfigScreen()
		return
	}

	title := widget.NewLabel("Select Configuration")
	title.TextStyle.Bold = true

	var configNames []string
	for _, cfg := range savedConfigs {
		info := fmt.Sprintf("%s | Email: %s | Sites: %d | Snapshots: %d",
			cfg.Name, cfg.Email, len(cfg.Websites), len(cfg.SnapshotIDs))
		configNames = append(configNames, info)
	}

	selectedIndex := 0
	radio := widget.NewRadioGroup(configNames, func(value string) {
		for i, name := range configNames {
			if name == value {
				selectedIndex = i
				break
			}
		}
	})

	radio.SetSelected(configNames[0])

	// replace NewMax with NewStack
	minRadio := NewMinSized(radio, fyne.NewSize(300, 400))
	scroll := container.NewScroll(minRadio)
	scroll.SetMinSize(fyne.NewSize(300, 400))

	loadBtn := widget.NewButton("Load Selected", func() {
		selectedConfig := savedConfigs[selectedIndex]

		// Load the configuration
		s.cfg = &config.Config{
			Email:    selectedConfig.Email,
			Websites: selectedConfig.Websites,
		}

		// Load associated snapshots
		s.snapshotsByURL = make(map[string]*snapshot.Snapshot)
		for url, snapshotID := range selectedConfig.SnapshotIDs {
			snap, err := snapshot.LoadByID(snapshotID)
			if err != nil {
				log.Printf("Warning: couldn't load snapshot %s for %s: %v\n",
					snapshotID, url, err)
			} else {
				s.snapshotsByURL[url] = snap
				log.Printf("✅ Loaded snapshot '%s' for %s\n", snap.Name, url)
			}
		}

		s.loadedFromSaved = true

		// Show configuration details and ask to run
		s.showRunConfirmationScreen(selectedConfig.Name)
	})

	backBtn := widget.NewButton("Back", func() {
		s.showLoadConfigScreen()
	})

	content := container.NewVBox(
		title,
		widget.NewLabel(""),
		widget.NewLabel("Select a configuration to load:"),
		scroll,
		widget.NewLabel(""),
		loadBtn,
		backBtn,
	)

	s.window.SetContent(content)
}

// Step 2b: Create new config
func (s *AppState) showNewConfigScreen() {
	title := widget.NewLabel("New Configuration")
	title.TextStyle.Bold = true

	websitesEntry := widget.NewEntry()
	websitesEntry.SetPlaceHolder("https://example.com, https://api.example.com")
	websitesEntry.MultiLine = true

	emailEntry := widget.NewEntry()
	emailEntry.SetPlaceHolder("alerts@example.com")

	nextBtn := widget.NewButton("Next", func() {
		websites := strings.Split(websitesEntry.Text, ",")
		for i := range websites {
			websites[i] = strings.TrimSpace(websites[i])
		}

		email := strings.TrimSpace(emailEntry.Text)

		if len(websites) == 0 || websites[0] == "" {
			dialog.ShowError(fmt.Errorf("please enter at least one website URL"), s.window)
			return
		}

		if email == "" {
			dialog.ShowError(fmt.Errorf("please enter an alert email"), s.window)
			return
		}

		s.cfg = &config.Config{
			Email:    email,
			Websites: websites,
		}

		config.Save(s.cfg)
		s.loadedFromSaved = false

		// Move to snapshot configuration
		s.showSnapshotConfigScreen()
	})

	backBtn := widget.NewButton("Back", func() {
		s.showLoadConfigScreen()
	})

	content := container.NewVBox(
		title,
		widget.NewLabel(""),
		widget.NewLabel("Enter website URLs (comma separated):"),
		websitesEntry,
		widget.NewLabel(""),
		widget.NewLabel("Enter your alert email:"),
		emailEntry,
		widget.NewLabel(""),
		nextBtn,
		backBtn,
	)

	s.window.SetContent(content)
}

// Step 3: Run confirmation for loaded config
func (s *AppState) showRunConfirmationScreen(configName string) {
	title := widget.NewLabel("Configuration Loaded")
	title.TextStyle.Bold = true

	infoText := fmt.Sprintf("Configuration: %s\nEmail: %s\nWebsites: %s\nSnapshots: %d",
		configName,
		s.cfg.Email,
		strings.Join(s.cfg.Websites, ", "),
		len(s.snapshotsByURL))

	runBtn := widget.NewButton("Start Monitoring", func() {
		s.startMonitoring()
	})

	cancelBtn := widget.NewButton("Cancel", func() {
		s.showLoadConfigScreen()
	})

	content := container.NewVBox(
		title,
		widget.NewLabel(""),
		widget.NewLabel(infoText),
		widget.NewLabel(""),
		widget.NewLabel("Would you like to start monitoring with this configuration?"),
		widget.NewLabel(""),
		runBtn,
		cancelBtn,
	)

	s.window.SetContent(content)
}

// Step 8: Save configuration
func (s *AppState) showSaveConfigScreen() {
	if s.loadedFromSaved {
		// Already loaded from saved, just start monitoring
		s.startMonitoring()
		return
	}

	title := widget.NewLabel("Save Configuration")
	title.TextStyle.Bold = true

	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("My monitoring config")

	saveBtn := widget.NewButton("Save & Start Monitoring", func() {
		configName := strings.TrimSpace(nameEntry.Text)

		if configName != "" {
			// Build snapshot ID map
			snapshotIDs := make(map[string]string)
			for url, snap := range s.snapshotsByURL {
				if snap != nil {
					snapshotIDs[url] = snap.ID
				}
			}

			err := config.SaveMonitorConfig(configName, s.cfg.Email, s.cfg.Websites, snapshotIDs)
			if err != nil {
				dialog.ShowError(fmt.Errorf("failed to save configuration: %v", err), s.window)
				return
			}
			log.Printf("✅ Configuration '%s' saved successfully!\n", configName)
		}

		s.startMonitoring()
	})

	skipBtn := widget.NewButton("Start Without Saving", func() {
		s.startMonitoring()
	})

	content := container.NewVBox(
		title,
		widget.NewLabel(""),
		widget.NewLabel("Would you like to save this configuration for later use?"),
		widget.NewLabel(""),
		widget.NewLabel("Configuration Name:"),
		nameEntry,
		widget.NewLabel(""),
		saveBtn,
		skipBtn,
	)

	s.window.SetContent(content)
}

