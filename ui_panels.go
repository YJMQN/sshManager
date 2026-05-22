package main

import (
	"fmt"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

func buildConnPanel() Widget {
	return Composite{
		Layout: VBox{Margins: Margins{10, 10, 5, 10}, Spacing: 8},
		Children: []Widget{
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					Label{Text: iconConn + " 连接管理", Font: fontTitle, TextColor: colorTitle},
					Label{Text: "SSH Connections", Font: fontUI, TextColor: colorTextMuted},
					HSpacer{},
					Label{Text: iconSearch, TextColor: colorTextMuted},
					LineEdit{
						AssignTo:    &connSearchInput,
						MinSize:     Size{120, 0},
						MaxSize:     Size{200, 24},
						Font:        fontUI,
						ToolTipText: "输入名称或主机过滤",
						OnKeyUp: func(key walk.Key) {
							refreshConnData()
						},
					},
				},
			},
			Composite{
				Layout: HBox{Margins: Margins{0, 6, 0, 6}, Spacing: 6},
				Children: []Widget{
					PushButton{
						Text:        iconAdd + " 新建",
						MinSize:     Size{85, 28},
						Font:        fontUI,
						ToolTipText: "添加新的 SSH 连接 (Ctrl+N)",
						OnClicked:   onAddConn,
					},
					PushButton{
						Text:        iconEdit + " 编辑",
						MinSize:     Size{85, 28},
						Font:        fontUI,
						ToolTipText: "编辑选中的连接",
						OnClicked:   onEditConn,
					},
					PushButton{
						Text:        iconDelete + " 删除",
						MinSize:     Size{85, 28},
						Font:        fontUI,
						ToolTipText: "删除选中的连接",
						OnClicked:   onDelConn,
					},
					PushButton{
						Text:        iconTest + " 测试",
						MinSize:     Size{85, 28},
						Font:        fontUI,
						ToolTipText: "测试当前 SSH 连接是否可用",
						OnClicked:   onTestConn,
					},
					PushButton{
						Text:        iconFile + " 文件",
						MinSize:     Size{80, 28},
						Font:        fontUI,
						ToolTipText: "浏览远程文件系统",
						OnClicked:   openFileBrowser,
					},
					PushButton{
						Text:        iconRun + " 执行",
						MinSize:     Size{85, 28},
						Font:        fontUI,
						ToolTipText: "打开执行中心",
						OnClicked: func() {
							connIdx := connTV.CurrentIndex()
							if connIdx < 0 {
								walk.MsgBox(mainWnd, "提示", "请先选中一个连接", walk.MsgBoxIconInformation)
								return
							}
							openExecuteDlgWithConn(connIdx)
						},
					},
					PushButton{
						Text:        iconQuick + " 快速执行",
						MinSize:     Size{95, 28},
						Font:        fontUI,
						ToolTipText: "快速执行脚本到当前连接",
						OnClicked:   openQuickExecDlg,
					},
				},
			},
			TableView{
				AssignTo:        &connTV,
				Model:           connData,
				MultiSelection:  false,
				Font:            fontUI,
				OnItemActivated: openQuickCmdDlg,
				OnCurrentIndexChanged: func() {
					idx := connTV.CurrentIndex()
					conns, _ := db.GetConnections()
					if idx >= 0 && idx < len(conns) {
						c := conns[idx]
						quickConnLabel.SetText(fmt.Sprintf("%s %s (%s:%d)", iconRun, c.Name, c.Host, c.Port))
						quickConnLabel.SetTextColor(colorTitle)
					} else {
						quickConnLabel.SetText("(未选择连接)")
						quickConnLabel.SetTextColor(colorTextMuted)
					}
				},
				ContextMenuItems: []MenuItem{
					Action{
						Text:        iconQuick + " 快速命令",
						OnTriggered: openQuickCmdDlg,
					},
					Action{
						Text:        iconQuick + " 快速执行脚本",
						OnTriggered: openQuickExecDlg,
					},
					Action{
						Text:        iconFile + " 文件浏览",
						OnTriggered: openFileBrowser,
					},
					Separator{},
					Action{
						Text:        iconEdit + " 编辑",
						OnTriggered: onEditConn,
					},
					Action{
						Text:        iconDelete + " 删除",
						OnTriggered: onDelConn,
					},
					Action{
						Text:        iconTest + " 测试连接",
						OnTriggered: onTestConn,
					},
				},
				StyleCell: func(style *walk.CellStyle) {
					col := style.Col()
					row := style.Row()
					if col == 5 {
						style.TextColor = colorLink
						return
					}
					if col == 0 && row >= 0 {
						connMu.RLock()
						if row < len(connCache) {
							id := connCache[row].ID
							ok := isTestedOK(id)
							connMu.RUnlock()
							if ok {
								style.TextColor = colorSuccessUI
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
			Label{AssignTo: &connCountLabel, Font: fontUI, TextColor: colorTextMuted},
		},
	}
}

func buildScriptPanel() Widget {
	return Composite{
		Layout: VBox{Margins: Margins{2, 10, 10, 10}, Spacing: 8},
		Children: []Widget{
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					Label{Text: iconScript + " 脚本管理", Font: fontTitle, TextColor: colorTitle},
					Label{Text: "Scripts", Font: fontUI, TextColor: colorTextMuted},
				},
			},
			Composite{
				Layout: HBox{Margins: Margins{0, 6, 0, 6}, Spacing: 6},
				Children: []Widget{
					PushButton{
						Text:        iconAdd + " 新建",
						MinSize:     Size{85, 28},
						Font:        fontUI,
						ToolTipText: "创建新脚本",
						OnClicked:   onAddScript,
					},
					PushButton{
						Text:        iconEdit + " 编辑",
						MinSize:     Size{85, 28},
						Font:        fontUI,
						ToolTipText: "编辑选中的脚本",
						OnClicked:   onEditScript,
					},
					PushButton{
						Text:        iconDelete + " 删除",
						MinSize:     Size{85, 28},
						Font:        fontUI,
						ToolTipText: "删除选中的脚本",
						OnClicked:   onDelScript,
					},
				},
			},
			TableView{
				AssignTo:        &scriptTV,
				Model:           scriptData,
				MultiSelection:  false,
				Font:            fontUI,
				OnItemActivated: onEditScript,
				Columns: []TableViewColumn{
					{Title: "名称", Width: 120, DataMember: "名称"},
					{Title: "解释器", Width: 80, Alignment: AlignCenter, DataMember: "解释器"},
					{Title: "内容预览", Width: 200, DataMember: "内容预览"},
				},
			},
			Label{AssignTo: &scriptCountLabel, Font: fontUI, TextColor: colorTextMuted},
		},
	}
}

func buildBottomBar() Widget {
	return Composite{
		Layout: VBox{Margins: Margins{5, 0, 5, 5}, Spacing: 6},
		Children: []Widget{
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					PushButton{
						Text:        iconExec + " 执行中心",
						MinSize:     Size{110, 28},
						Font:        fontUI,
						ToolTipText: "打开执行中心 (Ctrl+E)",
						OnClicked:   openExecuteDlg,
					},
					PushButton{
						Text:        iconHistory + " 执行历史",
						MinSize:     Size{110, 28},
						Font:        fontUI,
						ToolTipText: "查看执行历史 (Ctrl+H)",
						OnClicked:   openHistoryDlg,
					},
					PushButton{
						Text:        iconFile + " 文件浏览",
						MinSize:     Size{110, 28},
						Font:        fontUI,
						ToolTipText: "浏览远程文件系统",
						OnClicked:   openFileBrowser,
					},
					HSeparator{},
					PushButton{
						Text:        iconRefresh + " 刷新",
						Font:        fontUI,
						ToolTipText: "刷新所有数据",
						OnClicked:   refreshAll,
					},
					HSpacer{},
					Label{AssignTo: &statusLabel, Text: "就绪", Font: fontUI, TextColor: colorTextDark},
				},
			},
			Composite{
				Layout: HBox{MarginsZero: true, Spacing: 6},
				Children: []Widget{
					Label{Text: iconQuick + " 快捷命令", Font: fontUIBold},
					Label{AssignTo: &quickConnLabel, Text: "(未选择连接)", Font: fontUI, TextColor: colorTextMuted},
					LineEdit{
						AssignTo:    &quickCmdInput,
						MinSize:     Size{200, 0},
						Font:        fontCodeSmall,
						ToolTipText: "输入命令，回车快速执行",
						OnKeyUp: func(key walk.Key) {
							if key == walk.KeyReturn {
								execQuickCmd()
							}
						},
					},
					PushButton{
						Text:        iconRun + " 执行",
						MinSize:     Size{70, 26},
						Font:        fontUI,
						ToolTipText: "执行输入的命令",
						OnClicked:   execQuickCmd,
					},
				},
			},
		},
	}
}
