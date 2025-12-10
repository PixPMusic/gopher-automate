#!/bin/bash
# Local build script for Linux (AMD64 & ARM64) using custom Docker image

APP_NAME="GopherAutomate"
APP_ID="cc.pixp.GopherAutomate"
ICON_SOURCE="assets/app_icon.png"

# Extract version logic
VERSION=$(grep -m 1 "^## \[[0-9]" CHANGELOG.md | sed -E 's/## \[([^]]+)\].*/\1/')
if [ -z "$VERSION" ]; then VERSION="0.0.1"; fi
echo "Detected version: $VERSION"

echo "Checking for fyne-cross..."
if ! command -v fyne-cross &> /dev/null; then
    echo "Installing fyne-cross..."
    go install github.com/fyne-io/fyne-cross@latest
fi

# Build custom image with libasound2-dev for cross-compilation
# This image includes libs for both arm64 and amd64 targets
echo "Building custom Docker image (with cross-arch libasound2-dev)..."
docker build --no-cache -t fyneio/fyne-cross-images:linux -f scripts/docker/Dockerfile.linux .

# Verify image exists
if ! docker image inspect fyneio/fyne-cross-images:linux > /dev/null 2>&1; then
    echo "Error: Docker image not found!"
    exit 1
fi

echo "Building for Linux (amd64, arm64)..."
# Don't use -image flag - fyne-cross will use the standard image name which we've overridden
fyne-cross linux -arch=amd64,arm64 -icon "$ICON_SOURCE" -name "$APP_NAME" -app-id "$APP_ID" -app-version "$VERSION"

echo "Build complete. Artifacts in fyne-cross/bin/linux-*"
