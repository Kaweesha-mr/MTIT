package logging

import (
	"encoding/json"
	"log"
	"sync"
	"time"
)

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp   time.Time            `json:"timestamp"`
	Method      string               `json:"method"`
	Path        string               `json:"path"`
	Status      int                  `json:"status"`
	Duration    int64                `json:"duration_ms"`
	ClientIP    string               `json:"client_ip"`
	Query       string               `json:"query,omitempty"`
	Error       string               `json:"error,omitempty"`
	CacheHit    bool                 `json:"cache_hit"`
	RateLimited bool                 `json:"rate_limited"`
	Headers     map[string]string    `json:"headers,omitempty"`
	ResponseLen int                  `json:"response_len"`
}

// Logger manages in-memory logs with a circular buffer
type Logger struct {
	mu       sync.RWMutex
	entries  []LogEntry
	maxSize  int
	index    int
	isFull   bool
}

// NewLogger creates a new runtime logger with specified max entries
func NewLogger(maxSize int) *Logger {
	if maxSize <= 0 {
		maxSize = 1000 // default to 1000 entries
	}
	return &Logger{
		entries: make([]LogEntry, maxSize),
		maxSize: maxSize,
		index:   0,
		isFull:  false,
	}
}

// Log adds a new log entry to the circular buffer
func (l *Logger) Log(entry LogEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry.Timestamp = time.Now()
	l.entries[l.index] = entry
	l.index = (l.index + 1) % l.maxSize

	if l.index == 0 {
		l.isFull = true
	}
}

// GetEntries returns all log entries in chronological order
func (l *Logger) GetEntries() []LogEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make([]LogEntry, 0)

	if !l.isFull {
		// Buffer not full yet, just return entries up to current index
		for i := 0; i < l.index; i++ {
			if l.entries[i].Timestamp != (time.Time{}) {
				result = append(result, l.entries[i])
			}
		}
	} else {
		// Buffer is full, return in chronological order starting from current index
		for i := 0; i < l.maxSize; i++ {
			idx := (l.index + i) % l.maxSize
			result = append(result, l.entries[idx])
		}
	}

	return result
}

// GetLastN returns the last N entries
func (l *Logger) GetLastN(n int) []LogEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	all := make([]LogEntry, 0)
	if !l.isFull {
		for i := 0; i < l.index; i++ {
			if l.entries[i].Timestamp != (time.Time{}) {
				all = append(all, l.entries[i])
			}
		}
	} else {
		for i := 0; i < l.maxSize; i++ {
			idx := (l.index + i) % l.maxSize
			all = append(all, l.entries[idx])
		}
	}

	if len(all) <= n {
		return all
	}
	return all[len(all)-n:]
}

// GetStats returns aggregated statistics
func (l *Logger) GetStats() map[string]interface{} {
	l.mu.RLock()
	defer l.mu.RUnlock()

	entries := make([]LogEntry, 0)
	if !l.isFull {
		for i := 0; i < l.index; i++ {
			if l.entries[i].Timestamp != (time.Time{}) {
				entries = append(entries, l.entries[i])
			}
		}
	} else {
		for i := 0; i < l.maxSize; i++ {
			idx := (l.index + i) % l.maxSize
			entries = append(entries, l.entries[idx])
		}
	}

	stats := map[string]interface{}{
		"total_requests":    len(entries),
		"status_codes":      make(map[int]int),
		"methods":           make(map[string]int),
		"paths":             make(map[string]int),
		"avg_duration_ms":   float64(0),
		"rate_limited":      0,
		"errors":            0,
		"cache_hits":        0,
		"total_duration_ms": int64(0),
	}

	if len(entries) == 0 {
		return stats
	}

	statusCodes := stats["status_codes"].(map[int]int)
	methods := stats["methods"].(map[string]int)
	paths := stats["paths"].(map[string]int)
	var totalDuration int64 = 0
	var rateLimited, errors, cacheHits = 0, 0, 0

	for _, e := range entries {
		statusCodes[e.Status]++
		methods[e.Method]++
		paths[e.Path]++
		totalDuration += e.Duration
		if e.RateLimited {
			rateLimited++
		}
		if e.Error != "" {
			errors++
		}
		if e.CacheHit {
			cacheHits++
		}
	}

	stats["status_codes"] = statusCodes
	stats["methods"] = methods
	stats["paths"] = paths
	stats["total_duration_ms"] = totalDuration
	stats["avg_duration_ms"] = float64(totalDuration) / float64(len(entries))
	stats["rate_limited"] = rateLimited
	stats["errors"] = errors
	stats["cache_hits"] = cacheHits

	return stats
}

// Clear removes all entries
func (l *Logger) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.entries = make([]LogEntry, l.maxSize)
	l.index = 0
	l.isFull = false
	log.Println("logger: cleared all entries")
}

// MarshalJSON for JSON encoding
func (e LogEntry) MarshalJSON() ([]byte, error) {
	type Alias LogEntry
	return json.Marshal(&struct {
		Timestamp string `json:"timestamp"`
		*Alias
	}{
		Timestamp: e.Timestamp.Format("2006-01-02T15:04:05.000Z07:00"),
		Alias:     (*Alias)(&e),
	})
}
