package main

import (
	"fmt"
	"log"
	"strings"
	"time"
	"url-checker/internal/config"
	"url-checker/internal/monitor"
	"url-checker/internal/snapshot"
)

var ShowWorkerBrowser = true

func main() {
	var cfg *config.Config
	var err error

	// Load config	
	cfg, err = config.Load()
	if err != nil {
		fmt.Println("No saved configuration found. Let's set it up!")
		cfg = config.PromptUser()
	} else {
		fmt.Printf("Loaded saved configuration:\n📧 Email: %s\n🌐 Websites: %v\n", cfg.Email, cfg.Websites)
		fmt.Println("Would you like to use this configuration? (y/n)")
		var choice string
		fmt.Print("> ")
		fmt.Scanln(&choice)

		if strings.ToLower(choice) != "y" {
			cfg = config.PromptUser()
		}
	}

	//Snapshot prep
	fmt.Println("Would you like to see the snapshot replay in Chrome? (y/n)")
	var seeBrowser string
	fmt.Print("> ")
	fmt.Scanln(&seeBrowser)
	ShowWorkerBrowser = strings.ToLower(seeBrowser) == "y"
	snapshot.PromptSnapshotFlow(cfg)

	//Load snapshots
	allSnapshots, _ := snapshot.LoadAll()
	snapshotsByURL := map[string]*snapshot.Snapshot{}
	for _, s := range allSnapshots {
		snapshotsByURL[s.URL] = s
	}


	//Start workers
	const numWorkers = 30
	jobQueue := make(chan monitor.Job, len(cfg.Websites))

	for i := 1; i <= numWorkers; i++ {
		go monitor.Worker(i, jobQueue)
	}

	fmt.Printf("[START] Monitoring %d websites every %d minutes. Alerts to %s\n", len(cfg.Websites), config.WorkerSleepTime, cfg.Email)


	//Monitor
	for {
		for _, site := range cfg.Websites {
			jobQueue <- monitor.Job{
				Website:   site,
				Email:     cfg.Email,
				Snapshot: snapshotsByURL[site], 		
			}
		}

		time.Sleep(time.Duration(config.WorkerSleepTime) * time.Minute)
		log.Printf("Workers will resume in %d minutes", config.WorkerSleepTime)
	}
}

