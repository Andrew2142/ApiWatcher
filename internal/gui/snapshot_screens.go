package gui

import (
	"fmt"
	"log"
	"strings"
	"url-checker/internal/snapshot"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// Step 4: Snapshot configuration
func (s *AppState) showSnapshotConfigScreen() {
	title := widget.NewLabel("Snapshot Configuration")
	title.TextStyle.Bold = true

	useSnapshots := false
	checkbox := widget.NewCheck("Use snapshot run mode for some sites", func(checked bool) {
		useSnapshots = checked
	})

	nextBtn := widget.NewButton("Next", func() {
		if useSnapshots {
			s.showSnapshotSiteSelectionScreen()
		} else {
			s.showSaveConfigScreen()
		}
	})

	content := container.NewVBox(
		title,
		widget.NewLabel(""),
		widget.NewLabel("Snapshot mode allows you to record and replay user interactions"),
		widget.NewLabel("with your websites to test specific flows."),
		widget.NewLabel(""),
		checkbox,
		widget.NewLabel(""),
		nextBtn,
	)

	s.window.SetContent(content)
}

// Step 5: Select sites for snapshot
func (s *AppState) showSnapshotSiteSelectionScreen() {
	title := widget.NewLabel("Select Sites for Snapshots")
	title.TextStyle.Bold = true

	var checks []*widget.Check
	selectedSites := make(map[int]bool)

	for i, site := range s.cfg.Websites {
		idx := i
		check := widget.NewCheck(site, func(checked bool) {
			selectedSites[idx] = checked
		})
		checks = append(checks, check)
	}

	nextBtn := widget.NewButton("Configure Snapshots", func() {
		var sitesToConfigure []string
		for idx, checked := range selectedSites {
			if checked {
				sitesToConfigure = append(sitesToConfigure, s.cfg.Websites[idx])
			}
		}

		if len(sitesToConfigure) == 0 {
			s.showSaveConfigScreen()
			return
		}

		s.showSnapshotForSite(sitesToConfigure, 0)
	})

	skipBtn := widget.NewButton("Skip", func() {
		s.showSaveConfigScreen()
	})

	checksContainer := container.NewVBox()
	for _, check := range checks {
		checksContainer.Add(check)
	}

	// replace NewMax with NewStack
	minRadio := NewMinSized(checksContainer, fyne.NewSize(300, 400))
	scroll := container.NewScroll(minRadio)
	scroll.SetMinSize(fyne.NewSize(300, 400))

	content := container.NewVBox(
		title,
		widget.NewLabel(""),
		widget.NewLabel("Select the sites you want to configure snapshots for:"),
		widget.NewLabel(""),
		scroll,
		widget.NewLabel(""),
		nextBtn,
		skipBtn,
	)

	s.window.SetContent(content)
}

// Step 6: Configure snapshot for each site
func (s *AppState) showSnapshotForSite(sites []string, currentIndex int) {
	if currentIndex >= len(sites) {
		s.showSaveConfigScreen()
		return
	}

	url := sites[currentIndex]

	title := widget.NewLabel(fmt.Sprintf("Snapshot for %s", url))
	title.TextStyle.Bold = true

	// Check for existing snapshots
	existingSnapshots, _ := snapshot.LoadForURL(url)

	var selectedSnapshot *snapshot.Snapshot

	if len(existingSnapshots) > 0 {
		// Show existing snapshots
		var snapshotNames []string
		for _, snap := range existingSnapshots {
			displayName := snap.Name
			if displayName == "" {
				displayName = "(unnamed)"
			}
			info := fmt.Sprintf("%s - Created: %s", displayName, snap.CreatedAt.Format("2006-01-02 15:04:05"))
			snapshotNames = append(snapshotNames, info)
		}

		selectedIndex := 0
		radio := widget.NewRadioGroup(snapshotNames, func(value string) {
			for i, name := range snapshotNames {
				if name == value {
					selectedIndex = i
					break
				}
			}
		})
		radio.SetSelected(snapshotNames[0])

		useExistingBtn := widget.NewButton("Use Selected Snapshot", func() {
			selectedSnapshot = existingSnapshots[selectedIndex]
			s.snapshotsByURL[url] = selectedSnapshot
			log.Printf("Using existing snapshot '%s' for %s\n", selectedSnapshot.Name, url)

			// Move to next site
			s.showSnapshotForSite(sites, currentIndex+1)
		})

		createNewBtn := widget.NewButton("Create New Snapshot", func() {
			s.showCreateSnapshotScreen(url, sites, currentIndex)
		})

		skipBtn := widget.NewButton("Skip This Site", func() {
			s.showSnapshotForSite(sites, currentIndex+1)
		})

		// replace NewMax with NewStack
		minRadio := NewMinSized(radio, fyne.NewSize(300, 400))
		scroll := container.NewScroll(minRadio)
		scroll.SetMinSize(fyne.NewSize(300, 400))

		content := container.NewVBox(
			title,
			widget.NewLabel(""),
			widget.NewLabel(fmt.Sprintf("Found %d existing snapshots:", len(existingSnapshots))),
			widget.NewLabel(""),
			scroll,
			widget.NewLabel(""),
			useExistingBtn,
			createNewBtn,
			skipBtn,
		)

		s.window.SetContent(content)
	} else {
		// No existing snapshots, create new
		s.showCreateSnapshotScreen(url, sites, currentIndex)
	}
}

// Step 7: Create new snapshot
func (s *AppState) showCreateSnapshotScreen(url string, sites []string, currentIndex int) {
	title := widget.NewLabel(fmt.Sprintf("Create Snapshot for %s", url))
	title.TextStyle.Bold = true

	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Snapshot name (optional)")

	recordBtn := widget.NewButton("Record Snapshot", func() {
		snapshotName := strings.TrimSpace(nameEntry.Text)

		// Create channel for stopping the recording
		stopChan := make(chan bool)

		// Start recording in background
		go func() {
			snap, err := snapshot.RecordWithCallback(url, snapshotName, stopChan)
			if err != nil {
				log.Printf("Recording failed for %s: %v\n", url, err)
				return
			}

			// Ask to save
			err = snapshot.SaveToDisk(snap)
			if err != nil {
				log.Printf("Failed to save snapshot: %v\n", err)
			} else {
				s.snapshotsByURL[url] = snap
				log.Printf("Snapshot saved for %s\n", url)
			}

			// Move to next site
			fyne.Do(func() {
				s.showSnapshotForSite(sites, currentIndex+1)
			})
		}()

		// Show dialog that allows user to end recording
		confirmDialog := dialog.NewConfirm(
			"Recording",
			"Browser will open. Perform your actions, then click OK to finish recording.",
			func(confirmed bool) {
				if confirmed {
					stopChan <- false // false = not cancelled
				} else {
					stopChan <- true // true = cancelled
				}
			},
			s.window,
		)
		confirmDialog.SetConfirmText("OK")
		confirmDialog.SetDismissText("Cancel")
		confirmDialog.Show()
	})

	skipBtn := widget.NewButton("Skip This Site", func() {
		s.showSnapshotForSite(sites, currentIndex+1)
	})

	content := container.NewVBox(
		title,
		widget.NewLabel(""),
		widget.NewLabel("Enter a name for this snapshot (optional):"),
		nameEntry,
		widget.NewLabel(""),
		widget.NewLabel("Click 'Record' to open a browser and record your interactions."),
		widget.NewLabel(""),
		recordBtn,
		skipBtn,
	)

	s.window.SetContent(content)
}

