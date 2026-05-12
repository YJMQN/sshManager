package main

import (
	"fmt"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

// ============ Connection Actions ============

func onAddConn() {
	var dlg *walk.Dialog
	var name, host, port, user, pw *walk.LineEdit
	var authPW, authKey *walk.RadioButton

	_, err := Dialog{
		AssignTo: &dlg,
		Title:    "新建连接",
		MinSize:  Size{460, 280},
		Layout:   VBox{},
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 2, Spacing: 8},
				Children: []Widget{
					Label{Text: "连接名称 *:"},
					LineEdit{AssignTo: &name},
					Label{Text: "主机地址 *:"},
					LineEdit{AssignTo: &host},
					Label{Text: "端口 *:"},
					LineEdit{AssignTo: &port, Text: "22"},
					Label{Text: "用户名 *:"},
					LineEdit{AssignTo: &user},
					Label{Text: "认证方式:"},
					Composite{
						Layout: HBox{},
						Children: []Widget{
							RadioButton{AssignTo: &authPW, Text: "密码"},
							RadioButton{AssignTo: &authKey, Text: "密钥"},
						},
					},
					Label{Text: "密码 / 密钥路径:"},
					LineEdit{AssignTo: &pw},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					PushButton{Text: "浏览密钥...", OnClicked: func() {
						fd := new(walk.FileDialog)
						fd.Filter = "所有文件 (*.*)|*.*"
						fd.Title = "选择 SSH 私钥"
						if ok, _ := fd.ShowOpen(mainWnd); ok {
							pw.SetText(fd.FilePath)
						}
					}},
					HSpacer{},
					PushButton{Text: "确定", OnClicked: func() {
						if name.Text() == "" || host.Text() == "" || user.Text() == "" {
							walk.MsgBox(dlg, "输入错误", "请填写所有必填项", walk.MsgBoxIconError)
							return
						}
						authType := "password"
						if authKey.Checked() {
							authType = "key"
						}
						p := port.Text()
						portNum := 22
						fmt.Sscanf(p, "%d", &portNum)

						c := Connection{
							Name:     name.Text(),
							Host:     host.Text(),
							Port:     portNum,
							Username: user.Text(),
							AuthType: authType,
							Password: pw.Text(),
						}
						if _, err := db.AddConnection(&c); err != nil {
							walk.MsgBox(dlg, "错误", "添加失败: "+err.Error(), walk.MsgBoxIconError)
							return
						}
						refreshConnData()
						updateCounts()
						setStatus("✅ 连接 '" + c.Name + "' 已添加")
						dlg.Accept()
					}},
					PushButton{Text: "取消", OnClicked: func() { dlg.Cancel() }},
				},
			},
		},
	}.Run(mainWnd)

	// Set default radio button after creation
	if authPW != nil {
		authPW.SetChecked(true)
	}

	if err != nil {
		walk.MsgBox(mainWnd, "错误", err.Error(), walk.MsgBoxIconError)
	}
}

func onEditConn() {
	conn := getConnByID(getConnID())
	if conn == nil {
		walk.MsgBox(mainWnd, "提示", "请先选中一个连接", walk.MsgBoxIconInformation)
		return
	}

	var dlg *walk.Dialog
	var name, host, port, user, pw, keyPath *walk.LineEdit
	var authPW, authKey *walk.RadioButton

	_, err := Dialog{
		AssignTo: &dlg,
		Title:    "编辑连接 - " + conn.Name,
		MinSize:  Size{460, 300},
		Layout:   VBox{},
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 2, Spacing: 8},
				Children: []Widget{
					Label{Text: "连接名称 *:"},
					LineEdit{AssignTo: &name, Text: conn.Name},
					Label{Text: "主机地址 *:"},
					LineEdit{AssignTo: &host, Text: conn.Host},
					Label{Text: "端口 *:"},
					LineEdit{AssignTo: &port, Text: fmt.Sprintf("%d", conn.Port)},
					Label{Text: "用户名 *:"},
					LineEdit{AssignTo: &user, Text: conn.Username},
					Label{Text: "认证方式:"},
					Composite{
						Layout: HBox{},
						Children: []Widget{
							RadioButton{AssignTo: &authPW, Text: "密码"},
							RadioButton{AssignTo: &authKey, Text: "密钥"},
						},
					},
					Label{Text: "密码:"},
					LineEdit{AssignTo: &pw, Text: conn.Password},
					Label{Text: "密钥路径:"},
					LineEdit{AssignTo: &keyPath, Text: conn.KeyPath},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					PushButton{Text: "浏览密钥...", OnClicked: func() {
						fd := new(walk.FileDialog)
						fd.Filter = "所有文件 (*.*)|*.*"
						fd.Title = "选择 SSH 私钥"
						if ok, _ := fd.ShowOpen(mainWnd); ok {
							pw.SetText(fd.FilePath)
						}
					}},
					PushButton{Text: "🔌 测试连接", OnClicked: func() {
						authType := "password"
						if authKey.Checked() {
							authType = "key"
						}
						tmpConn := &Connection{
							Name:     name.Text(),
							Host:     host.Text(),
							Port:     22,
							Username: user.Text(),
							AuthType: authType,
							Password: pw.Text(),
							KeyPath:  keyPath.Text(),
						}
						fmt.Sscanf(port.Text(), "%d", &tmpConn.Port)

						client := &SSHClient{}
						ok, msg := client.TestConnection(tmpConn, 10)
						if ok {
							markTestedOK(conn.ID)
							walk.MsgBox(dlg, "✅ 测试成功",
								"连接测试通过!\n\n响应: "+msg,
								walk.MsgBoxIconInformation)
						} else {
							walk.MsgBox(dlg, "❌ 测试失败",
								"连接测试失败\n\n错误: "+msg,
								walk.MsgBoxIconError)
						}
					}},
					HSpacer{},
					PushButton{Text: "确定", OnClicked: func() {
						conn.Name = name.Text()
						conn.Host = host.Text()
						fmt.Sscanf(port.Text(), "%d", &conn.Port)
						conn.Username = user.Text()
						conn.AuthType = "password"
						if authKey.Checked() {
							conn.AuthType = "key"
						}
						conn.Password = pw.Text()
						conn.KeyPath = keyPath.Text()
						if err := db.UpdateConnection(conn); err != nil {
							walk.MsgBox(dlg, "错误", "更新失败: "+err.Error(), walk.MsgBoxIconError)
							return
						}
						refreshConnData()
						updateCounts()
						setStatus("✅ 连接 '" + conn.Name + "' 已更新")
						dlg.Accept()
					}},
					PushButton{Text: "取消", OnClicked: func() { dlg.Cancel() }},
				},
			},
		},
	}.Run(mainWnd)

	// Init auth type radio buttons based on existing conn
	if conn.AuthType == "key" {
		if authKey != nil {
			authKey.SetChecked(true)
		}
	} else {
		if authPW != nil {
			authPW.SetChecked(true)
		}
	}

	if err != nil {
		walk.MsgBox(mainWnd, "错误", err.Error(), walk.MsgBoxIconError)
	}
}

func onDelConn() {
	conn := getConnByID(getConnID())
	if conn == nil {
		walk.MsgBox(mainWnd, "提示", "请先选中一个连接", walk.MsgBoxIconInformation)
		return
	}
	if walk.MsgBox(mainWnd, "确认删除",
		fmt.Sprintf("确定删除连接 '%s' 吗？\n相关执行历史也会被删除。", conn.Name),
		walk.MsgBoxOKCancel|walk.MsgBoxIconWarning) == walk.DlgCmdOK {
		if err := db.DeleteConnection(conn.ID); err != nil {
			walk.MsgBox(mainWnd, "错误", "删除失败: "+err.Error(), walk.MsgBoxIconError)
			return
		}
		refreshConnData()
		updateCounts()
		setStatus("🗑 连接 '" + conn.Name + "' 已删除")
	}
}

func onTestConn() {
	conn := getConnByID(getConnID())
	if conn == nil {
		walk.MsgBox(mainWnd, "提示", "请先选中一个连接", walk.MsgBoxIconInformation)
		return
	}
	authLabel := "密码"
	if conn.AuthType == "key" {
		authLabel = "密钥"
	}
	detail := fmt.Sprintf("目标: %s@%s:%d\n认证: %s", conn.Username, conn.Host, conn.Port, authLabel)
	setStatus("🔌 正在测试连接 '" + conn.Name + "'... (" + conn.Host + ":" + fmt.Sprintf("%d", conn.Port) + ")")

	client := &SSHClient{}
	ok, msg := client.TestConnection(conn, 10)
	if ok {
		markTestedOK(conn.ID)
		setStatus("✅ 连接测试通过 (" + conn.Host + ":" + fmt.Sprintf("%d", conn.Port) + ")")
		walk.MsgBox(mainWnd, "✅ 测试成功",
			"连接 '"+conn.Name+"' 测试通过!\n\n"+detail+"\n\n响应: "+msg,
			walk.MsgBoxIconInformation)
	} else {
		errMsg := "❌ 连接测试失败: " + msg
		setStatus(errMsg)
		walk.MsgBox(mainWnd, "❌ 测试失败",
			"连接 '"+conn.Name+"' 测试失败\n\n"+detail+"\n\n错误: "+msg,
			walk.MsgBoxIconError)
	}

	// Refresh table to apply color updates
	refreshConnData()
}

// ============ Script Actions ============

func onAddScript() {
	var dlg *walk.Dialog
	var name, content *walk.TextEdit
	var interpreter *walk.ComboBox

	_, err := Dialog{
		AssignTo: &dlg,
		Title:    "新建脚本",
		MinSize:  Size{520, 420},
		Layout:   VBox{},
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 2, Spacing: 8},
				Children: []Widget{
					Label{Text: "脚本名称 *:"},
					TextEdit{AssignTo: &name, MinSize: Size{300, 24}, MaxSize: Size{0, 24}},
					Label{Text: "解释器:"},
					ComboBox{AssignTo: &interpreter, Model: interpreters, CurrentIndex: 0},
				},
			},
			Label{Text: "脚本内容:"},
			TextEdit{
				AssignTo: &content,
				Font:     Font{PointSize: 10, Family: "Consolas"},
				MinSize:  Size{0, 250},
				VScroll:  true,
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{Text: "确定", OnClicked: func() {
						n := name.Text()
						if n == "" {
							walk.MsgBox(dlg, "输入错误", "请输入脚本名称", walk.MsgBoxIconError)
							return
						}
						idx := interpreter.CurrentIndex()
						interp := "sh"
						if idx >= 0 && idx < len(interpreters) {
							interp = interpreters[idx]
						}
						s := Script{
							Name:        n,
							Content:     content.Text(),
							Interpreter: interp,
						}
						if _, err := db.AddScript(&s); err != nil {
							walk.MsgBox(dlg, "错误", "添加失败: "+err.Error(), walk.MsgBoxIconError)
							return
						}
						refreshScriptData()
						updateCounts()
						setStatus("✅ 脚本 '" + n + "' 已添加")
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

func onEditScript() {
	s := getScriptByID(getScriptID())
	if s == nil {
		walk.MsgBox(mainWnd, "提示", "请先选中一个脚本", walk.MsgBoxIconInformation)
		return
	}

	var dlg *walk.Dialog
	var name, content *walk.TextEdit
	var interpreter *walk.ComboBox

	curIdx := 0
	for i, v := range interpreters {
		if v == s.Interpreter {
			curIdx = i
			break
		}
	}

	_, err := Dialog{
		AssignTo: &dlg,
		Title:    "编辑脚本 - " + s.Name,
		MinSize:  Size{520, 420},
		Layout:   VBox{},
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 2, Spacing: 8},
				Children: []Widget{
					Label{Text: "脚本名称 *:"},
					TextEdit{AssignTo: &name, MinSize: Size{300, 24}, MaxSize: Size{0, 24}, Text: s.Name},
					Label{Text: "解释器:"},
					ComboBox{AssignTo: &interpreter, Model: interpreters, CurrentIndex: curIdx},
				},
			},
			Label{Text: "脚本内容:"},
			TextEdit{
				AssignTo: &content,
				Font:     Font{PointSize: 10, Family: "Consolas"},
				MinSize:  Size{0, 250},
				VScroll:  true,
				Text:     s.Content,
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{Text: "确定", OnClicked: func() {
						s.Name = name.Text()
						idx := interpreter.CurrentIndex()
						if idx >= 0 && idx < len(interpreters) {
							s.Interpreter = interpreters[idx]
						}
						s.Content = content.Text()
						if err := db.UpdateScript(s); err != nil {
							walk.MsgBox(dlg, "错误", "更新失败: "+err.Error(), walk.MsgBoxIconError)
							return
						}
						refreshScriptData()
						updateCounts()
						setStatus("✅ 脚本 '" + s.Name + "' 已更新")
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

func onDelScript() {
	s := getScriptByID(getScriptID())
	if s == nil {
		walk.MsgBox(mainWnd, "提示", "请先选中一个脚本", walk.MsgBoxIconInformation)
		return
	}
	if walk.MsgBox(mainWnd, "确认删除",
		fmt.Sprintf("确定删除脚本 '%s' 吗？\n相关执行历史也会被删除。", s.Name),
		walk.MsgBoxOKCancel|walk.MsgBoxIconWarning) == walk.DlgCmdOK {
		if err := db.DeleteScript(s.ID); err != nil {
			walk.MsgBox(mainWnd, "错误", "删除失败: "+err.Error(), walk.MsgBoxIconError)
			return
		}
		refreshScriptData()
		updateCounts()
		setStatus("🗑 脚本 '" + s.Name + "' 已删除")
	}
}
