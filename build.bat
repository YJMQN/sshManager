@echo off
chcp 65001 >nul
title SSH Manager - Go 编译构建
color 0A
REM ========================================
REM SSH Manager (Go 原生版) - 一键构建
REM ========================================

echo.
echo ╔═══════════════════════════════════════╗
echo ║   SSH Manager  Go 原生编译           ║
echo ╚═══════════════════════════════════════╝
echo.

:: ---- 检查 Go ----
where go >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo [错误] 未找到 Go 编译器
    echo 请安装 Go 1.20：https://go.dev/dl/go1.20.14.windows-amd64.msi
    pause
    exit /b 1
)

:: ---- 显示版本 ----
echo [信息] Go 版本:
go version
echo.

:: ---- 设置国内镜像 ----
echo [1/5] 设置 Go 代理 (国内加速)...
go env -w GO111MODULE=on
go env -w GOPROXY=https://goproxy.cn,direct
echo        GOPROXY = https://goproxy.cn

:: ---- 生成资源文件 (清单 + 图标) ----
echo.
echo [2/5] 嵌入资源文件 (清单 + 图标)...
where windres >nul 2>nul
if %ERRORLEVEL% equ 0 (
    windres -o app.syso -i app.rc
    if %ERRORLEVEL% equ 0 (
        echo       ✅ 资源已嵌入：清单 + 图标 (app.syso)
    ) else (
        echo       ⚠️  windres 执行失败，跳过资源
        echo        尝试手动运行: windres -o app.syso -i app.rc
    )
) else (
    echo       ⚠️  windres 未找到 (需要 TDM-GCC)
    echo        如果运行时闪退，请手动安装 rsrc:
    echo        go install github.com/akavel/rsrc@latest
    echo        rsrc -manifest app.manifest -o app.syso
    echo.
    echo        按任意键继续尝试编译...
    pause >nul
)

:: ---- 下载依赖 ----
echo.
echo [3/5] 下载依赖...
go mod tidy
if %ERRORLEVEL% neq 0 (
    echo [错误] 依赖下载失败，请检查网络
    pause
    exit /b 1
)
echo       ✅ 依赖下载完成

:: ---- 编译 ----
echo.
echo [4/5] 编译中 (静态链接 MinGW 库)...
set CGO_ENABLED=1
if not exist dist mkdir dist

:: 先用静态链接尝试
go build -ldflags="-s -w -H windowsgui -static" -o dist\SSHManager.exe .
if %ERRORLEVEL% neq 0 (
    echo       ⚠️ 静态链接失败，尝试动态链接...
    go build -ldflags="-s -w -H windowsgui" -o dist\SSHManager.exe .
    if %ERRORLEVEL% neq 0 (
        echo [错误] 编译失败
        pause
        exit /b 1
    )
)
echo       ✅ 编译成功

:: ---- UPX 压缩 ----
echo.
echo [5/5] 尝试压缩...
where upx >nul 2>nul
if %ERRORLEVEL% equ 0 (
    upx --best dist\SSHManager.exe >nul 2>&1
    echo       ✅ UPX 压缩完成
) else (
    echo       ⏭  UPX 未安装，跳过压缩
    echo       安装 UPX 可减小体积：https://upx.github.io/
)

:: ---- 完成 ----
echo.
echo ========================================
echo   ✅ 全部完成！
echo   产物: %cd%\dist\SSHManager.exe
echo.
for %%I in (dist\SSHManager.exe) do (
    set size=%%~zI
    if defined size (
        set /a size_kb=size/1024
        if !size_kb! geq 1024 (
            set /a size_mb=size_kb/1024
            echo   文件大小: !size_mb! MB
        ) else (
            echo   文件大小: !size_kb! KB
        )
    )
)
echo ========================================
echo.
pause
