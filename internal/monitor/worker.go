package monitor

import (
	"fmt"
	"log"
	"time"
	"url-checker/internal/alert"
	"url-checker/internal/email"
	"url-checker/internal/snapshot"
)

// ==========================
// Job Structure
// ==========================
type Job struct {
	Website string
	Email   string
	Snapshot *snapshot.Snapshot 
}

// ==========================
// Worker & Job Processing
// ==========================
func Worker(id int, jobs <-chan Job) {
	for job := range jobs {
		ProcessJob(id, job)
	}
}

// ProcessJob handles a single monitoring job - SHARED LOGIC for local and daemon
func ProcessJob(id int, job Job) error {
	startTime := time.Now()
	log.Printf("[WORKER %d] ⏱️  START checking %s", id, job.Website)
	
	// Check the website
	badRequests, err := CheckWebsite(job.Website)
	checkDuration := time.Since(startTime)
	
	if err != nil {
		log.Printf("[WORKER %d] ❌ ERROR after %v: %v", id, checkDuration, err)
		return err
	}
	
	log.Printf("[WORKER %d] 🔍 Scan completed in %v for %s", id, checkDuration, job.Website)

	// Load alert log
	alertLog, _ := alert.LoadLog()
	now := time.Now().Unix()
	fiveHours := int64(5 * 3600) // 5 hours in seconds

	// Handle failed requests
	if len(badRequests) > 0 {
		lastAlert, exists := alertLog[job.Website]

		body := "The following API calls failed:\n\n"
		for _, r := range badRequests {
			body += fmt.Sprintf("%d %s\n", r.StatusCode, r.URL)
		}

		if exists && now-lastAlert < fiveHours {
			log.Printf("[INFO] Skipping email for %s (sent recently)\n", job.Website)
		} else {
			if sendErr := email.Send(job.Email, "⚠️ API Errors Detected", body); sendErr != nil {
				log.Println("[ERROR] Failed to send email:", sendErr)
			} else {
				log.Println("[ALERT] Email sent successfully")
				alertLog[job.Website] = now
				if err := alert.SaveLog(alertLog); err != nil {
					log.Println("[ERROR] Failed to save alert log:", err)
				}
			}
		}
	} else {
		log.Println("[OK] No API errors detected for", job.Website)
	}
	
	// Run snapshot if configured
	if job.Snapshot != nil {
		log.Printf("[WORKER %d] Running snapshot for %s\n", id, job.Website)
		if err := snapshot.Replay(job.Snapshot); err != nil {
			log.Printf("[WORKER %d] Snapshot replay error for %s (%s): %v\n", id, job.Website, job.Snapshot.ID, err)
		} else {
			log.Printf("[WORKER %d] Snapshot replay finished for %s (%s)\n", id, job.Website, job.Snapshot.ID)
		}
	}
	
	return nil
}

