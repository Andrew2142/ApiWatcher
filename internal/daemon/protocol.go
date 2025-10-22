package daemon

import (
	"encoding/json"
	"fmt"
	"time"
	"url-checker/internal/config"
	"url-checker/internal/snapshot"
)

// Command represents a command sent to the daemon
type Command struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// Response represents a response from the daemon
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// Command types
const (
	CmdStatus          = "STATUS"
	CmdStart           = "START"
	CmdStop            = "STOP"
	CmdPause           = "PAUSE"
	CmdResume          = "RESUME"
	CmdSetConfig       = "SET_CONFIG"
	CmdGetConfig       = "GET_CONFIG"
	CmdGetLogs         = "GET_LOGS"
	CmdClearLogs       = "CLEAR_LOGS"
	CmdGetStats        = "GET_STATS"
	CmdGetWebsiteStats = "GET_WEBSITE_STATS"
	CmdSetSMTP         = "SET_SMTP"
	CmdGetSMTP         = "GET_SMTP"
	CmdPing            = "PING"
	CmdShutdown        = "SHUTDOWN"
)

// SetConfigPayload is the payload for SET_CONFIG command
type SetConfigPayload struct {
	Email       string            `json:"email"`
	Websites    []string          `json:"websites"`
	SnapshotIDs map[string]string `json:"snapshot_ids,omitempty"`
}

// GetLogsPayload is the payload for GET_LOGS command
type GetLogsPayload struct {
	Lines int `json:"lines"`
}

// SetSMTPPayload is the payload for SET_SMTP command
type SetSMTPPayload struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
}

// StatusData is the response data for STATUS command
type StatusData struct {
	State        State     `json:"state"`
	WebsiteCount int       `json:"website_count"`
	Email        string    `json:"email"`
	HasConfig    bool      `json:"has_config"`
	HasSMTP      bool      `json:"has_smtp"`
	Stats        StatsData `json:"stats"`
}

// WebsiteStatsResponse is the response data for individual website stats
type WebsiteStatsResponse struct {
	URL                   string  `json:"url"`
	TotalChecks           int     `json:"total_checks"`
	FailedChecks          int     `json:"failed_checks"`
	ConsecutiveFailures   int     `json:"consecutive_failures"`
	ConsecutiveSuccesses  int     `json:"consecutive_successes"`
	EmailsSent            int     `json:"emails_sent"`
	LastCheckTime         string  `json:"last_check_time"`
	LastFailureTime       string  `json:"last_failure_time"`
	LastSuccessTime       string  `json:"last_success_time"`
	FirstMonitoredAt      string  `json:"first_monitored_at"`
	AverageResponseTime   string  `json:"average_response_time"`
	UptimeLastHour        float64 `json:"uptime_last_hour"`
	UptimeLast24Hours     float64 `json:"uptime_last_24_hours"`
	UptimeLast7Days       float64 `json:"uptime_last_7_days"`
	OverallHealthPercent  float64 `json:"overall_health_percent"`
	LastDowntimeDuration  string  `json:"last_downtime_duration"`
	LongestDowntime       string  `json:"longest_downtime"`
	TotalDowntime         string  `json:"total_downtime"`
	LastAlertSent         string  `json:"last_alert_sent"`
	HealthTrend           string  `json:"health_trend"`
	CurrentStatus         string  `json:"current_status"`
}

// HandleCommand processes a command and returns a response
func (d *Daemon) HandleCommand(cmd Command) Response {
	switch cmd.Type {
	case CmdPing:
		return Response{Success: true, Message: "pong"}

	case CmdStatus:
		return d.handleStatus()

	case CmdStart:
		return d.handleStart()

	case CmdStop:
		return d.handleStop()

	case CmdPause:
		return d.handlePause()

	case CmdResume:
		return d.handleResume()

	case CmdSetConfig:
		return d.handleSetConfig(cmd.Payload)

	case CmdGetConfig:
		return d.handleGetConfig()

	case CmdGetLogs:
		return d.handleGetLogs(cmd.Payload)

	case CmdClearLogs:
		return d.handleClearLogs()

	case CmdGetStats:
		return d.handleGetStats()

	case CmdGetWebsiteStats:
		return d.handleGetWebsiteStats()

	case CmdSetSMTP:
		return d.handleSetSMTP(cmd.Payload)

	case CmdGetSMTP:
		return d.handleGetSMTP()

	default:
		return Response{
			Success: false,
			Message: fmt.Sprintf("unknown command: %s", cmd.Type),
		}
	}
}

func (d *Daemon) handleStatus() Response {
	cfg := d.GetConfig()
	stats := d.GetStatsData()

	// Check if SMTP is configured
	smtpConfig, _ := config.LoadSMTPConfig()
	hasSMTP := smtpConfig != nil && smtpConfig.Host != "" && smtpConfig.Port != ""

	data := StatusData{
		State:     d.GetState(),
		HasConfig: cfg != nil,
		HasSMTP:   hasSMTP,
		Stats:     stats,
	}

	if cfg != nil {
		data.WebsiteCount = len(cfg.Websites)
		data.Email = cfg.Email
	}

	return Response{
		Success: true,
		Data:    data,
	}
}

func (d *Daemon) handleStart() Response {
	if err := d.Start(); err != nil {
		return Response{Success: false, Message: err.Error()}
	}
	return Response{Success: true, Message: "monitoring started"}
}

func (d *Daemon) handleStop() Response {
	if err := d.Stop(); err != nil {
		return Response{Success: false, Message: err.Error()}
	}
	return Response{Success: true, Message: "monitoring stopped"}
}

func (d *Daemon) handlePause() Response {
	if err := d.Pause(); err != nil {
		return Response{Success: false, Message: err.Error()}
	}
	return Response{Success: true, Message: "monitoring paused"}
}

func (d *Daemon) handleResume() Response {
	if err := d.Resume(); err != nil {
		return Response{Success: false, Message: err.Error()}
	}
	return Response{Success: true, Message: "monitoring resumed"}
}

func (d *Daemon) handleSetConfig(payload json.RawMessage) Response {
	var configPayload SetConfigPayload
	if err := json.Unmarshal(payload, &configPayload); err != nil {
		return Response{Success: false, Message: fmt.Sprintf("invalid payload: %v", err)}
	}

	// Create config
	cfg := &config.Config{
		Email:    configPayload.Email,
		Websites: configPayload.Websites,
	}

	// Load snapshots if provided
	snapshots := make(map[string]*snapshot.Snapshot)
	for url, snapshotID := range configPayload.SnapshotIDs {
		if snapshotID != "" {
			snap, err := snapshot.LoadByID(snapshotID)
			if err != nil {
				d.Logf("[WARNING] Failed to load snapshot %s for %s: %v", snapshotID, url, err)
			} else {
				snapshots[url] = snap
				d.Logf("[CONFIG] Loaded snapshot %s for %s (%d actions)", snapshotID, url, len(snap.Actions))
			}
		}
	}

	if err := d.SetConfig(cfg, snapshots); err != nil {
		return Response{Success: false, Message: err.Error()}
	}

	d.Logf("[CONFIG] Configuration updated: %d websites, %d snapshots", len(cfg.Websites), len(snapshots))
	return Response{Success: true, Message: "configuration updated"}
}

func (d *Daemon) handleGetConfig() Response {
	cfg := d.GetConfig()
	if cfg == nil {
		return Response{Success: false, Message: "no configuration loaded"}
	}
	return Response{Success: true, Data: cfg}
}

func (d *Daemon) handleGetLogs(payload json.RawMessage) Response {
	var logsPayload GetLogsPayload
	logsPayload.Lines = 100 // Default

	if payload != nil {
		if err := json.Unmarshal(payload, &logsPayload); err != nil {
			// Ignore error, use default
		}
	}

	logs := d.GetLogs(logsPayload.Lines)
	return Response{Success: true, Data: logs}
}

func (d *Daemon) handleClearLogs() Response {
	d.ClearLogs()
	return Response{Success: true, Message: "logs cleared"}
}

func (d *Daemon) handleGetStats() Response {
	stats := d.GetStatsData()
	return Response{Success: true, Data: stats}
}

func (d *Daemon) handleGetWebsiteStats() Response {
	websiteStats := d.GetAllWebsiteStats()
	
	// Convert to response format with formatted strings
	responses := make([]WebsiteStatsResponse, 0, len(websiteStats))
	for _, stats := range websiteStats {
		response := WebsiteStatsResponse{
			URL:                  stats.URL,
			TotalChecks:          stats.TotalChecks,
			FailedChecks:         stats.FailedChecks,
			ConsecutiveFailures:  stats.ConsecutiveFailures,
			ConsecutiveSuccesses: stats.ConsecutiveSuccesses,
			EmailsSent:           stats.EmailsSent,
			LastCheckTime:        formatTimeString(stats.LastCheckTime),
			LastFailureTime:      formatTimeString(stats.LastFailureTime),
			LastSuccessTime:      formatTimeString(stats.LastSuccessTime),
			FirstMonitoredAt:     formatTimeString(stats.FirstMonitoredAt),
			AverageResponseTime:  stats.AverageResponseTime.String(),
			UptimeLastHour:       stats.UptimeLastHour,
			UptimeLast24Hours:    stats.UptimeLast24Hours,
			UptimeLast7Days:      stats.UptimeLast7Days,
			OverallHealthPercent: stats.OverallHealthPercent,
			LastDowntimeDuration: stats.LastDowntimeDuration.String(),
			LongestDowntime:      stats.LongestDowntime.String(),
			TotalDowntime:        stats.TotalDowntime.String(),
			LastAlertSent:        formatTimeString(stats.LastAlertSent),
			HealthTrend:          stats.HealthTrend,
			CurrentStatus:        stats.GetCurrentStatus(),
		}
		responses = append(responses, response)
	}
	
	return Response{Success: true, Data: responses}
}

func formatTimeString(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

func (d *Daemon) handleSetSMTP(payload json.RawMessage) Response {
	var smtpPayload SetSMTPPayload
	if err := json.Unmarshal(payload, &smtpPayload); err != nil {
		return Response{Success: false, Message: fmt.Sprintf("invalid payload: %v", err)}
	}

	// Convert to config.SMTPConfig
	smtpConfig := &config.SMTPConfig{
		Host:     smtpPayload.Host,
		Port:     smtpPayload.Port,
		Username: smtpPayload.Username,
		Password: smtpPayload.Password,
		From:     smtpPayload.From,
	}

	// Validate
	if err := config.ValidateSMTPConfig(smtpConfig); err != nil {
		return Response{Success: false, Message: fmt.Sprintf("validation error: %v", err)}
	}

	// Save to daemon's local storage
	if err := config.SaveSMTPConfig(smtpConfig); err != nil {
		return Response{Success: false, Message: fmt.Sprintf("failed to save SMTP config: %v", err)}
	}

	d.Logf("SMTP configuration updated successfully")
	return Response{Success: true, Message: "SMTP configuration saved"}
}

func (d *Daemon) handleGetSMTP() Response {
	smtpConfig, err := config.LoadSMTPConfig()
	if err != nil {
		return Response{Success: false, Message: fmt.Sprintf("failed to load SMTP config: %v", err)}
	}

	if smtpConfig == nil {
		return Response{Success: false, Message: "SMTP not configured"}
	}

	// Don't send password back for security
	response := map[string]string{
		"host":     smtpConfig.Host,
		"port":     smtpConfig.Port,
		"username": smtpConfig.Username,
		"from":     smtpConfig.From,
	}

	return Response{Success: true, Data: response}
}
