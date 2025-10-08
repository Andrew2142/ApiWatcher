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

