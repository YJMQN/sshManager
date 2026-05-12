package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

func openExecuteDlg() {
	openExecuteDlgWithConn(-1)
}

func openExecuteDlgWithConn(connIdx int) {
	var (
		dlg        *walk.Dialog
		connCB     *walk.ComboBox
		scriptCB   *walk.ComboBox
		outputTE   *walk.TextEdit
		runBtn     *walk.PushButton
		stopBtn    *walk.PushButton
		statusLbl  *walk.Label
		modeScript *walk.RadioButton
		modeCmd    *walk.RadioButton
		cmdInput   *walk.TextEdit
		cmdLabel   *walk.Label
	)

	conns, _ := db.GetConnections()
	scripts, _ := db.GetScripts()

	connNames := make([]string, len(conns))
	for i, c := range conns {
		connNames[i] = fmt.Sprintf("%s (%s:%d)", c.Name, c.Host, c.Port)
	}
	if len(connNames) == 0 {
		connNames = []string{"(无可用连接)"}
	}

	scriptNames := make([]string, len(scripts))
	for i, s := range scripts {
		scriptNames[i] = s.Name
	}
	if len(scriptNames) == 0 {
		scriptNames = []string{"(无可用脚本)"}
	}

	// Validate connIdx
	initConnIdx := 0
	if connIdx >= 0 && connIdx < len(conns) {
		initConnIdx = connIdx
	}

	var (
		mu            sync.Mutex
		running       bool
		currentClient *SSHClient
	)

	writeOut := func(text string) {
		if outputTE != nil {
			outputTE.AppendText(text)
		}
	}

	resetUI := func() {
		runBtn.SetEnabled(true)
		stopBtn.SetEnabled(false)
		mu.Lock()
		running = false
		mu.Unlock()
		if statusLbl != nil {
			statusLbl.SetText("就绪")
		}
		setStatus("就绪")
		updateCounts()
	}

	// Toggle script/cmd UI visibility
	toggleMode := func() {
		isCmd := modeCmd.Checked()
		scriptCB.SetEnabled(!isCmd)
		cmdInput.SetEnabled(isCmd)
		cmdLabel.SetEnabled(isCmd)
		if isCmd {
			cmdInput.SetFocus()
		}
	}

	execFunc := func() {
		mu.Lock()
		if running {
			mu.Unlock()
			return
		}
		running = true
		mu.Unlock()

		runBtn.SetEnabled(false)
		stopBtn.SetEnabled(true)

		connIdx := connCB.CurrentIndex()
		if connIdx < 0 || connIdx >= len(conns) {
			writeOut("请选择有效的连接\n")
			resetUI()
			return
		}

		conn := conns[connIdx]
		isCmdMode := modeCmd.Checked()

		var script *Script
		var scriptName, interpreter, content string

		if isCmdMode {
			// Direct command input mode
			cmdText := cmdInput.Text()
			if strings.TrimSpace(cmdText) == "" {
				writeOut("请输入要执行的命令\n")
				resetUI()
				return
			}
			scriptName = "(自定义命令)"
			interpreter = "sh"
			content = cmdText
			script = &Script{Name: scriptName, Interpreter: interpreter, Content: content}
		} else {
			// Script mode
			scriptIdx := scriptCB.CurrentIndex()
			if scriptIdx < 0 || scriptIdx >= len(scripts) {
				writeOut("请选择有效的脚本\n")
				resetUI()
				return
			}
			script = scripts[scriptIdx]
			scriptName = script.Name
			interpreter = script.Interpreter
			content = script.Content
		}

		// Add history record
		hid, _ := db.AddHistory(&ExecHistory{
			ConnectionID:   conn.ID,
			ConnectionName: conn.Name,
			ScriptID:       script.ID,
			ScriptName:     scriptName,
			Interpreter:    interpreter,
			Status:         "running",
		})

		outputTE.SetText("")
		writeOut(fmt.Sprintf("===== %s =====\n", time.Now().Format("15:04:05")))
		writeOut(fmt.Sprintf("▶ 连接: %s (%s:%d)\n", conn.Name, conn.Host, conn.Port))
		if isCmdMode {
			writeOut(fmt.Sprintf("▶ 命令: %s\n", content))
		} else {
			writeOut(fmt.Sprintf("▶ 脚本: %s | 解释器: %s\n", scriptName, interpreter))
		}
		writeOut(strings.Repeat("=", 50) + "\n")

		statusLbl.SetText("⏳ 执行中...")
		setStatus("⏳ 执行中...")

		go func(c *Connection, s *Script, hID int64) {
			startMs := time.Now().UnixMilli()
			client := &SSHClient{}
			mu.Lock()
			currentClient = client
			mu.Unlock()

			var outBuf, errBuf strings.Builder

			err := client.Connect(c.Host, c.Port, c.Username,
				c.AuthType, c.Password, c.KeyPath, 10*time.Second)
			if err != nil {
				errMsg := fmt.Sprintf("❌ 连接失败: %v\n", err)
				writeOut(errMsg)
				db.UpdateHistory(hID, "error", "", err.Error(),
					int(time.Now().UnixMilli()-startMs))
				mu.Lock()
				currentClient = nil
				mu.Unlock()
				resetUI()
				return
			}

			// Build command
			cmd := buildCommand(s.Interpreter, s.Content)
			_, stderr, exitCode, execErr := client.Execute(cmd,
				func(line, stream string) {
					writeOut(line)
					if stream == "stdout" {
						outBuf.WriteString(line)
					} else {
						errBuf.WriteString(line)
					}
				})

			elapsed := int(time.Now().UnixMilli() - startMs)
			client.Close()
			mu.Lock()
			currentClient = nil
			mu.Unlock()

			if execErr != nil {
				writeOut(fmt.Sprintf("\n❌ 执行异常: %v\n", execErr))
				db.UpdateHistory(hID, "error", outBuf.String(), execErr.Error(), elapsed)
			} else if exitCode != 0 {
				writeOut(fmt.Sprintf("\n⚠️ 退出码: %d (耗时: %dms)\n", exitCode, elapsed))
				db.UpdateHistory(hID, "error", outBuf.String(), errBuf.String(), elapsed)
			} else {
				writeOut(fmt.Sprintf("\n✅ 完成 (退出码: 0, 耗时: %dms)\n", elapsed))
				db.UpdateHistory(hID, "success", outBuf.String(), errBuf.String(), elapsed)
			}
			if stderr != "" {
				writeOut("\n--- STDERR ---\n" + stderr)
			}
			resetUI()
		}(conn, script, hid)
	}

	_, err := Dialog{
		AssignTo: &dlg,
		Title:    "执行中心",
		MinSize:  Size{700, 620},
		Layout:   VBox{Margins: Margins{10, 10, 10, 10}},
		Children: []Widget{
			// Connection selector
			Composite{
				Layout: HBox{Spacing: 8},
				Children: []Widget{
					Label{Text: "目标连接:"},
					ComboBox{AssignTo: &connCB, Model: connNames, CurrentIndex: initConnIdx, MinSize: Size{200, 0}},
				},
			},
			// Mode selector
			Composite{
				Layout: HBox{Spacing: 8},
				Children: []Widget{
					Label{Text: "执行方式:"},
					RadioButton{AssignTo: &modeScript, Text: "执行脚本", OnClicked: toggleMode},
					RadioButton{AssignTo: &modeCmd, Text: "输入命令", OnClicked: toggleMode},
				},
			},
			// Script selector (visible in script mode)
			Composite{
				Layout: HBox{Spacing: 8},
				Children: []Widget{
					Label{AssignTo: &cmdLabel, Text: "选择脚本:"},
					ComboBox{AssignTo: &scriptCB, Model: scriptNames, CurrentIndex: 0, MinSize: Size{200, 0}},
				},
			},
			// Command input (visible in command mode)
			Label{Text: "输入命令:", Font: Font{PointSize: 9}, TextColor: walk.RGB(100, 100, 100)},
			TextEdit{
				AssignTo: &cmdInput, Enabled: false,
				Font:    Font{PointSize: 10, Family: "Consolas"},
				MinSize: Size{0, 80},
				MaxSize: Size{0, 150},
				VScroll: true,
			},
			// Action buttons
			Composite{
				Layout: HBox{Spacing: 8},
				Children: []Widget{
					PushButton{AssignTo: &runBtn, Text: "▶ 执行", OnClicked: execFunc},
					PushButton{AssignTo: &stopBtn, Text: "⏹ 停止", Enabled: false, OnClicked: func() {
						mu.Lock()
						if currentClient != nil {
							currentClient.Close()
						}
						mu.Unlock()
						writeOut("\n⏹ 用户终止\n")
						resetUI()
					}},
					PushButton{Text: "清空输出", OnClicked: func() { outputTE.SetText("") }},
					HSpacer{},
					PushButton{Text: "关闭", OnClicked: func() { dlg.Cancel() }},
				},
			},
			// Output area
			Label{Text: "执行输出:"},
			TextEdit{
				AssignTo: &outputTE, ReadOnly: true,
				Font:    Font{PointSize: 10, Family: "Consolas"},
				MinSize: Size{0, 280},
				VScroll: true,
				MaxSize: Size{0, 1000},
			},
			Label{AssignTo: &statusLbl, Text: "就绪", TextColor: walk.RGB(100, 100, 100)},
		},
	}.Run(mainWnd)

	// Set initial checked radio button after creation
	if modeScript != nil {
		modeScript.SetChecked(true)
	}

	if err != nil {
		walk.MsgBox(mainWnd, "错误", err.Error(), walk.MsgBoxIconError)
	}
}

func openQuickExecDlg() {
	connIdx := connTV.CurrentIndex()
	if connIdx < 0 {
		walk.MsgBox(mainWnd, "提示", "请先选中一个连接", walk.MsgBoxIconInformation)
		return
	}

	conns, _ := db.GetConnections()
	scripts, _ := db.GetScripts()

	if connIdx >= len(conns) {
		return
	}

	conn := conns[connIdx]

	scriptNames := make([]string, len(scripts))
	for i, s := range scripts {
		scriptNames[i] = s.Name
	}
	if len(scriptNames) == 0 {
		scriptNames = []string{"(无可用脚本，请先在脚本管理中创建)"}
	}

	// Check if there are scripts available
	hasScripts := len(scripts) > 0

	var (
		dlg       *walk.Dialog
		scriptCB  *walk.ComboBox
		outputTE  *walk.TextEdit
		runBtn    *walk.PushButton
		stopBtn   *walk.PushButton
		statusLbl *walk.Label
		mu        sync.Mutex
		running   bool
		cli       *SSHClient
	)

	writeOut := func(text string) {
		if outputTE != nil {
			outputTE.AppendText(text)
		}
	}

	resetUI := func() {
		runBtn.SetEnabled(true)
		stopBtn.SetEnabled(false)
		mu.Lock()
		running = false
		mu.Unlock()
		if statusLbl != nil {
			statusLbl.SetText("就绪")
		}
	}

	_, err := Dialog{
		AssignTo: &dlg,
		Title:    fmt.Sprintf("⚡ 快速执行 — %s (%s:%d)", conn.Name, conn.Host, conn.Port),
		MinSize:  Size{580, 420},
		Layout:   VBox{Margins: Margins{10, 10, 10, 10}},
		Children: []Widget{
			// Connection info + Script selector
			Composite{
				Layout: HBox{Spacing: 8},
				Children: []Widget{
					Label{Text: "📡 主机:", Font: Font{PointSize: 10, Bold: true}},
					Label{Text: fmt.Sprintf("%s (%s:%d)", conn.Name, conn.Host, conn.Port),
						TextColor: walk.RGB(0, 80, 160)},
					Label{Text: "    选择脚本:"},
					ComboBox{
						AssignTo:     &scriptCB,
						Model:        scriptNames,
						CurrentIndex: 0,
						MinSize:      Size{180, 0},
						Enabled:      hasScripts,
					},
				},
			},
			HSeparator{},
			// Action buttons
			Composite{
				Layout: HBox{Spacing: 8},
				Children: []Widget{
					PushButton{AssignTo: &runBtn, Text: "▶ 执行", Enabled: hasScripts, OnClicked: func() {
						if !hasScripts {
							return
						}
						mu.Lock()
						if running {
							mu.Unlock()
							return
						}
						running = true
						mu.Unlock()

						runBtn.SetEnabled(false)
						stopBtn.SetEnabled(true)

						scriptIdx := scriptCB.CurrentIndex()
						if scriptIdx < 0 || scriptIdx >= len(scripts) {
							writeOut("请选择有效的脚本\n")
							resetUI()
							return
						}

						script := scripts[scriptIdx]

						hid, _ := db.AddHistory(&ExecHistory{
							ConnectionID:   conn.ID,
							ConnectionName: conn.Name,
							ScriptID:       script.ID,
							ScriptName:     script.Name,
							Interpreter:    script.Interpreter,
							Status:         "running",
						})

						outputTE.SetText("")
						writeOut(fmt.Sprintf("===== %s =====\n", time.Now().Format("15:04:05")))
						writeOut(fmt.Sprintf("▶ 连接: %s (%s:%d)\n", conn.Name, conn.Host, conn.Port))
						writeOut(fmt.Sprintf("▶ 脚本: %s | 解释器: %s\n", script.Name, script.Interpreter))
						writeOut(strings.Repeat("=", 50) + "\n")

						statusLbl.SetText("⏳ 执行中...")
						setStatus("⏳ 快速执行中...")

						go func(c *Connection, s *Script, hID int64) {
							startMs := time.Now().UnixMilli()
							client := &SSHClient{}
							mu.Lock()
							cli = client
							mu.Unlock()

							var outBuf, errBuf strings.Builder

							err := client.Connect(c.Host, c.Port, c.Username,
								c.AuthType, c.Password, c.KeyPath, 10*time.Second)
							if err != nil {
								errMsg := fmt.Sprintf("❌ 连接失败: %v\n", err)
								writeOut(errMsg)
								db.UpdateHistory(hID, "error", "", err.Error(),
									int(time.Now().UnixMilli()-startMs))
								mu.Lock()
								cli = nil
								mu.Unlock()
								resetUI()
								return
							}

							cmd := buildCommand(s.Interpreter, s.Content)
							_, stderr, exitCode, execErr := client.Execute(cmd,
								func(line, stream string) {
									writeOut(line)
									if stream == "stdout" {
										outBuf.WriteString(line)
									} else {
										errBuf.WriteString(line)
									}
								})

							elapsed := int(time.Now().UnixMilli() - startMs)
							client.Close()
							mu.Lock()
							cli = nil
							mu.Unlock()

							if execErr != nil {
								writeOut(fmt.Sprintf("\n❌ 执行异常: %v\n", execErr))
								db.UpdateHistory(hID, "error", outBuf.String(), execErr.Error(), elapsed)
							} else if exitCode != 0 {
								writeOut(fmt.Sprintf("\n⚠️ 退出码: %d (耗时: %dms)\n", exitCode, elapsed))
								db.UpdateHistory(hID, "error", outBuf.String(), errBuf.String(), elapsed)
							} else {
								writeOut(fmt.Sprintf("\n✅ 完成 (退出码: 0, 耗时: %dms)\n", elapsed))
								db.UpdateHistory(hID, "success", outBuf.String(), errBuf.String(), elapsed)
							}
							if stderr != "" {
								writeOut("\n--- STDERR ---\n" + stderr)
							}
							resetUI()
							setStatus(fmt.Sprintf("✅ 快速执行完成: %s → %s", conn.Name, s.Name))
						}(conn, script, hid)
					}},
					PushButton{AssignTo: &stopBtn, Text: "⏹ 停止", Enabled: false, OnClicked: func() {
						mu.Lock()
						if cli != nil {
							cli.Close()
						}
						mu.Unlock()
						writeOut("\n⏹ 用户终止\n")
						resetUI()
					}},
					PushButton{Text: "清空", OnClicked: func() { outputTE.SetText("") }},
					HSpacer{},
					PushButton{Text: "关闭", OnClicked: func() { dlg.Cancel() }},
				},
			},
			// Output
			Label{Text: "执行输出:"},
			TextEdit{
				AssignTo: &outputTE, ReadOnly: true,
				Font:    Font{PointSize: 10, Family: "Consolas"},
				MinSize: Size{0, 220},
				VScroll: true,
			},
			Label{AssignTo: &statusLbl, Text: "就绪", TextColor: walk.RGB(100, 100, 100)},
		},
	}.Run(mainWnd)

	if err != nil {
		walk.MsgBox(mainWnd, "错误", err.Error(), walk.MsgBoxIconError)
	}
}

// openQuickCmdDlg opens a minimal dialog for typing and executing a command quickly.
// Triggered by double-clicking a connection or via context menu.
func openQuickCmdDlg() {
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
	var cmdInput *walk.TextEdit
	var outputTE *walk.TextEdit
	var runBtn, closeBtn *walk.PushButton
	var statusLbl *walk.Label
	var running bool

	writeOut := func(text string) {
		if outputTE != nil {
			outputTE.AppendText(text)
		}
	}

	execFn := func() {
		if running {
			return
		}
		running = true
		runBtn.SetEnabled(false)

		cmdText := cmdInput.Text()
		if strings.TrimSpace(cmdText) == "" {
			writeOut("请输入要执行的命令\n")
			running = false
			runBtn.SetEnabled(true)
			return
		}

		outputTE.SetText("")
		writeOut(fmt.Sprintf("===== %s =====\n", time.Now().Format("15:04:05")))
		writeOut(fmt.Sprintf("▶ %s (%s:%d)\n", conn.Name, conn.Host, conn.Port))
		writeOut(fmt.Sprintf("▶ 命令: %s\n", cmdText))
		writeOut(strings.Repeat("=", 50) + "\n")

		statusLbl.SetText("⏳ 执行中...")

		go func() {
			startMs := time.Now().UnixMilli()
			client := &SSHClient{}

			err := client.Connect(conn.Host, conn.Port, conn.Username,
				conn.AuthType, conn.Password, conn.KeyPath, 10*time.Second)
			if err != nil {
				writeOut(fmt.Sprintf("❌ 连接失败: %v\n", err))
				running = false
				runBtn.SetEnabled(true)
				statusLbl.SetText("连接失败")
				return
			}

			cmd := buildCommand("sh", cmdText)
			_, stderr, exitCode, execErr := client.Execute(cmd,
				func(line, stream string) {
					writeOut(line)
				})

			elapsed := int(time.Now().UnixMilli() - startMs)
			client.Close()

			if execErr != nil {
				writeOut(fmt.Sprintf("\n❌ 执行异常: %v\n", execErr))
			} else if exitCode != 0 {
				writeOut(fmt.Sprintf("\n⚠️ 退出码: %d (耗时: %dms)\n", exitCode, elapsed))
			} else {
				writeOut(fmt.Sprintf("\n✅ 完成 (退出码: 0, 耗时: %dms)\n", elapsed))
			}
			if stderr != "" {
				writeOut("\n--- STDERR ---\n" + stderr)
			}
			running = false
			runBtn.SetEnabled(true)
			statusLbl.SetText("就绪")
		}()
	}

	_, err := Dialog{
		AssignTo: &dlg,
		Title:    fmt.Sprintf("⚡ 快速命令 — %s (%s:%d)", conn.Name, conn.Host, conn.Port),
		MinSize:  Size{600, 400},
		Layout:   VBox{Margins: Margins{10, 10, 10, 10}},
		Children: []Widget{
			Label{Text: "输入命令:", Font: Font{PointSize: 9, Bold: true}},
			TextEdit{
				AssignTo: &cmdInput,
				Font:     Font{PointSize: 10, Family: "Consolas"},
				MinSize:  Size{0, 60},
				MaxSize:  Size{0, 120},
				VScroll:  true,
			},
			Composite{
				Layout: HBox{Spacing: 8},
				Children: []Widget{
					PushButton{AssignTo: &runBtn, Text: "▶ 执行", OnClicked: execFn},
					PushButton{AssignTo: &closeBtn, Text: "关闭", OnClicked: func() { dlg.Cancel() }},
					HSpacer{},
					Label{AssignTo: &statusLbl, Text: "就绪", TextColor: walk.RGB(100, 100, 100)},
				},
			},
			Label{Text: "输出:"},
			TextEdit{
				AssignTo: &outputTE, ReadOnly: true,
				Font:    Font{PointSize: 10, Family: "Consolas"},
				MinSize: Size{0, 180},
				VScroll: true,
			},
		},
	}.Run(mainWnd)

	if err != nil {
		walk.MsgBox(mainWnd, "错误", err.Error(), walk.MsgBoxIconError)
	}
}

// execQuickCmd executes a command from the bottom quick command bar.
func execQuickCmd() {
	connIdx := connTV.CurrentIndex()
	if connIdx < 0 {
		walk.MsgBox(mainWnd, "提示", "请先选中一个连接", walk.MsgBoxIconInformation)
		return
	}

	cmdText := quickCmdInput.Text()
	if strings.TrimSpace(cmdText) == "" {
		return
	}

	conns, _ := db.GetConnections()
	if connIdx >= len(conns) {
		return
	}
	conn := conns[connIdx]

	// Update quick bar display
	quickConnLabel.SetText(fmt.Sprintf("▶ %s (%s:%d)", conn.Name, conn.Host, conn.Port))
	quickConnLabel.SetTextColor(walk.RGB(0, 128, 0))
	quickCmdInput.SetEnabled(false)

	setStatus(fmt.Sprintf("⏳ 执行: %s → %s", conn.Name, cmdText))

	go func() {
		startMs := time.Now().UnixMilli()
		client := &SSHClient{}
		var outBuf, errBuf strings.Builder

		err := client.Connect(conn.Host, conn.Port, conn.Username,
			conn.AuthType, conn.Password, conn.KeyPath, 10*time.Second)
		if err != nil {
			setStatus(fmt.Sprintf("❌ 连接失败: %s", err.Error()))
			quickCmdInput.SetEnabled(true)
			return
		}

		cmd := buildCommand("sh", cmdText)
		_, _, exitCode, execErr := client.Execute(cmd,
			func(line, stream string) {
				if stream == "stdout" {
					outBuf.WriteString(line)
				} else {
					errBuf.WriteString(line)
				}
			})

		elapsed := int(time.Now().UnixMilli() - startMs)
		client.Close()

		// Build result summary
		result := fmt.Sprintf("⚡ %s → %s", conn.Name, cmdText)
		result += fmt.Sprintf(" (耗时: %dms", elapsed)
		if execErr != nil {
			result += fmt.Sprintf(", 异常: %v", execErr)
		} else {
			result += fmt.Sprintf(", 退出码: %d", exitCode)
		}
		result += ")"

		// Show first line of output in status
		outLine := strings.TrimSpace(outBuf.String())
		if outLine != "" {
			// Take first line
			if idx := strings.Index(outLine, "\n"); idx > 0 {
				outLine = outLine[:idx]
			}
			if len(outLine) > 60 {
				outLine = outLine[:60] + "..."
			}
			result += " | " + outLine
		}

		setStatus(result)
		quickCmdInput.SetEnabled(true)
		quickCmdInput.SetText("")
		quickCmdInput.SetFocus()

		// Show error in quick bar label if failed
		if execErr != nil || exitCode != 0 {
			quickConnLabel.SetTextColor(walk.RGB(200, 0, 0))
		}
	}()
}

func buildCommand(interpreter, code string) string {
	switch interpreter {
	case "sh", "bash":
		code = strings.ReplaceAll(code, "'", "'\\''")
		return fmt.Sprintf("%s -c '%s'", interpreter, code)
	case "python3", "python", "perl", "ruby", "node", "php":
		return fmt.Sprintf("%s << 'SCRIPT_END'\n%s\nSCRIPT_END", interpreter, code)
	case "powershell":
		return fmt.Sprintf("powershell -Command \"%s\"", escapePS(code))
	default:
		return fmt.Sprintf("sh << 'SCRIPT_END'\n%s\nSCRIPT_END", code)
	}
}

func escapePS(s string) string {
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "; ")
	return s
}
