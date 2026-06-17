package providers

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type GarbageData struct {
	Events []GarbageEvent `json:"events"`
	Next   *GarbageEvent  `json:"next"`
}

type GarbageEvent struct {
	Date      string `json:"date"`
	Summary   string `json:"summary"`
	Icon      string `json:"icon"`
	DaysUntil int    `json:"days_until"`
}

var garbageCache struct {
	sync.Mutex
	entries map[string]garbageCacheEntry
}

type garbageCacheEntry struct {
	data    *GarbageData
	expires time.Time
}

func init() {
	garbageCache.entries = make(map[string]garbageCacheEntry)
}

func FetchGarbage(source, dataDir string, daysAhead, maxItems int) (*GarbageData, error) {
	key := fmt.Sprintf("%s:%d:%d", source, daysAhead, maxItems)
	garbageCache.Lock()
	if e, ok := garbageCache.entries[key]; ok && time.Now().Before(e.expires) {
		garbageCache.Unlock()
		return e.data, nil
	}
	garbageCache.Unlock()

	var reader io.ReadCloser
	var err error

	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		if err := ValidateURL(source); err != nil {
			return nil, fmt.Errorf("garbage source blocked: %w", err)
		}
		resp, e := httpClient.Get(source)
		if e != nil {
			return nil, fmt.Errorf("ical fetch: %w", e)
		}
		reader = limitedBody(resp.Body)
	} else {
		// local file — restrict to dataDir
		clean := filepath.Base(source)
		reader, err = os.Open(filepath.Join(dataDir, clean))
		if err != nil {
			return nil, fmt.Errorf("ical open: %w", err)
		}
	}
	defer reader.Close()

	events, err := parseICal(reader, daysAhead, maxItems)
	if err != nil {
		return nil, err
	}

	data := &GarbageData{Events: events}
	if len(events) > 0 {
		data.Next = &events[0]
	}

	garbageCache.Lock()
	garbageCache.entries[key] = garbageCacheEntry{data: data, expires: time.Now().Add(6 * time.Hour)}
	garbageCache.Unlock()

	return data, nil
}

func parseICal(r io.Reader, daysAhead, maxItems int) ([]GarbageEvent, error) {
	now := time.Now().Truncate(24 * time.Hour)
	cutoff := now.AddDate(0, 0, daysAhead)

	scanner := bufio.NewScanner(r)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// unfold continuation lines (RFC 5545)
	var unfolded []string
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		for i+1 < len(lines) && len(lines[i+1]) > 0 && (lines[i+1][0] == ' ' || lines[i+1][0] == '\t') {
			i++
			line += strings.TrimLeft(lines[i], " \t")
		}
		unfolded = append(unfolded, line)
	}

	var events []GarbageEvent
	inEvent := false
	var dtstart, summary string

	for _, line := range unfolded {
		switch {
		case line == "BEGIN:VEVENT":
			inEvent = true
			dtstart = ""
			summary = ""
		case line == "END:VEVENT":
			if inEvent && dtstart != "" && summary != "" {
				date := parseDateFromDTSTART(dtstart)
				if !date.IsZero() && !date.Before(now) && !date.After(cutoff) {
					daysUntil := int(date.Sub(now).Hours() / 24)
					events = append(events, GarbageEvent{
						Date:      date.Format("2006-01-02"),
						Summary:   summary,
						Icon:      garbageIcon(summary),
						DaysUntil: daysUntil,
					})
				}
			}
			inEvent = false
		case inEvent && strings.HasPrefix(line, "DTSTART"):
			// DTSTART, DTSTART;VALUE=DATE, DTSTART;TZID=...
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				dtstart = strings.TrimSpace(parts[1])
			}
		case inEvent && strings.HasPrefix(line, "SUMMARY"):
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				summary = unescapeICal(strings.TrimSpace(parts[1]))
			}
		}
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].Date < events[j].Date
	})

	if maxItems > 0 && len(events) > maxItems {
		events = events[:maxItems]
	}
	return events, nil
}

func parseDateFromDTSTART(s string) time.Time {
	s = strings.TrimSpace(s)
	// DATE format: 20240115
	if len(s) == 8 {
		t, err := time.Parse("20060102", s)
		if err == nil {
			return t
		}
	}
	// DATETIME: 20240115T070000Z (16 chars) or 20240115T070000 (15 chars)
	if len(s) >= 16 {
		t, err := time.Parse("20060102T150405Z", s[:16])
		if err == nil {
			return t.UTC().Truncate(24 * time.Hour)
		}
	}
	if len(s) >= 15 {
		t, err := time.Parse("20060102T150405", s[:15])
		if err == nil {
			return t.Truncate(24 * time.Hour)
		}
	}
	return time.Time{}
}

func unescapeICal(s string) string {
	s = strings.ReplaceAll(s, "\\n", "\n")
	s = strings.ReplaceAll(s, "\\,", ",")
	s = strings.ReplaceAll(s, "\\;", ";")
	s = strings.ReplaceAll(s, "\\\\", "\\")
	return s
}

func garbageIcon(summary string) string {
	lower := strings.ToLower(summary)
	switch {
	case strings.Contains(lower, "bio") || strings.Contains(lower, "grün"):
		return "🌿"
	case strings.Contains(lower, "papier") || strings.Contains(lower, "pappe") || strings.Contains(lower, "karton"):
		return "📦"
	case strings.Contains(lower, "gelb") || strings.Contains(lower, "plastik") || strings.Contains(lower, "wert") || strings.Contains(lower, "leicht"):
		return "🟡"
	case strings.Contains(lower, "glas"):
		return "🫙"
	case strings.Contains(lower, "sperr") || strings.Contains(lower, "möbel"):
		return "🪑"
	case strings.Contains(lower, "elektro"):
		return "💡"
	case strings.Contains(lower, "schrott") || strings.Contains(lower, "metall"):
		return "🔧"
	case strings.Contains(lower, "rest"):
		return "🗑️"
	default:
		return "🗑"
	}
}
