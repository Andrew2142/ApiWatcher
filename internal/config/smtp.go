package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SMTPConfig represents email configuration
type SMTPConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	To       string `json:"to"` // Email address to send alerts to
}

// GetSMTPConfigPath returns the path to SMTP configuration file
func GetSMTPConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot find home directory: %w", err)
	}
	dir := filepath.Join(home, ".apiwatcher", "smtp")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("cannot create SMTP config directory: %w", err)
	}
	return filepath.Join(dir, "smtp-config.json"), nil
}

// SaveSMTPConfig saves SMTP configuration to file
func SaveSMTPConfig(config *SMTPConfig) error {
	path, err := GetSMTPConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal SMTP config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write SMTP config: %w", err)
	}

	// Also generate/update .env file
	if err := GenerateEnvFile(config); err != nil {
		return fmt.Errorf("failed to generate .env file: %w", err)
	}

	return nil
}

// LoadSMTPConfig loads SMTP configuration from file
func LoadSMTPConfig() (*SMTPConfig, error) {
	path, err := GetSMTPConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No config exists yet
		}
		return nil, fmt.Errorf("failed to read SMTP config: %w", err)
	}

	var config SMTPConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SMTP config: %w", err)
	}

	return &config, nil
}

// GenerateEnvFile creates/updates the .env file based on SMTP config
func GenerateEnvFile(config *SMTPConfig) error {
	// Find project root (where go.mod is)
	projectRoot, err := findProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to find project root: %w", err)
	}

	envPath := filepath.Join(projectRoot, ".env")

	// Build .env content
	envContent := fmt.Sprintf(`# API Watcher SMTP Configuration
# Auto-generated from GUI settings
# You can also edit these values manually

SMTP_HOST=%s
SMTP_PORT=%s
SMTP_USER=%s
SMTP_PASS=%s
SMTP_FROM=%s

# Worker sleep time in minutes (default: 10)
WORKER_SLEEP=10
`, config.Host, config.Port, config.Username, config.Password, config.From)

	if err := os.WriteFile(envPath, []byte(envContent), 0600); err != nil {
		return fmt.Errorf("failed to write .env file: %w", err)
	}

	return nil
}

// findProjectRoot searches upward for the directory containing go.mod
func findProjectRoot() (string, error) {
	// Start from current working directory
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Search upward until we find go.mod
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root without finding go.mod
			return "", fmt.Errorf("could not find go.mod in any parent directory")
		}
		dir = parent
	}
}

// ValidateSMTPConfig validates SMTP configuration fields
func ValidateSMTPConfig(config *SMTPConfig) error {
	if config == nil {
		return fmt.Errorf("SMTP config is nil")
	}

	if strings.TrimSpace(config.Host) == "" {
		return fmt.Errorf("SMTP host is required")
	}

	if strings.TrimSpace(config.Port) == "" {
		return fmt.Errorf("SMTP port is required")
	}

	if strings.TrimSpace(config.Username) == "" {
		return fmt.Errorf("SMTP username is required")
	}

	if strings.TrimSpace(config.Password) == "" {
		return fmt.Errorf("SMTP password is required")
	}

	if strings.TrimSpace(config.From) == "" {
		return fmt.Errorf("SMTP from address is required")
	}

	// Basic email format validation for 'from' field
	if !strings.Contains(config.From, "@") {
		return fmt.Errorf("invalid from email address")
	}

	if strings.TrimSpace(config.To) == "" {
		return fmt.Errorf("alert email address is required")
	}

	// Basic email format validation for 'to' field
	if !strings.Contains(config.To, "@") {
		return fmt.Errorf("invalid alert email address")
	}

	return nil
}

// LoadOrCreateDefaultSMTPConfig loads existing config or returns a default template
func LoadOrCreateDefaultSMTPConfig() *SMTPConfig {
	config, err := LoadSMTPConfig()
	if err != nil || config == nil {
		// Return default template
		return &SMTPConfig{
			Host:     "smtp.gmail.com",
			Port:     "587",
			Username: "",
			Password: "",
			From:     "",
		}
	}
	return config
}
