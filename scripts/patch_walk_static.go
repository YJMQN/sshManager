// Patch Walk's static.go to ignore TTM_ADDTOOL error on Win7.
// This is a build-time workaround, not a permanent fork of Walk.
//
// Usage: go run scripts\patch_walk_static.go
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// Locate Walk module in GOPATH cache
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = filepath.Join(os.Getenv("USERPROFILE"), "go")
	}

	walkDir := filepath.Join(gopath, "pkg", "mod", "github.com", "lxn", "walk@v0.0.0-20210112085537-c389da54e794")
	staticPath := filepath.Join(walkDir, "static.go")

	data, err := os.ReadFile(staticPath)
	if err != nil {
		fmt.Printf("[WARN] Cannot read Walk static.go: %v\n", err)
		os.Exit(0) // non-fatal, build may still work
	}

	content := string(data)

	// Check if already patched
	if strings.Contains(content, "/* patched for Win7 */") {
		fmt.Println("[OK] Walk static.go already patched")
		return
	}

	old := `	if err := s.toolTip.AddTool(s); err != nil {
		return err
	}`
	new := `	if err := s.toolTip.AddTool(s); err != nil {
		/* patched for Win7 */ _ = err
	}`

	if !strings.Contains(content, old) {
		fmt.Println("[WARN] Walk static.go format unexpected, skipping patch")
		fmt.Printf("  Searched for: %q\n", old)
		os.Exit(0)
	}

	content = strings.ReplaceAll(content, old, new)

	if err := os.WriteFile(staticPath, []byte(content), 0644); err != nil {
		fmt.Printf("[ERROR] Cannot write Walk static.go: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("[OK] Walk static.go patched (TTM_ADDTOOL ignored)")
}
