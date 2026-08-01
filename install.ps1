#Requires -Version 5.1
<#
.SYNOPSIS
  Install kdeps on Windows.
.DESCRIPTION
  Downloads the latest (or a pinned) kdeps release zip from GitHub, extracts
  kdeps.exe into an install directory, and adds that directory to the user's
  PATH. Mirrors install.sh's -b/-tag flags for parity across platforms.
.PARAMETER BinDir
  Directory to install kdeps.exe into. Defaults to "$env:USERPROFILE\.local\bin".
.PARAMETER Tag
  Release tag to install (e.g. "v2.1.0"). Defaults to the latest release.
.EXAMPLE
  irm https://raw.githubusercontent.com/kdeps/kdeps/main/install.ps1 | iex
.EXAMPLE
  & ([scriptblock]::Create((irm https://raw.githubusercontent.com/kdeps/kdeps/main/install.ps1))) -Tag v2.1.0 -BinDir C:\tools\bin
#>
[CmdletBinding()]
param(
    [string]$BinDir = "$env:USERPROFILE\.local\bin",
    [string]$Tag = ""
)

$ErrorActionPreference = "Stop"

function Write-Info($msg) { Write-Host $msg -ForegroundColor Cyan }
function Write-Success($msg) { Write-Host $msg -ForegroundColor Green }
function Write-Err($msg) { Write-Host $msg -ForegroundColor Red }

$Owner = "kdeps"
$Repo = "kdeps"

$arch = if ([Environment]::Is64BitOperatingSystem) {
    if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64" -or $env:PROCESSOR_ARCHITEW6432 -eq "ARM64") { "arm64" } else { "x86_64" }
} else {
    "i386"
}

if ($arch -ne "x86_64") {
    Write-Err "kdeps Windows releases are only published for x86_64; detected '$arch'."
    exit 1
}

if ([string]::IsNullOrEmpty($Tag)) {
    Write-Info "Checking GitHub for the latest kdeps release..."
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Owner/$Repo/releases/latest" -Headers @{ "Accept" = "application/json" }
    $Tag = $release.tag_name
} else {
    Write-Info "Using pinned release $Tag"
}

if ([string]::IsNullOrEmpty($Tag)) {
    Write-Err "Unable to determine a release tag to install."
    exit 1
}

$asset = "kdeps_Windows_x86_64.zip"
$downloadUrl = "https://github.com/$Owner/$Repo/releases/download/$Tag/$asset"

Write-Info "Installing kdeps $Tag for windows/$arch"

$tmpDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.IO.Path]::GetRandomFileName())
New-Item -ItemType Directory -Path $tmpDir -Force | Out-Null

try {
    $zipPath = Join-Path $tmpDir $asset
    Write-Info "Downloading $downloadUrl"
    Invoke-WebRequest -Uri $downloadUrl -OutFile $zipPath -UseBasicParsing

    Write-Info "Extracting archive..."
    Expand-Archive -Path $zipPath -DestinationPath $tmpDir -Force

    $exeSrc = Get-ChildItem -Path $tmpDir -Filter "kdeps.exe" -Recurse | Select-Object -First 1
    if (-not $exeSrc) {
        Write-Err "kdeps.exe not found in downloaded archive."
        exit 1
    }

    if (-not (Test-Path $BinDir)) {
        New-Item -ItemType Directory -Path $BinDir -Force | Out-Null
    }

    $exeDest = Join-Path $BinDir "kdeps.exe"
    Copy-Item -Path $exeSrc.FullName -Destination $exeDest -Force
    Write-Success "  kdeps.exe -> $exeDest"
} finally {
    Remove-Item -Path $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
}

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
$pathEntries = @()
if ($userPath) { $pathEntries = $userPath -split ";" }
if ($pathEntries -notcontains $BinDir) {
    $newUserPath = if ($userPath) { "$userPath;$BinDir" } else { $BinDir }
    [Environment]::SetEnvironmentVariable("Path", $newUserPath, "User")
    Write-Info "Added $BinDir to your user PATH."
}
if (($env:Path -split ";") -notcontains $BinDir) {
    $env:Path = "$env:Path;$BinDir"
}

Write-Success "kdeps $Tag installed successfully."
Write-Host "Restart your terminal, then run:"
Write-Host "  kdeps --version" -ForegroundColor Yellow
