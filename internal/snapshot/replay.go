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
	chromedp.ListenTarget(ctx, func(ev any) {
		if ev, ok := ev.(*network.EventResponseReceived); ok {
			if ev.Response.Status >= 400 {
				apiURL := ev.Response.URL
				if !config.IsStaticAsset(apiURL) {
					log.Printf("[SNAPSHOT] API error detected: %d %s\n", int(ev.Response.Status), ev.Response.URL)
				}
			}
		}
	})
	// timeout to prevent infinite hangs
	runCtx, cancelRun := context.WithTimeout(ctx, 120*time.Second) // longer if needed
	defer cancelRun()

	log.Printf("[SNAPSHOT] Starting replay for %s (%s)\n", s.URL, s.ID)

	// Navigate to the initial URL
	if err := chromedp.Run(runCtx,
		chromedp.Navigate(s.URL),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Sleep(5000*time.Millisecond),
	); err != nil {
		return fmt.Errorf("initial navigation failed: %w", err)
	}

	var filteredActions = PreprocessActions(s.Actions)

	// Replay all actions
	for i, a := range filteredActions {
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

		// Small delay between actions
		_ = chromedp.Run(runCtx, chromedp.Sleep(500*time.Millisecond))
	}

	log.Printf("[SNAPSHOT] Replay finished for %s (%s)\n", s.URL, s.ID)
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
