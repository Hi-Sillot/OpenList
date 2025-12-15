#!/usr/bin/env pwsh

# ========================================
#   Windows Local Build Script
#   Prerequisite: Frontend assets are in public/dist directory
#   Dependencies: Go and Git for Windows must be installed
# ========================================

# Stop on any error
$ErrorActionPreference = "Stop"

Write-Host ""
Write-Host "? Checking environment..." -ForegroundColor Cyan

# 1. Check prerequisites
if (-not (Test-Path "public\dist")) {
    Write-Host "ERROR: 'public\dist' directory not found." -ForegroundColor Red
    Write-Host "Please build the frontend and place it in that directory."
    Read-Host -Prompt "Press Enter to exit"
    exit 1
}

if (-not (Get-Command "go" -ErrorAction SilentlyContinue)) {
    Write-Host "ERROR: Go compiler not found." -ForegroundColor Red
    Write-Host "Please ensure Go is installed and added to your system's PATH."
    Read-Host -Prompt "Press Enter to exit"
    exit 1
}

Write-Host ""
Write-Host "? Preparing build information..." -ForegroundColor Cyan

# 2. Set build variables
$appName = "openlist"

# Set default values first
$version = "v0.0.0"
$gitCommit = "unknown"

Write-Host "Attempting to get version from Git..."
try {
    # Try to get the latest tag
    $tag = & git describe --abbrev=0 --tags 2>$null
    if ($tag) {
        $version = $tag
        Write-Host "Found git tag: $version" -ForegroundColor Green
    } else {
        Write-Host "WARN: No git tags found. Using default version '$version'." -ForegroundColor Yellow
    }

    # Try to get the latest commit hash
    $commit = & git log --pretty=format:"%h" -1 2>$null
    if ($commit) {
        $gitCommit = $commit
        Write-Host "Found git commit: $gitCommit" -ForegroundColor Green
    } else {
        Write-Host "WARN: No git commits found. Using default commit hash '$gitCommit'." -ForegroundColor Yellow
    }
}
catch {
    # This will catch any other unexpected errors from git
    Write-Host "WARN: An error occurred while getting git info. Is this a valid git repository?" -ForegroundColor Yellow
    Write-Host "Using default version and commit hash." -ForegroundColor Yellow
}

$builtAt = Get-Date -UFormat "%Y-%m-%d %H:%M:%S %z"

$ldflags = @"
-w -s
-X 'github.com/OpenListTeam/OpenList/v4/internal/conf.BuiltAt=$builtAt'
-X 'github.com/OpenListTeam/OpenList/v4/internal/conf.GitAuthor=The OpenList Projects Contributors <noreply@openlist.team>'
-X 'github.com/OpenListTeam/OpenList/v4/internal/conf.GitCommit=$gitCommit'
-X 'github.com/OpenListTeam/OpenList/v4/internal/conf.Version=$version'
"@

Write-Host "Backend version: $version"
Write-Host "Frontend version: Local (embedded in public/dist)"

# 3. Prepare output directory
$OutputDir = "dist\windows"
if (Test-Path $OutputDir) {
    Remove-Item -Recurse -Force $OutputDir
}
New-Item -ItemType Directory -Path $OutputDir | Out-Null

Write-Host ""
Write-Host "? Building for windows-amd64..." -ForegroundColor Green

# 4. Build for Windows (AMD64)
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "1"

try {
    & go build -o ".\$OutputDir\$appName-windows-amd64.exe" -ldflags $ldflags -tags=jsoniter .
}
catch {
    Write-Host "? Build failed: $_" -ForegroundColor Red
    Read-Host -Prompt "Press Enter to exit"
    exit 1
}

# 5. Finish
Write-Host ""
Write-Host "--------------------------------------------------" -ForegroundColor Green
Write-Host "? Windows version built successfully!" -ForegroundColor Green
Write-Host "File location: .\$OutputDir\$appName-windows-amd64.exe"
Write-Host "--------------------------------------------------" -ForegroundColor Green
Read-Host -Prompt "Press Enter to exit"
