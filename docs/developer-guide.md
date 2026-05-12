# 开发者文档

> SSH Manager — 技术架构、开发环境搭建、项目结构及二次开发指南

---

## 📋 技术栈

| 组件 | 选型 | 说明 |
|------|------|------|
| 语言 | **Go 1.20** | Win7 最后支持的 Go 版本 |
| GUI 框架 | **Walk** (github.com/lxn/walk) | 原生 Win32 GUI，无 Web 包袱 |
| 数据库 | **SQLite** (github.com/mattn/go-sqlite3) | CGO 版，单文件数据库 |
| SSH 客户端 | **golang.org/x/crypto/ssh** | 标准 SSH 库 |
| 编译器 | **TDM-GCC 10.3.0** (MinGW-w64) | Windows 原生编译 |
| 压缩 | **UPX** (可选) | 减小 exe 体积 |

---

## 🛠 开发环境搭建

### Windows 环境

```cmd
:: 1. 安装 Go 1.20
::    下载 https://go.dev/dl/go1.20.14.windows-amd64.msi

:: 2. 安装 TDM-GCC 10.3.0（含 windres）
::    下载 https://jmeubank.github.io/tdm-gcc/

:: 3. 验证
go version         → go1.20.x
gcc --version      → 10.3.0
windres --version

:: 4. 设置国内镜像（可选）
go env -w GO111MODULE=on
go env -w GOPROXY=https://goproxy.cn,direct

:: 5. 克隆
git clone https://gitee.com/yjmqn/ssh-manager.git
cd ssh-manager
```

---

## 📁 项目结构

```
ssh-manager/
│
├── main.go                # 入口：数据库初始化、窗口创建、菜单
├── app.go                 # 全局变量 + 共享工具函数
├── models.go              # 数据模型 + 表格数据缓存 + 刷新
├── db.go                  # SQLite 持久层
├── ssh.go                 # SSH 客户端封装
│
├── ui_panels.go           # 主窗口面板布局
├── ui_actions.go          # 连接/脚本 CRUD 对话框
├── ui_execute.go          # 执行中心 + 快捷命令
├── ui_history.go          # 执行历史对话框
├── ui_filebrowser.go      # 远程文件浏览器
│
├── assets/                # 资源文件
│   ├── app.ico            # 程序图标
│   ├── app.rc             # Windows 资源脚本
│   └── app.manifest       # 通用控件清单（Win7 兼容）
│
├── scripts/
│   └── mkicon.py          # 图标生成脚本
│
├── tests/                 # 测试文件目录
│
├── docs/
│   ├── developer-guide.md # 本文件
│   └── architecture.md    # 架构设计文档
│
├── dist/                  # 构建产物
├── build.bat              # Windows 构建脚本
├── build.sh               # Linux/macOS 交叉编译脚本
├── go.mod / go.sum        # 模块依赖
├── README.md              # 用户文档
└── .gitignore
```

### 文件命名约定

| 前缀 | 含义 |
|------|------|
| `没有前缀` | 核心逻辑（db, ssh, models, main, app） |
| `ui_*` | 界面相关（面板、对话框、文件浏览器） |
| `assets/` | 编译期资源 |
| `scripts/` | 开发辅助工具 |
| `dist/` | 构建产物（gitignore） |

---

## 🔧 构建

### Windows

```cmd
:: 一键构建
build.bat

:: 或手动分步
cd assets
windres -o ..\app.syso -i app.rc
cd ..
go mod tidy
set CGO_ENABLED=1
go build -ldflags="-s -w -H windowsgui -extldflags=-static" -o dist\SSHManager.exe .
```

### Linux/macOS (交叉编译)

```bash
# 需要 MinGW-w64
# Ubuntu: sudo apt install gcc-mingw-w64-x86-64
chmod +x build.sh
./build.sh          # debug
./build.sh --release   # 发布版
```

### 发布版

```cmd
build.bat --release
```
会自动启用 UPX 压缩，产物约 2~3MB。

---

## 🧱 架构要点

### Walk TableView 数据绑定

Walk `TableView` 要求数据源为 `[]map[string]interface{}`：

```go
// models.go — refreshConnData()
connData[i] = map[string]interface{}{
    "名称": c.Name,    // DataMember 必须匹配列标题
    "主机": c.Host,
}
```

额外数据（密码、密钥路径等）通过 `connCache[]` / `scriptCache[]` 独立存储，
通过行索引反查。

### CellStyler 机制

```go
StyleCell: func(style *walk.CellStyle) {
    // 第 5 列（操作）：蓝色链接
    if col == 5 {
        style.TextColor = walk.RGB(0, 100, 200)
        return
    }
    // 第 0 列（名称）：测试通过变绿
    if col == 0 && isTestedOK(connCache[row].ID) {
        style.TextColor = walk.RGB(0, 128, 0)
    }
}
```

### 多线程安全

- `connMu` / `scriptMu`：保护表格数据 + 缓存的读写
- `testedOKMu`：保护测试状态映射
- SSH 执行在 goroutine 中，UI 更新通过 Walk 自动同步

### 文件间依赖

```
main.go
  ├── app.go (globals)
  ├── models.go (data)
  ├── db.go (persistence)
  ├── ssh.go (SSH)
  ├── ui_panels.go
  │     └── ui_actions.go
  │     └── ui_execute.go
  │     └── ui_history.go
  │     └── ui_filebrowser.go
```

所有 `ui_*.go` 共享 `app.go` 中的全局变量和 `models.go` 中的数据缓存。

---

## 🧪 测试

手动测试流程：

1. **连接测试**：添加可达的 SSH 服务器 → 点测试 → 名称变绿
2. **执行测试**：选连接 → 底部敲 `echo hello` → 回车 → 看状态栏
3. **文件浏览**：选连接 → 点 📂 文件 → 双击目录进入
4. **历史验证**：执行后检查执行历史完整性

---

## 📝 二次开发

### 添加新功能步骤

1. **新增数据库表** → `db.go` 加 DDL + CRUD
2. **新增模型** → `models.go` 加 struct + 缓存 + refresh
3. **新增 UI** → 新建 `ui_*.go`，引用 app.go 中的全局变量
4. **注册入口** → `ui_panels.go` 加按钮或 main.go 加菜单

### Walk 资源

- [Walk 仓库](https://github.com/lxn/walk)
- [Walk 示例](https://github.com/lxn/walk/tree/master/examples)
- [Go 1.20 标准库](https://pkg.go.dev/std@go1.20)

### 常见踩坑

| 问题 | 原因 | 解决 |
|------|------|------|
| Win7 闪退 | Go 1.20 以上不再支持 Win7 | 使用 Go 1.20 |
| TDM-GCC 链接失败 | `-static` 不识别 | 用 `-extldflags=-static` |
| 列显示 `<nil>` | map key 和 DataMember 不匹配 | 对齐两者名称 |
| 空文件列表 | `.Run()` 阻塞无法初始化 | 改用 `.Create()` + 手动 `.Run()` |
| 发送到错误 | walk 包 `TTM_ADDTOOL` 失败 | 应用层忽略，不修改 walk 源码 |
