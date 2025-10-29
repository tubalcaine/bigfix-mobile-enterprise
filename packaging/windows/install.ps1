# BigFix Enterprise Mobile (BEM) Server - Windows Installer
# Requires Administrator privileges

#Requires -RunAsAdministrator

$ErrorActionPreference = "Stop"

Write-Host "BigFix Enterprise Mobile (BEM) Server Installer" -ForegroundColor Cyan
Write-Host "================================================" -ForegroundColor Cyan
Write-Host ""

# Installation directory
$InstallDir = "C:\Program Files\BEM"
$DataDir = "C:\ProgramData\BEM"
$ConfigDir = "$DataDir\config"

Write-Host "Installing BEM Server to: $InstallDir" -ForegroundColor Green

# Create directories
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
New-Item -ItemType Directory -Force -Path $DataDir | Out-Null
New-Item -ItemType Directory -Force -Path $ConfigDir | Out-Null
New-Item -ItemType Directory -Force -Path "$DataDir\registrations" | Out-Null
New-Item -ItemType Directory -Force -Path "$DataDir\requests" | Out-Null

# Copy binary
Write-Host "Copying BEM executable..."
Copy-Item -Path ".\bem.exe" -Destination "$InstallDir\bem.exe" -Force

# Copy documentation
Write-Host "Copying documentation..."
if (Test-Path ".\README.md") {
    Copy-Item -Path ".\README.md" -Destination "$InstallDir\README.md" -Force
}
if (Test-Path ".\LICENSE") {
    Copy-Item -Path ".\LICENSE" -Destination "$InstallDir\LICENSE" -Force
}

# Add to PATH
Write-Host "Adding BEM to system PATH..."
$CurrentPath = [Environment]::GetEnvironmentVariable("Path", [EnvironmentVariableTarget]::Machine)
if ($CurrentPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable(
        "Path",
        "$CurrentPath;$InstallDir",
        [EnvironmentVariableTarget]::Machine
    )
    Write-Host "  Added $InstallDir to PATH" -ForegroundColor Green
} else {
    Write-Host "  $InstallDir already in PATH" -ForegroundColor Yellow
}

# Create firewall rule
Write-Host "Creating firewall rule for BEM Server..."
try {
    New-NetFirewallRule -DisplayName "BEM Server" `
                        -Direction Inbound `
                        -Program "$InstallDir\bem.exe" `
                        -Action Allow `
                        -Profile Any `
                        -ErrorAction SilentlyContinue | Out-Null
    Write-Host "  Firewall rule created" -ForegroundColor Green
} catch {
    Write-Host "  Warning: Could not create firewall rule" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "Installation Complete!" -ForegroundColor Green
Write-Host ""
Write-Host "Next Steps:" -ForegroundColor Cyan
Write-Host "  1. Create a configuration file at: $ConfigDir\bem.json"
Write-Host "  2. Generate TLS certificates (required for operation)"
Write-Host "  3. Run 'bem.exe -c $ConfigDir\bem.json' to start the server"
Write-Host ""
Write-Host "To run as a Windows Service:"
Write-Host "  Use NSSM (Non-Sucking Service Manager) or sc.exe to create a service"
Write-Host ""
Write-Host "Documentation: $InstallDir\README.md"
Write-Host ""
