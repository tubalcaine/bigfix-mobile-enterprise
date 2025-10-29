@echo off
echo BigFix Enterprise Mobile (BEM) Server Installer
echo ================================================
echo.
echo This installer requires Administrator privileges.
echo.

REM Check for administrator privileges
net session >nul 2>&1
if %errorLevel% == 0 (
    echo Running with Administrator privileges...
    echo.
    powershell.exe -ExecutionPolicy Bypass -File "%~dp0install.ps1"
    pause
) else (
    echo ERROR: This installer must be run as Administrator.
    echo.
    echo Right-click on install.bat and select "Run as administrator"
    echo.
    pause
    exit /b 1
)
