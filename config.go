package main

import (
	"golang.org/x/sys/windows/registry"
)

const (
	regKey  = `Software\SSH Manager`
	regVal  = "DBPath"
)

// loadConfig reads database path from Windows Registry.
// If the key or value is missing, returns empty string.
func loadConfig() *Config {
	cfg := &Config{}
	k, err := registry.OpenKey(registry.CURRENT_USER, regKey, registry.QUERY_VALUE)
	if err != nil {
		return cfg // key doesn't exist yet
	}
	defer k.Close()

	s, _, err := k.GetStringValue(regVal)
	if err == nil {
		cfg.DBPath = s
	}
	return cfg
}

// saveConfig writes database path to Windows Registry.
func saveConfig(cfg *Config) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, regKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	return k.SetStringValue(regVal, cfg.DBPath)
}

// Config represents user settings stored in Windows Registry.
type Config struct {
	DBPath string
}
