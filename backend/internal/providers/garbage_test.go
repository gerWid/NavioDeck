package providers

import (
	"strings"
	"testing"
)

// ---- parseDateFromDTSTART ----

func TestParseDateFromDTSTART_DateFormat(t *testing.T) {
	got := parseDateFromDTSTART("20300115")
	if got.IsZero() {
		t.Fatal("expected non-zero time for date format")
	}
	if got.Format("2006-01-02") != "2030-01-15" {
		t.Errorf("got %s, want 2030-01-15", got.Format("2006-01-02"))
	}
}

func TestParseDateFromDTSTART_DateTimeUTC(t *testing.T) {
	got := parseDateFromDTSTART("20300115T070000Z")
	if got.IsZero() {
		t.Fatal("expected non-zero time for UTC datetime")
	}
	if got.Format("2006-01-02") != "2030-01-15" {
		t.Errorf("got %s, want 2030-01-15", got.Format("2006-01-02"))
	}
}

func TestParseDateFromDTSTART_DateTimeLocal(t *testing.T) {
	got := parseDateFromDTSTART("20300115T120000")
	if got.IsZero() {
		t.Fatal("expected non-zero time for local datetime")
	}
	if got.Format("2006-01-02") != "2030-01-15" {
		t.Errorf("got %s, want 2030-01-15", got.Format("2006-01-02"))
	}
}

func TestParseDateFromDTSTART_Invalid(t *testing.T) {
	cases := []string{"", "not-a-date", "2024", "20241"}
	for _, c := range cases {
		if got := parseDateFromDTSTART(c); !got.IsZero() {
			t.Errorf("parseDateFromDTSTART(%q) = %v, want zero", c, got)
		}
	}
}

// ---- unescapeICal ----

func TestUnescapeICal(t *testing.T) {
	tests := []struct{ input, want string }{
		{`Hello\nWorld`, "Hello\nWorld"},
		{`comma\,here`, "comma,here"},
		{`semi\;colon`, "semi;colon"},
		{`back\\slash`, `back\slash`},
		{`plain text`, `plain text`},
		{`multiple\,commas\,here`, `multiple,commas,here`},
	}
	for _, tt := range tests {
		got := unescapeICal(tt.input)
		if got != tt.want {
			t.Errorf("unescapeICal(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ---- garbageIcon ----

func TestGarbageIcon(t *testing.T) {
	tests := []struct{ summary, want string }{
		{"Biotonne", "🌿"},
		{"Grünschnitt", "🌿"},
		{"BIOTONNE", "🌿"}, // case-insensitive
		{"Papiertonne", "📦"},
		{"Pappe", "📦"},
		{"Karton", "📦"},
		{"Gelbe Tonne", "🟡"},
		{"Plastik", "🟡"},
		{"Wertstoff", "🟡"},
		{"Leichtverpackung", "🟡"},
		{"Glascontainer", "🫙"},
		{"Sperrmüll", "🪑"},
		{"Möbelabholung", "🪑"},
		{"Schrottabfuhr", "🔧"},
		{"Metall", "🔧"},
		{"Elektroschrott", "💡"},
		{"Restmüll", "🗑️"},
		{"Unbekannte Tonne", "🗑"},
	}
	for _, tt := range tests {
		got := garbageIcon(tt.summary)
		if got != tt.want {
			t.Errorf("garbageIcon(%q) = %q, want %q", tt.summary, got, tt.want)
		}
	}
}

// ---- parseICal ----

const testICal = `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//Test//DE
BEGIN:VEVENT
DTSTART;VALUE=DATE:20300101
SUMMARY:Restmüll
END:VEVENT
BEGIN:VEVENT
DTSTART;VALUE=DATE:20300201
SUMMARY:Biotonne
END:VEVENT
BEGIN:VEVENT
DTSTART;VALUE=DATE:20300301
SUMMARY:Papiertonne
END:VEVENT
END:VCALENDAR`

func TestParseICal_Basic(t *testing.T) {
	events, err := parseICal(strings.NewReader(testICal), 365*10, 10)
	if err != nil {
		t.Fatalf("parseICal error: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
}

func TestParseICal_SortedByDate(t *testing.T) {
	events, err := parseICal(strings.NewReader(testICal), 365*10, 10)
	if err != nil {
		t.Fatalf("parseICal error: %v", err)
	}
	for i := 1; i < len(events); i++ {
		if events[i-1].Date > events[i].Date {
			t.Errorf("events not sorted: %s > %s", events[i-1].Date, events[i].Date)
		}
	}
}

func TestParseICal_MaxItems(t *testing.T) {
	events, err := parseICal(strings.NewReader(testICal), 365*10, 2)
	if err != nil {
		t.Fatalf("parseICal error: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("expected 2 events (maxItems=2), got %d", len(events))
	}
}

func TestParseICal_IconAssigned(t *testing.T) {
	events, err := parseICal(strings.NewReader(testICal), 365*10, 10)
	if err != nil {
		t.Fatalf("parseICal error: %v", err)
	}
	for _, ev := range events {
		if ev.Icon == "" {
			t.Errorf("event %q has empty icon", ev.Summary)
		}
	}
}

func TestParseICal_DateFormat(t *testing.T) {
	events, err := parseICal(strings.NewReader(testICal), 365*10, 10)
	if err != nil {
		t.Fatalf("parseICal error: %v", err)
	}
	for _, ev := range events {
		if len(ev.Date) != 10 || ev.Date[4] != '-' || ev.Date[7] != '-' {
			t.Errorf("event date %q not in YYYY-MM-DD format", ev.Date)
		}
	}
}

func TestParseICal_PastEventsFiltered(t *testing.T) {
	// Events in the past (year 2000) must not appear
	const old = `BEGIN:VCALENDAR
BEGIN:VEVENT
DTSTART;VALUE=DATE:20000101
SUMMARY:Alte Tonne
END:VEVENT
END:VCALENDAR`
	events, err := parseICal(strings.NewReader(old), 365, 10)
	if err != nil {
		t.Fatalf("parseICal error: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 past events, got %d", len(events))
	}
}

func TestParseICal_ICal_Unfolding(t *testing.T) {
	// Long SUMMARY lines may be folded per RFC 5545 (continuation line starts with space)
	const folded = `BEGIN:VCALENDAR
BEGIN:VEVENT
DTSTART;VALUE=DATE:20300101
SUMMARY:Rest
 müll
END:VEVENT
END:VCALENDAR`
	events, err := parseICal(strings.NewReader(folded), 365*10, 10)
	if err != nil {
		t.Fatalf("parseICal error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Summary != "Restmüll" {
		t.Errorf("unfolded summary = %q, want %q", events[0].Summary, "Restmüll")
	}
}

func TestParseICal_EmptyInput(t *testing.T) {
	events, err := parseICal(strings.NewReader(""), 30, 10)
	if err != nil {
		t.Fatalf("parseICal error on empty input: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events for empty input, got %d", len(events))
	}
}
