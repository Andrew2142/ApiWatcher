package daemon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// Client is a daemon control client
type Client struct {
	address string
	conn    net.Conn
}

// NewClient creates a new daemon client
func NewClient(address string) *Client {
	return &Client{
		address: address,
	}
}

// Connect connects to the daemon
func (c *Client) Connect() error {
	conn, err := net.DialTimeout("tcp", c.address, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	c.conn = conn
	return nil
}

// Close closes the connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// SendCommand sends a command and waits for a response
func (c *Client) SendCommand(cmd Command) (*Response, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	// Encode and send command
	encoder := json.NewEncoder(c.conn)
	if err := encoder.Encode(cmd); err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	// Read response
	decoder := json.NewDecoder(bufio.NewReader(c.conn))
	var response Response
	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return &response, nil
}

// Ping sends a ping command
func (c *Client) Ping() error {
	resp, err := c.SendCommand(Command{Type: CmdPing})
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("ping failed: %s", resp.Message)
	}
	return nil
}

// GetStatus gets the daemon status
func (c *Client) GetStatus() (*StatusData, error) {
	resp, err := c.SendCommand(Command{Type: CmdStatus})
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("failed to get status: %s", resp.Message)
	}

	// Convert data to StatusData
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	var status StatusData
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal status: %w", err)
	}

	return &status, nil
}

// Start starts monitoring
func (c *Client) Start() error {
	resp, err := c.SendCommand(Command{Type: CmdStart})
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("failed to start: %s", resp.Message)
	}
	return nil
}

// Stop stops monitoring
func (c *Client) Stop() error {
	resp, err := c.SendCommand(Command{Type: CmdStop})
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("failed to stop: %s", resp.Message)
	}
	return nil
}

// SetConfig sets the daemon configuration
func (c *Client) SetConfig(email string, websites []string, snapshotIDs map[string]string) error {
	payload := SetConfigPayload{
		Email:       email,
		Websites:    websites,
		SnapshotIDs: snapshotIDs,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	resp, err := c.SendCommand(Command{
		Type:    CmdSetConfig,
		Payload: payloadJSON,
	})
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("failed to set config: %s", resp.Message)
	}
	return nil
}

// GetLogs gets the last N log lines
func (c *Client) GetLogs(n int) ([]string, error) {
	payload := GetLogsPayload{Lines: n}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	resp, err := c.SendCommand(Command{
		Type:    CmdGetLogs,
		Payload: payloadJSON,
	})
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("failed to get logs: %s", resp.Message)
	}

	// Convert data to []string
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	var logs []string
	if err := json.Unmarshal(data, &logs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal logs: %w", err)
	}

	return logs, nil
}

