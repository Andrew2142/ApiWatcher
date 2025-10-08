package alert

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

// ==========================
// Alert Log
// ==========================
type Log map[string]int64

// ==========================
// Alert Log Persistence
// ==========================
func logPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Cannot find home directory:", err)
	}
	return filepath.Join(home, ".url-checker", "alert_log.json")
}

func LoadLog() (Log, error) {
	path := logPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(Log), nil
		}
		return nil, err
	}
	var logData Log
	if err := json.Unmarshal(data, &logData); err != nil {
		return nil, err
	}
	return logData, nil
}

func SaveLog(logData Log) error {
	path := logPath()
	os.MkdirAll(filepath.Dir(path), 0755)
	data, err := json.MarshalIndent(logData, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

