# BigFix Enterprise Mobile (BEM) Server - Windows Uninstaller
# Requires Administrator privileges

#Requires -RunAsAdministrator

$ErrorActionPreference = "Stop"

Write-Host "BigFix Enterprise Mobile (BEM) Server Uninstaller" -ForegroundColor Cyan
Write-Host "==================================================" -ForegroundColor Cyan
Write-Host ""

# Installation directory
$InstallDir = "C:\Program Files\BEM"
$DataDir = "C:\ProgramData\BEM"

# Check if installed
if (-not (Test-Path $InstallDir)) {
    Write-Host "BEM Server is not installed." -ForegroundColor Yellow
    exit 0
}

# Confirm uninstallation
$Confirm = Read-Host "Are you sure you want to uninstall BEM Server? (Y/N)"
if ($Confirm -ne "Y" -and $Confirm -ne "y") {
    Write-Host "Uninstall cancelled." -ForegroundColor Yellow
    exit 0
}

# Remove firewall rule
Write-Host "Removing firewall rule..."
try {
    Remove-NetFirewallRule -DisplayName "BEM Server" -ErrorAction SilentlyContinue | Out-Null
    Write-Host "  Firewall rule removed" -ForegroundColor Green
} catch {
    Write-Host "  Warning: Could not remove firewall rule" -ForegroundColor Yellow
}

# Remove from PATH
Write-Host "Removing BEM from system PATH..."
$CurrentPath = [Environment]::GetEnvironmentVariable("Path", [EnvironmentVariableTarget]::Machine)
$NewPath = ($CurrentPath -split ';' | Where-Object { $_ -ne $InstallDir }) -join ';'
[Environment]::SetEnvironmentVariable(
    "Path",
    $NewPath,
    [EnvironmentVariableTarget]::Machine
)
Write-Host "  Removed from PATH" -ForegroundColor Green

# Remove installation directory
Write-Host "Removing installation directory..."
Remove-Item -Path $InstallDir -Recurse -Force
Write-Host "  Installation directory removed" -ForegroundColor Green

# Ask about data directory
Write-Host ""
$RemoveData = Read-Host "Remove configuration and data directory? ($DataDir) (Y/N)"
if ($RemoveData -eq "Y" -or $RemoveData -eq "y") {
    Remove-Item -Path $DataDir -Recurse -Force
    Write-Host "  Data directory removed" -ForegroundColor Green
} else {
    Write-Host "  Data directory preserved" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "Uninstallation Complete!" -ForegroundColor Green
Write-Host ""
