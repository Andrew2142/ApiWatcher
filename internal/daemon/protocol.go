package daemon

import (
	"encoding/json"
	"fmt"
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
	CmdStatus    = "STATUS"
	CmdStart     = "START"
	CmdStop      = "STOP"
	CmdPause     = "PAUSE"
	CmdResume    = "RESUME"
	CmdSetConfig = "SET_CONFIG"
	CmdGetConfig = "GET_CONFIG"
	CmdGetLogs   = "GET_LOGS"
	CmdClearLogs = "CLEAR_LOGS"
	CmdGetStats  = "GET_STATS"
	CmdPing      = "PING"
	CmdShutdown  = "SHUTDOWN"
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

// StatusData is the response data for STATUS command
type StatusData struct {
	State        State     `json:"state"`
	WebsiteCount int       `json:"website_count"`
	Email        string    `json:"email"`
	HasConfig    bool      `json:"has_config"`
	Stats        StatsData `json:"stats"`
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

	data := StatusData{
		State:     d.GetState(),
		HasConfig: cfg != nil,
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
	// Note: Snapshot loading would happen here if IDs are provided
	// For now, we'll implement this when we handle snapshot transfer

	if err := d.SetConfig(cfg, snapshots); err != nil {
		return Response{Success: false, Message: err.Error()}
	}

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
