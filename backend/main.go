package main

import (
	"flag"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"naviodeck/internal/api"
	"naviodeck/internal/auth"
	"naviodeck/internal/config"
	"naviodeck/internal/watcher"
	"naviodeck/internal/ws"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dataDir := flag.String("data", "./data", "data directory")
	staticDir := flag.String("static", "./static", "frontend build directory")
	flag.Parse()

	if err := os.MkdirAll(*dataDir, 0750); err != nil {
		log.Fatalf("create data dir: %v", err)
	}

	// Server-side config/secrets from /data/config.yaml (host volume, not in image).
	// Env vars act as a fallback so existing deployments keep working.
	appCfg := config.LoadAppConfig(*dataDir)

	password := appCfg.DashboardPassword
	if password == "" {
		password = os.Getenv("DASHBOARD_PASSWORD")
	}
	if password == "" {
		log.Println("auth: disabled (no dashboard_password configured)")
	} else {
		log.Println("auth: password protection enabled")
	}

	fuelAPIKey := appCfg.TankerkoenigAPIKey
	if fuelAPIKey == "" {
		fuelAPIKey = os.Getenv("TANKERKOENIG_API_KEY")
	}

	sessions := auth.NewSessionStore()

	hub := ws.NewHub()

	store, err := config.NewStore(*dataDir, func(d *config.Dashboard) {
		hub.Broadcast(ws.Message{Type: "config", Payload: d})
	})
	if err != nil {
		log.Fatalf("init config store: %v", err)
	}

	if err := watcher.Watch(store.Path(), func() {
		if err := store.Reload(); err != nil {
			log.Printf("reload config: %v", err)
		}
	}); err != nil {
		log.Printf("file watcher unavailable: %v", err)
	}

	r := api.NewRouter(store, hub, *dataDir, password, sessions, fuelAPIKey)

	// Serve SvelteKit static build; fall back to index.html for SPA routing.
	// url.PathUnescape + filepath.Join guarantee no path-traversal outside staticDir.
	fs := http.FileServer(http.Dir(*staticDir))
	r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
		cleaned, err := url.PathUnescape(req.URL.Path)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		diskPath := filepath.Join(*staticDir, filepath.Clean("/"+cleaned))
		if _, err := os.Stat(diskPath); os.IsNotExist(err) {
			http.ServeFile(w, req, filepath.Join(*staticDir, "index.html"))
			return
		}
		fs.ServeHTTP(w, req)
	})

	srv := &http.Server{
		Addr:         *addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second, // higher for long-poll / WS upgrade
		IdleTimeout:  120 * time.Second,
	}
	log.Printf("naviodeck listening on %s", *addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
