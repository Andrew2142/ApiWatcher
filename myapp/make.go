package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/smtp"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"url-checker/myapp/structs"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
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

type Job struct {
	Website string
	Email   string
}

type AlertLog map[string]int64

// ==========================
// Configuration Helpers
// ==========================
func configPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Cannot find home directory:", err)
	}
	return filepath.Join(home, ".url-checker", "config.json")
}

func loadConfig() (*Config, error) {
	path := configPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func saveConfig(cfg *Config) error {
	path := configPath()
	os.MkdirAll(filepath.Dir(path), 0755)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func promptUser() *Config {
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
	saveConfig(cfg)
	fmt.Println("✅ Configuration saved to", configPath())
	return cfg
}

// ==========================
// Email Alert Function
// ==========================
func sendEmail(to, subject, body string) error {
	addr := SMTPHost + ":" + SMTPPort

	msg := []byte("To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n\r\n" +
		body + "\r\n")

	auth := smtp.PlainAuth("", SMTPUser, SMTPPass, SMTPHost)
	return smtp.SendMail(addr, auth, SMTPFrom, []string{to}, msg)
}

// ==========================
// Website Monitoring
// ==========================
func checkWebsite(url string) ([]*structs.APIRequest, error) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	var badRequests []*structs.APIRequest

	chromedp.ListenTarget(ctx, func(ev interface{}) {
		if resp, ok := ev.(*network.EventResponseReceived); ok {
			apiURL := resp.Response.URL

			// Skip static file types
			lower := strings.ToLower(apiURL)
			if strings.HasSuffix(lower, ".js") ||
				strings.HasSuffix(lower, ".css") ||
				strings.HasSuffix(lower, ".png") ||
				strings.HasSuffix(lower, ".jpg") ||
				strings.HasSuffix(lower, ".jpeg") ||
				strings.HasSuffix(lower, ".svg") ||
				strings.HasSuffix(lower, ".gif") ||
				strings.HasSuffix(lower, ".ico") ||
				strings.HasSuffix(lower, ".woff") ||
				strings.HasSuffix(lower, ".woff2") ||
				strings.HasSuffix(lower, ".ttf") {
				return
			}

			status := int(resp.Response.Status)
			fmt.Printf("[INFO] %d %s\n", status, apiURL)

			if status >= 400 {
				fmt.Printf("[WARN] Bad API status: %d -> %s\n", status, apiURL)
				badRequests = append(badRequests, structs.NewAPIRequest(apiURL, "", status, nil, nil, ""))
			}
		}
	})

	err := chromedp.Run(ctx,
		network.Enable(),
		chromedp.Navigate(url),
		chromedp.Sleep(time.Duration(WorkerSleepTime)*time.Second),
	)

	if err != nil {
		badRequests = append(badRequests, structs.NewAPIRequest(url, "", 0, nil, nil, err.Error()))
	}

	return badRequests, nil
}

// ==========================
// Worker & Job Processing
// ==========================
func worker(id int, jobs <-chan Job) {
	for job := range jobs {
		fmt.Printf("[WORKER %d] Checking %s\n", id, job.Website)
		badRequests, err := checkWebsite(job.Website)
		if err != nil {
			log.Println("[ERROR]", err)
			continue
		}

		alertLog, _ := loadAlertLog()
		now := time.Now().Unix()
		fiveHours := int64(5 * 3600) // 5 hours in seconds

		if len(badRequests) > 0 {
			lastAlert, exists := alertLog[job.Website]

			body := "The following API calls failed:\n\n"
			for _, r := range badRequests {
				body += fmt.Sprintf("%d %s\n", r.StatusCode, r.URL)
			}

			if exists && now-lastAlert < fiveHours {
				log.Printf("[INFO] Skipping email for %s (sent recently)\n", job.Website)
			} else {
				if sendErr := sendEmail(job.Email, "⚠️ API Errors Detected", body); sendErr != nil {
					log.Println("[ERROR] Failed to send email:", sendErr)
				} else {
					log.Println("[ALERT] Email sent successfully")
					alertLog[job.Website] = now
					if err := saveAlertLog(alertLog); err != nil {
						log.Println("[ERROR] Failed to save alert log:", err)
					}
				}
			}

		} else {
			log.Println("[OK] No API errors detected for", job.Website)
		}
		log.Printf("Workers will resume in %d minutes", WorkerSleepTime)
	}
}

// ==========================
// Alert Log Persistence
// ==========================
func alertLogPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Cannot find home directory:", err)
	}
	return filepath.Join(home, ".url-checker", "alert_log.json")
}

func loadAlertLog() (AlertLog, error) {
	path := alertLogPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(AlertLog), nil
		}
		return nil, err
	}
	var log AlertLog
	if err := json.Unmarshal(data, &log); err != nil {
		return nil, err
	}
	return log, nil
}

func saveAlertLog(log AlertLog) error {
	path := alertLogPath()
	os.MkdirAll(filepath.Dir(path), 0755)
	data, err := json.MarshalIndent(log, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// ==========================
// Main Application
// ==========================
func main() {
	var cfg *Config
	var err error

	cfg, err = loadConfig()
	if err != nil {
		fmt.Println("No saved configuration found. Let's set it up!")
		cfg = promptUser()
	} else {
		fmt.Printf("Loaded saved configuration:\n📧 Email: %s\n🌐 Websites: %v\n", cfg.Email, cfg.Websites)
		fmt.Println("Would you like to use this configuration? (y/n)")
		var choice string
		fmt.Print("> ")
		fmt.Scanln(&choice)
		if strings.ToLower(choice) != "y" {
			cfg = promptUser()
		}
	}

	const numWorkers = 30
	jobQueue := make(chan Job, len(cfg.Websites))

	for i := 1; i <= numWorkers; i++ {
		go worker(i, jobQueue)
	}

	fmt.Printf("[START] Monitoring %d websites every %d minutes. Alerts to %s\n", len(cfg.Websites), WorkerSleepTime, cfg.Email)
	for {
		for _, site := range cfg.Websites {
			jobQueue <- Job{Website: site, Email: cfg.Email}
		}
		time.Sleep(time.Duration(WorkerSleepTime) * time.Minute)
	}
}

