package monitor

import (
	"context"
	"fmt"
	"time"
	"url-checker/internal/config"
	"url-checker/internal/models"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// ==========================
// Website Monitoring
// ==========================
func CheckWebsite(url string) ([]*models.APIRequest, error) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	var badRequests []*models.APIRequest
	requestCount := 0
	okCount := 0
	errorCount := 0

	chromedp.ListenTarget(ctx, func(ev interface{}) {
		if resp, ok := ev.(*network.EventResponseReceived); ok {
			apiURL := resp.Response.URL

			// Skip static file types
			if config.IsStaticAsset(apiURL) {
				return
			}

			requestCount++
			status := int(resp.Response.Status)
			
			// Log EVERY non-static HTTP request with timing
			fmt.Printf("    📡 [%d] %s\n", status, apiURL)

			if status >= 400 {
				errorCount++
				fmt.Printf("    ⚠️  BAD API: %d -> %s\n", status, apiURL)
				badRequests = append(badRequests, models.NewAPIRequest(apiURL, "", status, nil, nil, ""))
			} else {
				okCount++
			}
		}
	})

	fmt.Printf("    🌐 Navigating to %s...\n", url)
	scanStart := time.Now()
	
	err := chromedp.Run(ctx,
		network.Enable(),
		chromedp.Navigate(url),
		chromedp.Sleep(time.Duration(config.WorkerSleepTime)*time.Second),
		)

	scanDuration := time.Since(scanStart)
	
	if err != nil {
		fmt.Printf("    ❌ Navigation error after %v: %v\n", scanDuration, err)
		badRequests = append(badRequests, models.NewAPIRequest(url, "", 0, nil, nil, err.Error()))
	}

	// Summary log
	fmt.Printf("    📊 Summary: %d total requests (%d OK, %d errors) in %v\n", 
		requestCount, okCount, errorCount, scanDuration)

	return badRequests, nil
}




