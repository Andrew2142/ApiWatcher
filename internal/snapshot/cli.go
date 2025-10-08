package snapshot

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"url-checker/internal/config"
)

// ==========================
// CLI: Snapshot creation flow
// ==========================
func PromptSnapshotFlow(cfg *config.Config) {
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
		snapshot, err := Record(url, nameLine)
		if err != nil {
			fmt.Println("Recording failed / cancelled:", err)
			continue
		}
		fmt.Printf("Save snapshot '%s' for %s ? (y/n)\n", snapshot.Name, snapshot.URL)
		fmt.Print("> ")
		saveLine, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(saveLine)) == "y" {
			if err := SaveToDisk(snapshot); err != nil {
				fmt.Println("Failed to save snapshot:", err)
			} else {
				fmt.Println("Snapshot saved.")
			}
		} else {
			fmt.Println("Discarded snapshot.")
		}
	}
}

