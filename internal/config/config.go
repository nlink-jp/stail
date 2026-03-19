// Package config manages stail configuration files and profiles.
// Config files are stored at ~/.config/stail/config.json with 0600 permissions.
// The schema mirrors scat's profile-based structure.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// DefaultConfigDir is the directory relative to home that holds the config file.
	DefaultConfigDir = ".config/stail"
	// DefaultConfigFile is the config file name.
	DefaultConfigFile = "config.json"
	// ModeServer is the value of STAIL_MODE that enables server mode.
	ModeServer = "server"
	// ProviderSlack is the slack provider name.
	ProviderSlack = "slack"
)

// Profile holds the settings for a single named profile.
type Profile struct {
	Provider string `json:"provider,omitempty"`
	Token    string `json:"token,omitempty"`
	AppToken string `json:"app_token,omitempty"`
	Channel  string `json:"channel,omitempty"`
	Username string `json:"username,omitempty"`
}

// Config is the top-level configuration structure.
// Field names mirror scat's config schema for consistency.
type Config struct {
	CurrentProfile string             `json:"current_profile"`
	Profiles       map[string]Profile `json:"profiles"`
}

// DefaultConfigPath returns the default path for the config file.
func DefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, DefaultConfigDir, DefaultConfigFile), nil
}

// Load reads and parses the configuration from path.
func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config %q: %w", path, err)
	}
	defer f.Close()

	var cfg Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]Profile)
	}
	return &cfg, nil
}

// Save writes cfg to path, creating parent directories as needed.
// The file is created (or overwritten) with 0600 permissions.
func Save(cfg *Config, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// GetProfile returns the profile identified by name.
// If name is empty, the config's CurrentProfile is used.
func (c *Config) GetProfile(name string) (Profile, error) {
	if name == "" {
		name = c.CurrentProfile
	}
	p, ok := c.Profiles[name]
	if !ok {
		return Profile{}, fmt.Errorf("profile %q not found", name)
	}
	return p, nil
}

// DefaultConfig returns an initial configuration with a placeholder default profile.
func DefaultConfig() *Config {
	return &Config{
		CurrentProfile: "default",
		Profiles: map[string]Profile{
			"default": {
				Provider: ProviderSlack,
				Token:    "",
				Channel:  "",
			},
		},
	}
}

// DetectServerMode validates and returns whether server mode is active.
// Returns an error if STAIL_MODE is set to an unsupported value.
func DetectServerMode() (bool, error) {
	mode := os.Getenv("STAIL_MODE")
	switch mode {
	case "":
		return false, nil
	case ModeServer:
		return true, nil
	default:
		return false, fmt.Errorf("invalid STAIL_MODE value %q: must be %q or unset", mode, ModeServer)
	}
}

// BuildConfigFromEnv builds a Config from environment variables for server mode.
// STAIL_PROVIDER and STAIL_TOKEN are required.
func BuildConfigFromEnv() (*Config, error) {
	provider := os.Getenv("STAIL_PROVIDER")
	if provider == "" {
		return nil, fmt.Errorf("STAIL_PROVIDER is required in server mode")
	}
	token := os.Getenv("STAIL_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("STAIL_TOKEN is required in server mode")
	}
	p := Profile{
		Provider: provider,
		Token:    token,
		AppToken: os.Getenv("STAIL_APP_TOKEN"),
		Channel:  os.Getenv("STAIL_CHANNEL"),
	}
	return &Config{
		CurrentProfile: "server",
		Profiles:       map[string]Profile{"server": p},
	}, nil
}
