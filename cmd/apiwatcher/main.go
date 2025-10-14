package main

import (
	"fmt"
	"log"
	"strings"
	"time"
	"url-checker/internal/config"
	"url-checker/internal/monitor"
	"url-checker/internal/snapshot"
	"bufio"
	"os"
)

 

func main() {
	var cfg *config.Config
	var snapshotsByURL map[string]*snapshot.Snapshot
	var loadedFromSaved bool

	// Check for saved monitor configurations first
	fmt.Println("Do you want to load a saved monitor configuration? (y/n)")
	var loadChoice string
	fmt.Print("> ")
	fmt.Scanln(&loadChoice)

	if strings.ToLower(loadChoice) == "y" {
		savedConfig, err := config.PromptSelectSavedConfig()
		if err != nil {
			fmt.Printf("Error loading saved configs: %v\n", err)
		}
		
		if savedConfig != nil {
			// Load the config from saved configuration
			cfg = &config.Config{
				Email:    savedConfig.Email,
				Websites: savedConfig.Websites,
			}
			
			fmt.Printf("Loaded configuration: %s\n", savedConfig.Name)
			fmt.Printf("📧 Email: %s\n🌐 Websites: %v\n", cfg.Email, cfg.Websites)
			
			// Load associated snapshots
			snapshotsByURL = make(map[string]*snapshot.Snapshot)
			for url, snapshotID := range savedConfig.SnapshotIDs {
				snap, err := snapshot.LoadByID(snapshotID)
				if err != nil {
					fmt.Printf("Warning: couldn't load snapshot %s for %s: %v\n", 
						snapshotID, url, err)
				} else {
					snapshotsByURL[url] = snap
					fmt.Printf("✅ Loaded snapshot '%s' for %s\n", snap.Name, url)
				}
			}
			
			// Ask if user wants to run this configuration
			fmt.Println("\nWould you like to run this configuration? (y/n)")
			var runChoice string
			fmt.Print("> ")
			fmt.Scanln(&runChoice)
			
			if strings.ToLower(runChoice) != "y" {
				fmt.Println("Exiting...")
				os.Exit(0)
			}
			
			loadedFromSaved = true
		}
	}

	// If no saved config was loaded, use the normal flow
	if cfg == nil {
		config.Load()
		cfg = config.PromptUser()

		// Prompt for snapshots if not loaded from saved config
		snapshotsByURL = snapshot.PromptSnapshotFlow(cfg)
	}

	// Start workers
	const numWorkers = 30
	jobQueue := make(chan monitor.Job, len(cfg.Websites))
	logger := &SimpleLogger{}

	for i := 1; i <= numWorkers; i++ {
		go monitor.Worker(i, jobQueue, logger)
	}

	fmt.Printf("[START] Monitoring %d websites every %d minutes. Alerts to %s\n", 
		len(cfg.Websites), config.WorkerSleepTime, cfg.Email)

	// Only ask to save if this is a new configuration (not loaded from saved)
	if !loadedFromSaved {
		fmt.Println("\nWould you like to save this configuration for later? (y/n)")
		var saveChoice string
		fmt.Print("> ")
		fmt.Scanln(&saveChoice)

		if strings.ToLower(saveChoice) == "y" {
			reader := bufio.NewReader(os.Stdin)
			fmt.Println("Enter a name for this configuration:")
			fmt.Print("> ")
			configName, _ := reader.ReadString('\n')
			configName = strings.TrimSpace(configName)
			
			if configName != "" {
				// Build snapshot ID map
				snapshotIDs := make(map[string]string)
				for url, snap := range snapshotsByURL {
					if snap != nil {
						snapshotIDs[url] = snap.ID
					}
				}
				
				err := config.SaveMonitorConfig(configName, cfg.Email, cfg.Websites, snapshotIDs)
				if err != nil {
					fmt.Printf("Failed to save configuration: %v\n", err)
				} else {
					fmt.Printf("✅ Configuration '%s' saved successfully!\n", configName)
				}
			}
		}
	}

	// Start monitoring loop
	for {
		for _, site := range cfg.Websites {
			jobQueue <- monitor.Job{
				Website:  site,
				Email:    cfg.Email,
				Snapshot: snapshotsByURL[site],
			}
		}

		time.Sleep(time.Duration(config.WorkerSleepTime) * time.Minute)
		log.Printf("Workers will resume in %d minutes", config.WorkerSleepTime)
	}
}

