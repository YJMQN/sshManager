package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config represents the persisted settings stored next to the executable.
type Config struct {
	DBPath string `json:"db_path"`
}

// configPath returns the config file path next to the exe.
func configPath() string {
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "config.json")
}

// loadConfig reads config.json next to the exe.
// Missing or broken file is silently ignored.
func loadConfig() *Config {
	cfg := &Config{}
	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(data, cfg)
	return cfg
}

// saveConfig writes config.json next to the exe.
func saveConfig(cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0644)
}
