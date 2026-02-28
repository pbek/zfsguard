// Package config provides configuration handling for zfsguard.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is the root configuration for zfsguard.
type Config struct {
	Monitor  MonitorConfig  `yaml:"monitor"`
	Notify   NotifyConfig   `yaml:"notify"`
	Defaults DefaultsConfig `yaml:"defaults"`
}

// MonitorConfig holds settings for the monitoring service.
type MonitorConfig struct {
	IntervalMinutes int      `yaml:"interval_minutes"`
	CheckZFS        bool     `yaml:"check_zfs"`
	CheckSMART      bool     `yaml:"check_smart"`
	SMARTDevices    []string `yaml:"smart_devices"`
}

// NotifyConfig holds notification service settings.
type NotifyConfig struct {
	// Shoutrrr URLs for remote notification services.
	// Examples:
	//   - "discord://token@id"
	//   - "telegram://token@telegram?channels=channel"
	//   - "pushover://shoutrrr:token@user"
	//   - "smtp://user:pass@host:port/?to=recipient"
	//   - "gotify://host/token"
	//   - "ntfy://ntfy.sh/topic"
	ShoutrrrURLs []string `yaml:"shoutrrr_urls"`

	// Desktop enables local Linux desktop notifications via notify-send / D-Bus.
	Desktop bool `yaml:"desktop"`
}

// DefaultsConfig holds default values for snapshot operations.
type DefaultsConfig struct {
	SnapshotPrefix string `yaml:"snapshot_prefix"`
}

// DefaultConfig returns a config with sane defaults.
func DefaultConfig() Config {
	return Config{
		Monitor: MonitorConfig{
			IntervalMinutes: 60,
			CheckZFS:        true,
			CheckSMART:      true,
		},
		Notify: NotifyConfig{
			Desktop: true,
		},
		Defaults: DefaultsConfig{
			SnapshotPrefix: "zfsguard",
		},
	}
}

// Load reads the config from the given path. If the file does not exist,
// it returns the default configuration.
func Load(path string) (Config, error) {
	cfg := DefaultConfig()

	if path == "" {
		path = defaultPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("failed to read config %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("failed to parse config %s: %w", path, err)
	}
	return cfg, nil
}

func defaultPath() string {
	// Check XDG_CONFIG_HOME first, then fall back to /etc
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "zfsguard", "config.yaml")
	}
	home, err := os.UserHomeDir()
	if err == nil {
		p := filepath.Join(home, ".config", "zfsguard", "config.yaml")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "/etc/zfsguard/config.yaml"
}
