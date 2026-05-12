package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

var (
	db               *Database
	mainWnd          *walk.MainWindow
	connTV           *walk.TableView
	scriptTV         *walk.TableView
	statusLabel      *walk.Label
	connCountLabel   *walk.Label
	scriptCountLabel *walk.Label
)

func main() {
	// Init DB in exe directory
	exe, _ := os.Executable()
	dbDir := filepath.Dir(exe)
	dbPath := filepath.Join(dbDir, "ssh_manager.db")

	var err error
	db, err = NewDatabase(dbPath)
	if err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	defer db.Close()

	// Load initial data
	refreshConnData()
	refreshScriptData()

	// Load icon from embedded resource (ID 100 from app.rc)
	appIcon, err := walk.NewIconFromResourceId(100)
	if err != nil {
		appIcon = nil // silently ignore, window will have no icon
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
					Action{Text: "执行中心...",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyE},
						OnTriggered: openExecuteDlg,
					},
					Action{Text: "执行历史...",
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

func showAbout() {
	walk.MsgBox(mainWnd, "关于 SSH Manager",
		"SSH Manager v1.0 (Go)\n\n"+
			"远程脚本执行管理工具\n"+
			"基于 Go + Walk 原生 GUI\n"+
			"• 编译型，启动快\n• 单文件，零依赖\n• 内存占用极低",
		walk.MsgBoxIconInformation)
}
