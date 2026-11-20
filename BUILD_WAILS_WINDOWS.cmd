@echo off
setlocal
cd /d "%~dp0"
chcp 65001 >nul
where powershell.exe >nul 2>nul
if errorlevel 1 (
  echo Windows PowerShell 5.1 was not found.
  pause
  exit /b 1
)
powershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -File "%~dp0BUILD_WAILS_WINDOWS.ps1"
set "BUILD_RC=%ERRORLEVEL%"
if not "%BUILD_RC%"=="0" (
  echo.
  echo Build process exited with code %BUILD_RC%.
  pause
)
exit /b %BUILD_RC%
