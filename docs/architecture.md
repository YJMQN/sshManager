# SSH Manager — 架构文档

> 版本: 1.0 | 更新: 2025-05-12

## 目录结构

```
ssh_manager_go/
├── main.go                 # 程序入口：数据库初始化、窗口创建、菜单
├── app.go                  # 全局状态 + 共享工具函数
├── models.go               # 数据模型 + 表格数据刷新
├── db.go                   # SQLite 持久层（增删改查）
├── ssh.go                  # SSH 客户端（连接、执行）
│
├── ui_panels.go            # 主窗口面板（连接管理 + 脚本管理）
├── ui_actions.go           # 连接/脚本 CRUD 对话框
├── ui_execute.go           # 执行中心 + 快捷执行
├── ui_history.go           # 执行历史对话框
├── ui_filebrowser.go       # 远程文件浏览器
│
├── assets/                 # 资源文件
│   ├── app.ico             # 程序图标
│   ├── app.rc              # Windows 资源脚本
│   └── app.manifest        # 清单文件（Win7 兼容）
│
├── scripts/                # 辅助脚本
│   └── mkicon.py           # 图标生成脚本
│
├── tests/                  # 测试文件
│   └── .gitkeep
│
├── docs/
│   ├── developer-guide.md  # 开发者文档
│   └── architecture.md     # 本文件
│
├── dist/                   # 构建产物
│   └── .gitkeep
│
├── build.bat               # Windows 构建脚本
├── build.sh                # Linux/macOS 交叉编译脚本
├── go.mod / go.sum         # Go 模块定义
├── README.md               # 用户文档
└── .gitignore
```

## 设计原则

| 原则 | 说明 |
|------|------|
| **单包架构** | 全部代码在 `package main` 中，避开 Go 子包跨文件变量可见性问题 |
| **功能分文件** | 每个文件对应一个功能域（DB / SSH / UI面板 / 对话框...） |
| **全局状态集中化** | `app.go` 声明 Walk 控件引用 + 共享函数，`models.go` 声明数据缓存 |
| **静态链接** | 编译时 `-extldflags=-static`，最终产物单 exe 零依赖 |
| **兼容 Win7** | Go 1.20, TDM-GCC 10.3, Walk 库, Win32 原生 API |

## 数据流

```
用户操作 → ui_actions.go 对话框
               ↓
         models.go 验证
               ↓
         db.go SQLite 持久化
               ↓
         models.go refreshConnData/refreshScriptData
               ↓
         app.go setModel → Walk TableView 自动刷新
```

## SSH 执行流

```
用户点击执行 → ui_execute.go or ui_filebrowser.go
               ↓
         ssh.go SSHClient.Connect()
               ↓
         ssh.go SSHClient.Execute()
               ↓
         回调写回输出控件 (goroutine → Synchronize)
               ↓
         db.go AddHistory()
```

## 模块职责

| 文件 | 职责 | 不负责 |
|------|------|--------|
| `main.go` | 程序入口、窗口布局、菜单 | 业务逻辑 |
| `app.go` | 全局变量、getConnID/setStatus 等共享函数 | 不包含 Walk 声明式布局 |
| `models.go` | 数据模型、缓存、表格刷新 | UI 交互、SSH 连接 |
| `db.go` | SQLite 增删改查 | 业务验证、UI |
| `ssh.go` | SSH 连接、命令执行 | UI、持久化 |
| `ui_panels.go` | 主窗口左右面板 + 底部栏布局 | 对话框、执行 |
| `ui_actions.go` | 连接/脚本的新建编辑删除对话框 | 执行、文件浏览 |
| `ui_execute.go` | 执行中心、快捷命令、快速执行 | 文件浏览、历史 |
| `ui_history.go` | 执行历史对话框 | 执行、文件管理 |
| `ui_filebrowser.go` | 远程文件浏览 | 脚本执行 |

## 关键依赖

| 依赖 | 用途 | 版本 |
|------|------|------|
| `github.com/lxn/walk` | Win32 原生 GUI | v0.0.0-20210112... |
| `github.com/lxn/win` | Win32 API 绑定 | Walk 间接依赖 |
| `github.com/mattn/go-sqlite3` | SQLite 驱动 | CGO 启用 |
| `golang.org/x/crypto/ssh` | SSH 客户端 | Go 标准扩展 |
| `golang.org/x/net` | 网络层 | SSH 间接依赖 |

## 编译指南

详见 [developer-guide.md](developer-guide.md)。
