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
func PromptSnapshotFlow(cfg *config.Config) map[string]*Snapshot {
	reader := bufio.NewReader(os.Stdin)
	snapshotsByURL := make(map[string]*Snapshot)
	fmt.Println("\nWould you like to use snapshot run mode for any of your configured sites? (y/n)")
	fmt.Print("> ")
	resp, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(resp)) != "y" {
		return snapshotsByURL
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
		return snapshotsByURL
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
		// Check for existing snapshots
		existingSnapshots, err := LoadForURL(url)
		if err != nil {
			fmt.Printf("Warning: couldn't check for existing snapshots: %v\n", err)
		}

		var selectedSnapshot *Snapshot

		// If there are existing snapshots, ask if user wants to use one
		if len(existingSnapshots) > 0 {
			fmt.Printf("\nFound %d existing snapshot(s) for %s\n", len(existingSnapshots), url)
			fmt.Println("Would you like to select a pre-saved snapshot? (y/n)")
			fmt.Print("> ")
			selectLine, _ := reader.ReadString('\n')

			if strings.TrimSpace(strings.ToLower(selectLine)) == "y" {
				// Display list of existing snapshots
				fmt.Println("\nAvailable snapshots:")
				for i, snap := range existingSnapshots {
					displayName := snap.Name
					if displayName == "" {
						displayName = "(unnamed)"
					}
					fmt.Printf("%d) %s - Created: %s (ID: %s)\n", 
						i+1, displayName, snap.CreatedAt.Format("2006-01-02 15:04:05"), snap.ID)
				}

				fmt.Println("Select a snapshot (enter number):")
				fmt.Print("> ")
				snapChoice, _ := reader.ReadString('\n')
				snapChoice = strings.TrimSpace(snapChoice)
				snapIdx, err := strconv.Atoi(snapChoice)

				if err == nil && snapIdx > 0 && snapIdx <= len(existingSnapshots) {
					selectedSnapshot = existingSnapshots[snapIdx-1]
					fmt.Printf("Selected snapshot: %s\n", selectedSnapshot.Name)
				} else {
					fmt.Println("Invalid selection, will create a new snapshot instead.")
				}
			}
		}

		// If no snapshot was selected, record a new one
		if selectedSnapshot == nil {
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
					snapshotsByURL[url] = snapshot
				}
			} else {
				fmt.Println("Discarded snapshot.")
			}
		} else {
		fmt.Printf("Using existing snapshot '%s' for monitoring.\n", selectedSnapshot.Name)
			snapshotsByURL[url] = selectedSnapshot		
		}
	}
	return snapshotsByURL
}

