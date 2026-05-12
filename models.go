package main

import (
	"fmt"
	"strings"
	"sync"
)

// --- Data types ---

type Connection struct {
	ID       int64
	Name     string
	Host     string
	Port     int
	Username string
	AuthType string
	Password string
	KeyPath  string
}

type Script struct {
	ID          int64
	Name        string
	Content     string
	Interpreter string
}

type ExecHistory struct {
	ID             int64
	ConnectionID   int64
	ConnectionName string
	ScriptID       int64
	ScriptName     string
	Interpreter    string
	Status         string
	Output         string
	Error          string
	StartedAt      string
	FinishedAt     string
	DurationMs     int
}

// Supported interpreters
var interpreters = []string{
	"sh", "bash", "python3", "python", "perl",
	"ruby", "node", "php", "powershell",
}

// ============ Table data sources ============
//
// Walk's declarative TableView ONLY accepts []map[string]interface{} as Model.
// Column order issue: Go map key iteration is random, causing misaligned columns.
// Solution: put ONLY the display keys in the table data, lookup extras via db.
// The *_cache slices store extra fields keyed by table row index.

type connCacheEntry struct {
	ID       int64
	Password string
	KeyPath  string
	AuthType string
	Port     int
}

type scriptCacheEntry struct {
	ID          int64
	Content     string
	Interpreter string
}

var (
	connData    []map[string]interface{}
	scriptData  []map[string]interface{}
	connCache   []connCacheEntry
	scriptCache []scriptCacheEntry
	connMu      sync.RWMutex
	scriptMu    sync.RWMutex
	testedOK    map[int64]bool
	testedOKMu  sync.RWMutex
)

func markTestedOK(id int64) {
	testedOKMu.Lock()
	if testedOK == nil {
		testedOK = make(map[int64]bool)
	}
	testedOK[id] = true
	testedOKMu.Unlock()
}

func isTestedOK(id int64) bool {
	testedOKMu.RLock()
	defer testedOKMu.RUnlock()
	if testedOK == nil {
		return false
	}
	return testedOK[id]
}

func refreshConnData() {
	conns, err := db.GetConnections()
	if err != nil {
		return
	}

	// Apply search filter
	filterText := ""
	if connSearchInput != nil {
		filterText = connSearchInput.Text()
	}

	connMu.Lock()
	if filterText != "" {
		filtered := make([]*Connection, 0, len(conns))
		for _, c := range conns {
			if containsCI(c.Name, filterText) || containsCI(c.Host, filterText) {
				filtered = append(filtered, c)
			}
		}
		conns = filtered
	}

	n := len(conns)
	connData = make([]map[string]interface{}, n)
	connCache = make([]connCacheEntry, n)
	for i, c := range conns {
		authLabel := "密码"
		if c.AuthType == "key" {
			authLabel = "密钥"
		}
		connData[i] = map[string]interface{}{
			"名称": c.Name,
			"主机": c.Host,
			"端口": fmt.Sprintf("%d", c.Port),
			"用户": c.Username,
			"认证": authLabel,
			"操作": "▶ 执行",
		}
		connCache[i] = connCacheEntry{
			ID:       c.ID,
			Password: c.Password,
			KeyPath:  c.KeyPath,
			AuthType: c.AuthType,
			Port:     c.Port,
		}
	}
	connMu.Unlock()
	if connTV != nil {
		connTV.SetModel(connData)
	}
}

// Case-insensitive contains
func containsCI(s, substr string) bool {
	s, substr = strings.ToLower(s), strings.ToLower(substr)
	return strings.Contains(s, substr)
}

func refreshScriptData() {
	scripts, err := db.GetScripts()
	if err != nil {
		return
	}
	scriptMu.Lock()
	n := len(scripts)
	scriptData = make([]map[string]interface{}, n)
	scriptCache = make([]scriptCacheEntry, n)
	for i, s := range scripts {
		preview := s.Content
		runes := []rune(preview)
		if len(runes) > 80 {
			preview = string(runes[:80]) + "..."
		}
		scriptData[i] = map[string]interface{}{
			"名称":    s.Name,
			"解释器":   s.Interpreter,
			"内容预览": preview,
		}
		scriptCache[i] = scriptCacheEntry{
			ID:          s.ID,
			Content:     s.Content,
			Interpreter: s.Interpreter,
		}
	}
	scriptMu.Unlock()
	if scriptTV != nil {
		scriptTV.SetModel(scriptData)
	}
}
