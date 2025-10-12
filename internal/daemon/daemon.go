package daemon

import (
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
	StateStopped    State = "stopped"
	StateRunning    State = "running"
	StatePaused     State = "paused"
	StateError      State = "error"
)

// Daemon represents the monitoring daemon
type Daemon struct {
	state           State
	config          *config.Config
	snapshotsByURL  map[string]*snapshot.Snapshot
	jobQueue        chan monitor.Job
	stopChan        chan bool
	mutex           sync.RWMutex
	logBuffer       *LogBuffer
	stats           *Stats
	dataDir         string
	monitoringActive bool
}

// Stats holds monitoring statistics
type Stats struct {
	StartedAt      time.Time
	TotalChecks    int
	FailedChecks   int
	AlertsSent     int
	LastCheckTime  time.Time
	mutex          sync.RWMutex
}

// DaemonState represents the persisted state
type DaemonState struct {
	State          State                      `json:"state"`
	Config         *config.Config             `json:"config"`
	SnapshotIDs    map[string]string          `json:"snapshot_ids"`
	Stats          *Stats                     `json:"stats"`
	LastSaved      time.Time                  `json:"last_saved"`
}

// New creates a new daemon instance
func New(dataDir string) (*Daemon, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	d := &Daemon{
		state:          StateStopped,
		snapshotsByURL: make(map[string]*snapshot.Snapshot),
		stopChan:       make(chan bool),
		logBuffer:      NewLogBuffer(1000), // Keep last 1000 log lines
		stats:          &Stats{},
		dataDir:        dataDir,
	}

	// Try to load previous state
	if err := d.loadState(); err != nil {
		log.Printf("Could not load previous state: %v (starting fresh)", err)
	}

	return d, nil
}

// GetState returns the current daemon state (thread-safe)
func (d *Daemon) GetState() State {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.state
}

// GetConfig returns the current configuration (thread-safe)
func (d *Daemon) GetConfig() *config.Config {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.config
}

// GetStats returns current statistics (thread-safe)
func (d *Daemon) GetStats() Stats {
	d.stats.mutex.RLock()
	defer d.stats.mutex.RUnlock()
	return *d.stats
}

// Start starts the monitoring with the current configuration
func (d *Daemon) Start() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.state == StateRunning {
		return fmt.Errorf("monitoring is already running")
	}

	if d.config == nil || len(d.config.Websites) == 0 {
		return fmt.Errorf("no configuration loaded")
	}

	d.state = StateRunning
	d.monitoringActive = true
	d.stats.StartedAt = time.Now()
	
	d.logBuffer.Add(fmt.Sprintf("[%s] Starting monitoring for %d websites", 
		time.Now().Format("15:04:05"), len(d.config.Websites)))

	// Save state
	if err := d.saveState(); err != nil {
		log.Printf("Warning: failed to save state: %v", err)
	}

	// Start monitoring in background
	go d.runMonitoring()

	return nil
}

// Stop stops the monitoring
func (d *Daemon) Stop() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.state != StateRunning && d.state != StatePaused {
		return fmt.Errorf("monitoring is not running")
	}

	d.state = StateStopped
	d.monitoringActive = false
	d.logBuffer.Add(fmt.Sprintf("[%s] Stopping monitoring", time.Now().Format("15:04:05")))

	// Signal stop
	select {
	case d.stopChan <- true:
	default:
	}

	// Save state
	if err := d.saveState(); err != nil {
		log.Printf("Warning: failed to save state: %v", err)
	}

	return nil
}

// Pause pauses the monitoring
func (d *Daemon) Pause() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.state != StateRunning {
		return fmt.Errorf("monitoring is not running")
	}

	d.state = StatePaused
	d.monitoringActive = false
	d.logBuffer.Add(fmt.Sprintf("[%s] Pausing monitoring", time.Now().Format("15:04:05")))

	if err := d.saveState(); err != nil {
		log.Printf("Warning: failed to save state: %v", err)
	}

	return nil
}

// Resume resumes the monitoring
func (d *Daemon) Resume() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.state != StatePaused {
		return fmt.Errorf("monitoring is not paused")
	}

	d.state = StateRunning
	d.monitoringActive = true
	d.logBuffer.Add(fmt.Sprintf("[%s] Resuming monitoring", time.Now().Format("15:04:05")))

	if err := d.saveState(); err != nil {
		log.Printf("Warning: failed to save state: %v", err)
	}

	// Restart monitoring
	go d.runMonitoring()

	return nil
}

// SetConfig sets a new configuration
func (d *Daemon) SetConfig(cfg *config.Config, snapshots map[string]*snapshot.Snapshot) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.state == StateRunning || d.state == StatePaused {
		return fmt.Errorf("cannot change configuration while monitoring is active")
	}

	d.config = cfg
	d.snapshotsByURL = snapshots

	d.logBuffer.Add(fmt.Sprintf("[%s] Configuration updated: %d websites", 
		time.Now().Format("15:04:05"), len(cfg.Websites)))

	if err := d.saveState(); err != nil {
		log.Printf("Warning: failed to save state: %v", err)
	}

	return nil
}

// GetLogs returns the last N log lines
func (d *Daemon) GetLogs(n int) []string {
	return d.logBuffer.GetLast(n)
}

// runMonitoring is the main monitoring loop
func (d *Daemon) runMonitoring() {
	const numWorkers = 30

	// Create job queue
	d.jobQueue = make(chan monitor.Job, 100) // Buffered for better performance

	// Start workers
	for i := 1; i <= numWorkers; i++ {
		go d.worker(i)
	}

	d.logBuffer.Add(fmt.Sprintf("[%s] Started %d workers", time.Now().Format("15:04:05"), numWorkers))

	for d.monitoringActive {
		startTime := time.Now()
		jobCount := len(d.config.Websites)
		
		// Queue jobs for all websites
		d.logBuffer.Add(fmt.Sprintf("[%s] Starting check cycle for %d websites", time.Now().Format("15:04:05"), jobCount))
		
		// Send jobs to workers
		for _, site := range d.config.Websites {
			if !d.monitoringActive {
				break
			}

			d.jobQueue <- monitor.Job{
				Website:  site,
				Email:    d.config.Email,
				Snapshot: d.snapshotsByURL[site],
			}
		}
		
		// Workers process jobs in background
		// The cycle logs will show when each job completes
		// We don't block here - workers log their own progress
		
		duration := time.Since(startTime)
		d.logBuffer.Add(fmt.Sprintf("[%s] Jobs queued in %v. Workers processing...", 
			time.Now().Format("15:04:05"), duration))

		// Update last check time
		d.stats.mutex.Lock()
		d.stats.LastCheckTime = time.Now()
		d.stats.mutex.Unlock()

		// Wait before next cycle
		sleepTime := time.Duration(config.WorkerSleepTime) * time.Minute
		d.logBuffer.Add(fmt.Sprintf("[%s] Next check cycle in %d minutes", 
			time.Now().Format("15:04:05"), config.WorkerSleepTime))
		
		select {
		case <-time.After(sleepTime):
			// Continue to next cycle
		case <-d.stopChan:
			// Stop monitoring
			d.logBuffer.Add(fmt.Sprintf("[%s] Monitoring stopped", time.Now().Format("15:04:05")))
			return
		}
	}

	d.logBuffer.Add(fmt.Sprintf("[%s] Monitoring loop ended", time.Now().Format("15:04:05")))
}

// worker processes monitoring jobs - uses shared ProcessJob logic
func (d *Daemon) worker(id int) {
	for job := range d.jobQueue {
		if !d.monitoringActive {
			return
		}

		// Log to buffer (visible in GUI)
		d.logBuffer.Add(fmt.Sprintf("[Worker %d] Checking %s", id, job.Website))

		// Track the check
		d.stats.mutex.Lock()
		d.stats.TotalChecks++
		d.stats.mutex.Unlock()

		// Use the SHARED ProcessJob function from monitor package
		// This is the SINGLE source of truth for monitoring logic
		err := monitor.ProcessJob(id, job)
		
		if err != nil {
			d.stats.mutex.Lock()
			d.stats.FailedChecks++
			d.stats.mutex.Unlock()
			d.logBuffer.Add(fmt.Sprintf("[Worker %d] ❌ Failed: %v", id, err))
		} else {
			d.logBuffer.Add(fmt.Sprintf("[Worker %d] ✅ Completed", id))
		}
	}
}

// saveState persists the daemon state to disk
func (d *Daemon) saveState() error {
	statePath := filepath.Join(d.dataDir, "daemon-state.json")

	// Build snapshot IDs map
	snapshotIDs := make(map[string]string)
	for url, snap := range d.snapshotsByURL {
		if snap != nil {
			snapshotIDs[url] = snap.ID
		}
	}

	state := DaemonState{
		State:       d.state,
		Config:      d.config,
		SnapshotIDs: snapshotIDs,
		Stats:       d.stats,
		LastSaved:   time.Now(),
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(statePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// loadState loads the daemon state from disk
func (d *Daemon) loadState() error {
	statePath := filepath.Join(d.dataDir, "daemon-state.json")

	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No previous state, that's okay
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

	// Load snapshots
	if state.SnapshotIDs != nil {
		for url, snapshotID := range state.SnapshotIDs {
			snap, err := snapshot.LoadByID(snapshotID)
			if err != nil {
				log.Printf("Warning: couldn't load snapshot %s for %s: %v", snapshotID, url, err)
			} else {
				d.snapshotsByURL[url] = snap
			}
		}
	}

	d.logBuffer.Add(fmt.Sprintf("[%s] Loaded previous state: %s", time.Now().Format("15:04:05"), d.state))

	// If state was running, we don't auto-start (that's intentional - requires manual start)
	if d.state == StateRunning || d.state == StatePaused {
		d.state = StateStopped
		d.logBuffer.Add("[System] Daemon restarted - monitoring stopped (use START to resume)")
	}

	return nil
}

