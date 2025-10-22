package snapshot

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
	"url-checker/internal/models"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

// ==========================
// Interactive Recorder
// ==========================

// Record records user interactions with a webpage.
// For GUI mode, use RecordWithCallback instead.
func Record(targetURL string, snapshotName string) (*Snapshot, error) {
	return RecordWithCallback(targetURL, snapshotName, nil)
}

// RecordWithCallback records user interactions with optional callback for GUI integration.
// If stopChan is provided, recording stops when a value is sent to it.
// If stopChan is nil, uses stdin (CLI mode).
func RecordWithCallback(targetURL string, snapshotName string, stopChan chan bool) (*Snapshot, error) {
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

	// Storage for recorded actions
	var actions []models.SnapshotAction
	var actionsMu sync.Mutex

	// Listen for CDP events
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *page.EventJavascriptDialogOpening:
			// Handle dialogs if needed
			log.Printf("[RECORDER] Dialog opened: %s", ev.Message)

		case *page.EventFrameNavigated:
			// Track navigation events using CDP page events
			if ev.Frame.ParentID == "" { // Only track main frame navigations
				actionsMu.Lock()
				actions = append(actions, models.SnapshotAction{
					Type:      "navigate",
					URL:       ev.Frame.URL,
					Timestamp: time.Now().UnixMilli(),
				})
				actionsMu.Unlock()
				log.Printf("[RECORDER] Navigation: %s", ev.Frame.URL)
			}

		case *runtime.EventBindingCalled:
			// Handle bound function calls from page
			if ev.Name == "recordAction" {
				var action models.SnapshotAction
				if err := json.Unmarshal([]byte(ev.Payload), &action); err == nil {
					actionsMu.Lock()
					actions = append(actions, action)
					actionsMu.Unlock()
					log.Printf("[RECORDER] Recorded %s action", action.Type)
				}
			}
		}
	})

	// Start the browser and navigate
	if err := chromedp.Run(ctx); err != nil {
		log.Println("[RECORDER] warning starting chromedp:", err)
	}

	// Enable necessary CDP domains and set up event tracking
	err := chromedp.Run(ctx,
		// Enable CDP domains
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Enable page events for navigation tracking
			if err := page.Enable().Do(ctx); err != nil {
				return err
			}
			// Enable runtime for binding support
			if err := runtime.Enable().Do(ctx); err != nil {
				return err
			}
			// Add binding to create a bridge between page and Go
			// This is the recommended CDP approach instead of polling with Evaluate
			if err := runtime.AddBinding("recordAction").Do(ctx); err != nil {
				return err
			}
			return nil
		}),
		// Navigate to target URL
		chromedp.Navigate(targetURL),
		chromedp.Sleep(800*time.Millisecond),
		// Set up minimal event listeners that use the binding
		// Note: Some JS is unavoidable for recording user interactions as CDP
		// doesn't expose DOM-level click/input events. Using runtime.AddBinding
		// is the cleanest CDP-based approach for this use case.
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, exp, err := runtime.Evaluate(`
				(function() {
					function getSelector(el) {
						if (!el) return "";
						if (el.id) return "#" + el.id;
						var path = [];
						while (el && el.nodeType === 1) {
							var tag = el.tagName.toLowerCase();
							var nth = 1;
							var sib = el;
							while ((sib = sib.previousElementSibling) != null) {
								if (sib.tagName === el.tagName) nth++;
							}
							path.unshift(tag + (nth > 1 ? (':nth-of-type(' + nth + ')') : ''));
							el = el.parentElement;
						}
						return path.join(' > ');
					}

					document.addEventListener('click', function(e) {
						recordAction(JSON.stringify({
							type: 'click',
							selector: getSelector(e.target),
							timestamp: Date.now(),
							url: location.href
						}));
					}, true);

					document.addEventListener('input', function(e) {
						recordAction(JSON.stringify({
							type: 'input',
							selector: getSelector(e.target),
							value: e.target.value,
							timestamp: Date.now(),
							url: location.href
						}));
					}, true);
				})();
			`).Do(ctx)
			if err != nil {
				return err
			}
			if exp != nil {
				return exp
			}
			return nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to set up recorder: %w", err)
	}

	fmt.Printf("\n[RECORDER] Chrome opened for %s\n", targetURL)
	fmt.Println("[RECORDER] Perform the actions in the opened browser window.")

	// Wait for stop signal
	if stopChan != nil {
		// GUI mode: wait for signal from channel
		fmt.Println("[RECORDER] Waiting for GUI signal to stop recording...")
		cancelled := <-stopChan
		if cancelled {
			return nil, fmt.Errorf("recording cancelled by user")
		}
	} else {
		// CLI mode: wait for Enter key
		fmt.Println("[RECORDER] When finished press ENTER here to capture and save the snapshot (or type 'cancel').")
		fmt.Print("> ")
		inputReader := bufio.NewReader(os.Stdin)
		line, _ := inputReader.ReadString('\n')
		line = strings.TrimSpace(line)
		if strings.ToLower(line) == "cancel" {
			return nil, fmt.Errorf("recording cancelled by user")
		}
	}

	// Return recorded actions
	actionsMu.Lock()
	defer actionsMu.Unlock()

	s := &Snapshot{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		URL:       targetURL,
		Name:      snapshotName,
		Actions:   actions,
		CreatedAt: time.Now(),
	}
	return s, nil
}

