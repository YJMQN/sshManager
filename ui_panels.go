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
				},
			},
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					PushButton{Text: "＋ 新建", MinSize: Size{80, 0}, OnClicked: onAddConn},
					PushButton{Text: "✎ 编辑", MinSize: Size{80, 0}, OnClicked: onEditConn},
					PushButton{Text: "🗑 删除", MinSize: Size{80, 0}, OnClicked: onDelConn},
					PushButton{Text: "🔌 测试", MinSize: Size{80, 0}, OnClicked: onTestConn},
					PushButton{Text: "▶ 执行", MinSize: Size{80, 0}, OnClicked: func() {
						connIdx := connTV.CurrentIndex()
						if connIdx < 0 {
							walk.MsgBox(mainWnd, "提示", "请先选中一个连接", walk.MsgBoxIconInformation)
							return
						}
						openExecuteDlgWithConn(connIdx)
					}},
				},
			},
			TableView{
				AssignTo:        &connTV,
				Model:           connData,
				MultiSelection:  false,
				OnItemActivated: onEditConn,
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
		Layout: HBox{Margins: Margins{5, 0, 5, 5}},
		Children: []Widget{
			PushButton{Text: "▶ 执行中心", MinSize: Size{110, 0}, OnClicked: openExecuteDlg},
			PushButton{Text: "📋 执行历史", MinSize: Size{110, 0}, OnClicked: openHistoryDlg},
			HSeparator{},
			PushButton{Text: "🔄 刷新", OnClicked: refreshAll},
			HSpacer{},
			Label{AssignTo: &statusLabel, Text: "就绪", TextColor: walk.RGB(100, 100, 100)},
		},
	}
}

func refreshAll() {
	refreshConnData()
	refreshScriptData()
	updateCounts()
	setStatus("已刷新")
}

func updateCounts() {
	conns, _ := db.GetConnections()
	scripts, _ := db.GetScripts()
	connCountLabel.SetText(fmt.Sprintf("共 %d 个连接", len(conns)))
	scriptCountLabel.SetText(fmt.Sprintf("共 %d 个脚本", len(scripts)))
}

func setStatus(s string) {
	if statusLabel != nil {
		statusLabel.SetText(s)
	}
}

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

func getConnByID(id int64) *Connection {
	conns, _ := db.GetConnections()
	for i := range conns {
		if conns[i].ID == id {
			return conns[i]
		}
	}
	return nil
}

func getScriptByID(id int64) *Script {
	scripts, _ := db.GetScripts()
	for i := range scripts {
		if scripts[i].ID == id {
			return scripts[i]
		}
	}
	return nil
}
