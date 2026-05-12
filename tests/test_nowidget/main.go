// test_nowidget — 最简 Walk 窗口，不包含任何子控件
// 测试 Win7 上 Walk 的 MainWindow 本身能不能显示
// 需要在同目录有 app.syso（含 manifest），见 build_tests.bat
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
	logPath := filepath.Join(filepath.Dir(exe), "test_nowidget.log")
	f, _ := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if f != nil {
		defer f.Close()
		fmt.Fprintln(f, msg)
	}
}

func main() {
	logToFile("=== test_nowidget START ===")

	var mw *walk.MainWindow

	_, err := MainWindow{
		AssignTo: &mw,
		Title:    "Test — Bare Window (No Widgets)",
		MinSize:  Size{400, 200},
		Layout:   VBox{},
	}.Run()

	if err != nil {
		logToFile(fmt.Sprintf("FAILED: %v", err))
		fmt.Fprintf(os.Stderr, "FAILED: %v\n", err)
		os.Exit(1)
	}

	logToFile("=== test_nowidget END ===")
}
