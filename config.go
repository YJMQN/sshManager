package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config represents user settings stored in %APPDATA%.
type Config struct {
	DBPath string `json:"db_path"`
}

// configPath returns %APPDATA%\SSH Manager\config.json
// This is the standard Windows per-user app data directory.
func configPath() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		// Fallback: next to exe (shouldn't happen on real Windows)
		exe, _ := os.Executable()
		appData = filepath.Dir(exe)
	}
	return filepath.Join(appData, "SSH Manager", "config.json")
}

// loadConfig reads config from %APPDATA%.
// File not found is silently ignored (first launch).
func loadConfig() *Config {
	cfg := &Config{}
	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(data, cfg)
	return cfg
}

// saveConfig writes config to %APPDATA%.
func saveConfig(cfg *Config) error {
	path := configPath()
	os.MkdirAll(filepath.Dir(path), 0755)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
