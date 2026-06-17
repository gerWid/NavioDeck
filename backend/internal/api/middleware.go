package api

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// redactedLogger is a chi-compatible middleware that logs requests like the
// default chi middleware.Logger but redacts the value of the "api_key" query
// parameter before writing the log line. This prevents API keys from appearing
// in server access logs.
func redactedLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()

		defer func() {
			logURI := redactQueryParam(r.RequestURI, "api_key")
			fmt.Printf("%s %s %s %d %dB %s\n",
				r.RemoteAddr,
				r.Method,
				logURI,
				ww.Status(),
				ww.BytesWritten(),
				time.Since(start),
			)
		}()

		next.ServeHTTP(ww, r)
	})
}

// redactQueryParam returns the raw URI with the named query parameter's value
// replaced by "[REDACTED]". If the URI cannot be parsed or the parameter is not
// present, the original string is returned unchanged.
func redactQueryParam(rawURI, param string) string {
	u, err := url.ParseRequestURI(rawURI)
	if err != nil {
		return rawURI
	}
	q := u.Query()
	if _, ok := q[param]; !ok {
		return rawURI
	}
	q.Set(param, "[REDACTED]")
	u.RawQuery = q.Encode()
	return u.String()
}

// ---- login rate limiting ----

const (
	loginRateWindow   = 10 * time.Minute
	loginRateMaxTries = 10
)

type loginAttempt struct {
	timestamps []time.Time
}

var loginRateMu sync.Mutex
var loginRateMap = make(map[string]*loginAttempt)

// loginRateLimiter is a middleware that applies IP-based rate limiting to the
// /api/login endpoint. It allows at most loginRateMaxTries attempts per IP
// within a loginRateWindow sliding window and returns HTTP 429 when exceeded.
func loginRateLimiter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := remoteIP(r)
		now := time.Now()
		cutoff := now.Add(-loginRateWindow)

		loginRateMu.Lock()
		entry, ok := loginRateMap[ip]
		if !ok {
			entry = &loginAttempt{}
			loginRateMap[ip] = entry
		}
		// Evict timestamps outside the sliding window.
		valid := entry.timestamps[:0]
		for _, ts := range entry.timestamps {
			if ts.After(cutoff) {
				valid = append(valid, ts)
			}
		}
		entry.timestamps = valid

		if len(entry.timestamps) >= loginRateMaxTries {
			loginRateMu.Unlock()
			w.Header().Set("Retry-After", "600")
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many login attempts, try again later"})
			return
		}
		entry.timestamps = append(entry.timestamps, now)
		loginRateMu.Unlock()

		next.ServeHTTP(w, r)
	})
}

// remoteIP extracts the client IP from the request, stripping the port.
func remoteIP(r *http.Request) string {
	addr := r.RemoteAddr
	// RemoteAddr is "host:port" for TCP connections.
	if i := strings.LastIndex(addr, ":"); i >= 0 {
		return addr[:i]
	}
	return addr
}
