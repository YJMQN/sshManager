@echo off
chcp 936 >nul
title SSH Manager Diagnostic Tests
color 0F

echo ====================================
echo  SSH Manager — 诊断测试
echo ====================================
echo.
echo 本脚本依次运行三个测试程序，定位问题。
echo.

set TEST_DIR=%~dp0

:: ---- Test 1: Config ----
echo [1/3] 测试 %%APPDATA%% 配置读写...
"%TEST_DIR%test_config.exe"
if %ERRORLEVEL% equ 0 (
    echo [OK] 配置读写正常
) else (
    echo [FAIL] 配置读写异常，退出码: %ERRORLEVEL%
)
echo.

:: ---- Test 2: SQLite ----
echo [2/3] 测试 SQLite 数据库...
"%TEST_DIR%test_db.exe"
if %ERRORLEVEL% equ 0 (
    echo [OK] SQLite 正常
) else (
    echo [FAIL] SQLite 异常，退出码: %ERRORLEVEL%
)
echo.

:: ---- Test 3: Walk GUI ----
echo [3/3] 测试 Walk 窗口...
echo 请观察是否出现测试窗口。如果出现，点击按钮测试对话框。
echo 日志会写入 test_walk.log
echo.
start /wait "" "%TEST_DIR%test_walk.exe"
echo.
if exist "%TEST_DIR%test_walk.log" (
    echo [LOG] test_walk.log 内容:
    type "%TEST_DIR%test_walk.log"
) else (
    echo [WARN] test_walk.log 未生成，程序可能未启动
)
echo.

echo ====================================
echo  测试完成！
echo  请根据结果判断问题所在。
echo ====================================
pause
