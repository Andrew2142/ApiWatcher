package daemon

import (
	"sync"
)

// LogBuffer maintains a circular buffer of log messages
type LogBuffer struct {
	lines    []string
	maxLines int
	mutex    sync.RWMutex
}

// NewLogBuffer creates a new log buffer
func NewLogBuffer(maxLines int) *LogBuffer {
	return &LogBuffer{
		lines:    make([]string, 0, maxLines),
		maxLines: maxLines,
	}
}

// Add adds a log line to the buffer
func (lb *LogBuffer) Add(line string) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	lb.lines = append(lb.lines, line)

	// Keep only the last maxLines
	if len(lb.lines) > lb.maxLines {
		lb.lines = lb.lines[len(lb.lines)-lb.maxLines:]
	}
}

// GetLast returns the last N log lines
func (lb *LogBuffer) GetLast(n int) []string {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()

	if n <= 0 || n > len(lb.lines) {
		n = len(lb.lines)
	}

	start := len(lb.lines) - n
	if start < 0 {
		start = 0
	}

	// Return a copy to avoid concurrent modification
	result := make([]string, n)
	copy(result, lb.lines[start:])
	return result
}

// GetAll returns all log lines
func (lb *LogBuffer) GetAll() []string {
	return lb.GetLast(-1)
}

// Clear clears the log buffer
func (lb *LogBuffer) Clear() {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	lb.lines = make([]string, 0, lb.maxLines)
}

