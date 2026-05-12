package main

import (
	"fmt"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

func buildConnPanel() Widget {
	return Composite{
		Layout: VBox{Margins: Margins{5, 5, 2, 5}},
		Children: []Widget{
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					Label{Text: "🖥 连接管理", Font: Font{PointSize: 11, Bold: true}},
					Label{Text: "SSH Connections", TextColor: walk.RGB(128, 128, 128)},
					HSpacer{},
					// Search box
					Label{Text: "🔍", TextColor: walk.RGB(128, 128, 128)},
					LineEdit{
						AssignTo: &connSearchInput,
						MinSize:  Size{120, 0},
						MaxSize:  Size{200, 22},
						ToolTipText: "输入名称或主机过滤",
						OnKeyUp: func(key walk.Key) {
							refreshConnData()
						},
					},
				},
			},
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					PushButton{Text: "＋ 新建", MinSize: Size{80, 0}, OnClicked: onAddConn},
					PushButton{Text: "✎ 编辑", MinSize: Size{80, 0}, OnClicked: onEditConn},
					PushButton{Text: "🗑 删除", MinSize: Size{80, 0}, OnClicked: onDelConn},
					PushButton{Text: "🔌 测试", MinSize: Size{80, 0}, OnClicked: onTestConn},
					PushButton{Text: "📂 文件", MinSize: Size{75, 0}, OnClicked: openFileBrowser},
					PushButton{Text: "▶ 执行", MinSize: Size{80, 0}, OnClicked: func() {
						connIdx := connTV.CurrentIndex()
						if connIdx < 0 {
							walk.MsgBox(mainWnd, "提示", "请先选中一个连接", walk.MsgBoxIconInformation)
							return
						}
						openExecuteDlgWithConn(connIdx)
					}},
					PushButton{Text: "⚡ 快速执行", MinSize: Size{95, 0}, OnClicked: openQuickExecDlg},
				},
			},
			TableView{
				AssignTo:        &connTV,
				Model:           connData,
				MultiSelection:  false,
				OnItemActivated: openQuickCmdDlg,
				OnCurrentIndexChanged: func() {
					idx := connTV.CurrentIndex()
					conns, _ := db.GetConnections()
					if idx >= 0 && idx < len(conns) {
						c := conns[idx]
						quickConnLabel.SetText(fmt.Sprintf("▶ %s (%s:%d)", c.Name, c.Host, c.Port))
						quickConnLabel.SetTextColor(walk.RGB(0, 80, 160))
					} else {
						quickConnLabel.SetText("(未选择连接)")
						quickConnLabel.SetTextColor(walk.RGB(128, 128, 128))
					}
				},
				ContextMenuItems: []MenuItem{
					Action{Text: "⚡ 快速命令", OnTriggered: openQuickCmdDlg},
					Action{Text: "⚡ 快速执行脚本", OnTriggered: openQuickExecDlg},
					Action{Text: "📂 文件浏览", OnTriggered: openFileBrowser},
					Separator{},
					Action{Text: "✎ 编辑", OnTriggered: onEditConn},
					Action{Text: "🗑 删除", OnTriggered: onDelConn},
					Action{Text: "🔌 测试连接", OnTriggered: onTestConn},
				},
				StyleCell: func(style *walk.CellStyle) {
					col := style.Col()
					row := style.Row()
					// "操作" column = green-blue link text
					if col == 5 {
						style.TextColor = walk.RGB(0, 100, 200)
						return
					}
					// "名称" column = green if tested OK
					if col == 0 && row >= 0 {
						connMu.RLock()
						if row < len(connCache) {
							id := connCache[row].ID
							ok := isTestedOK(id)
							connMu.RUnlock()
							if ok {
								style.TextColor = walk.RGB(0, 128, 0)
								return
							}
						} else {
							connMu.RUnlock()
						}
					}
				},
				Columns: []TableViewColumn{
					{Title: "名称", Width: 120, DataMember: "名称"},
					{Title: "主机", Width: 130, DataMember: "主机"},
					{Title: "端口", Width: 50, Alignment: AlignCenter, DataMember: "端口"},
					{Title: "用户", Width: 80, DataMember: "用户"},
					{Title: "认证", Width: 60, Alignment: AlignCenter, DataMember: "认证"},
					{Title: "操作", Width: 60, Alignment: AlignCenter, DataMember: "操作"},
				},
			},
			Label{AssignTo: &connCountLabel, TextColor: walk.RGB(128, 128, 128)},
		},
	}
}

func buildScriptPanel() Widget {
	return Composite{
		Layout: VBox{Margins: Margins{2, 5, 5, 5}},
		Children: []Widget{
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					Label{Text: "📜 脚本管理", Font: Font{PointSize: 11, Bold: true}},
					Label{Text: "Scripts", TextColor: walk.RGB(128, 128, 128)},
				},
			},
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					PushButton{Text: "＋ 新建", MinSize: Size{80, 0}, OnClicked: onAddScript},
					PushButton{Text: "✎ 编辑", MinSize: Size{80, 0}, OnClicked: onEditScript},
					PushButton{Text: "🗑 删除", MinSize: Size{80, 0}, OnClicked: onDelScript},
				},
			},
			TableView{
				AssignTo:        &scriptTV,
				Model:           scriptData,
				MultiSelection:  false,
				OnItemActivated: onEditScript,
				Columns: []TableViewColumn{
					{Title: "名称", Width: 120, DataMember: "名称"},
					{Title: "解释器", Width: 80, Alignment: AlignCenter, DataMember: "解释器"},
					{Title: "内容预览", Width: 200, DataMember: "内容预览"},
				},
			},
			Label{AssignTo: &scriptCountLabel, TextColor: walk.RGB(128, 128, 128)},
		},
	}
}

func buildBottomBar() Widget {
	return Composite{
		Layout: VBox{Margins: Margins{3, 0, 3, 3}, Spacing: 4},
		Children: []Widget{
			// Row 1: Action buttons
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					PushButton{Text: "▶ 执行中心", MinSize: Size{110, 0}, OnClicked: openExecuteDlg},
					PushButton{Text: "📋 执行历史", MinSize: Size{110, 0}, OnClicked: openHistoryDlg},
					PushButton{Text: "📂 文件浏览", MinSize: Size{110, 0}, OnClicked: openFileBrowser},
					HSeparator{},
					PushButton{Text: "🔄 刷新", OnClicked: refreshAll},
					HSpacer{},
					Label{AssignTo: &statusLabel, Text: "就绪", TextColor: walk.RGB(100, 100, 100)},
				},
			},
			// Row 2: Quick command bar
			Composite{
				Layout: HBox{MarginsZero: true, Spacing: 6},
				Children: []Widget{
					Label{Text: "⚡ 快捷命令", Font: Font{PointSize: 9, Bold: true}},
					Label{AssignTo: &quickConnLabel, Text: "(未选择连接)", TextColor: walk.RGB(128, 128, 128)},
					LineEdit{
						AssignTo: &quickCmdInput,
						MinSize:  Size{200, 0},
						ToolTipText: "输入命令，回车快速执行",
						OnKeyUp: func(key walk.Key) {
							if key == walk.KeyReturn {
								execQuickCmd()
							}
						},
					},
					PushButton{Text: "▶ 执行", MinSize: Size{60, 0}, OnClicked: execQuickCmd},
				},
			},
		},
	}
}
