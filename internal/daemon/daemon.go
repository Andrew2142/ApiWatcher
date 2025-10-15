package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"url-checker/internal/config"
	"url-checker/internal/monitor"
	"url-checker/internal/snapshot"
)

// State represents the current state of the daemon
type State string

const (
	StateStopped State = "stopped"
	StateRunning State = "running"
	StatePaused  State = "paused"
	StateError   State = "error"
)

// Daemon represents the monitoring daemon
type Daemon struct {
	state             State
	config            *config.Config
	snapshotsByURL    map[string]*snapshot.Snapshot
	jobQueue          chan monitor.Job
	stopChan          chan bool
	mutex             sync.RWMutex
	logBuffer         *LogBuffer
	stats             *Stats
	websiteStats      *WebsiteStatsMap
	dataDir           string
	monitoringActive  bool
	jobWaitGroup      sync.WaitGroup
	monitoringStopped chan bool
	cancelCtx         context.CancelFunc
}

// Stats holds monitoring statistics
type Stats struct {
	StartedAt     time.Time
	TotalChecks   int
	FailedChecks  int
	AlertsSent    int
	LastCheckTime time.Time
	mutex         sync.RWMutex
}

// Public stats for reading outside
type StatsData struct {
	StartedAt     time.Time
	TotalChecks   int
	FailedChecks  int
	AlertsSent    int
	LastCheckTime time.Time
}

// WebsiteStats holds statistics for a single website
type WebsiteStats struct {
	URL                  string
	TotalChecks          int
	FailedChecks         int
	ConsecutiveFailures  int
	ConsecutiveSuccesses int
	EmailsSent           int
	LastCheckTime        time.Time
	LastFailureTime      time.Time
	LastSuccessTime      time.Time
	FirstMonitoredAt     time.Time

	// Performance metrics
	ResponseTimes       []time.Duration // Ring buffer of last 100 response times
	AverageResponseTime time.Duration

	// Uptime tracking
	UptimeLastHour       float64 // Percentage
	UptimeLast24Hours    float64 // Percentage
	UptimeLast7Days      float64 // Percentage
	OverallHealthPercent float64 // Total success rate

	// Downtime tracking
	LastDowntimeStart    time.Time
	LastDowntimeEnd      time.Time
	LastDowntimeDuration time.Duration
	LongestDowntime      time.Duration
	TotalDowntime        time.Duration

	// Alert tracking
	LastAlertSent time.Time

	// Health trend
	HealthTrend string // "improving", "stable", "degrading"

	// Track recent checks for time-window calculations
	CheckHistory []CheckRecord // Recent checks for uptime calculations

	mutex sync.RWMutex
}

// CheckRecord represents a single check result for uptime calculations
type CheckRecord struct {
	Timestamp time.Time
	Success   bool
	Duration  time.Duration
}

// WebsiteStatsMap manages statistics for all monitored websites
type WebsiteStatsMap struct {
	stats map[string]*WebsiteStats
	mutex sync.RWMutex
}

// DaemonState represents the persisted state
type DaemonState struct {
	State        State                    `json:"state"`
	Config       *config.Config           `json:"config"`
	SnapshotIDs  map[string]string        `json:"snapshot_ids"`
	Stats        *Stats                   `json:"stats"`
	WebsiteStats map[string]*WebsiteStats `json:"website_stats"`
	LastSaved    time.Time                `json:"last_saved"`
}

// New creates a new daemon instance
func New(dataDir string) (*Daemon, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	d := &Daemon{
		state:          StateStopped,
		snapshotsByURL: make(map[string]*snapshot.Snapshot),
		stopChan:       make(chan bool),
		logBuffer:      NewLogBuffer(1000),
		stats:          &Stats{},
		websiteStats: &WebsiteStatsMap{
			stats: make(map[string]*WebsiteStats),
		},
		dataDir: dataDir,
	}

	_ = d.loadState() // silently ignore load errors

	return d, nil
}

func (d *Daemon) GetState() State {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.state
}

func (d *Daemon) GetConfig() *config.Config {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.config
}

func (d *Daemon) GetStatsData() StatsData {
	d.stats.mutex.RLock()
	defer d.stats.mutex.RUnlock()

	return StatsData{
		StartedAt:     d.stats.StartedAt,
		TotalChecks:   d.stats.TotalChecks,
		FailedChecks:  d.stats.FailedChecks,
		AlertsSent:    d.stats.AlertsSent,
		LastCheckTime: d.stats.LastCheckTime,
	}
}

// GetOrCreateWebsiteStats gets or creates stats for a website
func (d *Daemon) GetOrCreateWebsiteStats(url string) *WebsiteStats {
	d.websiteStats.mutex.Lock()
	defer d.websiteStats.mutex.Unlock()

	stats, exists := d.websiteStats.stats[url]
	if !exists {
		stats = &WebsiteStats{
			URL:              url,
			FirstMonitoredAt: time.Now(),
			ResponseTimes:    make([]time.Duration, 0, 100),
			CheckHistory:     make([]CheckRecord, 0, 1000),
		}
		d.websiteStats.stats[url] = stats
	}
	return stats
}

// GetWebsiteStats gets stats for a specific website
func (d *Daemon) GetWebsiteStats(url string) *WebsiteStats {
	d.websiteStats.mutex.RLock()
	defer d.websiteStats.mutex.RUnlock()
	return d.websiteStats.stats[url]
}

// GetAllWebsiteStats returns a copy of all website stats
func (d *Daemon) GetAllWebsiteStats() map[string]*WebsiteStats {
	d.websiteStats.mutex.RLock()
	defer d.websiteStats.mutex.RUnlock()

	result := make(map[string]*WebsiteStats, len(d.websiteStats.stats))
	for url, stats := range d.websiteStats.stats {
		// Return a copy to avoid race conditions
		stats.mutex.RLock()
		statsCopy := *stats
		stats.mutex.RUnlock()
		result[url] = &statsCopy
	}
	return result
}

func (d *Daemon) SetConfig(cfg *config.Config, snapshots map[string]*snapshot.Snapshot) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.state == StateRunning || d.state == StatePaused {
		return fmt.Errorf("cannot change configuration while monitoring is active")
	}

	d.config = cfg
	d.snapshotsByURL = snapshots

	_ = d.saveState()
	return nil
}

func (d *Daemon) GetLogs(n int) []string {
	return d.logBuffer.GetLast(n)
}

func (d *Daemon) ClearLogs() {
	d.logBuffer.Clear()
}

func (d *Daemon) saveState() error {
	statePath := filepath.Join(d.dataDir, "daemon-state.json")

	snapshotIDs := make(map[string]string)
	for url, snap := range d.snapshotsByURL {
		if snap != nil {
			snapshotIDs[url] = snap.ID
		}
	}

	// Get a copy of website stats for persistence
	websiteStats := d.GetAllWebsiteStats()

	state := DaemonState{
		State:        d.state,
		Config:       d.config,
		SnapshotIDs:  snapshotIDs,
		Stats:        d.stats,
		WebsiteStats: websiteStats,
		LastSaved:    time.Now(),
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	return os.WriteFile(statePath, data, 0644)
}

func (d *Daemon) loadState() error {
	statePath := filepath.Join(d.dataDir, "daemon-state.json")
	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read state file: %w", err)
	}

	var state DaemonState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}

	d.state = state.State
	d.config = state.Config
	d.stats = state.Stats

	// Restore website stats
	if state.WebsiteStats != nil {
		d.websiteStats.mutex.Lock()
		d.websiteStats.stats = state.WebsiteStats
		// Ensure URL field is set from map key
		for url, stats := range d.websiteStats.stats {
			if stats.URL == "" {
				stats.URL = url
			}
		}
		d.websiteStats.mutex.Unlock()
	}

	if state.SnapshotIDs != nil {
		for url, snapshotID := range state.SnapshotIDs {
			snap, _ := snapshot.LoadByID(snapshotID)
			d.snapshotsByURL[url] = snap
		}
	}

	if d.state == StateRunning || d.state == StatePaused {
		d.state = StateStopped
	}

	return nil
}

func (d *Daemon) Logf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log.Println(msg)
	d.logBuffer.Add(msg)
}

func (d *Daemon) Start() error {
	d.mutex.Lock()
	
	if d.state == StateRunning {
		d.mutex.Unlock()
		return fmt.Errorf("monitoring is already running")
	}

	if d.config == nil || len(d.config.Websites) == 0 {
		d.mutex.Unlock()
		return fmt.Errorf("no configuration loaded")
	}

	// If there's a previous monitoring session still cleaning up, wait for it
	if d.monitoringStopped != nil {
		d.Logf("Waiting for previous monitoring session to finish...")
		stoppedChan := d.monitoringStopped
		d.mutex.Unlock()
		
		// Wait with timeout
		select {
		case <-stoppedChan:
			d.Logf("Previous session finished")
		case <-time.After(10 * time.Second):
			d.Logf("Timeout waiting for previous session - proceeding anyway")
		}
		
		d.mutex.Lock()
	}

	// Create fresh channels and context for this monitoring session
	d.stopChan = make(chan bool)
	d.monitoringStopped = make(chan bool)
	
	// Create cancellable context for instant worker abort
	ctx, cancel := context.WithCancel(context.Background())
	d.cancelCtx = cancel

	d.state = StateRunning
	d.monitoringActive = true
	d.stats.StartedAt = time.Now()

	d.mutex.Unlock()
	
	_ = d.saveState()
	go d.runMonitoring(ctx)
	return nil
}

func (d *Daemon) Stop() error {
	d.mutex.Lock()
	
	if d.state != StateRunning && d.state != StatePaused {
		d.mutex.Unlock()
		return fmt.Errorf("monitoring is not running")
	}

	d.state = StateStopped
	d.monitoringActive = false

	// Cancel context to abort all workers instantly
	if d.cancelCtx != nil {
		d.cancelCtx()
	}

	// Close stop channel to signal monitoring loop
	select {
	case <-d.stopChan:
	default:
		close(d.stopChan)
	}
	
	d.mutex.Unlock()

	// Return immediately - let monitoring clean up in background
	d.Logf("Stop signal sent - workers aborting instantly")
	
	// Save state asynchronously
	go d.saveState()
	
	return nil
}

func (d *Daemon) Pause() error {
	d.monitoringActive = false
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.state != StateRunning {
		return fmt.Errorf("monitoring is not running")
	}

	d.state = StatePaused
	_ = d.saveState()
	return nil
}

func (d *Daemon) Resume() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.state != StatePaused {
		return fmt.Errorf("monitoring is not paused")
	}

	d.state = StateRunning
	d.monitoringActive = true
	
	// Create context for this session
	ctx, cancel := context.WithCancel(context.Background())
	d.cancelCtx = cancel
	d.stopChan = make(chan bool)

	_ = d.saveState()
	go d.runMonitoring(ctx)
	return nil
}

func (d *Daemon) runMonitoring(ctx context.Context) {
	defer func() {
		// Safely signal that monitoring has stopped
		d.mutex.Lock()
		if d.monitoringStopped != nil {
			close(d.monitoringStopped)
			d.monitoringStopped = nil
		}
		d.mutex.Unlock()
	}()
	
	const numWorkers = 30
	d.jobQueue = make(chan monitor.Job, 100)

	// Start workers with context
	for i := 1; i <= numWorkers; i++ {
		go d.worker(ctx, i)
	}

	for {
		select {
		case <-d.stopChan:
			d.Logf("Stop signal received, shutting down monitoring loop")
			close(d.jobQueue)
			// Don't wait for workers - context cancellation aborts them instantly
			return
		case <-ctx.Done():
			d.Logf("Context cancelled, shutting down monitoring loop")
			close(d.jobQueue)
			return
		default:
		}

		jobCount := len(d.config.Websites)
		d.Logf("Queueing %d jobs", jobCount)
		d.jobWaitGroup.Add(jobCount)

		for _, site := range d.config.Websites {
			job := monitor.Job{
				Website:  site,
				Email:    d.config.Email,
				Snapshot: d.snapshotsByURL[site],
			}

			select {
			case d.jobQueue <- job:
			case <-d.stopChan:
				d.jobWaitGroup.Done()
				break
			}
		}

		d.jobWaitGroup.Wait()

		d.stats.mutex.Lock()
		d.stats.LastCheckTime = time.Now()
		d.stats.mutex.Unlock()

		sleepTime := time.Duration(config.WorkerSleepTime) * time.Minute
		d.Logf("Sleeping for %v", sleepTime)

		select {
		case <-time.After(sleepTime):
		case <-d.stopChan:
			d.Logf("Stop signal received, shutting down monitoring loop")
			close(d.jobQueue)
			// Don't wait for workers - context cancellation aborts them instantly
			return
		case <-ctx.Done():
			d.Logf("Context cancelled, shutting down monitoring loop")
			close(d.jobQueue)
			return
		}
	}
}

func (d *Daemon) worker(ctx context.Context, id int) {
	for job := range d.jobQueue {
		// Check if context is cancelled (instant abort)
		select {
		case <-ctx.Done():
			d.Logf("[Worker %d] Context cancelled, aborting", id)
			d.jobWaitGroup.Done()
			return
		default:
		}

		// Also check stop signal
		select {
		case <-d.stopChan:
			d.Logf("[Worker %d] Stop signal received, exiting", id)
			d.jobWaitGroup.Done()
			return
		default:
		}

	// Check monitoringActive flag
	if !d.monitoringActive {
		d.Logf("[Worker %d] Monitoring inactive, skipping job", id)
		d.jobWaitGroup.Done()
		continue
	}

	d.stats.mutex.Lock()
	d.stats.TotalChecks++
	d.stats.mutex.Unlock()

	// Pass context to ProcessJob so it can abort mid-operation
	result := monitor.ProcessJob(ctx, id, job, d)

	// Update global stats
	if !result.Success {
		d.stats.mutex.Lock()
		d.stats.FailedChecks++
		d.stats.mutex.Unlock()
	}

	// Update per-website stats
	d.UpdateWebsiteStats(job.Website, result.Success, result.Duration, result.AlertSent)

	d.jobWaitGroup.Done()
	}

	d.Logf("[Worker %d] Job queue closed, exiting", id)
}
