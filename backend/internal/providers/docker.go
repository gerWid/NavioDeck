package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type DockerData struct {
	Running    int             `json:"running"`
	Stopped    int             `json:"stopped"`
	Total      int             `json:"total"`
	Containers []ContainerInfo `json:"containers"`
}

type ContainerInfo struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Image   string   `json:"image"`
	State   string   `json:"state"`
	Status  string   `json:"status"`
	Created int64    `json:"created"`
	Ports   []string `json:"ports"`
}

var dockerCache struct {
	sync.Mutex
	entries map[string]dockerCacheEntry
}

type dockerCacheEntry struct {
	data    *DockerData
	expires time.Time
}

func init() {
	dockerCache.entries = make(map[string]dockerCacheEntry)
}

// ValidateDockerEndpoint returns an error if the endpoint is not in the
// allowlist of permitted Docker socket paths. Only unix:// sockets pointing to
// /var/run/docker.sock are permitted. TCP/HTTP endpoints are intentionally
// rejected to prevent SSRF via the Docker API.
func ValidateDockerEndpoint(endpoint string) error {
	// Default empty string maps to the standard socket and is fine.
	if endpoint == "" || endpoint == "unix:///var/run/docker.sock" {
		return nil
	}
	if strings.HasPrefix(endpoint, "unix://") {
		path := strings.TrimPrefix(endpoint, "unix://")
		if path == "/var/run/docker.sock" {
			return nil
		}
		return fmt.Errorf("docker endpoint unix socket path %q is not allowed; only /var/run/docker.sock is permitted", path)
	}
	return fmt.Errorf("docker endpoint %q is not allowed; only unix:///var/run/docker.sock is permitted", endpoint)
}

func FetchDocker(endpoint string, showAll bool) (*DockerData, error) {
	if err := ValidateDockerEndpoint(endpoint); err != nil {
		return nil, err
	}

	key := fmt.Sprintf("%s:%v", endpoint, showAll)
	dockerCache.Lock()
	if e, ok := dockerCache.entries[key]; ok && time.Now().Before(e.expires) {
		dockerCache.Unlock()
		return e.data, nil
	}
	dockerCache.Unlock()

	client := dockerHTTPClient(endpoint)
	query := "/containers/json"
	if showAll {
		query += "?all=1"
	}
	baseURL := dockerBaseURL(endpoint)

	data, err := fetchDockerFromClient(client, baseURL+query)
	if err != nil {
		return nil, err
	}

	dockerCache.Lock()
	dockerCache.entries[key] = dockerCacheEntry{data: data, expires: time.Now().Add(30 * time.Second)}
	dockerCache.Unlock()

	return data, nil
}

// fetchDockerFromURL is used by tests: it calls the given URL with the default client.
func fetchDockerFromURL(url string, showAll bool) (*DockerData, error) {
	query := url + "/containers/json"
	if showAll {
		query += "?all=1"
	}
	return fetchDockerFromClient(&http.Client{Timeout: 10 * time.Second}, query)
}

func fetchDockerFromClient(client *http.Client, url string) (*DockerData, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("docker api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("docker api returned %d", resp.StatusCode)
	}

	var raw []struct {
		ID      string   `json:"Id"`
		Names   []string `json:"Names"`
		Image   string   `json:"Image"`
		State   string   `json:"State"`
		Status  string   `json:"Status"`
		Created int64    `json:"Created"`
		Ports   []struct {
			IP          string `json:"IP"`
			PrivatePort int    `json:"PrivatePort"`
			PublicPort  int    `json:"PublicPort"`
			Type        string `json:"Type"`
		} `json:"Ports"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("docker decode: %w", err)
	}

	data := &DockerData{}
	for _, c := range raw {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}
		ports := []string{}
		for _, p := range c.Ports {
			if p.PublicPort > 0 {
				ports = append(ports, fmt.Sprintf("%s:%d→%d", p.Type, p.PublicPort, p.PrivatePort))
			}
		}
		if c.State == "running" {
			data.Running++
		} else {
			data.Stopped++
		}
		data.Total++
		data.Containers = append(data.Containers, ContainerInfo{
			ID:      c.ID[:min(12, len(c.ID))],
			Name:    name,
			Image:   c.Image,
			State:   c.State,
			Status:  c.Status,
			Created: c.Created,
			Ports:   ports,
		})
	}
	return data, nil
}

func dockerHTTPClient(endpoint string) *http.Client {
	sockPath := "/var/run/docker.sock"
	if strings.HasPrefix(endpoint, "unix://") {
		sockPath = strings.TrimPrefix(endpoint, "unix://")
	} else if strings.HasPrefix(endpoint, "tcp://") || strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		return &http.Client{Timeout: 10 * time.Second}
	}
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", sockPath)
			},
		},
	}
}

func dockerBaseURL(endpoint string) string {
	if strings.HasPrefix(endpoint, "tcp://") {
		return "http://" + strings.TrimPrefix(endpoint, "tcp://")
	}
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		return endpoint
	}
	// unix socket — hostname is ignored but must be non-empty
	return "http://docker"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
