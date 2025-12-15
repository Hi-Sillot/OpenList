#!/bin/bash

# ========================================
#   Linux/WSL2 Local Build Script
#   Prerequisite: Frontend assets are in public/dist directory
#   Dependencies: Go and Git must be installed
# ========================================

set -e # Exit immediately if a command exits with a non-zero status.

# 1. Check prerequisites
echo
echo "🔍 Checking environment..."
if [ ! -d "public/dist" ]; then
    echo "❌ ERROR: 'public/dist' directory not found."
    echo "Please build the frontend and place it in that directory."
    exit 1
fi

if [ -z "$(ls -A public/dist)" ]; then
    echo "❌ ERROR: 'public/dist' directory is empty."
    echo "Please build the frontend and place it in that directory."
    exit 1
fi

if ! command -v go &> /dev/null; then
    echo "❌ ERROR: Go compiler not found."
    echo "Please ensure Go is installed and added to your PATH."
    exit 1
fi

echo
echo "📦 Preparing build information..." -C

# 2. Set build variables
appName="openlist"

# Set default values first
version="v0.0.0"
gitCommit="unknown"

echo "Attempting to get version from Git..."
if git describe --abbrev=0 --tags >/dev/null 2>&1; then
    version=$(git describe --abbrev=0 --tags)
    echo "Found git tag: $version"
else
    echo "WARN: No git tags found. Using default version '$version'."
fi

if git log -n 1 --pretty=format:"%h" >/dev/null 2>&1; then
    gitCommit=$(git log --pretty=format:"%h" -1)
    echo "Found git commit: $gitCommit"
else
    echo "WARN: No git commits found. Using default commit hash '$gitCommit'."
fi

builtAt=$(date +'%F %T %z')

ldflags="-w -s -X 'github.com/OpenListTeam/OpenList/v4/internal/conf.BuiltAt=$builtAt' -X 'github.com/OpenListTeam/OpenList/v4/internal/conf.GitAuthor=The OpenList Projects Contributors <noreply@openlist.team>' -X 'github.com/OpenListTeam/OpenList/v4/internal/conf.GitCommit=$gitCommit' -X 'github.com/OpenListTeam/OpenList/v4/internal/conf.Version=$version'"

echo "Backend version: $version"
echo "Frontend version: Local (embedded in public/dist)"

# 3. Prepare output directory
OutputDir="dist/linux"
rm -rf "$OutputDir"
mkdir -p "$OutputDir"

echo
echo "🔨 Building for linux-amd64..."

# 4. Build for Linux (AMD64)
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=1

go build -o "./$OutputDir/${appName}-linux-amd64" -ldflags="$ldflags" -tags=jsoniter .

# 5. Finish
echo
echo "--------------------------------------------------"
echo "✅ Linux version built successfully!"
echo "File location: ./$OutputDir/${appName}-linux-amd64"
echo "--------------------------------------------------"
