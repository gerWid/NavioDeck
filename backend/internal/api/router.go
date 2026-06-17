package api

import (
	"net/http"
	"path/filepath"
	"strings"

	"naviodeck/internal/auth"
	"naviodeck/internal/config"
	"naviodeck/internal/ws"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func NewRouter(store *config.Store, hub *ws.Hub, dataDir string, password string, sessions *auth.SessionStore, fuelKey string) *chi.Mux {
	r := chi.NewRouter()
	r.Use(redactedLogger)
	r.Use(middleware.Recoverer)
	// gzip-compress text responses (HTML/JS/CSS/JSON). Without this the server
	// ships large assets (JS bundles, news JSON) uncompressed, which can stall
	// over low-bandwidth/low-MTU links such as some VPN tunnels — the page then
	// "loads forever". Other web servers (nginx, …) compress by default; this
	// brings the Go server in line. Already-compressed images are skipped.
	r.Use(middleware.Compress(5))
	r.Use(middleware.RequestSize(1 << 20)) // 1 MB JSON body limit

	// Inject modern security headers
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; img-src 'self' data: blob: https: http:; script-src 'self' 'unsafe-inline'; connect-src 'self' ws: wss:; frame-ancestors 'none'; object-src 'none'")
			w.Header().Set("Permissions-Policy", "geolocation=(), camera=(), microphone=(), payment=(), usb=()")
			next.ServeHTTP(w, r)
		})
	})

	// CSRF prevention: validate Content-Type header on state-changing requests with body (POST, PUT, DELETE)
	// (Excluding multipart uploads which use multipart/form-data)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.Method == "POST" || req.Method == "PUT" || req.Method == "DELETE" {
				ct := req.Header.Get("Content-Type")
				// Skip validation for files upload endpoints or check for multipart
				isUpload := strings.HasPrefix(req.URL.Path, "/api/wallpapers") || strings.HasPrefix(req.URL.Path, "/api/icons")
				if isUpload {
					if !strings.HasPrefix(ct, "multipart/form-data") {
						http.Error(w, "invalid content type for upload", http.StatusBadRequest)
						return
					}
				} else {
					if !strings.HasPrefix(ct, "application/json") {
						http.Error(w, "invalid content-type: application/json required", http.StatusUnsupportedMediaType)
						return
					}
				}
			}
			next.ServeHTTP(w, req)
		})
	})

	r.Use(cors.Handler(cors.Options{
		AllowOriginFunc: func(r *http.Request, origin string) bool {
			if origin == "" {
				return true
			}
			// Allow same-origin (matches Host)
			if origin == "http://"+r.Host || origin == "https://"+r.Host {
				return true
			}
			// Allow development origins
			devs := []string{
				"http://localhost:5173", "http://127.0.0.1:5173",
				"http://localhost:4173", "http://127.0.0.1:4173",
				"http://localhost:8080", "http://127.0.0.1:8080",
				"http://localhost:3080", "http://127.0.0.1:3080",
			}
			for _, d := range devs {
				if origin == d {
					return true
				}
			}
			return false
		},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
	}))

	h := &handlers{store: store, hub: hub, dataDir: dataDir, password: password, sessions: sessions, fuelKey: fuelKey}

	// requireAuth returns 401 when password protection is active and the session is invalid.
	requireAuth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if password == "" {
				next.ServeHTTP(w, r)
				return
			}
			c, err := r.Cookie("session")
			if err != nil || !sessions.IsValid(c.Value) {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	r.Route("/api", func(r chi.Router) {
		// Always public — needed before login exists
		r.Get("/auth", h.authInfo)
		r.With(loginRateLimiter).Post("/login", h.login)
		r.Post("/logout", h.logout)

		// Everything else requires auth
		r.Group(func(r chi.Router) {
			r.Use(requireAuth)

			r.Get("/config", h.getConfig)
			r.Put("/config/theme", h.putTheme)

			r.Get("/widgets", h.getWidgets)
			r.Post("/widgets", h.postWidget)
			r.Put("/widgets/{id}", h.putWidget)
			r.Delete("/widgets/{id}", h.deleteWidget)
			r.Put("/widgets/layout", h.putLayout)

			r.Get("/weather", h.getWeather)
			r.Get("/news", h.getNews)
			r.Get("/docker", h.getDocker)
			r.Get("/garbage", h.getGarbage)
			r.Get("/calendar", h.getCalendar)
			r.Get("/rss", h.getRSS)
			r.Get("/fuel", h.getFuel)
			r.Get("/fuel/prices", h.getFuelPrices)

			r.Get("/wallpapers", h.getWallpapers)
			r.Post("/wallpapers", h.uploadWallpaper)
			r.Delete("/wallpapers/{filename}", h.deleteWallpaper)

			r.Get("/icons", h.getIcons)
			r.Post("/icons", h.uploadIcon)
			r.Delete("/icons/{filename}", h.deleteIcon)
		})
	})

	r.With(requireAuth).Get("/ws", hub.ServeWS)

	// Serve uploaded wallpaper files
	wallpaperDir := filepath.Join(dataDir, "wallpapers")
	r.Get("/wallpapers/{filename}", func(w http.ResponseWriter, req *http.Request) {
		name := filepath.Base(chi.URLParam(req, "filename"))
		http.ServeFile(w, req, filepath.Join(wallpaperDir, name))
	})

	// Serve uploaded icon files
	iconDir := filepath.Join(dataDir, "icons")
	r.Get("/icons/{filename}", func(w http.ResponseWriter, req *http.Request) {
		name := filepath.Base(chi.URLParam(req, "filename"))
		http.ServeFile(w, req, filepath.Join(iconDir, name))
	})

	return r
}
