package snapshot

import (
	"context"
	"fmt"
	"log"
	"time"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"url-checker/internal/config"
)



// Replay runs a saved snapshot in Chrome.
// It respects ShowWorkerBrowser: if true, opens a visible Chrome window.
func Replay(s *Snapshot) error {
	// Chrome allocator options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
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


