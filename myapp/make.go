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
var ShowWorkerBrowser = true
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
	Snapshot *structs.Snapshot 
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
			//fmt.Printf("[INFO] %d %s\n", status, apiURL)

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
// Interactive Recorder
// ==========================
func recordRunthrough(targetURL string, snapshotName string) (*structs.Snapshot, error) {
	// Launch a visible Chrome instance (non-headless)
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("start-maximized", true),
		)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Start the browser
	if err := chromedp.Run(ctx); err != nil {
		// not fatal here, try to continue
		log.Println("[RECORDER] warning starting chromedp:", err)
	}

	// JS to inject: collects clicks, inputs and navigations into window._aw_actions
	js := `
	(function(){
	if (!window._aw_actions) window._aw_actions = [];
	function getSelector(el){
	if(!el) return "";
	if(el.id) return "#"+el.id;
	var path=[];
	while(el && el.nodeType===1){
	var tag=el.tagName.toLowerCase();
	var nth=1;
	var sib=el;
	while((sib=sib.previousElementSibling)!=null){
	if(sib.tagName===el.tagName) nth++;
	}
	path.unshift(tag + (nth>1?(':nth-of-type('+nth+')'):'' ));
	el = el.parentElement;
	}
	return path.join(' > ');
	}
	document.addEventListener('click', function(e){
	try{ window._aw_actions.push({type:'click', selector:getSelector(e.target), timestamp:Date.now(), url:location.href}); }catch(err){}
	}, true);
	document.addEventListener('input', function(e){
	try{ window._aw_actions.push({type:'input', selector:getSelector(e.target), value:e.target.value, timestamp:Date.now(), url:location.href}); }catch(err){}
	}, true);
	window.addEventListener('hashchange', function(){ window._aw_actions.push({type:'navigate', url:location.href, timestamp:Date.now()}); });
	// pushState/replaceState
	(function(history){
	var push = history.pushState;
	history.pushState = function(){
	if(typeof push === 'function') push.apply(history, arguments);
	window._aw_actions.push({type:'navigate', url:location.href, timestamp:Date.now()});
	};
	var replace = history.replaceState;
	history.replaceState = function(){
	if(typeof replace === 'function') replace.apply(history, arguments);
	window._aw_actions.push({type:'navigate', url:location.href, timestamp:Date.now()});
	};
	})(window.history);
	})();
	`

	// Navigate, inject recorder JS
	err := chromedp.Run(ctx,
		chromedp.Navigate(targetURL),
		chromedp.Sleep(800*time.Millisecond),
		chromedp.Evaluate(js, nil),
		)
	if err != nil {
		return nil, err
	}

	fmt.Printf("\n[RECORDER] Chrome opened for %s\n", targetURL)
	fmt.Println("[RECORDER] Perform the actions in the opened browser window.")
	fmt.Println("[RECORDER] When finished press ENTER here to capture and save the snapshot (or type 'cancel').")
	fmt.Print("> ")
	// Wait for user to press Enter (so recording happens on the live page)
	inputReader := bufio.NewReader(os.Stdin)
	line, _ := inputReader.ReadString('\n')
	line = strings.TrimSpace(line)
	if strings.ToLower(line) == "cancel" {
		return nil, fmt.Errorf("recording cancelled by user")
	}

	// Grab actions from page
	var actionsJSON string
	evalErr := chromedp.Run(ctx,
		chromedp.Evaluate(`JSON.stringify(window._aw_actions || [])`, &actionsJSON),
		)
	if evalErr != nil {
		// if the browser was closed or evaluate failed, try fallback to empty list
		log.Println("[RECORDER] warning: couldn't fetch actions from page:", evalErr)
		actionsJSON = "[]"
	}

	var rawActions []structs.SnapshotAction
	_ = json.Unmarshal([]byte(actionsJSON), &rawActions)

	s := &structs.Snapshot{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		URL:       targetURL,
		Name:      snapshotName,
		Actions:   rawActions,
		CreatedAt: time.Now(),
	}
	return s, nil
}

// replaySnapshot runs a saved snapshot in Chrome.
// It respects ShowWorkerBrowser: if true, opens a visible Chrome window.
func replaySnapshot(s *structs.Snapshot) error {
	// Chrome allocator options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-gpu", false),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("start-maximized", true),
		)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	// Listen for network responses to catch API errors
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		if ev, ok := ev.(*network.EventResponseReceived); ok {
			if ev.Response.Status >= 400 {
				log.Printf("[SNAPSHOT] API error detected: %d %s\n", int(ev.Response.Status), ev.Response.URL)
			}
		}
	})

	// timeout to prevent infinite hangs
	runCtx, cancelRun := context.WithTimeout(ctx, 30*time.Second) // longer if needed
	defer cancelRun()

	log.Printf("[SNAPSHOT] Starting replay for %s (%s)\n", s.URL, s.ID)

	// Navigate to the initial URL
	if err := chromedp.Run(runCtx,
		chromedp.Navigate(s.URL),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),      	
		); err != nil {
		return fmt.Errorf("initial navigation failed: %w", err)
	}

	// Replay all actions
	for i, a := range s.Actions {
		switch a.Type {
		case "navigate":
			if a.URL != "" {
				log.Printf("[SNAPSHOT] Action %d: navigate -> %s\n", i+1, a.URL)
				if err := chromedp.Run(runCtx,
					chromedp.Navigate(a.URL),
					chromedp.WaitVisible("body", chromedp.ByQuery),
					); err != nil {
					log.Printf("[SNAPSHOT] navigation failed: %v\n", err)
				}
			}
		case "click":
			if a.Selector != "" {
				log.Printf("[SNAPSHOT] Action %d: click -> %s\n", i+1, a.Selector)
				if err := chromedp.Run(runCtx,
					chromedp.Click(a.Selector, chromedp.NodeVisible),
					); err != nil {
					log.Printf("[SNAPSHOT] click failed: %v\n", err)
				}
			}
		case "input":
			if a.Selector != "" {
				log.Printf("[SNAPSHOT] Action %d: input -> %s\n", i+1, a.Selector)
				if err := chromedp.Run(runCtx,
					chromedp.Click(a.Selector, chromedp.NodeVisible),
					chromedp.Sleep(150*time.Millisecond),
					chromedp.SendKeys(a.Selector, a.Value, chromedp.NodeVisible),
					); err != nil {
					log.Printf("[SNAPSHOT] input failed: %v\n", err)
				}
			}
		default:
			log.Printf("[SNAPSHOT] Action %d: unknown type '%s', skipping\n", i+1, a.Type)
		}

		// pause to see actions
		_ = chromedp.Run(runCtx, chromedp.Sleep(500*time.Millisecond))
	}

	log.Printf("[SNAPSHOT] Replay finished for %s (%s)\n", s.URL, s.ID)
	return nil
}


func runSnapshotsForWebsites(websites []string) {
	allSnapshots, err := loadAllSnapshots()
	if err != nil {
		log.Println("[SNAPSHOT] failed to load snapshots:", err)
		return
	}
	if len(allSnapshots) == 0 {
		return
	}
	// group by URL for simple matching
	matches := map[string][]*structs.Snapshot{}
	for _, s := range allSnapshots {
		matches[s.URL] = append(matches[s.URL], s)
	}

	for _, site := range websites {
		if snaps, ok := matches[site]; ok {
			log.Printf("[SNAPSHOT] Running %d snapshot(s) for %s\n", len(snaps), site)
			for _, snap := range snaps {
				if err := replaySnapshot(snap); err != nil {
					log.Printf("[SNAPSHOT] replay error for %s (%s): %v\n", site, snap.ID, err)
				} else {
					log.Printf("[SNAPSHOT] replay finished for %s (%s)\n", site, snap.ID)
				}
			}
		}
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
// Snapshot Persistence
// ==========================
func snapshotsDirPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Cannot find home directory:", err)
	}
	return filepath.Join(home, ".url-checker", "snapshots")
}

func saveSnapshotToDisk(s *structs.Snapshot) error {
	dir := snapshotsDirPath()
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

func loadSnapshotsForURL(url string) ([]*structs.Snapshot, error) {
	dir := snapshotsDirPath()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*structs.Snapshot{}, nil
		}
		return nil, err
	}
	var results []*structs.Snapshot
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var s structs.Snapshot
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

func loadAllSnapshots() ([]*structs.Snapshot, error) {
	dir := snapshotsDirPath()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*structs.Snapshot{}, nil
		}
		return nil, err
	}
	var results []*structs.Snapshot
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var s structs.Snapshot
		if err := json.Unmarshal(raw, &s); err != nil {
			continue
		}
		cp := s
		results = append(results, &cp)
	}
	return results, nil
}

// ==========================
// CLI: Snapshot creation flow
// ==========================
func promptSnapshotFlow(cfg *Config) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("\nWould you like to create a snapshot run-through for any of your configured sites? (y/n)")
	fmt.Print("> ")
	resp, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(resp)) != "y" {
		return
	}

	// List sites
	for i, s := range cfg.Websites {
		fmt.Printf("%d) %s\n", i+1, s)
	}
	fmt.Println("Select sites to record (comma separated indices, e.g. 1 or 1,3):")
	fmt.Print("> ")
	choiceLine, _ := reader.ReadString('\n')
	choiceLine = strings.TrimSpace(choiceLine)
	if choiceLine == "" {
		return
	}
	parts := strings.Split(choiceLine, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		idx, err := strconv.Atoi(p)
		if err != nil || idx <= 0 || idx > len(cfg.Websites) {
			fmt.Println("Skipping invalid selection:", p)
			continue
		}
		url := cfg.Websites[idx-1]
		fmt.Printf("Recording run-through for: %s\n", url)
		fmt.Println("Enter a name for this snapshot (optional):")
		fmt.Print("> ")
		nameLine, _ := reader.ReadString('\n')
		nameLine = strings.TrimSpace(nameLine)
		snapshot, err := recordRunthrough(url, nameLine)
		if err != nil {
			fmt.Println("Recording failed / cancelled:", err)
			continue
		}
		fmt.Printf("Save snapshot '%s' for %s ? (y/n)\n", snapshot.Name, snapshot.URL)
		fmt.Print("> ")
		saveLine, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(saveLine)) == "y" {
			if err := saveSnapshotToDisk(snapshot); err != nil {
				fmt.Println("Failed to save snapshot:", err)
			} else {
				fmt.Println("Snapshot saved.")
			}
		} else {
			fmt.Println("Discarded snapshot.")
		}
	}
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
		if job.Snapshot != nil {
			log.Printf("[WORKER %d] Running snapshot for %s\n", id, job.Website)
			if err := replaySnapshot(job.Snapshot); err != nil {
				log.Printf("[WORKER %d] Snapshot replay error for %s (%s): %v\n", id, job.Website, job.Snapshot.ID, err)
			} else {
				log.Printf("[WORKER %d] Snapshot replay finished for %s (%s)\n", id, job.Website, job.Snapshot.ID)
			}
		}
		log.Printf("[WORKER %d] will resume in %d minutes", id, WorkerSleepTime)
	}
}

// ==========================
// Main Application
// ==========================
func main() {
	var cfg *Config
	var err error

	// Load config	
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

	//Snapshot prep
	fmt.Println("Would you like to see the snapshot replay in Chrome? (y/n)")
	var seeBrowser string
	fmt.Print("> ")
	fmt.Scanln(&seeBrowser)
	ShowWorkerBrowser = strings.ToLower(seeBrowser) == "y"
	promptSnapshotFlow(cfg)

	//Load snapshots
	allSnapshots, _ := loadAllSnapshots()
	snapshotsByURL := map[string]*structs.Snapshot{}
	for _, s := range allSnapshots {
		snapshotsByURL[s.URL] = s
	}


	//Start workers
	const numWorkers = 30
	jobQueue := make(chan Job, len(cfg.Websites))

	for i := 1; i <= numWorkers; i++ {
		go worker(i, jobQueue)
	}

	fmt.Printf("[START] Monitoring %d websites every %d minutes. Alerts to %s\n", len(cfg.Websites), WorkerSleepTime, cfg.Email)


	//Monitor
	for {
		for _, site := range cfg.Websites {
			jobQueue <- Job{
				Website:   site,
				Email:     cfg.Email,
				Snapshot: snapshotsByURL[site], 		
			}
		}

		time.Sleep(time.Duration(WorkerSleepTime) * time.Minute)
		log.Printf("Workers will resume in %d minutes", WorkerSleepTime)
	}
}

