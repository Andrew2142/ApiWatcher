package gui

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// AppPreferences stores user preferences and last connection
type AppPreferences struct {
	LastConnectedServer string `json:"last_connected_server"`
	AutoConnect         bool   `json:"auto_connect"`
}

// GetPreferencesPath returns the path to the preferences file
func GetPreferencesPath() (string, error) {
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

// LoadPreferences loads user preferences
func LoadPreferences() (*AppPreferences, error) {
	path, err := GetPreferencesPath()
	if err != nil {
		return &AppPreferences{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &AppPreferences{}, nil // No preferences yet
		}
		return nil, err
	}

	var prefs AppPreferences
	if err := json.Unmarshal(data, &prefs); err != nil {
		return &AppPreferences{}, err
	}

	return &prefs, nil
}

// SavePreferences saves user preferences
func SavePreferences(prefs *AppPreferences) error {
	path, err := GetPreferencesPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// saveLastConnection saves the last connected server
func (s *AppState) saveLastConnection(serverName string) {
	prefs, err := LoadPreferences()
	if err != nil {
		return // Silently fail
	}

	prefs.LastConnectedServer = serverName
	SavePreferences(prefs)
}

