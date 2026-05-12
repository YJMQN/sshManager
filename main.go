package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

// dbPath is set from CLI arg or defaults to exe directory.
// Override: SSHManager.exe --db-path=C:\data\ssh_manager.db
var dbPath string

func main() {
	// ---- Parse --db-path CLI argument ----
	dbPath = resolveDBPath()

	var err error
	db, err = NewDatabase(dbPath)
	if err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	defer db.Close()

	testedOK = make(map[int64]bool)

	// Load initial data
	refreshConnData()
	refreshScriptData()

	setStatus(fmt.Sprintf("📂 数据库: %s", dbPath))

	// Load icon from embedded resource (ID 100 from app.rc)
	appIcon, err := walk.NewIconFromResourceId(100)
	if err != nil {
		appIcon = nil
	}

	if _, err := (MainWindow{
		AssignTo: &mainWnd,
		Title:    "SSH Manager — 远程脚本执行管理工具",
		Icon:     appIcon,
		MinSize:  Size{900, 550},
		Size:     Size{1000, 620},
		Layout:   VBox{MarginsZero: true},
		MenuItems: []MenuItem{
			Menu{
				Text: "文件",
				Items: []MenuItem{
					Action{Text: "⚡ 快速命令...",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyD},
						OnTriggered: openQuickCmdDlg,
					},
					Action{Text: "▶ 执行中心...",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyE},
						OnTriggered: openExecuteDlg,
					},
					Action{Text: "📋 执行历史...",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyH},
						OnTriggered: openHistoryDlg,
					},
					Separator{},
					Action{Text: "退出",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyQ},
						OnTriggered: func() { mainWnd.Close() },
					},
				},
			},
			Menu{
				Text: "帮助",
				Items: []MenuItem{
					Action{Text: "关于", OnTriggered: showAbout},
				},
			},
		},
		Children: []Widget{
			HSplitter{
				Children: []Widget{
					buildConnPanel(),
					buildScriptPanel(),
				},
			},
			buildBottomBar(),
		},
	}.Run()); err != nil {
		log.Fatal(err)
	}
}

// resolveDBPath determines the SQLite database path.
// Priority:
//
//	1. --db-path=<path>    CLI argument
//	2. SSH_MANAGER_DB     environment variable
//	3. <exe_dir>/ssh_manager.db   (default)
func resolveDBPath() string {
	// 1. CLI arg: --db-path=<path>
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "--db-path=") {
			p := strings.TrimPrefix(arg, "--db-path=")
			if p != "" {
				return ensureDir(p)
			}
		}
	}

	// 2. Environment variable
	if env := os.Getenv("SSH_MANAGER_DB"); env != "" {
		return ensureDir(env)
	}

	// 3. Default: exe directory
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "ssh_manager.db")
}

// ensureDir ensures the parent directory exists and returns the path.
func ensureDir(p string) string {
	dir := filepath.Dir(p)
	if dir != "." && dir != "" {
		os.MkdirAll(dir, 0755)
	}
	return p
}

func showAbout() {
	walk.MsgBox(mainWnd, "关于 SSH Manager",
		fmt.Sprintf("SSH Manager v1.0 (Go)\n\n"+
			"远程脚本执行管理工具\n"+
			"基于 Go + Walk 原生 GUI\n"+
			"• 编译型，启动快\n• 单文件，零依赖\n• 内存占用极低\n\n"+
			"📂 数据库:\n%s", dbPath),
		walk.MsgBoxIconInformation)
}
