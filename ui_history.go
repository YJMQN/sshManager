package main

import (
	"fmt"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

func openHistoryDlg() {
	var dlg *walk.Dialog
	var tv *walk.TableView
	var countLbl *walk.Label

	// Keep a reference to current items, bypassing walk's internal model
	var currentItems []map[string]interface{}

	loadHistory := func() {
		items, err := db.GetHistory(500)
		if err != nil {
			items = nil
		}

		currentItems = make([]map[string]interface{}, len(items))
		for i, h := range items {
			currentItems[i] = map[string]interface{}{
				"id":              h.ID,
				"connection_id":   h.ConnectionID,
				"connection_name": h.ConnectionName,
				"script_id":       h.ScriptID,
				"script_name":     h.ScriptName,
				"interpreter":     h.Interpreter,
				"status":          h.Status,
				"output":          h.Output,
				"error":           h.Error,
				"started_at":      h.StartedAt,
				"finished_at":     h.FinishedAt,
				"duration_ms":     h.DurationMs,
			}
		}

		tv.SetModel(currentItems)
		countLbl.SetText(fmt.Sprintf("共 %d 条记录", len(items)))
	}

	_, err := Dialog{
		AssignTo: &dlg,
		Title:    "执行历史",
		MinSize:  Size{800, 450},
		Layout:   VBox{},
		Children: []Widget{
			Composite{
				Layout: HBox{},
				Children: []Widget{
					PushButton{Text: "🔄 刷新", OnClicked: loadHistory},
					PushButton{Text: "查看详情", OnClicked: func() {
						idx := tv.CurrentIndex()
						if idx < 0 || idx >= len(currentItems) {
							walk.MsgBox(dlg, "提示", "请先选中一条记录", walk.MsgBoxIconInformation)
							return
						}
						hid, _ := currentItems[idx]["id"].(int64)
						detail, err := db.GetHistoryDetail(hid)
						if err != nil || detail == nil {
							walk.MsgBox(dlg, "错误", "无法获取详情", walk.MsgBoxIconError)
							return
						}
						showDetailDlg(dlg, detail)
					}},
					PushButton{Text: "🗑 删除选中", OnClicked: func() {
						idx := tv.CurrentIndex()
						if idx < 0 || idx >= len(currentItems) {
							return
						}
						hid, _ := currentItems[idx]["id"].(int64)
						if walk.MsgBox(dlg, "确认", "确定删除此条记录？",
							walk.MsgBoxOKCancel|walk.MsgBoxIconQuestion) == walk.DlgCmdOK {
							db.DeleteHistoryByID(hid)
							loadHistory()
						}
					}},
					HSpacer{},
					PushButton{Text: "清空全部", OnClicked: func() {
						if walk.MsgBox(dlg, "确认", "确定清空所有执行历史？此操作不可撤销！",
							walk.MsgBoxOKCancel|walk.MsgBoxIconWarning) == walk.DlgCmdOK {
							db.ClearHistory()
							loadHistory()
						}
					}},
				},
			},
			TableView{
				AssignTo:          &tv,
				ColumnsOrderable:  true,
				MultiSelection:    false,
				OnItemActivated: func() {
					// handled by 查看详情 button
				},
				Columns: []TableViewColumn{
					{Title: "ID", Width: 40, Alignment: AlignCenter},
					{Title: "目标连接", Width: 120},
					{Title: "脚本", Width: 120},
					{Title: "解释器", Width: 70, Alignment: AlignCenter},
					{Title: "状态", Width: 80},
					{Title: "时间", Width: 140},
					{Title: "耗时(ms)", Width: 70, Alignment: AlignFar},
				},
			},
			Label{AssignTo: &countLbl, Text: "共 0 条记录", TextColor: walk.RGB(128, 128, 128)},
		},
	}.Run(mainWnd)

	if err != nil {
		walk.MsgBox(mainWnd, "错误", err.Error(), walk.MsgBoxIconError)
	}
}

func showDetailDlg(owner walk.Form, h *ExecHistory) {
	statusEmoji := map[string]string{
		"running": "⏳ 运行中", "success": "✅ 成功", "error": "❌ 失败",
	}[h.Status]
	if statusEmoji == "" {
		statusEmoji = h.Status
	}

	output := h.Output
	if output == "" {
		output = "(无输出)"
	}
	errText := h.Error
	if errText == "" {
		errText = "(无错误)"
	}

	var dlg *walk.Dialog

	_, err := Dialog{
		AssignTo: &dlg,
		Title:    fmt.Sprintf("执行详情 #%d", h.ID),
		MinSize:  Size{600, 450},
		Layout:   VBox{},
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 2, Spacing: 6},
				Children: []Widget{
					Label{Text: "状态:"},
					Label{Text: statusEmoji, Font: Font{PointSize: 11, Bold: true}},
					Label{Text: "目标连接:"},
					Label{Text: h.ConnectionName},
					Label{Text: "脚本:"},
					Label{Text: h.ScriptName},
					Label{Text: "解释器:"},
					Label{Text: h.Interpreter},
					Label{Text: "开始时间:"},
					Label{Text: h.StartedAt},
					Label{Text: "完成时间:"},
					Label{Text: h.FinishedAt},
					Label{Text: "耗时:"},
					Label{Text: fmt.Sprintf("%d ms", h.DurationMs)},
				},
			},
			Label{Text: "stdout 输出:", Font: Font{PointSize: 9, Bold: true}},
			FormattedOutput(output, 120),
			Label{Text: "stderr 错误:", Font: Font{PointSize: 9, Bold: true}},
			FormattedOutput(errText, 80),
			Composite{
				Layout: HBox{},
				Children: []Widget{
					PushButton{Text: "📥 导出日志", OnClicked: func() {
						report := fmt.Sprintf("执行详情 #%d\n", h.ID) +
							fmt.Sprintf("状态: %s\n", statusEmoji) +
							fmt.Sprintf("目标连接: %s\n", h.ConnectionName) +
							fmt.Sprintf("脚本: %s\n", h.ScriptName) +
							fmt.Sprintf("解释器: %s\n", h.Interpreter) +
							fmt.Sprintf("开始时间: %s\n", h.StartedAt) +
							fmt.Sprintf("完成时间: %s\n", h.FinishedAt) +
							fmt.Sprintf("耗时: %d ms\n\n", h.DurationMs) +
							"=== stdout 输出 ===\n" + output + "\n\n" +
							"=== stderr 错误 ===\n" + errText
						exportText(dlg, report, fmt.Sprintf("history_%d_%s.txt", h.ID, time.Now().Format("20060102_150405")))
					}},
					HSpacer{},
					PushButton{Text: "关闭", OnClicked: func() { dlg.Cancel() }},
				},
			},
		},
	}.Run(owner)

	if err != nil {
		walk.MsgBox(owner, "错误", err.Error(), walk.MsgBoxIconError)
	}
}
