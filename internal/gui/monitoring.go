package gui

import (
	"fmt"
	"log"
	"time"
	"url-checker/internal/config"
	"url-checker/internal/monitor"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// Step 9: Start monitoring
func (s *AppState) startMonitoring() {
	title := widget.NewLabel("Monitoring Active")
	title.TextStyle.Bold = true

	infoText := fmt.Sprintf("Monitoring %d websites\nAlert email: %s\nCheck interval: %d minutes",
		len(s.cfg.Websites),
		s.cfg.Email,
		config.WorkerSleepTime)

	s.statusLabel.SetText("Monitoring...")

	logArea := widget.NewLabel("Starting workers...")
	logArea.Wrapping = fyne.TextWrapWord

	stopBtn := widget.NewButton("Stop Monitoring", func() {
		s.monitoringActive = false
		dialog.ShowInformation("Stopped", "Monitoring stopped.", s.window)
		s.showLoadConfigScreen()
	})

	content := container.NewVBox(
		title,
		widget.NewLabel(""),
		widget.NewLabel(infoText),
		widget.NewLabel(""),
		s.statusLabel,
		widget.NewLabel(""),
		container.NewScroll(logArea),
		widget.NewLabel(""),
		stopBtn,
	)

	s.window.SetContent(content)

	// Start monitoring
	s.monitoringActive = true
	const numWorkers = 30
	s.jobQueue = make(chan monitor.Job, len(s.cfg.Websites))

	for i := 1; i <= numWorkers; i++ {
		go monitor.Worker(i, s.jobQueue)
	}

	log.Printf("[START] Monitoring %d websites every %d minutes. Alerts to %s\n",
		len(s.cfg.Websites), config.WorkerSleepTime, s.cfg.Email)

	// Monitoring loop
	go func() {
		for s.monitoringActive {
			for _, site := range s.cfg.Websites {
				if !s.monitoringActive {
					break
				}
				s.jobQueue <- monitor.Job{
					Website:  site,
					Email:    s.cfg.Email,
					Snapshot: s.snapshotsByURL[site],
				}
			}

			time.Sleep(time.Duration(config.WorkerSleepTime) * time.Minute)
			log.Printf("Workers will resume in %d minutes", config.WorkerSleepTime)
			logArea := widget.NewLabel("Workers will resume in")
			logArea.Wrapping = fyne.TextWrapWord
			s.statusLabel.SetText(fmt.Sprintf("Last check: %s | Next in %d min",
				time.Now().Format("15:04:05"), config.WorkerSleepTime))
		}
	}()
}

