package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// AppSettings stores persistent application settings
type AppSettings struct {
	WorkerSleepTime     int  `json:"worker_sleep_time"`      // Minutes between monitoring cycles
	HeadlessBrowserMode bool `json:"headless_browser_mode"` // Enable headless browser mode for recordings and replays
}

var (
	currentSettings *AppSettings
	settingsMutex   sync.RWMutex
)

func init() {
	// Load settings on package init
	currentSettings = &AppSettings{
		WorkerSleepTime: 10, // default 10 minutes
	}
	_ = LoadSettings() // silently ignore load errors
}

// settingsPath returns the path to the settings file
func settingsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Cannot find home directory:", err)
	}
	return filepath.Join(home, ".url-checker", "app-settings.json")
}

// LoadSettings loads settings from disk
func LoadSettings() error {
	settingsMutex.Lock()
	defer settingsMutex.Unlock()

	path := settingsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet - that's ok, use defaults
			return nil
		}
		return fmt.Errorf("failed to read settings file: %w", err)
	}

	settings := &AppSettings{
		WorkerSleepTime: 10, // default
	}
	if err := json.Unmarshal(data, settings); err != nil {
		return fmt.Errorf("failed to unmarshal settings: %w", err)
	}

	currentSettings = settings
	return nil
}

// SaveSettings saves settings to disk
func SaveSettings(settings *AppSettings) error {
	settingsMutex.Lock()
	defer settingsMutex.Unlock()

	// Validate settings
	if settings.WorkerSleepTime < 1 {
		settings.WorkerSleepTime = 1
	}
	if settings.WorkerSleepTime > 1440 {
		settings.WorkerSleepTime = 1440
	}

	currentSettings = settings

	path := settingsPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create settings directory: %w", err)
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// GetSettings returns a copy of the current settings
func GetSettings() AppSettings {
	settingsMutex.RLock()
	defer settingsMutex.RUnlock()
	return *currentSettings
}

// GetWorkerSleepTime returns the current worker sleep time in minutes
func GetWorkerSleepTime() int {
	settingsMutex.RLock()
	defer settingsMutex.RUnlock()
	return currentSettings.WorkerSleepTime
}

// IsHeadlessBrowserMode returns whether headless browser mode is enabled
func IsHeadlessBrowserMode() bool {
	settingsMutex.RLock()
	defer settingsMutex.RUnlock()
	return currentSettings.HeadlessBrowserMode
}
