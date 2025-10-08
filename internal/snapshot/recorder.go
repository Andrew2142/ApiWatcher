package snapshot

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
	"url-checker/internal/models"

	"github.com/chromedp/chromedp"
)

// ==========================
// Interactive Recorder
// ==========================
func Record(targetURL string, snapshotName string) (*Snapshot, error) {
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

	var rawActions []models.SnapshotAction
	_ = json.Unmarshal([]byte(actionsJSON), &rawActions)

	s := &Snapshot{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		URL:       targetURL,
		Name:      snapshotName,
		Actions:   rawActions,
		CreatedAt: time.Now(),
	}
	return s, nil
}

