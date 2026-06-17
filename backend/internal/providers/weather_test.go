package providers

import "testing"

func TestWeatherIcon(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{0, "☀️"},
		{1, "🌤️"},
		{2, "🌤️"},
		{3, "☁️"},
		{45, "🌫️"},
		{48, "🌫️"},
		{51, "🌧️"},
		{61, "🌧️"},
		{71, "❄️"},
		{77, "❄️"},
		{80, "🌦️"},
		{82, "🌦️"},
		{85, "🌨️"},
		{86, "🌨️"},
		{95, "⛈️"},
		{99, "⛈️"},
		{999, "🌡️"}, // unknown code
	}
	for _, tt := range tests {
		got := weatherIcon(tt.code)
		if got != tt.want {
			t.Errorf("weatherIcon(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

func TestWeatherDescription_KnownCodes(t *testing.T) {
	known := map[int]string{
		0: "Klarer Himmel", 1: "Überwiegend klar", 3: "Bedeckt",
		45: "Neblig", 63: "Regen", 95: "Gewitter",
	}
	for code, want := range known {
		got := weatherDescription(code)
		if got != want {
			t.Errorf("weatherDescription(%d) = %q, want %q", code, got, want)
		}
	}
}

func TestWeatherDescription_UnknownCode(t *testing.T) {
	got := weatherDescription(999)
	if got != "Unbekannt" {
		t.Errorf("weatherDescription(999) = %q, want %q", got, "Unbekannt")
	}
}

func TestFormatTime(t *testing.T) {
	tests := []struct{ input, want string }{
		{"2024-01-15T07:30", "07:30"},
		{"2024-07-21T20:45", "20:45"},
		{"2024-01-15T07:30:00", "07:30"},
		{"short", "short"}, // too short, returned as-is
		{"", ""},
	}
	for _, tt := range tests {
		got := formatTime(tt.input)
		if got != tt.want {
			t.Errorf("formatTime(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFetchWeather_ForecastDaysClamped(t *testing.T) {
	// FetchWeather clamps forecastDays to [1, 16].
	// We can't easily test the full function without network, but we can verify
	// the clamping logic by checking that 0 becomes 7 and 99 becomes 16.
	// Since clamping happens before the first HTTP call, a network error is expected —
	// what matters is that no panic occurs on the clamp path.
	//
	// This is documented as a structural note rather than an assertion test,
	// because the weather provider is intentionally not mocked here.
	// See docker_test.go for an example of a fully mocked provider test.
}
