package remote

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ServerProfile represents a saved server configuration
type ServerProfile struct {
	Name       string     `json:"name"`
	Config     *SSHConfig `json:"config"`
	LastUsed   string     `json:"last_used"`
}

// GetProfilesDir returns the directory where profiles are stored
func GetProfilesDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".apiwatcher", "profiles")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

// SaveProfile saves a server profile
func SaveProfile(name string, config *SSHConfig) error {
	dir, err := GetProfilesDir()
	if err != nil {
		return fmt.Errorf("failed to get profiles directory: %w", err)
	}

	profile := ServerProfile{
		Name:   name,
		Config: config,
	}

	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal profile: %w", err)
	}

	filename := filepath.Join(dir, fmt.Sprintf("%s.json", name))
	if err := os.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("failed to write profile: %w", err)
	}

	return nil
}

// LoadProfile loads a server profile by name
func LoadProfile(name string) (*ServerProfile, error) {
	dir, err := GetProfilesDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get profiles directory: %w", err)
	}

	filename := filepath.Join(dir, fmt.Sprintf("%s.json", name))
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read profile: %w", err)
	}

	var profile ServerProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal profile: %w", err)
	}

	return &profile, nil
}

// ListProfiles lists all saved profiles
func ListProfiles() ([]ServerProfile, error) {
	dir, err := GetProfilesDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get profiles directory: %w", err)
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read profiles directory: %w", err)
	}

	var profiles []ServerProfile
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		name := file.Name()[:len(file.Name())-5] // Remove .json extension
		profile, err := LoadProfile(name)
		if err != nil {
			continue // Skip invalid profiles
		}
		profiles = append(profiles, *profile)
	}

	return profiles, nil
}

// DeleteProfile deletes a saved profile
func DeleteProfile(name string) error {
	dir, err := GetProfilesDir()
	if err != nil {
		return fmt.Errorf("failed to get profiles directory: %w", err)
	}

	filename := filepath.Join(dir, fmt.Sprintf("%s.json", name))
	if err := os.Remove(filename); err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}

	return nil
}

