package snapshot

import (
	"context"
	"fmt"
	"log"
	"time"
	"url-checker/internal/config"
	"url-checker/internal/models"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// Replay runs a saved snapshot in Chrome.
// It respects ShowWorkerBrowser: if true, opens a visible Chrome window.
func Replay(s *Snapshot) error {
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
	var apiErrorCount int
	chromedp.ListenTarget(ctx, func(ev any) {
		if ev, ok := ev.(*network.EventResponseReceived); ok {
			if ev.Response.Status >= 400 {
				apiURL := ev.Response.URL
				if !config.IsStaticAsset(apiURL) {
					apiErrorCount++
					log.Printf("[SNAPSHOT] 🚨 API Error #%d detected: %d %s\n", apiErrorCount, int(ev.Response.Status), ev.Response.URL)
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
		return fmt.Errorf("initial navigation failed: %w", err)
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
		case "click":
			if a.Selector != "" {
				log.Printf("[SNAPSHOT] 🖱️  Action %d/%d: Click on '%s'\n", i+1, len(filteredActions), a.Selector)
				if err := chromedp.Run(runCtx,
					chromedp.Click(a.Selector, chromedp.NodeVisible),
				); err != nil {
					log.Printf("[SNAPSHOT] ❌ Click failed on action %d: %v\n", i+1, err)
				} else {
					log.Printf("[SNAPSHOT] ✅ Click successful\n")
				}
			}
		case "input":
			if a.Selector != "" {
				log.Printf("[SNAPSHOT] ⌨️  Action %d/%d: Input text into '%s'\n", i+1, len(filteredActions), a.Selector)
				if err := chromedp.Run(runCtx,
					chromedp.Click(a.Selector, chromedp.NodeVisible),
					chromedp.Sleep(150*time.Millisecond),
					chromedp.SendKeys(a.Selector, a.Value, chromedp.NodeVisible),
				); err != nil {
					log.Printf("[SNAPSHOT] ❌ Input failed on action %d: %v\n", i+1, err)
				} else {
					log.Printf("[SNAPSHOT] ✅ Input successful\n")
				}
			}
		default:
			log.Printf("[SNAPSHOT] ⚠️  Action %d/%d: Unknown type '%s', skipping\n", i+1, len(filteredActions), a.Type)
		}

		// Small delay between actions
		_ = chromedp.Run(runCtx, chromedp.Sleep(500*time.Millisecond))
	}

	if apiErrorCount > 0 {
		log.Printf("[SNAPSHOT] ⚠️  Replay completed for %s (ID: %s) with %d API errors detected\n", s.URL, s.ID, apiErrorCount)
	} else {
		log.Printf("[SNAPSHOT] 🎉 Replay completed successfully for %s (ID: %s) with no API errors\n", s.URL, s.ID)
	}
	return nil
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
