package main

import (
	"apiwatcher/internal/config"
	"apiwatcher/internal/daemon"
	"apiwatcher/internal/remote"
	"apiwatcher/internal/snapshot"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// App represents the main application with all API methods
type App struct {
	daemonClient       *daemon.Client
	sshConn            *remote.SSHConnection
	cfg                *config.Config
	isLocalMode        bool
	cachedWebsiteStats []daemon.WebsiteStatsResponse
	preferences        *AppPreferences
	activeRecordings   map[string]chan bool
	recordingsMux      sync.Mutex
}

// AppPreferences stores user preferences
type AppPreferences struct {
	LastConnectedServer string `json:"last_connected_server"`
	AutoConnect         bool   `json:"auto_connect"`
}

// Response types for API
type ConnectionStatus struct {
	Connected    bool   `json:"connected"`
	IsLocal      bool   `json:"is_local"`
	Host         string `json:"host,omitempty"`
	User         string `json:"user,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type DaemonStatus struct {
	State   string `json:"state"`
	HasSMTP bool   `json:"has_smtp"`
	Error   string `json:"error,omitempty"`
}

type ProfileInfo struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Username string `json:"username"`
	Port     string `json:"port"`
}

type ConfigInfo struct {
	Name  string   `json:"name"`
	URLs  []string `json:"urls"`
	Error string   `json:"error,omitempty"`
}

type WebsiteStats struct {
	URL                  string  `json:"url"`
	TotalChecks          int     `json:"total_checks"`
	FailedChecks         int     `json:"failed_checks"`
	LastCheckTime        string  `json:"last_check_time"`
	CurrentStatus        string  `json:"current_status"`
	UptimeLastHour       float64 `json:"uptime_last_hour"`
	UptimeLast24Hours    float64 `json:"uptime_last_24_hours"`
	UptimeLast7Days      float64 `json:"uptime_last_7_days"`
	OverallHealthPercent float64 `json:"overall_health_percent"`
	AverageResponseTime  string  `json:"average_response_time"`
}

type DashboardData struct {
	ConnectionStatus ConnectionStatus `json:"connection_status"`
	DaemonStatus     DaemonStatus     `json:"daemon_status"`
	WebsiteStats     []WebsiteStats   `json:"website_stats"`
	Error            string           `json:"error,omitempty"`
}

type SnapshotInfo struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	CreatedAt string `json:"created_at"`
	Actions   int    `json:"actions"`
}

// NewApp creates a new App instance
func NewApp() *App {
	app := &App{
		preferences:      &AppPreferences{},
		activeRecordings: make(map[string]chan bool),
	}
	app.loadPreferences()
	return app
}

// ============ PREFERENCES ============

func (a *App) getPreferencesPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".apiwatcher")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "gui-preferences.json"), nil
}

func (a *App) loadPreferences() {
	path, err := a.getPreferencesPath()
	if err != nil {
		log.Printf("Failed to get preferences path: %v", err)
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Failed to read preferences: %v", err)
		}
		return
	}

	if err := json.Unmarshal(data, a.preferences); err != nil {
		log.Printf("Failed to unmarshal preferences: %v", err)
	}
}

func (a *App) savePreferences() error {
	path, err := a.getPreferencesPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(a.preferences, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// ============ CONNECTION MANAGEMENT ============

// ListSSHProfiles returns all saved SSH profiles
func (a *App) ListSSHProfiles() ([]ProfileInfo, error) {
	profiles, err := remote.ListProfiles()
	if err != nil {
		return nil, fmt.Errorf("failed to list profiles: %w", err)
	}

	var result []ProfileInfo
	for _, p := range profiles {
		result = append(result, ProfileInfo{
			Name:     p.Name,
			Host:     p.Config.Host,
			Username: p.Config.Username,
			Port:     p.Config.Port,
		})
	}
	return result, nil
}

// ConnectToServer connects to a remote server via SSH
func (a *App) ConnectToServer(host, username, password string) error {
	a.isLocalMode = false

	cfg := remote.SSHConfig{
		Host:       host,
		Port:       "22",
		Username:   username,
		Password:   password,
		AuthMethod: "password",
		DaemonPort: "9876",
	}

	conn, err := remote.Connect(&cfg)
	if err != nil {
		return fmt.Errorf("failed to create SSH connection: %w", err)
	}

	// Start tunnel to remote daemon
	tunnelPort, err := conn.StartTunnel()
	if err != nil {
		return fmt.Errorf("failed to start tunnel: %w", err)
	}

	// Connect to daemon through tunnel
	daemonAddr := fmt.Sprintf("localhost:%d", tunnelPort)
	a.daemonClient = daemon.NewClient(daemonAddr)
	if err := a.daemonClient.Connect(); err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}

	a.sshConn = conn
	a.preferences.LastConnectedServer = host
	_ = a.savePreferences()

	return nil
}

// StartLocalDaemon connects to a locally running daemon, auto-starting if needed
func (a *App) StartLocalDaemon() error {
	log.Println("Checking for local daemon on localhost:9876...")

	// Check if daemon is running
	conn, err := net.DialTimeout("tcp", "localhost:9876", 2*time.Second)
	if err != nil {
		log.Println("Daemon not running, attempting to auto-start...")

		// Try to start the daemon
		if err := a.startDaemonProcess(); err != nil {
			return fmt.Errorf("failed to start daemon: %w", err)
		}

		// Wait a moment for daemon to start
		time.Sleep(2 * time.Second)

		// Try to connect again
		conn, err = net.DialTimeout("tcp", "localhost:9876", 2*time.Second)
		if err != nil {
			return fmt.Errorf("daemon failed to start. Please check logs")
		}
	}
	conn.Close()

	a.isLocalMode = true
	a.daemonClient = daemon.NewClient("localhost:9876")
	if err := a.daemonClient.Connect(); err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}

	a.preferences.LastConnectedServer = "local"
	_ = a.savePreferences()

	return nil
}

// startDaemonProcess starts the daemon as a background process
func (a *App) startDaemonProcess() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot find home directory: %w", err)
	}

	daemonBinary := filepath.Join(home, ".apiwatcher", "bin", "apiwatcher-daemon")

	// Check if daemon binary exists
	if _, err := os.Stat(daemonBinary); os.IsNotExist(err) {
		return fmt.Errorf("daemon binary not found at %s. Please build and install the daemon first", daemonBinary)
	}

	// Start daemon in background
	cmd := exec.Command(daemonBinary)

	// Redirect output to daemon logs
	logFile := filepath.Join(home, ".apiwatcher", "logs", "daemon.log")
	os.MkdirAll(filepath.Dir(logFile), 0755)

	logOutput, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Warning: could not open daemon log file: %v", err)
	} else {
		defer logOutput.Close()
		cmd.Stdout = logOutput
		cmd.Stderr = logOutput
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon process: %w", err)
	}

	log.Printf("Daemon started (PID: %d)", cmd.Process.Pid)
	return nil
}

// GetConnectionStatus returns current connection status
func (a *App) GetConnectionStatus() ConnectionStatus {
	status := ConnectionStatus{}

	if a.daemonClient == nil {
		status.Connected = false
		return status
	}

	status.Connected = true
	status.IsLocal = a.isLocalMode

	if !a.isLocalMode && a.sshConn != nil {
		status.Host = a.sshConn.Config().Host
		status.User = a.sshConn.Config().Username
	}

	return status
}

// TestConnection tests SSH connectivity without fully connecting
func (a *App) TestConnection(host, username, password string) error {
	cfg := remote.SSHConfig{
		Host:       host,
		Port:       "22",
		Username:   username,
		Password:   password,
		AuthMethod: "password",
		DaemonPort: "9876",
	}

	conn, err := remote.Connect(&cfg)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	// Test the connection
	if err := conn.TestConnection(); err != nil {
		return fmt.Errorf("test command failed: %w", err)
	}

	return nil
}

// DisconnectFromServer closes the current connection
func (a *App) DisconnectFromServer() error {
	if a.daemonClient != nil {
		_ = a.daemonClient.Close()
		a.daemonClient = nil
	}

	if a.sshConn != nil {
		a.sshConn.Close()
		a.sshConn = nil
	}

	a.isLocalMode = false
	return nil
}

// ============ DASHBOARD & MONITORING ============

// GetDashboardData returns all dashboard data
// StartMonitoring starts monitoring the selected websites with snapshot preferences
func (a *App) StartMonitoring(monitoringConfigRaw interface{}) error {
	if a.daemonClient == nil {
		return fmt.Errorf("not connected to daemon")
	}

	// Get current status first
	status, err := a.daemonClient.GetStatus()
	if err != nil {
		return fmt.Errorf("failed to get daemon status: %w", err)
	}

	// If monitoring is currently active, stop it first
	if status.State == "running" || status.State == "paused" {
		log.Println("[MONITORING] Stopping active monitoring before starting new session")
		if err := a.daemonClient.Stop(); err != nil {
			return fmt.Errorf("failed to stop existing monitoring: %w", err)
		}
		// Give the daemon a moment to stop
		time.Sleep(2 * time.Second)
	}

	// Convert interface{} to map[string]interface{} with snapshot preferences
	monitoringConfig := make(map[string]bool) // URL -> enableSnapshots
	websites := []string{}

	switch v := monitoringConfigRaw.(type) {
	case map[string]interface{}:
		for url, config := range v {
			websites = append(websites, url)

			// Check if snapshots should be enabled for this URL
			if configMap, ok := config.(map[string]interface{}); ok {
				if enableSnaps, exists := configMap["enableSnapshots"]; exists {
					if enable, ok := enableSnaps.(bool); ok {
						monitoringConfig[url] = enable
					}
				}
			}
		}
	default:
		return fmt.Errorf("invalid monitoring config format")
	}

	if len(websites) == 0 {
		return fmt.Errorf("no websites selected")
	}

	// Load snapshots for enabled websites
	snapshotIDs := make(map[string]string)
	for url, enableSnapshots := range monitoringConfig {
		if enableSnapshots {
			// Load snapshots for this URL
			snapshots, err := snapshot.LoadForURL(url)
			if err == nil && len(snapshots) > 0 {
				// For now, use the most recent snapshot (could be enhanced to use all)
				snapshotIDs[url] = snapshots[0].ID
			}
		}
	}

	// Set the configuration with selected websites
	if err := a.daemonClient.SetConfig(status.Email, websites, snapshotIDs); err != nil {
		return fmt.Errorf("failed to set monitoring config: %w", err)
	}

	// Start the daemon
	if err := a.daemonClient.Start(); err != nil {
		return fmt.Errorf("failed to start monitoring: %w", err)
	}

	log.Println("[MONITORING] New monitoring session started with", len(websites), "websites")
	return nil
}

func (a *App) StopMonitoring() error {
	if a.daemonClient == nil {
		return fmt.Errorf("not connected to daemon")
	}

	if err := a.daemonClient.Stop(); err != nil {
		return fmt.Errorf("failed to stop monitoring: %w", err)
	}

	log.Println("[MONITORING] Monitoring stopped")
	return nil
}

func (a *App) GetDashboardData() (*DashboardData, error) {
	if a.daemonClient == nil {
		return nil, fmt.Errorf("not connected to daemon")
	}

	data := &DashboardData{
		ConnectionStatus: a.GetConnectionStatus(),
	}

	// Get daemon status
	status, err := a.daemonClient.GetStatus()
	if err != nil {
		data.Error = fmt.Sprintf("Failed to get daemon status: %v", err)
		return data, nil
	}

	data.DaemonStatus = DaemonStatus{
		State:   string(status.State),
		HasSMTP: status.HasSMTP,
	}

	// Get website stats
	stats, err := a.daemonClient.GetWebsiteStats()
	if err != nil {
		data.Error = fmt.Sprintf("Failed to get website stats: %v", err)
		return data, nil
	}

	a.cachedWebsiteStats = stats

	for _, stat := range stats {
		data.WebsiteStats = append(data.WebsiteStats, WebsiteStats{
			URL:                  stat.URL,
			TotalChecks:          stat.TotalChecks,
			FailedChecks:         stat.FailedChecks,
			LastCheckTime:        stat.LastCheckTime,
			CurrentStatus:        stat.CurrentStatus,
			UptimeLastHour:       stat.UptimeLastHour,
			UptimeLast24Hours:    stat.UptimeLast24Hours,
			UptimeLast7Days:      stat.UptimeLast7Days,
			OverallHealthPercent: stat.OverallHealthPercent,
			AverageResponseTime:  stat.AverageResponseTime,
		})
	}

	return data, nil
}

// GetWebsiteStats returns detailed stats for all websites
func (a *App) GetWebsiteStats() ([]WebsiteStats, error) {
	if a.daemonClient == nil {
		return nil, fmt.Errorf("not connected to daemon")
	}

	stats, err := a.daemonClient.GetWebsiteStats()
	if err != nil {
		return nil, err
	}

	a.cachedWebsiteStats = stats

	var result []WebsiteStats
	for _, stat := range stats {
		result = append(result, WebsiteStats{
			URL:                  stat.URL,
			TotalChecks:          stat.TotalChecks,
			FailedChecks:         stat.FailedChecks,
			LastCheckTime:        stat.LastCheckTime,
			CurrentStatus:        stat.CurrentStatus,
			UptimeLastHour:       stat.UptimeLastHour,
			UptimeLast24Hours:    stat.UptimeLast24Hours,
			UptimeLast7Days:      stat.UptimeLast7Days,
			OverallHealthPercent: stat.OverallHealthPercent,
			AverageResponseTime:  stat.AverageResponseTime,
		})
	}

	return result, nil
}

// GetDaemonLogs returns the last N log lines from the daemon
func (a *App) GetDaemonLogs(lines int) ([]string, error) {
	if a.daemonClient == nil {
		return nil, fmt.Errorf("not connected to daemon")
	}

	if lines <= 0 {
		lines = 100
	}

	return a.daemonClient.GetLogs(lines)
}

func (a *App) ClearLogs() error {
	if a.daemonClient == nil {
		return fmt.Errorf("not connected to daemon")
	}

	return a.daemonClient.ClearLogs()
}

// ============ DAEMON SETUP WIZARD ============

// DaemonSetupStatus represents the current daemon setup status
type DaemonSetupStatus struct {
	DaemonInstalled bool   `json:"daemon_installed"`
	DaemonRunning   bool   `json:"daemon_running"`
	Error           string `json:"error,omitempty"`
}

// CheckDaemonStatus checks if daemon is installed and running on remote server
func (a *App) CheckDaemonStatus(host, username, password string) (*DaemonSetupStatus, error) {
	status := &DaemonSetupStatus{}

	cfg := remote.SSHConfig{
		Host:       host,
		Port:       "22",
		Username:   username,
		Password:   password,
		AuthMethod: "password",
		DaemonPort: "9876",
	}

	conn, err := remote.Connect(&cfg)
	if err != nil {
		status.Error = fmt.Sprintf("SSH connection failed: %v", err)
		return status, nil
	}
	defer conn.Close()

	// Check if daemon is installed
	installed, err := conn.CheckDaemonInstalled()
	if err != nil {
		status.Error = fmt.Sprintf("Failed to check daemon installation: %v", err)
		return status, nil
	}
	status.DaemonInstalled = installed

	// Check if daemon is running
	if installed {
		running, err := conn.CheckDaemonRunning()
		if err != nil {
			status.Error = fmt.Sprintf("Failed to check daemon status: %v", err)
			return status, nil
		}
		status.DaemonRunning = running
	}

	return status, nil
}

// ============ CONFIGURATION MANAGEMENT ============

// ListConfigs returns all saved configurations
func (a *App) ListConfigs() ([]ConfigInfo, error) {
	configs, err := config.LoadAllSavedConfigs()
	if err != nil {
		return nil, fmt.Errorf("failed to list configs: %w", err)
	}

	var result []ConfigInfo
	for _, cfg := range configs {
		result = append(result, ConfigInfo{
			Name: cfg.Name,
			URLs: cfg.Websites,
		})
	}
	return result, nil
}

// LoadConfig loads a configuration by name
func (a *App) LoadConfig(name string) (*ConfigInfo, error) {
	configs, err := config.LoadAllSavedConfigs()
	if err != nil {
		return nil, fmt.Errorf("failed to load configs: %w", err)
	}

	for _, cfg := range configs {
		if cfg.Name == name {
			return &ConfigInfo{
				Name: cfg.Name,
				URLs: cfg.Websites,
			}, nil
		}
	}

	return nil, fmt.Errorf("config not found: %s", name)
}

// SaveConfig saves a configuration
func (a *App) SaveConfig(name string, urls interface{}) error {
	// Convert []interface{} to []string
	var urlsList []string

	switch v := urls.(type) {
	case []interface{}:
		for _, u := range v {
			if str, ok := u.(string); ok {
				urlsList = append(urlsList, str)
			}
		}
	case []string:
		urlsList = v
	default:
		return fmt.Errorf("invalid urls type")
	}

	return config.SaveMonitorConfig(name, "", urlsList, map[string]string{})
}

// CreateNewConfig creates a new configuration
func (a *App) CreateNewConfig(name string, urls interface{}) error {
	return a.SaveConfig(name, urls)
}

// DeleteConfig deletes a configuration by name
func (a *App) DeleteConfig(name string) error {
	return config.DeleteMonitorConfig(name)
}

// ============ SNAPSHOT MANAGEMENT ============

// ListSnapshots returns all snapshots for a URL
func (a *App) ListSnapshots(url string) ([]SnapshotInfo, error) {
	snapshots, err := snapshot.LoadForURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}

	var result []SnapshotInfo
	for _, snap := range snapshots {
		result = append(result, SnapshotInfo{
			ID:        snap.ID,
			URL:       snap.URL,
			CreatedAt: snap.CreatedAt.Format("2006-01-02 15:04:05"),
			Actions:   len(snap.Actions),
		})
	}
	return result, nil
}

// StartRecording starts a new recording session for a URL
// Returns a recording ID that can be used to stop the recording
func (a *App) StartRecording(url string) (string, error) {
	// Create a stop channel for this recording
	stopChan := make(chan bool, 1)
	recordingID := fmt.Sprintf("%d", time.Now().UnixNano())

	// Store the recording channel
	a.recordingsMux.Lock()
	a.activeRecordings[recordingID] = stopChan
	a.recordingsMux.Unlock()

	// Start recording in a goroutine
	go func() {
		snap, err := snapshot.RecordWithCallback(url, "GUI Snapshot", stopChan)
		if err != nil {
			log.Printf("[RECORDING] Failed to record: %v", err)
			return
		}

		// Snapshot is automatically saved by RecordWithCallback
		log.Printf("[RECORDING] Snapshot saved: %s", snap.ID)

		// Clean up
		a.recordingsMux.Lock()
		delete(a.activeRecordings, recordingID)
		a.recordingsMux.Unlock()
	}()

	return recordingID, nil
}

// FinishRecording stops a recording session
func (a *App) FinishRecording(recordingID string) error {
	a.recordingsMux.Lock()
	stopChan, exists := a.activeRecordings[recordingID]
	a.recordingsMux.Unlock()

	if !exists {
		return fmt.Errorf("recording not found: %s", recordingID)
	}

	// Send stop signal
	stopChan <- false // false = normal stop, true = cancel
	return nil
}

// CreateSnapshot creates a new instant snapshot for a URL (non-interactive)
func (a *App) CreateSnapshot(url string) (*SnapshotInfo, error) {
	// Create a channel that auto-signals immediately
	stopChan := make(chan bool, 1)
	stopChan <- false // Signal to stop immediately after page load

	snap, err := snapshot.RecordWithCallback(url, "Instant Snapshot", stopChan)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot: %w", err)
	}

	return &SnapshotInfo{
		ID:        snap.ID,
		URL:       snap.URL,
		CreatedAt: snap.CreatedAt.Format("2006-01-02 15:04:05"),
		Actions:   len(snap.Actions),
	}, nil
}

// DeleteSnapshot deletes a snapshot by ID
func (a *App) DeleteSnapshot(snapshotID string) error {
	return snapshot.DeleteFromDisk(snapshotID)
}

// ReplaySnapshot replays a saved snapshot in a headless browser
func (a *App) ReplaySnapshot(snapshotID string) error {
	snap, err := snapshot.LoadByID(snapshotID)
	if err != nil {
		return fmt.Errorf("failed to load snapshot: %w", err)
	}

	if snap == nil {
		return fmt.Errorf("snapshot not found: %s", snapshotID)
	}

	log.Printf("Replaying snapshot: %s (URL: %s)", snapshotID, snap.URL)
	return snapshot.Replay(snap)
}

// ============ SMTP CONFIGURATION ============

// ConfigureSMTP updates SMTP configuration
func (a *App) ConfigureSMTP(host string, port int, username, password, from, to string) error {
	if a.daemonClient == nil {
		return fmt.Errorf("not connected to daemon")
	}

	// Send SMTP config to daemon using SetSMTP
	portStr := fmt.Sprintf("%d", port)
	return a.daemonClient.SetSMTP(host, portStr, username, password, from, to)
}

// GetSMTPStatus returns current SMTP configuration status
func (a *App) GetSMTPStatus() (map[string]interface{}, error) {
	if a.daemonClient == nil {
		return nil, fmt.Errorf("not connected to daemon")
	}

	status, err := a.daemonClient.GetStatus()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"configured": status.HasSMTP,
	}, nil
}

// GetSMTPConfig returns the current SMTP configuration (without password)
func (a *App) GetSMTPConfig() (map[string]interface{}, error) {
	if a.daemonClient == nil {
		return nil, fmt.Errorf("not connected to daemon")
	}

	smtpData, err := a.daemonClient.GetSMTP()
	if err != nil {
		return nil, fmt.Errorf("failed to get SMTP config: %w", err)
	}

	// Convert string map to interface map and set defaults
	result := map[string]interface{}{
		"host":     smtpData["host"],
		"port":     "587", // default port
		"username": smtpData["username"],
		"from":     smtpData["from"],
		"to":       smtpData["to"],
		"password": "", // Never return password for security
	}

	// Try to parse port as int if available
	if port, ok := smtpData["port"]; ok && port != "" {
		result["port"] = port
	}

	return result, nil
}

// ============ UTILITIES ============

// Ping tests connection to daemon
func (a *App) Ping() bool {
	if a.daemonClient == nil {
		return false
	}
	return a.daemonClient.Ping() == nil
}

// GetLastConnectedServer returns the last server the user connected to
func (a *App) GetLastConnectedServer() string {
	return a.preferences.LastConnectedServer
}

// ============ SETTINGS ============

// GetAppSettings returns the current application settings
func (a *App) GetAppSettings() (map[string]interface{}, error) {
	settings := config.GetSettings()
	return map[string]interface{}{
		"worker_sleep_time": settings.WorkerSleepTime,
	}, nil
}

// SaveAppSettings saves application settings
func (a *App) SaveAppSettings(workerSleepTime int) error {
	settings := &config.AppSettings{
		WorkerSleepTime: workerSleepTime,
	}
	if err := config.SaveSettings(settings); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}
	log.Printf("Settings updated: worker_sleep_time=%d minutes", workerSleepTime)
	return nil
}
