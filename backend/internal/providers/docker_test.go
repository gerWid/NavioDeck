package providers

import (
	"net/http"
	"testing"
)

// ---- dockerBaseURL ----

func TestDockerBaseURL(t *testing.T) {
	tests := []struct {
		endpoint, want string
	}{
		{"unix:///var/run/docker.sock", "http://docker"},
		{"unix:///run/docker.sock", "http://docker"},
		{"tcp://192.168.1.1:2375", "http://192.168.1.1:2375"},
		{"http://192.168.1.1:2375", "http://192.168.1.1:2375"},
		{"https://192.168.1.1:2376", "https://192.168.1.1:2376"},
		{"http://host.docker.internal:2375", "http://host.docker.internal:2375"},
	}
	for _, tt := range tests {
		got := dockerBaseURL(tt.endpoint)
		if got != tt.want {
			t.Errorf("dockerBaseURL(%q) = %q, want %q", tt.endpoint, got, tt.want)
		}
	}
}

// ---- dockerHTTPClient ----

func TestDockerHTTPClient_UnixSocket(t *testing.T) {
	c := dockerHTTPClient("unix:///var/run/docker.sock")
	if c.Timeout != 10e9 {
		t.Errorf("timeout = %v, want 10s", c.Timeout)
	}
	if c.Transport == nil {
		t.Error("unix socket client must have a custom Transport")
	}
	if _, ok := c.Transport.(*http.Transport); !ok {
		t.Error("transport must be *http.Transport")
	}
}

func TestDockerHTTPClient_TCP(t *testing.T) {
	c := dockerHTTPClient("tcp://192.168.1.1:2375")
	if c.Transport != nil {
		t.Error("tcp client must use default transport (nil)")
	}
}

func TestDockerHTTPClient_HTTP(t *testing.T) {
	c := dockerHTTPClient("http://192.168.1.1:2375")
	if c.Transport != nil {
		t.Error("http client must use default transport (nil)")
	}
}

func TestDockerHTTPClient_DefaultFallback(t *testing.T) {
	// Unknown scheme falls through to the unix-socket path with default sock
	c := dockerHTTPClient("/var/run/docker.sock")
	if c.Transport == nil {
		t.Error("default client must have a custom Transport (unix socket)")
	}
}

// ---- FetchDocker via mock HTTP server ----

func TestFetchDocker_MockServer(t *testing.T) {
	// Build a fake Docker API response with one running and one exited container.
	const dockerJSON = `[
		{
			"Id":    "abc123def456",
			"Names": ["/myapp"],
			"Image": "nginx:latest",
			"State": "running",
			"Status": "Up 2 hours",
			"Created": 1700000000,
			"Ports": [{"IP":"0.0.0.0","PrivatePort":80,"PublicPort":8080,"Type":"tcp"}]
		},
		{
			"Id":    "def789abc012",
			"Names": ["/stopped"],
			"Image": "redis:7",
			"State": "exited",
			"Status": "Exited (0) 1 hour ago",
			"Created": 1699990000,
			"Ports": []
		}
	]`

	srv := mockServer(t, dockerJSON)
	defer srv.Close()

	// Clear cache so our test hits the mock server
	dockerCache.Lock()
	dockerCache.entries = make(map[string]dockerCacheEntry)
	dockerCache.Unlock()

	data, err := fetchDockerFromURL(srv.URL, true)
	if err != nil {
		t.Fatalf("FetchDocker error: %v", err)
	}

	if data.Total != 2 {
		t.Errorf("total = %d, want 2", data.Total)
	}
	if data.Running != 1 {
		t.Errorf("running = %d, want 1", data.Running)
	}
	if data.Stopped != 1 {
		t.Errorf("stopped = %d, want 1", data.Stopped)
	}
	if len(data.Containers) != 2 {
		t.Fatalf("containers len = %d, want 2", len(data.Containers))
	}

	c := data.Containers[0]
	if c.Name != "myapp" {
		t.Errorf("container name = %q, want %q", c.Name, "myapp")
	}
	// ID truncated to 12 chars
	if len(c.ID) != 12 {
		t.Errorf("container ID len = %d, want 12", len(c.ID))
	}
	// Port should be formatted as "tcp:8080→80"
	if len(c.Ports) != 1 || c.Ports[0] != "tcp:8080→80" {
		t.Errorf("ports = %v, want [tcp:8080→80]", c.Ports)
	}

	// Stopped container must have empty (not nil) ports slice
	stopped := data.Containers[1]
	if stopped.Ports == nil {
		t.Error("stopped container ports must be [] not nil")
	}
}
