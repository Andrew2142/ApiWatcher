package snapshot

import (
	"time"
	"url-checker/internal/models"
)

// ==========================
// Snapshot Structure
// ==========================
type Snapshot struct {
	ID        string           `json:"id"`
	URL       string           `json:"url"`
	Name      string           `json:"name,omitempty"`
	Actions   []models.SnapshotAction `json:"actions"`
	CreatedAt time.Time        `json:"created_at"`
}

