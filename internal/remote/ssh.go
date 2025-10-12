package remote

import (
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHConnection represents an SSH connection to a remote server
type SSHConnection struct {
	client     *ssh.Client
	config     *SSHConfig
	tunnelConn net.Conn
}

// Config returns the SSH configuration
func (c *SSHConnection) Config() *SSHConfig {
	return c.config
}

// SSHConfig holds SSH connection configuration
type SSHConfig struct {
	Host            string
	Port            string
	Username        string
	AuthMethod      string // "password", "key", or "agent"
	Password        string
	KeyPath         string
	DaemonPort      string
}

// Connect establishes an SSH connection
func Connect(cfg *SSHConfig) (*SSHConnection, error) {
	// Build auth methods
	var authMethods []ssh.AuthMethod

	switch cfg.AuthMethod {
	case "password":
		if cfg.Password == "" {
			return nil, fmt.Errorf("password is required for password authentication")
		}
		authMethods = append(authMethods, ssh.Password(cfg.Password))

	case "key":
		if cfg.KeyPath == "" {
			return nil, fmt.Errorf("key path is required for key authentication")
		}
		key, err := os.ReadFile(cfg.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key: %w", err)
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			// Try with passphrase (we'll need to add support for this)
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))

	case "agent":
		// SSH agent support (for future implementation)
		return nil, fmt.Errorf("SSH agent authentication not yet implemented")

	default:
		return nil, fmt.Errorf("invalid authentication method: %s", cfg.AuthMethod)
	}

	// Build SSH client config
	sshConfig := &ssh.ClientConfig{
		User:            cfg.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Implement proper host key verification
		Timeout:         10 * time.Second,
	}

	// Connect to SSH server
	address := net.JoinHostPort(cfg.Host, cfg.Port)
	client, err := ssh.Dial("tcp", address, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH server: %w", err)
	}

	conn := &SSHConnection{
		client: client,
		config: cfg,
	}

	return conn, nil
}

// Close closes the SSH connection
func (c *SSHConnection) Close() error {
	if c.tunnelConn != nil {
		c.tunnelConn.Close()
	}
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// TestConnection tests if the connection is working
func (c *SSHConnection) TestConnection() error {
	session, err := c.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput("echo 'test'")
	if err != nil {
		return fmt.Errorf("test command failed: %w", err)
	}

	if string(output) != "test\n" {
		return fmt.Errorf("unexpected output: %s", output)
	}

	return nil
}

// RunCommand runs a command on the remote server
func (c *SSHConnection) RunCommand(cmd string) (string, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return string(output), fmt.Errorf("command failed: %w", err)
	}

	return string(output), nil
}

// StartTunnel starts a local port forward to the remote daemon
func (c *SSHConnection) StartTunnel() (int, error) {
	// Listen on a random local port
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, fmt.Errorf("failed to create local listener: %w", err)
	}

	localPort := listener.Addr().(*net.TCPAddr).Port

	// Start forwarding in background
	go func() {
		defer listener.Close()

		for {
			localConn, err := listener.Accept()
			if err != nil {
				return
			}

			go c.handleTunnelConnection(localConn)
		}
	}()

	return localPort, nil
}

// handleTunnelConnection handles a single tunnel connection
func (c *SSHConnection) handleTunnelConnection(localConn net.Conn) {
	defer localConn.Close()

	// Connect to remote daemon
	remoteAddr := fmt.Sprintf("localhost:%s", c.config.DaemonPort)
	remoteConn, err := c.client.Dial("tcp", remoteAddr)
	if err != nil {
		return
	}
	defer remoteConn.Close()

	// Bidirectional copy
	done := make(chan bool, 2)

	go func() {
		io.Copy(remoteConn, localConn)
		done <- true
	}()

	go func() {
		io.Copy(localConn, remoteConn)
		done <- true
	}()

	<-done
}

// CheckDaemonInstalled checks if the daemon is installed on the remote server
func (c *SSHConnection) CheckDaemonInstalled() (bool, error) {
	output, err := c.RunCommand("test -f ~/.apiwatcher/bin/apiwatcher-daemon && echo 'exists' || echo 'not found'")
	if err != nil {
		return false, err
	}

	return output == "exists\n", nil
}

// CheckDaemonRunning checks if the daemon is running on the remote server
func (c *SSHConnection) CheckDaemonRunning() (bool, error) {
	output, err := c.RunCommand("pgrep -f apiwatcher-daemon > /dev/null && echo 'running' || echo 'not running'")
	if err != nil {
		return false, err
	}

	return output == "running\n", nil
}

// UploadFile uploads a file to the remote server using a simple cat method
func (c *SSHConnection) UploadFile(localPath, remotePath string) error {
	// Read local file
	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read local file: %w", err)
	}

	// Encode data as base64 for safe transfer
	encoded := base64.StdEncoding.EncodeToString(data)

	// Create session
	session, err := c.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Upload using base64 encoded data and decode on server
	// This is more reliable than SCP protocol
	cmd := fmt.Sprintf("echo '%s' | base64 -d > %s", encoded, remotePath)
	
	if err := session.Run(cmd); err != nil {
		return fmt.Errorf("upload command failed: %w", err)
	}

	return nil
}

