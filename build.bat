@echo off
chcp 65001 >nul
title SSH Manager - Go Build
color 0A
REM ============================================================
REM SSH Manager — 一键构建脚本
REM 用法: build.bat [--release]
REM --release: 启用 UPX 压缩、嵌入版本信息
REM ============================================================

setlocal enabledelayedexpansion

echo.
echo ╔═══════════════════════════════════════╗
echo ║   SSH Manager  Go Build              ║
echo ╚═══════════════════════════════════════╝
echo.

:check_go
where go >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo [ERROR] Go not found. Install Go 1.20:
    echo         https://go.dev/dl/go1.20.14.windows-amd64.msi
    pause
    exit /b 1
)
echo [INFO] Go version:
go version
echo.

:: ---- Step 1: Environment ----
echo [1/5] Setting Go proxy...
go env -w GO111MODULE=on
go env -w GOPROXY=https://goproxy.cn,direct
echo        GOPROXY = https://goproxy.cn

:: ---- Step 2: Generate resources (icon + manifest) ----
echo.
echo [2/5] Embedding resources (icon + manifest)...
where windres >nul 2>nul
if %ERRORLEVEL% equ 0 (
    pushd assets
    windres -o ..\app.syso -i app.rc
    popd
    if %ERRORLEVEL% equ 0 (
        echo       [OK] app.syso generated from assets/
    ) else (
        echo       [WARN] windres failed
    )
) else (
    echo       [WARN] windres not found (install TDM-GCC for static linking)
)
echo.

:: ---- Step 3: Download dependencies ----
echo [3/5] Downloading dependencies...
go mod tidy
if %ERRORLEVEL% neq 0 (
    echo [ERROR] Dependency download failed
    pause
    exit /b 1
)
echo       [OK]

:: ---- Step 4: Compile ----
echo.
echo [4/5] Compiling (static link MinGW)...
set CGO_ENABLED=1
if not exist dist mkdir dist

set LDFLAGS=-s -w -H windowsgui -extldflags=-static
if "%1"=="--release" set LDFLAGS=-s -w -H windowsgui -extldflags=-static -X main.version=1.0

go build -ldflags="%LDFLAGS%" -o dist\SSHManager.exe .
if %ERRORLEVEL% neq 0 (
    echo       [WARN] Static link failed, trying dynamic...
    go build -ldflags="-s -w -H windowsgui" -o dist\SSHManager.exe .
    if %ERRORLEVEL% neq 0 (
        echo [ERROR] Build failed
        pause
        exit /b 1
    )
)
echo       [OK] dist\SSHManager.exe

:: ---- Step 5: Compress (UPX, optional) ----
if "%1"=="--release" (
    echo.
    echo [5/5] Compressing with UPX...
    where upx >nul 2>nul
    if %ERRORLEVEL% equ 0 (
        upx --best dist\SSHManager.exe >nul 2>&1
        echo       [OK] UPX compression done
    ) else (
        echo       [SKIP] UPX not installed
    )
) else (
    echo.
    echo [5/5] Skipping compression. Use build.bat --release for UPX.
)

:: ---- Done ----
call :show_size dist\SSHManager.exe
echo.
echo ========================================
echo   DONE!
echo   Output: %cd%\dist\SSHManager.exe
echo ========================================
echo.
pause
goto :eof

:show_size
if exist %1 (
    for %%I in (%1) do (
        set sz=%%~zI
        set /a skb=sz/1024
        if !skb! geq 1024 (
            set /a smb=skb/1024
            echo   File size: !smb! MB (!skb! KB)
        ) else (
            echo   File size: !skb! KB
        )
    )
)
goto :eof
