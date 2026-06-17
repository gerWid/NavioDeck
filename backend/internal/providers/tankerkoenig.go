package providers

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

var fuelPriceCache struct {
	sync.Mutex
	entries map[string]fuelPriceCacheEntry
}

type fuelPriceCacheEntry struct {
	data    *FuelPricesData
	expires time.Time
}

type FuelData struct {
	Stations []FuelStation `json:"stations"`
	Location string        `json:"location"`
}

type FuelStation struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Brand  string  `json:"brand"`
	Street string  `json:"street"`
	City   string  `json:"city"`
	Dist   float64 `json:"dist"`
	E5     float64 `json:"e5"`
	E10    float64 `json:"e10"`
	Diesel float64 `json:"diesel"`
	IsOpen bool    `json:"is_open"`
}

type FuelPricesData struct {
	Prices map[string]FuelPrice `json:"prices"`
}

type FuelPrice struct {
	E5     float64 `json:"e5"`
	E10    float64 `json:"e10"`
	Diesel float64 `json:"diesel"`
	IsOpen bool    `json:"is_open"`
}

var fuelCache struct {
	sync.Mutex
	entries map[string]fuelCacheEntry
}

type fuelCacheEntry struct {
	data    *FuelData
	expires time.Time
}

func init() {
	fuelCache.entries = make(map[string]fuelCacheEntry)
	fuelPriceCache.entries = make(map[string]fuelPriceCacheEntry)
}

func FetchFuel(apiKey, location string, lat, lng, radius float64, sortBy string, maxStations int) (*FuelData, error) {
	var resolvedLat, resolvedLng float64
	var resolvedLocation string

	if lat != 0 && lng != 0 {
		resolvedLat = lat
		resolvedLng = lng
		resolvedLocation = location
	} else if location != "" {
		geoURL := fmt.Sprintf(
			"https://geocoding-api.open-meteo.com/v1/search?name=%s&count=1&language=de&format=json",
			url.QueryEscape(location),
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
			return nil, fmt.Errorf("location not found: %s", location)
		}
		resolvedLat = geo.Results[0].Lat
		resolvedLng = geo.Results[0].Lon
		resolvedLocation = geo.Results[0].Name
	} else {
		return nil, fmt.Errorf("location or coordinates required")
	}

	if radius <= 0 {
		radius = 5
	}
	if maxStations <= 0 {
		maxStations = 5
	}

	key := fmt.Sprintf("%.4f:%.4f:%.1f:%s:%s:%d", resolvedLat, resolvedLng, radius, apiKey, sortBy, maxStations)
	fuelCache.Lock()
	if e, ok := fuelCache.entries[key]; ok && time.Now().Before(e.expires) {
		fuelCache.Unlock()
		return e.data, nil
	}
	fuelCache.Unlock()

	tkURL := fmt.Sprintf(
		"https://creativecommons.tankerkoenig.de/json/list.php?lat=%.6f&lng=%.6f&rad=%.1f&sort=dist&type=all&apikey=%s",
		resolvedLat, resolvedLng, radius, url.QueryEscape(apiKey),
	)
	resp, err := httpClient.Get(tkURL)
	if err != nil {
		return nil, fmt.Errorf("tankerkoenig fetch: %w", err)
	}
	defer resp.Body.Close()

	var raw struct {
		OK       bool `json:"ok"`
		Stations []struct {
			ID     string  `json:"id"`
			Name   string  `json:"name"`
			Brand  string  `json:"brand"`
			Street string  `json:"street"`
			Place  string  `json:"place"`
			Dist   float64 `json:"dist"`
			E5     float64 `json:"e5"`
			E10    float64 `json:"e10"`
			Diesel float64 `json:"diesel"`
			IsOpen bool    `json:"isOpen"`
		} `json:"stations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("tankerkoenig decode: %w", err)
	}
	if !raw.OK {
		return nil, fmt.Errorf("tankerkoenig api error (check API key)")
	}

	stations := make([]FuelStation, 0, len(raw.Stations))
	for _, s := range raw.Stations {
		stations = append(stations, FuelStation{
			ID:     s.ID,
			Name:   s.Name,
			Brand:  strings.TrimSpace(s.Brand),
			Street: s.Street,
			City:   s.Place,
			Dist:   s.Dist,
			E5:     s.E5,
			E10:    s.E10,
			Diesel: s.Diesel,
			IsOpen: s.IsOpen,
		})
	}

	if sortBy == "price_e5" {
		sort.Slice(stations, func(i, j int) bool {
			ai, aj := stations[i].E5, stations[j].E5
			if ai <= 0 { return false }
			if aj <= 0 { return true }
			return ai < aj
		})
	} else if sortBy == "price_e10" {
		sort.Slice(stations, func(i, j int) bool {
			ai, aj := stations[i].E10, stations[j].E10
			if ai <= 0 { return false }
			if aj <= 0 { return true }
			return ai < aj
		})
	} else if sortBy == "price_diesel" {
		sort.Slice(stations, func(i, j int) bool {
			ai, aj := stations[i].Diesel, stations[j].Diesel
			if ai <= 0 { return false }
			if aj <= 0 { return true }
			return ai < aj
		})
	}
	// default: already sorted by dist from API

	if maxStations < len(stations) {
		stations = stations[:maxStations]
	}

	data := &FuelData{
		Stations: stations,
		Location: resolvedLocation,
	}

	fuelCache.Lock()
	fuelCache.entries[key] = fuelCacheEntry{data: data, expires: time.Now().Add(10 * time.Minute)}
	fuelCache.Unlock()

	return data, nil
}

// FetchFuelPrices fetches fresh prices for specific station IDs via the Tankerkönig prices.php endpoint.
// Tankerkönig returns `false` (not a number) for prices when a station is closed, so we use
// json.RawMessage to handle the mixed type safely.
func FetchFuelPrices(apiKey string, ids []string) (*FuelPricesData, error) {
	if len(ids) == 0 {
		return &FuelPricesData{Prices: map[string]FuelPrice{}}, nil
	}

	key := apiKey + ":" + strings.Join(ids, ",")
	fuelPriceCache.Lock()
	if e, ok := fuelPriceCache.entries[key]; ok && time.Now().Before(e.expires) {
		fuelPriceCache.Unlock()
		return e.data, nil
	}
	fuelPriceCache.Unlock()

	tkURL := fmt.Sprintf(
		"https://creativecommons.tankerkoenig.de/json/prices.php?ids=%s&apikey=%s",
		url.QueryEscape(strings.Join(ids, ",")),
		url.QueryEscape(apiKey),
	)
	resp, err := httpClient.Get(tkURL)
	if err != nil {
		return nil, fmt.Errorf("tankerkoenig prices fetch: %w", err)
	}
	defer resp.Body.Close()

	// prices.php returns float or boolean false for price fields → use RawMessage
	var raw struct {
		OK     bool `json:"ok"`
		Prices map[string]struct {
			Status string          `json:"status"`
			E5     json.RawMessage `json:"e5"`
			E10    json.RawMessage `json:"e10"`
			Diesel json.RawMessage `json:"diesel"`
		} `json:"prices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("tankerkoenig prices decode: %w", err)
	}
	if !raw.OK {
		return nil, fmt.Errorf("tankerkoenig prices api error (check API key)")
	}

	data := &FuelPricesData{Prices: make(map[string]FuelPrice, len(raw.Prices))}
	for id, p := range raw.Prices {
		data.Prices[id] = FuelPrice{
			E5:     rawToFloat(p.E5),
			E10:    rawToFloat(p.E10),
			Diesel: rawToFloat(p.Diesel),
			IsOpen: p.Status == "open",
		}
	}

	fuelPriceCache.Lock()
	fuelPriceCache.entries[key] = fuelPriceCacheEntry{data: data, expires: time.Now().Add(5 * time.Minute)}
	fuelPriceCache.Unlock()

	return data, nil
}

func rawToFloat(msg json.RawMessage) float64 {
	var f float64
	if err := json.Unmarshal(msg, &f); err == nil {
		return f
	}
	return 0
}
