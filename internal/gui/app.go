package gui

import (
	"log"
	"url-checker/internal/config"
	"url-checker/internal/daemon"
	"url-checker/internal/monitor"
	"url-checker/internal/remote"
	"url-checker/internal/snapshot"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/widget"
)

type AppState struct {
	app              fyne.App
	window           fyne.Window
	cfg              *config.Config
	snapshotsByURL   map[string]*snapshot.Snapshot
	loadedFromSaved  bool
	monitoringActive bool
	jobQueue         chan monitor.Job
	statusLabel      *widget.Label
	sshConn          *remote.SSHConnection
	daemonClient     *daemon.Client
	localTunnelPort  int
}

func Run() {
	log.Println("Initializing Fyne application...")
	myApp := app.New()

	log.Println("Creating window...")
	myWindow := myApp.NewWindow("API Watcher")
	myWindow.Resize(fyne.NewSize(700, 500))

	log.Println("Setting up application state...")
	state := &AppState{
		app:            myApp,
		window:         myWindow,
		snapshotsByURL: make(map[string]*snapshot.Snapshot),
		statusLabel:    widget.NewLabel("Ready"),
	}

	log.Println("Showing initial screen...")
	// Show initial screen - SSH connection
	state.showSSHConnectionScreen()

	log.Println("Starting GUI event loop...")
	myWindow.ShowAndRun()
	log.Println("GUI closed.")
}
