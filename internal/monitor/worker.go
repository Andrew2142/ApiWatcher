package monitor

import (
	"fmt"
	"time"
	"url-checker/internal/alert"
	"url-checker/internal/email"
	"url-checker/internal/snapshot"
)

// Logger interface for dependency injection
type Logger interface {
	Logf(format string, args ...interface{})
}

// ==========================
// Job Structure
// ==========================
type Job struct {
	Website  string
	Email    string
	Snapshot *snapshot.Snapshot
}

// ==========================
// Worker & Job Processing
// ==========================
func Worker(id int, jobs <-chan Job, logger Logger) {
	for job := range jobs {
		ProcessJob(id, job, logger)
	}
}

// ProcessJob handles a single monitoring job - SHARED LOGIC for local and daemon
func ProcessJob(id int, job Job, logger Logger) error {
	startTime := time.Now()
	logger.Logf("[WORKER %d] ⏱️  START checking %s", id, job.Website)

	// Check the website
	badRequests, err := CheckWebsite(job.Website)
	checkDuration := time.Since(startTime)

	if err != nil {
		logger.Logf("[WORKER %d] ❌ ERROR after %v: %v", id, checkDuration, err)
		return err
	}

	logger.Logf("[WORKER %d] 🔍 Scan completed in %v for %s", id, checkDuration, job.Website)

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
			logger.Logf("[INFO] Skipping email for %s (sent recently)", job.Website)
		} else {
			if sendErr := email.Send(job.Email, "⚠️ API Errors Detected", body); sendErr != nil {
				logger.Logf("[ERROR] Failed to send email: %v", sendErr)
			} else {
				logger.Logf("[ALERT] Email sent successfully")
				alertLog[job.Website] = now
				if err := alert.SaveLog(alertLog); err != nil {
					logger.Logf("[ERROR] Failed to save alert log: %v", err)
				}
			}
		}
	} else {
		logger.Logf("[OK] No API errors detected for %s", job.Website)
	}

	// Run snapshot if configured
	if job.Snapshot != nil {
		logger.Logf("[WORKER %d] Running snapshot for %s", id, job.Website)
		if err := snapshot.Replay(job.Snapshot); err != nil {
			logger.Logf("[WORKER %d] Snapshot replay error for %s (%s): %v", id, job.Website, job.Snapshot.ID, err)
		} else {
			logger.Logf("[WORKER %d] Snapshot replay finished for %s (%s)", id, job.Website, job.Snapshot.ID)
		}
	}

	return nil
}
