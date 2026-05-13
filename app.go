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
	"os"
	"time"

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


// exportRichText opens a file-save dialog and writes the RichText content to a
// plain-text file. Returns the exported file path (empty if cancelled).
func exportRichText(owner walk.Form, rt *RichText, suggestedName string) string {
	if rt == nil {
		walk.MsgBox(owner, "提示", "没有可导出的内容", walk.MsgBoxIconInformation)
		return ""
	}
	text := rt.GetText()
	if text == "" {
		walk.MsgBox(owner, "提示", "没有可导出的内容", walk.MsgBoxIconInformation)
		return ""
	}
	fd := new(walk.FileDialog)
	fd.Title = "导出输出日志"
	fd.Filter = "文本文件 (*.txt)|*.txt|日志文件 (*.log)|*.log|所有文件 (*.*)|*.*"
	if suggestedName == "" {
		suggestedName = fmt.Sprintf("output_%s.txt", time.Now().Format("20060102_150405"))
	}
	fd.FilePath = suggestedName
	ok, err := fd.ShowSave(owner)
	if err != nil {
		walk.MsgBox(owner, "错误", "保存对话框错误: "+err.Error(), walk.MsgBoxIconError)
		return ""
	}
	if !ok {
		return ""
	}
	if err := os.WriteFile(fd.FilePath, []byte(text), 0644); err != nil {
		walk.MsgBox(owner, "错误", "导出失败: "+err.Error(), walk.MsgBoxIconError)
		return ""
	}
	walk.MsgBox(owner, "导出成功", fmt.Sprintf("已导出到:\n%s", fd.FilePath), walk.MsgBoxIconInformation)
	return fd.FilePath
}

// exportText opens a file-save dialog and writes a string to a plain-text file.
func exportText(owner walk.Form, content string, suggestedName string) {
	if content == "" {
		walk.MsgBox(owner, "提示", "没有可导出的内容", walk.MsgBoxIconInformation)
		return
	}
	fd := new(walk.FileDialog)
	fd.Title = "导出为文本文件"
	fd.Filter = "文本文件 (*.txt)|*.txt|日志文件 (*.log)|*.log|所有文件 (*.*)|*.*"
	fd.FilePath = suggestedName
	ok, err := fd.ShowSave(owner)
	if err != nil {
		walk.MsgBox(owner, "错误", "保存对话框错误: "+err.Error(), walk.MsgBoxIconError)
		return
	}
	if !ok {
		return
	}
	if err := os.WriteFile(fd.FilePath, []byte(content), 0644); err != nil {
		walk.MsgBox(owner, "错误", "导出失败: "+err.Error(), walk.MsgBoxIconError)
		return
	}
	walk.MsgBox(owner, "导出成功", fmt.Sprintf("已导出到:\n%s", fd.FilePath), walk.MsgBoxIconInformation)
}