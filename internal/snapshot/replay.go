package snapshot

import (
	"apiwatcher/internal/config"
	"apiwatcher/internal/models"
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// ReplayResult holds the result of a snapshot replay including any API errors detected
type ReplayResult struct {
	SnapshotID string          // The snapshot ID
	Success    bool            // Whether replay completed without errors
	APIErrors  []*APIErrorInfo // List of API errors detected during replay
	Duration   time.Duration   // Time taken to complete replay
}

// APIErrorInfo holds information about a detected API error
type APIErrorInfo struct {
	URL        string    // The API URL that returned an error
	StatusCode int       // HTTP status code
	Timestamp  time.Time // When the error was detected
}

// Replay runs a saved snapshot in Chrome and returns an error (backward compatible).
// For detailed error information, use ReplayWithResult instead.
func Replay(s *Snapshot) error {
	_, err := ReplayWithResult(s)
	return err
}

// ReplayWithResult runs a saved snapshot in Chrome and returns detailed result information
// including any API errors detected during the replay.
func ReplayWithResult(s *Snapshot) (*ReplayResult, error) {
	startTime := time.Now()
	result := &ReplayResult{
		SnapshotID: s.ID,
		Success:    true,
		APIErrors:  make([]*APIErrorInfo, 0),
	}

	// Get headless mode setting from config
	headlessMode := config.IsHeadlessBrowserMode()

	// Chrome allocator options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", headlessMode),
		chromedp.Flag("disable-gpu", false),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("start-maximized", !headlessMode), // Only maximize if not headless
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	// Listen for network responses to catch API errors (async to avoid blocking)
	var apiErrorsMu sync.Mutex
	chromedp.ListenTarget(ctx, func(ev any) {
		if ev, ok := ev.(*network.EventResponseReceived); ok {
			if ev.Response.Status >= 400 {
				apiURL := ev.Response.URL
				if !config.IsStaticAsset(apiURL) {
					// Run in goroutine to avoid blocking the event handler
					go func(url string, status int64) {
						apiErrorsMu.Lock()
						result.APIErrors = append(result.APIErrors, &APIErrorInfo{
							URL:        url,
							StatusCode: int(status),
							Timestamp:  time.Now(),
						})
						apiErrorsMu.Unlock()
						log.Printf("[SNAPSHOT] 🚨 API Error #%d detected: %d %s\n", len(result.APIErrors), status, url)
					}(apiURL, ev.Response.Status)
				}
			}
		}
	})
	// timeout to prevent infinite hangs
	runCtx, cancelRun := context.WithTimeout(ctx, 120*time.Second) // longer if needed
	defer cancelRun()

	log.Printf("[SNAPSHOT] 🎬 Starting replay for %s (ID: %s)\n", s.URL, s.ID)

	// Navigate to the initial URL
	log.Printf("[SNAPSHOT] 🌐 Navigating to initial URL: %s\n", s.URL)
	if err := chromedp.Run(runCtx,
		chromedp.Navigate(s.URL),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Sleep(5000*time.Millisecond),
	); err != nil {
		log.Printf("[SNAPSHOT] ❌ Initial navigation failed: %v\n", err)
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("initial navigation failed: %w", err)
	}
	log.Printf("[SNAPSHOT] ✅ Initial page loaded successfully\n")

	originalActionCount := len(s.Actions)
	var filteredActions = PreprocessActions(s.Actions)
	if originalActionCount != len(filteredActions) {
		log.Printf("[SNAPSHOT] 📋 Preprocessed actions: %d -> %d (filtered %d duplicate inputs)\n",
			originalActionCount, len(filteredActions), originalActionCount-len(filteredActions))
	} else {
		log.Printf("[SNAPSHOT] 📋 Total actions to replay: %d\n", len(filteredActions))
	}

	// Replay all actions
	for i, a := range filteredActions {
		switch a.Type {
		case "navigate":
			if a.URL != "" {
				log.Printf("[SNAPSHOT] 🔄 Action %d/%d: Navigate to %s\n", i+1, len(filteredActions), a.URL)
				if err := chromedp.Run(runCtx,
					chromedp.Navigate(a.URL),
					chromedp.WaitVisible("body", chromedp.ByQuery),
				); err != nil {
					log.Printf("[SNAPSHOT] ❌ Navigation failed on action %d: %v\n", i+1, err)
				} else {
					log.Printf("[SNAPSHOT] ✅ Navigation successful\n")
				}
			}
		case "click", "mousedown":
			if a.Selector != "" {
				actionType := "Click"
				if a.Type == "mousedown" {
					actionType = "MouseDown (dropdown selection)"
				}
				desc := a.Selector
				if a.Text != "" {
					desc = fmt.Sprintf("%s (%s)", a.Selector, a.Text)
				}
				log.Printf("[SNAPSHOT] 🖱️  Action %d/%d: %s on '%s'\n", i+1, len(filteredActions), actionType, desc)
				if err := chromedp.Run(runCtx,
					chromedp.WaitVisible(a.Selector, chromedp.ByQuery),
					chromedp.ScrollIntoView(a.Selector, chromedp.ByQuery),
					chromedp.Click(a.Selector, chromedp.ByQuery),
				); err != nil {
					log.Printf("[SNAPSHOT] ❌ %s failed on action %d: %v\n", actionType, i+1, err)
				} else {
					log.Printf("[SNAPSHOT] ✅ %s successful\n", actionType)
				}
			}
		case "input":
			if a.Selector != "" {
				log.Printf("[SNAPSHOT] ⌨️  Action %d/%d: Input text into '%s'\n", i+1, len(filteredActions), a.Selector)
				if err := chromedp.Run(runCtx,
					chromedp.WaitVisible(a.Selector, chromedp.ByQuery),
					chromedp.ScrollIntoView(a.Selector, chromedp.ByQuery),
					chromedp.Focus(a.Selector, chromedp.ByQuery),
					chromedp.SendKeys(a.Selector, a.Value, chromedp.ByQuery),
				); err != nil {
					log.Printf("[SNAPSHOT] ❌ Input failed on action %d: %v\n", i+1, err)
				} else {
					log.Printf("[SNAPSHOT] ✅ Input successful\n")
				}
			}
		case "change":
			if a.Selector != "" {
				log.Printf("[SNAPSHOT] 📝 Action %d/%d: Change '%s' to '%s'\n", i+1, len(filteredActions), a.Selector, a.Value)
				// Skip change actions - they are usually redundant with input actions
				// and can cause hangs with custom form components
				log.Printf("[SNAPSHOT] ⚠️  Skipping change action (use input actions instead)\n")
			}
		case "keydown":
			if a.Selector != "" {
				log.Printf("[SNAPSHOT] ⌨️  Action %d/%d: Key press '%s' on '%s'\n", i+1, len(filteredActions), a.Key, a.Selector)
				if err := chromedp.Run(runCtx,
					chromedp.Focus(a.Selector, chromedp.ByQuery),
					chromedp.SendKeys(a.Selector, a.Key, chromedp.ByQuery),
				); err != nil {
					log.Printf("[SNAPSHOT] ❌ Keydown failed on action %d: %v\n", i+1, err)
				} else {
					log.Printf("[SNAPSHOT] ✅ Keydown successful\n")
				}
			}
		default:
			log.Printf("[SNAPSHOT] ⚠️  Action %d/%d: Unknown type '%s', skipping\n", i+1, len(filteredActions), a.Type)
		}

		// Small delay between actions
		_ = chromedp.Run(runCtx, chromedp.Sleep(500*time.Millisecond))
	}

	// Set duration and success flag
	result.Duration = time.Since(startTime)
	if len(result.APIErrors) > 0 {
		result.Success = false
		log.Printf("[SNAPSHOT] ⚠️  Replay completed for %s (ID: %s) with %d API errors detected\n", s.URL, s.ID, len(result.APIErrors))
	} else {
		result.Success = true
		log.Printf("[SNAPSHOT] 🎉 Replay completed successfully for %s (ID: %s) with no API errors\n", s.URL, s.ID)
	}
	return result, nil
}

// PreprocessActions filters out intermediate input events
func PreprocessActions(actions []models.SnapshotAction) []models.SnapshotAction {
	var filtered []models.SnapshotAction
	for i := 0; i < len(actions); i++ {
		a := actions[i]
		// Go to end of array for input
		if a.Type == "input" {
			last := a
			for j := i + 1; j < len(actions); j++ {
				next := actions[j]
				// Stop
				if next.Type != "input" || next.Selector != a.Selector {
					break
				}
				last = next
				i = j
			}
			filtered = append(filtered, last)
			continue
		}

		// Keep non-input actions as is
		filtered = append(filtered, a)
	}
	return filtered
}
