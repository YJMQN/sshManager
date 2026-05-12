// test_walk — 测试 Walk 但不使用 Label（绕过 static.go bug）
// 用 LineEdit{ReadOnly: true} 替代 Label
// Compile: go build -ldflags="-s -w -H windowsgui -extldflags=-static" -o test_walk.exe .
// 日志写到同目录 test_walk.log
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

func logToFile(msg string) {
	exe, _ := os.Executable()
	logPath := filepath.Join(filepath.Dir(exe), "test_walk.log")
	f, _ := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if f != nil {
		defer f.Close()
		fmt.Fprintln(f, msg)
	}
}

func main() {
	logToFile("=== test_walk START ===")

	var mw *walk.MainWindow
	var btn *walk.PushButton

	_, err := MainWindow{
		AssignTo: &mw,
		Title:    "Walk Test — Win7 (No Labels)",
		MinSize:  Size{400, 200},
		Layout:   VBox{},
		Children: []Widget{
			// Use read-only LineEdit instead of Label to avoid TTM_ADDTOOL bug
			LineEdit{
				Text:      "如果看到这个，Walk 可以正常使用！",
				ReadOnly:  true,
				MaxSize:   Size{0, 30},
			},
			PushButton{
				AssignTo: &btn,
				Text:     "测试对话框",
				OnClicked: func() {
					walk.MsgBox(mw, "✓ 成功", "对话框工作正常！", walk.MsgBoxIconInformation)
				},
			},
		},
	}.Run()

	if err != nil {
		logToFile(fmt.Sprintf("FAILED: %v", err))
		fmt.Fprintf(os.Stderr, "FAILED: %v\n", err)
		os.Exit(1)
	}
	logToFile("=== test_walk END ===")
}
