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

type CalendarData struct {
	Events []CalendarEvent `json:"events"`
}

type CalendarEvent struct {
	Date      string `json:"date"`
	Time      string `json:"time"`
	EndDate   string `json:"end_date,omitempty"`
	EndTime   string `json:"end_time,omitempty"`
	AllDay    bool   `json:"all_day"`
	Summary   string `json:"summary"`
	Location  string `json:"location,omitempty"`
	DaysUntil int    `json:"days_until"`
}

var calendarCache struct {
	sync.Mutex
	entries map[string]calendarCacheEntry
}

type calendarCacheEntry struct {
	data    *CalendarData
	expires time.Time
}

func init() {
	calendarCache.entries = make(map[string]calendarCacheEntry)
}

func FetchCalendar(source, dataDir string, daysAhead, maxItems int) (*CalendarData, error) {
	key := fmt.Sprintf("cal:%s:%d:%d", source, daysAhead, maxItems)
	calendarCache.Lock()
	if e, ok := calendarCache.entries[key]; ok && time.Now().Before(e.expires) {
		calendarCache.Unlock()
		return e.data, nil
	}
	calendarCache.Unlock()

	var reader io.ReadCloser
	var err error

	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		if err := ValidateURL(source); err != nil {
			return nil, fmt.Errorf("calendar source blocked: %w", err)
		}
		resp, e := httpClient.Get(source) //nolint:noctx
		if e != nil {
			return nil, fmt.Errorf("ical fetch: %w", e)
		}
		reader = limitedBody(resp.Body)
	} else {
		clean := filepath.Base(source)
		reader, err = os.Open(filepath.Join(dataDir, clean))
		if err != nil {
			return nil, fmt.Errorf("ical open: %w", err)
		}
	}
	defer reader.Close()

	events, err := parseCalendarICal(reader, daysAhead, maxItems)
	if err != nil {
		return nil, err
	}

	data := &CalendarData{Events: events}

	calendarCache.Lock()
	calendarCache.entries[key] = calendarCacheEntry{data: data, expires: time.Now().Add(30 * time.Minute)}
	calendarCache.Unlock()

	return data, nil
}

func parseCalendarICal(r io.Reader, daysAhead, maxItems int) ([]CalendarEvent, error) {
	now := time.Now()
	today := now.Truncate(24 * time.Hour)
	cutoff := today.AddDate(0, 0, daysAhead)

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
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

	var events []CalendarEvent
	inEvent := false
	var dtstart, dtend, summary, location string

	for _, line := range unfolded {
		switch {
		case line == "BEGIN:VEVENT":
			inEvent = true
			dtstart, dtend, summary, location = "", "", "", ""
		case line == "END:VEVENT":
			if inEvent && dtstart != "" && summary != "" {
				if ev, ok := buildCalendarEvent(dtstart, dtend, summary, location, today, cutoff); ok {
					events = append(events, ev)
				}
			}
			inEvent = false
		case inEvent && strings.HasPrefix(line, "DTSTART"):
			if _, val, ok := splitICalLine(line); ok {
				dtstart = val
			}
		case inEvent && strings.HasPrefix(line, "DTEND"):
			if _, val, ok := splitICalLine(line); ok {
				dtend = val
			}
		case inEvent && strings.HasPrefix(line, "SUMMARY"):
			if _, val, ok := splitICalLine(line); ok {
				summary = unescapeICal(val)
			}
		case inEvent && strings.HasPrefix(line, "LOCATION"):
			if _, val, ok := splitICalLine(line); ok {
				location = unescapeICal(val)
			}
		}
	}

	sort.Slice(events, func(i, j int) bool {
		if events[i].Date != events[j].Date {
			return events[i].Date < events[j].Date
		}
		return events[i].Time < events[j].Time
	})

	if maxItems > 0 && len(events) > maxItems {
		events = events[:maxItems]
	}
	return events, nil
}

// splitICalLine splits "KEY;PARAMS:VALUE" → returns (key, value, ok).
func splitICalLine(line string) (string, string, bool) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return "", "", false
	}
	return line[:idx], strings.TrimSpace(line[idx+1:]), true
}

func buildCalendarEvent(dtstart, dtend, summary, location string, today, cutoff time.Time) (CalendarEvent, bool) {
	startDate, startTime, allDay := parseDT(dtstart)
	if startDate.IsZero() {
		return CalendarEvent{}, false
	}
	if startDate.Before(today) || startDate.After(cutoff) {
		return CalendarEvent{}, false
	}

	daysUntil := int(startDate.Sub(today).Hours() / 24)
	ev := CalendarEvent{
		Date:      startDate.Format("2006-01-02"),
		Time:      startTime,
		AllDay:    allDay,
		Summary:   summary,
		Location:  location,
		DaysUntil: daysUntil,
	}

	if dtend != "" {
		endDate, endTime, _ := parseDT(dtend)
		if !endDate.IsZero() {
			ev.EndDate = endDate.Format("2006-01-02")
			ev.EndTime = endTime
		}
	}

	return ev, true
}

// parseDT parses an iCal DTSTART/DTEND value and returns (date, timeStr, allDay).
func parseDT(s string) (time.Time, string, bool) {
	s = strings.TrimSpace(s)
	// All-day: "20240115"
	if len(s) == 8 {
		t, err := time.Parse("20060102", s)
		if err == nil {
			return t, "", true
		}
	}
	// With Z suffix: "20240115T140000Z"
	if len(s) >= 16 && s[15] == 'Z' {
		t, err := time.Parse("20060102T150405Z", s[:16])
		if err == nil {
			local := t.Local()
			return local.Truncate(24 * time.Hour), local.Format("15:04"), false
		}
	}
	// Local time: "20240115T140000"
	if len(s) >= 15 {
		t, err := time.Parse("20060102T150405", s[:15])
		if err == nil {
			return t.Truncate(24 * time.Hour), t.Format("15:04"), false
		}
	}
	return time.Time{}, "", false
}
