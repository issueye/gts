# GoScript VSCode Extension Packaging Script
# Usage: .\package.ps1 [-Local] [-Clean]

param(
    [switch]$Local,
    [switch]$Clean
)

$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

Push-Location $ScriptDir

try {
    if ($Clean) {
        Write-Host "Cleaning old vsix files..."
        Remove-Item -Path "*.vsix" -ErrorAction SilentlyContinue
    }

    Write-Host "Packaging GoScript VSCode extension..."
    $args = @("package", "--allow-missing-repository")
    if ($Clean) {
        $args += "--skip-version-check"
    }

    npx --yes @vscode/vsce @args
    if ($LASTEXITCODE -ne 0) {
        throw "vsce package failed"
    }

    $vsix = Get-ChildItem -Filter "*.vsix" | Sort-Object LastWriteTime -Descending | Select-Object -First 1
    Write-Host "Package created: $($vsix.Name)"

    if ($Local) {
        Write-Host "Installing extension locally..."
        code --install-extension $vsix.Name
        if ($LASTEXITCODE -ne 0) {
            throw "code --install-extension failed"
        }
        Write-Host "Installation complete. Reload VSCode windows to activate."
    }
} finally {
    Pop-Location
}
