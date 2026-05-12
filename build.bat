@echo off
title SSH Manager Build
color 0A

echo ==============================
echo   SSH Manager Build Script
echo ==============================
echo.

:: Check Go
where go >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo [ERROR] Go not found.
    pause
    exit /b 1
)
echo [OK] Go found
go version
echo.

:: Step 1
echo [1/5] Setting Go proxy...
go env -w GO111MODULE=on
go env -w GOPROXY=https://goproxy.cn,direct
echo.

:: Step 2
echo [2/5] Embedding resources...
where windres >nul 2>nul
if %ERRORLEVEL% equ 0 (
    pushd assets
    windres -o ..\app.syso -i app.rc
    popd
    if %ERRORLEVEL% equ 0 (
        echo [OK] app.syso generated
    ) else (
        echo [WARN] windres failed
    )
) else (
    echo [WARN] windres not found
)
echo.

:: Step 3
echo [3/5] Downloading dependencies...
go mod tidy
if %ERRORLEVEL% neq 0 (
    echo [ERROR] go mod tidy failed
    pause
    exit /b 1
)
echo [OK] Dependencies ready
echo.

:: Step 4
echo [4/5] Compiling...
set CGO_ENABLED=1
if not exist dist mkdir dist

go build -ldflags="-s -w -H windowsgui -extldflags=-static" -o dist\SSHManager.exe .
if %ERRORLEVEL% neq 0 (
    echo [WARN] Static link failed, trying dynamic...
    go build -ldflags="-s -w -H windowsgui" -o dist\SSHManager.exe .
    if %ERRORLEVEL% neq 0 (
        echo [ERROR] Build failed
        pause
        exit /b 1
    )
)
echo [OK] Build done
echo.

:: Step 5
echo [5/5] Compressing (optional)...
where upx >nul 2>nul
if %ERRORLEVEL% equ 0 (
    upx --best dist\SSHManager.exe >nul 2>&1
    echo [OK] UPX done
) else (
    echo [SKIP] UPX not found
)
echo.

:: Done
echo ==============================
echo   Done!
echo   Output: dist\SSHManager.exe
echo ==============================
pause
