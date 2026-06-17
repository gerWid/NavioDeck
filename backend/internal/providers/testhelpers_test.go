package providers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockServer creates an httptest.Server that always returns the given JSON body.
func mockServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
}
