package monitor

import (
	"apiwatcher/internal/alert"
	"apiwatcher/internal/email"
	"apiwatcher/internal/snapshot"
	"context"
	"fmt"
	"time"
)

// Logger interface for dependency injection
type Logger interface {
	Logf(format string, args ...interface{})
}

// ==========================
// Job Structures
// ==========================
type APIJob struct {
	Website string
	Email   string
}

type SnapshotJob struct {
	Website   string
	Email     string // Email address for sending alerts on errors
	Snapshots []*snapshot.Snapshot // Multiple snapshots per URL
}

// Legacy Job struct (kept for backwards compatibility during transition)
type Job struct {
	Website  string
	Email    string
	Snapshot *snapshot.Snapshot
}

// JobResult contains the result of processing a job
type JobResult struct {
	Success     bool
	Duration    time.Duration
	AlertSent   bool
	ErrorCount  int
	SnapshotRan bool
	Error       error
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

	// Handle failed requests
	if len(badRequests) > 0 {
		result.Success = false
		result.ErrorCount = len(badRequests)

		body := "The following API calls failed:\n\n"
		for _, r := range badRequests {
			body += fmt.Sprintf("%d %s\n", r.StatusCode, r.URL)
		}

		result.AlertSent = sendErrorAlert(job.Website, job.Email, "⚠️ API Errors Detected", body, alertLog, logger)
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

// ==========================
// Two-Phase Processing
// ==========================

// ProcessAPIJob handles API checking for a website (Phase 1)
func ProcessAPIJob(ctx context.Context, id int, job APIJob, logger Logger) JobResult {
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

	// Handle failed requests
	if len(badRequests) > 0 {
		result.Success = false
		result.ErrorCount = len(badRequests)

		body := "The following API calls failed:\n\n"
		for _, r := range badRequests {
			body += fmt.Sprintf("%d %s\n", r.StatusCode, r.URL)
		}

		result.AlertSent = sendErrorAlert(job.Website, job.Email, "⚠️ API Errors Detected", body, alertLog, logger)
	} else {
		logger.Logf("[OK] No API errors detected for %s", job.Website)
	}

	return result
}

// ProcessSnapshots handles all snapshot replays for a website sequentially (Phase 2)
// This is called AFTER all API checks are complete
func ProcessSnapshots(job SnapshotJob, logger Logger) {
	if len(job.Snapshots) == 0 {
		return
	}

	logger.Logf("[SNAPSHOTS] Processing %d snapshot(s) for %s", len(job.Snapshots), job.Website)

	for _, snap := range job.Snapshots {
		if snap == nil {
			continue
		}

		logger.Logf("[SNAPSHOT] 🎬 Starting replay for %s (ID: %s, Actions: %d)",
			job.Website, snap.ID, len(snap.Actions))

		// Use ReplayWithResult to get detailed error information
		result, err := snapshot.ReplayWithResult(snap)
		if err != nil {
			logger.Logf("[SNAPSHOT] ❌ Replay FAILED after %v for %s (ID: %s): %v",
				result.Duration, job.Website, snap.ID, err)
		} else if len(result.APIErrors) > 0 {
			// Snapshot completed but with API errors detected
			logger.Logf("[SNAPSHOT] ⚠️  Replay completed with %d API errors for %s (ID: %s)",
				len(result.APIErrors), job.Website, snap.ID)

			// Send alert email for snapshot API errors
			if job.Email != "" {
				alertLog, _ := alert.LoadLog()
				body := fmt.Sprintf(`Snapshot Replay Error Alert

Snapshot: %s
Website: %s

API Errors Detected: %d

Failed API Calls:
`, snap.ID, job.Website, len(result.APIErrors))

				for _, apiErr := range result.APIErrors {
					body += fmt.Sprintf("  %d %s\n", apiErr.StatusCode, apiErr.URL)
				}

				subject := fmt.Sprintf("⚠️ Snapshot Replay - API Errors Detected for %s", job.Website)
				sendErrorAlert("snapshot_"+snap.ID, job.Email, subject, body, alertLog, logger)
			}
		} else {
			// Successful replay with no API errors
			logger.Logf("[SNAPSHOT] ✅ Replay COMPLETED in %v for %s (ID: %s)",
				result.Duration, job.Website, snap.ID)
		}
	}

	logger.Logf("[SNAPSHOTS] All snapshots completed for %s", job.Website)
}

// sendErrorAlert sends an email alert for API errors with throttling to prevent email floods
// alertKey is used to track when the last alert was sent (can be website name or "snapshot_" + snapshotID)
func sendErrorAlert(alertKey string, recipientEmail string, subject string, body string, alertLog alert.Log, logger Logger) bool {
	now := time.Now().Unix()
	fiveHours := int64(5 * 3600) // 5 hours in seconds

	// Check if we've sent an alert for this key recently
	if lastAlert, exists := alertLog[alertKey]; exists && now-lastAlert < fiveHours {
		logger.Logf("[INFO] Skipping email for %s (sent recently)", alertKey)
		return false
	}

	// Send email
	if err := email.Send(recipientEmail, subject, body); err != nil {
		logger.Logf("[ERROR] Failed to send email: %v", err)
		return false
	}

	// Update alert log
	alertLog[alertKey] = now
	if err := alert.SaveLog(alertLog); err != nil {
		logger.Logf("[ERROR] Failed to save alert log: %v", err)
		return false
	}

	logger.Logf("[ALERT] Email sent successfully for %s to %s", alertKey, recipientEmail)
	return true
}
