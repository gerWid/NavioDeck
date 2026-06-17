package api

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"naviodeck/internal/auth"
	"naviodeck/internal/config"
	"naviodeck/internal/providers"
	"naviodeck/internal/ws"

	"github.com/go-chi/chi/v5"
)

type handlers struct {
	store    *config.Store
	hub      *ws.Hub
	dataDir  string
	password string
	sessions *auth.SessionStore
	fuelKey  string // server-side Tankerkönig key from config.yaml/env
}

// ---- auth ----

func (h *handlers) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	// constant-time compare + fixed delay defeats timing attacks and brute-force
	match := h.password != "" &&
		subtle.ConstantTimeCompare([]byte(req.Password), []byte(h.password)) == 1
	if !match {
		time.Sleep(500 * time.Millisecond)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Falsches Passwort"})
		return
	}
	token, err := h.sessions.Create()
	if err != nil {
		internalError(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(auth.SessionTTL.Seconds()),
	})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *handlers) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("session"); err == nil {
		h.sessions.Delete(c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
	w.WriteHeader(http.StatusNoContent)
}

// authInfo returns whether password protection is enabled.
// This endpoint is always public so the frontend knows whether to show a login form.
func (h *handlers) authInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"enabled": h.password != ""})
}

// ---- helpers ----

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func readJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// internalError logs err and writes a generic 500 response to the client so
// internal implementation details are never exposed in API responses.
func internalError(w http.ResponseWriter, err error) {
	log.Printf("internal error: %v", err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
}

func (h *handlers) broadcast() {
	h.hub.Broadcast(ws.Message{Type: "config", Payload: h.store.Get()})
}

func (h *handlers) getConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.store.Get())
}

func (h *handlers) putTheme(w http.ResponseWriter, r *http.Request) {
	var t config.Theme
	if err := readJSON(r, &t); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := h.store.UpdateTheme(t); err != nil {
		internalError(w, err)
		return
	}
	h.broadcast()
	writeJSON(w, http.StatusOK, h.store.Get().Theme)
}

func (h *handlers) getWidgets(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.store.Get().Widgets)
}

func (h *handlers) postWidget(w http.ResponseWriter, r *http.Request) {
	var widget config.Widget
	if err := readJSON(r, &widget); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := h.store.UpsertWidget(widget); err != nil {
		internalError(w, err)
		return
	}
	h.broadcast()
	writeJSON(w, http.StatusCreated, widget)
}

func (h *handlers) putWidget(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var widget config.Widget
	if err := readJSON(r, &widget); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	widget.ID = id
	if err := h.store.UpsertWidget(widget); err != nil {
		internalError(w, err)
		return
	}
	h.broadcast()
	writeJSON(w, http.StatusOK, widget)
}

func (h *handlers) deleteWidget(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.store.DeleteWidget(id); err != nil {
		internalError(w, err)
		return
	}
	h.broadcast()
	w.WriteHeader(http.StatusNoContent)
}

func (h *handlers) putLayout(w http.ResponseWriter, r *http.Request) {
	type layoutItem struct {
		ID string `json:"id"`
		config.Position
	}
	var items []layoutItem
	if err := readJSON(r, &items); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	positions := make([]struct {
		ID string `json:"id"`
		config.Position
	}, len(items))
	for i, item := range items {
		positions[i].ID = item.ID
		positions[i].Position = item.Position
	}
	if err := h.store.UpdateLayouts(positions); err != nil {
		internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handlers) getWeather(w http.ResponseWriter, r *http.Request) {
	city := r.URL.Query().Get("city")
	if city == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "city required"})
		return
	}
	units := r.URL.Query().Get("units")
	if units == "" {
		units = "celsius"
	}
	forecastDays := 7
	if fd := r.URL.Query().Get("forecast_days"); fd != "" {
		if v, err := strconv.Atoi(fd); err == nil {
			forecastDays = v
		}
	}
	data, err := providers.FetchWeather(city, units, forecastDays)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (h *handlers) getDocker(w http.ResponseWriter, r *http.Request) {
	endpoint := r.URL.Query().Get("endpoint")
	if endpoint == "" {
		endpoint = "unix:///var/run/docker.sock"
	}
	if err := providers.ValidateDockerEndpoint(endpoint); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	showStopped := r.URL.Query().Get("show_stopped") == "true"
	maxItems := 10
	if v, err := strconv.Atoi(r.URL.Query().Get("max")); err == nil && v > 0 {
		maxItems = v
	}

	// Always fetch all containers so the running/stopped/total counts are
	// accurate, regardless of how many we end up displaying.
	data, err := providers.FetchDocker(endpoint, true)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	// Build a trimmed response (counts + only the containers actually shown).
	// On hosts with many containers the full list is large and stalls over slow
	// links; capping it server-side keeps the payload small. We copy into a new
	// struct rather than mutate data.Containers, which is shared via the cache.
	resp := struct {
		Running    int                       `json:"running"`
		Stopped    int                       `json:"stopped"`
		Total      int                       `json:"total"`
		Containers []providers.ContainerInfo `json:"containers"`
	}{Running: data.Running, Stopped: data.Stopped, Total: data.Total}
	for _, c := range data.Containers {
		if !showStopped && c.State != "running" {
			continue
		}
		resp.Containers = append(resp.Containers, c)
		if len(resp.Containers) >= maxItems {
			break
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *handlers) getGarbage(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	if source == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "source (url or filename) required"})
		return
	}
	daysAhead := 30
	maxItems := 20
	if v, err := strconv.Atoi(r.URL.Query().Get("days")); err == nil && v > 0 {
		daysAhead = v
	}
	if v, err := strconv.Atoi(r.URL.Query().Get("max")); err == nil && v > 0 {
		maxItems = v
	}
	data, err := providers.FetchGarbage(source, h.dataDir, daysAhead, maxItems)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, data)
}

// fuelAPIKey resolves the Tankerkönig key by precedence: an explicit per-widget
// key from the query wins; otherwise the server-side key loaded from
// /data/config.yaml (or the TANKERKOENIG_API_KEY env fallback). This keeps the
// secret out of the committed dashboard config and out of the browser.
func (h *handlers) fuelAPIKey(q string) string {
	if q != "" {
		return q
	}
	return h.fuelKey
}

func (h *handlers) getFuel(w http.ResponseWriter, r *http.Request) {
	apiKey := h.fuelAPIKey(r.URL.Query().Get("api_key"))
	if apiKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "api_key required (set widget config or tankerkoenig_api_key in config.yaml)"})
		return
	}
	location := r.URL.Query().Get("location")
	var lat, lng, radius float64
	fmt.Sscanf(r.URL.Query().Get("lat"), "%f", &lat)
	fmt.Sscanf(r.URL.Query().Get("lng"), "%f", &lng)
	fmt.Sscanf(r.URL.Query().Get("radius"), "%f", &radius)
	sortBy := r.URL.Query().Get("sort")
	maxStations := 5
	if v, err := strconv.Atoi(r.URL.Query().Get("max")); err == nil && v > 0 {
		maxStations = v
	}
	data, err := providers.FetchFuel(apiKey, location, lat, lng, radius, sortBy, maxStations)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (h *handlers) getCalendar(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	if source == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "source (url or filename) required"})
		return
	}
	daysAhead := 30
	maxItems := 20
	if v, err := strconv.Atoi(r.URL.Query().Get("days")); err == nil && v > 0 {
		daysAhead = v
	}
	if v, err := strconv.Atoi(r.URL.Query().Get("max")); err == nil && v > 0 {
		maxItems = v
	}
	data, err := providers.FetchCalendar(source, h.dataDir, daysAhead, maxItems)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (h *handlers) getFuelPrices(w http.ResponseWriter, r *http.Request) {
	apiKey := h.fuelAPIKey(r.URL.Query().Get("api_key"))
	if apiKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "api_key required (set widget config or tankerkoenig_api_key in config.yaml)"})
		return
	}
	idsParam := r.URL.Query().Get("ids")
	if idsParam == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "ids required"})
		return
	}
	ids := strings.Split(idsParam, ",")
	data, err := providers.FetchFuelPrices(apiKey, ids)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, data)
}

var allowedExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true,
	".gif": true, ".webp": true, ".avif": true,
}

var allowedMIME = map[string]bool{
	"image/jpeg": true, "image/png": true, "image/gif": true,
	"image/webp": true, "image/avif": true,
}

// checkImageFile reads up to 512 bytes to detect the actual MIME type and
// returns an io.Reader that includes those bytes for the subsequent copy.
func checkImageFile(f io.Reader) (io.Reader, error) {
	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return nil, err
	}
	mime := http.DetectContentType(buf[:n])
	// http.DetectContentType may return "image/jpeg; charset=…" — strip params
	if i := len(mime); i > 0 {
		for j := 0; j < len(mime); j++ {
			if mime[j] == ';' {
				mime = mime[:j]
				break
			}
		}
	}
	if !allowedMIME[mime] {
		return nil, fmt.Errorf("unsupported content type: %s", mime)
	}
	return io.MultiReader(strings.NewReader(string(buf[:n])), f), nil
}

type wallpaperEntry struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func (h *handlers) getWallpapers(w http.ResponseWriter, r *http.Request) {
	dir := filepath.Join(h.dataDir, "wallpapers")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusOK, []wallpaperEntry{})
			return
		}
		internalError(w, err)
		return
	}
	result := make([]wallpaperEntry, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if allowedExts[ext] {
			result = append(result, wallpaperEntry{Name: e.Name(), URL: "/wallpapers/" + e.Name()})
		}
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *handlers) uploadWallpaper(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid multipart form"})
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file required"})
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedExts[ext] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported file type"})
		return
	}
	checked, err := checkImageFile(file)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	dir := filepath.Join(h.dataDir, "wallpapers")
	if err := os.MkdirAll(dir, 0750); err != nil {
		internalError(w, err)
		return
	}

	name := filepath.Base(header.Filename)
	fullPath := filepath.Join(dir, name)
	if _, err := os.Stat(fullPath); err == nil {
		// File already exists. Append Unix timestamp to prevent silent overwrite.
		ext := filepath.Ext(name)
		baseName := strings.TrimSuffix(name, ext)
		name = fmt.Sprintf("%s_%d%s", baseName, time.Now().Unix(), ext)
		fullPath = filepath.Join(dir, name)
	}

	dst, err := os.Create(fullPath)
	if err != nil {
		internalError(w, err)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, checked); err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, wallpaperEntry{Name: name, URL: "/wallpapers/" + name})
}

func (h *handlers) deleteWallpaper(w http.ResponseWriter, r *http.Request) {
	name := filepath.Base(chi.URLParam(r, "filename"))
	path := filepath.Join(h.dataDir, "wallpapers", name)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- news proxy ----

func (h *handlers) getNews(w http.ResponseWriter, r *http.Request) {
	apiBase := "https://www.tagesschau.de/api2u/news/"
	params := url.Values{}
	if v := r.URL.Query().Get("regions"); v != "" {
		params.Set("regions", v)
	}
	if v := r.URL.Query().Get("ressort"); v != "" {
		params.Set("ressort", v)
	}
	if v := r.URL.Query().Get("pageSize"); v != "" {
		params.Set("pageSize", v)
	}
	reqURL := apiBase
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, reqURL, nil)
	if err != nil {
		internalError(w, err)
		return
	}
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body) //nolint:errcheck
}

// ---- rss proxy ----

func (h *handlers) getRSS(w http.ResponseWriter, r *http.Request) {
	sourceURLs := r.URL.Query()["source"]
	names := r.URL.Query()["name"]
	maxItems := 30
	if v := r.URL.Query().Get("max_items"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxItems = n
		}
	}

	sources := make([]struct{ URL, Name string }, len(sourceURLs))
	for i, u := range sourceURLs {
		name := ""
		if i < len(names) {
			name = names[i]
		}
		sources[i] = struct{ URL, Name string }{URL: u, Name: name}
	}

	items := providers.FetchRssMulti(sources, maxItems)
	if items == nil {
		items = []providers.RssItem{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// ---- icons (same pattern as wallpapers) ----

type iconEntry struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func (h *handlers) getIcons(w http.ResponseWriter, r *http.Request) {
	dir := filepath.Join(h.dataDir, "icons")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusOK, []iconEntry{})
			return
		}
		internalError(w, err)
		return
	}
	result := make([]iconEntry, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if allowedExts[ext] {
			result = append(result, iconEntry{Name: e.Name(), URL: "/icons/" + e.Name()})
		}
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *handlers) uploadIcon(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid multipart form"})
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file required"})
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedExts[ext] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported file type"})
		return
	}
	checked, err := checkImageFile(file)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	dir := filepath.Join(h.dataDir, "icons")
	if err := os.MkdirAll(dir, 0750); err != nil {
		internalError(w, err)
		return
	}

	name := filepath.Base(header.Filename)
	fullPath := filepath.Join(dir, name)
	if _, err := os.Stat(fullPath); err == nil {
		// File already exists. Append Unix timestamp to prevent silent overwrite.
		ext := filepath.Ext(name)
		baseName := strings.TrimSuffix(name, ext)
		name = fmt.Sprintf("%s_%d%s", baseName, time.Now().Unix(), ext)
		fullPath = filepath.Join(dir, name)
	}

	dst, err := os.Create(fullPath)
	if err != nil {
		internalError(w, err)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, checked); err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, iconEntry{Name: name, URL: "/icons/" + name})
}

func (h *handlers) deleteIcon(w http.ResponseWriter, r *http.Request) {
	name := filepath.Base(chi.URLParam(r, "filename"))
	path := filepath.Join(h.dataDir, "icons", name)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
