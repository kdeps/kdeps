#Requires -Version 5.1
<#
.SYNOPSIS
  PowerShell equivalent of `make build` (Makefile has no native Windows runner).
.DESCRIPTION
  Mirrors the Makefile's `build` target: same VERSION default, same COMMIT
  detection via git, same ldflags, same output name relative to this script.
.EXAMPLE
  .\build.ps1
.EXAMPLE
  .\build.ps1 -Version 2.1.0
#>
[CmdletBinding()]
param(
    [string]$Version = "2.0.0-dev"
)

$ErrorActionPreference = "Stop"

$commit = "dev"
try {
    $rev = git rev-parse --short HEAD 2>$null
    if ($LASTEXITCODE -eq 0 -and $rev) { $commit = $rev.Trim() }
} catch {}

Write-Host "Building kdeps v$Version..."

$ldflags = "-X github.com/kdeps/kdeps/v2/pkg/version.Version=$Version " +
           "-X github.com/kdeps/kdeps/v2/pkg/version.Commit=$commit"

go build -ldflags $ldflags -o kdeps.exe main.go

Write-Host "Build complete: .\kdeps.exe" -ForegroundColor Green
