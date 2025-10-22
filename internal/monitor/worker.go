package monitor

import (
	"context"
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

// JobResult contains the result of processing a job
type JobResult struct {
	Success      bool
	Duration     time.Duration
	AlertSent    bool
	ErrorCount   int
	SnapshotRan  bool
	Error        error
}

// ==========================
// Worker & Job Processing
// ==========================
func Worker(id int, jobs <-chan Job, logger Logger) {
	for job := range jobs {
		ProcessJob(nil, id, job, logger)
	}
}

// ProcessJob handles a single monitoring job with optional context for cancellation
func ProcessJob(ctx context.Context, id int, job Job, logger Logger) JobResult {
	result := JobResult{
		Success:    true,
		AlertSent:  false,
		ErrorCount: 0,
	}

	// Check if context is cancelled before starting
	if ctx != nil {
		select {
		case <-ctx.Done():
			logger.Logf("[WORKER %d] Aborted before starting %s", id, job.Website)
			result.Success = false
			result.Error = ctx.Err()
			return result
		default:
		}
	}
	startTime := time.Now()
	logger.Logf("[WORKER %d] ⏱️  START checking %s", id, job.Website)

	// Check the website with context
	badRequests, err := CheckWebsite(ctx, job.Website)
	result.Duration = time.Since(startTime)

	if err != nil {
		logger.Logf("[WORKER %d] ❌ ERROR after %v: %v", id, result.Duration, err)
		result.Success = false
		result.Error = err
		return result
	}

	logger.Logf("[WORKER %d] 🔍 Scan completed in %v for %s", id, result.Duration, job.Website)

	// Load alert log
	alertLog, _ := alert.LoadLog()
	now := time.Now().Unix()
	fiveHours := int64(5 * 3600) // 5 hours in seconds

	// Handle failed requests
	if len(badRequests) > 0 {
		result.Success = false
		result.ErrorCount = len(badRequests)
		
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
				result.AlertSent = true
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
		snapshotStartTime := time.Now()
		logger.Logf("[WORKER %d] 🎬 Starting snapshot replay for %s (Snapshot ID: %s, Actions: %d)",
			id, job.Website, job.Snapshot.ID, len(job.Snapshot.Actions))

		if err := snapshot.Replay(job.Snapshot); err != nil {
			snapshotDuration := time.Since(snapshotStartTime)
			logger.Logf("[WORKER %d] ❌ Snapshot replay FAILED after %v for %s (ID: %s): %v",
				id, snapshotDuration, job.Website, job.Snapshot.ID, err)
		} else {
			snapshotDuration := time.Since(snapshotStartTime)
			logger.Logf("[WORKER %d] ✅ Snapshot replay COMPLETED in %v for %s (ID: %s)",
				id, snapshotDuration, job.Website, job.Snapshot.ID)
			result.SnapshotRan = true
		}
	}

	return result
}
