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

// dbPath holds the current SQLite database path.
//
// Priority:
//
//	1. CLI arg:       --db-path=<path>
//	2. Config file:   %APPDATA%\SSH Manager\config.json  (via UI)
//	3. Env var:       SSH_MANAGER_DB
//	4. Default:       <exe_dir>/ssh_manager.db
var dbPath string

func main() {
	dbPath = resolveDBPath()

	// ---- First launch: prompt for db path BEFORE main window ----
	if !configExists() {
		p := promptDBPath() // modal dialog, owner=nil
		if p == "" {
			// User cancelled — cannot proceed
			return
		}
		dbPath = p
	}

	// ---- Open database ----
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
					Action{Text: "📂 设置数据库路径...", OnTriggered: onSetDBPath},
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

// ============================================================
// First-launch setup dialog (before main window exists)
// ============================================================

// configExists checks if the config file has a valid db_path.
func configExists() bool {
	cfg := loadConfig()
	return cfg.DBPath != ""
}

// promptDBPath shows a modal dialog for the user to choose a database path.
// Called before the main window exists, so owner is nil.
// Returns the chosen path, or "" if cancelled.
func promptDBPath() string {
	exe, _ := os.Executable()
	defaultPath := filepath.Join(filepath.Dir(exe), "ssh_manager.db")

	var dlg *walk.Dialog
	var pathInput *walk.LineEdit
	result := ""

	_, err := Dialog{
		AssignTo: &dlg,
		Title:    "👋 欢迎使用 SSH Manager — 选择数据库位置",
		MinSize:  Size{580, 220},
		Layout:   VBox{Margins: Margins{15, 15, 15, 15}},
		Children: []Widget{
			Label{Text: "首次启动，请指定数据库文件存储位置：",
				Font: Font{PointSize: 10, Bold: true}},
			Label{Text: "数据库文件路径:", Font: Font{PointSize: 9}},
			Composite{
				Layout: HBox{Spacing: 6},
				Children: []Widget{
					LineEdit{
						AssignTo:    &pathInput,
						Text:        defaultPath,
						MinSize:     Size{360, 0},
						ToolTipText: "输入或浏览选择 .db 文件路径",
					},
					PushButton{Text: "浏览...", OnClicked: func() {
						fd := new(walk.FileDialog)
						fd.Title = "选择或新建数据库文件"
						fd.Filter = "数据库文件 (*.db)|*.db|所有文件 (*.*)|*.*"
						if ok, _ := fd.ShowSave(nil); ok {
							pathInput.SetText(fd.FilePath)
						}
					}},
				},
			},
			Label{Text: "路径不存在会自动创建，可随时在菜单中更改",
				TextColor: walk.RGB(128, 128, 128)},
			Composite{
				Layout: HBox{Spacing: 8},
				Children: []Widget{
					HSpacer{},
					PushButton{Text: "✅ 确认并启动", MinSize: Size{120, 0}, OnClicked: func() {
						p := pathInput.Text()
						if p == "" {
							walk.MsgBox(dlg, "错误", "请输入数据库路径", walk.MsgBoxIconError)
							return
						}
						// Verify the path works
						testDB, err := NewDatabase(p)
						if err != nil {
							walk.MsgBox(dlg, "错误",
								"无法创建/打开数据库: "+err.Error(), walk.MsgBoxIconError)
							return
						}
						testDB.Close()

						// Save to config
						_ = saveConfig(&Config{DBPath: p})
						result = p
						dlg.Accept()
					}},
				},
			},
		},
	}.Run(nil)

	if err != nil {
		walk.MsgBox(nil, "错误", "启动设置失败: "+err.Error(), walk.MsgBoxIconError)
		return ""
	}
	return result
}

// ============================================================
// Database path resolution
// ============================================================

func resolveDBPath() string {
	// 1. CLI arg — highest priority
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "--db-path=") {
			p := strings.TrimPrefix(arg, "--db-path=")
			if p != "" {
				return ensureDir(p)
			}
		}
	}

	// 2. Config file (set via UI, stored in %APPDATA%)
	cfg := loadConfig()
	if cfg.DBPath != "" {
		return ensureDir(cfg.DBPath)
	}

	// 3. Environment variable
	if env := os.Getenv("SSH_MANAGER_DB"); env != "" {
		return ensureDir(env)
	}

	// 4. Default
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "ssh_manager.db")
}

// ensureDir ensures the parent directory exists.
func ensureDir(p string) string {
	dir := filepath.Dir(p)
	if dir != "." && dir != "" {
		os.MkdirAll(dir, 0755)
	}
	return p
}

// ============================================================
// UI: Change DB path dialog (menu item, main window exists)
// ============================================================

func onSetDBPath() {
	var dlg *walk.Dialog
	var pathInput *walk.LineEdit

	_, err := Dialog{
		AssignTo: &dlg,
		Title:    "设置数据库路径",
		MinSize:  Size{560, 190},
		Layout:   VBox{Margins: Margins{10, 10, 10, 10}},
		Children: []Widget{
			Label{Text: "数据库文件路径:", Font: Font{PointSize: 9, Bold: true}},
			Composite{
				Layout: HBox{Spacing: 6},
				Children: []Widget{
					LineEdit{
						AssignTo:    &pathInput,
						Text:        dbPath,
						MinSize:     Size{340, 0},
						ToolTipText: "输入 .db 文件路径，或点击浏览选择",
					},
					PushButton{Text: "浏览...", OnClicked: func() {
						fd := new(walk.FileDialog)
						fd.Title = "选择 SQLite 数据库文件"
						fd.Filter = "数据库文件 (*.db)|*.db|所有文件 (*.*)|*.*"
						if ok, _ := fd.ShowOpen(mainWnd); ok {
							pathInput.SetText(fd.FilePath)
						}
					}},
				},
			},
			Label{Text: "更改后将重新加载数据库", TextColor: walk.RGB(128, 128, 128)},
			Composite{
				Layout: HBox{Spacing: 8},
				Children: []Widget{
					HSpacer{},
					PushButton{Text: "确定并切换", OnClicked: func() {
						newPath := pathInput.Text()
						if newPath == "" {
							walk.MsgBox(dlg, "错误", "路径不能为空", walk.MsgBoxIconError)
							return
						}
						if newPath == dbPath {
							dlg.Cancel()
							return
						}

						newDB, err := NewDatabase(newPath)
						if err != nil {
							walk.MsgBox(dlg, "错误",
								"无法打开数据库: "+err.Error(), walk.MsgBoxIconError)
							return
						}

						db.Close()
						db = newDB
						dbPath = newPath

						_ = saveConfig(&Config{DBPath: newPath})

						testedOK = make(map[int64]bool)
						refreshConnData()
						refreshScriptData()
						updateCounts()
						setStatus(fmt.Sprintf("📂 数据库已切换: %s", dbPath))

						dlg.Accept()
					}},
					PushButton{Text: "取消", OnClicked: func() { dlg.Cancel() }},
				},
			},
		},
	}.Run(mainWnd)

	if err != nil {
		walk.MsgBox(mainWnd, "错误", err.Error(), walk.MsgBoxIconError)
	}
}

// ============================================================
// About
// ============================================================

func showAbout() {
	walk.MsgBox(mainWnd, "关于 SSH Manager",
		fmt.Sprintf("SSH Manager v1.0 (Go)\n\n"+
			"远程脚本执行管理工具\n"+
			"基于 Go + Walk 原生 GUI\n"+
			"• 编译型，启动快\n• 单文件，零依赖\n• 内存占用极低\n\n"+
			"📂 数据库:\n%s\n\n⚙️ 配置文件:\n%s", dbPath, configPath()),
		walk.MsgBoxIconInformation)
}
