package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// AppConfig holds server-side settings and secrets. It is read once at startup
// from <dataDir>/config.yaml — a file that lives in the mounted data volume on
// the host, so it is never baked into the image nor committed to the repo.
type AppConfig struct {
	TankerkoenigAPIKey string `yaml:"tankerkoenig_api_key"`
	DashboardPassword  string `yaml:"dashboard_password"`
}

// LoadAppConfig reads <dataDir>/config.yaml. A missing or unparsable file yields
// a zero-value config (all fields empty) rather than an error, so the app still
// starts when no server config is provided.
func LoadAppConfig(dataDir string) AppConfig {
	var c AppConfig
	raw, err := os.ReadFile(filepath.Join(dataDir, "config.yaml"))
	if err != nil {
		return c
	}
	_ = yaml.Unmarshal(raw, &c)
	return c
}
