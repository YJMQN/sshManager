package main

import (
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

type FileInfo struct {
	Name       string
	Size       string
	ModTime    string
	Permission string
	IsDir      bool
}

// parseLSOutput parses the output of `ls -la` into FileInfo entries.
// Handles both standard and long-iso time formats.
// Skips the "total" line and entries starting with "." (hidden files).
func parseLSOutput(output string) []FileInfo {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	files := make([]FileInfo, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip "total" line
		if strings.HasPrefix(line, "total ") {
			continue
		}
		// Split into fields
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		perm := fields[0]
		if len(perm) < 1 || perm[0] != '-' && perm[0] != 'd' && perm[0] != 'l' {
			continue
		}

		isDir := perm[0] == 'd'
		isLink := perm[0] == 'l'
		sizeStr := fields[4]

		// Determine filename position
		// Standard format: perm links owner group size month day time/year filename
		// With long-iso:   perm links owner group size date time filename
		// Fields 0-4 are: perm, links, owner, group, size = 5 fields
		// Then we have time fields: month/day/time or date/time
		// So filename starts at field index 8 (0-indexed: 0=perm,1=links,2=owner,3=group,4=size,5=month/date,6=day/time,7=time/year,8+=filename)
		// Actually let's count more carefully.
		// Standard:   -rw-r--r-- 1 user group 1234 Jan 15 10:30 filename
		// Fields:     0          1 2    3     4    5   6  7     8+
		// long-iso:   -rw-r--r-- 1 user group 1234 2024-01-15 10:30 filename
		// Fields:     0          1 2    3     4    5          6     7+

		var nameIdx int
		var modTime string
		if len(fields) >= 10 {
			// Standard format: month(5) day(6) time(7) name(8+)
			modTime = fmt.Sprintf("%s %s %s", fields[5], fields[6], fields[7])
			nameIdx = 8
		} else {
			// long-iso format: date(5) time(6) name(7+)
			modTime = fmt.Sprintf("%s %s", fields[5], fields[6])
			nameIdx = 7
		}

		// Reconstruct filename (may contain spaces)
		filename := strings.Join(fields[nameIdx:], " ")

		// Skip hidden files
		if strings.HasPrefix(filename, ".") {
			continue
		}

		// Handle symlinks: "linkname -> target"
		if isLink {
			if idx := strings.Index(filename, " -> "); idx >= 0 {
				filename = filename[:idx]
			}
		}

		files = append(files, FileInfo{
			Name:       filename,
			Size:       formatSize(sizeStr),
			ModTime:    modTime,
			Permission: perm,
			IsDir:      isDir,
		})
	}

	return files
}

func formatSize(size string) string {
	n, err := strconv.ParseInt(size, 10, 64)
	if err != nil {
		return size
	}
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	} else if n < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	} else if n < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(n)/1024/1024)
	}
	return fmt.Sprintf("%.1f GB", float64(n)/1024/1024/1024)
}

func openFileBrowser() {
	connIdx := connTV.CurrentIndex()
	if connIdx < 0 {
		walk.MsgBox(mainWnd, "提示", "请先选中一个连接", walk.MsgBoxIconInformation)
		return
	}

	conns, _ := db.GetConnections()
	if connIdx >= len(conns) {
		return
	}
	conn := conns[connIdx]

	var dlg *walk.Dialog
	var fileTV *walk.TableView
	var pathLabel *walk.Label
	var statusLbl *walk.Label
	var refBtn, upBtn *walk.PushButton

	currentPath := "/"

	fileData := make([]map[string]interface{}, 0)

	refreshFiles := func(dir string) {
		statusLbl.SetText(fmt.Sprintf("⏳ 读取 %s ...", dir))
		refBtn.SetEnabled(false)
		upBtn.SetEnabled(false)

		go func() {
			client := &SSHClient{}
			err := client.Connect(conn.Host, conn.Port, conn.Username,
				conn.AuthType, conn.Password, conn.KeyPath, 10*time.Second)
			if err != nil {
				statusLbl.SetText(fmt.Sprintf("❌ 连接失败: %v", err))
				refBtn.SetEnabled(true)
				return
			}

			// Try with long-iso first, fall back to standard
			cmd := buildCommand("sh", fmt.Sprintf("ls -la --time-style=long-iso %s", escapePath(dir)))
			stdout, _, _, execErr := client.Execute(cmd, nil)
			if execErr != nil || stdout == "" {
				// Fallback without --time-style
				cmd = buildCommand("sh", fmt.Sprintf("ls -la %s", escapePath(dir)))
				stdout, _, _, execErr = client.Execute(cmd, nil)
			}
			client.Close()

			if execErr != nil {
				statusLbl.SetText(fmt.Sprintf("❌ 读取失败: %v", execErr))
				refBtn.SetEnabled(true)
				return
			}

			files := parseLSOutput(stdout)
			entries := make([]map[string]interface{}, len(files))
			for i, f := range files {
				icon := "📄"
				if f.IsDir {
					icon = "📁"
				}
				entries[i] = map[string]interface{}{
					"":       icon,
					"名称":    f.Name,
					"大小":    f.Size,
					"修改时间": f.ModTime,
					"权限":    f.Permission,
				}
			}

			// Update UI on main thread
			mainWnd.Synchronize(func() {
				fileData = entries
				fileTV.SetModel(fileData)
				pathLabel.SetText("📂 " + dir)
				currentPath = dir
				statusLbl.SetText(fmt.Sprintf("共 %d 条", len(files)))
				refBtn.SetEnabled(true)
				upBtn.SetEnabled(dir != "/")
			})
		}()
	}

	// Navigate into directory on double-click
	onDoubleClick := func() {
		idx := fileTV.CurrentIndex()
		if idx < 0 || idx >= len(fileData) {
			return
		}
		entry := fileData[idx]
		name, _ := entry["名称"].(string)
		isDir := false
		icon, _ := entry[""].(string)
		if icon == "📁" {
			isDir = true
		}
		if !isDir {
			return
		}
		newPath := path.Join(currentPath, name)
		refreshFiles(newPath)
	}

	// Create dialog first (non-blocking)
	err := Dialog{
		AssignTo: &dlg,
		Title:    fmt.Sprintf("📂 文件浏览 — %s (%s:%d)", conn.Name, conn.Host, conn.Port),
		MinSize:  Size{680, 450},
		Layout:   VBox{Margins: Margins{10, 10, 10, 10}},
		Children: []Widget{
			// Path + action bar
			Composite{
				Layout: HBox{Spacing: 8},
				Children: []Widget{
					Label{AssignTo: &pathLabel, Text: "📂 /", Font: Font{PointSize: 9, Bold: true}},
					HSpacer{},
					PushButton{AssignTo: &upBtn, Text: "⬆ 上级", OnClicked: func() {
						parent := path.Dir(currentPath)
						if parent != currentPath {
							refreshFiles(parent)
						}
					}},
					PushButton{AssignTo: &refBtn, Text: "🔄 刷新", OnClicked: func() {
						refreshFiles(currentPath)
					}},
					PushButton{Text: "关闭", OnClicked: func() { dlg.Cancel() }},
				},
			},
			// File list
			TableView{
				AssignTo:          &fileTV,
				Model:             fileData,
				MultiSelection:    false,
				OnItemActivated:   onDoubleClick,
				Columns: []TableViewColumn{
					{Title: "", Width: 30, DataMember: ""},
					{Title: "名称", Width: 220, DataMember: "名称"},
					{Title: "大小", Width: 90, Alignment: AlignFar, DataMember: "大小"},
					{Title: "修改时间", Width: 160, DataMember: "修改时间"},
					{Title: "权限", Width: 100, Alignment: AlignCenter, DataMember: "权限"},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					Label{AssignTo: &statusLbl, Text: "就绪", TextColor: walk.RGB(100, 100, 100)},
					Label{Text: "提示: 双击 📁 目录进入, 按 ⬆ 上级返回"},
				},
			},
		},
	}.Create(mainWnd)

	if err != nil {
		walk.MsgBox(mainWnd, "错误", err.Error(), walk.MsgBoxIconError)
		return
	}

	// Load initial file list now that widgets exist
	refreshFiles("/")

	// Show and run the dialog
	dlg.Run()
}

func escapePath(dir string) string {
	// Simple quoting for shell
	return "'" + strings.ReplaceAll(dir, "'", "'\\''") + "'"
}
