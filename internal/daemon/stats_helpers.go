package daemon

import (
	"time"
)

// UpdateWebsiteStats updates statistics after a check
func (d *Daemon) UpdateWebsiteStats(url string, success bool, duration time.Duration, alertSent bool) {
	stats := d.GetOrCreateWebsiteStats(url)
	stats.mutex.Lock()
	defer stats.mutex.Unlock()

	// Update basic counters
	stats.TotalChecks++
	stats.LastCheckTime = time.Now()

	// Record check in history
	checkRecord := CheckRecord{
		Timestamp: time.Now(),
		Success:   success,
		Duration:  duration,
	}
	stats.CheckHistory = append(stats.CheckHistory, checkRecord)

	// Keep only last 7 days of history (assuming ~5 min intervals = ~2000 checks)
	if len(stats.CheckHistory) > 2000 {
		stats.CheckHistory = stats.CheckHistory[len(stats.CheckHistory)-2000:]
	}

	// Update response times ring buffer
	if duration > 0 {
		stats.ResponseTimes = append(stats.ResponseTimes, duration)
		if len(stats.ResponseTimes) > 100 {
			stats.ResponseTimes = stats.ResponseTimes[1:]
		}
		stats.AverageResponseTime = calculateAverageResponseTime(stats.ResponseTimes)
	}

	if success {
		stats.ConsecutiveSuccesses++
		stats.ConsecutiveFailures = 0
		stats.LastSuccessTime = time.Now()

		// If this is recovery from downtime, record it
		if !stats.LastDowntimeStart.IsZero() && stats.LastDowntimeEnd.IsZero() {
			stats.LastDowntimeEnd = time.Now()
			stats.LastDowntimeDuration = stats.LastDowntimeEnd.Sub(stats.LastDowntimeStart)
			stats.TotalDowntime += stats.LastDowntimeDuration

			if stats.LastDowntimeDuration > stats.LongestDowntime {
				stats.LongestDowntime = stats.LastDowntimeDuration
			}
		}
	} else {
		stats.FailedChecks++
		stats.ConsecutiveFailures++
		stats.ConsecutiveSuccesses = 0
		stats.LastFailureTime = time.Now()

		// Start tracking downtime if this is the first failure
		if stats.ConsecutiveFailures == 1 {
			stats.LastDowntimeStart = time.Now()
			stats.LastDowntimeEnd = time.Time{} // Reset end time
		}

		if alertSent {
			stats.EmailsSent++
			stats.LastAlertSent = time.Now()
		}
	}

	// Calculate health percentages
	stats.OverallHealthPercent = calculateHealthPercentage(stats.TotalChecks, stats.FailedChecks)
	stats.UptimeLastHour = calculateUptimePercentage(stats.CheckHistory, 1*time.Hour)
	stats.UptimeLast24Hours = calculateUptimePercentage(stats.CheckHistory, 24*time.Hour)
	stats.UptimeLast7Days = calculateUptimePercentage(stats.CheckHistory, 7*24*time.Hour)

	// Calculate health trend
	stats.HealthTrend = calculateHealthTrend(stats.CheckHistory)
}

// calculateHealthPercentage calculates overall health percentage
func calculateHealthPercentage(totalChecks, failedChecks int) float64 {
	if totalChecks == 0 {
		return 100.0
	}
	successRate := float64(totalChecks-failedChecks) / float64(totalChecks) * 100.0
	return successRate
}

// calculateUptimePercentage calculates uptime for a specific time window
func calculateUptimePercentage(history []CheckRecord, duration time.Duration) float64 {
	if len(history) == 0 {
		return 100.0
	}

	cutoffTime := time.Now().Add(-duration)
	var totalChecks, successfulChecks int

	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Timestamp.Before(cutoffTime) {
			break
		}
		totalChecks++
		if history[i].Success {
			successfulChecks++
		}
	}

	if totalChecks == 0 {
		return 100.0
	}

	return float64(successfulChecks) / float64(totalChecks) * 100.0
}

// calculateHealthTrend determines if health is improving, stable, or degrading
func calculateHealthTrend(history []CheckRecord) string {
	if len(history) < 10 {
		return "stable"
	}

	// Look at last 20 checks, compare first 10 vs last 10
	recentCount := 20
	if len(history) < recentCount {
		recentCount = len(history)
	}

	recentHistory := history[len(history)-recentCount:]
	midpoint := len(recentHistory) / 2

	firstHalf := recentHistory[:midpoint]
	secondHalf := recentHistory[midpoint:]

	firstSuccess := 0
	for _, check := range firstHalf {
		if check.Success {
			firstSuccess++
		}
	}

	secondSuccess := 0
	for _, check := range secondHalf {
		if check.Success {
			secondSuccess++
		}
	}

	firstRate := float64(firstSuccess) / float64(len(firstHalf))
	secondRate := float64(secondSuccess) / float64(len(secondHalf))

	diff := secondRate - firstRate

	if diff > 0.1 {
		return "improving"
	} else if diff < -0.1 {
		return "degrading"
	}
	return "stable"
}

// calculateAverageResponseTime calculates average from slice of durations
func calculateAverageResponseTime(times []time.Duration) time.Duration {
	if len(times) == 0 {
		return 0
	}

	var total time.Duration
	for _, t := range times {
		total += t
	}

	return total / time.Duration(len(times))
}

// GetCurrentStatus returns the current status string based on consecutive failures
func (ws *WebsiteStats) GetCurrentStatus() string {
	ws.mutex.RLock()
	defer ws.mutex.RUnlock()

	if ws.ConsecutiveFailures > 0 {
		return "Down"
	} else if ws.ConsecutiveSuccesses > 0 {
		return "Up"
	}
	return "Unknown"
}
