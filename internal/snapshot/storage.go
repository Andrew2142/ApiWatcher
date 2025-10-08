package snapshot

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

// ==========================
// Snapshot Persistence
// ==========================
func dirPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Cannot find home directory:", err)
	}
	return filepath.Join(home, ".url-checker", "snapshots")
}

func SaveToDisk(s *Snapshot) error {
	dir := dirPath()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	filename := filepath.Join(dir, s.ID+".json")
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func LoadForURL(url string) ([]*Snapshot, error) {
	dir := dirPath()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Snapshot{}, nil
		}
		return nil, err
	}
	var results []*Snapshot
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var s Snapshot
		if err := json.Unmarshal(raw, &s); err != nil {
			continue
		}
		if s.URL == url {
			// copy to heap
			cp := s
			results = append(results, &cp)
		}
	}
	return results, nil
}

func LoadAll() ([]*Snapshot, error) {
	dir := dirPath()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Snapshot{}, nil
		}
		return nil, err
	}
	var results []*Snapshot
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var s Snapshot
		if err := json.Unmarshal(raw, &s); err != nil {
			continue
		}
		cp := s
		results = append(results, &cp)
	}
	return results, nil
}

