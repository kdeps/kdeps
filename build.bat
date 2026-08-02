@echo off
REM Batch equivalent of `make build` / build.ps1 for cmd.exe users who want to
REM avoid PowerShell's script execution policy entirely.
REM Usage: build.bat [version]

setlocal enabledelayedexpansion

set VERSION=2.0.0-dev
if not "%~1"=="" set VERSION=%~1

set COMMIT=dev
for /f "delims=" %%i in ('git rev-parse --short HEAD 2^>nul') do set COMMIT=%%i

echo Building kdeps v%VERSION%...

go build -ldflags "-X github.com/kdeps/kdeps/v2/pkg/version.Version=%VERSION% -X github.com/kdeps/kdeps/v2/pkg/version.Commit=%COMMIT%" -o kdeps.exe main.go
if errorlevel 1 (
    echo Build failed.
    exit /b 1
)

echo Build complete: .\kdeps.exe
