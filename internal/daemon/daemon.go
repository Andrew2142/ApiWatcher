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
	StateStopped State = "stopped"
	StateRunning State = "running"
	StatePaused  State = "paused"
	StateError   State = "error"
)

// Daemon represents the monitoring daemon
type Daemon struct {
	state            State
	config           *config.Config
	snapshotsByURL   map[string]*snapshot.Snapshot
	jobQueue         chan monitor.Job
	stopChan         chan bool
	mutex            sync.RWMutex
	logBuffer        *LogBuffer
	stats            *Stats
	dataDir          string
	monitoringActive bool
	jobWaitGroup     sync.WaitGroup
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

// DaemonState represents the persisted state
type DaemonState struct {
	State       State             `json:"state"`
	Config      *config.Config    `json:"config"`
	SnapshotIDs map[string]string `json:"snapshot_ids"`
	Stats       *Stats            `json:"stats"`
	LastSaved   time.Time         `json:"last_saved"`
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
		dataDir:        dataDir,
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
	defer d.mutex.Unlock()

	if d.state == StateRunning {
		return fmt.Errorf("monitoring is already running")
	}

	if d.config == nil || len(d.config.Websites) == 0 {
		return fmt.Errorf("no configuration loaded")
	}

	// Only recreate stopChan if nil or closed
	if d.stopChan == nil {
		d.stopChan = make(chan bool)
	} else {
		select {
		case <-d.stopChan:
			d.stopChan = make(chan bool)
		default:
		}
	}

	d.state = StateRunning
	d.monitoringActive = true
	d.stats.StartedAt = time.Now()

	_ = d.saveState()
	go d.runMonitoring()
	return nil
}

func (d *Daemon) Stop() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.state != StateRunning && d.state != StatePaused {
		return fmt.Errorf("monitoring is not running")
	}

	d.state = StateStopped
	d.monitoringActive = false

	// Safe stopChan close to signal workers and monitoring loop to stop
	select {
	case <-d.stopChan:
	default:
		close(d.stopChan)
	}

	// Don't wait for workers - let them exit on their own
	// This prevents blocking the GUI while workers finish current jobs
	d.Logf("Stop signal sent - workers will exit shortly")
	
	_ = d.saveState()
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

	// Only recreate stopChan if nil or closed
	if d.stopChan == nil {
		d.stopChan = make(chan bool)
	} else {
		select {
		case <-d.stopChan:
			d.stopChan = make(chan bool)
		default:
		}
	}

	_ = d.saveState()
	go d.runMonitoring()
	d.monitoringActive = true
	return nil
}

func (d *Daemon) runMonitoring() {
	const numWorkers = 30
	d.jobQueue = make(chan monitor.Job, 100)

	// Start workers
	for i := 1; i <= numWorkers; i++ {
		go d.worker(i)
	}

	for {
		select {
		case <-d.stopChan:
			d.Logf("Stop signal received, shutting down monitoring loop")
			close(d.jobQueue)
			d.jobWaitGroup.Wait()
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
			d.jobWaitGroup.Wait()
			return
		}
	}
}

func (d *Daemon) worker(id int) {
	for job := range d.jobQueue {
		// Check if we should stop before processing this job
		select {
		case <-d.stopChan:
			d.Logf("[Worker %d] Stop signal received, exiting", id)
			d.jobWaitGroup.Done() // Must call Done() since job was pulled from queue
			return
		default:
		}

		// Also check monitoringActive flag
		if !d.monitoringActive {
			d.Logf("[Worker %d] Monitoring inactive, skipping job", id)
			d.jobWaitGroup.Done()
			continue
		}

		d.stats.mutex.Lock()
		d.stats.TotalChecks++
		d.stats.mutex.Unlock()

		if err := monitor.ProcessJob(id, job, d); err != nil {
			d.stats.mutex.Lock()
			d.stats.FailedChecks++
			d.stats.mutex.Unlock()
		}

		d.jobWaitGroup.Done()
	}

	d.Logf("[Worker %d] Job queue closed, exiting", id)
}
