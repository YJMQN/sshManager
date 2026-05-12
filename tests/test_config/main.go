// test_config — Tests %APPDATA% config read/write (console app)
// Compile: go build -o test_config.exe .
// Run from cmd: test_config.exe
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	DBPath string `json:"db_path"`
}

func main() {
	appData := os.Getenv("APPDATA")
	fmt.Printf("[test] APPDATA = %q\n", appData)

	if appData == "" {
		fmt.Println("[FAIL] APPDATA is not set!")
		os.Exit(1)
	}

	configPath := filepath.Join(appData, "SSH Manager", "config.json")
	fmt.Printf("[test] config path = %s\n", configPath)

	// Write test
	cfg := Config{DBPath: "C:\\test\\ssh.db"}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.MkdirAll(filepath.Dir(configPath), 0755)
	err := os.WriteFile(configPath, data, 0644)
	if err != nil {
		fmt.Printf("[FAIL] Write config: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("[OK] Write config succeeded")

	// Read test
	data2, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("[FAIL] Read config: %v\n", err)
		os.Exit(1)
	}
	var cfg2 Config
	json.Unmarshal(data2, &cfg2)
	fmt.Printf("[OK] Read config: db_path = %q\n", cfg2.DBPath)

	if cfg2.DBPath != "C:\\test\\ssh.db" {
		fmt.Println("[FAIL] Config value mismatch!")
		os.Exit(1)
	}
	fmt.Println("[OK] Config value matches!")

	// Cleanup
	os.Remove(configPath)
	os.Remove(filepath.Dir(configPath))
	fmt.Println("[OK] Cleanup done")
	fmt.Println("[PASS] All config tests passed!")
}
