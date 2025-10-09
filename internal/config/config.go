package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"github.com/joho/godotenv"
)

// ==========================
// Environment & Config Setup
// ==========================
var WorkerSleepTime int
var SMTPFrom, SMTPUser, SMTPPass, SMTPHost, SMTPPort string

func init() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, please make one from example.env")
	}

	// Load environment variables
	SMTPFrom = os.Getenv("SMTP_FROM")
	SMTPUser = os.Getenv("SMTP_USER")
	SMTPPass = os.Getenv("SMTP_PASS")
	SMTPHost = os.Getenv("SMTP_HOST")
	SMTPPort = os.Getenv("SMTP_PORT")

	// Worker sleep time
	sleepEnv := os.Getenv("WORKER_SLEEP")
	if val, err := strconv.Atoi(sleepEnv); err == nil {
		WorkerSleepTime = val
	} else {
		WorkerSleepTime = 10 // default 10 minutes
	}
}

// ==========================
// Data Structures
// ==========================
type Config struct {
	Email    string   `json:"email"`
	Websites []string `json:"websites"`
}

type SavedMonitorConfig struct {
	Name        string            `json:"name"`         // User-friendly name
	Email       string            `json:"email"`
	Websites    []string          `json:"websites"`
	SnapshotIDs map[string]string `json:"snapshot_ids"` // URL -> Snapshot ID mapping
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// ==========================
// Configuration Helpers
// ==========================
func path() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Cannot find home directory:", err)
	}
	return filepath.Join(home, ".url-checker", "config.json")
}

func Load() (*Config, error) {
	configPath := path()
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	configPath := path()
	os.MkdirAll(filepath.Dir(configPath), 0755)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

func PromptUser() *Config {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Enter website URLs (comma separated):")
	fmt.Print("> ")
	sitesInput, _ := reader.ReadString('\n')
	websites := strings.Split(sitesInput, ",")
	for i := range websites {
		websites[i] = strings.TrimSpace(websites[i])
	}

	fmt.Println("Enter your alert email:")
	fmt.Print("> ")
	email, _ := reader.ReadString('\n')
	email = strings.TrimSpace(email)

	cfg := &Config{
		Email:    email,
		Websites: websites,
	}
	Save(cfg)
	fmt.Println("✅ Configuration saved to", path())
	return cfg
}

// savedConfigsPath returns the path to saved monitor configurations
func savedConfigsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Cannot find home directory:", err)
	}
	return filepath.Join(home, ".url-checker", "saved-configs")
}

// SaveMonitorConfig saves a complete monitoring configuration
func SaveMonitorConfig(name, email string, websites []string, snapshotIDs map[string]string) error {
	dir := savedConfigsPath()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	config := SavedMonitorConfig{
		Name:        name,
		Email:       email,
		Websites:    websites,
		SnapshotIDs: snapshotIDs,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	filename := filepath.Join(dir, sanitizeFilename(name)+".json")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

// LoadAllSavedConfigs returns all saved monitor configurations
func LoadAllSavedConfigs() ([]*SavedMonitorConfig, error) {
	dir := savedConfigsPath()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*SavedMonitorConfig{}, nil
		}
		return nil, err
	}

	var results []*SavedMonitorConfig
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var config SavedMonitorConfig
		if err := json.Unmarshal(raw, &config); err != nil {
			continue
		}
		results = append(results, &config)
	}
	return results, nil
}

// sanitizeFilename makes a string safe for use as a filename
func sanitizeFilename(name string) string {
	// Replace spaces with underscores and remove special characters
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return -1
	}, name)
	return name
}

// PromptSelectSavedConfig allows user to select from saved configurations
func PromptSelectSavedConfig() (*SavedMonitorConfig, error) {
	savedConfigs, err := LoadAllSavedConfigs()
	if err != nil {
		return nil, err
	}

	if len(savedConfigs) == 0 {
		return nil, nil // No saved configs
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("\n=== Saved Monitor Configurations ===")
	for i, cfg := range savedConfigs {
		snapshotCount := len(cfg.SnapshotIDs)
		fmt.Printf("%d [%s]\n", i+1, cfg.Name)
		fmt.Printf("   Email: %s | Sites: %s | Snapshots: %d\n",
			cfg.Email,
			strings.Join(cfg.Websites, ", "),
			snapshotCount)

		fmt.Printf("   Created: %s\n", cfg.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	fmt.Println("\nSelect a configuration (enter number, or 0 to create new):")
	fmt.Print("> ")
	choiceLine, _ := reader.ReadString('\n')
	choiceLine = strings.TrimSpace(choiceLine)

	idx, err := strconv.Atoi(choiceLine)
	if err != nil || idx < 0 || idx > len(savedConfigs) {
		fmt.Println("Invalid selection")
		return nil, nil
	}

	if idx == 0 {
		return nil, nil // User wants to create new
	}

	return savedConfigs[idx-1], nil
}

func IsStaticAsset(url string) bool {
	if idx := strings.IndexAny(url, "?#"); idx != -1 {
		url = url[:idx]
	}
	lower := strings.ToLower(url)
	// Skip by extension
	exts := []string{".js", ".css", ".png", ".jpg", ".jpeg", ".svg", ".gif", ".ico", ".woff", ".woff2", ".ttf"}
	for _, ext := range exts {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	// Skip specific domains (optional)
	if strings.Contains(lower, "fonts.gstatic.com") || strings.Contains(lower, "cdn.example.com") {
		return true
	}
	return false
}


