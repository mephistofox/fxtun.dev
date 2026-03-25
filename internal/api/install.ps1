$ErrorActionPreference = "Stop"

$BinaryName = "fxtunnel"
$InstallDir = "$env:LOCALAPPDATA\fxTunnel"
$BaseURL = "{{.BaseURL}}"
$WebsiteURL = "{{.WebsiteURL}}"

function Main {
    Write-Host ""
    Write-Host "  fxTunnel Installer for Windows" -ForegroundColor Cyan
    Write-Host ""

    # Check existing installation
    $existing = Get-Command $BinaryName -ErrorAction SilentlyContinue
    if ($existing) {
        $version = & $existing.Source version 2>$null
        Write-Host "  fxtun is already installed ($version). Reinstalling..." -ForegroundColor Yellow
    }

    # Create install directory
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    }

    # Download
    $DownloadURL = "$BaseURL/cli-windows-amd64"
    $Target = Join-Path $InstallDir "$BinaryName.exe"
    $TempFile = Join-Path $env:TEMP "fxtunnel-install.exe"

    Write-Host "  Downloading fxtun for Windows/amd64..." -ForegroundColor White
    try {
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        $ProgressPreference = 'SilentlyContinue'
        Invoke-WebRequest -Uri $DownloadURL -OutFile $TempFile -UseBasicParsing
    } catch {
        Write-Host "  Error: download failed - $_" -ForegroundColor Red
        exit 1
    }

    if (-not (Test-Path $TempFile) -or (Get-Item $TempFile).Length -eq 0) {
        Write-Host "  Error: download failed (empty file)" -ForegroundColor Red
        exit 1
    }

    # Install
    Write-Host "  Installing to $InstallDir..." -ForegroundColor White
    Move-Item -Force $TempFile $Target

    # Create fxtun alias
    $Alias = Join-Path $InstallDir "fxtun.exe"
    Copy-Item $Target $Alias -Force

    # Save website URL
    Set-Content (Join-Path $InstallDir ".fxtunnel-website") $WebsiteURL -NoNewline

    # Add to PATH
    $UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($UserPath -notlike "*$InstallDir*") {
        [Environment]::SetEnvironmentVariable("Path", "$InstallDir;$UserPath", "User")
        $env:Path = "$InstallDir;$env:Path"
        Write-Host "  Added $InstallDir to PATH." -ForegroundColor Green
    }

    # Verify
    Write-Host ""
    try {
        & $Target version
    } catch {
        Write-Host "  Installed successfully (version check skipped)" -ForegroundColor Yellow
    }

    Write-Host ""
    Write-Host "  fxtun installed successfully!" -ForegroundColor Green
    Write-Host "  Available as: fxtun, fxtunnel" -ForegroundColor White
    Write-Host ""
    Write-Host "  Restart your terminal to use fxtun from any directory." -ForegroundColor Yellow
    Write-Host ""
}

Main
