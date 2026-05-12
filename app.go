// Package main — SSH Manager
//
// Architecture:
//
//	main.go            Entry point (minimal — db init + window setup)
//	app.go             Globals + shared helper functions
//	models.go          Data models + table data refresh
//	db.go              SQLite persistence layer
//	ssh.go             SSH client
//	ui_panels.go       Main window panels & layout
//	ui_actions.go      Connection/script CRUD dialogs
//	ui_execute.go      Execution center & quick functions
//	ui_history.go      Execution history dialog
//	ui_filebrowser.go  Remote file browser
//	assets/            Icons, manifest, RC resources
//	scripts/           Helper scripts (mkicon.py)
//	docs/              Developer & user documentation
//	dist/              Build output
package main

import (
	"fmt"

	"github.com/lxn/walk"
)

// ============================================================
// Globals — widget references shared across all ui_*.go files.
// Walk's declarative API requires passing pointers, so these
// are centralized here rather than passed through constructors.
// ============================================================

var (
	// Core
	db      *Database
	mainWnd *walk.MainWindow

	// Connection panel
	connTV           *walk.TableView
	connSearchInput  *walk.LineEdit

	// Script panel
	scriptTV *walk.TableView

	// Status bar & counts
	statusLabel      *walk.Label
	connCountLabel   *walk.Label
	scriptCountLabel *walk.Label

	// Quick command bar (bottom)
	quickCmdInput  *walk.LineEdit
	quickConnLabel *walk.Label
)

// ============================================================
// Shared helpers
// ============================================================

// setStatus updates the status bar label.
func setStatus(s string) {
	if statusLabel != nil {
		statusLabel.SetText(s)
	}
}

// updateCounts refreshes the connection/script count labels.
func updateCounts() {
	conns, _ := db.GetConnections()
	scripts, _ := db.GetScripts()
	connCountLabel.SetText(fmt.Sprintf("共 %d 个连接", len(conns)))
	scriptCountLabel.SetText(fmt.Sprintf("共 %d 个脚本", len(scripts)))
}

// refreshAll reloads both panels from DB and updates counts.
func refreshAll() {
	refreshConnData()
	refreshScriptData()
	updateCounts()
	setStatus("已刷新")
}

// getConnID returns the ID of the currently selected connection.
func getConnID() int64 {
	if connTV == nil {
		return 0
	}
	idx := connTV.CurrentIndex()
	if idx < 0 {
		return 0
	}
	connMu.RLock()
	defer connMu.RUnlock()
	if idx >= len(connCache) {
		return 0
	}
	return connCache[idx].ID
}

// getScriptID returns the ID of the currently selected script.
func getScriptID() int64 {
	if scriptTV == nil {
		return 0
	}
	idx := scriptTV.CurrentIndex()
	if idx < 0 {
		return 0
	}
	scriptMu.RLock()
	defer scriptMu.RUnlock()
	if idx >= len(scriptCache) {
		return 0
	}
	return scriptCache[idx].ID
}

// getConnByID retrieves a connection from the database by ID.
func getConnByID(id int64) *Connection {
	conns, _ := db.GetConnections()
	for i := range conns {
		if conns[i].ID == id {
			return conns[i]
		}
	}
	return nil
}

// getScriptByID retrieves a script from the database by ID.
func getScriptByID(id int64) *Script {
	scripts, _ := db.GetScripts()
	for i := range scripts {
		if scripts[i].ID == id {
			return scripts[i]
		}
	}
	return nil
}
