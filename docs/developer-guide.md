# 开发者文档

> SSH Manager 技术架构、开发环境搭建、项目结构及二次开发指南

---

## 📋 技术栈

| 组件 | 选型 | 说明 |
|------|------|------|
| 语言 | **Go 1.20** | Win7 最后支持的 Go 版本 |
| GUI 框架 | **Walk** (github.com/lxn/walk) | 原生 Win32 GUI，无 Web 包袱 |
| 数据库 | **SQLite** (modernc.org/sqlite) | 纯 Go 实现，零 CGo 依赖 |
| SSH 客户端 | **golang.org/x/crypto/ssh** | 标准 SSH 库 |
| 编译器 | **TDM-GCC 10.3.0** (MinGW-w64) | Windows 原生编译 |
| 打包压缩 | **UPX** (可选) | 减小 exe 体积 |

---

## 🛠 开发环境搭建

### Windows 环境

#### 1. 安装 Go 1.20

下载 [Go 1.20.14](https://go.dev/dl/go1.20.14.windows-amd64.msi) 安装。

#### 2. 安装 TDM-GCC

下载 [TDM-GCC 10.3.0](https://jmeubank.github.io/tdm-gcc/)，安装时确保勾选"Add to PATH"。

验证安装：

```cmd
go version     :: 需显示 go1.20.x
gcc --version  :: 需显示 10.3.0
windres --version
```

#### 3. 设置国内镜像（可选）

```cmd
go env -w GO111MODULE=on
go env -w GOPROXY=https://goproxy.cn,direct
```

#### 4. 克隆项目

```cmd
git clone https://gitee.com/yjmqn/ssh-manager.git
cd ssh-manager
```

---

## 📁 项目结构

```
ssh-manager/
├── main.go          # 入口：MainWindow 构建、菜单、关于
├── models.go        # 数据模型、表数据源、缓存、测试状态
├── db.go            # SQLite 数据库操作
├── ssh.go           # SSH 客户端封装（连接、执行）
│
├── ui_panels.go     # 主窗口面板布局（连接表、脚本表）
├── ui_actions.go    # 增删改查对话框（连接、脚本）
├── ui_execute.go    # 执行中心对话框（支持脚本 + 命令模式）
├── ui_history.go    # 执行历史对话框
│
├── app.rc           # Windows 资源脚本（清单 + 图标）
├── app.manifest     # Windows 通用控件清单（Win7 兼容）
├── app.ico          # 程序图标
├── mkicon.py        # 图标生成脚本（可选）
│
├── build.bat        # 一键构建脚本
├── go.mod / go.sum  # Go 模块依赖
├── README.md        # 用户文档
│
└── docs/
    └── developer-guide.md  # 开发者文档（本文件）
```

---

## 🔧 构建

### 一键构建

双击 `build.bat`，产物在 `dist/SSHManager.exe`。

### 手动构建

```cmd
:: 1. 生成资源文件（清单 + 图标）
windres -o app.syso -i app.rc

:: 2. 下载依赖
go mod tidy

:: 3. 编译（推荐静态链接）
set CGO_ENABLED=1
go build -ldflags="-s -w -H windowsgui -static" -o dist\SSHManager.exe .

:: 4. （可选）UPX 压缩
upx --best dist\SSHManager.exe
```

### 跨平台构建说明

本项目仅支持 Windows 平台编译（依赖 Win32 API）。  
在 Linux/macOS 上无法通过 CGo 编译 Walk 程序。

如需在 CI 中构建，可使用 Windows runner。

---

## 🧱 核心架构

### 数据流

```
用户操作 -> 对话框（ui_*.go）-> db.go（SQLite）-> models.go（缓存 + TableModel）
                                                        |
                                                   ui_panels.go（TableView 渲染）
```

### 执行流程

```
执行中心 -> 选择连接 + 脚本/命令
              |
         ssh.go: Connect() -> SSH 握手
              |
         ssh.go: Execute() -> 远程执行
              |
         实时回调写入输出 TextEdit
              |
         db.go: AddHistory() / UpdateHistory() -> 记录执行历史
```

### Walk TableView 数据绑定

Walk 的 `TableView` 要求数据源为 `[]map[string]interface{}`。

- `models.go` 中的 `refreshConnData()` / `refreshScriptData()` 构建 map 数组
- 列的 `DataMember` 必须匹配 map 的 key（已在 UI 定义中设置）
- 额外数据（ID、密码等）通过 `connCache` / `scriptCache` 独立存储
- 通过 `connCache[idx].ID` 从行索引反查数据库 ID

### CellStyler 机制

`ui_panels.go` 中使用 `StyleCell` 回调控制单元格样式：

- **第 0 列（名称）**：测试通过 -> 文字变绿
- **第 5 列（操作）**：蓝色链接样式

```go
StyleCell: func(style *walk.CellStyle) {
    if col == 5 {
        style.TextColor = walk.RGB(0, 100, 200)
        return
    }
    if col == 0 && isTestedOK(connCache[row].ID) {
        style.TextColor = walk.RGB(0, 128, 0)
    }
}
```

---

## 🧪 测试

目前项目无单元测试框架，手动测试可通过：

1. **连接测试**：添加一个已知可达的 SSH 服务器，点击测试
2. **执行测试**：选择连接后执行 `echo hello` 或 `whoami`
3. **历史验证**：执行后检查历史记录是否完整

---

## 📝 二次开发指南

### 添加新功能

1. **新增数据库表** -> 在 `db.go` 中添加 SQL DDL 和 CRUD 方法
2. **新增模型** -> 在 `models.go` 中添加 struct + 缓存 + refresh 函数
3. **新增 UI 面板** -> 在 `ui_*.go` 中添加对话框或面板布局
4. **注册入口** -> 在 `main.go` 或 `ui_panels.go` 中添加按钮/菜单

### Walk 资源

- [Walk 官方文档](https://github.com/lxn/walk)
- [Walk 示例代码](https://github.com/lxn/walk/tree/master/examples)
- [Go 1.20 标准库](https://pkg.go.dev/std@go1.20)

### 常见坑

1. **Win7 兼容**：Go 1.20 是最后支持 Win7 的版本，Go 1.21+ 不再支持
2. **Tooltip 初始化失败**：Walk 在某些 Win7 环境 `TTM_ADDTOOL` 会失败，已在应用层忽略该错误
3. **`-static` 链接失败**：TDM-GCC 版本过旧可能导致，升级到 10.3.0 或改用 `-static-libgcc -static-libstdc++`
4. **TableView 列显示**：必须设置 `DataMember` 匹配 map key，否则显示 `<nil>`
