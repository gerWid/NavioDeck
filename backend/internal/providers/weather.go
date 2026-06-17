package providers

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"
)

type WeatherData struct {
	City          string        `json:"city"`
	Country       string        `json:"country"`
	Lat           float64       `json:"lat"`
	Lon           float64       `json:"lon"`
	Temperature   float64       `json:"temperature"`
	FeelsLike     float64       `json:"feels_like"`
	Humidity      int           `json:"humidity"`
	WindSpeed     float64       `json:"wind_speed"`
	WindDirection int           `json:"wind_direction"`
	WindGusts     float64       `json:"wind_gusts"`
	WeatherCode   int           `json:"weather_code"`
	Icon          string        `json:"icon"`
	Description   string        `json:"description"`
	UVIndex       float64       `json:"uv_index"`
	Pressure      float64       `json:"pressure"`
	Visibility    float64       `json:"visibility"` // km
	DewPoint      float64       `json:"dew_point"`
	CloudCover    int           `json:"cloud_cover"`
	Precipitation float64       `json:"precipitation"`
	Forecast      []ForecastDay `json:"forecast"`
	Units         string        `json:"units"`
}

type ForecastDay struct {
	Date         string  `json:"date"`
	WeatherCode  int     `json:"weather_code"`
	Icon         string  `json:"icon"`
	TempMax      float64 `json:"temp_max"`
	TempMin      float64 `json:"temp_min"`
	PrecipProb   float64 `json:"precip_prob"`
	Sunrise      string  `json:"sunrise"`
	Sunset       string  `json:"sunset"`
	UVIndexMax   float64 `json:"uv_index_max"`
	PrecipSum    float64 `json:"precip_sum"`
	WindMax      float64 `json:"wind_max"`
	WindDominant int     `json:"wind_dominant"`
}

var cache struct {
	sync.Mutex
	entries map[string]cacheEntry
}

type cacheEntry struct {
	data    *WeatherData
	expires time.Time
}

func init() {
	cache.entries = make(map[string]cacheEntry)
}

func FetchWeather(city, units string, forecastDays int) (*WeatherData, error) {
	if forecastDays < 1 {
		forecastDays = 7
	}
	if forecastDays > 16 {
		forecastDays = 16
	}

	key := fmt.Sprintf("%s:%s:%d", city, units, forecastDays)
	cache.Lock()
	if e, ok := cache.entries[key]; ok && time.Now().Before(e.expires) {
		cache.Unlock()
		return e.data, nil
	}
	cache.Unlock()

	// Geocode city
	geoURL := fmt.Sprintf(
		"https://geocoding-api.open-meteo.com/v1/search?name=%s&count=1&language=de&format=json",
		url.QueryEscape(city),
	)
	resp, err := httpClient.Get(geoURL)
	if err != nil {
		return nil, fmt.Errorf("geocoding: %w", err)
	}
	defer resp.Body.Close()

	var geo struct {
		Results []struct {
			Name    string  `json:"name"`
			Country string  `json:"country"`
			Lat     float64 `json:"latitude"`
			Lon     float64 `json:"longitude"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&geo); err != nil || len(geo.Results) == 0 {
		return nil, fmt.Errorf("city not found: %s", city)
	}
	loc := geo.Results[0]

	tempUnit := "celsius"
	windUnit := "kmh"
	if units == "fahrenheit" {
		tempUnit = "fahrenheit"
	}

	weatherURL := fmt.Sprintf(
		"https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f"+
			"&current=temperature_2m,apparent_temperature,relative_humidity_2m,wind_speed_10m,wind_direction_10m,wind_gusts_10m,weather_code,surface_pressure,visibility,uv_index,dew_point_2m,cloud_cover,precipitation"+
			"&daily=temperature_2m_max,temperature_2m_min,precipitation_probability_max,weather_code,sunrise,sunset,uv_index_max,precipitation_sum,wind_speed_10m_max,wind_direction_10m_dominant"+
			"&temperature_unit=%s&wind_speed_unit=%s&timezone=auto&forecast_days=%d",
		loc.Lat, loc.Lon, tempUnit, windUnit, forecastDays,
	)
	resp2, err := httpClient.Get(weatherURL)
	if err != nil {
		return nil, fmt.Errorf("weather fetch: %w", err)
	}
	defer resp2.Body.Close()

	var raw struct {
		Current struct {
			Temp          float64 `json:"temperature_2m"`
			ApparentT     float64 `json:"apparent_temperature"`
			Humidity      float64 `json:"relative_humidity_2m"`
			WindSpeed     float64 `json:"wind_speed_10m"`
			WindDirection float64 `json:"wind_direction_10m"`
			WindGusts     float64 `json:"wind_gusts_10m"`
			WeatherCode   float64 `json:"weather_code"`
			Pressure      float64 `json:"surface_pressure"`
			Visibility    float64 `json:"visibility"`
			UVIndex       float64 `json:"uv_index"`
			DewPoint      float64 `json:"dew_point_2m"`
			CloudCover    float64 `json:"cloud_cover"`
			Precipitation float64 `json:"precipitation"`
		} `json:"current"`
		Daily struct {
			Time         []string  `json:"time"`
			TempMax      []float64 `json:"temperature_2m_max"`
			TempMin      []float64 `json:"temperature_2m_min"`
			PrecipProb   []float64 `json:"precipitation_probability_max"`
			WeatherCode  []float64 `json:"weather_code"`
			Sunrise      []string  `json:"sunrise"`
			Sunset       []string  `json:"sunset"`
			UVIndexMax   []float64 `json:"uv_index_max"`
			PrecipSum    []float64 `json:"precipitation_sum"`
			WindMax      []float64 `json:"wind_speed_10m_max"`
			WindDominant []float64 `json:"wind_direction_10m_dominant"`
		} `json:"daily"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("parse weather: %w", err)
	}

	code := int(raw.Current.WeatherCode)
	data := &WeatherData{
		City:          loc.Name,
		Country:       loc.Country,
		Lat:           loc.Lat,
		Lon:           loc.Lon,
		Temperature:   raw.Current.Temp,
		FeelsLike:     raw.Current.ApparentT,
		Humidity:      int(raw.Current.Humidity),
		WindSpeed:     raw.Current.WindSpeed,
		WindDirection: int(raw.Current.WindDirection),
		WindGusts:     raw.Current.WindGusts,
		WeatherCode:   code,
		Icon:          weatherIcon(code),
		Description:   weatherDescription(code),
		UVIndex:       raw.Current.UVIndex,
		Pressure:      raw.Current.Pressure,
		Visibility:    raw.Current.Visibility / 1000, // m → km
		DewPoint:      raw.Current.DewPoint,
		CloudCover:    int(raw.Current.CloudCover),
		Precipitation: raw.Current.Precipitation,
		Units:         units,
	}

	for i, t := range raw.Daily.Time {
		wc := int(raw.Daily.WeatherCode[i])
		sunrise := ""
		sunset := ""
		if i < len(raw.Daily.Sunrise) {
			sunrise = formatTime(raw.Daily.Sunrise[i])
		}
		if i < len(raw.Daily.Sunset) {
			sunset = formatTime(raw.Daily.Sunset[i])
		}
		uvMax := 0.0
		if i < len(raw.Daily.UVIndexMax) {
			uvMax = raw.Daily.UVIndexMax[i]
		}
		precipSum := 0.0
		if i < len(raw.Daily.PrecipSum) {
			precipSum = raw.Daily.PrecipSum[i]
		}
		windMax := 0.0
		if i < len(raw.Daily.WindMax) {
			windMax = raw.Daily.WindMax[i]
		}
		windDom := 0
		if i < len(raw.Daily.WindDominant) {
			windDom = int(raw.Daily.WindDominant[i])
		}
		data.Forecast = append(data.Forecast, ForecastDay{
			Date:         t,
			WeatherCode:  wc,
			Icon:         weatherIcon(wc),
			TempMax:      raw.Daily.TempMax[i],
			TempMin:      raw.Daily.TempMin[i],
			PrecipProb:   raw.Daily.PrecipProb[i],
			Sunrise:      sunrise,
			Sunset:       sunset,
			UVIndexMax:   uvMax,
			PrecipSum:    precipSum,
			WindMax:      windMax,
			WindDominant: windDom,
		})
	}

	cache.Lock()
	cache.entries[key] = cacheEntry{data: data, expires: time.Now().Add(30 * time.Minute)}
	cache.Unlock()

	return data, nil
}

// formatTime extracts HH:MM from an ISO datetime string like "2024-01-15T07:30"
func formatTime(s string) string {
	if len(s) >= 16 {
		return s[11:16]
	}
	return s
}

func weatherIcon(code int) string {
	switch {
	case code == 0:
		return "☀️"
	case code <= 2:
		return "🌤️"
	case code == 3:
		return "☁️"
	case code <= 49:
		return "🌫️"
	case code <= 57:
		return "🌧️"
	case code <= 67:
		return "🌧️"
	case code <= 77:
		return "❄️"
	case code <= 82:
		return "🌦️"
	case code <= 86:
		return "🌨️"
	case code <= 99:
		return "⛈️"
	default:
		return "🌡️"
	}
}

func weatherDescription(code int) string {
	descriptions := map[int]string{
		0: "Klarer Himmel", 1: "Überwiegend klar", 2: "Teilweise bewölkt",
		3: "Bedeckt", 45: "Neblig", 48: "Eisnebel",
		51: "Leichter Nieselregen", 53: "Nieselregen", 55: "Starker Nieselregen",
		61: "Leichter Regen", 63: "Regen", 65: "Starker Regen",
		71: "Leichter Schnee", 73: "Schnee", 75: "Starker Schnee",
		77: "Schneekörner", 80: "Leichte Schauer", 81: "Schauer", 82: "Starke Schauer",
		85: "Leichte Schneeschauer", 86: "Starke Schneeschauer",
		95: "Gewitter", 96: "Gewitter mit Hagel", 99: "Gewitter mit starkem Hagel",
	}
	if d, ok := descriptions[code]; ok {
		return d
	}
	return "Unbekannt"
}
