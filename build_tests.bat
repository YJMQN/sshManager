@echo off
chcp 936 >nul
title Build Diagnostic Tests
cd /d "%~dp0"

echo ====================================
echo  Building Diagnostic Tests
echo ====================================
echo.

:: ---- Step 1: Create resource file for each test ----
echo [1/4] Preparing .syso for test_walk (with manifest)...
copy /y ..\assets\app.rc tests\test_walk\ >nul 2>&1
copy /y ..\assets\app.manifest tests\test_walk\ >nul 2>&1
copy /y ..\assets\app.ico tests\test_walk\ >nul 2>&1
pushd tests\test_walk
windres -o app.syso -i app.rc
popd
if %ERRORLEVEL% neq 0 (echo FAILED & pause & exit /b 1)
echo OK

echo [2/4] Preparing .syso for test_nowidget (with manifest)...
copy /y ..\assets\app.rc tests\test_nowidget\ >nul 2>&1
copy /y ..\assets\app.manifest tests\test_nowidget\ >nul 2>&1
copy /y ..\assets\app.ico tests\test_nowidget\ >nul 2>&1
pushd tests\test_nowidget
windres -o app.syso -i app.rc
popd
if %ERRORLEVEL% neq 0 (echo FAILED & pause & exit /b 1)
echo OK

:: ---- Step 2: Build test_config ----
echo [3/4] Building test_config.exe (no CGO needed)...
go build -o tests\test_config.exe tests\test_config\main.go
if %ERRORLEVEL% neq 0 (echo FAILED & pause & exit /b 1)
echo OK

:: ---- Step 3: Build test_walk ----
echo [4/4] Building test_walk.exe + test_nowidget.exe + test_db.exe...
go build -ldflags="-s -w -H windowsgui -extldflags=-static" -o tests\test_walk.exe tests\test_walk\main.go
if %ERRORLEVEL% neq 0 (echo FAILED - test_walk & pause & exit /b 1)
echo   test_walk OK

go build -ldflags="-s -w -H windowsgui -extldflags=-static" -o tests\test_nowidget.exe tests\test_nowidget\main.go
if %ERRORLEVEL% neq 0 (echo FAILED - test_nowidget & pause & exit /b 1)
echo   test_nowidget OK

go build -ldflags="-s -w -extldflags=-static" -o tests\test_db.exe tests\test_db\main.go
if %ERRORLEVEL% neq 0 (echo FAILED - test_db & pause & exit /b 1)
echo   test_db OK

echo.
echo ====================================
echo  All tests built!
echo  Output in tests/ directory:
echo    test_config.exe  - %%APPDATA%% config test
echo    test_db.exe      - SQLite database test
echo    test_walk.exe    - Walk GUI window (Label + button)
echo    test_nowidget.exe - Walk bare window (no widgets)
echo ====================================
pause
