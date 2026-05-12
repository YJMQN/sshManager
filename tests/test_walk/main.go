// test_walk — Minimal Walk + Dialog test
// Compile: go build -ldflags="-s -w -H windowsgui" -o test_walk.exe .
// Run in cmd: test_walk.exe
// If a window with the message appears, Walk itself is fine.
// Logs to test_walk.log next to the exe.
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

	var mainWnd *walk.MainWindow
	var btn *walk.PushButton

	_, err := MainWindow{
		AssignTo: &mainWnd,
		Title:    "Walk Test — Win7",
		MinSize:  Size{400, 300},
		Layout:   VBox{},
		Children: []Widget{
			Label{Text: "如果看到这个窗口，Walk 正常！"},
			PushButton{
				AssignTo: &btn,
				Text:     "测试对话框",
				OnClicked: func() {
					walk.MsgBox(mainWnd, "成功", "对话框工作正常！", walk.MsgBoxIconInformation)
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
