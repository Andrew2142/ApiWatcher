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

	chromedp.ListenTarget(ctx, func(ev interface{}) {
		if resp, ok := ev.(*network.EventResponseReceived); ok {
			apiURL := resp.Response.URL

			// Skip static file types
			if config.IsStaticAsset(apiURL) {
				return
			}


			status := int(resp.Response.Status)
			//fmt.Printf("[INFO] %d %s\n", status, apiURL)

			if status >= 400 {
				fmt.Printf("[WARN] Bad API status: %d -> %s\n", status, apiURL)
				badRequests = append(badRequests, models.NewAPIRequest(apiURL, "", status, nil, nil, ""))
			}
		}
	})

	err := chromedp.Run(ctx,
		network.Enable(),
		chromedp.Navigate(url),
		chromedp.Sleep(time.Duration(config.WorkerSleepTime)*time.Second),
		)

	if err != nil {
		badRequests = append(badRequests, models.NewAPIRequest(url, "", 0, nil, nil, err.Error()))
	}

	return badRequests, nil
}




